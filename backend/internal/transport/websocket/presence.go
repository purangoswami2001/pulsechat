package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"

	"github.com/pulsechat/backend/internal/domain"
)

var mentionPattern = regexp.MustCompile(`(?i)@(all|[a-zA-Z0-9_]+)`)

type GroupMentionNotification struct {
	GroupID    string `json:"group_id"`
	GroupName  string `json:"group_name"`
	MessageID  string `json:"message_id"`
	SenderID   string `json:"sender_id"`
	SenderName string `json:"sender_name"`
	Preview    string `json:"preview"`
	MentionAll bool   `json:"mention_all,omitempty"`
}

type DirectMessageNotification struct {
	RoomID     string `json:"room_id"`
	MessageID  string `json:"message_id"`
	SenderID   string `json:"sender_id"`
	SenderName string `json:"sender_name"`
	Preview    string `json:"preview"`
}

func (h *Hub) broadcastGlobalPresenceUpdate(userID string, status string) {
	ctx := context.Background()
	contacts, err := h.UserSvc.GetContacts(ctx, userID)
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

func (h *Hub) sendOnlineContactsSnapshot(userID string, clientObj interface{}) {
	type senderInterface interface {
		SendEvent(eventType string, payload interface{})
	}
	client, ok := clientObj.(senderInterface)
	if !ok {
		return
	}

	ctx := context.Background()
	contacts, err := h.UserSvc.GetContacts(ctx, userID)
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

func (h *Hub) NotifyMessageMentions(roomID, senderID, senderName, messageID, content string) {
	if h.RoomSvc == nil || content == "" {
		return
	}

	ctx := context.Background()
	roomType, roomName, members, err := h.RoomSvc.GetRoomMembersForMentions(ctx, roomID)
	if err != nil {
		slog.Error("failed to load room for mentions", "room_id", roomID, "error", err)
		return
	}

	if roomType != "group" && roomType != "private" {
		return
	}

	targets := resolveMentionTargets(content, members, senderID)
	if len(targets) == 0 {
		return
	}

	preview := content
	if len(preview) > 120 {
		preview = preview[:117] + "..."
	}

	mentionAll := false
	for _, match := range mentionPattern.FindAllStringSubmatch(content, -1) {
		if len(match) >= 2 && strings.EqualFold(match[1], "all") {
			mentionAll = true
			break
		}
	}

	for _, userID := range targets {
		h.NotifyGroupMention(userID, GroupMentionNotification{
			GroupID:    roomID,
			GroupName:  roomName,
			MessageID:  messageID,
			SenderID:   senderID,
			SenderName: senderName,
			Preview:    preview,
			MentionAll: mentionAll,
		})
	}
}

func resolveMentionTargets(content string, members []domain.MemberBrief, senderID string) []string {
	seen := make(map[string]bool)
	var targets []string

	nameToID := make(map[string]string, len(members))
	for _, m := range members {
		nameToID[strings.ToLower(m.Username)] = m.UserID
	}

	matches := mentionPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		token := strings.ToLower(match[1])

		if token == "all" {
			for _, m := range members {
				if m.UserID == senderID || seen[m.UserID] {
					continue
				}
				seen[m.UserID] = true
				targets = append(targets, m.UserID)
			}
			continue
		}

		if uid, ok := nameToID[token]; ok && uid != senderID && !seen[uid] {
			seen[uid] = true
			targets = append(targets, uid)
		}
	}

	return targets
}

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

func (h *Hub) NotifyDirectMessage(roomID, senderID, senderName, messageID, content string) {
	if h.RoomSvc == nil || content == "" {
		return
	}

	ctx := context.Background()
	roomType, _, members, err := h.RoomSvc.GetRoomMembersForMentions(ctx, roomID)
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
