package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestXMLToJSONLeafText(t *testing.T) {
	in := `<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER></ENVELOPE>`
	out, ok := xmlToJSON(in, false)
	if !ok {
		t.Fatal("conversion failed")
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out)
	}
	env := got["ENVELOPE"].(map[string]any)
	hdr := env["HEADER"].(map[string]any)
	if hdr["STATUS"] != "1" {
		t.Fatalf("STATUS wrong: %#v", hdr["STATUS"])
	}
}

func TestXMLToJSONAttributesAndText(t *testing.T) {
	in := `<R><X TYPE="Amount">100.00</X></R>`
	out, _ := xmlToJSON(in, false)
	var got map[string]any
	_ = json.Unmarshal([]byte(out), &got)
	x := got["R"].(map[string]any)["X"].(map[string]any)
	if x["@TYPE"] != "Amount" || x["#text"] != "100.00" {
		t.Fatalf("got %#v", x)
	}
}

func TestXMLToJSONRepeatedChildrenBecomeArray(t *testing.T) {
	in := `<LIST><ITEM>a</ITEM><ITEM>b</ITEM><ITEM>c</ITEM></LIST>`
	out, _ := xmlToJSON(in, false)
	var got map[string]any
	_ = json.Unmarshal([]byte(out), &got)
	items := got["LIST"].(map[string]any)["ITEM"].([]any)
	if len(items) != 3 || items[0] != "a" || items[2] != "c" {
		t.Fatalf("got %#v", items)
	}
}

func TestXMLToJSONEmptyElement(t *testing.T) {
	in := `<R><EMPTY/></R>`
	out, _ := xmlToJSON(in, false)
	if !strings.Contains(out, `"EMPTY":""`) {
		t.Fatalf("expected empty leaf as empty string, got %s", out)
	}
}

func TestXMLToJSONInvalidFallsBack(t *testing.T) {
	if _, ok := xmlToJSON("not xml at all", false); ok {
		// Permissive parser may still accept non-XML; we just require renderOutput
		// to fall back to raw on the empty-tree edge case.
	}
	got := renderOutput("not xml at all", "json", false)
	if got != "not xml at all" {
		t.Fatalf("expected fallback to raw, got %q", got)
	}
}

func TestRenderOutputXMLPretty(t *testing.T) {
	in := `<A><B>1</B></A>`
	got := renderOutput(in, "xml", true)
	if !strings.Contains(got, "\n  ") {
		t.Fatalf("expected indented xml, got %q", got)
	}
}

func TestRenderOutputJSONPrettyIndents(t *testing.T) {
	in := `<A><B>1</B></A>`
	got := renderOutput(in, "json", true)
	if !strings.Contains(got, "\n  ") {
		t.Fatalf("expected indented json, got %q", got)
	}
}
