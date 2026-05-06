package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestObjectBuildsAndPosts(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		got = string(b)
		_, _ = w.Write([]byte(`<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER><BODY><DATA/></BODY></ENVELOPE>`))
	}))
	defer srv.Close()

	out := &bytes.Buffer{}
	errb := &bytes.Buffer{}
	code := runObjectWithIO([]string{
		"--host", hostOf(srv.URL), "--port", portOf(srv.URL),
		"--company", "ABC",
		"--subtype", "Ledger",
		"--id", "Customer ABC",
		"--fetch", "Name,Parent,ClosingBalance",
	}, out, errb)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, errb.String())
	}
	if !strings.Contains(got, "<SUBTYPE>Ledger</SUBTYPE>") {
		t.Errorf("server received: %s", got)
	}
	if !strings.Contains(got, `<ID TYPE="Name">Customer ABC</ID>`) {
		t.Errorf("server received: %s", got)
	}
}
