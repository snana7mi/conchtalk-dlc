package relay

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MessageHandler processes incoming messages from the relay server.
type MessageHandler interface {
	HandleMessage(msg IncomingMessage)
}

// Client manages a WebSocket connection to the relay server with
// automatic reconnection, heartbeat pings, and thread-safe message sending.
type Client struct {
	serverURL string
	token     string
	handler   MessageHandler

	conn   *websocket.Conn
	connMu sync.Mutex

	done chan struct{}

	// OnConnect is called after each successful WebSocket connection.
	// It can be used to send capabilities or perform other setup.
	OnConnect func()

	// attempt tracks reconnection attempts for backoff; reset on successful connect.
	attempt *int
}

// NewClient creates a new relay client. Call Run() to start the connection loop.
func NewClient(serverURL, token string, handler MessageHandler) *Client {
	return &Client{
		serverURL: serverURL,
		token:     token,
		handler:   handler,
		done:      make(chan struct{}),
	}
}

// Run is the main loop: connect, read messages, and auto-reconnect with
// exponential backoff on disconnect. It blocks until Stop() is called.
func (c *Client) Run() {
	attempt := 0
	c.attempt = &attempt
	for {
		select {
		case <-c.done:
			return
		default:
		}

		err := c.connectAndServe()
		if err != nil {
			log.Printf("[relay] connection error: %v", err)
		}

		select {
		case <-c.done:
			return
		default:
		}

		delay := time.Duration(math.Min(
			float64(2*time.Second)*math.Pow(2, float64(attempt)),
			float64(30*time.Second),
		))
		log.Printf("[relay] reconnecting in %v (attempt %d)", delay, attempt+1)

		select {
		case <-time.After(delay):
		case <-c.done:
			return
		}
		attempt++
	}
}

// Stop gracefully shuts down the client, closing the WebSocket connection.
func (c *Client) Stop() {
	select {
	case <-c.done:
		// Already closed.
	default:
		close(c.done)
	}

	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn != nil {
		_ = c.conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		_ = c.conn.Close()
		c.conn = nil
	}
}

// Send sends a message to the relay server. It is safe to call from
// multiple goroutines.
func (c *Client) Send(msg OutgoingMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// IsConnected reports whether the client currently has an active WebSocket
// connection.
func (c *Client) IsConnected() bool {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	return c.conn != nil
}

// connectAndServe dials the relay server, runs the heartbeat, and reads
// messages until the connection is lost or Stop() is called.
func (c *Client) connectAndServe() error {
	u, err := url.Parse(c.serverURL)
	if err != nil {
		return fmt.Errorf("parse server URL: %w", err)
	}
	q := u.Query()
	q.Set("role", "daemon")
	q.Set("token", c.token)
	q.Set("v", "1")
	u.RawQuery = q.Encode()

	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	// Reset backoff on successful connection.
	if c.attempt != nil {
		*c.attempt = 0
	}

	log.Printf("[relay] connected to %s", c.serverURL)

	if c.OnConnect != nil {
		c.OnConnect()
	}

	// Start heartbeat in a separate goroutine.
	heartbeatDone := make(chan struct{})
	go c.heartbeat(heartbeatDone)

	// Read loop.
	defer func() {
		close(heartbeatDone)
		c.connMu.Lock()
		c.conn = nil
		_ = conn.Close()
		c.connMu.Unlock()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		var msg IncomingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("[relay] invalid message: %v", err)
			continue
		}

		c.handler.HandleMessage(msg)
	}
}

// heartbeat sends a WebSocket ping frame and an application-level JSON ping
// every 30 seconds until the heartbeatDone channel is closed.
func (c *Client) heartbeat(done chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	ping := []byte(`{"type":"ping"}`)

	for {
		select {
		case <-done:
			return
		case <-c.done:
			return
		case <-ticker.C:
			c.connMu.Lock()
			if c.conn != nil {
				// Protocol-level ping for keepalive
				if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("[relay] heartbeat error: %v", err)
				}
				// Application-level ping so the DO can update last_seen
				_ = c.conn.WriteMessage(websocket.TextMessage, ping)
			}
			c.connMu.Unlock()
		}
	}
}
