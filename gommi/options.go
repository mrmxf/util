//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

// gommi  - Golang Minimal Modular InterWeb
// Option definitions — by design gommi provides very few options.
package gommi

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/mrmxf/util/slogger"
)

var opt *Options

type Options struct {
	AbortOnError bool
	Logger       *slog.Logger
	Port         int    // not used yet
	portStr      string // not used yet
}

var defaultOptions = Options{
	AbortOnError: true,
	Logger:       slog.New(slogger.NewPrettyHandler(os.Stdout, nil)),
	Port:         8080,
	portStr:      "8080",
}

func processOptions(opts []Options) {
	if len(opts) == 0 {
		opt = &defaultOptions
		return
	}

	opt = &opts[0]

	if opt.Port == 0 {
		opt.Port = defaultOptions.Port
	}
	opt.portStr = fmt.Sprintf("%d", opt.Port)

	if opt.Logger == nil {
		opt.Logger = defaultOptions.Logger
	}
}
