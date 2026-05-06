package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPingSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER><BODY><DATA/></BODY></ENVELOPE>`))
	}))
	defer srv.Close()

	code := runPingWithIO([]string{"--host", hostOf(srv.URL), "--port", portOf(srv.URL)},
		&bytes.Buffer{}, &bytes.Buffer{})
	if code != ExitOK {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestPingConnectError(t *testing.T) {
	// Use a port nothing listens on
	code := runPingWithIO([]string{"--host", "127.0.0.1", "--port", "1", "--timeout", "300ms"},
		&bytes.Buffer{}, &bytes.Buffer{})
	if code != ExitConnect && code != ExitTimeout {
		t.Fatalf("expected connect or timeout, got %d", code)
	}
}
