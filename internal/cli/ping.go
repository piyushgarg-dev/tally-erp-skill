package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/piyushgarg/tally-skill/internal/tally"
)

func RunPing(args []string) int { return runPingWithIO(args, os.Stdout, os.Stderr) }

func runPingWithIO(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ping", flag.ContinueOnError)
	fs.SetOutput(stderr)
	g := registerGlobals(fs)
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}

	body, err := tally.BuildCollection(tally.CollectionRequest{ID: "List of Companies"})
	if err != nil {
		fmt.Fprintf(stderr, "tally ping: %v\n", err)
		return ExitUsage
	}

	c := tally.NewClient(g.URL(), g.Timeout)
	resp, err := c.Post(context.Background(), body)
	if err != nil {
		return reportTransportError(stderr, err)
	}
	st := tally.ParseStatus(resp)
	if !st.Parsed {
		fmt.Fprintln(stderr, "tally: response not valid XML")
		return ExitBadResponse
	}
	if !st.Success() {
		fmt.Fprintf(stderr, "tally: %s\n", st.Message)
		return ExitTallyFailure
	}
	fmt.Fprintln(stdout, "tally: ok")
	return ExitOK
}
