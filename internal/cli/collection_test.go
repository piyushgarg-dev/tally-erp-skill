package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCollectionBuildsAndPosts(t *testing.T) {
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
	code := runCollectionWithIO([]string{
		"--host", hostOf(srv.URL), "--port", portOf(srv.URL),
		"--company", "ABC",
		"--id", "List of Ledgers",
	}, out, errb)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, errb.String())
	}
	if !strings.Contains(got, "<ID>List of Ledgers</ID>") {
		t.Errorf("server got: %s", got)
	}
}
