package chat

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
)

// Event represents a WebSocket messaging envelope schema.
type Event struct {
	Type    string          `json:"type"`
	RoomID  string          `json:"room_id"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// BroadcastPacket wraps outbound payloads targetting a room.
type BroadcastPacket struct {
	RoomID        string
	Payload       []byte
	ExcludeClient interface{} // ExcludeClient skips broadcasting back to sender (e.g. typing)
}

// Hub coordinates all active WebSocket client loops.
type Hub struct {
	// Active connections mapped by RoomID -> Client Map
	rooms   map[string]map[interface{}]bool
	roomsMu sync.RWMutex

	// Channel inputs
	Broadcast  chan *BroadcastPacket
	Register   chan interface{} // Accepts *Client
	Unregister chan interface{} // Accepts *Client

	// Domain helpers
	Store    MessageStore
	Contacts ContactsStore
	Members  MembersStore
	Presence PresenceManager
	PubSub   PubSub

	// Tracks active PubSub subscriptions per room
	roomSubs   map[string]context.CancelFunc
	roomSubsMu sync.Mutex

	// User-level notification delivery (all connections for a user)
	userClients   map[string]map[interface{}]bool
	userClientsMu sync.RWMutex
	userSubs      map[string]context.CancelFunc
	userSubsMu    sync.Mutex
}

// NewHub initializes a new WebSocket routing Hub.
func NewHub(store MessageStore, contacts ContactsStore, members MembersStore, presence PresenceManager, ps PubSub) *Hub {
	return &Hub{
		rooms:      make(map[string]map[interface{}]bool),
		Broadcast:  make(chan *BroadcastPacket, 256),
		Register:   make(chan interface{}, 64),
		Unregister: make(chan interface{}, 64),
		Store:      store,
		Contacts:   contacts,
		Members:    members,
		Presence:   presence,
		PubSub:     ps,
		roomSubs:      make(map[string]context.CancelFunc),
		userClients:   make(map[string]map[interface{}]bool),
		userSubs:      make(map[string]context.CancelFunc),
	}
}

// Run starts the main background Hub event listener loop.
func (h *Hub) Run(ctx context.Context) {
	slog.Info("starting global WebSocket Hub execution loop")
	for {
		select {
		case <-ctx.Done():
			slog.Info("stopping global WebSocket Hub execution loop due to context cancel")
			// Cancel all active room subscriptions
			h.roomSubsMu.Lock()
			for roomID, cancel := range h.roomSubs {
				cancel()
				delete(h.roomSubs, roomID)
			}
			h.roomSubsMu.Unlock()
			h.userSubsMu.Lock()
			for userID, cancel := range h.userSubs {
				cancel()
				delete(h.userSubs, userID)
			}
			h.userSubsMu.Unlock()
			return

		case clientObj := <-h.Register:
			h.handleRegister(clientObj, ctx)

		case clientObj := <-h.Unregister:
			h.handleUnregister(clientObj)

		case packet := <-h.Broadcast:
			h.handleBroadcast(packet)
		}
	}
}

// ensureRoomSubscription creates a PubSub subscription for a room if one doesn't exist.
// Messages received from the PubSub layer are forwarded to local clients.
func (h *Hub) ensureRoomSubscription(roomID string, parentCtx context.Context) {
	h.roomSubsMu.Lock()
	defer h.roomSubsMu.Unlock()

	if _, exists := h.roomSubs[roomID]; exists {
		return // already subscribed
	}

	subCtx, cancel := context.WithCancel(parentCtx)

	ch, err := h.PubSub.Subscribe(subCtx, "room:"+roomID)
	if err != nil {
		slog.Error("failed to subscribe to PubSub for room", "room_id", roomID, "error", err)
		cancel()
		return
	}

	h.roomSubs[roomID] = cancel

	// Background goroutine reads from PubSub and delivers to local clients
	go func() {
		defer func() {
			h.roomSubsMu.Lock()
			delete(h.roomSubs, roomID)
			h.roomSubsMu.Unlock()
			slog.Debug("PubSub subscription closed for room", "room_id", roomID)
		}()

		for msg := range ch {
			// Deliver to local clients via the Broadcast channel
			h.Broadcast <- &BroadcastPacket{
				RoomID:  roomID,
				Payload: msg,
			}
		}
	}()

	slog.Debug("PubSub subscription established for room", "room_id", roomID)
}

// removeRoomSubscription cancels the PubSub subscription for a room if no clients remain.
func (h *Hub) removeRoomSubscription(roomID string) {
	h.roomSubsMu.Lock()
	defer h.roomSubsMu.Unlock()

	h.roomsMu.RLock()
	clientCount := len(h.rooms[roomID])
	h.roomsMu.RUnlock()

	if clientCount == 0 {
		if cancel, exists := h.roomSubs[roomID]; exists {
			cancel()
			delete(h.roomSubs, roomID)
			slog.Debug("PubSub subscription removed for empty room", "room_id", roomID)
		}
	}
}

// handleRegister binds a client to a room mapping, records presence, and sends snapshots.
func (h *Hub) handleRegister(clientObj interface{}, parentCtx context.Context) {
	type clientInterface interface {
		GetRoomID() string
		GetUserID() string
		GetUsername() string
		SendEvent(eventType string, payload interface{})
		IsNotifyOnly() bool
	}

	client, ok := clientObj.(clientInterface)
	if !ok {
		return
	}

	userID := client.GetUserID()
	username := client.GetUsername()

	h.registerUserClient(userID, clientObj, parentCtx)

	if client.IsNotifyOnly() {
		client.SendEvent("connected", map[string]string{"status": "ok"})
		return
	}

	roomID := client.GetRoomID()

	h.roomsMu.Lock()
	if _, exists := h.rooms[roomID]; !exists {
		h.rooms[roomID] = make(map[interface{}]bool)
	}
	h.rooms[roomID][clientObj] = true
	h.roomsMu.Unlock()

	slog.Debug("client registered with Hub", "user_id", userID, "username", username, "room_id", roomID)

	h.ensureRoomSubscription(roomID, parentCtx)

	wasOnline := h.Presence.IsOnline(userID, roomID)

	avatarURL := ""
	if clientWithAvatar, ok := clientObj.(interface{ GetAvatarURL() string }); ok {
		avatarURL = clientWithAvatar.GetAvatarURL()
	}

	h.Presence.Join(userID, roomID, username, avatarURL)

	if !wasOnline {
		joinPayload := map[string]string{
			"user_id":    userID,
			"username":   username,
			"avatar_url": avatarURL,
		}
		h.BroadcastRoom(roomID, "presence.join", joinPayload, clientObj)
	}

	onlineUsers := h.Presence.OnlineUsers(roomID)
	snapshotPayload := map[string]interface{}{
		"online_users": onlineUsers,
	}
	client.SendEvent("presence.snapshot", snapshotPayload)
}

func (h *Hub) registerUserClient(userID string, clientObj interface{}, parentCtx context.Context) {
	h.userClientsMu.Lock()
	_, alreadyConnected := h.userClients[userID]
	if !alreadyConnected {
		h.userClients[userID] = make(map[interface{}]bool)
	}
	h.userClients[userID][clientObj] = true
	h.userClientsMu.Unlock()

	h.ensureUserSubscription(userID, parentCtx)

	// If first connection, broadcast online status to contacts
	if !alreadyConnected {
		go h.broadcastGlobalPresenceUpdate(userID, "online")
		go h.sendOnlineContactsSnapshot(userID, clientObj)
	}
}

func (h *Hub) unregisterUserClient(userID string, clientObj interface{}) {
	h.userClientsMu.Lock()
	remaining := 0
	if clients, exists := h.userClients[userID]; exists {
		delete(clients, clientObj)
		remaining = len(clients)
		if remaining == 0 {
			delete(h.userClients, userID)
		}
	}
	h.userClientsMu.Unlock()

	if remaining == 0 {
		h.removeUserSubscription(userID)
		// Broadcast offline status to contacts
		go h.broadcastGlobalPresenceUpdate(userID, "offline")
	}
}

func (h *Hub) ensureUserSubscription(userID string, parentCtx context.Context) {
	h.userSubsMu.Lock()
	defer h.userSubsMu.Unlock()

	if _, exists := h.userSubs[userID]; exists {
		return
	}

	subCtx, cancel := context.WithCancel(parentCtx)
	ch, err := h.PubSub.Subscribe(subCtx, "user:"+userID)
	if err != nil {
		slog.Error("failed to subscribe to user notifications", "user_id", userID, "error", err)
		cancel()
		return
	}

	h.userSubs[userID] = cancel

	go func() {
		defer func() {
			h.userSubsMu.Lock()
			delete(h.userSubs, userID)
			h.userSubsMu.Unlock()
		}()

		for msg := range ch {
			h.deliverToUser(userID, msg)
		}
	}()
}

func (h *Hub) removeUserSubscription(userID string) {
	h.userSubsMu.Lock()
	defer h.userSubsMu.Unlock()

	if cancel, exists := h.userSubs[userID]; exists {
		cancel()
		delete(h.userSubs, userID)
	}
}

// handleUnregister cleans up room mappings, updates presence, and broadcasts leaves.
func (h *Hub) handleUnregister(clientObj interface{}) {
	type clientInterface interface {
		GetRoomID() string
		GetUserID() string
		GetUsername() string
		CloseSend()
		IsNotifyOnly() bool
	}

	client, ok := clientObj.(clientInterface)
	if !ok {
		return
	}

	roomID := client.GetRoomID()
	userID := client.GetUserID()
	username := client.GetUsername()

	h.unregisterUserClient(userID, clientObj)

	if client.IsNotifyOnly() {
		client.CloseSend()
		return
	}

	h.roomsMu.Lock()
	if clientsMap, exists := h.rooms[roomID]; exists {
		if _, exists := clientsMap[clientObj]; exists {
			delete(clientsMap, clientObj)
			client.CloseSend()
			slog.Debug("client unregistered from Hub", "user_id", userID, "room_id", roomID)
		}
		if len(clientsMap) == 0 {
			delete(h.rooms, roomID)
		}
	}
	h.roomsMu.Unlock()

	// Update presence mapping
	h.Presence.Leave(userID, roomID)
	stillOnline := h.Presence.IsOnline(userID, roomID)

	// Broadcast leave event if the user has completely closed all client sockets in the room
	if !stillOnline {
		// Get avatar URL from any remaining client or use empty string
		avatarURL := ""
		h.BroadcastRoom(roomID, "presence.leave", map[string]string{
			"user_id":    userID,
			"username":  username,
			"avatar_url": avatarURL,
		}, nil)
	}

	// Clean up PubSub subscription if room is now empty
	h.removeRoomSubscription(roomID)
}

// handleBroadcast multicasts a raw packet across room subscribers.
func (h *Hub) handleBroadcast(packet *BroadcastPacket) {
	h.roomsMu.RLock()
	defer h.roomsMu.RUnlock()

	clientsMap, exists := h.rooms[packet.RoomID]
	if !exists {
		return
	}

	type senderInterface interface {
		SendRaw(payload []byte)
	}

	type clientIdentity interface {
		GetUserID() string
	}

	// Typing events echo back through PubSub; skip the user who is typing.
	excludeUserID := ""
	var ev Event
	if err := json.Unmarshal(packet.Payload, &ev); err == nil {
		if ev.Type == "typing.start" || ev.Type == "typing.stop" {
			var typingPayload struct {
				UserID string `json:"user_id"`
			}
			if err := json.Unmarshal(ev.Payload, &typingPayload); err == nil {
				excludeUserID = typingPayload.UserID
			}
		}
	}

	for clientObj := range clientsMap {
		if clientObj == packet.ExcludeClient {
			continue
		}
		if excludeUserID != "" {
			if client, ok := clientObj.(clientIdentity); ok && client.GetUserID() == excludeUserID {
				continue
			}
		}
		if client, ok := clientObj.(senderInterface); ok {
			client.SendRaw(packet.Payload)
		}
	}
}

// BroadcastRoom encodes an event and publishes it via the PubSub layer.
// The PubSub subscription goroutine will receive it and forward to local clients.
func (h *Hub) BroadcastRoom(roomID string, eventType string, payload interface{}, exclude interface{}) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal broadcast event payload", "type", eventType, "error", err)
		return
	}

	ev := Event{
		Type:    eventType,
		RoomID:  roomID,
		Payload: json.RawMessage(payloadBytes),
	}

	evBytes, err := json.Marshal(ev)
	if err != nil {
		slog.Error("failed to marshal broadcast event envelope", "type", eventType, "error", err)
		return
	}

	// Publish via PubSub — will be received by all instances (including this one)
	if err := h.PubSub.Publish(context.Background(), "room:"+roomID, evBytes); err != nil {
		slog.Error("failed to publish event via PubSub", "type", eventType, "room_id", roomID, "error", err)
		// Fallback: deliver directly to local clients if PubSub fails
		h.Broadcast <- &BroadcastPacket{
			RoomID:        roomID,
			Payload:       evBytes,
			ExcludeClient: exclude,
		}
	}
}
