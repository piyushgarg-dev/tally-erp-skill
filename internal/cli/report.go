package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/piyushgarg/tally-skill/internal/tally"
)

type repeatableVar struct{ vals map[string]string }

func (r *repeatableVar) String() string { return "" }
func (r *repeatableVar) Set(v string) error {
	if r.vals == nil {
		r.vals = map[string]string{}
	}
	parts := strings.SplitN(v, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected KEY=VALUE, got %q", v)
	}
	r.vals[parts[0]] = parts[1]
	return nil
}

func RunReport(args []string) int { return runReportWithIO(args, os.Stdout, os.Stderr) }

func runReportWithIO(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	g := registerGlobals(fs)
	id := fs.String("id", "", "Report name (e.g. \"Day Book\")")
	from := fs.String("from", "", "From date (YYYY-MM-DD)")
	to := fs.String("to", "", "To date (YYYY-MM-DD)")
	ledger := fs.String("ledger", "", "Ledger name for ledger reports")
	group := fs.String("group", "", "Group name for group reports")
	explode := fs.Bool("explode", false, "Set EXPLODEFLAG=Yes")
	voucherType := fs.String("voucher-type", "", "Filter by voucher type (e.g. Sales, Purchase)")
	filter := fs.String("filter", "", "TDL filter expression")
	chunk := fs.String("chunk", "", "Date chunking granularity: daily, weekly, or monthly")
	vars := &repeatableVar{}
	fs.Var(vars, "var", "Repeatable; KEY=VALUE for arbitrary STATICVARIABLE")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	if *id == "" {
		fmt.Fprintln(stderr, "tally report: --id is required")
		return ExitUsage
	}

	fromTally, err := isoToTallyDate(*from)
	if err != nil {
		fmt.Fprintf(stderr, "tally report: --from: %v\n", err)
		return ExitUsage
	}
	toTally, err := isoToTallyDate(*to)
	if err != nil {
		fmt.Fprintf(stderr, "tally report: --to: %v\n", err)
		return ExitUsage
	}

	extra := map[string]string{}
	for k, v := range vars.vals {
		extra[k] = v
	}
	if *ledger != "" {
		extra["LedgerName"] = *ledger
	}
	if *group != "" {
		extra["GroupName"] = *group
	}
	if *explode {
		extra["EXPLODEFLAG"] = "Yes"
	}

	tdl := tally.BuildReportTDL(tally.ReportFilter{
		ReportID:    *id,
		VoucherType: *voucherType,
		Filter:      *filter,
	})

	c := tally.NewClient(g.URL(), g.Timeout)

	if *chunk != "" && fromTally != "" && toTally != "" {
		chunks, err := tally.ChunkDates(fromTally, toTally, *chunk)
		if err != nil {
			fmt.Fprintf(stderr, "tally report: --chunk: %v\n", err)
			return ExitUsage
		}

		var responses []string
		allFailed := true
		for i, ch := range chunks {
			fmt.Fprintf(stderr, "tally: fetching chunk %d/%d (%s to %s)...\n", i+1, len(chunks), ch[0], ch[1])
			body, err := tally.BuildReport(tally.ReportRequest{
				ID:       *id,
				Company:  g.Company,
				FromDate: ch[0],
				ToDate:   ch[1],
				Vars:     extra,
				TDL:      tdl,
			})
			if err != nil {
				fmt.Fprintf(stderr, "tally report: chunk %d: %v\n", i+1, err)
				continue
			}
			resp, err := c.Post(context.Background(), body)
			if err != nil {
				fmt.Fprintf(stderr, "tally: chunk %d failed: %v\n", i+1, err)
				continue
			}
			responses = append(responses, resp)
			allFailed = false
		}
		if allFailed {
			fmt.Fprintln(stderr, "tally: all chunks failed")
			return ExitConnect
		}
		merged := tally.MergeResponses(responses)
		out := renderOutput(merged, g.Format, g.Pretty)
		fmt.Fprintln(stdout, out)
		return ExitOK
	}

	body, err := tally.BuildReport(tally.ReportRequest{
		ID:       *id,
		Company:  g.Company,
		FromDate: fromTally,
		ToDate:   toTally,
		Vars:     extra,
		TDL:      tdl,
	})
	if err != nil {
		fmt.Fprintf(stderr, "tally report: %v\n", err)
		return ExitUsage
	}

	resp, err := c.Post(context.Background(), body)
	if err != nil {
		return reportTransportError(stderr, err)
	}
	out := renderOutput(resp, g.Format, g.Pretty)
	fmt.Fprintln(stdout, out)
	return statusToExit(stderr, resp)
}
