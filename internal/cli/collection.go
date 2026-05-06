package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/piyushgarg/tally-skill/internal/tally"
)

func RunCollection(args []string) int { return runCollectionWithIO(args, os.Stdout, os.Stderr) }

func runCollectionWithIO(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("collection", flag.ContinueOnError)
	fs.SetOutput(stderr)
	g := registerGlobals(fs)
	id := fs.String("id", "", "Collection name (e.g. \"List of Ledgers\")")
	fetch := fs.String("fetch", "", "Comma-separated FETCH fields")
	parent := fs.String("parent", "", "Filter to children of this group (TDL CHILD OF)")
	fields := fs.String("fields", "", "Comma-separated NATIVEMETHOD fields to return")
	filter := fs.String("filter", "", "TDL filter expression (e.g. \"$ClosingBalance > 10000\")")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	if *id == "" {
		fmt.Fprintln(stderr, "tally collection: --id is required")
		return ExitUsage
	}

	tdl := tally.BuildCollectionTDL(tally.CollectionFilter{
		CollectionID: *id,
		ChildOf:      *parent,
		Fields:       splitCSV(*fields),
		Filter:       *filter,
	})

	body, err := tally.BuildCollection(tally.CollectionRequest{
		ID:      *id,
		Company: g.Company,
		Fetch:   splitCSV(*fetch),
		TDL:     tdl,
	})
	if err != nil {
		fmt.Fprintf(stderr, "tally collection: %v\n", err)
		return ExitUsage
	}

	c := tally.NewClient(g.URL(), g.Timeout)
	resp, err := c.Post(context.Background(), body)
	if err != nil {
		return reportTransportError(stderr, err)
	}
	out := renderOutput(resp, g.Format, g.Pretty)
	fmt.Fprintln(stdout, out)
	return statusToExit(stderr, resp)
}
