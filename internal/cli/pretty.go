package cli

import (
	"bytes"
	"encoding/xml"
	"strings"
)

// pretty re-encodes the XML with two-space indent. Returns input unchanged if it isn't parseable.
func pretty(in string) string {
	dec := xml.NewDecoder(strings.NewReader(in))
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		if err := enc.EncodeToken(tok); err != nil {
			return in
		}
	}
	if err := enc.Flush(); err != nil {
		return in
	}
	out := buf.String()
	if out == "" {
		return in
	}
	return out
}
