package httpapi

import (
	"sync"

	Websocket "github.com/gorilla/websocket"
)

type Peer struct {
	Conn	*Websocket.Conn
	Send	chan any
	Done	chan struct{}
	Once	sync.Once
}

func (peer *Peer) Close()	{ peer.Once.Do(func() { close(peer.Done); _ = peer.Conn.Close() }) }

type Hub struct {
	mu		sync.RWMutex
	rooms		map[string]map[*Peer]bool
	presence	map[int64]map[string]*presenceSession
	callSignals	map[string]*callSignalBuffer
	bus		*redisBus
}

func NewHub() *Hub {
	return &Hub{
		rooms:	map[string]map[*Peer]bool{}, presence: map[int64]map[string]*presenceSession{},
		callSignals:	map[string]*callSignalBuffer{},
	}
}
func (hub *Hub) Add(room string, peer *Peer) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if hub.rooms[room] == nil {
		hub.rooms[room] = map[*Peer]bool{}
	}
	hub.rooms[room][peer] = true
}
func (hub *Hub) Remove(room string, peer *Peer) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	delete(hub.rooms[room], peer)
	if len(hub.rooms[room]) == 0 {
		delete(hub.rooms, room)
	}
}
func (hub *Hub) Broadcast(room string, value any)	{ hub.publish(room, value) }
func (hub *Hub) broadcastLocal(room string, value any) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	for peer := range hub.rooms[room] {
		select {
		case peer.Send <- value:
		case <-peer.Done:
		default:
			peer.Close()
		}
	}
}

// Count is used to observe that a connection joined a room before publishing.
func (hub *Hub) Count(room string) int {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return len(hub.rooms[room])
}
