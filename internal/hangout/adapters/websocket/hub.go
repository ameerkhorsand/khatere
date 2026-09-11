package websocket

import (
	"sync"

	"github.com/google/uuid"
)

type Hub struct {
	mu    sync.Mutex
	rooms map[uuid.UUID]map[*Client]bool
}

func NewHub() *Hub {
	return &Hub{rooms: make(map[uuid.UUID]map[*Client]bool)}
}

func (h *Hub) Register(hangoutID uuid.UUID, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[hangoutID] == nil {
		h.rooms[hangoutID] = make(map[*Client]bool)
	}
	h.rooms[hangoutID][c] = true
}

func (h *Hub) Unregister(hangoutID uuid.UUID, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.rooms[hangoutID][c]; ok {
		delete(h.rooms[hangoutID], c)
		close(c.send)
	}
}

func (h *Hub) Broadcast(hangoutID uuid.UUID, payload []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.rooms[hangoutID] {
		select {
		case c.send <- payload:
		default:
			delete(h.rooms[hangoutID], c)
			close(c.send)
		}
	}
}
