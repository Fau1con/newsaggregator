package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"news/internal/config"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	logFile      = "news.log"
	errorLogFile = "news_error.log"
)

func New(cfg config.LoggerConfig) (*slog.Logger, error) {
	logLevel := parseLogLevel(cfg.Level)
	logWriter, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", logFile, err)
	}
	errorWriter, err := os.OpenFile(errorLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open error log file %s: %v", errorLogFile, err)
	}
	handler := NewLevelDispatcherHandler(logWriter, errorWriter, &slog.HandlerOptions{
		AddSource: true,
		Level:     logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				if source, ok := a.Value.Any().(*slog.Source); ok {
					source.File = filepath.Base(source.File)
				}
			}
			return a
		},
	})
	return slog.New(handler), nil
}

// parseLogLevel преобразует строку из конфига в уровень slog.
func parseLogLevel(levelStr string) slog.Level {
	switch levelStr {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type LevelDispatcherHandler struct {
	defaultHandler slog.Handler
	errorHandlers  slog.Handler
}

func NewLevelDispatcherHandler(defaultOut, errorOut io.Writer, opts *slog.HandlerOptions) *LevelDispatcherHandler {
	return &LevelDispatcherHandler{
		defaultHandler: NewReadableHandler(defaultOut, opts),
		errorHandlers:  NewReadableHandler(errorOut, opts),
	}
}
func (h *LevelDispatcherHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.defaultHandler.Enabled(ctx, level)
}
func (h *LevelDispatcherHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError {
		return h.errorHandlers.Handle(ctx, r)
	}
	return h.defaultHandler.Handle(ctx, r)
}
func (h *LevelDispatcherHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LevelDispatcherHandler{
		defaultHandler: h.defaultHandler.WithAttrs(attrs),
		errorHandlers:  h.errorHandlers.WithAttrs(attrs),
	}
}
func (h *LevelDispatcherHandler) WithGroup(name string) slog.Handler {
	return &LevelDispatcherHandler{
		defaultHandler: h.defaultHandler.WithGroup(name),
		errorHandlers:  h.errorHandlers.WithGroup(name),
	}
}

type ReadableHandler struct {
	w    io.Writer
	opts *slog.HandlerOptions
}

func NewReadableHandler(w io.Writer, opts *slog.HandlerOptions) *ReadableHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &ReadableHandler{w: w, opts: opts}
}
func (h *ReadableHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}
func (h *ReadableHandler) Handle(ctx context.Context, r slog.Record) error {
	timeStr := r.Time.Format("15:04:05.000")
	levelStr := h.formatLevel(r.Level)
	var component, operation, source string
	var attrs []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "component":
			component = a.Value.String()
		case "op":
			operation = a.Value.String()
		case slog.SourceKey:
			if sourceVal, ok := a.Value.Any().(*slog.Source); ok {
				source = fmt.Sprintf("%s:%d", filepath.Base(sourceVal.File), sourceVal.Line)
			}
		default:
			attrs = append(attrs, a)
		}
		return true
	})
	var prefix strings.Builder
	prefix.WriteString(fmt.Sprintf("[%s] %s", timeStr, levelStr))
	if component != "" {
		prefix.WriteString(fmt.Sprintf(" [%s]", component))
	}
	if operation != "" {
		prefix.WriteString(fmt.Sprintf(" (%s)", operation))
	}
	if source != "" && h.opts.AddSource {
		prefix.WriteString(fmt.Sprintf(" <%s>", source))
	}
	message := r.Message
	var attrParts []string
	for _, attr := range attrs {
		attrParts = append(attrParts, h.formatAttr(attr))
	}
	if len(attrParts) > 0 {
		message += " | " + strings.Join(attrParts, ", ")
	}
	_, err := fmt.Fprintf(h.w, "%s: %s\n", prefix.String(), message)
	return err
}
func (h *ReadableHandler) formatLevel(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "DEBUG"
	case slog.LevelInfo:
		return "INFO"
	case slog.LevelWarn:
		return "WARN"
	case slog.LevelError:
		return "ERROR"
	default:
		return "UNKNW"
	}
}
func (h *ReadableHandler) formatAttr(attr slog.Attr) string {
	switch attr.Key {
	case "error":
		return fmt.Sprintf("error=%q", attr.Value.String())
	case "url":
		return fmt.Sprintf("url=%s", h.shortenURL(attr.Value.String()))
	case "duration":
		if duration, err := time.ParseDuration(attr.Value.String()); err != nil {
			return fmt.Sprintf("took=%s", duration.Round(time.Millisecond))
		}
		return fmt.Sprintf("duration=%s", attr.Value.String())
	case "count", "limit", "items_found":
		return fmt.Sprintf("%s=%s", attr.Key, attr.Value.String())
	default:
		return fmt.Sprintf("%s=%s", attr.Key, attr.Value.String())
	}
}
func (h *ReadableHandler) shortenURL(url string) string {
	if len(url) > 50 {
		parts := strings.Split(url, "/")
		if len(parts) >= 3 {
			return fmt.Sprintf("%s//%s/...", parts[0], parts[2])
		}
	}
	return url
}
func (h *ReadableHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}
func (h *ReadableHandler) WithGroup(name string) slog.Handler {
	return h
}
