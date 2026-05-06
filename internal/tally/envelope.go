package tally

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

type staticVar struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

type envelope struct {
	XMLName xml.Name `xml:"ENVELOPE"`
	Header  header   `xml:"HEADER"`
	Body    body     `xml:"BODY"`
}

type header struct {
	Version      string `xml:"VERSION"`
	TallyRequest string `xml:"TALLYREQUEST"`
	Type         string `xml:"TYPE"`
	Subtype      string `xml:"SUBTYPE,omitempty"`
	ID           *idTag `xml:"ID,omitempty"`
}

type idTag struct {
	Type  string `xml:"TYPE,attr,omitempty"`
	Value string `xml:",chardata"`
}

type body struct {
	Desc desc `xml:"DESC"`
}

type desc struct {
	Static    *staticBlock `xml:"STATICVARIABLES,omitempty"`
	FetchList *fetchList   `xml:"FETCHLIST,omitempty"`
	TDL       string       `xml:",innerxml"`
}

type staticBlock struct {
	Vars []staticVar
}

type fetchList struct {
	Fetches []string `xml:"FETCH"`
}

// ObjectRequest describes an Export Object query.
type ObjectRequest struct {
	Subtype string
	IDType  string // typically "Name"
	ID      string
	Company string
	Fetch   []string
}

// BuildObject returns a Tally XML envelope for an Export Object request.
func BuildObject(r ObjectRequest) (string, error) {
	if r.Subtype == "" || r.ID == "" {
		return "", fmt.Errorf("subtype and id are required")
	}
	idType := r.IDType
	if idType == "" {
		idType = "Name"
	}
	env := envelope{
		Header: header{
			Version:      "1",
			TallyRequest: "Export",
			Type:         "Object",
			Subtype:      r.Subtype,
			ID:           &idTag{Type: idType, Value: r.ID},
		},
	}
	env.Body.Desc.Static = newStatics(r.Company, "", "", nil)
	if len(r.Fetch) > 0 {
		env.Body.Desc.FetchList = &fetchList{Fetches: r.Fetch}
	}
	return marshal(env)
}

func newStatics(company, fromDate, toDate string, extra map[string]string) *staticBlock {
	sb := &staticBlock{}
	if company != "" {
		sb.Vars = append(sb.Vars, sv("SVCURRENTCOMPANY", company))
	}
	if fromDate != "" {
		sb.Vars = append(sb.Vars, sv("SVFROMDATE", fromDate))
	}
	if toDate != "" {
		sb.Vars = append(sb.Vars, sv("SVTODATE", toDate))
	}
	for k, v := range extra {
		sb.Vars = append(sb.Vars, sv(k, v))
	}
	sb.Vars = append(sb.Vars, sv("SVEXPORTFORMAT", "$$SysName:XML"))
	return sb
}

func sv(name, value string) staticVar {
	return staticVar{XMLName: xml.Name{Local: name}, Value: value}
}

func marshal(v interface{}) (string, error) {
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// CollectionRequest describes an Export Collection query.
type CollectionRequest struct {
	ID      string
	Company string
	Fetch   []string
}

func BuildCollection(r CollectionRequest) (string, error) {
	if r.ID == "" {
		return "", fmt.Errorf("id is required")
	}
	env := envelope{
		Header: header{
			Version:      "1",
			TallyRequest: "Export",
			Type:         "Collection",
			ID:           &idTag{Value: r.ID},
		},
	}
	env.Body.Desc.Static = newStatics(r.Company, "", "", nil)
	if len(r.Fetch) > 0 {
		env.Body.Desc.FetchList = &fetchList{Fetches: r.Fetch}
	}
	return marshal(env)
}

// ReportRequest describes an Export Data report query.
type ReportRequest struct {
	ID       string
	Company  string
	FromDate string // YYYYMMDD
	ToDate   string // YYYYMMDD
	Vars     map[string]string
}

func BuildReport(r ReportRequest) (string, error) {
	if r.ID == "" {
		return "", fmt.Errorf("id is required")
	}
	env := envelope{
		Header: header{
			Version:      "1",
			TallyRequest: "Export",
			Type:         "Data",
			ID:           &idTag{Value: r.ID},
		},
	}
	env.Body.Desc.Static = newStatics(r.Company, r.FromDate, r.ToDate, r.Vars)
	return marshal(env)
}

func (i idTag) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "ID"
	if i.Type != "" {
		start.Attr = []xml.Attr{{Name: xml.Name{Local: "TYPE"}, Value: i.Type}}
	} else {
		start.Attr = nil
	}
	return e.EncodeElement(i.Value, start)
}

// MarshalStatic gives encoding/xml a stable shape for staticBlock.
func (sb staticBlock) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "STATICVARIABLES"
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	for _, v := range sb.Vars {
		if err := e.EncodeElement(v.Value, xml.StartElement{Name: v.XMLName}); err != nil {
			return err
		}
	}
	return e.EncodeToken(start.End())
}
