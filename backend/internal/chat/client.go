package chat

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 8192
)

// Client represents a single active WebSocket connection.
type Client struct {
	Hub  *Hub
	Conn *websocket.Conn
	Send chan []byte

	// Client metadata
	RoomID     string
	UserID     string
	Username   string
	AvatarURL  string
	NotifyOnly bool

	// Token-bucket rate limiter properties
	tokens     float64
	rate       float64 // tokens refilled per second (e.g. 1.0 token/sec = 10 tokens per 10 seconds)
	capacity   float64 // max burst capacity (e.g. 10.0 tokens)
	lastRefill time.Time
	limiterMu  sync.Mutex

	closeSendOnce sync.Once
}

// NewClient instantiates a new Client with a token-bucket rate limiter.
func NewClient(hub *Hub, conn *websocket.Conn, roomID, userID, username, avatarURL string) *Client {
	return &Client{
		Hub:        hub,
		Conn:       conn,
		Send:       make(chan []byte, 256),
		RoomID:     roomID,
		UserID:     userID,
		Username:   username,
		AvatarURL:  avatarURL,
		tokens:     10.0, // Initial balance
		rate:       1.0,  // Refill rate (1 token per second = 10 tokens per 10s)
		capacity:   10.0, // Max burst size
		lastRefill: time.Now(),
	}
}

// Interface Getters for Hub compliance
func (c *Client) GetRoomID() string   { return c.RoomID }
func (c *Client) GetUserID() string   { return c.UserID }
func (c *Client) GetUsername() string { return c.Username }
func (c *Client) GetAvatarURL() string { return c.AvatarURL }
func (c *Client) IsNotifyOnly() bool   { return c.NotifyOnly }

// NewNotifyClient creates a WebSocket client for user-level notifications only.
func NewNotifyClient(hub *Hub, conn *websocket.Conn, userID, username, avatarURL string) *Client {
	return &Client{
		Hub:        hub,
		Conn:       conn,
		Send:       make(chan []byte, 256),
		RoomID:     "",
		UserID:     userID,
		Username:   username,
		AvatarURL:  avatarURL,
		NotifyOnly: true,
		tokens:     10.0,
		rate:       1.0,
		capacity:   10.0,
		lastRefill: time.Now(),
	}
}

// CloseSend safely closes the send channel once.
func (c *Client) CloseSend() {
	c.closeSendOnce.Do(func() {
		close(c.Send)
	})
}

// SendRaw enqueues raw bytes directly onto the write channel.
// Safe if the client disconnected before the send completes.
func (c *Client) SendRaw(payload []byte) {
	defer func() {
		if recover() != nil {
			slog.Debug("dropped send to disconnected client", "user_id", c.UserID)
		}
	}()

	select {
	case c.Send <- payload:
	default:
		// Send buffer full, drop connection
		select {
		case c.Hub.Unregister <- c:
		default:
		}
	}
}

// SendEvent wraps and encodes an event to the client's write channel.
func (c *Client) SendEvent(eventType string, payload interface{}) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal client event payload", "type", eventType, "error", err)
		return
	}

	ev := Event{
		Type:    eventType,
		RoomID:  c.RoomID,
		Payload: json.RawMessage(payloadBytes),
	}

	evBytes, err := json.Marshal(ev)
	if err != nil {
		slog.Error("failed to marshal client event envelope", "type", eventType, "error", err)
		return
	}

	c.SendRaw(evBytes)
}

// Allow evaluates the token-bucket state. Returns true if request is allowed.
func (c *Client) Allow() bool {
	c.limiterMu.Lock()
	defer c.limiterMu.Unlock()

	now := time.Now()
	elapsed := now.Sub(c.lastRefill)
	c.lastRefill = now

	// Refill tokens
	c.tokens += elapsed.Seconds() * c.rate
	if c.tokens > c.capacity {
		c.tokens = c.capacity
	}

	// Consume token
	if c.tokens >= 1.0 {
		c.tokens -= 1.0
		return true
	}

	return false
}

// ReadPump loops reading incoming messages, applying rate limits, and processing events.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	
	// Keep renewing read deadlines upon receiving pongs
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("websocket read connection closed unexpectedly", "error", err)
			}
			break
		}

		// Parse envelope
		var ev Event
		if err := json.Unmarshal(messageBytes, &ev); err != nil {
			c.SendEvent("error", "Invalid JSON messaging envelope")
			continue
		}

		// Verify event targets the client's registered channel
		if !c.NotifyOnly && ev.RoomID != c.RoomID {
			c.SendEvent("error", "Mismatch room ID target")
			continue
		}

		// Handle events
		switch ev.Type {
		case "ping":
			c.SendEvent("pong", nil)

		case "typing.start", "typing.stop":
			if c.NotifyOnly {
				c.SendEvent("error", "Unsupported event type")
				continue
			}
			c.Hub.BroadcastRoom(c.RoomID, ev.Type, map[string]string{
				"user_id":  c.UserID,
				"username": c.Username,
			}, c)

		case "message.send":
			if c.NotifyOnly {
				c.SendEvent("error", "Unsupported event type")
				continue
			}
			// Rate limiting check
			if !c.Allow() {
				c.SendEvent("error", "Rate limit exceeded. Maximum 10 messages per 10 seconds.")
				continue
			}

			// Parse content and optional attachment
			var msgPayload struct {
				Content        string `json:"content"`
				AttachmentURL  string `json:"attachment_url"`
				AttachmentType string `json:"attachment_type"`
			}
			if err := json.Unmarshal(ev.Payload, &msgPayload); err != nil {
				c.SendEvent("error", "Invalid message payload")
				continue
			}

			// Require at least content or attachment
			if msgPayload.Content == "" && msgPayload.AttachmentURL == "" {
				c.SendEvent("error", "Message content or attachment is required")
				continue
			}

			// Construct database entity
			msg := Message{
				ID:              uuid.New().String(),
				RoomID:          c.RoomID,
				SenderID:        c.UserID,
				SenderName:      c.Username,
				SenderAvatarURL: c.AvatarURL,
				Content:         msgPayload.Content,
				AttachmentURL:   msgPayload.AttachmentURL,
				AttachmentType:  msgPayload.AttachmentType,
				CreatedAt:       time.Now(),
			}

			// Persist in background database context
			err := c.Hub.Store.Save(context.Background(), msg)
			if err != nil {
				slog.Error("failed to persist client message", "error", err)
				c.SendEvent("error", "Internal server error saving message")
				continue
			}

			// Broadcast message to everyone in the room
			c.Hub.BroadcastRoom(c.RoomID, "message.new", msg, nil)

			go c.Hub.NotifyMessageMentions(c.RoomID, c.UserID, c.Username, msg.ID, msg.Content)
			go c.Hub.NotifyDirectMessage(c.RoomID, c.UserID, c.Username, msg.ID, msg.Content)

		default:
			c.SendEvent("error", "Unsupported event type")
		}
	}
}

// WritePump handles flushing messages and ping tickers to connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Flush buffered messages in queue
			n := len(c.Send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte("\n"))
				_, _ = w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
