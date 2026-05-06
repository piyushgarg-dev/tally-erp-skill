package tally

import (
	"encoding/xml"
	"strings"
)

type Status struct {
	Parsed   bool
	Code     int    // STATUS value (1 = success, 0 = failure, -1 unknown)
	Message  string // best-effort human message from <DATA>
	RawError string // full <DATA> contents for failures
}

func (s Status) Success() bool { return s.Parsed && s.Code == 1 }

type statusEnvelope struct {
	XMLName xml.Name `xml:"ENVELOPE"`
	Header  struct {
		Status string `xml:"STATUS"`
	} `xml:"HEADER"`
	Body struct {
		Data struct {
			Inner string `xml:",innerxml"`
		} `xml:"DATA"`
	} `xml:"BODY"`
}

func ParseStatus(body string) Status {
	var env statusEnvelope
	if err := xml.Unmarshal([]byte(body), &env); err != nil {
		return Status{Parsed: false, Code: -1}
	}
	s := Status{Parsed: true, RawError: env.Body.Data.Inner}
	switch strings.TrimSpace(env.Header.Status) {
	case "1":
		s.Code = 1
	case "0":
		s.Code = 0
		s.Message = extractMessage(env.Body.Data.Inner)
	default:
		s.Code = -1
	}
	return s
}

func extractMessage(data string) string {
	data = strings.TrimSpace(data)
	if strings.HasPrefix(data, "<LINEERROR>") {
		end := strings.Index(data, "</LINEERROR>")
		if end > 0 {
			return strings.TrimSpace(data[len("<LINEERROR>"):end])
		}
	}
	if strings.HasPrefix(data, "<") {
		// Some other structured failure; return as-is
		return data
	}
	return data
}
