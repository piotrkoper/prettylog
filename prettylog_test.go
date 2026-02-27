package prettylog

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestNewHandler_WritesToStdout(t *testing.T) {
	h := NewHandler(nil)
	if h == nil {
		t.Fatal("NewHandler returned nil")
	}
	if h.writer == nil {
		t.Fatal("NewHandler writer is nil")
	}
	if !h.outputEmptyAttrs {
		t.Error("NewHandler should set outputEmptyAttrs to true")
	}
}

func TestNew_DefaultsToNilWriter(t *testing.T) {
	h := New(nil)
	if h == nil {
		t.Fatal("New returned nil")
	}
	if h.writer != nil {
		t.Error("New without options should have nil writer")
	}
	if h.outputEmptyAttrs {
		t.Error("New without options should not set outputEmptyAttrs")
	}
}

func TestHandle_OutputFormat(t *testing.T) {
	var buf bytes.Buffer
	h := New(nil, WithDestinationWriter(&buf))

	r := slog.NewRecord(time.Date(2024, 1, 15, 10, 30, 45, 123000000, time.UTC), slog.LevelInfo, "hello world", 0)

	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "10:30:45.123") {
		t.Errorf("expected timestamp in output, got: %s", out)
	}
	if !strings.Contains(out, "INFO") {
		t.Errorf("expected level INFO in output, got: %s", out)
	}
	if !strings.Contains(out, "hello world") {
		t.Errorf("expected message in output, got: %s", out)
	}
	if strings.Contains(out, "{") {
		t.Errorf("expected no JSON attrs in output, got: %s", out)
	}
}

func TestHandle_WithAttrsInOutput(t *testing.T) {
	var buf bytes.Buffer
	h := New(nil, WithDestinationWriter(&buf))

	r := slog.NewRecord(time.Now(), slog.LevelDebug, "test msg", 0)
	r.AddAttrs(slog.String("key", "value"), slog.Int("count", 42))

	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"key":"value"`) {
		t.Errorf("expected key/value attr in output, got: %s", out)
	}
	if !strings.Contains(out, `"count":42`) {
		t.Errorf("expected count attr in output, got: %s", out)
	}
}

func TestHandle_OutputEmptyAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := New(nil, WithDestinationWriter(&buf), WithOutputEmptyAttrs())

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)

	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "{}") {
		t.Errorf("expected empty JSON object in output when outputEmptyAttrs=true, got: %s", out)
	}
}

func TestHandle_NoEmptyAttrsByDefault(t *testing.T) {
	var buf bytes.Buffer
	h := New(nil, WithDestinationWriter(&buf))

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)

	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "{}") {
		t.Errorf("expected no empty JSON object when outputEmptyAttrs=false, got: %s", out)
	}
}

func TestEnabled(t *testing.T) {
	h := New(&slog.HandlerOptions{Level: slog.LevelWarn}, WithDestinationWriter(&bytes.Buffer{}))

	if h.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("expected Debug to be disabled when level is Warn")
	}
	if h.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("expected Info to be disabled when level is Warn")
	}
	if !h.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("expected Warn to be enabled when level is Warn")
	}
	if !h.Enabled(context.Background(), slog.LevelError) {
		t.Error("expected Error to be enabled when level is Warn")
	}
}

func TestWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := New(nil, WithDestinationWriter(&buf))

	h2 := h.WithAttrs([]slog.Attr{slog.String("persistent", "yes")})

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	if err := h2.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"persistent":"yes"`) {
		t.Errorf("expected persistent attr in output from WithAttrs, got: %s", out)
	}
}

func TestWithGroup(t *testing.T) {
	var buf bytes.Buffer
	h := New(nil, WithDestinationWriter(&buf))

	h2 := h.WithGroup("mygroup")

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.String("k", "v"))
	if err := h2.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "mygroup") {
		t.Errorf("expected group name in output from WithGroup, got: %s", out)
	}
}

func TestHandle_ReplaceAttr_SuppressLevel(t *testing.T) {
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				return slog.Attr{}
			}
			return a
		},
	}
	h := New(opts, WithDestinationWriter(&buf))

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "INFO") {
		t.Errorf("expected level to be suppressed, got: %s", out)
	}
}

func TestHandle_LevelFormatting(t *testing.T) {
	levels := []struct {
		level    slog.Level
		expected string
	}{
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelInfo, "INFO"},
		{slog.LevelWarn, "WARN"},
		{slog.LevelError, "ERROR"},
	}

	for _, tc := range levels {
		tc := tc

		t.Run(tc.expected, func(t *testing.T) {
			var buf bytes.Buffer
			h := New(&slog.HandlerOptions{Level: slog.LevelDebug}, WithDestinationWriter(&buf))

			r := slog.NewRecord(time.Now(), tc.level, "msg", 0)
			if err := h.Handle(context.Background(), r); err != nil {
				t.Fatalf("Handle returned error for level %v: %v", tc.level, err)
			}

			out := buf.String()
			if !strings.Contains(out, tc.expected) {
				t.Errorf("expected level %s in output, got: %s", tc.expected, out)
			}
		})
	}
}
