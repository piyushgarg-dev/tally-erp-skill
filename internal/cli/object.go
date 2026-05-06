package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/piyushgarg/tally-skill/internal/tally"
)

func RunObject(args []string) int { return runObjectWithIO(args, os.Stdout, os.Stderr) }

func runObjectWithIO(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("object", flag.ContinueOnError)
	fs.SetOutput(stderr)
	g := registerGlobals(fs)
	subtype := fs.String("subtype", "", "Object subtype (Ledger, Group, StockItem, Voucher, ...)")
	id := fs.String("id", "", "Object identifier")
	idType := fs.String("id-type", "Name", "Type of the identifier")
	fetch := fs.String("fetch", "", "Comma-separated FETCH fields")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	if *subtype == "" || *id == "" {
		fmt.Fprintln(stderr, "tally object: --subtype and --id are required")
		return ExitUsage
	}

	body, err := tally.BuildObject(tally.ObjectRequest{
		Subtype: *subtype,
		IDType:  *idType,
		ID:      *id,
		Company: g.Company,
		Fetch:   splitCSV(*fetch),
	})
	if err != nil {
		fmt.Fprintf(stderr, "tally object: %v\n", err)
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
