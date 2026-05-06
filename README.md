# tally-erp

A [Claude Code](https://claude.com/claude-code) skill that gives Claude read-only access to a running **TallyPrime** instance over its built-in XML/HTTP gateway.

Ask Claude things like:

> *"Show me the Day Book for last month."*
> *"What's the closing balance on Customer ABC's ledger?"*
> *"List all stock items under the Electronics group."*
> *"Pull the trial balance for FY2025–26."*

Claude calls a bundled Go CLI (`tally`) which constructs the right XML envelope, talks to TallyPrime over HTTP, and returns the response. You don't write XML, Claude doesn't hallucinate URLs.

## Status

Read-only. Queries Tally; never writes, alters, or cancels anything.

## Prerequisites

1. **TallyPrime** running on Windows.
2. A company **loaded** in TallyPrime.
3. **HTTP gateway enabled** — in TallyPrime: `F1 → Settings → Connectivity → Client/Server configuration → TallyPrime acts as Server`, default port `9000`.
4. Reachable at `http://<host>:9000` from where Claude Code runs.

## Install

### As a Claude Code plugin (recommended)

Add this repo as a marketplace and install the plugin:

```text
/plugin marketplace add piyushgarg/tally-skill
/plugin install tally-erp@piyushgarg-tally
```

The plugin's `bin/` directory is added to `PATH` automatically while the plugin is enabled, so the CLI is available as `tally-windows-amd64.exe` in any session.

### Local development / testing

Clone and load directly with `--plugin-dir`:

```bash
git clone https://github.com/piyushgarg/tally-skill
claude --plugin-dir ./tally-skill
```

Run `/reload-plugins` to pick up edits without restarting.

### Standalone CLI use

The CLI binary works without Claude too:

```bash
./bin/tally-windows-amd64.exe ping
./bin/tally-windows-amd64.exe companies --pretty
./bin/tally-windows-amd64.exe report --company "ABC" --id "Trial Balance" --from 2026-04-01 --to 2026-04-30 --pretty
```

## What you can query

**Reports:** Day Book, Trial Balance, Profit & Loss, Balance Sheet, Ledger statement, Ledger/Group Outstandings, Bills Receivable/Payable, Sales/Purchase Register, Cash Flow, Funds Flow, Stock Summary, Godown Summary, Movement Analysis.

**Collections (lists):** Companies, Groups, Ledgers, Stock Items, Stock Groups, Cost Centres, Godowns, Units, Voucher Types, Currencies, Budgets.

**Single objects:** any Ledger, Group, Stock Item, Voucher, Cost Centre, Godown, Unit, Currency, or Voucher Type by name.

**Custom XML:** anything Tally accepts via its XML gateway, via `tally raw` or the templates in `templates/`.

## CLI reference (short)

```
tally ping                                    # connectivity check
tally companies                               # list loaded companies
tally object --subtype Ledger --id "Customer ABC" --fetch Name,ClosingBalance
tally collection --id "List of Ledgers"
tally collection --id "List of Ledgers" --parent "Sundry Debtors" --fields Name,ClosingBalance
tally report --id "Day Book" --from 2026-04-01 --to 2026-04-30
tally report --id "Day Book" --from 2026-04-01 --to 2026-04-30 --voucher-type Sales
tally template --name reports/day_book --from 2026-04-01 --to 2026-04-30
tally raw --file my-request.xml
```

Global flags (any subcommand): `--scheme` (default `http`, use `https` for TLS), `--host`, `--port`, `--company`, `--timeout`, `--pretty`.

Collection filtering flags: `--parent`, `--fields`, `--filter`.
Report filtering flags: `--voucher-type`, `--filter`.

Full reference: see [skills/tally-erp/SKILL.md](./skills/tally-erp/SKILL.md).

## Repository layout

```
tally-skill/
├── .claude-plugin/
│   ├── plugin.json          # Plugin manifest
│   └── marketplace.json     # Self-hosted marketplace manifest
├── skills/
│   └── tally/
│       └── SKILL.md         # Skill instructions for Claude
├── bin/                     # Auto-added to PATH when plugin enabled
│   └── tally-windows-amd64.exe
├── cmd/tally/               # CLI entrypoint (Go)
├── internal/
│   ├── tally/               # XML envelope builders, HTTP client, status parsing
│   └── cli/                 # Subcommand implementations
├── templates/               # ~33 reusable XML request envelopes with {{PLACEHOLDERS}}
├── README.md                # Human-facing
├── go.mod
└── Makefile                 # `make build` / `make build-all` / `make test`
```

## Build from source

Requires Go 1.26+.

```bash
make test          # run all tests
make build         # local dev build (bin/tally)
make build-all     # cross-compile windows release (bin/tally-windows-amd64.exe)
make checksums     # SHA-256 of release binaries
```

No external dependencies — standard library only.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Tally returned `<STATUS>0</STATUS>` (envelope on stdout, reason on stderr) |
| 2 | Tally unreachable |
| 3 | Bad CLI args |
| 4 | HTTP timeout |
| 5 | Response not valid XML |

## Why a Go CLI instead of letting Claude POST XML directly?

- **Token efficiency** — Claude doesn't waste context constructing repetitive XML envelopes.
- **Reliability** — escaping, status parsing, and date formatting are deterministic.
- **Reusability** — the binary works without Claude too (scripts, cron, ad-hoc CLI use).

## License

MIT — see frontmatter in `SKILL.md`.

## Author

Piyush Garg
