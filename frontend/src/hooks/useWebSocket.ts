import { useEffect, useRef, useState, useCallback } from 'react';

const WS_BASE_URL = import.meta.env.VITE_WS_BASE_URL || 'ws://localhost:8080';

// WebSocket event interface payload
export interface WSEvent<T = unknown> {
  type: string;
  room_id: string;
  payload: T;
}

export interface WSMessagePayload {
  id: string;
  room_id: string;
  sender_id: string;
  sender_name: string;
  sender_avatar_url?: string;
  content: string;
  attachment_url?: string;
  attachment_type?: string;
  created_at: string;
}

export interface WSTypingPayload {
  user_id: string;
  avatar_url?: string;
  username: string;
}

export interface WSPresencePayload {
  user_id: string;
  username: string;
  avatar_url?: string;
  online_users: { user_id: string; username: string; avatar_url?: string }[]; // for snapshots
}

export function useWebSocket(roomID: string | null, token: string | null, currentUserId: string | null = null) {
  const [isConnected, setIsConnected] = useState(false);
  const [socketMessages, setSocketMessages] = useState<WSMessagePayload[]>([]);
  const [typingUsers, setTypingUsers] = useState<string[]>([]);
  const [onlineUsers, setOnlineUsers] = useState<{ user_id: string; username: string; avatar_url?: string }[]>([]);
  const [socketError, setSocketError] = useState<string | null>(null);
  
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<number | null>(null);

  // Connect function
  const connect = useCallback(() => {
    if (!roomID || !token) return;

    // Close existing socket if open
    if (wsRef.current) {
      wsRef.current.close();
    }

    const wsUrl = `${WS_BASE_URL}/ws?room_id=${encodeURIComponent(roomID)}&token=${encodeURIComponent(token)}`;
    console.log('Connecting to WebSocket:', wsUrl);
    
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      console.log('WebSocket connected successfully');
      setIsConnected(true);
      // Reset states
      setSocketMessages([]);
      setTypingUsers([]);
    };

    ws.onmessage = (event) => {
      try {
        const rawData = event.data as string;
        const lines = rawData.split('\n');
        for (const line of lines) {
          if (!line.trim()) continue;
          
          const data: WSEvent = JSON.parse(line);
          console.log('Received WebSocket event:', data);

          switch (data.type) {
            case 'message.new': {
              const payload = data.payload as WSMessagePayload;
              payload.room_id = data.room_id;
              setSocketMessages((prev) => [...prev, payload]);
              break;
            }
            case 'typing.start': {
              const payload = data.payload as WSTypingPayload;
              if (currentUserId && payload.user_id === currentUserId) break;
              setTypingUsers((prev) => {
                if (prev.includes(payload.username)) return prev;
                return [...prev, payload.username];
              });
              break;
            }
            case 'typing.stop': {
              const payload = data.payload as WSTypingPayload;
              if (currentUserId && payload.user_id === currentUserId) break;
              setTypingUsers((prev) => prev.filter((u) => u !== payload.username));
              break;
            }
            case 'presence.join': {
              const payload = data.payload as WSPresencePayload;
              setOnlineUsers((prev) => {
                if (prev.some((u) => u.user_id === payload.user_id)) return prev;
                return [...prev, { user_id: payload.user_id, username: payload.username, avatar_url: payload.avatar_url }];
              });
              break;
            }
            case 'presence.leave': {
              const payload = data.payload as WSPresencePayload;
              setOnlineUsers((prev) => prev.filter((u) => u.user_id !== payload.user_id));
              setTypingUsers((prev) => prev.filter((u) => u !== payload.username));
              break;
            }
            case 'presence.snapshot': {
              const payload = data.payload as WSPresencePayload;
              setOnlineUsers(payload.online_users || []);
              break;
            }
            case 'error': {
              const errMsg = data.payload as string;
              console.error('WebSocket server error event:', errMsg);
              setSocketError(errMsg);
              setTimeout(() => {
                setSocketError(null);
              }, 4000);
              break;
            }
            default:
              console.warn('Unknown WebSocket event type:', data.type);
          }
        }
      } catch (err) {
        console.error('Failed to parse WebSocket message:', err);
      }
    };

    ws.onclose = (e) => {
      console.log('WebSocket closed:', e.reason);
      setIsConnected(false);
      // Attempt reconnection if not closed intentionally
      if (e.code !== 1000) {
        console.log('WebSocket connection lost. Reconnecting in 3 seconds...');
        reconnectTimeoutRef.current = window.setTimeout(() => {
          connect();
        }, 3000);
      }
    };

    ws.onerror = (err) => {
      console.error('WebSocket connection error:', err);
      ws.close();
    };

  }, [roomID, token, currentUserId]);

  // Handle connection hook lifecycles
  useEffect(() => {
    connect();

    return () => {
      if (wsRef.current) {
        wsRef.current.close(1000, 'Component unmounted');
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
    };
  }, [connect]);

  // Method: Send Message
  const sendMessage = useCallback((content: string, attachmentURL?: string, attachmentType?: string) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN || !roomID) {
      console.warn('Cannot send message: WebSocket is not open');
      return;
    }

    const eventPayload = {
      type: 'message.send',
      room_id: roomID,
      payload: {
        content: content,
        attachment_url: attachmentURL,
        attachment_type: attachmentType,
      },
    };

    wsRef.current.send(JSON.stringify(eventPayload));
  }, [roomID]);

  // Method: Send Typing Start
  const sendTypingStart = useCallback(() => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN || !roomID) return;

    const payload = {
      type: 'typing.start',
      room_id: roomID,
    };

    wsRef.current.send(JSON.stringify(payload));
  }, [roomID]);

  // Method: Send Typing Stop
  const sendTypingStop = useCallback(() => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN || !roomID) return;

    const payload = {
      type: 'typing.stop',
      room_id: roomID,
    };

    wsRef.current.send(JSON.stringify(payload));
  }, [roomID]);

  return {
    isConnected,
    socketMessages,
    typingUsers,
    onlineUsers,
    socketError,
    sendMessage,
    sendTypingStart,
    sendTypingStop,
  };
}
