package tally

import (
	"fmt"
	"time"
)

// ChunkDates splits a date range [from, to] (both YYYYMMDD) into sub-ranges
// based on the given granularity: "daily", "weekly", or "monthly".
func ChunkDates(from, to, granularity string) ([][2]string, error) {
	start, err := parseTallyDate(from)
	if err != nil {
		return nil, fmt.Errorf("from date: %w", err)
	}
	end, err := parseTallyDate(to)
	if err != nil {
		return nil, fmt.Errorf("to date: %w", err)
	}
	if end.Before(start) {
		return nil, fmt.Errorf("to date %s is before from date %s", to, from)
	}

	var chunks [][2]string
	cur := start

	for !cur.After(end) {
		var chunkEnd time.Time
		switch granularity {
		case "daily":
			chunkEnd = cur
		case "weekly":
			chunkEnd = cur.AddDate(0, 0, 6)
		case "monthly":
			chunkEnd = endOfMonth(cur)
		default:
			return nil, fmt.Errorf("unknown granularity %q (use daily, weekly, or monthly)", granularity)
		}

		if chunkEnd.After(end) {
			chunkEnd = end
		}
		chunks = append(chunks, [2]string{
			cur.Format("20060102"),
			chunkEnd.Format("20060102"),
		})
		cur = chunkEnd.AddDate(0, 0, 1)
	}
	return chunks, nil
}

func parseTallyDate(s string) (time.Time, error) {
	t, err := time.Parse("20060102", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected YYYYMMDD, got %q", s)
	}
	return t, nil
}

func endOfMonth(t time.Time) time.Time {
	y, m, _ := t.Date()
	return time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC)
}
