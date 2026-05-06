package tally

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientPostsXML(t *testing.T) {
	var gotBody string
	var gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotCT = r.Header.Get("Content-Type")
		_, _ = w.Write([]byte(`<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER><BODY><DATA>ok</DATA></BODY></ENVELOPE>`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	resp, err := c.Post(context.Background(), `<ENVELOPE/>`)
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}
	if !strings.Contains(resp, "<STATUS>1</STATUS>") {
		t.Errorf("unexpected response: %s", resp)
	}
	if gotBody != `<ENVELOPE/>` {
		t.Errorf("server received body %q", gotBody)
	}
	if !strings.HasPrefix(gotCT, "text/xml") {
		t.Errorf("unexpected content-type %q", gotCT)
	}
}

func TestClientTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 50*time.Millisecond)
	_, err := c.Post(context.Background(), `<ENVELOPE/>`)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
