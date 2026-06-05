//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package gommi

import (
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// NewFileServer serves static files from mountPath on the OS filesystem at prefix.
func (m *Mux) NewFileServer(prefix string, mountPath string) error {
	abs, err := filepath.Abs(mountPath)
	if err != nil {
		msg := "gommi.NewFileServer cannot find mountPath"
		slog.Error(msg, "mountPath", mountPath)
		if opt.AbortOnError {
			os.Exit(1)
		}
		return errors.New(msg)
	}
	m.webFs = os.DirFS(abs)
	slog.Info("initialising os file server on prefix", "prefix", prefix, "mountPath", abs)
	return m.fileServerFs(prefix)
}

// NewEmbedFileServer serves static files from an embed.FS sub-tree at prefix.
func (m *Mux) NewEmbedFileServer(embedFs embed.FS, prefix string, mountPath string) error {
	sub, err := fs.Sub(embedFs, mountPath)
	if err != nil {
		msg := "gommi.NewEmbedFileServer cannot find mountPath"
		slog.Error(msg, "mountPath", mountPath)
		return errors.New(msg)
	}
	m.webFs = sub
	slog.Info("initialising embed file server on prefix", "prefix", prefix, "mountPath", mountPath)
	return m.fileServerFs(prefix)
}

// fileServerFs registers GET handlers for the normalised prefix tree.
func (m *Mux) fileServerFs(prefix string) error {
	if strings.ContainsAny(prefix, "{}*") {
		msg := "gommi.fileServerFs route does not permit URL parameters"
		slog.Error(msg)
		return errors.New(msg)
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if prefix != "/" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// Trim the trailing slash for StripPrefix so "/static/" strips "/static".
	stripped := strings.TrimSuffix(prefix, "/")
	fileServer := http.StripPrefix(stripped, http.FileServer(http.FS(m.webFs)))

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		setContentType(w, req)
		fileServer.ServeHTTP(w, req)
	})

	// Register both the exact prefix and the wildcard subtree.
	m.mux.Handle("GET "+prefix+"{path...}", handler)
	if prefix != "/" {
		m.mux.Handle("GET "+strings.TrimSuffix(prefix, "/"), handler)
	}
	return nil
}

func setContentType(w http.ResponseWriter, r *http.Request) {
	ext := filepath.Ext(r.RequestURI)
	switch ext {
	// Text & web
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".csv":
		w.Header().Set("Content-Type", "text/csv")
	case ".htm", ".html":
		w.Header().Set("Content-Type", "text/html")
	case ".js", ".mjs":
		w.Header().Set("Content-Type", "text/javascript")
	case ".md":
		w.Header().Set("Content-Type", "text/markdown")
	case ".txt":
		w.Header().Set("Content-Type", "text/plain")
	// Data formats
	case ".json":
		w.Header().Set("Content-Type", "application/json")
	case ".toml":
		w.Header().Set("Content-Type", "application/toml")
	case ".wasm":
		w.Header().Set("Content-Type", "application/wasm")
	case ".xml":
		w.Header().Set("Content-Type", "application/xml")
	case ".yaml", ".yml":
		w.Header().Set("Content-Type", "application/yaml")
	case ".pdf":
		w.Header().Set("Content-Type", "application/pdf")
	case ".zip":
		w.Header().Set("Content-Type", "application/zip")
	case ".7z":
		w.Header().Set("Content-Type", "application/x-7z-compressed")
	case ".bz2":
		w.Header().Set("Content-Type", "application/x-bzip2")
	case ".gz":
		w.Header().Set("Content-Type", "application/gzip")
	case ".rar":
		w.Header().Set("Content-Type", "application/vnd.rar")
	case ".tar":
		w.Header().Set("Content-Type", "application/x-tar")
	case ".xz":
		w.Header().Set("Content-Type", "application/x-xz")
	case ".zst":
		w.Header().Set("Content-Type", "application/zstd")
	// Source code
	case ".c", ".h":
		w.Header().Set("Content-Type", "text/x-csrc")
	case ".cpp", ".cc", ".cxx", ".hpp":
		w.Header().Set("Content-Type", "text/x-c++src")
	case ".cs":
		w.Header().Set("Content-Type", "text/x-csharp")
	case ".go":
		w.Header().Set("Content-Type", "text/x-go")
	case ".java":
		w.Header().Set("Content-Type", "text/x-java")
	case ".kt":
		w.Header().Set("Content-Type", "text/x-kotlin")
	case ".lua":
		w.Header().Set("Content-Type", "text/x-lua")
	case ".php":
		w.Header().Set("Content-Type", "application/x-httpd-php")
	case ".py":
		w.Header().Set("Content-Type", "text/x-python")
	case ".rb":
		w.Header().Set("Content-Type", "text/x-ruby")
	case ".rs":
		w.Header().Set("Content-Type", "text/x-rustsrc")
	case ".sh", ".bash":
		w.Header().Set("Content-Type", "text/x-shellscript")
	case ".swift":
		w.Header().Set("Content-Type", "text/x-swift")
	case ".tsx":
		w.Header().Set("Content-Type", "text/typescript")
	case ".jsx":
		w.Header().Set("Content-Type", "text/jsx")
	// 3D objects
	case ".dae":
		w.Header().Set("Content-Type", "model/vnd.collada+xml")
	case ".fbx":
		w.Header().Set("Content-Type", "application/octet-stream")
	case ".gltf":
		w.Header().Set("Content-Type", "model/gltf+json")
	case ".glb":
		w.Header().Set("Content-Type", "model/gltf-binary")
	case ".obj":
		w.Header().Set("Content-Type", "model/obj")
	case ".ply":
		w.Header().Set("Content-Type", "model/ply")
	case ".stl":
		w.Header().Set("Content-Type", "model/stl")
	case ".usdz":
		w.Header().Set("Content-Type", "model/vnd.usdz+zip")
	case ".x3d":
		w.Header().Set("Content-Type", "model/x3d+xml")
	// CAD
	case ".dxf":
		w.Header().Set("Content-Type", "image/vnd.dxf")
	case ".dwg":
		w.Header().Set("Content-Type", "image/vnd.dwg")
	case ".fcstd", ".fcstd1":
		w.Header().Set("Content-Type", "application/x-freecad")
	case ".brep", ".brp":
		w.Header().Set("Content-Type", "application/octet-stream")
	case ".iges", ".igs":
		w.Header().Set("Content-Type", "model/iges")
	case ".step", ".stp":
		w.Header().Set("Content-Type", "model/step")
	case ".3mf":
		w.Header().Set("Content-Type", "model/3mf")
	case ".amf":
		w.Header().Set("Content-Type", "model/amf")
	// Images
	case ".avif":
		w.Header().Set("Content-Type", "image/avif")
	case ".bmp":
		w.Header().Set("Content-Type", "image/bmp")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".tif", ".tiff":
		w.Header().Set("Content-Type", "image/tiff")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	// Video
	case ".avi":
		w.Header().Set("Content-Type", "video/x-msvideo")
	case ".m4v":
		w.Header().Set("Content-Type", "video/x-m4v")
	case ".mkv":
		w.Header().Set("Content-Type", "video/x-matroska")
	case ".mov":
		w.Header().Set("Content-Type", "video/quicktime")
	case ".mp4":
		w.Header().Set("Content-Type", "video/mp4")
	case ".mpeg", ".mpg":
		w.Header().Set("Content-Type", "video/mpeg")
	case ".mxf":
		w.Header().Set("Content-Type", "application/mxf")
	case ".ogv":
		w.Header().Set("Content-Type", "video/ogg")
	case ".ts":
		w.Header().Set("Content-Type", "video/mp2t")
	case ".webm":
		w.Header().Set("Content-Type", "video/webm")
	// Audio
	case ".aac":
		w.Header().Set("Content-Type", "audio/aac")
	case ".flac":
		w.Header().Set("Content-Type", "audio/flac")
	case ".m4a":
		w.Header().Set("Content-Type", "audio/mp4")
	case ".mp3":
		w.Header().Set("Content-Type", "audio/mpeg")
	case ".oga", ".ogg":
		w.Header().Set("Content-Type", "audio/ogg")
	case ".opus":
		w.Header().Set("Content-Type", "audio/opus")
	case ".wav":
		w.Header().Set("Content-Type", "audio/wav")
	case ".weba":
		w.Header().Set("Content-Type", "audio/webm")
	// Fonts
	case ".eot":
		w.Header().Set("Content-Type", "application/vnd.ms-fontobject")
	case ".otf":
		w.Header().Set("Content-Type", "font/otf")
	case ".ttf":
		w.Header().Set("Content-Type", "font/ttf")
	case ".woff":
		w.Header().Set("Content-Type", "font/woff")
	case ".woff2":
		w.Header().Set("Content-Type", "font/woff2")
	default:
		w.Header().Set("Content-Type", "text/html")
	}
}
