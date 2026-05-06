package tally

import (
	"strings"
)

// MergeResponses combines multiple Tally XML responses into a single envelope.
// It extracts the inner content from each response's <COLLECTION> or <TALLYMESSAGE>
// blocks and wraps them in a unified envelope.
//
// Empty responses and error responses (STATUS=0) are skipped.
func MergeResponses(responses []string) string {
	var bodies []string
	for _, resp := range responses {
		inner := extractBody(resp)
		if inner != "" {
			bodies = append(bodies, inner)
		}
	}
	if len(bodies) == 0 {
		return `<ENVELOPE><HEADER><VERSION>1</VERSION><STATUS>1</STATUS></HEADER><BODY><DATA></DATA></BODY></ENVELOPE>`
	}
	var sb strings.Builder
	sb.WriteString("<ENVELOPE>\n<HEADER><VERSION>1</VERSION><STATUS>1</STATUS></HEADER>\n<BODY><DATA>\n")
	for _, b := range bodies {
		sb.WriteString(b)
		sb.WriteString("\n")
	}
	sb.WriteString("</DATA></BODY>\n</ENVELOPE>")
	return sb.String()
}

// extractBody pulls the meaningful data content from a Tally XML response.
// It looks for content inside <DATA>...</DATA>, <COLLECTION>...</COLLECTION>,
// or <TALLYMESSAGE>...</TALLYMESSAGE> tags.
func extractBody(resp string) string {
	if resp == "" {
		return ""
	}

	// Skip error responses
	if strings.Contains(resp, "<STATUS>0</STATUS>") {
		return ""
	}

	// Try to extract <COLLECTION>...</COLLECTION> content (for TYPE=Collection responses)
	if start, end := findTag(resp, "COLLECTION"); start >= 0 {
		return resp[start:end]
	}

	// Try <DATA>...</DATA> inner content
	if start, end := findTagInner(resp, "DATA"); start >= 0 {
		inner := strings.TrimSpace(resp[start:end])
		if inner != "" {
			return inner
		}
	}

	// Try <TALLYMESSAGE>...</TALLYMESSAGE>
	if start, end := findTag(resp, "TALLYMESSAGE"); start >= 0 {
		return resp[start:end]
	}

	return ""
}

// findTag returns the start and end byte positions of the outermost occurrence
// of <TAG ...>...</TAG> (inclusive of the tags themselves).
func findTag(s, tag string) (int, int) {
	openPrefix := "<" + tag
	closeTag := "</" + tag + ">"

	start := indexOfTag(s, openPrefix)
	if start < 0 {
		return -1, -1
	}
	end := strings.LastIndex(s, closeTag)
	if end < 0 {
		return -1, -1
	}
	return start, end + len(closeTag)
}

// findTagInner returns the byte range of content BETWEEN <TAG> and </TAG>.
func findTagInner(s, tag string) (int, int) {
	openPrefix := "<" + tag
	closeTag := "</" + tag + ">"

	start := indexOfTag(s, openPrefix)
	if start < 0 {
		return -1, -1
	}
	// find the end of the opening tag
	gt := strings.Index(s[start:], ">")
	if gt < 0 {
		return -1, -1
	}
	contentStart := start + gt + 1

	end := strings.LastIndex(s, closeTag)
	if end < 0 || end <= contentStart {
		return -1, -1
	}
	return contentStart, end
}

// indexOfTag finds the first occurrence of a tag opening like "<TAG" or "<TAG "
// ensuring it's actually a tag and not part of another word.
func indexOfTag(s, prefix string) int {
	idx := 0
	for {
		pos := strings.Index(s[idx:], prefix)
		if pos < 0 {
			return -1
		}
		pos += idx
		afterPrefix := pos + len(prefix)
		if afterPrefix >= len(s) {
			return pos
		}
		ch := s[afterPrefix]
		if ch == '>' || ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t' {
			return pos
		}
		idx = afterPrefix
	}
}
