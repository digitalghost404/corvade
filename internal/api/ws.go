package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
)

// Event represents a real-time event to be broadcasted to connected clients.
type Event struct {
	Type string      `json:"event"`
	Data interface{} `json:"data"`
}

// Client represents a connected WebSocket client.
type Client struct {
	hub  *Hub
	send chan []byte
}

// Hub manages all connected WebSocket clients and broadcasts events to them.
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
	}
}

// Run starts the hub's event loop, handling client registrations,
// unregistrations, and event broadcasts.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client's send channel is full, drop the message to avoid deadlock
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(c *Client) {
	h.register <- c
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

// Broadcast sends a marshalled Event to all connected clients.
func (h *Hub) Broadcast(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.broadcast <- data
}

// Emit creates and broadcasts an event with the given type and data.
// This implements the proxy.EventEmitter interface.
func (h *Hub) Emit(event string, data interface{}) {
	h.Broadcast(Event{Type: event, Data: data})
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// HandleWebSocket upgrades an HTTP connection to WebSocket and starts
// reading/writing messages for the connected client.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &Client{hub: h, send: make(chan []byte, 256)}
	h.Register(client)

	// Writer goroutine: send messages from channel to WebSocket
	go func() {
		defer conn.Close()
		for msg := range client.send {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				break
			}
		}
	}()

	// Reader goroutine: drain incoming messages and clean up on disconnect
	go func() {
		defer func() {
			h.Unregister(client)
			conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}
