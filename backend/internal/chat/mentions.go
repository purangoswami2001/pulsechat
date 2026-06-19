package chat

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
)

var mentionPattern = regexp.MustCompile(`(?i)@(all|[a-zA-Z0-9_]+)`)

// MemberBrief holds minimal member info for mention resolution.
type MemberBrief struct {
	UserID   string
	Username string
}

// MembersStore provides room member data for mention notifications.
type MembersStore interface {
	GetRoomForMentions(ctx context.Context, roomID string) (roomType string, roomName string, members []MemberBrief, err error)
}

// GroupMentionNotification is sent when a user is @mentioned in a group.
type GroupMentionNotification struct {
	GroupID    string `json:"group_id"`
	GroupName  string `json:"group_name"`
	MessageID  string `json:"message_id"`
	SenderID   string `json:"sender_id"`
	SenderName string `json:"sender_name"`
	Preview    string `json:"preview"`
	MentionAll bool   `json:"mention_all,omitempty"`
}

// NotifyMessageMentions parses @mentions in a group message and notifies affected users.
func (h *Hub) NotifyMessageMentions(roomID, senderID, senderName, messageID, content string) {
	if h.Members == nil || content == "" {
		return
	}

	ctx := context.Background()
	roomType, roomName, members, err := h.Members.GetRoomForMentions(ctx, roomID)
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

func resolveMentionTargets(content string, members []MemberBrief, senderID string) []string {
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
