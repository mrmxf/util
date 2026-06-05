//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package slogger

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestNATSHandler_NoConnection(t *testing.T) {
	prettyHandler := NewPrettyHandler(os.Stderr, &PrettyHandlerOptions{Level: slog.LevelInfo})
	
	natsHandler, err := NewNATSHandler(&NATSHandlerOptions{
		NATSUrl:       "nats://nonexistent:4222",
		SubjectBase:   "logs",
		AppName:       "test-app",
		ParentHandler: prettyHandler,
	})
	
	if err != nil {
		t.Fatalf("NewNATSHandler should not return error when connection fails: %v", err)
	}
	
	if natsHandler == nil {
		t.Fatal("NATSHandler should not be nil even when connection fails")
	}
	
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	err = natsHandler.Handle(context.Background(), rec)
	if err != nil {
		t.Errorf("Handle should not return error when NATS publish fails: %v", err)
	}
}

func TestNATSHandler_Enabled(t *testing.T) {
	prettyHandler := NewPrettyHandler(os.Stderr, &PrettyHandlerOptions{Level: slog.LevelInfo})
	
	natsHandler, _ := NewNATSHandler(&NATSHandlerOptions{
		NATSUrl:       "nats://localhost:4222",
		SubjectBase:   "logs",
		AppName:       "test-app",
		ParentHandler: prettyHandler,
	})
	
	if !natsHandler.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Handler should be enabled for LevelInfo")
	}
	
	if natsHandler.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("Handler should not be enabled for LevelDebug")
	}
}

func TestNATSHandler_WithAttrs(t *testing.T) {
	prettyHandler := NewPrettyHandler(os.Stderr, &PrettyHandlerOptions{Level: slog.LevelInfo})
	
	natsHandler, _ := NewNATSHandler(&NATSHandlerOptions{
		NATSUrl:       "nats://localhost:4222",
		SubjectBase:   "logs",
		AppName:       "test-app",
		ParentHandler: prettyHandler,
	})
	
	attrs := []slog.Attr{
		slog.String("key", "value"),
	}
	
	newHandler := natsHandler.WithAttrs(attrs)
	if newHandler == nil {
		t.Error("WithAttrs should return a new handler")
	}
}

func TestNATSHandler_WithGroup(t *testing.T) {
	prettyHandler := NewPrettyHandler(os.Stderr, &PrettyHandlerOptions{Level: slog.LevelInfo})
	
	natsHandler, _ := NewNATSHandler(&NATSHandlerOptions{
		NATSUrl:       "nats://localhost:4222",
		SubjectBase:   "logs",
		AppName:       "test-app",
		ParentHandler: prettyHandler,
	})
	
	newHandler := natsHandler.WithGroup("test-group")
	if newHandler == nil {
		t.Error("WithGroup should return a new handler")
	}
}
