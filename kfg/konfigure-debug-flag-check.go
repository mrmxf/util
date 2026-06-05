//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

package kfg

import (
	"log/slog"

	"github.com/mrmxf/util/slogger"
)

func setSlogLevelDebugIfDebugFlag(options *KonfigureOpt) {
	if options == nil || options.AppArgs == nil {
		return
	}
	for _, arg := range *options.AppArgs {
		if "--debug" == arg {
			slogger.UsePrettyLogger(slog.LevelDebug)
			return
		}
	}
}
