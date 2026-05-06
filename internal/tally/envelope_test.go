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

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q\n--- output ---\n%s", needle, haystack)
	}
}
