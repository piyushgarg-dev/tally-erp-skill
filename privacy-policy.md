# Privacy Policy

_Last updated: 2026-05-06_

This privacy policy describes how the **Tally ERP** Claude Code plugin (the "Plugin") handles your data.

## Summary

The Plugin is a thin, read-only client for a TallyPrime instance you run yourself. It does not collect, store, or transmit your data to the plugin author or any third party. All communication happens directly between your machine (where Claude Code runs) and your TallyPrime gateway.

## What the Plugin Does

The Plugin ships a command-line binary (`tally`) and XML request templates. When invoked by Claude Code, the binary:

1. Constructs an XML request based on the subcommand and flags provided.
2. Sends that request over HTTP to the TallyPrime gateway address you configure (default `http://localhost:9000`).
3. Returns the XML response to Claude Code on stdout.

No telemetry, analytics, or background network calls are made.

## Data the Plugin Accesses

When you run a command, the Plugin reads accounting data from your TallyPrime instance, which may include:

- Company names and metadata
- Ledger, group, stock item, and voucher records
- Standard reports (Day Book, Trial Balance, P&L, Balance Sheet, etc.)
- Party details (names, addresses, GSTINs, contact info) stored in your ledgers

This data is returned to Claude Code so the model can answer your questions. The Plugin itself does not persist it.

## Where Data Goes

- **TallyPrime gateway** — your local or networked TallyPrime instance, controlled entirely by you.
- **Claude Code** — the response is passed to the Claude model that invoked the command, subject to [Anthropic's Privacy Policy](https://www.anthropic.com/legal/privacy) and your Claude Code data handling settings.
- **Nowhere else** — the Plugin author does not receive any of your data.

## Write Access

The Plugin is **read-only by design**. It does not create, modify, or delete vouchers, masters, or any other records in TallyPrime.

## Network Exposure

The Plugin only contacts the host and port you specify via `--host` / `--port` (default `localhost:9000`). Ensure your TallyPrime gateway is bound to interfaces and networks you trust — exposing it to the public internet is not recommended.

## Logs

The Plugin writes responses to stdout and error messages to stderr. It does not write log files. Anything you see in your terminal or Claude Code transcript is governed by your own environment and Claude Code's data settings.

## Your Responsibilities

- You control which company data is loaded in TallyPrime and reachable by the gateway.
- You control which Claude Code session has access to run the Plugin.
- You are responsible for complying with applicable data protection and accounting record laws (e.g. GDPR, India's DPDP Act) regarding the data you query and share with the model.

## Changes

If this policy changes, the updated version will replace this file in the plugin repository. The "Last updated" date at the top reflects the most recent revision.

## Contact

For questions about this Plugin, open an issue in the plugin's source repository.
