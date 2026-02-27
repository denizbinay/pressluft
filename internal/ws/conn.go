package ws

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

type Conn struct {
	conn     *websocket.Conn
	serverID int64
	lastSeen time.Time
	mu       sync.Mutex
}

func NewConn(wsConn *websocket.Conn, serverID int64) *Conn {
	return &Conn{
		conn:     wsConn,
		serverID: serverID,
		lastSeen: time.Now(),
	}
}

func (c *Conn) ServerID() int64 {
	return c.serverID
}

func (c *Conn) Send(ctx context.Context, env Envelope) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(env)
	if err != nil {
		return err
	}

	return c.conn.Write(ctx, websocket.MessageText, data)
}

func (c *Conn) Receive(ctx context.Context) (Envelope, error) {
	_, data, err := c.conn.Read(ctx)
	if err != nil {
		return Envelope{}, err
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return Envelope{}, err
	}

	return env, nil
}

func (c *Conn) Close() error {
	return c.conn.Close(websocket.StatusNormalClosure, "")
}

func (c *Conn) UpdateLastSeen() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastSeen = time.Now()
}

func (c *Conn) LastSeen() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastSeen
}
