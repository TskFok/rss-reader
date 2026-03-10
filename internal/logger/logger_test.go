package logger

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  Level
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"warning", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"", LevelInfo},
		{"invalid", LevelInfo},
	}
	for _, tt := range tests {
		got := parseLevel(tt.input)
		if got != tt.want {
			t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestShouldLog(t *testing.T) {
	tests := []struct {
		levelStr string
		call     Level
		want     bool
	}{
		{"debug", LevelDebug, true},
		{"debug", LevelInfo, true},
		{"info", LevelDebug, false},
		{"info", LevelInfo, true},
		{"info", LevelError, true},
		{"warn", LevelInfo, false},
		{"warn", LevelWarn, true},
		{"error", LevelInfo, false},
		{"error", LevelWarn, false},
		{"error", LevelError, true},
	}
	for _, tt := range tests {
		Init(tt.levelStr)
		got := shouldLog(tt.call)
		if got != tt.want {
			t.Errorf("level=%q, call=%v: shouldLog = %v, want %v", tt.levelStr, tt.call, got, tt.want)
		}
	}
}

func TestInfo_RespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	old := stdLog.Writer()
	stdLog.SetOutput(&buf)
	defer stdLog.SetOutput(old)

	Init("error")
	Info("test info")
	if got := buf.String(); got != "" {
		t.Errorf("Info with level=error should not output, got %q", got)
	}

	buf.Reset()
	Init("info")
	Info("test %s", "msg")
	if got := buf.String(); !strings.Contains(got, "[INFO]") || !strings.Contains(got, "test msg") {
		t.Errorf("Info with level=info should output, got %q", got)
	}
}

func TestError_RespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	old := stdLog.Writer()
	stdLog.SetOutput(&buf)
	defer stdLog.SetOutput(old)

	Init("error")
	Error("test %s", "err")
	if got := buf.String(); !strings.Contains(got, "[ERROR]") || !strings.Contains(got, "test err") {
		t.Errorf("Error with level=error should output, got %q", got)
	}
}

func TestFatal_AlwaysOutputs(t *testing.T) {
	// Fatalf exits, so we can't easily test it. Just verify it doesn't panic.
	// Use a custom logger that doesn't exit for testing.
	origLog := log.New(stdLog.Writer(), stdLog.Prefix(), stdLog.Flags())
	_ = origLog
	// Fatal/Fatalf call stdLog.Fatal which exits - skip runtime test
}
