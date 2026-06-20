package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/pulsechat/backend/internal/pubsub"
	"github.com/pulsechat/backend/internal/service"
)

type Hub struct {
	rooms   map[string]map[interface{}]bool
	roomsMu sync.RWMutex

	Broadcast  chan *BroadcastPacket
	Register   chan interface{}
	Unregister chan interface{}

	// Services instead of direct DB dependencies
	UserSvc     *service.UserService
	RoomSvc     *service.RoomService
	MsgSvc      *service.MessageService
	PresenceSvc *service.PresenceService
	PubSub      pubsub.PubSub

	roomSubs   map[string]context.CancelFunc
	roomSubsMu sync.Mutex

	userClients   map[string]map[interface{}]bool
	userClientsMu sync.RWMutex
	userSubs      map[string]context.CancelFunc
	userSubsMu    sync.Mutex
}

func NewHub(
	userSvc *service.UserService,
	roomSvc *service.RoomService,
	msgSvc *service.MessageService,
	presenceSvc *service.PresenceService,
	ps pubsub.PubSub,
) *Hub {
	return &Hub{
		rooms:       make(map[string]map[interface{}]bool),
		Broadcast:   make(chan *BroadcastPacket, 256),
		Register:    make(chan interface{}, 64),
		Unregister:  make(chan interface{}, 64),
		UserSvc:     userSvc,
		RoomSvc:     roomSvc,
		MsgSvc:      msgSvc,
		PresenceSvc: presenceSvc,
		PubSub:      ps,
		roomSubs:    make(map[string]context.CancelFunc),
		userClients: make(map[string]map[interface{}]bool),
		userSubs:    make(map[string]context.CancelFunc),
	}
}

func (h *Hub) Run(ctx context.Context) {
	slog.Info("starting global WebSocket Hub execution loop")
	for {
		select {
		case <-ctx.Done():
			slog.Info("stopping global WebSocket Hub execution loop")
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

func (h *Hub) ensureRoomSubscription(roomID string, parentCtx context.Context) {
	h.roomSubsMu.Lock()
	defer h.roomSubsMu.Unlock()

	if _, exists := h.roomSubs[roomID]; exists {
		return
	}

	subCtx, cancel := context.WithCancel(parentCtx)
	ch, err := h.PubSub.Subscribe(subCtx, "room:"+roomID)
	if err != nil {
		slog.Error("failed to subscribe to PubSub for room", "room_id", roomID, "error", err)
		cancel()
		return
	}

	h.roomSubs[roomID] = cancel

	go func() {
		defer func() {
			h.roomSubsMu.Lock()
			delete(h.roomSubs, roomID)
			h.roomSubsMu.Unlock()
		}()

		for msg := range ch {
			h.Broadcast <- &BroadcastPacket{
				RoomID:  roomID,
				Payload: msg,
			}
		}
	}()
}

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
		}
	}
}

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

	h.ensureRoomSubscription(roomID, parentCtx)

	wasOnline := h.PresenceSvc.IsOnline(userID, roomID)

	avatarURL := ""
	if clientWithAvatar, ok := clientObj.(interface{ GetAvatarURL() string }); ok {
		avatarURL = clientWithAvatar.GetAvatarURL()
	}

	h.PresenceSvc.Join(userID, roomID, username, avatarURL)

	if !wasOnline {
		joinPayload := map[string]string{
			"user_id":    userID,
			"username":   username,
			"avatar_url": avatarURL,
		}
		h.BroadcastRoom(roomID, "presence.join", joinPayload, clientObj)
	}

	onlineUsers := h.PresenceSvc.OnlineUsers(roomID)
	h.userClientsMu.RLock()
	stillConnected := false
	if clients, exists := h.userClients[userID]; exists {
		stillConnected = clients[clientObj]
	}
	h.userClientsMu.RUnlock()

	if stillConnected {
		client.SendEvent("presence.snapshot", map[string]interface{}{
			"online_users": onlineUsers,
		})
	}
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
		}
		if len(clientsMap) == 0 {
			delete(h.rooms, roomID)
		}
	}
	h.roomsMu.Unlock()

	h.PresenceSvc.Leave(userID, roomID)
	stillOnline := h.PresenceSvc.IsOnline(userID, roomID)

	if !stillOnline {
		h.BroadcastRoom(roomID, "presence.leave", map[string]string{
			"user_id":    userID,
			"username":   username,
			"avatar_url": "",
		}, nil)
	}

	h.removeRoomSubscription(roomID)
}

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

	if err := h.PubSub.Publish(context.Background(), "room:"+roomID, evBytes); err != nil {
		slog.Error("failed to publish event via PubSub", "type", eventType, "room_id", roomID, "error", err)
		h.Broadcast <- &BroadcastPacket{
			RoomID:        roomID,
			Payload:       evBytes,
			ExcludeClient: exclude,
		}
	}
}

func (h *Hub) NotifyGroupInvite(userID string, payload interface{}) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}

	ev := Event{
		Type:    "notification.group.invite",
		Payload: json.RawMessage(payloadBytes),
	}

	evBytes, err := json.Marshal(ev)
	if err != nil {
		return
	}

	_ = h.PubSub.Publish(context.Background(), "user:"+userID, evBytes)
}
