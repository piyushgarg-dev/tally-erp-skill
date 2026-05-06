package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/piyushgarg/tally-skill/internal/tally"
)

func RunCompanies(args []string) int { return runCompaniesWithIO(args, os.Stdout, os.Stderr) }

func runCompaniesWithIO(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("companies", flag.ContinueOnError)
	fs.SetOutput(stderr)
	g := registerGlobals(fs)
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}

	body, err := tally.BuildCollection(tally.CollectionRequest{ID: "List of Companies"})
	if err != nil {
		fmt.Fprintf(stderr, "tally companies: %v\n", err)
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
