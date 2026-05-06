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

TallyPrime only runs on Windows, so the bundled binary is `bin/tally-windows-amd64.exe`. Use that path directly:

```bash
BIN="$SKILL_DIR/bin/tally-windows-amd64.exe"
```

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

`templates/` ships ~30 reusable XML request envelopes with placeholders (`{{COMPANY}}`, `{{FROMDATE}}` in `YYYYMMDD`, `{{TODATE}}`, `{{LEDGER}}`, `{{GROUP}}`, `{{STOCKITEM}}`, `{{VOUCHERTYPE}}`, `{{VOUCHERNUMBER}}`).

The recommended way to invoke them is `tally template --name <relative/path>` (see above), which handles placeholder substitution and date conversion for you. They can also be used as references when constructing custom `tally raw` requests.
