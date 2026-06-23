package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		raw   string
		level slog.Level
		err   bool
	}{
		{"debug", slog.LevelDebug, false},
		{"INFO", slog.LevelInfo, false},
		{"warn", slog.LevelWarn, false},
		{"warning", slog.LevelWarn, false},
		{"error", slog.LevelError, false},
		{"", slog.LevelError, false},
		{"verbose", slog.LevelError, true},
	}

	for _, tt := range tests {
		level, err := ParseLevel(tt.raw)
		if tt.err {
			if err == nil {
				t.Fatalf("ParseLevel(%q) expected error", tt.raw)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseLevel(%q) error = %v", tt.raw, err)
		}
		if level != tt.level {
			t.Fatalf("ParseLevel(%q) = %v, want %v", tt.raw, level, tt.level)
		}
	}
}

func TestInitFiltersByLevel(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
	logger = slog.New(handler)
	slog.SetDefault(logger)

	logger.Info("should-not-appear")
	logger.Error("should-appear")

	out := buf.String()
	if strings.Contains(out, "should-not-appear") {
		t.Fatalf("info log leaked at error level: %q", out)
	}
	if !strings.Contains(out, "should-appear") {
		t.Fatalf("error log missing at error level: %q", out)
	}
}
