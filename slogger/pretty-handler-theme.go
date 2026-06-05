//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package slogger

import (
	"fmt"
	"log/slog"
)

type ANSIMod string

var ResetMod = ToANSICode(Reset)

const (
	Reset = iota
	Bold
	Faint
	Italic
	Underline
	CrossedOut = 9
)

const (
	Black = iota + 30
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	Gray
)

const (
	BrightBlack = iota + 90
	BrightRed
	BrightGreen
	BrightYellow
	BrightBlue
	BrightMagenta
	BrightCyan
	White
)

func (c ANSIMod) String() string {
	return string(c)
}

func ToANSICode(modes ...int) ANSIMod {
	if len(modes) == 0 {
		return ""
	}

	var s string
	for i, m := range modes {
		if i > 0 {
			s += ";"
		}
		s += fmt.Sprintf("%d", m)
	}
	return ANSIMod("\x1b[" + s + "m")
}

type Theme interface {
	Name() string
	Timestamp() ANSIMod
	Source() ANSIMod

	Message() ANSIMod
	MessageDebug() ANSIMod
	AttrKey() ANSIMod
	AttrValue() ANSIMod
	AttrValueError() ANSIMod
	LevelEmergency() ANSIMod
	LevelFatal() ANSIMod
	LevelError() ANSIMod
	LevelWarn() ANSIMod
	LevelSuccess() ANSIMod
	LevelInfo() ANSIMod
	LevelDebug() ANSIMod
	LevelTrace() ANSIMod
	Level(level slog.Level) ANSIMod
}

type ThemeDef struct {
	name           string
	timestamp      ANSIMod
	source         ANSIMod
	message        ANSIMod
	messageDebug   ANSIMod
	attrKey        ANSIMod
	attrValue      ANSIMod
	attrValueError ANSIMod
	levelEmergency ANSIMod
	levelFatal     ANSIMod
	levelError     ANSIMod
	levelWarn      ANSIMod
	levelSuccess   ANSIMod
	levelInfo      ANSIMod
	levelDebug     ANSIMod
	levelTrace     ANSIMod
}

func (t ThemeDef) Name() string            { return t.name }
func (t ThemeDef) Timestamp() ANSIMod      { return t.timestamp }
func (t ThemeDef) Source() ANSIMod         { return t.source }
func (t ThemeDef) Message() ANSIMod        { return t.message }
func (t ThemeDef) MessageDebug() ANSIMod   { return t.messageDebug }
func (t ThemeDef) AttrKey() ANSIMod        { return t.attrKey }
func (t ThemeDef) AttrValue() ANSIMod      { return t.attrValue }
func (t ThemeDef) AttrValueError() ANSIMod { return t.attrValueError }
func (t ThemeDef) LevelEmergency() ANSIMod { return t.levelEmergency }
func (t ThemeDef) LevelFatal() ANSIMod     { return t.levelFatal }
func (t ThemeDef) LevelError() ANSIMod     { return t.levelError }
func (t ThemeDef) LevelWarn() ANSIMod      { return t.levelWarn }
func (t ThemeDef) LevelSuccess() ANSIMod   { return t.levelSuccess }
func (t ThemeDef) LevelInfo() ANSIMod      { return t.levelInfo }
func (t ThemeDef) LevelDebug() ANSIMod     { return t.levelDebug }
func (t ThemeDef) LevelTrace() ANSIMod     { return t.levelTrace }
func (t ThemeDef) Level(level slog.Level) ANSIMod {
	switch {
	case level >= LevelEmergency:
		return t.LevelEmergency()
	case level >= LevelFatal:
		return t.LevelFatal()
	case level >= LevelError:
		return t.LevelError()
	case level >= LevelWarn:
		return t.LevelWarn()
	case level >= LevelSuccess:
		return t.LevelSuccess()
	case level >= slog.LevelInfo:
		return t.LevelInfo()
	case level >= slog.LevelDebug:
		return t.LevelDebug()
	default:
		return t.LevelTrace()
	}
}

func NewDefaultTheme() Theme {
	return ThemeDef{
		name:           "Default",
		timestamp:      ToANSICode(BrightBlack),
		source:         ToANSICode(Bold, BrightBlack),
		message:        ToANSICode(),
		messageDebug:   ToANSICode(),
		attrKey:        ToANSICode(Cyan),
		attrValue:      ToANSICode(),
		attrValueError: ToANSICode(Bold, Red),
		levelEmergency: ToANSICode(Bold, Red),
		levelFatal:     ToANSICode(BrightRed),
		levelError:     ToANSICode(Red),
		levelWarn:      ToANSICode(Magenta),
		levelSuccess:   ToANSICode(Green),
		levelInfo:      ToANSICode(BrightYellow),
		levelDebug:     ToANSICode(),
		levelTrace:     ToANSICode(Gray),
	}
}

func NewBrightTheme() Theme {
	return ThemeDef{
		name:           "Bright",
		timestamp:      ToANSICode(Gray),
		source:         ToANSICode(Bold, Gray),
		message:        ToANSICode(Bold, White),
		messageDebug:   ToANSICode(),
		attrKey:        ToANSICode(BrightCyan),
		attrValue:      ToANSICode(),
		attrValueError: ToANSICode(Bold, BrightRed),
		levelEmergency: ToANSICode(Bold, Red),
		levelFatal:     ToANSICode(BrightRed),
		levelError:     ToANSICode(BrightRed),
		levelWarn:      ToANSICode(BrightYellow),
		levelSuccess:   ToANSICode(BrightGreen),
		levelInfo:      ToANSICode(BrightGreen),
		levelDebug:     ToANSICode(),
		levelTrace:     ToANSICode(Gray),
	}
}
