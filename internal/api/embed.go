package api

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dashboard_dist
var dashboardFS embed.FS

// DashboardHandler returns an http.Handler that serves the embedded Next.js
// static export. It handles clean URLs by serving .html files for paths
// without extensions (e.g., /sessions → sessions.html).
func DashboardHandler() http.Handler {
	sub, err := fs.Sub(dashboardFS, "dashboard_dist")
	if err != nil {
		panic("embed: failed to sub into dashboard_dist: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Clean URL routing for Next.js static export:
		// /sessions → sessions.html, /diff → diff.html, etc.
		if path != "/" && !strings.Contains(path, ".") {
			// Try serving path.html
			htmlPath := strings.TrimPrefix(path, "/") + ".html"
			if f, err := sub.Open(htmlPath); err == nil {
				f.Close()
				r.URL.Path = path + ".html"
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		fileServer.ServeHTTP(w, r)
	})
}
