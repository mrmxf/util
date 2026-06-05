//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

// gommi - Golang Minimal Modular InterWeb
//
// Simple static web server tools that will serve a static site from inside
// a container.
//
//	r, _ := gommi.Bare()
//	r.NewEmbedFileServer(eFS, "/", "embedWWW/")
//	http.ListenAndServe("0.0.0.0:8080", r)
package gommi

import (
	"io/fs"
	"log/slog"
	"net/http"
)

// Mux wraps http.ServeMux with middleware support and file-serving helpers.
type Mux struct {
	mux         *http.ServeMux
	webFs       fs.FS
	middlewares []func(http.Handler) http.Handler
}

var muxInstance *Mux

// Bare returns a Mux pre-wired with Recovery and Logger middleware.
func Bare(opts ...Options) (*Mux, error) {
	processOptions(opts)
	slog.SetDefault(opt.Logger)
	muxInstance = &Mux{mux: http.NewServeMux()}
	muxInstance.Use(Recovery)
	muxInstance.Use(Logger(opt.Logger))
	return muxInstance, nil
}

// Use appends a middleware to the chain. Call before ListenAndServe.
func (m *Mux) Use(middleware func(http.Handler) http.Handler) {
	m.middlewares = append(m.middlewares, middleware)
}

// Get registers a handler for GET requests matching pattern.
func (m *Mux) Get(pattern string, handler http.HandlerFunc) {
	m.mux.Handle("GET "+pattern, handler)
}

// Post registers a handler for POST requests matching pattern.
func (m *Mux) Post(pattern string, handler http.HandlerFunc) {
	m.mux.Handle("POST "+pattern, handler)
}

// ServeHTTP implements http.Handler, applying the middleware chain on each request.
func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := http.Handler(m.mux)
	for i := len(m.middlewares) - 1; i >= 0; i-- {
		h = m.middlewares[i](h)
	}
	h.ServeHTTP(w, r)
}

// GetLogger returns the slog.Logger configured in the most recent Bare() call.
func GetLogger() *slog.Logger { return opt.Logger }

// GetMux returns the Mux created by the most recent Bare() call.
func GetMux() *Mux { return muxInstance }
