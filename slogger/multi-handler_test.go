//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package slogger_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/mrmxf/util/slogger"
	. "github.com/smartystreets/goconvey/convey"
)

// countingHandler counts how many records it receives and records the level of
// the last one. Used to verify fan-out without I/O side-effects.
type countingHandler struct {
	minLevel slog.Level
	count    int
	last     slog.Level
}

func (h *countingHandler) Enabled(_ context.Context, l slog.Level) bool { return l >= h.minLevel }
func (h *countingHandler) Handle(_ context.Context, r slog.Record) error {
	h.count++
	h.last = r.Level
	return nil
}
func (h *countingHandler) WithAttrs(_ []slog.Attr) slog.Handler  { return h }
func (h *countingHandler) WithGroup(_ string) slog.Handler       { return h }

// attrCapturingHandler records the attrs passed to WithAttrs so we can verify
// that MultiHandler clones them independently for each child.
type attrCapturingHandler struct {
	countingHandler
	attrs []slog.Attr
}

func (h *attrCapturingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := &attrCapturingHandler{
		countingHandler: h.countingHandler,
		attrs:           append([]slog.Attr{}, attrs...),
	}
	return clone
}

func TestMultiHandler(t *testing.T) {
	Convey("MultiHandler", t, func() {

		Convey("NewMultiHandler returns a non-nil handler", func() {
			m := slogger.NewMultiHandler()
			So(m, ShouldNotBeNil)
		})

		Convey("fan-out: both handlers receive the record", func() {
			a := &countingHandler{minLevel: slog.LevelDebug}
			b := &countingHandler{minLevel: slog.LevelDebug}
			m := slogger.NewMultiHandler(a, b)

			rec := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
			So(m.Handle(context.Background(), rec), ShouldBeNil)
			So(a.count, ShouldEqual, 1)
			So(b.count, ShouldEqual, 1)
		})

		Convey("Enabled returns true if ANY handler is enabled", func() {
			low := &countingHandler{minLevel: slog.LevelDebug}
			high := &countingHandler{minLevel: slog.LevelError}
			m := slogger.NewMultiHandler(low, high)
			ctx := context.Background()

			So(m.Enabled(ctx, slog.LevelInfo), ShouldBeTrue)  // low accepts it
			So(m.Enabled(ctx, slog.LevelDebug), ShouldBeTrue) // low accepts it
		})

		Convey("Enabled returns false when no handler accepts the level", func() {
			high1 := &countingHandler{minLevel: slog.LevelError}
			high2 := &countingHandler{minLevel: slog.LevelError}
			m := slogger.NewMultiHandler(high1, high2)
			ctx := context.Background()

			So(m.Enabled(ctx, slog.LevelInfo), ShouldBeFalse)
		})

		Convey("Handle skips handlers that are not Enabled for the record level", func() {
			low := &countingHandler{minLevel: slog.LevelDebug}
			high := &countingHandler{minLevel: slog.LevelError}
			m := slogger.NewMultiHandler(low, high)
			ctx := context.Background()

			rec := slog.NewRecord(time.Now(), slog.LevelInfo, "info only", 0)
			So(m.Handle(ctx, rec), ShouldBeNil)
			So(low.count, ShouldEqual, 1)
			So(high.count, ShouldEqual, 0) // Error handler skips Info
		})

		Convey("WithAttrs returns a new MultiHandler that wraps children", func() {
			a := &attrCapturingHandler{}
			m := slogger.NewMultiHandler(a)
			attrs := []slog.Attr{slog.String("key", "val")}
			m2 := m.WithAttrs(attrs)
			So(m2, ShouldNotBeNil)
		})

		Convey("WithGroup returns a new MultiHandler", func() {
			a := &countingHandler{minLevel: slog.LevelDebug}
			m := slogger.NewMultiHandler(a)
			m2 := m.WithGroup("grp")
			So(m2, ShouldNotBeNil)
		})
	})
}
