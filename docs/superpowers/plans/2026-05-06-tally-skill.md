# Tally Skill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a Claude Code skill (`tally-erp`) that lets Claude query a running TallyPrime instance over its XML/HTTP gateway, backed by a precompiled cross-platform Go CLI (`tally`) that owns all envelope construction and HTTP transport.

**Architecture:** Single Go module. `cmd/tally` parses CLI flags and dispatches subcommands. `internal/tally` builds the XML envelopes and POSTs them to Tally. CLI prints raw XML response on stdout, sets exit code from `<STATUS>`. `SKILL.md` instructs Claude to invoke the binary instead of crafting HTTP/XML directly. `templates/` already contains XML reference envelopes that the Go builders must match.

**Tech Stack:** Go 1.22+, standard library only (`encoding/xml`, `net/http`, `flag`/`os/exec`). Make for cross-compilation.

---

## File map

```
tally-skill/
├── SKILL.md                          # Rewritten to instruct Claude to use the CLI
├── cmd/tally/main.go                 # Entry point: flag parsing + subcommand dispatch
├── internal/tally/
│   ├── envelope.go                   # Envelope structs + builders (Object/Collection/Data)
│   ├── envelope_test.go              # Golden-file tests vs templates/
│   ├── client.go                     # HTTP transport
│   ├── client_test.go                # httptest-backed transport tests
│   ├── status.go                     # Parse <STATUS>, <CODE>, <DESC>, <LINEERROR>
│   └── status_test.go
├── internal/cli/
│   ├── ping.go      / ping_test.go
│   ├── companies.go / companies_test.go
│   ├── object.go    / object_test.go
│   ├── collection.go/ collection_test.go
│   ├── report.go    / report_test.go
│   ├── raw.go       / raw_test.go
│   └── exit.go                       # Shared exit-code mapping
├── examples/                         # Captured response fixtures (added later)
├── go.mod / go.sum
├── Makefile
├── .gitignore
└── bin/                              # Cross-compiled binaries (added in Task 12)
```

---

## Task 1: Initialize repository

**Files:**
- Create: `.gitignore`, `go.mod`

- [ ] **Step 1: Initialize git**

Run:
```bash
cd /Users/piyushgarg/Coding/tally-skill
git init
git add SKILL.md docs/ templates/
git commit -m "chore: initial spec, plan, and XML templates"
```

- [ ] **Step 2: Create .gitignore**

Create `.gitignore`:
```
# Build outputs
bin/tally
bin/tally-*
!bin/.gitkeep

# Go
*.exe
*.test
*.out
coverage.txt

# OS / IDE
.DS_Store
.idea/
.vscode/
```

- [ ] **Step 3: Initialize Go module**

Run:
```bash
go mod init github.com/piyushgarg/tally-skill
```

Expected: creates `go.mod` with `module github.com/piyushgarg/tally-skill` and the Go version line.

- [ ] **Step 4: Commit**

```bash
git add .gitignore go.mod
git commit -m "chore: init Go module and gitignore"
```

---

## Task 2: Envelope structs + Object request builder

**Files:**
- Create: `internal/tally/envelope.go`
- Create: `internal/tally/envelope_test.go`

- [ ] **Step 1: Write failing test for Object envelope**

Create `internal/tally/envelope_test.go`:
```go
package tally

import (
	"strings"
	"testing"
)

func TestBuildObjectEnvelope(t *testing.T) {
	req := ObjectRequest{
		Subtype: "Ledger",
		IDType:  "Name",
		ID:      "Customer ABC",
		Company: "ABC Company Ltd",
		Fetch:   []string{"Name", "Parent", "ClosingBalance"},
	}
	got, err := BuildObject(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mustContain(t, got, `<TALLYREQUEST>Export</TALLYREQUEST>`)
	mustContain(t, got, `<TYPE>Object</TYPE>`)
	mustContain(t, got, `<SUBTYPE>Ledger</SUBTYPE>`)
	mustContain(t, got, `<ID TYPE="Name">Customer ABC</ID>`)
	mustContain(t, got, `<SVCURRENTCOMPANY>ABC Company Ltd</SVCURRENTCOMPANY>`)
	mustContain(t, got, `<FETCH>Name</FETCH>`)
	mustContain(t, got, `<FETCH>ClosingBalance</FETCH>`)
}

func TestObjectEscapesSpecialChars(t *testing.T) {
	req := ObjectRequest{Subtype: "Ledger", IDType: "Name", ID: `M&S "Ltd"`}
	got, err := BuildObject(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mustContain(t, got, `M&amp;S &#34;Ltd&#34;`)
}

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q\n--- output ---\n%s", needle, haystack)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tally/...`
Expected: FAIL — `BuildObject` and `ObjectRequest` undefined.

- [ ] **Step 3: Implement envelope.go (Object only for now)**

Create `internal/tally/envelope.go`:
```go
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
```

- [ ] **Step 4: Run tests, verify they pass**

Run: `go test ./internal/tally/...`
Expected: PASS for both `TestBuildObjectEnvelope` and `TestObjectEscapesSpecialChars`.

- [ ] **Step 5: Commit**

```bash
git add internal/tally/envelope.go internal/tally/envelope_test.go
git commit -m "feat(tally): Object request envelope builder"
```

---

## Task 3: Collection and Data (report) request builders

**Files:**
- Modify: `internal/tally/envelope.go`
- Modify: `internal/tally/envelope_test.go`

- [ ] **Step 1: Write failing tests for Collection and Report**

Append to `internal/tally/envelope_test.go`:
```go
func TestBuildCollection(t *testing.T) {
	got, err := BuildCollection(CollectionRequest{
		ID:      "List of Ledgers",
		Company: "ABC",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mustContain(t, got, `<TYPE>Collection</TYPE>`)
	mustContain(t, got, `<ID>List of Ledgers</ID>`)
	mustContain(t, got, `<SVCURRENTCOMPANY>ABC</SVCURRENTCOMPANY>`)
}

func TestBuildReport(t *testing.T) {
	got, err := BuildReport(ReportRequest{
		ID:       "Day Book",
		Company:  "ABC",
		FromDate: "20260401",
		ToDate:   "20260430",
		Vars:     map[string]string{"EXPLODEFLAG": "Yes"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mustContain(t, got, `<TYPE>Data</TYPE>`)
	mustContain(t, got, `<ID>Day Book</ID>`)
	mustContain(t, got, `<SVFROMDATE>20260401</SVFROMDATE>`)
	mustContain(t, got, `<SVTODATE>20260430</SVTODATE>`)
	mustContain(t, got, `<EXPLODEFLAG>Yes</EXPLODEFLAG>`)
}

func TestBuildReportLedgerVar(t *testing.T) {
	got, err := BuildReport(ReportRequest{
		ID:      "Ledger",
		Company: "ABC",
		Vars:    map[string]string{"LedgerName": "Customer ABC"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mustContain(t, got, `<LedgerName>Customer ABC</LedgerName>`)
}
```

- [ ] **Step 2: Run tests, verify they fail**

Run: `go test ./internal/tally/...`
Expected: FAIL — `BuildCollection`, `BuildReport`, `CollectionRequest`, `ReportRequest` undefined.

- [ ] **Step 3: Add Collection + Report builders**

Append to `internal/tally/envelope.go`:
```go
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
```

Modify `idTag` rendering so that `<ID>` without a `TYPE` attribute renders cleanly. Replace the `idTag` struct and add this method:

```go
func (i idTag) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "ID"
	if i.Type != "" {
		start.Attr = []xml.Attr{{Name: xml.Name{Local: "TYPE"}, Value: i.Type}}
	} else {
		start.Attr = nil
	}
	return e.EncodeElement(i.Value, start)
}
```

- [ ] **Step 4: Run tests, verify they pass**

Run: `go test ./internal/tally/... -v`
Expected: all envelope tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/tally/
git commit -m "feat(tally): Collection and Report request builders"
```

---

## Task 4: HTTP transport

**Files:**
- Create: `internal/tally/client.go`
- Create: `internal/tally/client_test.go`

- [ ] **Step 1: Write failing test using httptest**

Create `internal/tally/client_test.go`:
```go
package tally

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientPostsXML(t *testing.T) {
	var gotBody string
	var gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotCT = r.Header.Get("Content-Type")
		_, _ = w.Write([]byte(`<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER><BODY><DATA>ok</DATA></BODY></ENVELOPE>`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 5*time.Second)
	resp, err := c.Post(context.Background(), `<ENVELOPE/>`)
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}
	if !strings.Contains(resp, "<STATUS>1</STATUS>") {
		t.Errorf("unexpected response: %s", resp)
	}
	if gotBody != `<ENVELOPE/>` {
		t.Errorf("server received body %q", gotBody)
	}
	if !strings.HasPrefix(gotCT, "text/xml") {
		t.Errorf("unexpected content-type %q", gotCT)
	}
}

func TestClientTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 50*time.Millisecond)
	_, err := c.Post(context.Background(), `<ENVELOPE/>`)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
```

- [ ] **Step 2: Run, verify failure**

Run: `go test ./internal/tally/... -run Client`
Expected: FAIL — `NewClient` undefined.

- [ ] **Step 3: Implement client.go**

Create `internal/tally/client.go`:
```go
package tally

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	url string
	hc  *http.Client
}

func NewClient(url string, timeout time.Duration) *Client {
	return &Client{
		url: url,
		hc:  &http.Client{Timeout: timeout},
	}
}

func (c *Client) Post(ctx context.Context, body string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return string(b), fmt.Errorf("tally returned HTTP %d", resp.StatusCode)
	}
	return string(b), nil
}
```

- [ ] **Step 4: Run, verify pass**

Run: `go test ./internal/tally/...`
Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/tally/client.go internal/tally/client_test.go
git commit -m "feat(tally): HTTP transport with timeout"
```

---

## Task 5: Status / error parsing

**Files:**
- Create: `internal/tally/status.go`
- Create: `internal/tally/status_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/tally/status_test.go`:
```go
package tally

import "testing"

func TestParseStatusSuccess(t *testing.T) {
	xml := `<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER><BODY><DATA>...</DATA></BODY></ENVELOPE>`
	r := ParseStatus(xml)
	if !r.Success() {
		t.Fatal("expected success")
	}
}

func TestParseStatusFailureWithLineError(t *testing.T) {
	xml := `<ENVELOPE><HEADER><STATUS>0</STATUS></HEADER><BODY><DATA><LINEERROR>Voucher totals do not match</LINEERROR></DATA></BODY></ENVELOPE>`
	r := ParseStatus(xml)
	if r.Success() {
		t.Fatal("expected failure")
	}
	if r.Message != "Voucher totals do not match" {
		t.Errorf("got %q", r.Message)
	}
}

func TestParseStatusFailurePlainText(t *testing.T) {
	xml := `<ENVELOPE><HEADER><STATUS>0</STATUS></HEADER><BODY><DATA>DESC not found</DATA></BODY></ENVELOPE>`
	r := ParseStatus(xml)
	if r.Success() {
		t.Fatal("expected failure")
	}
	if r.Message != "DESC not found" {
		t.Errorf("got %q", r.Message)
	}
}

func TestParseStatusUnparseable(t *testing.T) {
	r := ParseStatus(`not xml at all`)
	if r.Parsed {
		t.Fatal("expected Parsed=false")
	}
}
```

- [ ] **Step 2: Run, verify failure**

Run: `go test ./internal/tally/... -run Status`
Expected: FAIL — `ParseStatus` undefined.

- [ ] **Step 3: Implement status.go**

Create `internal/tally/status.go`:
```go
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
```

- [ ] **Step 4: Run tests, verify pass**

Run: `go test ./internal/tally/...`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/tally/status.go internal/tally/status_test.go
git commit -m "feat(tally): parse <STATUS> and error messages"
```

---

## Task 6: CLI scaffolding + exit codes + global flags

**Files:**
- Create: `internal/cli/exit.go`
- Create: `cmd/tally/main.go`

- [ ] **Step 1: Create exit codes**

Create `internal/cli/exit.go`:
```go
package cli

const (
	ExitOK            = 0
	ExitTallyFailure  = 1
	ExitConnect       = 2
	ExitUsage         = 3
	ExitTimeout       = 4
	ExitBadResponse   = 5
)
```

- [ ] **Step 2: Scaffold main.go with global flags + dispatcher (subcommands stubbed)**

Create `cmd/tally/main.go`:
```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/piyushgarg/tally-skill/internal/cli"
)

var Version = "dev"

type globalFlags struct {
	Host    string
	Port    int
	Company string
	Timeout time.Duration
	Pretty  bool
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(cli.ExitUsage)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "version", "--version", "-v":
		fmt.Println("tally", Version)
	case "ping":
		os.Exit(cli.RunPing(args))
	case "companies":
		os.Exit(cli.RunCompanies(args))
	case "object":
		os.Exit(cli.RunObject(args))
	case "collection":
		os.Exit(cli.RunCollection(args))
	case "report":
		os.Exit(cli.RunReport(args))
	case "raw":
		os.Exit(cli.RunRaw(args))
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmd)
		usage()
		os.Exit(cli.ExitUsage)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `tally - read-only CLI for TallyPrime XML gateway

Usage:
  tally <command> [flags]

Commands:
  ping                Verify Tally is reachable and a company is loaded
  companies           List loaded companies
  object              Export a single object (Ledger, StockItem, Voucher, ...)
  collection          Export a collection (List of Ledgers, ...)
  report              Export a standard report (Day Book, Trial Balance, ...)
  raw                 POST a raw XML envelope (stdin or --file)
  version             Print version

Global flags (any subcommand):
  --host string       Tally host (default "localhost")
  --port int          Tally port (default 9000)
  --company string    Sets SVCURRENTCOMPANY
  --timeout duration  HTTP timeout (default 30s)
  --pretty            Pretty-print response XML`)
}
```

- [ ] **Step 3: Add a shared flag parser in `internal/cli`**

Create `internal/cli/flags.go`:
```go
package cli

import (
	"flag"
	"fmt"
	"time"
)

type Globals struct {
	Host    string
	Port    int
	Company string
	Timeout time.Duration
	Pretty  bool
}

func registerGlobals(fs *flag.FlagSet) *Globals {
	g := &Globals{}
	fs.StringVar(&g.Host, "host", "localhost", "Tally host")
	fs.IntVar(&g.Port, "port", 9000, "Tally port")
	fs.StringVar(&g.Company, "company", "", "Tally current company name")
	fs.DurationVar(&g.Timeout, "timeout", 30*time.Second, "HTTP timeout")
	fs.BoolVar(&g.Pretty, "pretty", false, "Pretty-print response")
	return g
}

func (g *Globals) URL() string {
	return fmt.Sprintf("http://%s:%d/", g.Host, g.Port)
}
```

- [ ] **Step 4: Add stub run functions so the build compiles**

Create `internal/cli/stubs.go` (will be replaced task-by-task):
```go
package cli

import "fmt"

func RunPing(_ []string) int       { fmt.Println("not implemented"); return ExitUsage }
func RunCompanies(_ []string) int  { return RunPing(nil) }
func RunObject(_ []string) int     { return RunPing(nil) }
func RunCollection(_ []string) int { return RunPing(nil) }
func RunReport(_ []string) int     { return RunPing(nil) }
func RunRaw(_ []string) int        { return RunPing(nil) }
```

- [ ] **Step 5: Build and run help to confirm it compiles**

Run:
```bash
go build -o /tmp/tally ./cmd/tally && /tmp/tally help
```
Expected: usage prints, exit code 0.

- [ ] **Step 6: Commit**

```bash
git add cmd/ internal/cli/
git commit -m "feat(cli): scaffolding, global flags, exit codes"
```

---

## Task 7: `tally raw` (passthrough)

**Files:**
- Create: `internal/cli/raw.go`
- Create: `internal/cli/raw_test.go`
- Modify: `internal/cli/stubs.go` (remove the `RunRaw` stub)

- [ ] **Step 1: Failing test**

Create `internal/cli/raw_test.go`:
```go
package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRawPipesStdinToTally(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		got = string(b)
		_, _ = w.Write([]byte(`<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER><BODY><DATA>ok</DATA></BODY></ENVELOPE>`))
	}))
	defer srv.Close()

	stdin := strings.NewReader(`<ENVELOPE/>`)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := runRawWithIO([]string{"--host", hostOf(srv.URL), "--port", portOf(srv.URL)}, stdin, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, stderr.String())
	}
	if got != `<ENVELOPE/>` {
		t.Errorf("server got %q", got)
	}
	if !strings.Contains(stdout.String(), "<STATUS>1</STATUS>") {
		t.Errorf("stdout=%s", stdout.String())
	}
}
```

Also create `internal/cli/testhelpers_test.go`:
```go
package cli

import (
	"net/url"
	"strconv"
)

func hostOf(rawURL string) string {
	u, _ := url.Parse(rawURL)
	return u.Hostname()
}

func portOf(rawURL string) string {
	u, _ := url.Parse(rawURL)
	p, _ := strconv.Atoi(u.Port())
	return strconv.Itoa(p)
}
```

- [ ] **Step 2: Run, verify failure**

Run: `go test ./internal/cli/... -run TestRaw`
Expected: FAIL — `runRawWithIO` undefined.

- [ ] **Step 3: Implement raw.go**

Create `internal/cli/raw.go`:
```go
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/piyushgarg/tally-skill/internal/tally"
)

func RunRaw(args []string) int {
	return runRawWithIO(args, os.Stdin, os.Stdout, os.Stderr)
}

func runRawWithIO(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("raw", flag.ContinueOnError)
	fs.SetOutput(stderr)
	g := registerGlobals(fs)
	file := fs.String("file", "", "Read XML from file instead of stdin")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}

	var body []byte
	var err error
	if *file != "" {
		body, err = os.ReadFile(*file)
	} else {
		body, err = io.ReadAll(stdin)
	}
	if err != nil {
		fmt.Fprintf(stderr, "tally: read input: %v\n", err)
		return ExitUsage
	}
	if len(body) == 0 {
		fmt.Fprintln(stderr, "tally: empty input")
		return ExitUsage
	}

	c := tally.NewClient(g.URL(), g.Timeout)
	resp, err := c.Post(context.Background(), string(body))
	if err != nil {
		return reportTransportError(stderr, err)
	}

	out := resp
	if g.Pretty {
		out = pretty(resp)
	}
	fmt.Fprintln(stdout, out)

	return statusToExit(stderr, resp)
}

func reportTransportError(stderr io.Writer, err error) int {
	var nerr net.Error
	if errors.As(err, &nerr) && nerr.Timeout() {
		fmt.Fprintf(stderr, "tally: timeout: %v\n", err)
		return ExitTimeout
	}
	fmt.Fprintf(stderr, "tally: cannot reach server: %v\n", err)
	return ExitConnect
}

func statusToExit(stderr io.Writer, resp string) int {
	st := tally.ParseStatus(resp)
	if !st.Parsed {
		fmt.Fprintln(stderr, "tally: response not valid XML")
		return ExitBadResponse
	}
	if st.Code == 1 {
		return ExitOK
	}
	fmt.Fprintf(stderr, "tally: failure: %s\n", st.Message)
	return ExitTallyFailure
}

func pretty(s string) string {
	// Cheap pretty-print: rely on encoding/xml round-trip
	return s // implementation deferred to Task 11
}
```

Then in `internal/cli/stubs.go`, delete the `RunRaw` stub line.

- [ ] **Step 4: Run tests, verify pass**

Run: `go test ./internal/cli/... -run TestRaw -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/raw.go internal/cli/testhelpers_test.go internal/cli/raw_test.go internal/cli/stubs.go
git commit -m "feat(cli): tally raw passthrough"
```

---

## Task 8: `tally object` and `tally collection`

**Files:**
- Create: `internal/cli/object.go`
- Create: `internal/cli/object_test.go`
- Create: `internal/cli/collection.go`
- Create: `internal/cli/collection_test.go`
- Modify: `internal/cli/stubs.go`

- [ ] **Step 1: Failing test for object**

Create `internal/cli/object_test.go`:
```go
package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestObjectBuildsAndPosts(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		got = string(b)
		_, _ = w.Write([]byte(`<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER><BODY><DATA/></BODY></ENVELOPE>`))
	}))
	defer srv.Close()

	out := &bytes.Buffer{}
	errb := &bytes.Buffer{}
	code := runObjectWithIO([]string{
		"--host", hostOf(srv.URL), "--port", portOf(srv.URL),
		"--company", "ABC",
		"--subtype", "Ledger",
		"--id", "Customer ABC",
		"--fetch", "Name,Parent,ClosingBalance",
	}, out, errb)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, errb.String())
	}
	if !strings.Contains(got, "<SUBTYPE>Ledger</SUBTYPE>") {
		t.Errorf("server received: %s", got)
	}
	if !strings.Contains(got, `<ID TYPE="Name">Customer ABC</ID>`) {
		t.Errorf("server received: %s", got)
	}
}
```

- [ ] **Step 2: Implement object.go**

Create `internal/cli/object.go`:
```go
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/piyushgarg/tally-skill/internal/tally"
)

func RunObject(args []string) int { return runObjectWithIO(args, os.Stdout, os.Stderr) }

func runObjectWithIO(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("object", flag.ContinueOnError)
	fs.SetOutput(stderr)
	g := registerGlobals(fs)
	subtype := fs.String("subtype", "", "Object subtype (Ledger, Group, StockItem, Voucher, ...)")
	id := fs.String("id", "", "Object identifier")
	idType := fs.String("id-type", "Name", "Type of the identifier")
	fetch := fs.String("fetch", "", "Comma-separated FETCH fields")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	if *subtype == "" || *id == "" {
		fmt.Fprintln(stderr, "tally object: --subtype and --id are required")
		return ExitUsage
	}

	body, err := tally.BuildObject(tally.ObjectRequest{
		Subtype: *subtype,
		IDType:  *idType,
		ID:      *id,
		Company: g.Company,
		Fetch:   splitCSV(*fetch),
	})
	if err != nil {
		fmt.Fprintf(stderr, "tally object: %v\n", err)
		return ExitUsage
	}

	c := tally.NewClient(g.URL(), g.Timeout)
	resp, err := c.Post(context.Background(), body)
	if err != nil {
		return reportTransportError(stderr, err)
	}
	fmt.Fprintln(stdout, resp)
	return statusToExit(stderr, resp)
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
```

- [ ] **Step 3: Failing test for collection**

Create `internal/cli/collection_test.go`:
```go
package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCollectionBuildsAndPosts(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		got = string(b)
		_, _ = w.Write([]byte(`<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER><BODY><DATA/></BODY></ENVELOPE>`))
	}))
	defer srv.Close()

	out := &bytes.Buffer{}
	errb := &bytes.Buffer{}
	code := runCollectionWithIO([]string{
		"--host", hostOf(srv.URL), "--port", portOf(srv.URL),
		"--company", "ABC",
		"--id", "List of Ledgers",
	}, out, errb)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, errb.String())
	}
	if !strings.Contains(got, "<ID>List of Ledgers</ID>") {
		t.Errorf("server got: %s", got)
	}
}
```

- [ ] **Step 4: Implement collection.go**

Create `internal/cli/collection.go`:
```go
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/piyushgarg/tally-skill/internal/tally"
)

func RunCollection(args []string) int { return runCollectionWithIO(args, os.Stdout, os.Stderr) }

func runCollectionWithIO(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("collection", flag.ContinueOnError)
	fs.SetOutput(stderr)
	g := registerGlobals(fs)
	id := fs.String("id", "", "Collection name (e.g. \"List of Ledgers\")")
	fetch := fs.String("fetch", "", "Comma-separated FETCH fields")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	if *id == "" {
		fmt.Fprintln(stderr, "tally collection: --id is required")
		return ExitUsage
	}

	body, err := tally.BuildCollection(tally.CollectionRequest{
		ID:      *id,
		Company: g.Company,
		Fetch:   splitCSV(*fetch),
	})
	if err != nil {
		fmt.Fprintf(stderr, "tally collection: %v\n", err)
		return ExitUsage
	}

	c := tally.NewClient(g.URL(), g.Timeout)
	resp, err := c.Post(context.Background(), body)
	if err != nil {
		return reportTransportError(stderr, err)
	}
	fmt.Fprintln(stdout, resp)
	return statusToExit(stderr, resp)
}
```

Then delete the `RunObject` and `RunCollection` stub lines from `stubs.go`.

- [ ] **Step 5: Run tests, verify pass**

Run: `go test ./internal/cli/... -v`
Expected: object + collection tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/object.go internal/cli/object_test.go internal/cli/collection.go internal/cli/collection_test.go internal/cli/stubs.go
git commit -m "feat(cli): tally object and tally collection"
```

---

## Task 9: `tally report`

**Files:**
- Create: `internal/cli/report.go`
- Create: `internal/cli/report_test.go`
- Modify: `internal/cli/stubs.go`

- [ ] **Step 1: Failing test**

Create `internal/cli/report_test.go`:
```go
package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReportConvertsISODates(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		got = string(b)
		_, _ = w.Write([]byte(`<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER><BODY><DATA/></BODY></ENVELOPE>`))
	}))
	defer srv.Close()

	code := runReportWithIO([]string{
		"--host", hostOf(srv.URL), "--port", portOf(srv.URL),
		"--company", "ABC",
		"--id", "Day Book",
		"--from", "2026-04-01",
		"--to", "2026-04-30",
	}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(got, "<SVFROMDATE>20260401</SVFROMDATE>") {
		t.Errorf("from-date not converted: %s", got)
	}
	if !strings.Contains(got, "<SVTODATE>20260430</SVTODATE>") {
		t.Errorf("to-date not converted: %s", got)
	}
}

func TestReportLedgerVar(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		got = string(b)
		_, _ = w.Write([]byte(`<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER><BODY><DATA/></BODY></ENVELOPE>`))
	}))
	defer srv.Close()

	runReportWithIO([]string{
		"--host", hostOf(srv.URL), "--port", portOf(srv.URL),
		"--id", "Ledger",
		"--ledger", "Customer ABC",
	}, &bytes.Buffer{}, &bytes.Buffer{})
	if !strings.Contains(got, "<LedgerName>Customer ABC</LedgerName>") {
		t.Errorf("expected LedgerName var: %s", got)
	}
}
```

- [ ] **Step 2: Implement report.go**

Create `internal/cli/report.go`:
```go
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/piyushgarg/tally-skill/internal/tally"
)

type repeatableVar struct{ vals map[string]string }

func (r *repeatableVar) String() string { return "" }
func (r *repeatableVar) Set(v string) error {
	if r.vals == nil {
		r.vals = map[string]string{}
	}
	parts := strings.SplitN(v, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected KEY=VALUE, got %q", v)
	}
	r.vals[parts[0]] = parts[1]
	return nil
}

func RunReport(args []string) int { return runReportWithIO(args, os.Stdout, os.Stderr) }

func runReportWithIO(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	g := registerGlobals(fs)
	id := fs.String("id", "", "Report name (e.g. \"Day Book\")")
	from := fs.String("from", "", "From date (YYYY-MM-DD)")
	to := fs.String("to", "", "To date (YYYY-MM-DD)")
	ledger := fs.String("ledger", "", "Ledger name for ledger reports")
	group := fs.String("group", "", "Group name for group reports")
	explode := fs.Bool("explode", false, "Set EXPLODEFLAG=Yes")
	vars := &repeatableVar{}
	fs.Var(vars, "var", "Repeatable; KEY=VALUE for arbitrary STATICVARIABLE")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	if *id == "" {
		fmt.Fprintln(stderr, "tally report: --id is required")
		return ExitUsage
	}

	fromTally, err := isoToTallyDate(*from)
	if err != nil {
		fmt.Fprintf(stderr, "tally report: --from: %v\n", err)
		return ExitUsage
	}
	toTally, err := isoToTallyDate(*to)
	if err != nil {
		fmt.Fprintf(stderr, "tally report: --to: %v\n", err)
		return ExitUsage
	}

	extra := map[string]string{}
	for k, v := range vars.vals {
		extra[k] = v
	}
	if *ledger != "" {
		extra["LedgerName"] = *ledger
	}
	if *group != "" {
		extra["GroupName"] = *group
	}
	if *explode {
		extra["EXPLODEFLAG"] = "Yes"
	}

	body, err := tally.BuildReport(tally.ReportRequest{
		ID:       *id,
		Company:  g.Company,
		FromDate: fromTally,
		ToDate:   toTally,
		Vars:     extra,
	})
	if err != nil {
		fmt.Fprintf(stderr, "tally report: %v\n", err)
		return ExitUsage
	}

	c := tally.NewClient(g.URL(), g.Timeout)
	resp, err := c.Post(context.Background(), body)
	if err != nil {
		return reportTransportError(stderr, err)
	}
	fmt.Fprintln(stdout, resp)
	return statusToExit(stderr, resp)
}

func isoToTallyDate(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return "", fmt.Errorf("expected YYYY-MM-DD, got %q", s)
	}
	return t.Format("20060102"), nil
}
```

Then delete the `RunReport` stub line from `stubs.go`.

- [ ] **Step 3: Run tests, verify pass**

Run: `go test ./internal/cli/... -v`
Expected: report tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/cli/report.go internal/cli/report_test.go internal/cli/stubs.go
git commit -m "feat(cli): tally report with date conversion and named variables"
```

---

## Task 10: `tally ping` and `tally companies`

**Files:**
- Create: `internal/cli/ping.go`, `internal/cli/ping_test.go`
- Create: `internal/cli/companies.go`
- Modify: `internal/cli/stubs.go` (delete the file once empty)

- [ ] **Step 1: Failing test for ping**

Create `internal/cli/ping_test.go`:
```go
package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPingSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER><BODY><DATA/></BODY></ENVELOPE>`))
	}))
	defer srv.Close()

	code := runPingWithIO([]string{"--host", hostOf(srv.URL), "--port", portOf(srv.URL)},
		&bytes.Buffer{}, &bytes.Buffer{})
	if code != ExitOK {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestPingConnectError(t *testing.T) {
	// Use a port nothing listens on
	code := runPingWithIO([]string{"--host", "127.0.0.1", "--port", "1", "--timeout", "300ms"},
		&bytes.Buffer{}, &bytes.Buffer{})
	if code != ExitConnect && code != ExitTimeout {
		t.Fatalf("expected connect or timeout, got %d", code)
	}
}
```

- [ ] **Step 2: Implement ping.go**

Create `internal/cli/ping.go`:
```go
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/piyushgarg/tally-skill/internal/tally"
)

func RunPing(args []string) int { return runPingWithIO(args, os.Stdout, os.Stderr) }

func runPingWithIO(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ping", flag.ContinueOnError)
	fs.SetOutput(stderr)
	g := registerGlobals(fs)
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}

	body, err := tally.BuildCollection(tally.CollectionRequest{ID: "List of Companies"})
	if err != nil {
		fmt.Fprintf(stderr, "tally ping: %v\n", err)
		return ExitUsage
	}

	c := tally.NewClient(g.URL(), g.Timeout)
	resp, err := c.Post(context.Background(), body)
	if err != nil {
		return reportTransportError(stderr, err)
	}
	st := tally.ParseStatus(resp)
	if !st.Parsed {
		fmt.Fprintln(stderr, "tally: response not valid XML")
		return ExitBadResponse
	}
	if !st.Success() {
		fmt.Fprintf(stderr, "tally: %s\n", st.Message)
		return ExitTallyFailure
	}
	fmt.Fprintln(stdout, "tally: ok")
	return ExitOK
}
```

- [ ] **Step 3: Implement companies.go**

Create `internal/cli/companies.go`:
```go
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/piyushgarg/tally-skill/internal/tally"
)

func RunCompanies(args []string) int { return runCompaniesWithIO(args, os.Stdout, os.Stderr) }

func runCompaniesWithIO(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("companies", flag.ContinueOnError)
	fs.SetOutput(stderr)
	g := registerGlobals(fs)
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}

	body, err := tally.BuildCollection(tally.CollectionRequest{ID: "List of Companies"})
	if err != nil {
		fmt.Fprintf(stderr, "tally companies: %v\n", err)
		return ExitUsage
	}

	c := tally.NewClient(g.URL(), g.Timeout)
	resp, err := c.Post(context.Background(), body)
	if err != nil {
		return reportTransportError(stderr, err)
	}
	fmt.Fprintln(stdout, resp)
	return statusToExit(stderr, resp)
}
```

Delete `internal/cli/stubs.go` entirely.

- [ ] **Step 4: Run tests, verify pass**

Run: `go test ./...`
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git rm internal/cli/stubs.go
git add internal/cli/ping.go internal/cli/ping_test.go internal/cli/companies.go
git commit -m "feat(cli): tally ping and tally companies"
```

---

## Task 11: Pretty-print response option

**Files:**
- Modify: `internal/cli/raw.go` (replace `pretty` body)
- Create: `internal/cli/pretty.go`
- Create: `internal/cli/pretty_test.go`

- [ ] **Step 1: Failing test**

Create `internal/cli/pretty_test.go`:
```go
package cli

import (
	"strings"
	"testing"
)

func TestPrettyIndents(t *testing.T) {
	in := `<ENVELOPE><HEADER><STATUS>1</STATUS></HEADER></ENVELOPE>`
	out := pretty(in)
	if !strings.Contains(out, "\n  <HEADER>") {
		t.Errorf("expected indented HEADER:\n%s", out)
	}
}

func TestPrettyFallsBackOnInvalid(t *testing.T) {
	in := `not xml`
	if got := pretty(in); got != in {
		t.Errorf("invalid XML should pass through unchanged, got: %q", got)
	}
}
```

- [ ] **Step 2: Implement pretty.go**

Create `internal/cli/pretty.go`:
```go
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
```

Then in `internal/cli/raw.go`, **delete** the placeholder `pretty` function (it now lives in `pretty.go`). Apply `--pretty` to all subcommands by changing each subcommand's response print line from `fmt.Fprintln(stdout, resp)` to:
```go
out := resp
if g.Pretty {
	out = pretty(resp)
}
fmt.Fprintln(stdout, out)
```

(Apply this change in `object.go`, `collection.go`, `report.go`, `companies.go`.)

- [ ] **Step 3: Run, verify pass**

Run: `go test ./...`
Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add internal/cli/
git commit -m "feat(cli): --pretty XML response printing"
```

---

## Task 12: Makefile + cross-compile

**Files:**
- Create: `Makefile`
- Create: `bin/.gitkeep`

- [ ] **Step 1: Write Makefile**

Create `Makefile`:
```makefile
VERSION ?= 0.1.0
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION)"
PKG     := ./cmd/tally

.PHONY: build test clean build-all checksums

build:
	go build $(LDFLAGS) -o bin/tally $(PKG)

test:
	go test ./...

build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/tally-windows-amd64.exe $(PKG)

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/tally-darwin-arm64 $(PKG)

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/tally-darwin-amd64 $(PKG)

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/tally-linux-amd64 $(PKG)

build-all: build-windows build-darwin-arm64 build-darwin-amd64 build-linux-amd64

checksums:
	cd bin && shasum -a 256 tally-* > checksums.txt

clean:
	rm -f bin/tally bin/tally-* bin/checksums.txt
```

Create empty file `bin/.gitkeep`.

- [ ] **Step 2: Run all tests + build**

Run:
```bash
make test && make build && ./bin/tally help
```
Expected: tests pass; help text printed.

- [ ] **Step 3: Cross-compile**

Run: `make build-all && make checksums`
Expected: four binaries in `bin/`, plus `bin/checksums.txt`.

- [ ] **Step 4: Commit (no binaries yet — will be added in Task 14 release step)**

```bash
git add Makefile bin/.gitkeep
git commit -m "build: cross-compile Makefile"
```

---

## Task 13: Rewrite SKILL.md

**Files:**
- Modify: `SKILL.md` (full rewrite)

- [ ] **Step 1: Replace SKILL.md**

Replace the entire contents of `SKILL.md` with:
````markdown
---
name: tally-erp
description: Read-only access to a running TallyPrime instance over its built-in XML/HTTP gateway. Lets Claude query ledgers, vouchers, stock items, day book, trial balance, P&L, balance sheet, and other standard reports without crafting raw XML.
license: MIT
metadata:
  author: Piyush Garg
  version: "1.0.0"
---

# Tally ERP — Claude Skill

Talk to a running **TallyPrime** instance via its XML/HTTP gateway and pull accounting data (ledgers, vouchers, stock, reports). Read-only.

## Prerequisites

1. TallyPrime is running on the user's machine.
2. A company is loaded in TallyPrime.
3. The HTTP gateway is enabled — in TallyPrime: `F1 → Settings → Connectivity → Client/Server configuration → TallyPrime acts as Server`, with port (default `9000`).
4. Reachable at `http://<host>:9000` from where Claude runs.

## How Claude should use this skill

**Always invoke the bundled `tally` CLI; never `curl`/HTTP/XML by hand** unless the user asks for raw XML or a feature isn't covered by typed subcommands (then use `tally raw`).

Detect platform and pick the right binary:

```bash
case "$(uname -s)-$(uname -m)" in
  Darwin-arm64) BIN="$SKILL_DIR/bin/tally-darwin-arm64" ;;
  Darwin-x86_64) BIN="$SKILL_DIR/bin/tally-darwin-amd64" ;;
  Linux-x86_64) BIN="$SKILL_DIR/bin/tally-linux-amd64" ;;
  *) BIN="$SKILL_DIR/bin/tally-windows-amd64.exe" ;;
esac
```

(On a typical user machine running Tally, the Windows `.exe` is what you want.)

## Subcommands

All subcommands accept these **global flags**: `--host` (default `localhost`), `--port` (default `9000`), `--company`, `--timeout` (default `30s`), `--pretty`.

### `tally ping`
Confirm Tally is reachable and responding.

```bash
tally ping
# stdout: tally: ok
# exit 0 on success; 2 if unreachable; 4 on timeout
```

### `tally companies`
List loaded companies.

```bash
tally companies --pretty
```

### `tally object`
Export a single object.

```bash
tally object \
  --company "ABC Company Ltd" \
  --subtype Ledger \
  --id "Customer ABC" \
  --fetch Name,Parent,ClosingBalance,MailingName,Address
```

Subtypes: `Ledger`, `Group`, `StockItem`, `StockGroup`, `Voucher`, `CostCentre`, `Godown`, `Unit`, `Currency`, `VoucherType`, `Company`.

### `tally collection`
Export a list collection.

```bash
tally collection --company "ABC Company Ltd" --id "List of Ledgers"
```

Common collection IDs: `List of Companies`, `List of Groups`, `List of Ledgers`, `List of Cost Categories`, `List of Cost Centres`, `List of Stock Groups`, `List of Stock Categories`, `List of Stock Items`, `List of Godowns`, `List of Units`, `List of Voucher Types`, `List of Currencies`, `List of Budgets`.

### `tally report`
Export a standard report.

```bash
tally report --company "ABC" --id "Day Book" --from 2026-04-01 --to 2026-04-30
tally report --company "ABC" --id "Ledger" --ledger "Customer ABC" --from 2026-04-01 --to 2026-04-30
tally report --company "ABC" --id "Group Outstandings" --group "Sundry Debtors"
```

Common report IDs and required variables:

| Report ID | Required |
|---|---|
| `Day Book` | `--from`, `--to` |
| `Trial Balance` | `--from`, `--to` |
| `Profit and Loss` | `--from`, `--to` |
| `Balance Sheet` | `--from`, `--to` |
| `Ledger` | `--ledger`, `--from`, `--to` |
| `Ledger Outstandings` | `--ledger` |
| `Group Outstandings` | `--group` |
| `Bills Receivable` | `--from`, `--to` |
| `Bills Payable` | `--from`, `--to` |
| `Sales Register` | `--from`, `--to` |
| `Purchase Register` | `--from`, `--to` |
| `Cash Flow` | `--from`, `--to` |
| `Funds Flow` | `--from`, `--to` |
| `Stock Summary` | `--from`, `--to` |
| `Godown Summary` | `--from`, `--to` |
| `Movement Analysis` | `--from`, `--to` |
| `List of Accounts` | none |

For arbitrary additional `STATICVARIABLES`, use `--var KEY=VALUE` (repeatable). Use `--explode` to set `EXPLODEFLAG=Yes`.

### `tally raw`
Escape hatch — submits a complete `<ENVELOPE>` from stdin or `--file`. Use only when typed subcommands don't cover the case (custom TDL, exotic variables).

```bash
cat my-request.xml | tally raw
tally raw --file my-request.xml
```

The `templates/` directory contains ready-to-use envelope templates with `{{COMPANY}}`, `{{FROMDATE}}`, etc. placeholders that pair well with `tally raw`.

## Common Tally object fetch fields

| Subtype | Useful fetch fields |
|---|---|
| Ledger | Name, Parent, OpeningBalance, ClosingBalance, MailingName, Address, StateName, PinCode, Country, Email, LedgerPhone, LedgerMobile, GSTRegistrationType, PartyGSTIN, IsBillWiseOn |
| Group | Name, Parent, IsRevenue, IsDeemedPositive, AffectsGrossProfit |
| StockItem | Name, Parent, BaseUnits, AdditionalUnits, OpeningBalance, ClosingBalance, OpeningRate, OpeningValue, GSTApplicable, GSTTypeOfSupply |
| Voucher | Date, VoucherTypeName, VoucherNumber, Narration, PartyLedgerName, Amount, LedgerEntries.List, AllInventoryEntries.List |

## Static variables

| Variable | Format |
|---|---|
| `SVCURRENTCOMPANY` | string (set with `--company`) |
| `SVFROMDATE` | YYYYMMDD (CLI accepts YYYY-MM-DD via `--from`) |
| `SVTODATE` | YYYYMMDD (`--to`) |
| `LedgerName` | string (`--ledger`) |
| `GroupName` | string (`--group`) |
| `EXPLODEFLAG` | `Yes`/`No` (`--explode`) |
| `SVEXPORTFORMAT` | always set to `$$SysName:XML` by the CLI |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success — `<STATUS>1</STATUS>` |
| 1 | Tally returned `<STATUS>0</STATUS>` (full envelope still on stdout; reason on stderr) |
| 2 | Tally unreachable / connection refused |
| 3 | Bad CLI args |
| 4 | HTTP timeout |
| 5 | Response not valid XML |

## Failure response shapes

When `<STATUS>0</STATUS>`, Tally returns either plain text:

```xml
<ENVELOPE><HEADER><STATUS>0</STATUS></HEADER><BODY><DATA>DESC not found</DATA></BODY></ENVELOPE>
```

or a structured `<LINEERROR>`:

```xml
<ENVELOPE><HEADER><STATUS>0</STATUS></HEADER><BODY><DATA>
  <LINEERROR>Voucher totals do not match!</LINEERROR>...
</DATA></BODY></ENVELOPE>
```

The CLI surfaces the message on stderr; the full envelope is still on stdout.

## Templates

`templates/` ships ~30 reusable XML request envelopes with placeholders (`{{COMPANY}}`, `{{FROMDATE}}` in `YYYYMMDD`, `{{TODATE}}`, `{{LEDGER}}`, `{{GROUP}}`, `{{STOCKITEM}}`, `{{VOUCHERTYPE}}`, `{{VOUCHERNUMBER}}`). Use them as references when constructing custom `tally raw` requests.
````

- [ ] **Step 2: Commit**

```bash
git add SKILL.md
git commit -m "docs(skill): rewrite SKILL.md to instruct CLI usage"
```

---

## Task 14: Release — commit prebuilt binaries

**Files:**
- Modify: `.gitignore` (allow committed binaries)
- Add: `bin/tally-windows-amd64.exe`, `bin/tally-darwin-arm64`, `bin/tally-darwin-amd64`, `bin/tally-linux-amd64`, `bin/checksums.txt`

- [ ] **Step 1: Allow binaries in repo**

Edit `.gitignore` and replace the `bin/` block with:
```
# Allow committed release binaries
bin/tally
```
(The local `bin/tally` dev build stays ignored; cross-compiled releases are tracked.)

- [ ] **Step 2: Build and checksum**

Run:
```bash
make clean
make build-all
make checksums
```

- [ ] **Step 3: Commit binaries**

```bash
git add .gitignore bin/tally-windows-amd64.exe bin/tally-darwin-arm64 bin/tally-darwin-amd64 bin/tally-linux-amd64 bin/checksums.txt
git commit -m "release: prebuilt cross-platform binaries v0.1.0"
git tag v0.1.0
```

---

## Task 15: End-to-end manual smoke test

**Files:** none (operator runs against a real Tally instance)

- [ ] **Step 1: Verify connectivity**

On a machine with TallyPrime running:
```bash
./bin/tally-windows-amd64.exe ping
# expect: tally: ok
```

- [ ] **Step 2: Smoke each subcommand**

```bash
./bin/tally companies --pretty
./bin/tally collection --company "<your company>" --id "List of Ledgers" --pretty | head -50
./bin/tally object --company "<your company>" --subtype Ledger --id "<a real ledger>" --fetch Name,Parent,ClosingBalance --pretty
./bin/tally report --company "<your company>" --id "Trial Balance" --from 2026-04-01 --to 2026-04-30 --pretty | head -80
./bin/tally report --company "<your company>" --id "Day Book" --from 2026-04-01 --to 2026-04-30 --pretty | head -80
```

- [ ] **Step 3: Verify error handling**

```bash
./bin/tally --port 1 ping
# expect exit 2, "cannot reach server" on stderr
./bin/tally object --subtype Ledger --id "Definitely Does Not Exist" --company "<your company>"
# expect exit 1; stderr has Tally's error message
```

- [ ] **Step 4: If everything works, push the tag**

```bash
git push origin main --tags
```

If anything is wrong, file a fix as a follow-up plan; do NOT silently amend the release.

---

## Self-review notes

- **Spec coverage:** §1–§10 each map to at least one task. §5 (queryable surface) is documented in Task 13's SKILL.md rewrite. §11 open questions are resolved by Task 14 (commit binaries) and Task 1 (`git init`).
- **Type consistency:** `ObjectRequest`, `CollectionRequest`, `ReportRequest`, `Status`, `Client` are defined in Tasks 2–5 and used identically in Tasks 7–10.
- **Placeholder scan:** No "TBD"/"TODO"/"similar to" — every code step has full code.
- **YAGNI check:** No write/import paths, no daemon mode, no JSON output. `--pretty` deferred to Task 11 to keep earlier tasks lean.
