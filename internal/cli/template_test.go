package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTallyOKServer(t *testing.T, capture *string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		if capture != nil {
			*capture = string(b)
		}
		_, _ = w.Write([]byte(`<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER><BODY><DATA>ok</DATA></BODY></ENVELOPE>`))
	}))
	return srv
}

func writeTemplate(t *testing.T, dir, rel, body string) {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTemplateLoadsAndSubstitutes(t *testing.T) {
	tmp := t.TempDir()
	writeTemplate(t, tmp, "foo.xml", `<ENV><CO>{{COMPANY}}</CO><F>{{FROMDATE}}</F></ENV>`)

	var got string
	srv := newTallyOKServer(t, &got)
	defer srv.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := runTemplateWithIO([]string{
		"--host", hostOf(srv.URL), "--port", portOf(srv.URL),
		"--templates-dir", tmp,
		"--name", "foo",
		"--company", "X",
		"--from", "2026-04-01",
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, stderr.String())
	}
	if got != `<ENV><CO>X</CO><F>20260401</F></ENV>` {
		t.Errorf("server got %q", got)
	}
}

func TestTemplateUnknownName(t *testing.T) {
	tmp := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := runTemplateWithIO([]string{
		"--templates-dir", tmp,
		"--name", "missing",
	}, stdout, stderr)
	if code != ExitUsage {
		t.Fatalf("exit %d, want %d, stderr=%s", code, ExitUsage, stderr.String())
	}
	if stderr.Len() == 0 {
		t.Error("expected stderr message")
	}
}

func TestTemplateExtensionOptional(t *testing.T) {
	tmp := t.TempDir()
	writeTemplate(t, tmp, "bar.xml", `<X>{{COMPANY}}</X>`)

	for _, name := range []string{"bar", "bar.xml"} {
		var got string
		srv := newTallyOKServer(t, &got)
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		code := runTemplateWithIO([]string{
			"--host", hostOf(srv.URL), "--port", portOf(srv.URL),
			"--templates-dir", tmp,
			"--name", name,
			"--company", "C",
		}, stdout, stderr)
		srv.Close()
		if code != 0 {
			t.Fatalf("name=%s exit %d, stderr=%s", name, code, stderr.String())
		}
		if got != `<X>C</X>` {
			t.Errorf("name=%s server got %q", name, got)
		}
	}
}

func TestTemplateRepeatableVar(t *testing.T) {
	tmp := t.TempDir()
	writeTemplate(t, tmp, "v.xml", `<R>{{FOO}}-{{BAZ}}</R>`)

	var got string
	srv := newTallyOKServer(t, &got)
	defer srv.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := runTemplateWithIO([]string{
		"--host", hostOf(srv.URL), "--port", portOf(srv.URL),
		"--templates-dir", tmp,
		"--name", "v",
		"--var", "FOO=bar",
		"--var", "BAZ=qux",
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, stderr.String())
	}
	if got != `<R>bar-qux</R>` {
		t.Errorf("server got %q", got)
	}
}

func TestTemplateXMLEscaping(t *testing.T) {
	tmp := t.TempDir()
	writeTemplate(t, tmp, "e.xml", `<N>{{COMPANY}}</N>`)

	var got string
	srv := newTallyOKServer(t, &got)
	defer srv.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := runTemplateWithIO([]string{
		"--host", hostOf(srv.URL), "--port", portOf(srv.URL),
		"--templates-dir", tmp,
		"--name", "e",
		"--company", `M&S "Ltd"`,
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(got, "M&amp;S &#34;Ltd&#34;") {
		t.Errorf("expected XML-escaped value, server got %q", got)
	}
}

func TestTemplateUnfilledPlaceholderWarning(t *testing.T) {
	tmp := t.TempDir()
	writeTemplate(t, tmp, "w.xml", `<X>{{COMPANY}}</X><Y>{{MISSING}}</Y>`)

	srv := newTallyOKServer(t, nil)
	defer srv.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := runTemplateWithIO([]string{
		"--host", hostOf(srv.URL), "--port", portOf(srv.URL),
		"--templates-dir", tmp,
		"--name", "w",
		"--company", "X",
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "{{MISSING}}") {
		t.Errorf("expected warning about unfilled placeholder, stderr=%s", stderr.String())
	}
}

func TestTemplateIsoDateConverted(t *testing.T) {
	tmp := t.TempDir()
	writeTemplate(t, tmp, "d.xml", `<D>{{FROMDATE}}|{{TODATE}}</D>`)

	var got string
	srv := newTallyOKServer(t, &got)
	defer srv.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := runTemplateWithIO([]string{
		"--host", hostOf(srv.URL), "--port", portOf(srv.URL),
		"--templates-dir", tmp,
		"--name", "d",
		"--from", "2026-04-01",
		"--to", "2026-04-30",
	}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(got, "20260401") || !strings.Contains(got, "20260430") {
		t.Errorf("server got %q", got)
	}
}
