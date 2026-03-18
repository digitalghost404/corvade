package api

import (
	"net/http"
	"strings"

	"github.com/corvade/corvade/internal/api/handlers"
	"github.com/corvade/corvade/internal/capture"
)

// RegisterRoutes wires all REST and WebSocket endpoints onto the given mux.
func RegisterRoutes(mux *http.ServeMux, store *capture.Store, hub *Hub) {
	th := handlers.NewTracesHandler(store)
	sh := handlers.NewSessionsHandler(store)
	topo := handlers.NewTopologyHandler(store)
	exp := handlers.NewExportHandler(store)
	diff := handlers.NewDiffHandler(store)

	mux.HandleFunc("/api/traces", th.List)
	mux.HandleFunc("/api/traces/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/traces/")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		th.Get(w, r, id)
	})

	mux.HandleFunc("/api/sessions", sh.List)
	mux.HandleFunc("/api/sessions/", func(w http.ResponseWriter, r *http.Request) {
		// Path: /api/sessions/{id}
		//       /api/sessions/{id}/graph
		//       /api/sessions/{id1}/diff/{id2}
		rest := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
		if rest == "" {
			http.NotFound(w, r)
			return
		}

		parts := strings.SplitN(rest, "/", 3)
		id := parts[0]

		if len(parts) == 1 {
			sh.Get(w, r, id)
			return
		}

		if len(parts) == 2 && parts[1] == "graph" {
			topo.Get(w, r, id)
			return
		}

		if len(parts) == 2 && parts[1] == "narrative" {
			topo.GetNarrative(w, r, id)
			return
		}

		if len(parts) == 3 && parts[1] == "narrative" && parts[2] == "enhance" {
			topo.EnhanceNarrative(w, r, id)
			return
		}

		if len(parts) == 3 && parts[1] == "diff" {
			diff.Diff(w, r, id, parts[2])
			return
		}

		http.NotFound(w, r)
	})

	mux.HandleFunc("/api/stats", exp.Stats)

	mux.HandleFunc("/ws", hub.HandleWebSocket)
}
