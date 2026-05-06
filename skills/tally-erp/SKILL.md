---
name: tally-erp
description: Read-only access to a running TallyPrime instance over its built-in XML/HTTP gateway. Lets Claude query ledgers, vouchers, stock items, day book, trial balance, P&L, balance sheet, and other standard reports without crafting raw XML.
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

The binaries are in the `bin/` directory relative to this skill's root. Pick the binary matching the OS where the agent is running:

| OS | Binary |
|---|---|
| Windows | `bin/tally-windows-amd64.exe` |
| macOS (Apple Silicon) | `bin/tally-darwin-arm64` |
| macOS (Intel) | `bin/tally-darwin-amd64` |
| Linux (x86_64) | `bin/tally-linux-amd64` |
| Linux (ARM64) | `bin/tally-linux-arm64` |

To detect the OS at runtime, run `uname -s` (returns `Darwin`, `Linux`, or use `.exe` on Windows) and `uname -m` (returns `arm64` or `x86_64`). Determine the skill root from the path of this SKILL.md file and run the binary from there:

```bash
# SKILL_DIR is the directory containing this SKILL.md
SKILL_DIR="<path-to-this-skill>"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
[ "$ARCH" = "x86_64" ] && ARCH="amd64"
[ "$OS" = "darwin" ] || [ "$OS" = "linux" ] && TALLY="$SKILL_DIR/bin/tally-${OS}-${ARCH}" || TALLY="$SKILL_DIR/bin/tally-windows-amd64.exe"

$TALLY ping
$TALLY report --company "ABC" --id "Day Book" --from 2026-04-01 --to 2026-04-30
```

TallyPrime itself only runs on Windows, but the CLI can run from any OS as long as the Tally gateway is network-reachable (use `--host` to point to the Windows machine).

(Throughout this document the binary is referred to as `tally` for brevity.)

## Subcommands

All subcommands accept these **global flags**: `--scheme` (default `http`; set `https` for TLS), `--host` (default `localhost`), `--port` (default `9000`), `--company`, `--timeout` (default `30s`), `--pretty`.

### `tally ping`
Confirm Tally is reachable and responding. Does **not** require `--company` — pings the gateway with a `List of Companies` request, so it works even when no company is loaded.

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

**Filtering flags** (use these to avoid fetching all items from large collections):

| Flag | Purpose |
|---|---|
| `--parent` | Filter to children of a group (TDL `CHILD OF`). E.g. `--parent "Sundry Debtors"` |
| `--fields` | Comma-separated fields to return via TDL `NATIVEMETHOD`. E.g. `--fields Name,Parent,ClosingBalance` |
| `--filter` | Raw TDL filter expression. E.g. `--filter "$ClosingBalance > 10000"` |
| `--fetch` | Comma-separated FETCHLIST fields (standard XML mechanism) |

```bash
# Only ledgers under Sundry Debtors, returning just Name and ClosingBalance
tally collection --company "ABC" --id "List of Ledgers" \
    --parent "Sundry Debtors" --fields Name,Parent,ClosingBalance

# Stock items with a filter expression
tally collection --company "ABC" --id "List of Stock Items" \
    --filter "$ClosingBalance > 0" --fields Name,Parent,ClosingBalance
```

Common collection IDs: `List of Companies`, `List of Groups`, `List of Ledgers`, `List of Cost Categories`, `List of Cost Centres`, `List of Stock Groups`, `List of Stock Categories`, `List of Stock Items`, `List of Godowns`, `List of Voucher Types`, `List of Budgets`.

> ⚠️ **Avoid the built-in `List of Currencies`, `List of Units`, and `List of Vouchers` collection IDs** — they crash TallyPrime when exported as XML. Use the bundled templates instead, which substitute custom TDL collections (`TYPE=Currency`, `TYPE=Unit`, `TYPE=Voucher`):
>
> ```bash
> tally template --name collections/list_currencies --company "ABC"
> tally template --name collections/list_units --company "ABC"
> tally template --name collections/list_vouchers_dated --company "ABC" --from 2026-04-01 --to 2026-04-30
> ```

### `tally report`
Export a standard report.

```bash
tally report --company "ABC" --id "Day Book" --from 2026-04-01 --to 2026-04-30
tally report --company "ABC" --id "Ledger" --ledger "Customer ABC" --from 2026-04-01 --to 2026-04-30
tally report --company "ABC" --id "Group Outstandings" --group "Sundry Debtors"

# Filter Day Book to only Sales vouchers
tally report --company "ABC" --id "Day Book" --from 2026-04-01 --to 2026-04-30 --voucher-type Sales

# Arbitrary TDL filter on a report
tally report --company "ABC" --id "Day Book" --from 2026-04-01 --to 2026-04-30 --filter "$Amount > 50000"
```

**Filtering flags:**

| Flag | Purpose |
|---|---|
| `--voucher-type` | Filter report by voucher type name (e.g. `Sales`, `Purchase`, `Payment`) |
| `--filter` | Raw TDL filter expression |
| `--chunk` | Date-based chunking: `daily`, `weekly`, or `monthly`. Splits the date range into sub-requests and merges results. Use for large companies where a single full-year request may timeout. |

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

### `tally template`
Load a bundled XML template, substitute `{{KEY}}` placeholders from CLI flags, and POST it to Tally. This is the recommended way to use the bundled `templates/` envelopes — much nicer than `sed` + `tally raw`.

```bash
tally template --name reports/day_book \
    --company "ABC Co" \
    --from 2026-04-01 --to 2026-04-30
```

Substitution flags (in addition to globals):

| Flag | Placeholder |
|---|---|
| `--name` (required) | (selects template; relative path under `templates/`, `.xml` optional) |
| `--templates-dir` | (override; default auto: `<exe>/../templates`, then `./templates`) |
| `--company` (global) | `{{COMPANY}}` |
| `--from` | `{{FROMDATE}}` (ISO `YYYY-MM-DD` -> Tally `YYYYMMDD`) |
| `--to` | `{{TODATE}}` |
| `--ledger` | `{{LEDGER}}` |
| `--group` | `{{GROUP}}` |
| `--stockitem` | `{{STOCKITEM}}` |
| `--vouchertype` | `{{VOUCHERTYPE}}` |
| `--vouchernumber` | `{{VOUCHERNUMBER}}` |
| `--chunk` | Date chunking: `daily`, `weekly`, or `monthly` (splits date range, merges results) |
| `--var KEY=VALUE` | `{{KEY}}` (repeatable, for any other placeholder) |

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

## Troubleshooting: Tally Unreachable (exit code 2)

When `tally ping` fails with exit code 2 (connection refused / unreachable), ask the user:

1. **Is TallyPrime currently running?** It must be open with a company loaded.
2. **What port is TallyPrime's HTTP server running on?** The default is `9000`, but it may differ.

Then provide these instructions so the user can verify and configure their TallyPrime HTTP server:

> **How to check/enable the TallyPrime HTTP server:**
>
> 1. In TallyPrime, press **F1** (Help).
> 2. Press **S** to open **Settings**.
> 3. Press **N** to open **Connectivity**.
> 4. Select **Client/Server Configuration**.
> 5. Set **Tally Acts as** → **Server**.
> 6. Set **Enable ODBC Server** → **Yes**.
> 7. Note the **Port** number shown (default `9000`) — this is the port to use with `--port`.
> 8. Accept/save the settings and restart TallyPrime if prompted.

Once the user confirms the port, retry with the correct port:

```bash
tally ping --port <user-provided-port>
```

If the port differs from `9000`, use `--port` on all subsequent commands.

## Failure response shapes

When `<STATUS>0</STATUS>`, Tally returns one of three formats:

**Plain text:**
```xml
<ENVELOPE><HEADER><STATUS>0</STATUS></HEADER><BODY><DATA>DESC not found</DATA></BODY></ENVELOPE>
```

**Structured `<LINEERROR>`:**
```xml
<ENVELOPE><HEADER><STATUS>0</STATUS></HEADER><BODY><DATA>
  <LINEERROR>Voucher totals do not match!</LINEERROR>...
</DATA></BODY></ENVELOPE>
```

**Structured `<STATUS.LIST>` with error code:**
```xml
<ENVELOPE><HEADER><STATUS>0</STATUS></HEADER><BODY><DATA>
  <STATUS.LIST><STATUS><CODE>123</CODE><DESC>Invalid Request</DESC></STATUS></STATUS.LIST>
</DATA></BODY></ENVELOPE>
```

The CLI parses all three formats and surfaces the message on stderr; the full envelope is still on stdout.

## Large data guidance

Tally companies can have thousands of ledgers, stock items, and vouchers. **Always scope your queries** to avoid overwhelming the response:

1. **Use `--parent` for collections** to filter by group hierarchy instead of fetching all items.
2. **Use `--fields` for collections** to return only the fields you need (e.g. `--fields Name,ClosingBalance`) instead of all properties.
3. **Use narrow date ranges** for reports like Day Book, Sales Register, etc. Prefer monthly ranges over full-year queries.
4. **Use `--voucher-type`** when querying Day Book or similar reports to filter by transaction type.
5. **Use `--filter`** for arbitrary TDL filter expressions when you need custom scoping.
6. **Prefer `tally object`** (single item by name) over `tally collection` (all items) when you know the specific entity name.
7. **Use `--chunk monthly`** on `tally report` or `tally template` when fetching a full financial year from a large company. This splits the request into monthly sub-requests and merges the results, avoiding timeouts.

**TDL filter expression syntax** (for `--filter`):
- Field access: `$FieldName` (e.g. `$ClosingBalance`, `$Name`, `$VoucherTypeName`)
- Comparison: `=`, `>`, `<`, `>=`, `<=`, `<>` (not equal)
- Boolean: `AND`, `OR`, `NOT`
- Functions: `$$IsLedgerProfit`, `$$IsGroup`, etc.
- Example: `$ClosingBalance > 10000 AND $Parent = "Sundry Debtors"`

## Templates

`templates/` ships ~33 reusable XML request envelopes with placeholders (`{{COMPANY}}`, `{{FROMDATE}}` in `YYYYMMDD`, `{{TODATE}}`, `{{LEDGER}}`, `{{GROUP}}`, `{{STOCKITEM}}`, `{{STOCKGROUP}}`, `{{VOUCHERTYPE}}`, `{{VOUCHERNUMBER}}`).

The recommended way to invoke them is `tally template --name <relative/path>` (see above), which handles placeholder substitution, XML escaping, and date conversion for you. Unfilled placeholders generate a warning on stderr.

**Filtered collection templates** (use these to scope large collections):

| Template | Placeholders | Purpose |
|---|---|---|
| `collections/list_ledgers_by_group` | `{{COMPANY}}`, `{{GROUP}}` | Ledgers under a specific group |
| `collections/list_stock_items_by_group` | `{{COMPANY}}`, `{{STOCKGROUP}}` | Stock items under a specific stock group |
| `collections/list_vouchers_by_type` | `{{COMPANY}}`, `{{FROMDATE}}`, `{{TODATE}}`, `{{VOUCHERTYPE}}` | Vouchers filtered by type (Sales, Purchase, etc.) |
| `collections/list_vouchers_with_items` | `{{COMPANY}}`, `{{FROMDATE}}`, `{{TODATE}}`, `{{VOUCHERTYPE}}` | Vouchers with inventory item details (STOCKITEMNAME, ACTUALQTY, RATE, AMOUNT) |

**Custom-TDL collection templates** (workarounds for built-in report IDs that crash Tally):

| Template | Placeholders | Why custom |
|---|---|---|
| `collections/list_currencies` | `{{COMPANY}}` | Built-in `List of Currencies` crashes Tally; uses `TYPE=Currency` |
| `collections/list_units` | `{{COMPANY}}` | Built-in `List of Units` crashes Tally; uses `TYPE=Unit` |
| `collections/list_vouchers_dated` | `{{COMPANY}}`, `{{FROMDATE}}`, `{{TODATE}}` | Built-in `List of Vouchers` crashes Tally; uses TDL `TYPE=Voucher` collection |

Templates can also be used as references when constructing custom `tally raw` requests.
