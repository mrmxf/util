//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package slogger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

var bufferPool = &sync.Pool{
	New: func() any { return new(buffer) },
}

var cwd, _ = os.Getwd()

// PrettyHandlerOptions are options for a ConsoleHandler.
// A zero PrettyHandlerOptions consists entirely of default values.
type PrettyHandlerOptions struct {
	// AddSource causes the handler to compute the source code position
	// of the log statement and add a SourceKey attribute to the output.
	AddSource bool

	// Bare - only output the message
	// Log -IN "First part "
	// Log -IBN ", and second part"
	// Log -IB ", of a 3 part console only log"
	Bare bool

	// Level reports the minimum record level that will be logged.
	// The handler discards records with lower levels.
	// If Level is nil, the handler assumes LevelInfo.
	// The handler calls Level.Level for each record processed;
	// to adjust the minimum level dynamically, use a LevelVar.
	Level slog.Leveler

	// Disable colorized output
	NoColor bool

	// Disable newline
	NoNewLine bool

	// TimeFormat is the format used for time.DateTime
	TimeFormat string

	// Theme defines the colorized output using ANSI escape sequences
	Theme Theme
}

type Handler struct {
	opts    PrettyHandlerOptions
	out     io.Writer
	group   string
	context buffer
	enc     *encoder
}

var _ slog.Handler = (*Handler)(nil)

// NewPrettyHandler creates a Handler that writes to w,
// using the given options.
// If opts is nil, the default options are used.
func NewPrettyHandler(out io.Writer, opts *PrettyHandlerOptions) *Handler {
	if opts == nil {
		opts = new(PrettyHandlerOptions)
	}
	if opts.Level == nil {
		opts.Level = LevelInfo
	}
	if opts.TimeFormat == "" {
		opts.TimeFormat = time.DateTime
	}
	if opts.Theme == nil {
		opts.Theme = NewDefaultTheme()
	}
	return &Handler{
		opts:    *opts, // Copy struct
		out:     out,
		group:   "",
		context: nil,
		enc:     &encoder{opts: *opts},
	}
}

// Enabled implements slog.Handler.
func (h *Handler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.opts.Level.Level()
}

// Handle implements slog.Handler.
func (h *Handler) Handle(_ context.Context, rec slog.Record) error {
	buf := bufferPool.Get().(*buffer)

	h.enc.writeTimestamp(buf, rec.Time)
	h.enc.writeLevel(buf, rec.Level)
	if h.opts.AddSource && rec.PC > 0 {
		h.enc.writeSource(buf, rec.PC, cwd)
	}
	h.enc.writeMessage(buf, rec.Level, rec.Message)
	buf.copy(&h.context)
	rec.Attrs(func(a slog.Attr) bool {
		h.enc.writeAttr(buf, a, h.group)
		return true
	})
	h.enc.ColorOff(buf)
	h.enc.NewLine(buf)
	if _, err := buf.WriteTo(h.out); err != nil {
		buf.Reset()
		bufferPool.Put(buf)
		return err
	}
	bufferPool.Put(buf)
	return nil
}

// WithAttrs implements slog.Handler.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newCtx := h.context
	for _, a := range attrs {
		h.enc.writeAttr(&newCtx, a, h.group)
	}
	newCtx.Clip()
	return &Handler{
		opts:    h.opts,
		out:     h.out,
		group:   h.group,
		context: newCtx,
		enc:     h.enc,
	}
}

// WithGroup implements slog.Handler.
func (h *Handler) WithGroup(name string) slog.Handler {
	name = strings.TrimSpace(name)
	if h.group != "" {
		name = h.group + "." + name
	}
	return &Handler{
		opts:    h.opts,
		out:     h.out,
		group:   name,
		context: h.context,
		enc:     h.enc,
	}
}
