//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package slogger

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

type NATSHandlerOptions struct {
	NATSUrl       string
	SubjectBase   string
	AppName       string
	ParentHandler slog.Handler
}

type NATSHandler struct {
	opts   NATSHandlerOptions
	nc     *nats.Conn
	parent slog.Handler
}

type LogRecord struct {
	Msg        string                 `json:"msg"`
	Time       string                 `json:"time"`
	Level      string                 `json:"level"`
	App        string                 `json:"app"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

func NewNATSHandler(opts *NATSHandlerOptions) (*NATSHandler, error) {
	if opts == nil {
		return nil, nil
	}

	nc, err := nats.Connect(opts.NATSUrl, nats.Timeout(5*time.Second))
	if err != nil {
		return &NATSHandler{
			opts:   *opts,
			nc:     nil,
			parent: opts.ParentHandler,
		}, nil
	}

	return &NATSHandler{
		opts:   *opts,
		nc:     nc,
		parent: opts.ParentHandler,
	}, nil
}

func (h *NATSHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.parent.Enabled(ctx, level)
}

func (h *NATSHandler) Handle(ctx context.Context, rec slog.Record) error {
	if err := h.parent.Handle(ctx, rec); err != nil {
		return err
	}

	if h.nc != nil && h.nc.IsConnected() {
		go h.publishToNATS(rec)
	}

	return nil
}

func (h *NATSHandler) publishToNATS(rec slog.Record) {
	attrs := make(map[string]interface{})
	rec.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	logRec := LogRecord{
		Msg:        rec.Message,
		Time:       rec.Time.Format(time.RFC3339),
		Level:      rec.Level.String(),
		App:        h.opts.AppName,
		Attributes: attrs,
	}

	data, err := json.Marshal(logRec)
	if err != nil {
		return
	}

	subject := h.opts.SubjectBase + "." + rec.Level.String() + "." + h.opts.AppName

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		h.nc.Publish(subject, data)
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}
}

func (h *NATSHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &NATSHandler{
		opts:   h.opts,
		nc:     h.nc,
		parent: h.parent.WithAttrs(attrs),
	}
}

func (h *NATSHandler) WithGroup(name string) slog.Handler {
	return &NATSHandler{
		opts:   h.opts,
		nc:     h.nc,
		parent: h.parent.WithGroup(name),
	}
}

func (h *NATSHandler) Close() error {
	if h.nc != nil && h.nc.IsConnected() {
		h.nc.Close()
	}
	return nil
}
