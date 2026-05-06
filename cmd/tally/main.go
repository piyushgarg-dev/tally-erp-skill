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
	case "template":
		os.Exit(cli.RunTemplate(args))
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
  template            Substitute placeholders in a templates/*.xml file and POST it
  version             Print version

Global flags (any subcommand):
  --host string       Tally host (default "localhost")
  --port int          Tally port (default 9000)
  --company string    Sets SVCURRENTCOMPANY
  --timeout duration  HTTP timeout (default 30s)
  --pretty            Pretty-print response XML`)
}
