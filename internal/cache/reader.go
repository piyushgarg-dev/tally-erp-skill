package cache

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// QueryOpts specifies filters for reading from the cache.
type QueryOpts struct {
	Company  string
	Type     string
	Party    string // glob pattern
	Stock    string // glob pattern
	From     string // dd-mm-yyyy
	To       string // dd-mm-yyyy
	CacheDir string
}

// QueryResult holds filtered rows and summary info.
type QueryResult struct {
	Rows []Row
}

// Query reads from the file tree and returns filtered rows.
func Query(opts QueryOpts) (QueryResult, error) {
	dir := filepath.Join(opts.CacheDir, SanitizeFilename(opts.Company), opts.Type)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return QueryResult{}, fmt.Errorf("no cache found at %s (run tally sync first)", dir)
	}

	var rows []Row
	var err error

	switch {
	case opts.Party != "" && !strings.Contains(opts.Party, "*"):
		// Exact party match — try direct file
		fname := SanitizeFilename(opts.Party) + ".csv"
		path := filepath.Join(dir, "by-party", fname)
		rows, err = readCSVFile(path)
		if err != nil {
			// Fall back to scanning by-party directory
			rows, err = scanDir(filepath.Join(dir, "by-party"), opts.Party)
		}
	case opts.Party != "":
		// Glob party match
		rows, err = scanDir(filepath.Join(dir, "by-party"), opts.Party)
	case opts.Stock != "" && !strings.Contains(opts.Stock, "*"):
		fname := SanitizeFilename(opts.Stock) + ".csv"
		path := filepath.Join(dir, "by-stock", fname)
		rows, err = readCSVFile(path)
		if err != nil {
			rows, err = scanDir(filepath.Join(dir, "by-stock"), opts.Stock)
		}
	case opts.Stock != "":
		rows, err = scanDir(filepath.Join(dir, "by-stock"), opts.Stock)
	case opts.From != "" || opts.To != "":
		rows, err = readMonthRange(filepath.Join(dir, "by-month"), opts.From, opts.To)
	default:
		rows, err = readCSVFile(filepath.Join(dir, "all.csv"))
	}

	if err != nil {
		return QueryResult{}, err
	}

	// Apply additional filters in-memory
	rows = applyFilters(rows, opts)
	return QueryResult{Rows: rows}, nil
}

func applyFilters(rows []Row, opts QueryOpts) []Row {
	var out []Row
	for _, r := range rows {
		if opts.Party != "" && !MatchGlob(opts.Party, r.Party) {
			continue
		}
		if opts.Stock != "" && !MatchGlob(opts.Stock, r.Stock) {
			continue
		}
		if opts.From != "" && r.Date < opts.From {
			continue
		}
		if opts.To != "" && r.Date > opts.To {
			continue
		}
		out = append(out, r)
	}
	return out
}

func scanDir(dir, pattern string) ([]Row, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}
	var all []Row
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".csv") {
			continue
		}
		// Match against the original names by reading the CSV content
		path := filepath.Join(dir, e.Name())
		rows, err := readCSVFile(path)
		if err != nil {
			continue
		}
		all = append(all, rows...)
	}
	return all, nil
}

func readMonthRange(dir, from, to string) ([]Row, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var all []Row
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".csv") {
			continue
		}
		month := strings.TrimSuffix(e.Name(), ".csv") // e.g. "2025-04"
		// Simple range check on month strings (they sort lexicographically)
		fromMonth := ""
		toMonth := ""
		if from != "" && len(from) >= 7 {
			fromMonth = from[6:10] + "-" + from[3:5]
		}
		if to != "" && len(to) >= 7 {
			toMonth = to[6:10] + "-" + to[3:5]
		}
		if fromMonth != "" && month < fromMonth {
			continue
		}
		if toMonth != "" && month > toMonth {
			continue
		}
		rows, err := readCSVFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		all = append(all, rows...)
	}
	return all, nil
}

func readCSVFile(path string) ([]Row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	// Skip header
	if _, err := r.Read(); err != nil {
		return nil, err
	}

	var rows []Row
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if len(record) < 8 {
			continue
		}
		qty, _ := strconv.ParseFloat(strings.TrimSpace(record[6]), 64)
		amt, _ := strconv.ParseFloat(strings.TrimSpace(record[7]), 64)
		rows = append(rows, Row{
			Date:     record[0],
			Invoice:  record[1],
			Party:    record[2],
			Stock:    record[3],
			Rate:     record[4],
			Discount: record[5],
			Qty:      qty,
			Amount:   amt,
		})
	}
	return rows, nil
}
