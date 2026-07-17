// Package hub tracks live websocket connections per match and broadcasts
// JSON messages to the players connected to a match.
package hub

import (
	"sync"

	"github.com/gorilla/websocket"
)

// conn wraps a websocket connection with a write mutex since gorilla/websocket
// does not allow concurrent writers on the same connection.
type conn struct {
	ws *websocket.Conn
	mu sync.Mutex
}

func (c *conn) writeJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ws.WriteJSON(v)
}

// Hub is a registry of connected players, keyed by match then user.
type Hub struct {
	mu    sync.Mutex
	conns map[int64]map[int64]*conn
}

func New() *Hub {
	return &Hub{conns: make(map[int64]map[int64]*conn)}
}

// Register adds a connection for userID in matchID.
func (h *Hub) Register(matchID, userID int64, ws *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[matchID] == nil {
		h.conns[matchID] = make(map[int64]*conn)
	}
	h.conns[matchID][userID] = &conn{ws: ws}
}

// Unregister removes userID's connection from matchID.
func (h *Hub) Unregister(matchID, userID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.conns[matchID]; ok {
		delete(conns, userID)
		if len(conns) == 0 {
			delete(h.conns, matchID)
		}
	}
}

// ConnectedCount returns how many players currently have an open connection
// to matchID.
func (h *Hub) ConnectedCount(matchID int64) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.conns[matchID])
}

// Broadcast sends v as JSON to every connection currently registered for
// matchID.
func (h *Hub) Broadcast(matchID int64, v any) {
	h.mu.Lock()
	conns := make([]*conn, 0, len(h.conns[matchID]))
	for _, c := range h.conns[matchID] {
		conns = append(conns, c)
	}
	h.mu.Unlock()

	for _, c := range conns {
		_ = c.writeJSON(v)
	}
}
