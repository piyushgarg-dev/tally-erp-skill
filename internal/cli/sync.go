package cli

import (
	"context"
	"encoding/xml"
	"flag"
	"fmt"
	"html"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/piyushgarg/tally-skill/internal/cache"
	"github.com/piyushgarg/tally-skill/internal/tally"
)

func RunSync(args []string) int { return runSyncWithIO(args, os.Stdout, os.Stderr) }

func runSyncWithIO(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.SetOutput(stderr)
	g := registerGlobals(fs)
	dataType := fs.String("type", "", "Data type to sync: sales, purchase")
	from := fs.String("from", "", "From date (YYYY-MM-DD)")
	to := fs.String("to", "", "To date (YYYY-MM-DD)")
	cacheDir := fs.String("cache-dir", ".tally-cache", "Cache directory path")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	if *dataType == "" {
		fmt.Fprintln(stderr, "tally sync: --type is required (sales or purchase)")
		return ExitUsage
	}
	if *from == "" || *to == "" {
		fmt.Fprintln(stderr, "tally sync: --from and --to are required")
		return ExitUsage
	}

	fromTally, err := isoToTallyDate(*from)
	if err != nil {
		fmt.Fprintf(stderr, "tally sync: --from: %v\n", err)
		return ExitUsage
	}
	toTally, err := isoToTallyDate(*to)
	if err != nil {
		fmt.Fprintf(stderr, "tally sync: --to: %v\n", err)
		return ExitUsage
	}

	voucherType := "Sales"
	if *dataType == "purchase" {
		voucherType = "Purchase"
	}

	// Chunk into monthly requests
	chunks, err := tally.ChunkDates(fromTally, toTally, "monthly")
	if err != nil {
		fmt.Fprintf(stderr, "tally sync: %v\n", err)
		return ExitUsage
	}

	c := tally.NewClient(g.URL(), g.Timeout)
	templateBody := buildSyncTemplate(g.Company, voucherType)

	var allXML strings.Builder
	allXML.WriteString("<ROOT>")
	allFailed := true

	for i, ch := range chunks {
		fmt.Fprintf(stderr, "tally: syncing chunk %d/%d (%s to %s)...\n", i+1, len(chunks), ch[0], ch[1])
		body := strings.ReplaceAll(templateBody, "{{FROMDATE}}", ch[0])
		body = strings.ReplaceAll(body, "{{TODATE}}", ch[1])

		resp, err := c.Post(context.Background(), body)
		if err != nil {
			fmt.Fprintf(stderr, "tally: chunk %d failed: %v\n", i+1, err)
			continue
		}
		allXML.WriteString(resp)
		allFailed = false
	}
	allXML.WriteString("</ROOT>")

	if allFailed {
		fmt.Fprintln(stderr, "tally sync: all chunks failed")
		return ExitConnect
	}

	// Parse XML into rows
	rows := parseVoucherXML(allXML.String())
	fmt.Fprintf(stderr, "tally: parsed %d line items\n", len(rows))

	if len(rows) == 0 {
		fmt.Fprintln(stderr, "tally sync: no data found")
		return ExitOK
	}

	// Write to file tree
	if err := cache.WriteCache(*cacheDir, g.Company, *dataType, *from, *to, rows); err != nil {
		fmt.Fprintf(stderr, "tally sync: write error: %v\n", err)
		return ExitTallyFailure
	}

	fmt.Fprintf(stdout, "Synced %d rows to %s/%s/%s/\n", len(rows), *cacheDir, cache.SanitizeFilename(g.Company), *dataType)
	return ExitOK
}

func buildSyncTemplate(company, voucherType string) string {
	var escaped strings.Builder
	xml.EscapeText(&escaped, []byte(company))
	companyEsc := escaped.String()

	escaped.Reset()
	xml.EscapeText(&escaped, []byte(voucherType))
	vtypeEsc := escaped.String()

	return `<ENVELOPE>
  <HEADER>
    <VERSION>1</VERSION>
    <TALLYREQUEST>Export</TALLYREQUEST>
    <TYPE>Collection</TYPE>
    <ID>VouchersWithItems</ID>
  </HEADER>
  <BODY>
    <DESC>
      <STATICVARIABLES>
        <SVCURRENTCOMPANY>` + companyEsc + `</SVCURRENTCOMPANY>
        <SVFROMDATE TYPE="Date">{{FROMDATE}}</SVFROMDATE>
        <SVTODATE TYPE="Date">{{TODATE}}</SVTODATE>
        <SVEXPORTFORMAT>$$SysName:XML</SVEXPORTFORMAT>
      </STATICVARIABLES>
      <TDL>
        <TDLMESSAGE>
          <COLLECTION NAME="VouchersWithItems">
            <TYPE>Voucher</TYPE>
            <FETCH>DATE, VOUCHERNUMBER, PARTYLEDGERNAME, VOUCHERTYPENAME, AMOUNT, GUID</FETCH>
            <FETCH>ALLINVENTORYENTRIES.LIST : STOCKITEMNAME, ACTUALQTY, BILLEDQTY, RATE, AMOUNT, DISCOUNT</FETCH>
            <FILTER>VchTypeFilter</FILTER>
          </COLLECTION>
          <SYSTEM TYPE="Formulae" NAME="VchTypeFilter">$VoucherTypeName = "` + vtypeEsc + `"</SYSTEM>
        </TDLMESSAGE>
      </TDL>
    </DESC>
  </BODY>
</ENVELOPE>`
}

var qtyNumRe = regexp.MustCompile(`[\d.]+`)

func parseVoucherXML(xmlData string) []cache.Row {
	var rows []cache.Row

	voucherRe := regexp.MustCompile(`<VOUCHER\b[^>]*>(.+?)</VOUCHER>`)
	vouchers := voucherRe.FindAllStringSubmatch(xmlData, -1)

	for _, vm := range vouchers {
		block := vm[1]
		party := extractField(block, "PARTYLEDGERNAME")
		vchNum := extractField(block, "VOUCHERNUMBER")
		dateRaw := extractField(block, "DATE")

		dateStr := ""
		if len(dateRaw) == 8 {
			dateStr = dateRaw[6:8] + "-" + dateRaw[4:6] + "-" + dateRaw[0:4]
		}

		invRe := regexp.MustCompile(`<ALLINVENTORYENTRIES\.LIST>(.*?)</ALLINVENTORYENTRIES\.LIST>`)
		invBlocks := invRe.FindAllStringSubmatch(block, -1)

		for _, inv := range invBlocks {
			invBlock := inv[1]
			stockName := extractField(invBlock, "STOCKITEMNAME")
			if stockName == "" {
				continue
			}
			rate := extractField(invBlock, "RATE")
			discount := extractField(invBlock, "DISCOUNT")
			amountStr := extractField(invBlock, "AMOUNT")
			qtyStr := extractField(invBlock, "ACTUALQTY")

			amount, _ := strconv.ParseFloat(strings.ReplaceAll(amountStr, ",", ""), 64)
			qtyMatch := qtyNumRe.FindString(strings.TrimSpace(qtyStr))
			qty, _ := strconv.ParseFloat(qtyMatch, 64)

			rows = append(rows, cache.Row{
				Date:     dateStr,
				Invoice:  vchNum,
				Party:    html.UnescapeString(party),
				Stock:    html.UnescapeString(stockName),
				Rate:     rate,
				Discount: strings.TrimSpace(discount),
				Qty:      qty,
				Amount:   amount,
			})
		}
	}
	return rows
}

func extractField(block, tag string) string {
	re := regexp.MustCompile(`<` + tag + `[^>]*>(.*?)</` + tag + `>`)
	m := re.FindStringSubmatch(block)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[1])
}
