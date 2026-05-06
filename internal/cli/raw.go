package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/piyushgarg/tally-skill/internal/tally"
)

func RunRaw(args []string) int {
	return runRawWithIO(args, os.Stdin, os.Stdout, os.Stderr)
}

func runRawWithIO(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("raw", flag.ContinueOnError)
	fs.SetOutput(stderr)
	g := registerGlobals(fs)
	file := fs.String("file", "", "Read XML from file instead of stdin")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}

	var body []byte
	var err error
	if *file != "" {
		body, err = os.ReadFile(*file)
	} else {
		body, err = io.ReadAll(stdin)
	}
	if err != nil {
		fmt.Fprintf(stderr, "tally: read input: %v\n", err)
		return ExitUsage
	}
	if len(body) == 0 {
		fmt.Fprintln(stderr, "tally: empty input")
		return ExitUsage
	}

	c := tally.NewClient(g.URL(), g.Timeout)
	resp, err := c.Post(context.Background(), string(body))
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