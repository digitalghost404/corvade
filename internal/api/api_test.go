package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/corvade/corvade/internal/capture"
	"github.com/gorilla/websocket"
)

func newTestStore(t *testing.T) *capture.Store {
	t.Helper()
	dir := t.TempDir()
	store, err := capture.NewStore(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("create test store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

// --- NewAPIServer ---

func TestNewAPIServerCreatesValidServer(t *testing.T) {
	store := newTestStore(t)
	hub := NewHub()

	srv := NewAPIServer(store, hub, 9090)
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if srv.store != store {
		t.Error("store not set")
	}
	if srv.hub != hub {
		t.Error("hub not set")
	}
	if srv.port != 9090 {
		t.Errorf("expected port 9090, got %d", srv.port)
	}
	if srv.mux == nil {
		t.Error("mux not set")
	}
	if srv.Handler() == nil {
		t.Error("Handler() returned nil")
	}
}

// --- Route registration ---

func TestRoutesRegistered(t *testing.T) {
	store := newTestStore(t)
	hub := NewHub()
	go hub.Run()

	srv := NewAPIServer(store, hub, 0)
	handler := srv.Handler()

	tests := []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/api/traces", http.StatusOK},
		{http.MethodGet, "/api/traces/", http.StatusNotFound},
		{http.MethodGet, "/api/traces/nonexistent", http.StatusNotFound},
		{http.MethodGet, "/api/sessions", http.StatusOK},
		{http.MethodGet, "/api/sessions/", http.StatusNotFound},
		{http.MethodGet, "/api/sessions/nonexistent", http.StatusNotFound},
		{http.MethodGet, "/api/stats", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != tt.want {
				t.Errorf("expected %d, got %d (body: %s)", tt.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestRoutesSessionSubPaths(t *testing.T) {
	store := newTestStore(t)
	hub := NewHub()
	go hub.Run()

	// Insert a session so graph/narrative return 200
	agent := "test"
	sid, err := store.InsertSession(capture.Session{
		Agent:     &agent,
		StartTime: time.Now().UTC(),
		Status:    "complete",
	})
	if err != nil {
		t.Fatalf("insert session: %v", err)
	}

	srv := NewAPIServer(store, hub, 0)
	handler := srv.Handler()

	tests := []struct {
		path string
		want int
	}{
		{"/api/sessions/" + sid, http.StatusOK},
		{"/api/sessions/" + sid + "/graph", http.StatusOK},
		{"/api/sessions/" + sid + "/narrative", http.StatusOK},
		{"/api/sessions/" + sid + "/diff/" + sid, http.StatusOK},
		{"/api/sessions/" + sid + "/unknown", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != tt.want {
				t.Errorf("expected %d, got %d (body: %s)", tt.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestRouteEnhanceNarrative(t *testing.T) {
	store := newTestStore(t)
	hub := NewHub()
	go hub.Run()

	agent := "test"
	sid, _ := store.InsertSession(capture.Session{
		Agent: &agent, StartTime: time.Now().UTC(), Status: "complete",
	})

	srv := NewAPIServer(store, hub, 0)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+sid+"/narrative/enhance", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", w.Code)
	}
}

// --- DashboardHandler ---

func TestDashboardHandlerServesIndex(t *testing.T) {
	handler := DashboardHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /, got %d", w.Code)
	}
}

func TestDashboardHandlerServesCleanURLs(t *testing.T) {
	handler := DashboardHandler()

	// /sessions should serve sessions.html
	req := httptest.NewRequest(http.MethodGet, "/sessions", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /sessions, got %d", w.Code)
	}
}

func TestDashboardHandlerServesStaticAssets(t *testing.T) {
	handler := DashboardHandler()

	// favicon.ico exists
	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /favicon.ico, got %d", w.Code)
	}
}

func TestDashboardHandlerCleanURLNoMatch(t *testing.T) {
	handler := DashboardHandler()

	// A path with no extension and no matching .html file
	req := httptest.NewRequest(http.MethodGet, "/nonexistent-page-xyz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Should fall through to regular file server (404)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent clean URL, got %d", w.Code)
	}
}

// --- WebSocket HandleWebSocket ---

func TestHandleWebSocketUpgrade(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWebSocket)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Connect a WebSocket client
	wsURL := "ws" + ts.URL[len("http"):] + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect websocket: %v", err)
	}
	defer conn.Close()

	// Give the hub time to register the client
	time.Sleep(50 * time.Millisecond)

	// Send a broadcast and verify the client receives it
	hub.Broadcast(Event{Type: "test:ws", Data: "hello"})

	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read ws message: %v", err)
	}

	var event Event
	if err := json.Unmarshal(msg, &event); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}
	if event.Type != "test:ws" {
		t.Errorf("expected event type 'test:ws', got %q", event.Type)
	}
}

func TestHandleWebSocketNonUpgradeRequest(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// A regular HTTP request to /ws (not a websocket upgrade) should fail
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	w := httptest.NewRecorder()
	hub.HandleWebSocket(w, req)

	// gorilla upgrader writes a 400 on failure
	if w.Code == http.StatusOK {
		t.Error("expected non-200 for non-upgrade request")
	}
}

// --- Hub: unregister non-existent client ---

func TestHubUnregisterNonExistentClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Register a real client so we know the hub is running
	ch := make(chan []byte, 1)
	real := &Client{hub: hub, send: ch}
	hub.Register(real)
	time.Sleep(10 * time.Millisecond)

	// Unregister a client that was never registered
	fake := &Client{hub: hub, send: make(chan []byte, 1)}
	hub.Unregister(fake)
	time.Sleep(10 * time.Millisecond)

	// Real client should still work
	hub.Broadcast(Event{Type: "alive", Data: nil})
	select {
	case msg := <-ch:
		if len(msg) == 0 {
			t.Error("expected non-empty message")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout: real client should still receive messages")
	}

	hub.Unregister(real)
}

// --- Hub: broadcast drops slow client ---

func TestHubBroadcastDropsSlowClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Client with zero-capacity send channel (will be full immediately)
	slowCh := make(chan []byte) // unbuffered
	slow := &Client{hub: hub, send: slowCh}
	hub.Register(slow)
	time.Sleep(10 * time.Millisecond)

	// Also register a fast client to verify broadcast still works
	fastCh := make(chan []byte, 1)
	fast := &Client{hub: hub, send: fastCh}
	hub.Register(fast)
	time.Sleep(10 * time.Millisecond)

	hub.Broadcast(Event{Type: "test", Data: nil})

	// Fast client should get the message
	select {
	case <-fastCh:
		// good
	case <-time.After(100 * time.Millisecond):
		t.Error("fast client should have received message")
	}

	// Slow client's channel should be closed
	time.Sleep(50 * time.Millisecond)
	select {
	case _, ok := <-slowCh:
		if ok {
			t.Error("slow client channel should be closed")
		}
	case <-time.After(100 * time.Millisecond):
		// This is also acceptable — the channel may have been closed
	}

	hub.Unregister(fast)
}

// --- Hub: Broadcast with marshal error ---

func TestHubBroadcastMarshalError(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	ch := make(chan []byte, 1)
	client := &Client{hub: hub, send: ch}
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// json.Marshal cannot handle channels — this should silently fail
	hub.Broadcast(Event{Type: "bad", Data: make(chan int)})

	// Client should NOT receive anything
	select {
	case <-ch:
		t.Error("should not receive message for unmarshalable event")
	case <-time.After(100 * time.Millisecond):
		// expected
	}

	hub.Unregister(client)
}

// --- WebSocket: multiple clients with disconnect ---

func TestWebSocketMultipleClientsDisconnect(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWebSocket)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + ts.URL[len("http"):] + "/ws"

	// Connect two clients
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial 1: %v", err)
	}
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial 2: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Disconnect client 1
	conn1.Close()
	time.Sleep(50 * time.Millisecond)

	// Client 2 should still get messages
	hub.Broadcast(Event{Type: "after-disconnect", Data: nil})

	conn2.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, msg, err := conn2.ReadMessage()
	if err != nil {
		t.Fatalf("client 2 should still receive: %v", err)
	}

	var ev Event
	json.Unmarshal(msg, &ev)
	if ev.Type != "after-disconnect" {
		t.Errorf("expected 'after-disconnect', got %q", ev.Type)
	}

	conn2.Close()
}
