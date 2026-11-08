package logsink

import (
	"testing"
)

func TestSnapshotAndSinceDoNotStallAfterWrap(t *testing.T) {
	r := NewRingBuffer(4)
	for i := 1; i <= 6; i++ {
		r.Add(LogEntry{Msg: "entry"})
	}

	replay, cursor := r.Snapshot(2)
	if len(replay) != 2 {
		t.Fatalf("Snapshot(2) returned %d entries, want 2", len(replay))
	}
	if cursor != 6 {
		t.Fatalf("cursor = %d, want 6", cursor)
	}

	if got, _ := r.Since(cursor); len(got) != 0 {
		t.Fatalf("Since(cursor) returned %d entries, want 0", len(got))
	}

	// The ring is full; a new entry must still be observed. The old
	// Len()-based loop stalled here because Len() never exceeds capacity.
	r.Add(LogEntry{Msg: "after-wrap"})
	fresh, next := r.Since(cursor)
	if len(fresh) != 1 || fresh[0].Msg != "after-wrap" {
		t.Fatalf("Since after wrap = %#v, want the new entry", fresh)
	}
	if next != 7 {
		t.Fatalf("next cursor = %d, want 7", next)
	}
}

func TestSnapshotAndSinceNoDuplicates(t *testing.T) {
	r := NewRingBuffer(8)
	for i := 0; i < 3; i++ {
		r.Add(LogEntry{Msg: "entry"})
	}
	replay, cursor := r.Snapshot(10)
	if len(replay) != 3 {
		t.Fatalf("replay = %d, want 3", len(replay))
	}
	// Everything in the replay is already covered by the cursor.
	if got, _ := r.Since(cursor); len(got) != 0 {
		t.Fatalf("Since after replay returned %d duplicates", len(got))
	}
}
