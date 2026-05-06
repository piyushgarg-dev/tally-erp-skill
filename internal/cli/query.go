package cli

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/piyushgarg/tally-skill/internal/cache"
)

func RunQuery(args []string) int { return runQueryWithIO(args, os.Stdout, os.Stderr) }

func runQueryWithIO(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("query", flag.ContinueOnError)
	fs.SetOutput(stderr)
	company := fs.String("company", "", "Company name (matches cache directory)")
	dataType := fs.String("type", "sales", "Data type: sales, purchase")
	party := fs.String("party", "", "Party name filter (supports * glob)")
	stock := fs.String("stock", "", "Stock item filter (supports * glob)")
	from := fs.String("from", "", "From date filter (YYYY-MM-DD)")
	to := fs.String("to", "", "To date filter (YYYY-MM-DD)")
	cacheDir := fs.String("cache-dir", ".tally-cache", "Cache directory path")
	format := fs.String("format", "csv", "Output format: csv, json, summary")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	if *company == "" {
		fmt.Fprintln(stderr, "tally query: --company is required")
		return ExitUsage
	}

	// Convert ISO dates to dd-mm-yyyy for comparison
	fromDate := ""
	toDate := ""
	if *from != "" {
		fromDate = isoToDDMMYYYY(*from)
	}
	if *to != "" {
		toDate = isoToDDMMYYYY(*to)
	}

	result, err := cache.Query(cache.QueryOpts{
		Company:  *company,
		Type:     *dataType,
		Party:    *party,
		Stock:    *stock,
		From:     fromDate,
		To:       toDate,
		CacheDir: *cacheDir,
	})
	if err != nil {
		fmt.Fprintf(stderr, "tally query: %v\n", err)
		return ExitTallyFailure
	}

	if len(result.Rows) == 0 {
		fmt.Fprintln(stderr, "tally query: no matching rows")
		return ExitOK
	}

	switch strings.ToLower(*format) {
	case "json":
		outputJSON(stdout, result.Rows)
	case "summary":
		outputSummary(stdout, result.Rows, *party != "", *stock != "")
	default:
		outputCSV(stdout, result.Rows)
	}
	return ExitOK
}

func isoToDDMMYYYY(iso string) string {
	// 2025-04-01 -> 01-04-2025
	parts := strings.Split(iso, "-")
	if len(parts) != 3 {
		return iso
	}
	return parts[2] + "-" + parts[1] + "-" + parts[0]
}

func outputCSV(w io.Writer, rows []cache.Row) {
	cw := csv.NewWriter(w)
	cw.Write([]string{"Date", "Invoice", "Party", "StockItem", "Rate", "Discount%", "Qty", "Amount"})
	for _, r := range rows {
		cw.Write([]string{
			r.Date, r.Invoice, r.Party, r.Stock, r.Rate, r.Discount,
			fmt.Sprintf("%.1f", r.Qty),
			fmt.Sprintf("%.2f", r.Amount),
		})
	}
	cw.Flush()
}

func outputJSON(w io.Writer, rows []cache.Row) {
	type jsonRow struct {
		Date     string  `json:"date"`
		Invoice  string  `json:"invoice"`
		Party    string  `json:"party"`
		Stock    string  `json:"stock_item"`
		Rate     string  `json:"rate"`
		Discount string  `json:"discount"`
		Qty      float64 `json:"qty"`
		Amount   float64 `json:"amount"`
	}
	out := make([]jsonRow, len(rows))
	for i, r := range rows {
		out[i] = jsonRow{r.Date, r.Invoice, r.Party, r.Stock, r.Rate, r.Discount, r.Qty, r.Amount}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	enc.Encode(out)
}

func outputSummary(w io.Writer, rows []cache.Row, groupByParty, groupByStock bool) {
	type summary struct {
		Key    string
		Items  int
		Qty    float64
		Amount float64
	}

	grouped := map[string]*summary{}
	for _, r := range rows {
		var key string
		switch {
		case groupByStock:
			key = r.Stock
		case groupByParty:
			key = r.Party
		default:
			key = r.Party
		}
		s, ok := grouped[key]
		if !ok {
			s = &summary{Key: key}
			grouped[key] = s
		}
		s.Items++
		s.Qty += r.Qty
		s.Amount += r.Amount
	}

	sorted := make([]*summary, 0, len(grouped))
	for _, s := range grouped {
		sorted = append(sorted, s)
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Amount > sorted[j].Amount })

	fmt.Fprintf(w, "%-50s %6s %10s %14s\n", "Name", "Items", "Qty", "Amount")
	fmt.Fprintf(w, "%s\n", strings.Repeat("-", 84))
	totalItems := 0
	totalQty := 0.0
	totalAmt := 0.0
	for _, s := range sorted {
		name := s.Key
		if len(name) > 50 {
			name = name[:47] + "..."
		}
		fmt.Fprintf(w, "%-50s %6d %10.1f %14.2f\n", name, s.Items, s.Qty, s.Amount)
		totalItems += s.Items
		totalQty += s.Qty
		totalAmt += s.Amount
	}
	fmt.Fprintf(w, "%s\n", strings.Repeat("-", 84))
	fmt.Fprintf(w, "%-50s %6d %10.1f %14.2f\n", "TOTAL", totalItems, totalQty, totalAmt)
}
