package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Index holds metadata about a synced data cache.
type Index struct {
	Company    string   `json:"company"`
	Type       string   `json:"type"`
	From       string   `json:"from"`
	To         string   `json:"to"`
	SyncedAt   string   `json:"synced_at"`
	TotalRows  int      `json:"total_rows"`
	Parties    []string `json:"parties"`
	StockItems []string `json:"stock_items"`
	Months     []string `json:"months"`
}

// WriteIndex writes index.json to the given directory.
func WriteIndex(dir string, idx Index) error {
	if idx.SyncedAt == "" {
		idx.SyncedAt = time.Now().Format(time.RFC3339)
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "index.json"), data, 0644)
}

// ReadIndex reads index.json from the given directory.
func ReadIndex(dir string) (Index, error) {
	data, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		return Index{}, err
	}
	var idx Index
	err = json.Unmarshal(data, &idx)
	return idx, err
}
