package logs_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/asaidimu/hestia/core/system/logs"
)

func TestRingBufferAddAndRecent(t *testing.T) {
	ring := logs.NewRingBuffer(5)

	for i := 0; i < 3; i++ {
		ring.Add(logs.LogEntry{Level: "info", Msg: "test"})
	}

	if ring.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", ring.Len())
	}

	recent := ring.Recent(2)
	if len(recent) != 2 {
		t.Fatalf("Recent(2) returned %d entries, want 2", len(recent))
	}
	if recent[0].Msg != "test" {
		t.Errorf("entry Msg = %q, want %q", recent[0].Msg, "test")
	}
}

func TestRingBufferOverflow(t *testing.T) {
	ring := logs.NewRingBuffer(3)

	for i := 0; i < 5; i++ {
		ring.Add(logs.LogEntry{Level: "info", Msg: string(rune('A' + i))})
	}

	if ring.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", ring.Len())
	}

	recent := ring.Recent(0)
	if len(recent) != 3 {
		t.Fatalf("Recent(0) returned %d entries, want 3", len(recent))
	}
	// After overflow, should have C, D, E
	if recent[0].Msg != "C" {
		t.Errorf("first entry Msg = %q, want %q", recent[0].Msg, "C")
	}
	if recent[2].Msg != "E" {
		t.Errorf("last entry Msg = %q, want %q", recent[2].Msg, "E")
	}
}

func TestRingBufferConcurrent(t *testing.T) {
	ring := logs.NewRingBuffer(100)
	done := make(chan struct{})

	go func() {
		for i := 0; i < 200; i++ {
			ring.Add(logs.LogEntry{Level: "info", Msg: "concurrent"})
		}
		close(done)
	}()

	for i := 0; i < 100; i++ {
		_ = ring.Recent(10)
	}

	<-done
	if ring.Len() != 100 {
		t.Errorf("Len() = %d, want 100", ring.Len())
	}
}

func TestReaderQuery(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")

	entries := `{"level":"info","ts":1787134069.78,"caller":"a.go:1","msg":"hello"}
{"level":"warn","ts":1787134070.00,"caller":"b.go:2","msg":"warning here"}
{"level":"error","ts":1787134071.00,"caller":"c.go:3","msg":"something broke"}
{"level":"info","ts":1787134072.00,"caller":"d.go:4","msg":"back to info"}
`
	if err := os.WriteFile(logFile, []byte(entries), 0644); err != nil {
		t.Fatal(err)
	}

	reader := logs.NewReader(logFile)

	// Query all
	result, err := reader.Query(logs.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 4 {
		t.Errorf("Total = %d, want 4", result.Total)
	}

	// Query by level
	result, err = reader.Query(logs.Query{Level: "warn"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 {
		t.Errorf("warn Total = %d, want 1", result.Total)
	}
	if result.Entries[0].Msg != "warning here" {
		t.Errorf("warn entry Msg = %q, want %q", result.Entries[0].Msg, "warning here")
	}

	// Query by search
	result, err = reader.Query(logs.Query{Search: "broke"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 {
		t.Errorf("search Total = %d, want 1", result.Total)
	}

	// Query with limit
	result, err = reader.Query(logs.Query{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 2 {
		t.Errorf("limited entries = %d, want 2", len(result.Entries))
	}
	if !result.HasMore {
		t.Error("HasMore should be true")
	}

	// Query with offset
	result, err = reader.Query(logs.Query{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 2 {
		t.Errorf("offset entries = %d, want 2", len(result.Entries))
	}
	if result.Total != 4 {
		t.Errorf("offset Total = %d, want 4", result.Total)
	}
}

func TestReaderQueryByTime(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")

	entries := `{"level":"info","ts":1787134069.78,"caller":"a.go:1","msg":"early"}
{"level":"info","ts":1787134100.00,"caller":"b.go:2","msg":"middle"}
{"level":"info","ts":1787134200.00,"caller":"c.go:3","msg":"late"}
`
	if err := os.WriteFile(logFile, []byte(entries), 0644); err != nil {
		t.Fatal(err)
	}

	reader := logs.NewReader(logFile)

	from := time.Unix(1787134070, 0)
	to := time.Unix(1787134150, 0)
	result, err := reader.Query(logs.Query{From: from, To: to})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 {
		t.Errorf("time range Total = %d, want 1", result.Total)
	}
	if result.Entries[0].Msg != "middle" {
		t.Errorf("time range entry Msg = %q, want %q", result.Entries[0].Msg, "middle")
	}
}

func TestReaderTail(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")

	var content string
	for i := 0; i < 10; i++ {
		content += `{"level":"info","ts":1787134069.78,"caller":"a.go:1","msg":"line` + string(rune('0'+i)) + `"}` + "\n"
	}
	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	reader := logs.NewReader(logFile)
	entries, err := reader.Tail(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("Tail(3) returned %d entries, want 3", len(entries))
	}
	if entries[0].Msg != "line7" {
		t.Errorf("first tail entry Msg = %q, want %q", entries[0].Msg, "line7")
	}
	if entries[2].Msg != "line9" {
		t.Errorf("last tail entry Msg = %q, want %q", entries[2].Msg, "line9")
	}
}

func TestReaderQueryMalformedLines(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")

	entries := `{"level":"info","ts":1787134069.78,"caller":"a.go:1","msg":"good"}
not json at all
{"level":"warn","ts":1787134070.00,"caller":"b.go:2","msg":"also good"}
`
	if err := os.WriteFile(logFile, []byte(entries), 0644); err != nil {
		t.Fatal(err)
	}

	reader := logs.NewReader(logFile)
	result, err := reader.Query(logs.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 {
		t.Errorf("Total = %d, want 2 (malformed lines skipped)", result.Total)
	}
}

func TestPolicies(t *testing.T) {
	bindings := logs.Policies()
	if len(bindings) != 2 {
		t.Fatalf("Policies() returned %d bindings, want 2", len(bindings))
	}

	expected := map[string]string{
		"system:logs:list":   "administrator",
		"system:logs:stream": "administrator",
	}

	for _, b := range bindings {
		want, ok := expected[b.Name]
		if !ok {
			t.Errorf("unexpected binding %q", b.Name)
			continue
		}
		if b.RuleKey != want {
			t.Errorf("%s ruleKey = %q, want %q", b.Name, b.RuleKey, want)
		}
	}
}
