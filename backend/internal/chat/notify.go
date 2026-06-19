package chat

import (
	"context"
	"encoding/json"
	"log/slog"
)

// GroupInviteNotification is sent when a user is added to a group.
type GroupInviteNotification struct {
	GroupID     string `json:"group_id"`
	GroupName   string `json:"group_name"`
	InviterID   string `json:"inviter_id"`
	InviterName string `json:"inviter_name"`
}

// DirectMessageNotification is sent when a user receives a direct chat message.
type DirectMessageNotification struct {
	RoomID     string `json:"room_id"`
	MessageID  string `json:"message_id"`
	SenderID   string `json:"sender_id"`
	SenderName string `json:"sender_name"`
	Preview    string `json:"preview"`
}

// NotifyGroupMention pushes a real-time @mention notification to a user.
func (h *Hub) NotifyGroupMention(userID string, notification GroupMentionNotification) {
	if userID == notification.SenderID {
		return
	}

	payloadBytes, err := json.Marshal(notification)
	if err != nil {
		slog.Error("failed to marshal group mention notification", "error", err)
		return
	}

	ev := Event{
		Type:    "notification.group.mention",
		RoomID:  notification.GroupID,
		Payload: json.RawMessage(payloadBytes),
	}

	evBytes, err := json.Marshal(ev)
	if err != nil {
		slog.Error("failed to marshal group mention event", "error", err)
		return
	}

	if err := h.PubSub.Publish(context.Background(), "user:"+userID, evBytes); err != nil {
		slog.Warn("failed to publish mention notification", "user_id", userID, "error", err)
		h.deliverToUser(userID, evBytes)
	}
}

// NotifyDirectMessage pushes a real-time notification for a direct chat message.
func (h *Hub) NotifyDirectMessage(roomID, senderID, senderName, messageID, content string) {
	if h.Members == nil || content == "" {
		return
	}

	ctx := context.Background()
	roomType, _, members, err := h.Members.GetRoomForMentions(ctx, roomID)
	if err != nil || roomType != "direct" {
		return
	}

	preview := content
	if len(preview) > 120 {
		preview = preview[:117] + "..."
	}

	for _, member := range members {
		if member.UserID == senderID {
			continue
		}
		payloadBytes, err := json.Marshal(DirectMessageNotification{
			RoomID:     roomID,
			MessageID:  messageID,
			SenderID:   senderID,
			SenderName: senderName,
			Preview:    preview,
		})
		if err != nil {
			slog.Error("failed to marshal direct message notification", "error", err)
			continue
		}

		ev := Event{
			Type:    "notification.direct.message",
			RoomID:  roomID,
			Payload: json.RawMessage(payloadBytes),
		}

		evBytes, err := json.Marshal(ev)
		if err != nil {
			slog.Error("failed to marshal direct message event", "error", err)
			continue
		}

		if err := h.PubSub.Publish(context.Background(), "user:"+member.UserID, evBytes); err != nil {
			slog.Warn("failed to publish direct message notification", "user_id", member.UserID, "error", err)
			h.deliverToUser(member.UserID, evBytes)
		}
	}
}

// NotifyGroupInvite pushes a real-time invite notification to a user.
func (h *Hub) NotifyGroupInvite(userID string, notification GroupInviteNotification) {
	payloadBytes, err := json.Marshal(notification)
	if err != nil {
		slog.Error("failed to marshal group invite notification", "error", err)
		return
	}

	ev := Event{
		Type:    "notification.group.invite",
		RoomID:  notification.GroupID,
		Payload: json.RawMessage(payloadBytes),
	}

	evBytes, err := json.Marshal(ev)
	if err != nil {
		slog.Error("failed to marshal group invite event", "error", err)
		return
	}

	// Publish for delivery to all instances (local subscription forwards to user connections)
	if err := h.PubSub.Publish(context.Background(), "user:"+userID, evBytes); err != nil {
		slog.Warn("failed to publish user notification", "user_id", userID, "error", err)
		// Fallback: deliver locally if PubSub fails
		h.deliverToUser(userID, evBytes)
	}
}

func (h *Hub) deliverToUser(userID string, payload []byte) {
	h.userClientsMu.RLock()
	defer h.userClientsMu.RUnlock()

	clients, exists := h.userClients[userID]
	if !exists {
		return
	}

	type notifyClient interface {
		SendRaw([]byte)
		IsNotifyOnly() bool
	}

	for clientObj := range clients {
		client, ok := clientObj.(notifyClient)
		if !ok || !client.IsNotifyOnly() {
			continue
		}
		client.SendRaw(payload)
	}
}

// broadcastGlobalPresenceUpdate notifies all contacts of a user's global status change.
func (h *Hub) broadcastGlobalPresenceUpdate(userID string, status string) {
	ctx := context.Background()
	contacts, err := h.Contacts.GetUserContacts(ctx, userID)
	if err != nil {
		slog.Error("failed to fetch user contacts for presence broadcast", "user_id", userID, "error", err)
		return
	}

	for _, contactID := range contacts {
		payload := map[string]interface{}{
			"user_id": userID,
			"status":  status,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			continue
		}
		ev := Event{
			Type:    "presence.global_update",
			Payload: json.RawMessage(payloadBytes),
		}
		evBytes, err := json.Marshal(ev)
		if err != nil {
			continue
		}
		_ = h.PubSub.Publish(ctx, "user:"+contactID, evBytes)
	}
}

// sendOnlineContactsSnapshot delivers the list of currently online contacts to a newly connected client.
func (h *Hub) sendOnlineContactsSnapshot(userID string, clientObj interface{}) {
	type senderInterface interface {
		SendEvent(eventType string, payload interface{})
	}
	client, ok := clientObj.(senderInterface)
	if !ok {
		return
	}

	ctx := context.Background()
	contacts, err := h.Contacts.GetUserContacts(ctx, userID)
	if err != nil {
		slog.Error("failed to fetch user contacts for snapshot", "user_id", userID, "error", err)
		return
	}

	h.userClientsMu.RLock()
	onlineContacts := make([]string, 0)
	stillConnected := false
	if clients, exists := h.userClients[userID]; exists {
		stillConnected = clients[clientObj]
		for _, contactID := range contacts {
			if contactClients, ok := h.userClients[contactID]; ok && len(contactClients) > 0 {
				onlineContacts = append(onlineContacts, contactID)
			}
		}
	}
	h.userClientsMu.RUnlock()

	if !stillConnected {
		return
	}

	client.SendEvent("presence.global_snapshot", map[string]interface{}{
		"online_user_ids": onlineContacts,
	})
}
