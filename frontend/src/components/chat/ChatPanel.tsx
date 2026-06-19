import React, { useState, useEffect, useRef } from 'react';
import {
  MessageCircle,
  Users,
  Send,
  Paperclip,
  Loader2,
  AlertCircle,
  Smile,
  X,
  Download,
  FileText,
  Image as ImageIcon,
  Video,
  UserPlus,
  MoreVertical,
  Trash2,
} from 'lucide-react';

import {
  api,
  RoomResponse,
  MessageResponse,
  UserResponse,
  RoomMemberResponse,
  getFileURL,
} from '../../services/api';
import { useWebSocket } from '../../hooks/useWebSocket';
import { useAlert } from '../../contexts/AlertContext';
import EmojiPicker from '../EmojiPicker';
import GroupMembersPanel from './GroupMembersPanel';
import MediaLightbox from './MediaLightbox';
import MessageContent from './MessageContent';
import MentionPicker from './MentionPicker';
import { getDateLabel, getRoomLabel, isDirectChat, isGroupChat } from './utils';

// Helper to determine if a string is a single emoji
const isSingleEmoji = (str: string): boolean => {
  const trimmed = str.trim();
  if (!trimmed) return false;
  try {
    if (typeof (Intl as any).Segmenter === 'function') {
      const segmenter = new (Intl as any).Segmenter(undefined, { granularity: 'grapheme' });
      const segments = Array.from(segmenter.segment(trimmed)) as any[];
      if (segments.length === 1) {
        const char = segments[0].segment;
        const hasEmoji = /\p{Extended_Pictographic}/u.test(char) || /\p{Emoji_Presentation}/u.test(char);
        const isPlainText = /^[a-zA-Z0-9\s,\.\?!@#\$%\^&\*\(\)\-_\+=\{\}\[\]\\|;:'"<>~/`]+$/.test(char);
        return hasEmoji && !isPlainText;
      }
    }
  } catch {
    const simpleEmojiRegex = /^(\p{Emoji_Presentation}|\p{Emoji}\uFE0F)(?:\p{Emoji_Modifier})?$/u;
    return simpleEmojiRegex.test(trimmed);
  }
  return false;
};

interface ChatPanelProps {
  room: RoomResponse;
  currentUser: UserResponse | null;
  token: string | null;
  roomMembers: RoomMemberResponse[];
  globalOnlineUserIds: Set<string>;
  onDeleteGroup: () => void;
  onOpenInvite: () => void;
  onRemoveMember: (userId: string) => void;
  onOnlineUsersChange?: (userIds: string[]) => void;
}

export default function ChatPanel({
  room,
  currentUser,
  token,
  roomMembers,
  globalOnlineUserIds,
  onDeleteGroup,
  onOpenInvite,
  onRemoveMember,
  onOnlineUsersChange,
}: ChatPanelProps) {
  const { showAlert } = useAlert();
  const [messages, setMessages] = useState<MessageResponse[]>([]);
  const [isFetchingMessages, setIsFetchingMessages] = useState(true);
  const [inputText, setInputText] = useState('');
  const [showMembersPanel, setShowMembersPanel] = useState(false);
  const [showGroupMenu, setShowGroupMenu] = useState(false);
  const [showEmojiPicker, setShowEmojiPicker] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [pendingAttachment, setPendingAttachment] = useState<{ url: string; type: string; name: string } | null>(null);
  const [lightboxMedia, setLightboxMedia] = useState<{ url: string; type: 'image' | 'video' } | null>(null);
  const [mentionOpen, setMentionOpen] = useState(false);
  const [mentionQuery, setMentionQuery] = useState('');
  const [mentionStart, setMentionStart] = useState(0);

  const messagesEndRef = useRef<HTMLDivElement | null>(null);
  const typingTimeoutRef = useRef<number | null>(null);
  const isTypingRef = useRef(false);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const inputRef = useRef<HTMLTextAreaElement | null>(null);
  const emojiPickerRef = useRef<HTMLDivElement | null>(null);
  const groupMenuRef = useRef<HTMLDivElement | null>(null);

  const {
    socketMessages,
    typingUsers,
    onlineUsers,
    socketError,
    sendMessage: wsSendMessage,
    sendTypingStart,
    sendTypingStop,
  } = useWebSocket(room.id, token, currentUser?.id ?? null);

  const isCurrentUserAdmin = roomMembers.some((m) => m.id === currentUser?.id && m.is_admin);
  const roomOnlineUserIds = new Set(onlineUsers.map((u) => u.user_id));
  const otherUserOnline =
    isDirectChat(room) && room.other_user_id ? globalOnlineUserIds.has(room.other_user_id) : false;

  useEffect(() => {
    onOnlineUsersChange?.(onlineUsers.map((u) => u.user_id));
  }, [onlineUsers, onOnlineUsersChange]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  useEffect(() => {
    const el = inputRef.current;
    if (!el) return;
    el.style.height = 'auto';
    el.style.height = `${Math.min(el.scrollHeight, 128)}px`;
  }, [inputText]);

  useEffect(() => {
    setShowMembersPanel(false);
    setShowGroupMenu(false);
    setShowEmojiPicker(false);
    setInputText('');
    setPendingAttachment(null);
    isTypingRef.current = false;
  }, [room.id]);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (emojiPickerRef.current && !emojiPickerRef.current.contains(e.target as Node)) {
        setShowEmojiPicker(false);
      }
      if (groupMenuRef.current && !groupMenuRef.current.contains(e.target as Node)) {
        setShowGroupMenu(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  useEffect(() => {
    let cancelled = false;

    const fetchMessages = async () => {
      setIsFetchingMessages(true);
      setMessages([]);
      try {
        const history = await api.listMessages(room.id);
        if (!cancelled) setMessages(history);
      } catch (err) {
        console.error('Failed to sync messages history:', err);
      } finally {
        if (!cancelled) setIsFetchingMessages(false);
      }
    };

    fetchMessages();
    return () => {
      cancelled = true;
    };
  }, [room.id]);

  useEffect(() => {
    if (socketMessages.length === 0) return;
    const latestMsg = socketMessages[socketMessages.length - 1];
    if (latestMsg.room_id !== room.id) return;

    setMessages((prev) => {
      if (prev.some((m) => m.id === latestMsg.id)) return prev;
      const mapped: MessageResponse = {
        id: latestMsg.id,
        room_id: latestMsg.room_id,
        sender_id: latestMsg.sender_id,
        sender_name: latestMsg.sender_name,
        sender_avatar_url: latestMsg.sender_avatar_url,
        content: latestMsg.content,
        attachment_url: latestMsg.attachment_url,
        attachment_type: latestMsg.attachment_type,
        created_at: latestMsg.created_at,
      };
      return [...prev, mapped];
    });
  }, [socketMessages, room.id]);

  useEffect(() => {
    return () => {
      if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current);
      if (isTypingRef.current) {
        sendTypingStop();
        isTypingRef.current = false;
      }
    };
  }, [room.id, sendTypingStop]);

  const stopTyping = () => {
    if (isTypingRef.current) {
      isTypingRef.current = false;
      sendTypingStop();
    }
    if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current);
  };

  const notifyTyping = (value: string) => {
    if (value.trim() !== '') {
      if (!isTypingRef.current) {
        isTypingRef.current = true;
        sendTypingStart();
      }
      if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current);
      typingTimeoutRef.current = window.setTimeout(() => {
        isTypingRef.current = false;
        sendTypingStop();
      }, 3000);
    } else {
      stopTyping();
    }
  };

  const handleSendMessage = (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputText.trim() && !pendingAttachment) return;
    wsSendMessage(inputText.trim(), pendingAttachment?.url, pendingAttachment?.type);
    stopTyping();
    setInputText('');
    setPendingAttachment(null);
    setShowEmojiPicker(false);
    setMentionOpen(false);
  };

  const handleInputKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (!inputText.trim() && !pendingAttachment) return;
      wsSendMessage(inputText.trim(), pendingAttachment?.url, pendingAttachment?.type);
      stopTyping();
      setInputText('');
      setPendingAttachment(null);
      setShowEmojiPicker(false);
      setMentionOpen(false);
    }
  };

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (file.size > 50 * 1024 * 1024) {
      showAlert('File must be under 50MB', 'error');
      return;
    }
    setIsUploading(true);
    try {
      const result = await api.upload(file);
      setPendingAttachment({ url: result.url, type: result.type, name: result.name });
    } catch (err: unknown) {
      showAlert(err instanceof Error ? err.message : 'Failed to upload file', 'error');
    } finally {
      setIsUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = '';
    }
  };

  const handleEmojiSelect = (emoji: string) => {
    setInputText((prev) => {
      const next = prev + emoji;
      notifyTyping(next);
      return next;
    });
    inputRef.current?.focus();
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const value = e.target.value;
    setInputText(value);
    notifyTyping(value);

    if (!isGroupChat(room)) {
      setMentionOpen(false);
      return;
    }

    const cursor = e.target.selectionStart ?? value.length;
    const before = value.slice(0, cursor);
    const match = before.match(/@(\w*)$/);
    if (match) {
      setMentionOpen(true);
      setMentionQuery(match[1]);
      setMentionStart(cursor - match[0].length);
    } else {
      setMentionOpen(false);
    }
  };

  const handleMentionSelect = (username: string) => {
    const cursor = inputRef.current?.selectionStart ?? inputText.length;
    const before = inputText.slice(0, mentionStart);
    const after = inputText.slice(cursor);
    const next = `${before}@${username} ${after}`;
    setInputText(next);
    setMentionOpen(false);
    notifyTyping(next);
    requestAnimationFrame(() => {
      const pos = mentionStart + username.length + 2;
      inputRef.current?.setSelectionRange(pos, pos);
      inputRef.current?.focus();
    });
  };

  const remoteTypingUsers = typingUsers.filter((u) => u !== currentUser?.username);
  const roomLabel = getRoomLabel(room);

  const renderAttachment = (attachmentUrl: string, attachmentType: string) => {
    const fullUrl = getFileURL(attachmentUrl);
    const isImage = attachmentType.startsWith('image/');
    const isVideo = attachmentType.startsWith('video/');

    if (isImage) {
      return (
        <button
          type="button"
          onClick={() => setLightboxMedia({ url: attachmentUrl, type: 'image' })}
          className="block mt-2 text-left"
        >
          <img
            src={fullUrl}
            alt="Attachment"
            className="max-w-xs max-h-60 rounded-xl object-cover border border-slate-200/50 dark:border-slate-700/50 hover:opacity-90 transition-opacity cursor-pointer"
          />
        </button>
      );
    }

    if (isVideo) {
      return (
        <button
          type="button"
          onClick={() => setLightboxMedia({ url: attachmentUrl, type: 'video' })}
          className="block mt-2 max-w-xs rounded-xl overflow-hidden border border-slate-200/50 dark:border-slate-700/50 bg-black relative group text-left"
        >
          <video
            src={fullUrl}
            preload="metadata"
            className="w-full max-h-60 object-contain pointer-events-none"
          />
          <div className="absolute inset-0 flex items-center justify-center bg-black/30 group-hover:bg-black/40 transition-colors">
            <Video className="w-10 h-10 text-white drop-shadow-lg" />
          </div>
        </button>
      );
    }

    const filename = attachmentUrl.split('/').pop() || 'file';
    return (
      <a
        href={fullUrl}
        target="_blank"
        rel="noopener noreferrer"
        className="flex items-center gap-2 mt-2 px-3 py-2 bg-slate-200/50 dark:bg-slate-700/30 rounded-xl hover:bg-slate-300/50 dark:hover:bg-slate-600/30 transition-colors"
      >
        <FileText className="w-4 h-4 text-slate-500 shrink-0" />
        <span className="text-xs text-slate-600 dark:text-slate-300 truncate">{filename}</span>
        <Download className="w-3.5 h-3.5 text-slate-400 shrink-0 ml-auto" />
      </a>
    );
  };

  return (
    <div className="flex flex-1 min-w-0 h-full">
      <div className="flex flex-col flex-1 min-w-0 bg-white/60 dark:bg-slate-950/60 backdrop-blur-sm transition-colors duration-300">
        <header className="h-14 border-b border-slate-200/80 dark:border-slate-800/80 px-5 flex items-center justify-between shrink-0 glass-subtle">
          <div className="flex items-center gap-3 min-w-0">
            {isDirectChat(room) ? (
              <>
                <MessageCircle className="w-5 h-5 text-accent shrink-0" />
                <div className="min-w-0">
                  <h2 className="font-bold text-slate-800 dark:text-slate-200 text-base truncate">{roomLabel}</h2>
                  <p className={`text-[11px] font-medium ${otherUserOnline ? 'text-emerald-500' : 'text-slate-400'}`}>
                    {otherUserOnline ? 'Online' : 'Offline'}
                  </p>
                </div>
              </>
            ) : (
              <>
                <Users className="w-5 h-5 text-accent shrink-0" />
                <h2 className="font-bold text-slate-800 dark:text-slate-200 text-base truncate">{roomLabel}</h2>
                <span className="text-[11px] text-slate-500 bg-slate-100 dark:bg-slate-800 px-2 py-0.5 rounded-full font-medium shrink-0">
                  Group
                </span>
              </>
            )}
          </div>

          {(isGroupChat(room) || isDirectChat(room)) && (
            <div className="flex items-center gap-1 shrink-0">
              {isGroupChat(room) && (
                <button
                  type="button"
                  onClick={() => {
                    setShowMembersPanel((v) => !v);
                    setShowGroupMenu(false);
                  }}
                  className={`p-2 rounded-lg transition-all duration-200 ${
                    showMembersPanel
                      ? 'bg-accent-muted text-accent'
                      : 'text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800'
                  }`}
                  title="View members"
                >
                  <Users className="w-5 h-5" />
                </button>
              )}

              {((isGroupChat(room) && isCurrentUserAdmin) || isDirectChat(room)) && (
                <div className="relative" ref={groupMenuRef}>
                  <button
                    type="button"
                    onClick={() => setShowGroupMenu((v) => !v)}
                    className={`p-2 rounded-lg transition-all duration-200 ${
                      showGroupMenu
                        ? 'bg-accent-muted text-accent'
                        : 'text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800'
                    }`}
                    title="Chat options"
                  >
                    <MoreVertical className="w-5 h-5" />
                  </button>

                  {showGroupMenu && (
                    <div className="absolute right-0 top-full mt-1 w-48 glass rounded-xl shadow-xl border border-slate-200/60 dark:border-slate-700/60 py-1 z-50 animate-scale-in">
                      {isGroupChat(room) && isCurrentUserAdmin && (
                        <button
                          type="button"
                          onClick={() => {
                            onOpenInvite();
                            setShowGroupMenu(false);
                          }}
                          className="w-full flex items-center gap-2 px-3 py-2 text-sm text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800/60 transition-colors"
                        >
                          <UserPlus className="w-4 h-4 text-accent" />
                          Add members
                        </button>
                      )}
                      <button
                        type="button"
                        onClick={() => {
                          setShowGroupMenu(false);
                          setShowMembersPanel(false);
                          onDeleteGroup();
                        }}
                        className="w-full flex items-center gap-2 px-3 py-2 text-sm text-rose-600 dark:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-500/10 transition-colors"
                      >
                        <Trash2 className="w-4 h-4" />
                        {isGroupChat(room) ? 'Delete group' : 'Delete chat'}
                      </button>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
        </header>

        <div className="flex-1 overflow-y-auto px-6 py-4 min-h-0">
          {isFetchingMessages ? (
            <div className="flex flex-col items-center justify-center h-full text-slate-400 gap-3">
              <Loader2 className="w-6 h-6 animate-spin text-accent" />
              <span className="text-sm">Loading messages...</span>
            </div>
          ) : (
            <>
              <div className="text-center py-4 mb-2">
                {!isDirectChat(room) && (
                  <div className="inline-flex items-center justify-center p-2.5 bg-accent-muted text-accent rounded-2xl mb-2">
                    <Users className="w-5 h-5" />
                  </div>
                )}
                <h3 className="text-base font-bold text-slate-800 dark:text-slate-200">
                  {isDirectChat(room) ? `Direct chat with ${roomLabel}` : `Welcome to ${roomLabel}`}
                </h3>
                <p className="text-xs text-slate-400 mt-0.5">
                  {isDirectChat(room)
                    ? 'This is the beginning of your conversation.'
                    : 'Only group members can see and send messages here.'}
                </p>
              </div>

              {messages.map((msg, idx) => {
                const isMe = msg.sender_id === currentUser?.id;
                const senderInitial = msg.sender_name ? msg.sender_name.slice(0, 2) : 'U';
                const timestamp = new Date(msg.created_at).toLocaleTimeString([], {
                  hour: '2-digit',
                  minute: '2-digit',
                });
                const showDateDivider =
                  idx === 0 || getDateLabel(msg.created_at) !== getDateLabel(messages[idx - 1].created_at);
                const avatarUrl =
                  msg.sender_avatar_url ||
                  (isMe ? currentUser?.avatar_url : isDirectChat(room) ? room.other_user_avatar_url : null);
                const showOnlineOnAvatar =
                  !isMe && isGroupChat(room) && roomOnlineUserIds.has(msg.sender_id);
                const isSingleEmo = msg.content ? isSingleEmoji(msg.content) : false;

                return (
                  <React.Fragment key={msg.id}>
                    {showDateDivider && (
                      <div className="flex items-center gap-3 my-6">
                        <div className="flex-1 h-px bg-slate-200 dark:bg-slate-800" />
                        <span className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider px-2">
                          {getDateLabel(msg.created_at)}
                        </span>
                        <div className="flex-1 h-px bg-slate-200 dark:bg-slate-800" />
                      </div>
                    )}

                    <div
                      className={`flex gap-3 mb-4 max-w-2xl ${isMe ? 'ml-auto flex-row-reverse animate-slide-in-right' : 'animate-slide-in-left'}`}
                    >
                      <div className="relative shrink-0 self-start">
                        {avatarUrl ? (
                          <img
                            src={getFileURL(avatarUrl)}
                            alt={msg.sender_name || 'User'}
                            className="w-9 h-9 rounded-full object-cover shadow-sm border border-slate-200 dark:border-slate-700"
                          />
                        ) : (
                          <div
                            className={`w-9 h-9 rounded-full flex items-center justify-center font-bold text-xs uppercase shadow-sm ${
                              isMe
                                ? 'bg-accent-gradient text-white'
                                : 'bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-slate-700'
                            }`}
                          >
                            {senderInitial}
                          </div>
                        )}
                        {showOnlineOnAvatar && (
                          <div className="absolute -bottom-0.5 -right-0.5 w-3 h-3 bg-emerald-500 rounded-full border-2 border-white dark:border-slate-900" />
                        )}
                      </div>

                      <div className="space-y-1 min-w-0">
                        <div className={`flex items-center gap-2 text-xs ${isMe ? 'justify-end' : ''}`}>
                          <span className="font-semibold text-slate-700 dark:text-slate-300">
                            {isMe ? 'You' : msg.sender_name || 'User'}
                          </span>
                          <span className="text-slate-400">{timestamp}</span>
                        </div>

                        <div
                          className={
                            isSingleEmo
                              ? `text-6xl py-1 select-all transition-all duration-200 ${isMe ? 'text-right' : 'text-left'}`
                              : `px-4 py-2.5 rounded-2xl text-sm leading-relaxed transition-all duration-200 ${
                                  isMe
                                    ? 'bg-accent-gradient text-white rounded-tr-md shadow-accent'
                                    : 'glass text-slate-800 dark:text-slate-200 rounded-tl-md shadow-sm'
                                }`
                          }
                        >
                          {msg.content && (
                            isSingleEmo ? (
                              <p className="leading-none inline-block">{msg.content}</p>
                            ) : (
                              <MessageContent
                                content={msg.content}
                                members={isGroupChat(room) ? roomMembers : []}
                                isMe={isMe}
                              />
                            )
                          )}
                          {msg.attachment_url &&
                            msg.attachment_type &&
                            renderAttachment(msg.attachment_url, msg.attachment_type)}
                        </div>
                      </div>
                    </div>
                  </React.Fragment>
                );
              })}
              <div ref={messagesEndRef} />
            </>
          )}
        </div>

        <div className="px-5 pb-4 pt-2 border-t border-slate-200/60 dark:border-slate-800/40 glass-subtle shrink-0">
          {socketError && (
            <div className="mb-3 p-2.5 bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/20 rounded-xl text-rose-600 dark:text-rose-400 text-xs flex items-center gap-2 animate-fade-in">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{socketError}</span>
            </div>
          )}

          {pendingAttachment && (
            <div className="mb-2 flex items-center gap-2 p-2 bg-slate-100 dark:bg-slate-800/50 rounded-xl animate-fade-in">
              {pendingAttachment.type.startsWith('image/') ? (
                <ImageIcon className="w-4 h-4 text-accent shrink-0" />
              ) : pendingAttachment.type.startsWith('video/') ? (
                <Video className="w-4 h-4 text-accent shrink-0" />
              ) : (
                <FileText className="w-4 h-4 text-accent shrink-0" />
              )}
              <span className="text-xs text-slate-600 dark:text-slate-300 truncate flex-1">{pendingAttachment.name}</span>
              <button
                onClick={() => setPendingAttachment(null)}
                className="p-1 hover:bg-slate-200 dark:hover:bg-slate-700 rounded-md transition-colors"
              >
                <X className="w-3.5 h-3.5 text-slate-400" />
              </button>
            </div>
          )}

          <div className="h-5 flex items-center text-xs text-slate-400 mb-1.5 px-1">
            {remoteTypingUsers.length > 0 && (
              <div className="flex items-center gap-2 animate-fade-in">
                <div className="flex gap-1">
                  <span className="w-1.5 h-1.5 bg-accent rounded-full animate-typing-dot typing-dot-1" />
                  <span className="w-1.5 h-1.5 bg-accent rounded-full animate-typing-dot typing-dot-2" />
                  <span className="w-1.5 h-1.5 bg-accent rounded-full animate-typing-dot typing-dot-3" />
                </div>
                <span>
                  <strong className="text-slate-600 dark:text-slate-300">{remoteTypingUsers.join(', ')}</strong>{' '}
                  {remoteTypingUsers.length === 1 ? 'is' : 'are'} typing...
                </span>
              </div>
            )}
          </div>

          <form onSubmit={handleSendMessage} className="flex gap-2.5 items-end">
            <div className="flex-1 glass rounded-2xl px-4 py-3 flex items-end gap-3 focus-within:ring-2 focus-within:ring-accent/25 focus-within:border-accent/40 transition-all duration-200 shadow-sm min-w-0 relative">
              <button
                type="button"
                onClick={() => fileInputRef.current?.click()}
                disabled={isUploading}
                className="text-slate-400 hover:text-accent transition-colors mb-0.5 shrink-0"
              >
                {isUploading ? <Loader2 className="w-5 h-5 animate-spin" /> : <Paperclip className="w-5 h-5" />}
              </button>
              <input
                ref={fileInputRef}
                type="file"
                onChange={handleFileSelect}
                className="hidden"
                accept="image/*,video/*,.pdf,.txt,.zip"
              />

              <div className="relative flex-1 min-w-0">
                <textarea
                  ref={inputRef}
                  value={inputText}
                  onChange={handleInputChange}
                  onKeyDown={handleInputKeyDown}
                  placeholder={isGroupChat(room) ? `Message ${roomLabel}... (@ to mention)` : `Message ${roomLabel}...`}
                  rows={1}
                  className="w-full bg-transparent text-sm text-slate-800 dark:text-slate-100 focus:outline-none placeholder-slate-400 dark:placeholder-slate-500 resize-none max-h-32 min-h-[24px] leading-relaxed"
                />

                {mentionOpen && isGroupChat(room) && (
                  <MentionPicker
                    members={roomMembers}
                    currentUserId={currentUser?.id}
                    query={mentionQuery}
                    onSelect={handleMentionSelect}
                  />
                )}
              </div>

              <div className="relative shrink-0 mb-0.5" ref={emojiPickerRef}>
                <button
                  type="button"
                  onClick={() => setShowEmojiPicker(!showEmojiPicker)}
                  className={`p-1 rounded-lg transition-colors ${showEmojiPicker ? 'text-accent bg-accent-muted' : 'text-slate-400 hover:text-accent hover:bg-accent-muted/50'}`}
                >
                  <Smile className="w-5 h-5" />
                </button>
                {showEmojiPicker && (
                  <EmojiPicker onEmojiSelect={handleEmojiSelect} onClose={() => setShowEmojiPicker(false)} />
                )}
              </div>
            </div>

            <button
              type="submit"
              disabled={!inputText.trim() && !pendingAttachment}
              className="bg-accent-gradient text-white p-3.5 rounded-2xl transition-all duration-200 shadow-accent hover:opacity-90 hover:scale-[1.02] active:scale-[0.98] shrink-0 disabled:opacity-30 disabled:shadow-none disabled:cursor-not-allowed disabled:hover:scale-100"
            >
              <Send className="w-5 h-5" />
            </button>
          </form>
        </div>
      </div>

      {showMembersPanel && isGroupChat(room) && (
        <GroupMembersPanel
          members={roomMembers}
          currentUserId={currentUser?.id}
          isAdmin={isCurrentUserAdmin}
          onlineUserIds={roomOnlineUserIds}
          onClose={() => setShowMembersPanel(false)}
          onAddMembers={() => {
            onOpenInvite();
            setShowMembersPanel(false);
          }}
          onRemoveMember={onRemoveMember}
        />
      )}

      {lightboxMedia && (
        <MediaLightbox
          url={lightboxMedia.url}
          type={lightboxMedia.type}
          onClose={() => setLightboxMedia(null)}
        />
      )}
    </div>
  );
}
