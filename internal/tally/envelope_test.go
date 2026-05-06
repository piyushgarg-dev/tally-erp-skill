package tally

import (
	"strings"
	"testing"
)

func TestBuildObjectEnvelope(t *testing.T) {
	req := ObjectRequest{
		Subtype: "Ledger",
		IDType:  "Name",
		ID:      "Customer ABC",
		Company: "ABC Company Ltd",
		Fetch:   []string{"Name", "Parent", "ClosingBalance"},
	}
	got, err := BuildObject(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mustContain(t, got, `<TALLYREQUEST>Export</TALLYREQUEST>`)
	mustContain(t, got, `<TYPE>Object</TYPE>`)
	mustContain(t, got, `<SUBTYPE>Ledger</SUBTYPE>`)
	mustContain(t, got, `<ID TYPE="Name">Customer ABC</ID>`)
	mustContain(t, got, `<SVCURRENTCOMPANY>ABC Company Ltd</SVCURRENTCOMPANY>`)
	mustContain(t, got, `<FETCH>Name</FETCH>`)
	mustContain(t, got, `<FETCH>ClosingBalance</FETCH>`)
}

func TestObjectEscapesSpecialChars(t *testing.T) {
	req := ObjectRequest{Subtype: "Ledger", IDType: "Name", ID: `M&S "Ltd"`}
	got, err := BuildObject(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mustContain(t, got, `M&amp;S &#34;Ltd&#34;`)
}

func TestBuildCollection(t *testing.T) {
	got, err := BuildCollection(CollectionRequest{
		ID:      "List of Ledgers",
		Company: "ABC",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mustContain(t, got, `<TYPE>Collection</TYPE>`)
	mustContain(t, got, `<ID>List of Ledgers</ID>`)
	mustContain(t, got, `<SVCURRENTCOMPANY>ABC</SVCURRENTCOMPANY>`)
}

func TestBuildReport(t *testing.T) {
	got, err := BuildReport(ReportRequest{
		ID:       "Day Book",
		Company:  "ABC",
		FromDate: "20260401",
		ToDate:   "20260430",
		Vars:     map[string]string{"EXPLODEFLAG": "Yes"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mustContain(t, got, `<TYPE>Data</TYPE>`)
	mustContain(t, got, `<ID>Day Book</ID>`)
	mustContain(t, got, `<SVFROMDATE>20260401</SVFROMDATE>`)
	mustContain(t, got, `<SVTODATE>20260430</SVTODATE>`)
	mustContain(t, got, `<EXPLODEFLAG>Yes</EXPLODEFLAG>`)
}

func TestBuildReportLedgerVar(t *testing.T) {
	got, err := BuildReport(ReportRequest{
		ID:      "Ledger",
		Company: "ABC",
		Vars:    map[string]string{"LedgerName": "Customer ABC"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mustContain(t, got, `<LedgerName>Customer ABC</LedgerName>`)
}

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q\n--- output ---\n%s", needle, haystack)
	}
}
