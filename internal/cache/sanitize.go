package cache

import (
	"regexp"
	"strings"
)

var unsafeChars = regexp.MustCompile(`[^\w\-.]`)

// SanitizeFilename converts a party/stock name into a safe filesystem name.
// Spaces become underscores, special characters are stripped, result is trimmed to 100 chars.
func SanitizeFilename(name string) string {
	s := strings.TrimSpace(name)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = unsafeChars.ReplaceAllString(s, "")
	if len(s) > 100 {
		s = s[:100]
	}
	if s == "" {
		s = "_unnamed_"
	}
	return s
}
