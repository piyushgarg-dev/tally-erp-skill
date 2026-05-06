package tally

// SanitizeXML strips bytes that are illegal in XML 1.0 character data.
// Tally occasionally emits raw control characters (notably 0x03) inside fields
// like <PARENTSTRUCTURE> as path separators, which makes encoding/xml reject
// the response. Allowed: 0x09, 0x0A, 0x0D, and any rune >= 0x20 (excluding
// the non-character code points 0xFFFE and 0xFFFF).
func SanitizeXML(s string) string {
	if !needsSanitize(s) {
		return s
	}
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if r == 0x09 || r == 0x0A || r == 0x0D || (r >= 0x20 && r != 0xFFFE && r != 0xFFFF) {
			out = append(out, string(r)...)
		}
	}
	return string(out)
}

func needsSanitize(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x20 && c != 0x09 && c != 0x0A && c != 0x0D {
			return true
		}
	}
	return false
}
