package cli

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"strings"
)

// renderOutput formats a Tally XML response according to the user's --format and --pretty flags.
// Unknown formats fall back to raw XML.
func renderOutput(resp string, format string, pretty bool) string {
	switch strings.ToLower(format) {
	case "json":
		if out, ok := xmlToJSON(resp, pretty); ok {
			return out
		}
		return resp
	case "", "xml":
		if pretty {
			return prettyXML(resp)
		}
		return resp
	default:
		return resp
	}
}

// prettyXML re-encodes XML with two-space indent. Returns input unchanged on parse failure.
func prettyXML(in string) string { return pretty(in) }

// xmlToJSON converts a Tally XML envelope into JSON.
//
// Conventions:
//   - Each element becomes a JSON object.
//   - Attributes are stored as keys prefixed with "@".
//   - Text content for an element with no children/attrs collapses to a string.
//   - Text content for an element with attrs is stored under "#text".
//   - Repeated children with the same name become arrays.
//   - Empty leaf elements (<X/>) become "".
func xmlToJSON(in string, pretty bool) (string, bool) {
	dec := xml.NewDecoder(strings.NewReader(in))
	dec.Strict = false

	root, ok := decodeElement(dec, nil)
	if !ok {
		return "", false
	}
	// Wrap root element in a single-key object: {"<rootName>": <value>}
	wrapped := map[string]any{root.name: elementToValue(root)}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(wrapped); err != nil {
		return "", false
	}
	out := strings.TrimRight(buf.String(), "\n")
	return out, true
}

// element is an intermediate tree node built from the XML decoder.
type element struct {
	name     string
	attrs    map[string]string
	text     string
	children []*element
}

// decodeElement reads tokens until it has fully consumed one element. If start is nil,
// it skips through the prologue/whitespace until it finds the first start element.
func decodeElement(dec *xml.Decoder, start *xml.StartElement) (*element, bool) {
	if start == nil {
		for {
			tok, err := dec.Token()
			if err != nil {
				return nil, false
			}
			if se, ok := tok.(xml.StartElement); ok {
				start = &se
				break
			}
		}
	}
	el := &element{name: start.Name.Local}
	if len(start.Attr) > 0 {
		el.attrs = make(map[string]string, len(start.Attr))
		for _, a := range start.Attr {
			el.attrs[a.Name.Local] = a.Value
		}
	}
	var textBuf strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, false
		}
		switch t := tok.(type) {
		case xml.StartElement:
			child, ok := decodeElement(dec, &t)
			if !ok {
				return nil, false
			}
			el.children = append(el.children, child)
		case xml.EndElement:
			el.text = textBuf.String()
			return el, true
		case xml.CharData:
			textBuf.Write(t)
		}
	}
}

// elementToValue converts a parsed element into its JSON-shaped value.
func elementToValue(el *element) any {
	hasAttrs := len(el.attrs) > 0
	hasChildren := len(el.children) > 0
	text := strings.TrimSpace(el.text)

	if !hasAttrs && !hasChildren {
		return text
	}

	obj := map[string]any{}
	for k, v := range el.attrs {
		obj["@"+k] = v
	}
	// Group children by name to detect repeats.
	grouped := make(map[string][]*element)
	order := []string{}
	for _, c := range el.children {
		if _, seen := grouped[c.name]; !seen {
			order = append(order, c.name)
		}
		grouped[c.name] = append(grouped[c.name], c)
	}
	for _, name := range order {
		group := grouped[name]
		if len(group) == 1 {
			obj[name] = elementToValue(group[0])
			continue
		}
		arr := make([]any, 0, len(group))
		for _, c := range group {
			arr = append(arr, elementToValue(c))
		}
		obj[name] = arr
	}
	if text != "" {
		obj["#text"] = text
	}
	return obj
}
