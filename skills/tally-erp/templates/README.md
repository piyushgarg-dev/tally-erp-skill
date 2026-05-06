# Tally XML Request Templates

Read-only `Export` requests for TallyPrime's HTTP gateway (default `http://localhost:9000`).

## Placeholders

Replace before posting (values are XML-escaped automatically when using `tally template`):

| Placeholder | Format | Example |
|---|---|---|
| `{{COMPANY}}` | Company name as loaded in Tally | `ABC Company Ltd` |
| `{{FROMDATE}}` | `YYYYMMDD` | `20260401` |
| `{{TODATE}}` | `YYYYMMDD` | `20260430` |
| `{{LEDGER}}` | Ledger name | `Customer ABC` |
| `{{GROUP}}` | Group name | `Sundry Debtors` |
| `{{STOCKITEM}}` | Stock item name | `Sony TV 14"` |
| `{{STOCKGROUP}}` | Stock group name | `Electronics` |
| `{{VOUCHERTYPE}}` | Voucher type name | `Sales` |
| `{{VOUCHERNUMBER}}` | Voucher number | `1` |

## Layout

- `utility/` — health checks, company list
- `collections/` — list-style queries (`List of Ledgers`, etc.)
  - Includes **filtered templates** that use TDL to scope by group or voucher type
- `objects/` — single-object queries with `<FETCHLIST>`
- `reports/` — standard reports (Day Book, Trial Balance, P&L, etc.)

### Filtered collection templates

These templates use inline TDL (`ISMODIFY="Yes"`) to filter collections server-side, avoiding large responses:

| Template | Placeholders | Purpose |
|---|---|---|
| `collections/list_ledgers_by_group.xml` | `{{COMPANY}}`, `{{GROUP}}` | Ledgers under a specific group |
| `collections/list_stock_items_by_group.xml` | `{{COMPANY}}`, `{{STOCKGROUP}}` | Stock items under a specific stock group |
| `collections/list_vouchers_by_type.xml` | `{{COMPANY}}`, `{{FROMDATE}}`, `{{TODATE}}`, `{{VOUCHERTYPE}}` | Day Book filtered by voucher type |

### Custom-TDL collection templates (Tally crash workarounds)

The built-in collection IDs `List of Currencies`, `List of Units`, and `List of Vouchers` crash TallyPrime when exported as XML. These templates define the collection inline via `<TDLMESSAGE>` with the underlying object `<TYPE>` (`Currency`, `Unit`, `Voucher`) — bypassing the buggy report:

| Template | Placeholders | Replaces |
|---|---|---|
| `collections/list_currencies.xml` | `{{COMPANY}}` | `List of Currencies` |
| `collections/list_units.xml` | `{{COMPANY}}` | `List of Units` |
| `collections/list_vouchers_dated.xml` | `{{COMPANY}}`, `{{FROMDATE}}`, `{{TODATE}}` | `List of Vouchers` |

### Ping

`utility/ping.xml` queries `List of Companies` with no `SVCURRENTCOMPANY` — useful as a connectivity check that works even when no company is loaded.

## Usage

The recommended way to use templates is via `tally template`:

```bash
tally template --name collections/list_ledgers_by_group \
    --company "ABC Company Ltd" --group "Sundry Debtors"

tally template --name collections/list_vouchers_by_type \
    --company "ABC Company Ltd" --from 2026-04-01 --to 2026-04-30 \
    --vouchertype Sales
```

Alternatively, pipe through `tally raw`:

```bash
sed -e 's/{{COMPANY}}/ABC Company Ltd/' \
    -e 's/{{FROMDATE}}/20260401/' \
    -e 's/{{TODATE}}/20260430/' \
    templates/reports/day_book.xml \
  | tally raw
```

## Writing custom TDL-filtered templates

To create your own filtered collection template, use the `ISMODIFY` pattern:

```xml
<TDL>
  <TDLMESSAGE>
    <COLLECTION NAME="List of Ledgers" ISMODIFY="Yes">
      <ADD>CHILD OF : {{GROUP}}</ADD>
      <NATIVEMETHOD>Name</NATIVEMETHOD>
      <NATIVEMETHOD>ClosingBalance</NATIVEMETHOD>
      <FILTERS>MyFilter</FILTERS>
    </COLLECTION>
    <SYSTEM TYPE="Formulae" NAME="MyFilter">$ClosingBalance > 0</SYSTEM>
  </TDLMESSAGE>
</TDL>
```

Key TDL elements:
- `ISMODIFY="Yes"` modifies the built-in collection instead of creating a new one
- `<ADD>CHILD OF : GroupName</ADD>` filters to children of a group
- `<NATIVEMETHOD>` specifies which fields to return
- `<FILTERS>` + `<SYSTEM TYPE="Formulae">` adds a filter expression
