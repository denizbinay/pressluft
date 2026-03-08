package ws

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

type Conn struct {
	conn       *websocket.Conn
	serverID   int64
	lastSeen   time.Time
	version    string
	cpuPercent float64
	memUsedMB  int64
	memTotalMB int64
	mu         sync.RWMutex
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
	if c == nil || c.conn == nil {
		return errors.New("websocket transport not connected")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(env)
	if err != nil {
		return err
	}

	return c.conn.Write(ctx, websocket.MessageText, data)
}

func (c *Conn) Receive(ctx context.Context) (Envelope, error) {
	if c == nil || c.conn == nil {
		return Envelope{}, errors.New("websocket transport not connected")
	}
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
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close(websocket.StatusNormalClosure, "")
}

func (c *Conn) UpdateLastSeen() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastSeen = time.Now()
}

func (c *Conn) LastSeen() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastSeen
}

// UpdateFromHeartbeat updates connection state from a heartbeat message.
func (c *Conn) UpdateFromHeartbeat(hb Heartbeat) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastSeen = time.Now()
	c.version = hb.Version
	c.cpuPercent = hb.CPUPercent
	c.memUsedMB = hb.MemUsedMB
	c.memTotalMB = hb.MemTotalMB
}

// Version returns the agent version.
func (c *Conn) Version() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.version
}

// Metrics returns the last known CPU and memory metrics.
func (c *Conn) Metrics() (cpuPercent float64, memUsedMB, memTotalMB int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cpuPercent, c.memUsedMB, c.memTotalMB
}
