package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRawPipesStdinToTally(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		got = string(b)
		_, _ = w.Write([]byte(`<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER><BODY><DATA>ok</DATA></BODY></ENVELOPE>`))
	}))
	defer srv.Close()

	stdin := strings.NewReader(`<ENVELOPE/>`)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := runRawWithIO([]string{"--host", hostOf(srv.URL), "--port", portOf(srv.URL)}, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, stderr.String())
	}
	if got != `<ENVELOPE/>` {
		t.Errorf("server got %q", got)
	}
	if !strings.Contains(stdout.String(), "<STATUS>1</STATUS>") {
		t.Errorf("stdout=%s", stdout.String())
	}
}
