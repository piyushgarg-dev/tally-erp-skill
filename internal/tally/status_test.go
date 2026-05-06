package tally

import "testing"

func TestParseStatusSuccess(t *testing.T) {
	xml := `<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER><BODY><DATA>...</DATA></BODY></ENVELOPE>`
	r := ParseStatus(xml)
	if !r.Success() {
		t.Fatal("expected success")
	}
}

func TestParseStatusFailureWithLineError(t *testing.T) {
	xml := `<ENVELOPE><HEADER><STATUS>0</STATUS></HEADER><BODY><DATA><LINEERROR>Voucher totals do not match</LINEERROR></DATA></BODY></ENVELOPE>`
	r := ParseStatus(xml)
	if r.Success() {
		t.Fatal("expected failure")
	}
	if r.Message != "Voucher totals do not match" {
		t.Errorf("got %q", r.Message)
	}
}

func TestParseStatusFailurePlainText(t *testing.T) {
	xml := `<ENVELOPE><HEADER><STATUS>0</STATUS></HEADER><BODY><DATA>DESC not found</DATA></BODY></ENVELOPE>`
	r := ParseStatus(xml)
	if r.Success() {
		t.Fatal("expected failure")
	}
	if r.Message != "DESC not found" {
		t.Errorf("got %q", r.Message)
	}
}

func TestParseStatusUnparseable(t *testing.T) {
	r := ParseStatus(`not xml at all`)
	if r.Parsed {
		t.Fatal("expected Parsed=false")
	}
}

func TestParseStatusFailureWithStatusList(t *testing.T) {
	x := `<ENVELOPE><HEADER><STATUS>0</STATUS></HEADER><BODY><DATA><STATUS.LIST><STATUS><CODE>123</CODE><DESC>Invalid Request</DESC></STATUS></STATUS.LIST></DATA></BODY></ENVELOPE>`
	r := ParseStatus(x)
	if r.Success() {
		t.Fatal("expected failure")
	}
	if r.Message != "[123] Invalid Request" {
		t.Errorf("got %q", r.Message)
	}
}

func TestParseStatusFailureStatusListDescOnly(t *testing.T) {
	x := `<ENVELOPE><HEADER><STATUS>0</STATUS></HEADER><BODY><DATA><STATUS.LIST><STATUS><DESC>Bad envelope</DESC></STATUS></STATUS.LIST></DATA></BODY></ENVELOPE>`
	r := ParseStatus(x)
	if r.Success() {
		t.Fatal("expected failure")
	}
	if r.Message != "Bad envelope" {
		t.Errorf("got %q", r.Message)
	}
}
