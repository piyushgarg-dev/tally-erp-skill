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

	c := tally.NewClient(g.URL(), g.Timeout)
	resp, err := c.Post(context.Background(), body)
	if err != nil {
		return reportTransportError(stderr, err)
	}
	out := resp
	if g.Pretty {
		out = pretty(resp)
	}
	fmt.Fprintln(stdout, out)
	return statusToExit(stderr, resp)
}
