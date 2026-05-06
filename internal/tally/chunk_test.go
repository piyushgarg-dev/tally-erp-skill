package tally

import (
	"testing"
)

func TestChunkDatesMonthly(t *testing.T) {
	chunks, err := ChunkDates("20250401", "20250630", "monthly")
	if err != nil {
		t.Fatal(err)
	}
	want := [][2]string{
		{"20250401", "20250430"},
		{"20250501", "20250531"},
		{"20250601", "20250630"},
	}
	if len(chunks) != len(want) {
		t.Fatalf("got %d chunks, want %d", len(chunks), len(want))
	}
	for i, ch := range chunks {
		if ch != want[i] {
			t.Errorf("chunk %d: got %v, want %v", i, ch, want[i])
		}
	}
}

func TestChunkDatesMonthlyFullYear(t *testing.T) {
	chunks, err := ChunkDates("20250401", "20260331", "monthly")
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 12 {
		t.Fatalf("got %d chunks, want 12", len(chunks))
	}
	if chunks[0] != [2]string{"20250401", "20250430"} {
		t.Errorf("first chunk: got %v", chunks[0])
	}
	if chunks[11] != [2]string{"20260301", "20260331"} {
		t.Errorf("last chunk: got %v", chunks[11])
	}
}

func TestChunkDatesWeekly(t *testing.T) {
	chunks, err := ChunkDates("20250401", "20250420", "weekly")
	if err != nil {
		t.Fatal(err)
	}
	want := [][2]string{
		{"20250401", "20250407"},
		{"20250408", "20250414"},
		{"20250415", "20250420"},
	}
	if len(chunks) != len(want) {
		t.Fatalf("got %d chunks, want %d", len(chunks), len(want))
	}
	for i, ch := range chunks {
		if ch != want[i] {
			t.Errorf("chunk %d: got %v, want %v", i, ch, want[i])
		}
	}
}

func TestChunkDatesDaily(t *testing.T) {
	chunks, err := ChunkDates("20250401", "20250403", "daily")
	if err != nil {
		t.Fatal(err)
	}
	want := [][2]string{
		{"20250401", "20250401"},
		{"20250402", "20250402"},
		{"20250403", "20250403"},
	}
	if len(chunks) != len(want) {
		t.Fatalf("got %d chunks, want %d", len(chunks), len(want))
	}
	for i, ch := range chunks {
		if ch != want[i] {
			t.Errorf("chunk %d: got %v, want %v", i, ch, want[i])
		}
	}
}

func TestChunkDatesSingleDay(t *testing.T) {
	chunks, err := ChunkDates("20250415", "20250415", "monthly")
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("got %d chunks, want 1", len(chunks))
	}
	if chunks[0] != [2]string{"20250415", "20250415"} {
		t.Errorf("got %v", chunks[0])
	}
}

func TestChunkDatesInvalidGranularity(t *testing.T) {
	_, err := ChunkDates("20250401", "20250430", "yearly")
	if err == nil {
		t.Fatal("expected error for invalid granularity")
	}
}

func TestChunkDatesReversedDates(t *testing.T) {
	_, err := ChunkDates("20250430", "20250401", "monthly")
	if err == nil {
		t.Fatal("expected error for reversed dates")
	}
}

func TestChunkDatesFebruary(t *testing.T) {
	chunks, err := ChunkDates("20260201", "20260228", "monthly")
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("got %d chunks, want 1", len(chunks))
	}
	if chunks[0] != [2]string{"20260201", "20260228"} {
		t.Errorf("got %v", chunks[0])
	}
}

func TestChunkDatesLeapYear(t *testing.T) {
	chunks, err := ChunkDates("20240201", "20240229", "monthly")
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("got %d chunks, want 1", len(chunks))
	}
	if chunks[0] != [2]string{"20240201", "20240229"} {
		t.Errorf("got %v", chunks[0])
	}
}
