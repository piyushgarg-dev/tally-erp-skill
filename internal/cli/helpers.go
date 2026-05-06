package cli

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/piyushgarg/tally-skill/internal/tally"
)

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

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func isoToTallyDate(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return "", fmt.Errorf("expected YYYY-MM-DD, got %q", s)
	}
	return t.Format("20060102"), nil
}
