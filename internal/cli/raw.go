package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
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

func reportTransportError(stderr io.Writer, err error) int {
	var nerr net.Error
	if errors.As(err, &nerr) && nerr.Timeout() {
		fmt.Fprintf(stderr, "tally: timeout: %v\n", err)
		return ExitTimeout
	}
	fmt.Fprintf(stderr, "tally: cannot reach server: %v\n", err)
	return ExitConnect
}

func statusToExit(stderr io.Writer, resp string) int {
	st := tally.ParseStatus(resp)
	if !st.Parsed {
		fmt.Fprintln(stderr, "tally: response not valid XML")
		return ExitBadResponse
	}
	if st.Code == 1 {
		return ExitOK
	}
	fmt.Fprintf(stderr, "tally: failure: %s\n", st.Message)
	return ExitTallyFailure
}