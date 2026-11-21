package httpapi

import (
	"sync"

	Websocket "github.com/gorilla/websocket"
)

type Peer struct {
	Conn *Websocket.Conn
	Send chan any
	Done chan struct{}
	Once sync.Once
}

func (p *Peer) Close() { p.Once.Do(func() { close(p.Done); _ = p.Conn.Close() }) }

type Hub struct {
	mu          sync.RWMutex
	rooms       map[string]map[*Peer]bool
	presence    map[int64]map[string]*presenceSession
	callSignals map[string]*callSignalBuffer
	bus         *redisBus
}

func NewHub() *Hub {
	return &Hub{
		rooms: map[string]map[*Peer]bool{}, presence: map[int64]map[string]*presenceSession{},
		callSignals: map[string]*callSignalBuffer{},
	}
}
func (h *Hub) Add(room string, p *Peer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[room] == nil {
		h.rooms[room] = map[*Peer]bool{}
	}
	h.rooms[room][p] = true
}
func (h *Hub) Remove(room string, p *Peer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms[room], p)
	if len(h.rooms[room]) == 0 {
		delete(h.rooms, room)
	}
}
func (h *Hub) Broadcast(room string, v any) { h.publish(room, v) }
func (h *Hub) broadcastLocal(room string, v any) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for p := range h.rooms[room] {
		select {
		case p.Send <- v:
		case <-p.Done:
		default:
			p.Close()
		}
	}
}

// Count is used to observe that a connection joined a room before publishing.
func (h *Hub) Count(room string) int { h.mu.RLock(); defer h.mu.RUnlock(); return len(h.rooms[room]) }
