package tally

import "testing"

func TestSanitizeXML_RawControlChars(t *testing.T) {
	in := "<PARENT>\x03 Primary</PARENT>"
	want := "<PARENT> Primary</PARENT>"
	got := SanitizeXML(in)
	if got != want {
		t.Errorf("raw control char: got %q, want %q", got, want)
	}
}

func TestSanitizeXML_DecimalCharRef(t *testing.T) {
	in := "<PARENT>&#4; Primary</PARENT>"
	want := "<PARENT> Primary</PARENT>"
	got := SanitizeXML(in)
	if got != want {
		t.Errorf("decimal char ref: got %q, want %q", got, want)
	}
}

func TestSanitizeXML_HexCharRef(t *testing.T) {
	in := "<PARENT>&#x4; Primary</PARENT>"
	want := "<PARENT> Primary</PARENT>"
	got := SanitizeXML(in)
	if got != want {
		t.Errorf("hex char ref: got %q, want %q", got, want)
	}
}

func TestSanitizeXML_ValidCharRefPreserved(t *testing.T) {
	in := "<X>&#9;tab&#10;newline&#65;A</X>"
	got := SanitizeXML(in)
	if got != in {
		t.Errorf("valid refs should be preserved: got %q, want %q", got, in)
	}
}

func TestSanitizeXML_NoChange(t *testing.T) {
	in := "<ENVELOPE><HEADER>OK</HEADER></ENVELOPE>"
	got := SanitizeXML(in)
	if got != in {
		t.Errorf("clean input should be unchanged: got %q", got)
	}
}

func TestSanitizeXML_MultipleInvalidRefs(t *testing.T) {
	in := "<A>&#3;x&#4;y&#1;z</A>"
	want := "<A>xyz</A>"
	got := SanitizeXML(in)
	if got != want {
		t.Errorf("multiple refs: got %q, want %q", got, want)
	}
}

func TestSanitizeXML_InvalidUTF8_Latin1(t *testing.T) {
	// 0xA0 = Latin-1 non-breaking space, invalid as lone UTF-8 byte
	in := "<NAME>Chajju\xa0Majra</NAME>"
	want := "<NAME>Chajju\u00a0Majra</NAME>"
	got := SanitizeXML(in)
	if got != want {
		t.Errorf("latin1 byte: got %q, want %q", got, want)
	}
}

func TestSanitizeXML_CombinedIssues(t *testing.T) {
	in := "<A>&#4;foo\xa0bar\x03baz</A>"
	want := "<A>foo\u00a0barbaz</A>"
	got := SanitizeXML(in)
	if got != want {
		t.Errorf("combined: got %q, want %q", got, want)
	}
}
