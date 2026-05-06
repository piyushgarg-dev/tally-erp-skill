package tally

import (
	"strings"
	"testing"
)

func TestMergeResponsesSingle(t *testing.T) {
	resp := `<ENVELOPE><HEADER><VERSION>1</VERSION><STATUS>1</STATUS></HEADER><BODY><DATA><TALLYMESSAGE><VOUCHER>v1</VOUCHER></TALLYMESSAGE></DATA></BODY></ENVELOPE>`
	merged := MergeResponses([]string{resp})
	if !strings.Contains(merged, "<VOUCHER>v1</VOUCHER>") {
		t.Error("merged should contain voucher v1")
	}
	if !strings.Contains(merged, "<STATUS>1</STATUS>") {
		t.Error("merged should have status 1")
	}
}

func TestMergeResponsesMultiple(t *testing.T) {
	r1 := `<ENVELOPE><HEADER><VERSION>1</VERSION><STATUS>1</STATUS></HEADER><BODY><DATA><TALLYMESSAGE><VOUCHER>v1</VOUCHER></TALLYMESSAGE></DATA></BODY></ENVELOPE>`
	r2 := `<ENVELOPE><HEADER><VERSION>1</VERSION><STATUS>1</STATUS></HEADER><BODY><DATA><TALLYMESSAGE><VOUCHER>v2</VOUCHER></TALLYMESSAGE></DATA></BODY></ENVELOPE>`
	merged := MergeResponses([]string{r1, r2})
	if !strings.Contains(merged, "<VOUCHER>v1</VOUCHER>") {
		t.Error("merged should contain v1")
	}
	if !strings.Contains(merged, "<VOUCHER>v2</VOUCHER>") {
		t.Error("merged should contain v2")
	}
}

func TestMergeResponsesSkipsErrors(t *testing.T) {
	good := `<ENVELOPE><HEADER><VERSION>1</VERSION><STATUS>1</STATUS></HEADER><BODY><DATA><TALLYMESSAGE><VOUCHER>v1</VOUCHER></TALLYMESSAGE></DATA></BODY></ENVELOPE>`
	bad := `<ENVELOPE><HEADER><VERSION>1</VERSION><STATUS>0</STATUS></HEADER><BODY><DATA>Error occurred</DATA></BODY></ENVELOPE>`
	merged := MergeResponses([]string{good, bad})
	if !strings.Contains(merged, "<VOUCHER>v1</VOUCHER>") {
		t.Error("merged should contain v1")
	}
	if strings.Contains(merged, "Error occurred") {
		t.Error("merged should not contain error response content")
	}
}

func TestMergeResponsesEmpty(t *testing.T) {
	merged := MergeResponses([]string{})
	if !strings.Contains(merged, "<DATA></DATA>") {
		t.Error("empty merge should produce empty DATA")
	}
}

func TestMergeResponsesSkipsEmptyStrings(t *testing.T) {
	good := `<ENVELOPE><HEADER><VERSION>1</VERSION><STATUS>1</STATUS></HEADER><BODY><DATA><TALLYMESSAGE><VOUCHER>v1</VOUCHER></TALLYMESSAGE></DATA></BODY></ENVELOPE>`
	merged := MergeResponses([]string{"", good, ""})
	if !strings.Contains(merged, "<VOUCHER>v1</VOUCHER>") {
		t.Error("merged should contain v1")
	}
}

func TestMergeResponsesCollectionFormat(t *testing.T) {
	resp := `<ENVELOPE><HEADER><VERSION>1</VERSION><STATUS>1</STATUS></HEADER><BODY><DESC></DESC><DATA><COLLECTION><VOUCHER NAME="a">data1</VOUCHER><VOUCHER NAME="b">data2</VOUCHER></COLLECTION></DATA></BODY></ENVELOPE>`
	merged := MergeResponses([]string{resp})
	if !strings.Contains(merged, "data1") {
		t.Error("merged should contain data1")
	}
	if !strings.Contains(merged, "data2") {
		t.Error("merged should contain data2")
	}
}
