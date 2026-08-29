package logs

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"time"
)

// Reader reads log entries from an NDJSON file.
type Reader struct {
	path string
}

// NewReader creates a log reader for the given file path.
func NewReader(path string) *Reader {
	return &Reader{path: path}
}

// Query holds filter criteria for log queries.
type Query struct {
	Level   string    // filter by level (info, warn, error, etc.); empty = all
	From    time.Time // start time (zero = no lower bound)
	To      time.Time // end time (zero = no upper bound)
	Search  string    // substring match on message
	Limit   int       // max entries to return; 0 = default 500
	Offset  int       // skip first N matching entries (for pagination)
}

// Result holds a page of log entries.
type Result struct {
	Entries   []LogEntry `json:"entries"`
	Total     int        `json:"total"`
	HasMore   bool       `json:"has_more"`
	QueryPath string     `json:"query_path"`
}

// Query scans the log file and returns entries matching the filter.
func (r *Reader) Query(q Query) (*Result, error) {
	f, err := os.Open(r.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	limit := q.Limit
	if limit <= 0 {
		limit = 500
	}

	var entries []LogEntry
	total := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // skip malformed lines
		}

		if !matchEntry(entry, q) {
			continue
		}

		total++
		if total-1 < q.Offset {
			continue
		}
		if len(entries) < limit {
			entries = append(entries, entry)
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return nil, err
	}

	return &Result{
		Entries:   entries,
		Total:     total,
		HasMore:   total > q.Offset+len(entries),
		QueryPath: r.path,
	}, nil
}

// Tail reads the last n entries from the log file.
func (r *Reader) Tail(n int) ([]LogEntry, error) {
	if n <= 0 {
		n = 100
	}

	f, err := os.Open(r.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read lines into a ring buffer of size n
	buf := make([]LogEntry, 0, n)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry LogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if len(buf) < n {
			buf = append(buf, entry)
		} else {
			copy(buf, buf[1:])
			buf[len(buf)-1] = entry
		}
	}

	return buf, scanner.Err()
}

func matchEntry(entry LogEntry, q Query) bool {
	if q.Level != "" && entry.Level != q.Level {
		return false
	}
	t := entry.Time()
	if !q.From.IsZero() && t.Before(q.From) {
		return false
	}
	if !q.To.IsZero() && t.After(q.To) {
		return false
	}
	if q.Search != "" {
		if containsIgnoreCase(entry.Msg, q.Search) {
			return true
		}
		// Also search inside structured fields and extra top-level keys
		for _, m := range []map[string]any{entry.Fields, entry.Extra} {
			for k, v := range m {
				if containsIgnoreCase(k, q.Search) {
					return true
				}
				if s, ok := v.(string); ok && containsIgnoreCase(s, q.Search) {
					return true
				}
			}
		}
		return false
	}
	return true
}

func containsIgnoreCase(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalIgnoreCase(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
