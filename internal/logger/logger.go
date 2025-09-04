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

// New создает и настраивает логгер приложения на основе конфигурации.
// Открывает файлы для обычных логов и ошибок, настраивает обработчики
// с маршрутизацией по уровням и применяет параметры форматирования.
// Возвращает ошибку при проблемах с созданием файлов логов.
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

// parseLogLevel преобразует строковое представление уровня логирования в тип slog.Level.
// Поддерживает уровни: debug, info, warn, error.
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

// LevelDispatcherHandler реализует slog.Handler с маршрутизацией сообщений по уровням.
// Сообщения уровня ERROR и выше направляются в errorHandlers, остальные - в defaultHandler.
type LevelDispatcherHandler struct {
	defaultHandler slog.Handler
	errorHandlers  slog.Handler
}

// NewLevelDispatcherHandler создает новый обработчик логов с маршрутизацией по уровням.
// Сообщения с уровнем ERROR и выше направляются в errorOut, остальные - в defaultOut.
// Позволяет разделять вывод ошибок и обычных сообщений для удобства мониторинга.
func NewLevelDispatcherHandler(defaultOut, errorOut io.Writer, opts *slog.HandlerOptions) *LevelDispatcherHandler {
	return &LevelDispatcherHandler{
		defaultHandler: NewReadableHandler(defaultOut, opts),
		errorHandlers:  NewReadableHandler(errorOut, opts),
	}
}

// Enabled определяет, обрабатывается ли указанный уровень логирования.
// Использует настройки уровня из defaultHandler для согласованности.
func (h *LevelDispatcherHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.defaultHandler.Enabled(ctx, level)
}

// Handle обрабатывает запись лога, направляя её в соответствующий обработчик.
// Сообщения уровня ERROR и выше направляются в errorHandlers,
// остальные сообщения обрабатываются defaultHandler.
func (h *LevelDispatcherHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError {
		return h.errorHandlers.Handle(ctx, r)
	}
	return h.defaultHandler.Handle(ctx, r)
}

// WithAttrs создает новый обработчик с добавленными атрибутами.
// Распространяет атрибуты на оба внутренних обработчика для согласованности.
func (h *LevelDispatcherHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LevelDispatcherHandler{
		defaultHandler: h.defaultHandler.WithAttrs(attrs),
		errorHandlers:  h.errorHandlers.WithAttrs(attrs),
	}
}

// WithGroup создает новый обработчик с добавленной группой атрибутов.
// Распространяет группу на оба внутренних обработчика.
func (h *LevelDispatcherHandler) WithGroup(name string) slog.Handler {
	return &LevelDispatcherHandler{
		defaultHandler: h.defaultHandler.WithGroup(name),
		errorHandlers:  h.errorHandlers.WithGroup(name),
	}
}

// ReadableHandler реализует slog.Handler с удобочитаемым форматированием логов.
// Форматирует сообщения в человекочитаемом виде с временными метками,
// уровнями логирования, компонентами и структурированными атрибутами.
type ReadableHandler struct {
	w    io.Writer
	opts *slog.HandlerOptions
}

// NewReadableHandler создает новый обработчик с читаемым форматированием.
// Если opts равен nil, используются настройки по умолчанию.
func NewReadableHandler(w io.Writer, opts *slog.HandlerOptions) *ReadableHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &ReadableHandler{w: w, opts: opts}
}

// Enabled определяет, обрабатывается ли указанный уровень логирования.
// Учитывает минимальный уровень, установленный в опциях обработчика.
func (h *ReadableHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

// Handle форматирует и записывает запись лога в удобочитаемом формате.
// Включает время, уровень, компонент, операцию, источник и атрибуты.
// Сообщения форматируются в едином стиле для удобства чтения и анализа.
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

// formatLevel преобразует уровень логирования в строковое представление.
// Использует заглавные буквы для consistency с общепринятыми практиками.
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

// formatAttr форматирует атрибут лога в зависимости от его типа и ключа.
// Специальное форматирование для ошибок, URL, длительностей и числовых значений.
// Обеспечивает единообразное представление часто используемых атрибутов.
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

// shortenURL сокращает длинные URL для удобства чтения в логах.
// Обрезает URL до 50 символов, оставляя только схему и домен.
func (h *ReadableHandler) shortenURL(url string) string {
	if len(url) > 50 {
		parts := strings.Split(url, "/")
		if len(parts) >= 3 {
			return fmt.Sprintf("%s//%s/...", parts[0], parts[2])
		}
	}
	return url
}

// WithAttrs возвращает тот же обработчик без изменений.
// Реализация интерфейса slog.Handler для совместимости.
func (h *ReadableHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// WithGroup возвращает тот же обработчик без изменений.
// Реализация интерфейса slog.Handler для совместимости.
func (h *ReadableHandler) WithGroup(name string) slog.Handler {
	return h
}
