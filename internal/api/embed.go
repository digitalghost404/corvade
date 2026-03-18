package api

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dashboard_dist
var dashboardFS embed.FS

// DashboardHandler returns an http.Handler that serves the embedded Next.js
// static export from the dashboard_dist directory.
func DashboardHandler() http.Handler {
	sub, err := fs.Sub(dashboardFS, "dashboard_dist")
	if err != nil {
		panic("embed: failed to sub into dashboard_dist: " + err.Error())
	}
	return http.FileServer(http.FS(sub))
}
