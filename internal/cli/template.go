package cli

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

	final := substitute(body, subs)

	c := tally.NewClient(g.URL(), g.Timeout)
	resp, err := c.Post(context.Background(), string(final))
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
		out = bytes.ReplaceAll(out, []byte("{{"+k+"}}"), []byte(v))
	}
	return out
}
