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
	if msg := extractStatusList(data); msg != "" {
		return msg
	}
	if strings.HasPrefix(data, "<") {
		return data
	}
	return data
}

type statusListEnvelope struct {
	Statuses []struct {
		Code string `xml:"CODE"`
		Desc string `xml:"DESC"`
	} `xml:"STATUS"`
}

func extractStatusList(data string) string {
	if !strings.Contains(data, "<STATUS.LIST>") {
		return ""
	}
	start := strings.Index(data, "<STATUS.LIST>")
	end := strings.Index(data, "</STATUS.LIST>")
	if start < 0 || end < 0 || end <= start {
		return ""
	}
	fragment := data[start : end+len("</STATUS.LIST>")]
	var sl statusListEnvelope
	if err := xml.Unmarshal([]byte(fragment), &sl); err != nil {
		return ""
	}
	if len(sl.Statuses) == 0 {
		return ""
	}
	s := sl.Statuses[0]
	code := strings.TrimSpace(s.Code)
	desc := strings.TrimSpace(s.Desc)
	if code != "" && desc != "" {
		return "[" + code + "] " + desc
	}
	if desc != "" {
		return desc
	}
	if code != "" {
		return "error code " + code
	}
	return ""
}
