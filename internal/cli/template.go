package cli

import (
	"bytes"
	"context"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/piyushgarg/tally-skill/internal/tally"
)

type templateVar struct{ vals map[string]string }

func (r *templateVar) String() string { return "" }
func (r *templateVar) Set(v string) error {
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

func RunTemplate(args []string) int { return runTemplateWithIO(args, os.Stdout, os.Stderr) }

func runTemplateWithIO(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("template", flag.ContinueOnError)
	fs.SetOutput(stderr)
	g := registerGlobals(fs)
	name := fs.String("name", "", "Template path under templates/ (with or without .xml)")
	templatesDir := fs.String("templates-dir", "", "Override templates directory")
	from := fs.String("from", "", "From date (YYYY-MM-DD) -> {{FROMDATE}}")
	to := fs.String("to", "", "To date (YYYY-MM-DD) -> {{TODATE}}")
	ledger := fs.String("ledger", "", "{{LEDGER}}")
	group := fs.String("group", "", "{{GROUP}}")
	stockitem := fs.String("stockitem", "", "{{STOCKITEM}}")
	vouchertype := fs.String("vouchertype", "", "{{VOUCHERTYPE}}")
	vouchernumber := fs.String("vouchernumber", "", "{{VOUCHERNUMBER}}")
	chunk := fs.String("chunk", "", "Date chunking granularity: daily, weekly, or monthly")
	vars := &templateVar{}
	fs.Var(vars, "var", "Repeatable; KEY=VALUE for arbitrary {{KEY}} placeholder")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	if *name == "" {
		fmt.Fprintln(stderr, "tally template: --name is required")
		return ExitUsage
	}

	dir, err := resolveTemplatesDir(*templatesDir)
	if err != nil {
		fmt.Fprintf(stderr, "tally template: %v\n", err)
		return ExitUsage
	}

	body, err := loadTemplate(dir, *name)
	if err != nil {
		fmt.Fprintf(stderr, "tally template: %v\n", err)
		return ExitUsage
	}

	fromTally, err := isoToTallyDate(*from)
	if err != nil {
		fmt.Fprintf(stderr, "tally template: --from: %v\n", err)
		return ExitUsage
	}
	toTally, err := isoToTallyDate(*to)
	if err != nil {
		fmt.Fprintf(stderr, "tally template: --to: %v\n", err)
		return ExitUsage
	}

	subs := map[string]string{}
	for k, v := range vars.vals {
		subs[k] = v
	}
	if g.Company != "" {
		subs["COMPANY"] = g.Company
	}
	if fromTally != "" {
		subs["FROMDATE"] = fromTally
	}
	if toTally != "" {
		subs["TODATE"] = toTally
	}
	if *ledger != "" {
		subs["LEDGER"] = *ledger
	}
	if *group != "" {
		subs["GROUP"] = *group
	}
	if *stockitem != "" {
		subs["STOCKITEM"] = *stockitem
	}
	if *vouchertype != "" {
		subs["VOUCHERTYPE"] = *vouchertype
	}
	if *vouchernumber != "" {
		subs["VOUCHERNUMBER"] = *vouchernumber
	}

	c := tally.NewClient(g.URL(), g.Timeout)

	if *chunk != "" && fromTally != "" && toTally != "" {
		chunks, err := tally.ChunkDates(fromTally, toTally, *chunk)
		if err != nil {
			fmt.Fprintf(stderr, "tally template: --chunk: %v\n", err)
			return ExitUsage
		}

		var responses []string
		allFailed := true
		for i, ch := range chunks {
			fmt.Fprintf(stderr, "tally: fetching chunk %d/%d (%s to %s)...\n", i+1, len(chunks), ch[0], ch[1])
			chunkSubs := make(map[string]string, len(subs))
			for k, v := range subs {
				chunkSubs[k] = v
			}
			chunkSubs["FROMDATE"] = ch[0]
			chunkSubs["TODATE"] = ch[1]

			final := substitute(body, chunkSubs)
			warnUnfilledPlaceholders(final, stderr)

			resp, err := c.Post(context.Background(), string(final))
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

	final := substitute(body, subs)
	warnUnfilledPlaceholders(final, stderr)

	resp, err := c.Post(context.Background(), string(final))
	if err != nil {
		return reportTransportError(stderr, err)
	}
	out := renderOutput(resp, g.Format, g.Pretty)
	fmt.Fprintln(stdout, out)
	return statusToExit(stderr, resp)
}

func resolveTemplatesDir(override string) (string, error) {
	if override != "" {
		if isDir(override) {
			return override, nil
		}
		return "", fmt.Errorf("templates dir not found: %s", override)
	}
	exe, err := os.Executable()
	if err == nil {
		if real, err2 := filepath.EvalSymlinks(exe); err2 == nil {
			exe = real
		}
		candidate := filepath.Join(filepath.Dir(exe), "..", "templates")
		if isDir(candidate) {
			return candidate, nil
		}
	}
	if isDir("templates") {
		return "templates", nil
	}
	return "", fmt.Errorf("no templates directory found (tried <exe>/../templates and ./templates); pass --templates-dir")
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func loadTemplate(dir, name string) ([]byte, error) {
	candidates := []string{name}
	if !strings.HasSuffix(strings.ToLower(name), ".xml") {
		candidates = append(candidates, name+".xml")
	}
	for _, c := range candidates {
		full := filepath.Join(dir, c)
		if data, err := os.ReadFile(full); err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("template not found: %s (in %s)", name, dir)
}

func substitute(in []byte, vars map[string]string) []byte {
	out := in
	for k, v := range vars {
		var escaped bytes.Buffer
		xml.EscapeText(&escaped, []byte(v))
		out = bytes.ReplaceAll(out, []byte("{{"+k+"}}"), escaped.Bytes())
	}
	return out
}

var placeholderRe = regexp.MustCompile(`\{\{[A-Za-z_][A-Za-z0-9_]*\}\}`)

func warnUnfilledPlaceholders(data []byte, stderr io.Writer) {
	matches := placeholderRe.FindAll(data, -1)
	seen := map[string]bool{}
	for _, m := range matches {
		s := string(m)
		if !seen[s] {
			seen[s] = true
			fmt.Fprintf(stderr, "tally template: warning: unfilled placeholder %s\n", s)
		}
	}
}
