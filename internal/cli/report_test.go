package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReportConvertsISODates(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		got = string(b)
		_, _ = w.Write([]byte(`<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER><BODY><DATA/></BODY></ENVELOPE>`))
	}))
	defer srv.Close()

	code := runReportWithIO([]string{
		"--host", hostOf(srv.URL), "--port", portOf(srv.URL),
		"--company", "ABC",
		"--id", "Day Book",
		"--from", "2026-04-01",
		"--to", "2026-04-30",
	}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(got, "<SVFROMDATE>20260401</SVFROMDATE>") {
		t.Errorf("from-date not converted: %s", got)
	}
	if !strings.Contains(got, "<SVTODATE>20260430</SVTODATE>") {
		t.Errorf("to-date not converted: %s", got)
	}
}

func TestReportLedgerVar(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		got = string(b)
		_, _ = w.Write([]byte(`<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER><BODY><DATA/></BODY></ENVELOPE>`))
	}))
	defer srv.Close()

	runReportWithIO([]string{
		"--host", hostOf(srv.URL), "--port", portOf(srv.URL),
		"--id", "Ledger",
		"--ledger", "Customer ABC",
	}, &bytes.Buffer{}, &bytes.Buffer{})
	if !strings.Contains(got, "<LedgerName>Customer ABC</LedgerName>") {
		t.Errorf("expected LedgerName var: %s", got)
	}
}
