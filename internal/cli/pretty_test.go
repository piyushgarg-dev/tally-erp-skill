package cli

import (
	"strings"
	"testing"
)

func TestPrettyIndents(t *testing.T) {
	in := `<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER></ENVELOPE>`
	out := pretty(in)
	if !strings.Contains(out, "\n  <HEADER>") {
		t.Errorf("expected indented HEADER:\n%s", out)
	}
}

func TestPrettyFallsBackOnInvalid(t *testing.T) {
	in := `not xml`
	if got := pretty(in); got != in {
		t.Errorf("invalid XML should pass through unchanged, got: %q", got)
	}
}
