package tally

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"
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
	keys := make([]string, 0, len(extra))
	for k := range extra {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		sb.Vars = append(sb.Vars, sv(k, extra[k]))
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
	TDL     string // raw TDL XML to inject inside <DESC>
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
	env.Body.Desc.TDL = r.TDL
	return marshal(env)
}

// ReportRequest describes an Export Data report query.
type ReportRequest struct {
	ID       string
	Company  string
	FromDate string // YYYYMMDD
	ToDate   string // YYYYMMDD
	Vars     map[string]string
	TDL      string // raw TDL XML to inject inside <DESC>
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
	env.Body.Desc.TDL = r.TDL
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

// CollectionFilter specifies TDL-based filtering for a collection export.
type CollectionFilter struct {
	CollectionID string   // e.g. "List of Ledgers"
	ChildOf      string   // parent group, maps to CHILD OF
	Fields       []string // NATIVEMETHOD entries
	Filter       string   // raw TDL filter expression, e.g. "$ClosingBalance > 10000"
	FilterName   string   // internal filter name (auto-generated if empty)
}

// BuildCollectionTDL generates a <TDL> block that modifies a built-in collection.
func BuildCollectionTDL(f CollectionFilter) string {
	if f.ChildOf == "" && len(f.Fields) == 0 && f.Filter == "" {
		return ""
	}
	filterName := f.FilterName
	if filterName == "" {
		filterName = "CLIFilter"
	}

	var buf bytes.Buffer
	buf.WriteString("<TDL><TDLMESSAGE>")

	buf.WriteString(`<COLLECTION NAME="`)
	xml.Escape(&buf, []byte(f.CollectionID))
	buf.WriteString(`" ISMODIFY="Yes">`)

	if f.ChildOf != "" {
		buf.WriteString("<ADD>CHILD OF : ")
		xml.Escape(&buf, []byte(f.ChildOf))
		buf.WriteString("</ADD>")
	}
	for _, field := range f.Fields {
		buf.WriteString("<NATIVEMETHOD>")
		xml.Escape(&buf, []byte(field))
		buf.WriteString("</NATIVEMETHOD>")
	}
	if f.Filter != "" {
		buf.WriteString("<FILTERS>")
		xml.Escape(&buf, []byte(filterName))
		buf.WriteString("</FILTERS>")
	}
	buf.WriteString("</COLLECTION>")

	if f.Filter != "" {
		buf.WriteString(`<SYSTEM TYPE="Formulae" NAME="`)
		xml.Escape(&buf, []byte(filterName))
		buf.WriteString(`">`)
		xml.Escape(&buf, []byte(f.Filter))
		buf.WriteString("</SYSTEM>")
	}

	buf.WriteString("</TDLMESSAGE></TDL>")
	return buf.String()
}

// ReportFilter specifies TDL-based filtering for a report export.
type ReportFilter struct {
	ReportID    string // e.g. "Day Book"
	VoucherType string // filter by voucher type name
	Filter      string // raw TDL filter expression
	FilterName  string // internal filter name (auto-generated if empty)
}

// BuildReportTDL generates a <TDL> block that modifies a built-in report.
func BuildReportTDL(f ReportFilter) string {
	if f.VoucherType == "" && f.Filter == "" {
		return ""
	}

	filterName := f.FilterName
	if filterName == "" {
		filterName = "CLIFilter"
	}

	var buf bytes.Buffer
	buf.WriteString("<TDL><TDLMESSAGE>")

	buf.WriteString(`<REPORT NAME="`)
	xml.Escape(&buf, []byte(f.ReportID))
	buf.WriteString(`" ISMODIFY="Yes" ISFIXED="No" ISINITIALIZE="No" ISOPTION="No" ISINTERNAL="No">`)

	if f.VoucherType != "" {
		buf.WriteString(`<LOCAL>Collection : Default : Add :Filter : `)
		xml.Escape(&buf, []byte(filterName))
		buf.WriteString(`</LOCAL>`)
		buf.WriteString(`<LOCAL>Collection : Default : Add :Fetch : VoucherTypeName</LOCAL>`)
	} else if f.Filter != "" {
		buf.WriteString(`<LOCAL>Collection : Default : Add :Filter : `)
		xml.Escape(&buf, []byte(filterName))
		buf.WriteString(`</LOCAL>`)
	}

	buf.WriteString("</REPORT>")

	if f.VoucherType != "" {
		buf.WriteString(`<SYSTEM TYPE="Formulae" NAME="`)
		xml.Escape(&buf, []byte(filterName))
		buf.WriteString(`">$VoucherTypeName=`)
		xml.Escape(&buf, []byte(f.VoucherType))
		buf.WriteString("</SYSTEM>")
	} else if f.Filter != "" {
		buf.WriteString(`<SYSTEM TYPE="Formulae" NAME="`)
		xml.Escape(&buf, []byte(filterName))
		buf.WriteString(`">`)
		xml.Escape(&buf, []byte(f.Filter))
		buf.WriteString("</SYSTEM>")
	}

	buf.WriteString("</TDLMESSAGE></TDL>")
	return buf.String()
}
