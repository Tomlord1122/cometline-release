package sse

import (
	"strings"
	"testing"
)

func TestScanner_AllowsLargeEventLines(t *testing.T) {
	large := strings.Repeat("x", 128*1024)
	s := NewScanner(strings.NewReader("event: response.created\ndata: " + large + "\n\n"))

	if !s.Next() {
		t.Fatalf("expected large event, err=%v", s.Err())
	}

	got := s.Event()
	if got.Type != "response.created" {
		t.Fatalf("type = %q, want response.created", got.Type)
	}
	if got.Data != large {
		t.Fatalf("data length = %d, want %d", len(got.Data), len(large))
	}
	if s.Next() {
		t.Fatalf("unexpected second event")
	}
	if err := s.Err(); err != nil {
		t.Fatalf("unexpected scanner err: %v", err)
	}
}
