package prettylog

// https://github.com/dusted-go/logging/blob/main/prettylog/prettylog.go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

const (
	timeFormat = "[15:04:05.000]"
)

type Handler struct {
	h                slog.Handler
	r                func([]string, slog.Attr) slog.Attr
	b                *bytes.Buffer
	m                *sync.Mutex
	writer           io.Writer
	outputEmptyAttrs bool
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.h.Enabled(ctx, level)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{h: h.h.WithAttrs(attrs), b: h.b, r: h.r, m: h.m, writer: h.writer, outputEmptyAttrs: h.outputEmptyAttrs}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{h: h.h.WithGroup(name), b: h.b, r: h.r, m: h.m, writer: h.writer, outputEmptyAttrs: h.outputEmptyAttrs}
}

func (h *Handler) computeAttrs(
	ctx context.Context,
	r slog.Record,
) (map[string]any, error) {
	h.m.Lock()
	defer func() {
		h.b.Reset()
		h.m.Unlock()
	}()
	if err := h.h.Handle(ctx, r); err != nil {
		return nil, fmt.Errorf("error when calling inner handler's Handle: %w", err)
	}

	var attrs map[string]any
	err := json.Unmarshal(h.b.Bytes(), &attrs)
	if err != nil {
		return nil, fmt.Errorf("error when unmarshaling inner handler's Handle result: %w", err)
	}
	return attrs, nil
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {

	var level string
	levelAttr := slog.Attr{
		Key:   slog.LevelKey,
		Value: slog.AnyValue(r.Level),
	}
	if h.r != nil {
		levelAttr = h.r([]string{}, levelAttr)
	}

	if !levelAttr.Equal(slog.Attr{}) {
		level = levelAttr.Value.String() + ":"
	}

	var timestamp string
	timeAttr := slog.Attr{
		Key:   slog.TimeKey,
		Value: slog.StringValue(r.Time.Format(timeFormat)),
	}

	if h.r != nil {
		timeAttr = h.r([]string{}, timeAttr)
	}

	if !timeAttr.Equal(slog.Attr{}) {
		timestamp = timeAttr.Value.String()
	}

	var msg string
	msgAttr := slog.Attr{
		Key:   slog.MessageKey,
		Value: slog.StringValue(r.Message),
	}

	if h.r != nil {
		msgAttr = h.r([]string{}, msgAttr)
	}

	if !msgAttr.Equal(slog.Attr{}) {
		msg = msgAttr.Value.String()
	}

	attrs, err := h.computeAttrs(ctx, r)
	if err != nil {
		return err
	}

	var attrsAsBytes []byte
	if h.outputEmptyAttrs || len(attrs) > 0 {
		attrsAsBytes, err = json.Marshal(attrs)
		if err != nil {
			return fmt.Errorf("error when marshaling attrs: %w", err)
		}
	}

	out := strings.Builder{}
	if len(timestamp) > 0 {
		out.WriteString(timestamp)
		out.WriteString(" ")
	}
	if len(level) > 0 {
		// padd left level to 7 characters for better alignment
		fmt.Fprintf(&out, "%7s", level)
		out.WriteString(" ")
	}

	if len(msg) > 0 {
		out.WriteString(msg)
		out.WriteString(" ")
	}

	if len(attrsAsBytes) > 0 {
		out.WriteString(string(attrsAsBytes))
	}

	_, err = io.WriteString(h.writer, out.String()+"\n")
	if err != nil {
		return err
	}

	return nil
}

func suppressDefaults(
	next func([]string, slog.Attr) slog.Attr,
) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey ||
			a.Key == slog.LevelKey ||
			a.Key == slog.MessageKey {
			return slog.Attr{}
		}
		if next == nil {
			return a
		}
		return next(groups, a)
	}
}

func New(handlerOptions *slog.HandlerOptions, options ...Option) *Handler {
	if handlerOptions == nil {
		handlerOptions = &slog.HandlerOptions{}
	}

	buf := &bytes.Buffer{}
	handler := &Handler{
		b: buf,
		h: slog.NewJSONHandler(buf, &slog.HandlerOptions{
			Level:       handlerOptions.Level,
			AddSource:   handlerOptions.AddSource,
			ReplaceAttr: suppressDefaults(handlerOptions.ReplaceAttr),
		}),
		r: handlerOptions.ReplaceAttr,
		m: &sync.Mutex{},
	}

	for _, opt := range options {
		opt(handler)
	}

	return handler
}

func NewHandler(opts *slog.HandlerOptions) *Handler {
	return New(opts, WithDestinationWriter(os.Stdout), WithOutputEmptyAttrs())
}

type Option func(h *Handler)

func WithDestinationWriter(writer io.Writer) Option {
	return func(h *Handler) {
		h.writer = writer
	}
}

func WithOutputEmptyAttrs() Option {
	return func(h *Handler) {
		h.outputEmptyAttrs = true
	}
}
