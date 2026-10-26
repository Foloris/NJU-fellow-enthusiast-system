package realtime

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"campus-fellow/server/internal/service"
	"github.com/gorilla/websocket"
)

type client struct {
	connection *websocket.Conn
	send       chan []byte
	done       chan struct{}
	once       sync.Once
}

func (c *client) close() { c.once.Do(func() { close(c.done); c.connection.Close() }) }

type Hub struct {
	mu      sync.RWMutex
	clients map[int64]map[*client]struct{}
	closed  bool
}

func New() *Hub { return &Hub{clients: make(map[int64]map[*client]struct{})} }

func (h *Hub) Publish(users []int64, event service.Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, id := range users {
		for c := range h.clients[id] {
			select {
			case c.send <- data:
			case <-c.done:
			default:
				c.close()
			}
		}
	}
}
func (h *Hub) Disconnect(user int64) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[user] {
		c.close()
	}
}
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	for _, clients := range h.clients {
		for c := range clients {
			c.close()
		}
	}
}

func (h *Hub) Serve(w http.ResponseWriter, r *http.Request, user int64) error {
	upgrader := websocket.Upgrader{HandshakeTimeout: 5 * time.Second}
	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	c := &client{connection: connection, send: make(chan []byte, 64), done: make(chan struct{})}
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		c.close()
		return nil
	}
	if h.clients[user] == nil {
		h.clients[user] = make(map[*client]struct{})
	}
	h.clients[user][c] = struct{}{}
	h.mu.Unlock()
	c.send <- []byte(`{"type":"connected"}`)
	go func() {
		defer c.close()
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-c.done:
				return
			case data := <-c.send:
				connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if connection.WriteMessage(websocket.TextMessage, data) != nil {
					return
				}
			case <-ticker.C:
				connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if connection.WriteMessage(websocket.PingMessage, nil) != nil {
					return
				}
			}
		}
	}()
	connection.SetReadLimit(4096)
	connection.SetReadDeadline(time.Now().Add(60 * time.Second))
	connection.SetPongHandler(func(string) error { return connection.SetReadDeadline(time.Now().Add(60 * time.Second)) })
	defer func() {
		c.close()
		h.mu.Lock()
		delete(h.clients[user], c)
		if len(h.clients[user]) == 0 {
			delete(h.clients, user)
		}
		h.mu.Unlock()
	}()
	for {
		if _, _, err := connection.ReadMessage(); err != nil {
			return nil
		}
	}
}
