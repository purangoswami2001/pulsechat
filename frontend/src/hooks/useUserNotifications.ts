import { useEffect, useRef, useCallback } from 'react';

const WS_BASE_URL = import.meta.env.VITE_WS_BASE_URL || 'ws://localhost:8080';
const SEEN_LIMIT = 200;

export interface GroupInviteNotification {
  group_id: string;
  group_name: string;
  inviter_id: string;
  inviter_name: string;
}

export interface GroupMentionNotification {
  group_id: string;
  group_name: string;
  message_id: string;
  sender_id: string;
  sender_name: string;
  preview: string;
  mention_all?: boolean;
}

export interface DirectMessageNotification {
  room_id: string;
  message_id: string;
  sender_id: string;
  sender_name: string;
  preview: string;
}

interface WSEvent<T = unknown> {
  type: string;
  room_id: string;
  payload: T;
}

function notificationKey(type: string, payload: Record<string, unknown>): string {
  if (type === 'notification.group.mention') {
    return `mention:${payload.message_id}:${payload.group_id}`;
  }
  if (type === 'notification.group.invite') {
    return `invite:${payload.group_id}:${payload.inviter_id}`;
  }
  if (type === 'notification.direct.message') {
    return `direct:${payload.message_id}:${payload.room_id}`;
  }
  return `${type}:${JSON.stringify(payload)}`;
}

function showBrowserNotification(title: string, body: string, onClick?: () => void) {
  if (!('Notification' in window)) return;
  if (Notification.permission === 'granted') {
    const n = new Notification(title, { body, icon: '/favicon.ico' });
    if (onClick) {
      n.onclick = () => {
        onClick();
        window.focus();
        n.close();
      };
    }
  }
}

export function useUserNotifications(
  token: string | null,
  activeRoomId: string | null,
  onGroupInvite?: (notification: GroupInviteNotification) => void,
  onGlobalPresenceUpdate?: (userId: string, status: 'online' | 'offline') => void,
  onGlobalPresenceSnapshot?: (userIds: string[]) => void,
  onGroupMention?: (notification: GroupMentionNotification) => void,
  onDirectMessage?: (notification: DirectMessageNotification) => void,
) {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectRef = useRef<number | null>(null);
  const seenNotificationsRef = useRef<Set<string>>(new Set());
  const activeRoomRef = useRef(activeRoomId);
  activeRoomRef.current = activeRoomId;

  const onInviteRef = useRef(onGroupInvite);
  onInviteRef.current = onGroupInvite;

  const onUpdateRef = useRef(onGlobalPresenceUpdate);
  onUpdateRef.current = onGlobalPresenceUpdate;

  const onSnapshotRef = useRef(onGlobalPresenceSnapshot);
  onSnapshotRef.current = onGlobalPresenceSnapshot;

  const onMentionRef = useRef(onGroupMention);
  onMentionRef.current = onGroupMention;

  const onDirectMessageRef = useRef(onDirectMessage);
  onDirectMessageRef.current = onDirectMessage;

  const shouldNotify = useCallback((key: string) => {
    const seen = seenNotificationsRef.current;
    if (seen.has(key)) return false;
    seen.add(key);
    if (seen.size > SEEN_LIMIT) {
      const first = seen.values().next().value;
      if (first) seen.delete(first);
    }
    return true;
  }, []);

  const connect = useCallback(() => {
    if (!token) return;

    const existing = wsRef.current;
    if (
      existing &&
      (existing.readyState === WebSocket.OPEN || existing.readyState === WebSocket.CONNECTING)
    ) {
      return;
    }

    if (existing) {
      existing.close();
    }

    const wsUrl = `${WS_BASE_URL}/ws?notify=1&token=${encodeURIComponent(token)}`;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      if (reconnectRef.current) {
        clearTimeout(reconnectRef.current);
        reconnectRef.current = null;
      }
    };

    ws.onmessage = (event) => {
      try {
        const lines = (event.data as string).split('\n');
        for (const line of lines) {
          if (!line.trim()) continue;
          const data: WSEvent = JSON.parse(line);

          if (data.type === 'notification.group.invite') {
            const payload = data.payload as GroupInviteNotification;
            const key = notificationKey(data.type, payload as unknown as Record<string, unknown>);
            if (!shouldNotify(key)) continue;

            const viewingRoom =
              activeRoomRef.current === payload.group_id || activeRoomRef.current === data.room_id;
            if (!viewingRoom) {
              showBrowserNotification(
                `Added to ${payload.group_name}`,
                `${payload.inviter_name} added you to the group`,
                () => onInviteRef.current?.(payload),
              );
            }
            onInviteRef.current?.(payload);
          } else if (data.type === 'notification.group.mention') {
            const payload = data.payload as GroupMentionNotification;
            const key = notificationKey(data.type, payload as unknown as Record<string, unknown>);
            if (!shouldNotify(key)) continue;

            const viewingRoom =
              activeRoomRef.current === payload.group_id || activeRoomRef.current === data.room_id;
            if (!viewingRoom) {
              const title = payload.mention_all
                ? `@all in ${payload.group_name}`
                : `Mentioned in ${payload.group_name}`;
              showBrowserNotification(
                title,
                `${payload.sender_name}: ${payload.preview}`,
                () => onMentionRef.current?.(payload),
              );
            }
            onMentionRef.current?.(payload);
          } else if (data.type === 'notification.direct.message') {
            const payload = data.payload as DirectMessageNotification;
            const key = notificationKey(data.type, payload as unknown as Record<string, unknown>);
            if (!shouldNotify(key)) continue;

            const viewingRoom =
              activeRoomRef.current === payload.room_id || activeRoomRef.current === data.room_id;
            if (!viewingRoom) {
              showBrowserNotification(
                payload.sender_name,
                payload.preview,
                () => onDirectMessageRef.current?.(payload),
              );
            }
            onDirectMessageRef.current?.(payload);
          } else if (data.type === 'presence.global_snapshot') {
            const payload = data.payload as { online_user_ids: string[] };
            onSnapshotRef.current?.(payload.online_user_ids || []);
          } else if (data.type === 'presence.global_update') {
            const payload = data.payload as { user_id: string; status: 'online' | 'offline' };
            onUpdateRef.current?.(payload.user_id, payload.status);
          }
        }
      } catch (err) {
        console.error('Failed to parse notification:', err);
      }
    };

    ws.onclose = (e) => {
      wsRef.current = null;
      if (e.code !== 1000) {
        reconnectRef.current = window.setTimeout(connect, 3000);
      }
    };

    ws.onerror = () => ws.close();
  }, [token, shouldNotify]);

  useEffect(() => {
    if ('Notification' in window && Notification.permission === 'default') {
      Notification.requestPermission().catch(() => {});
    }

    connect();

    return () => {
      if (reconnectRef.current) clearTimeout(reconnectRef.current);
      if (wsRef.current) {
        wsRef.current.close(1000, 'unmount');
        wsRef.current = null;
      }
    };
  }, [connect]);
}
