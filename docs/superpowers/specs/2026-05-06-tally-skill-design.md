# Tally ERP Prime Skill for Claude Code — Design

**Date:** 2026-05-06
**Author:** Piyush Garg
**Status:** Draft for review

## 1. Goal

Build a Claude Code skill that lets Claude query a running TallyPrime instance over its built-in XML/HTTP gateway. The skill ships with a precompiled Go CLI (`tally`) that owns all XML envelope construction and HTTP transport. Claude invokes the CLI with structured arguments and reads Tally's raw XML response from stdout.

**Scope:** read-only. No imports, creates, alters, deletes, or executions. Operations are limited to TallyPrime's `Export` request type (`OBJECT`, `COLLECTION`, `DATA`).

## 2. Architecture

```
Claude (in Claude Code)
    │ runs Bash
    ▼
tally CLI (Go binary, shipped in skill)
    │ POST text/xml
    ▼
TallyPrime HTTP server (localhost:9000 by default)
```

- **Claude ↔ CLI channel:** CLI subcommands and flags in, raw XML on stdout, human summary on stderr, exit code carries success/failure class.
- **CLI ↔ Tally channel:** HTTP POST of a `<ENVELOPE>` body to `http://<host>:<port>/`, response body is `<ENVELOPE>` with `<STATUS>` and `<DATA>`.

The CLI is a transport and envelope builder. It does not parse, transform, or filter Tally's response payloads — fidelity to Tally's schema is preserved end-to-end.

## 3. Folder layout

```
tally-skill/
├── SKILL.md                          # Skill instructions for Claude
├── bin/
│   ├── tally-windows-amd64.exe       # Primary target (Tally is Windows-only)
│   ├── tally-darwin-arm64            # For dev / Mac→Windows-Tally over LAN
│   └── tally-linux-amd64             # For dev / CI
├── cmd/tally/
│   └── main.go                       # CLI entrypoint, flag parsing, dispatch
├── internal/
│   ├── tally/                        # Envelope structs, HTTP client, status parsing
│   ├── export/                       # object / collection / report builders
│   └── xmlutil/                      # XML escape, pretty-print, status extraction
├── examples/                         # Sample response XML captures
├── templates/                        # Reusable XML request templates with placeholders
│   ├── README.md
│   ├── utility/                      # ping, list_companies
│   ├── collections/                  # list_ledgers, list_groups, list_stock_items, ...
│   ├── objects/                      # ledger, group, stock_item, voucher
│   └── reports/                      # day_book, trial_balance, profit_and_loss, ...
├── docs/superpowers/specs/           # This document and future specs
├── go.mod
├── go.sum
└── Makefile                          # Cross-compile all platforms; checksum
```

Binaries are committed so the skill works with no build step on the user's machine. `SKILL.md` selects the right binary by detecting OS at runtime.

## 4. CLI surface

All commands accept the **global flags** below. Output is **raw XML to stdout**, error/diagnostic line to stderr, exit code per Section 7.

### 4.1 Global flags

| Flag | Default | Purpose |
|---|---|---|
| `--host` | `localhost` | Tally HTTP server host |
| `--port` | `9000` | Tally HTTP server port |
| `--company` | (empty) | Sets `SVCURRENTCOMPANY` static variable |
| `--timeout` | `30s` | HTTP request timeout |
| `--pretty` | `false` | Pretty-print response XML |
| `--version` | — | Print CLI version |

### 4.2 Subcommands

#### `tally ping`
Health check. Sends a minimal `Export Collection` request for `List of Companies` and verifies a `<STATUS>1</STATUS>` response. Exits 0 if Tally is reachable and a company is loaded.

#### `tally companies`
Convenience wrapper around `Export Collection` for `List of Companies`. Returns the loaded company list.

#### `tally object`
Export a single object (Ledger, Stock Item, Voucher, Group, etc.).

| Flag | Required | Purpose |
|---|---|---|
| `--subtype` | yes | Object subtype (`Ledger`, `Group`, `StockItem`, `Voucher`, `CostCentre`, `Godown`, `Unit`, `Currency`, `VoucherType`, `Company`) |
| `--id` | yes | Object identifier (typically the Name) |
| `--id-type` | no | `Name` (default) or other Tally-supported ID type |
| `--fetch` | no | Comma-separated field list for `<FETCHLIST>` (e.g., `Name,Parent,ClosingBalance`). Omit to fetch all default fields. |

Example:
```
tally object --subtype Ledger --id "Customer ABC" --fetch Name,Parent,ClosingBalance
```

#### `tally collection`
Export a list collection.

| Flag | Required | Purpose |
|---|---|---|
| `--id` | yes | Collection name (`List of Ledgers`, `List of Groups`, `List of Stock Items`, etc.) |
| `--fetch` | no | Optional `<FETCHLIST>` fields |
| `--filter` | no | Optional system filter name |

Example:
```
tally collection --id "List of Ledgers" --fetch Name,Parent,ClosingBalance
```

#### `tally report`
Export a standard report.

| Flag | Required | Purpose |
|---|---|---|
| `--id` | yes | Report name (`Day Book`, `Trial Balance`, `Profit and Loss`, `Balance Sheet`, `Ledger`, `Ledger Outstandings`, `Bills Payable`, `Bills Receivable`, `Group Outstandings`, `Sales Register`, `Purchase Register`, `Stock Summary`, etc.) |
| `--from` | conditional | `SVFROMDATE` (ISO `YYYY-MM-DD`; converted to Tally `YYYYMMDD`) |
| `--to` | conditional | `SVTODATE` |
| `--ledger` | conditional | `LedgerName` static variable (required by `Ledger`, `Ledger Outstandings`) |
| `--group` | conditional | `GroupName` static variable (required by `Group Outstandings`) |
| `--explode` | no | Sets `EXPLODEFLAG=Yes` for detailed mode |
| `--var KEY=VAL` | no | Repeatable; injects an arbitrary `STATICVARIABLE` for reports needing variables not covered by named flags |

Example:
```
tally report --id "Day Book" --from 2026-04-01 --to 2026-04-30
tally report --id "Ledger" --ledger "Customer ABC" --from 2026-04-01 --to 2026-04-30
```

#### `tally raw`
Passthrough escape hatch. Reads a complete `<ENVELOPE>` XML from stdin or `--file <path>`, POSTs it to Tally, returns the response. Used when the typed subcommands don't cover a request (custom TDL, exotic variables).

```
cat my-request.xml | tally raw
tally raw --file my-request.xml
```

## 5. Comprehensive queryable surface (in SKILL.md)

`SKILL.md` includes a full reference so Claude can pick the right command without external lookups.

### 5.1 Reports (`tally report --id ...`)

| Report ID | Required variables | Notes |
|---|---|---|
| Day Book | `SVFROMDATE`, `SVTODATE` | All vouchers in date range |
| Trial Balance | `SVFROMDATE`, `SVTODATE` | Group-wise balances |
| Profit and Loss | `SVFROMDATE`, `SVTODATE` | |
| Balance Sheet | `SVFROMDATE`, `SVTODATE` | |
| Ledger | `LedgerName`, `SVFROMDATE`, `SVTODATE` | Voucher-level ledger detail |
| Ledger Outstandings | `LedgerName` | Bill-wise pending |
| Group Outstandings | `GroupName` | |
| Bills Payable | `SVTODATE` | |
| Bills Receivable | `SVTODATE` | |
| Sales Register | `SVFROMDATE`, `SVTODATE` | |
| Purchase Register | `SVFROMDATE`, `SVTODATE` | |
| Cash Flow | `SVFROMDATE`, `SVTODATE` | |
| Funds Flow | `SVFROMDATE`, `SVTODATE` | |
| Stock Summary | `SVFROMDATE`, `SVTODATE` | |
| Godown Summary | `SVFROMDATE`, `SVTODATE` | |
| Movement Analysis | `SVFROMDATE`, `SVTODATE` | |
| List of Accounts | none | All masters dump |

### 5.2 Collections (`tally collection --id ...`)

`List of Companies`, `List of Groups`, `List of Ledgers`, `List of Cost Categories`, `List of Cost Centres`, `List of Stock Groups`, `List of Stock Categories`, `List of Stock Items`, `List of Godowns`, `List of Units`, `List of Voucher Types`, `List of Currencies`, `List of Budgets`, `List of Vouchers`.

### 5.3 Objects (`tally object --subtype ...`)

| Subtype | Common fetch fields |
|---|---|
| `Ledger` | Name, Parent, OpeningBalance, ClosingBalance, MailingName, Address, StateName, PinCode, Country, Email, LedgerPhone, LedgerMobile, GSTRegistrationType, PartyGSTIN, IsBillWiseOn |
| `Group` | Name, Parent, IsRevenue, IsDeemedPositive, AffectsGrossProfit |
| `StockItem` | Name, Parent, BaseUnits, AdditionalUnits, OpeningBalance, ClosingBalance, OpeningRate, OpeningValue, GSTApplicable, GSTTypeOfSupply |
| `StockGroup` | Name, Parent |
| `Voucher` | Date, VoucherTypeName, VoucherNumber, Narration, PartyLedgerName, Amount, LedgerEntries.List, AllInventoryEntries.List |
| `CostCentre` | Name, Parent, Category |
| `Godown` | Name, Parent, Address |
| `Unit` | Name, IsSimpleUnit, BaseUnits, AdditionalUnits, Conversion |
| `Currency` | Name, OriginalName, MailingName, DecimalPlaces, IsBaseCurrency |
| `VoucherType` | Name, Parent, NumberingMethod, IsDeemedPositive |
| `Company` | Name, MailingName, Address, StartingFrom, BooksFrom |

### 5.4 Static variables reference

| Variable | Format | Purpose |
|---|---|---|
| `SVCURRENTCOMPANY` | string | Active company name |
| `SVFROMDATE` | YYYYMMDD | Period start |
| `SVTODATE` | YYYYMMDD | Period end |
| `SVEXPORTFORMAT` | `$$SysName:XML` | XML output (default in CLI) |
| `LedgerName` | string | Required by Ledger reports |
| `GroupName` | string | Required by Group reports |
| `EXPLODEFLAG` | `Yes`/`No` | Detailed expansion |

### 5.5 Common error codes (`<STATUS>0</STATUS>` body)

The CLI extracts `<CODE>` and `<DESC>` from failure envelopes and echoes them on stderr. Known codes will be tabulated in `SKILL.md` after empirical testing during implementation.

## 5.6 Templates folder

`templates/` ships ready-to-use XML request envelopes with `{{PLACEHOLDER}}` markers. They serve three purposes:

1. **Source of truth** for the Go envelope builders — the binary's generated XML must match these byte-for-byte (modulo placeholder substitution).
2. **Reference for Claude** when crafting `tally raw` requests for cases not covered by typed subcommands.
3. **Documentation** — readable XML examples for users new to the Tally schema.

Placeholders: `{{COMPANY}}`, `{{FROMDATE}}` (YYYYMMDD), `{{TODATE}}`, `{{LEDGER}}`, `{{GROUP}}`, `{{STOCKITEM}}`, `{{VOUCHERTYPE}}`, `{{VOUCHERNUMBER}}`.

Files included:

- **utility/**: `ping.xml`, `list_companies.xml`
- **collections/**: `list_ledgers`, `list_groups`, `list_stock_items`, `list_stock_groups`, `list_godowns`, `list_voucher_types`, `list_cost_centres`, `list_units`, `list_currencies`, `list_vouchers_dated`
- **objects/**: `ledger`, `group`, `stock_item`, `voucher`
- **reports/**: `day_book`, `trial_balance`, `profit_and_loss`, `balance_sheet`, `ledger`, `ledger_outstandings`, `group_outstandings`, `bills_receivable`, `bills_payable`, `sales_register`, `purchase_register`, `cash_flow`, `funds_flow`, `stock_summary`, `godown_summary`, `movement_analysis`, `list_of_accounts`

## 6. SKILL.md structure (replaces current file)

1. Frontmatter (`name`, `description`, `license`, `metadata`).
2. **What it does** — 1 paragraph.
3. **Prerequisites** — TallyPrime running, company loaded, HTTP server enabled on port 9000 (F1 → Settings → Connectivity → Client/Server configuration → TallyPrime acts as Server, Port 9000).
4. **How Claude should use this skill** — always invoke `bin/tally-<os>-<arch>` (with OS detection snippet); never craft raw HTTP/XML; prefer typed subcommands; fall back to `tally raw` only for unsupported cases.
5. **Subcommand reference** — example invocation + truncated response per subcommand.
6. **Queryable surface** — Section 5 content.
7. **Error handling** — exit codes, how to interpret `<STATUS>0</STATUS>` responses.
8. **Examples** — pointer to `examples/` for full request/response XML samples.

## 7. Error handling & exit codes

| Exit | Condition | stderr | stdout |
|---|---|---|---|
| 0 | `<STATUS>1</STATUS>` | (silent) | full XML response |
| 1 | `<STATUS>0</STATUS>` (Tally-reported failure) | `tally error: code=<CODE> desc=<DESC>` | full XML response |
| 2 | Connection error / Tally unreachable | `tally: cannot reach <host>:<port>: <err>` | (empty) |
| 3 | Invalid CLI arguments | usage message | (empty) |
| 4 | HTTP timeout | `tally: timeout after <duration>` | (empty) |
| 5 | Malformed response from Tally (not valid XML) | `tally: invalid response: <err>` | raw bytes received |

stdout is always reserved for the response body so Claude can pipe it to other tools.

## 8. Build & distribution

- `Makefile` targets: `build-windows`, `build-darwin`, `build-linux`, `build-all`, `checksums`, `clean`, `test`.
- Cross-compile via standard `GOOS`/`GOARCH`. No CGO.
- Each binary committed to `bin/`. SHA-256 checksums recorded in `bin/checksums.txt`.
- Versioning via `-ldflags "-X main.version=..."`.

## 9. Testing strategy

- **Unit tests** for envelope builders and XML escaping (table-driven, no Tally needed).
- **Integration tests** behind a `-tags integration` build tag, hitting a configurable Tally instance. Skipped by default in CI.
- **Recorded fixtures** in `examples/` capturing real Tally responses for the major report/collection/object queries — used both as documentation and as parser-stability test fixtures.

## 10. Out of scope (explicit non-goals)

- Write operations of any kind (`Import`, `Execute`, `Action`, masters/voucher creation, alteration, cancellation).
- TDL authoring helpers (custom reports must be hand-written and submitted via `tally raw`).
- Response transformation/filtering, JSON conversion, formatting, or summarization.
- Authentication beyond Tally's local HTTP gateway (Tally Prime currently has no per-request auth).
- Cross-company joins; one company per invocation via `--company`.
- A long-running daemon mode.

## 11. Open questions

- Confirm the precise list of report IDs that work without TDL on a fresh TallyPrime (5.x) install — to be verified during implementation and `SKILL.md` polished against actual outputs.
- Confirm whether `bin/` checked-in binaries are acceptable for the user's distribution channel (vs. a `make build` step on first use).
