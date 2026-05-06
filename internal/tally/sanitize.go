package tally

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// SanitizeXML cleans a Tally XML response so encoding/xml can parse it.
//
// Tally responses may contain:
//  1. Raw control characters (0x03, 0x04) used as path separators.
//  2. Numeric character references to invalid codepoints (&#4;, &#x4;).
//  3. Invalid UTF-8 bytes (lone 0x80-0xFF from Windows-1252/Latin-1 data).
//
// This function fixes all three: it strips illegal char refs, re-encodes stray
// high bytes as their Latin-1 Unicode equivalents, and drops control characters.
func SanitizeXML(s string) string {
	s = stripIllegalCharRefs(s)
	s = fixLatin1Bytes(s)
	if !needsSanitize(s) {
		return s
	}
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if isValidXMLChar(r) {
			out = append(out, string(r)...)
		}
	}
	return string(out)
}

// fixLatin1Bytes replaces invalid UTF-8 byte sequences with their Latin-1
// Unicode equivalents. Tally often emits 0x80-0xFF bytes from Windows-1252
// without proper UTF-8 encoding.
func fixLatin1Bytes(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	var out []byte
	b := []byte(s)
	for i := 0; i < len(b); {
		r, size := utf8.DecodeRune(b[i:])
		if r == utf8.RuneError && size == 1 {
			// Stray byte -- interpret as Latin-1 codepoint
			out = append(out, string(rune(b[i]))...)
			i++
		} else {
			out = append(out, b[i:i+size]...)
			i += size
		}
	}
	return string(out)
}

func isValidXMLChar(r rune) bool {
	return r == 0x09 || r == 0x0A || r == 0x0D ||
		(r >= 0x20 && r <= 0xD7FF) ||
		(r >= 0xE000 && r <= 0xFFFD) ||
		(r >= 0x10000 && r <= 0x10FFFF)
}

// illegalCharRefRe matches decimal (&#N;) and hex (&#xN;) XML character references.
var illegalCharRefRe = regexp.MustCompile(`&#x?[0-9a-fA-F]+;`)

// stripIllegalCharRefs removes &#N; and &#xN; references that decode to
// characters invalid in XML 1.0.
func stripIllegalCharRefs(s string) string {
	if !strings.Contains(s, "&#") {
		return s
	}
	return illegalCharRefRe.ReplaceAllStringFunc(s, func(ref string) string {
		inner := ref[2 : len(ref)-1] // strip "&#" prefix and ";" suffix
		var cp int64
		if strings.HasPrefix(inner, "x") || strings.HasPrefix(inner, "X") {
			cp, _ = strconv.ParseInt(inner[1:], 16, 32)
		} else {
			cp, _ = strconv.ParseInt(inner, 10, 32)
		}
		if isValidXMLChar(rune(cp)) {
			return ref
		}
		return ""
	})
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
