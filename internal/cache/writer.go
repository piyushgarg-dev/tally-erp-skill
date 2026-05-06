package cache

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Row represents a single sales/purchase line item.
type Row struct {
	Date     string
	Invoice  string
	Party    string
	Stock    string
	Rate     string
	Discount string
	Qty      float64
	Amount   float64
}

var csvHeader = []string{"Date", "Invoice", "Party", "StockItem", "Rate", "Discount%", "Qty", "Amount"}

// WriteCache writes all rows to a partitioned file tree under baseDir.
// Structure: baseDir/<company>/<type>/all.csv + by-party/ + by-stock/ + by-month/ + index.json
func WriteCache(baseDir, company, dataType, from, to string, rows []Row) error {
	dir := filepath.Join(baseDir, SanitizeFilename(company), dataType)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write all.csv
	if err := writeCSV(filepath.Join(dir, "all.csv"), rows); err != nil {
		return fmt.Errorf("writing all.csv: %w", err)
	}

	// Partition by party
	byParty := partition(rows, func(r Row) string { return r.Party })
	partyDir := filepath.Join(dir, "by-party")
	if err := os.MkdirAll(partyDir, 0755); err != nil {
		return err
	}
	partyNames := make([]string, 0, len(byParty))
	for name, prows := range byParty {
		partyNames = append(partyNames, name)
		fname := SanitizeFilename(name) + ".csv"
		if err := writeCSV(filepath.Join(partyDir, fname), prows); err != nil {
			return fmt.Errorf("writing party %s: %w", name, err)
		}
	}
	sort.Strings(partyNames)

	// Partition by stock item
	byStock := partition(rows, func(r Row) string { return r.Stock })
	stockDir := filepath.Join(dir, "by-stock")
	if err := os.MkdirAll(stockDir, 0755); err != nil {
		return err
	}
	stockNames := make([]string, 0, len(byStock))
	for name, srows := range byStock {
		stockNames = append(stockNames, name)
		fname := SanitizeFilename(name) + ".csv"
		if err := writeCSV(filepath.Join(stockDir, fname), srows); err != nil {
			return fmt.Errorf("writing stock %s: %w", name, err)
		}
	}
	sort.Strings(stockNames)

	// Partition by month (YYYY-MM from Date field dd-mm-yyyy)
	byMonth := partition(rows, func(r Row) string {
		if len(r.Date) >= 10 {
			return r.Date[6:10] + "-" + r.Date[3:5]
		}
		return "unknown"
	})
	monthDir := filepath.Join(dir, "by-month")
	if err := os.MkdirAll(monthDir, 0755); err != nil {
		return err
	}
	monthNames := make([]string, 0, len(byMonth))
	for name, mrows := range byMonth {
		monthNames = append(monthNames, name)
		if err := writeCSV(filepath.Join(monthDir, name+".csv"), mrows); err != nil {
			return fmt.Errorf("writing month %s: %w", name, err)
		}
	}
	sort.Strings(monthNames)

	// Write index
	idx := Index{
		Company:    company,
		Type:       dataType,
		From:       from,
		To:         to,
		TotalRows:  len(rows),
		Parties:    partyNames,
		StockItems: stockNames,
		Months:     monthNames,
	}
	return WriteIndex(dir, idx)
}

func writeCSV(path string, rows []Row) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Write(csvHeader)
	for _, r := range rows {
		w.Write([]string{
			r.Date,
			r.Invoice,
			r.Party,
			r.Stock,
			r.Rate,
			r.Discount,
			fmt.Sprintf("%.1f", r.Qty),
			fmt.Sprintf("%.2f", r.Amount),
		})
	}
	w.Flush()
	return w.Error()
}

func partition(rows []Row, keyFn func(Row) string) map[string][]Row {
	m := map[string][]Row{}
	for _, r := range rows {
		k := keyFn(r)
		m[k] = append(m[k], r)
	}
	return m
}

// ParseMonth extracts YYYY-MM from a date string in dd-mm-yyyy format.
func ParseMonth(date string) string {
	if len(date) >= 10 {
		return date[6:10] + "-" + date[3:5]
	}
	return ""
}

// MatchGlob performs case-insensitive glob matching on a string.
func MatchGlob(pattern, s string) bool {
	p := strings.ToLower(pattern)
	v := strings.ToLower(s)
	if !strings.Contains(p, "*") {
		return strings.Contains(v, p)
	}
	parts := strings.Split(p, "*")
	idx := 0
	for _, part := range parts {
		if part == "" {
			continue
		}
		pos := strings.Index(v[idx:], part)
		if pos < 0 {
			return false
		}
		idx += pos + len(part)
	}
	return true
}
