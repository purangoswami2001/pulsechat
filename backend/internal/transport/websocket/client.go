package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pulsechat/backend/internal/domain"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 8192
)

type Client struct {
	Hub  *Hub
	Conn *websocket.Conn
	Send chan []byte

	RoomID     string
	UserID     string
	Username   string
	AvatarURL  string
	NotifyOnly bool

	tokens     float64
	rate       float64
	capacity   float64
	lastRefill time.Time
	limiterMu  sync.Mutex

	closeSendOnce sync.Once
}

func NewClient(hub *Hub, conn *websocket.Conn, roomID, userID, username, avatarURL string) *Client {
	return &Client{
		Hub:        hub,
		Conn:       conn,
		Send:       make(chan []byte, 256),
		RoomID:     roomID,
		UserID:     userID,
		Username:   username,
		AvatarURL:  avatarURL,
		tokens:     10.0,
		rate:       1.0,
		capacity:   10.0,
		lastRefill: time.Now(),
	}
}

func (c *Client) GetRoomID() string   { return c.RoomID }
	func (c *Client) GetUserID() string   { return c.UserID }
	func (c *Client) GetUsername() string { return c.Username }
	func (c *Client) GetAvatarURL() string { return c.AvatarURL }
	func (c *Client) IsNotifyOnly() bool   { return c.NotifyOnly }

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

func (c *Client) CloseSend() {
	c.closeSendOnce.Do(func() {
		close(c.Send)
	})
}

func (c *Client) SendRaw(payload []byte) {
	defer func() {
		if recover() != nil {
			slog.Debug("dropped send to disconnected client", "user_id", c.UserID)
		}
	}()

	select {
	case c.Send <- payload:
	default:
		select {
		case c.Hub.Unregister <- c:
		default:
		}
	}
}

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

func (c *Client) Allow() bool {
	c.limiterMu.Lock()
	defer c.limiterMu.Unlock()

	now := time.Now()
	elapsed := now.Sub(c.lastRefill)
	c.lastRefill = now

	c.tokens += elapsed.Seconds() * c.rate
	if c.tokens > c.capacity {
		c.tokens = c.capacity
	}

	if c.tokens >= 1.0 {
		c.tokens -= 1.0
		return true
	}

	return false
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
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

		var ev Event
		if err := json.Unmarshal(messageBytes, &ev); err != nil {
			c.SendEvent("error", "Invalid JSON messaging envelope")
			continue
		}

		if !c.NotifyOnly && ev.RoomID != c.RoomID {
			c.SendEvent("error", "Mismatch room ID target")
			continue
		}

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
			if !c.Allow() {
				c.SendEvent("error", "Rate limit exceeded. Maximum 10 messages per 10 seconds.")
				continue
			}

			var msgPayload struct {
				Content        string `json:"content"`
				AttachmentURL  string `json:"attachment_url"`
				AttachmentType string `json:"attachment_type"`
			}
			if err := json.Unmarshal(ev.Payload, &msgPayload); err != nil {
				c.SendEvent("error", "Invalid message payload")
				continue
			}

			if msgPayload.Content == "" && msgPayload.AttachmentURL == "" {
				c.SendEvent("error", "Message content or attachment is required")
				continue
			}

			msg := domain.Message{
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

			saved, err := c.Hub.MsgSvc.SaveMessage(context.Background(), msg)
			if err != nil {
				slog.Error("failed to persist client message", "error", err)
				c.SendEvent("error", "Internal server error saving message")
				continue
			}

			c.Hub.BroadcastRoom(c.RoomID, "message.new", saved, nil)

			go c.Hub.NotifyMessageMentions(c.RoomID, c.UserID, c.Username, saved.ID, saved.Content)
			go c.Hub.NotifyDirectMessage(c.RoomID, c.UserID, c.Username, saved.ID, saved.Content)

		default:
			c.SendEvent("error", "Unsupported event type")
		}
	}
}

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
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

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
