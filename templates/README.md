# Tally XML Request Templates

Read-only `Export` requests for TallyPrime's HTTP gateway (default `http://localhost:9000`).

## Placeholders

Replace before posting:

| Placeholder | Format | Example |
|---|---|---|
| `{{COMPANY}}` | Company name as loaded in Tally | `ABC Company Ltd` |
| `{{FROMDATE}}` | `YYYYMMDD` | `20260401` |
| `{{TODATE}}` | `YYYYMMDD` | `20260430` |
| `{{LEDGER}}` | Ledger name | `Customer ABC` |
| `{{GROUP}}` | Group name | `Sundry Debtors` |
| `{{STOCKITEM}}` | Stock item name | `Sony TV 14"` |
| `{{VOUCHERTYPE}}` | Voucher type name | `Sales` |
| `{{VOUCHERNUMBER}}` | Voucher number | `1` |

## Layout

- `utility/` — health checks, company list
- `collections/` — list-style queries (`List of Ledgers`, etc.)
- `objects/` — single-object queries with `<FETCHLIST>`
- `reports/` — standard reports (Day Book, Trial Balance, P&L, etc.)

## Usage

```bash
# Pipe through the CLI's raw passthrough:
sed -e 's/{{COMPANY}}/ABC Company Ltd/' \
    -e 's/{{FROMDATE}}/20260401/' \
    -e 's/{{TODATE}}/20260430/' \
    templates/reports/day_book.xml \
  | tally raw
```
