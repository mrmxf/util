//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

// Package bare provides a single-call convenience wrapper around gommi for
// serving an embedded filesystem over HTTP.
package bare

import (
	"embed"
	"fmt"
	"log"
	"net/http"

	"github.com/mrmxf/util/gommi"
)

// Serve starts an HTTP server that mounts embedFs at mountPath under prefix,
// listening on the given port.  It calls log.Fatal if setup or serving fails.
func Serve(embedFs embed.FS, prefix, mountPath string, port int) {
	r, err := gommi.Bare()
	if err != nil {
		log.Fatal(err)
	}
	if err := r.NewEmbedFileServer(embedFs, prefix, mountPath); err != nil {
		log.Fatal(err)
	}
	addr := fmt.Sprintf(":%d", port)
	log.Fatal(http.ListenAndServe(addr, r))
}
