package slogpretty

import (
	"context"
	"fmt"
	"io"
	stdLog "log"
	"log/slog"
	"strings"

	"github.com/fatih/color"
)

type PrettyHandlerOptions struct {
	SlogOpts *slog.HandlerOptions
}

type PrettyHandler struct {
	opts PrettyHandlerOptions
	slog.Handler
	l     *stdLog.Logger
	attrs []slog.Attr
}

func (opts PrettyHandlerOptions) NewPrettyHandler(
	out io.Writer,
) *PrettyHandler {
	return &PrettyHandler{
		// Используем JSONHandler как базу для реализации интерфейса,
		// хотя Handle мы переопределяем полностью
		Handler: slog.NewJSONHandler(out, opts.SlogOpts),
		l:       stdLog.New(out, "", 0),
	}
}

func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	level := r.Level.String() + ":"

	switch r.Level {
	case slog.LevelDebug:
		level = color.MagentaString(level)
	case slog.LevelInfo:
		level = color.BlueString(level)
	case slog.LevelWarn:
		level = color.YellowString(level)
	case slog.LevelError:
		level = color.RedString(level)
	}

	// Собираем атрибуты в строку формата [key: value]
	var b strings.Builder

	// 1. Добавляем атрибуты из WithAttrs (контекстные)
	for _, a := range h.attrs {
		h.appendAttr(&b, a)
	}

	// 2. Добавляем атрибуты из текущей записи
	r.Attrs(func(a slog.Attr) bool {
		h.appendAttr(&b, a)
		return true
	})

	timeStr := color.WhiteString(r.Time.Format("[15:04:05.000]"))
	msg := color.CyanString(r.Message)

	h.l.Println(
		timeStr,
		level,
		msg,
		b.String(),
	)

	return nil
}

// Вспомогательный метод для единообразного форматирования атрибута
func (h *PrettyHandler) appendAttr(b *strings.Builder, a slog.Attr) {
	// Игнорируем пустые атрибуты
	if a.Equal(slog.Attr{}) {
		return
	}

	b.WriteString(" ") // Отступ между атрибутами
	b.WriteString(color.WhiteString("["))
	b.WriteString(color.New(color.FgHiWhite).Sprint(a.Key))
	b.WriteString(": ")
	b.WriteString(fmt.Sprintf("%v", a.Value.Any()))
	b.WriteString(color.WhiteString("]"))
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Важно делать append, чтобы атрибуты наслаивались при цепочке вызовов .With()
	return &PrettyHandler{
		Handler: h.Handler,
		l:       h.l,
		attrs:   append(h.attrs, attrs...),
		opts:    h.opts,
	}
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	// Для упрощения в "pretty" формате группы часто игнорируют
	// или просто проксируют в базовый handler
	return &PrettyHandler{
		Handler: h.Handler.WithGroup(name),
		l:       h.l,
		attrs:   h.attrs,
		opts:    h.opts,
	}
}
