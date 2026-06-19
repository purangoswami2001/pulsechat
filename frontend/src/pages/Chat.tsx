import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { MessageSquare, X } from 'lucide-react';

import { api, RoomResponse, UserResponse, UserSearchResult, RoomMemberResponse } from '../services/api';
import { useUserNotifications, GroupMentionNotification, DirectMessageNotification } from '../hooks/useUserNotifications';
import { useAlert } from '../contexts/AlertContext';
import UserSearch from '../components/UserSearch';
import ChatSidebar from '../components/chat/ChatSidebar';
import ChatPanel from '../components/chat/ChatPanel';
import { isGroupChat } from '../components/chat/utils';

const LAST_ROOM_KEY = 'pulsechat_last_room';

export default function Chat() {
  const navigate = useNavigate();
  const { roomId: routeRoomId } = useParams<{ roomId?: string }>();
  const token = localStorage.getItem('pulsechat_token');
  const { showAlert, showConfirm } = useAlert();

  const [rooms, setRooms] = useState<RoomResponse[]>([]);
  const [currentUser, setCurrentUser] = useState<UserResponse | null>(null);
  const [roomMembers, setRoomMembers] = useState<RoomMemberResponse[]>([]);
  const [globalOnlineUserIds, setGlobalOnlineUserIds] = useState<Set<string>>(new Set());

  const [isFetchingRooms, setIsFetchingRooms] = useState(false);
  const [isCreatingGroup, setIsCreatingGroup] = useState(false);
  const [showNewDM, setShowNewDM] = useState(false);
  const [showInviteMembers, setShowInviteMembers] = useState(false);
  const [groupName, setGroupName] = useState('');
  const [invitedMembers, setInvitedMembers] = useState<UserSearchResult[]>([]);
  const [isSavingGroup, setIsSavingGroup] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');
  const [searchQuery, setSearchQuery] = useState('');

  const activeRoomId =
    routeRoomId && rooms.some((r) => r.id === routeRoomId) ? routeRoomId : null;

  const goToRoom = useCallback(
    (roomId: string, replace = false) => {
      sessionStorage.setItem(LAST_ROOM_KEY, roomId);
      navigate(`/chat/${roomId}`, { replace });
    },
    [navigate],
  );

  const handleUnauthorized = useCallback(() => {
    localStorage.removeItem('pulsechat_token');
    localStorage.removeItem('pulsechat_user');
    navigate('/login');
  }, [navigate]);

  useEffect(() => {
    const userStr = localStorage.getItem('pulsechat_user');
    if (userStr) {
      try {
        setCurrentUser(JSON.parse(userStr));
      } catch (err) {
        console.error('Failed to parse user session info:', err);
      }
    }
  }, []);

  useEffect(() => {
    const fetchRooms = async () => {
      setIsFetchingRooms(true);
      setErrorMsg('');
      try {
        const roomsList = await api.listRooms();
        setRooms(roomsList || []);
      } catch (err: unknown) {
        if (err instanceof Error && (err.message.includes('401') || err.message.includes('Unauthorized'))) {
          handleUnauthorized();
        } else {
          setErrorMsg('Failed to load chats.');
        }
      } finally {
        setIsFetchingRooms(false);
      }
    };

    if (token) fetchRooms();
  }, [token, handleUnauthorized]);

  // Keep URL in sync: use route room if valid, otherwise latest chat
  useEffect(() => {
    if (isFetchingRooms) return;

    if (rooms.length === 0) {
      if (routeRoomId) navigate('/', { replace: true });
      return;
    }

    if (routeRoomId && rooms.some((r) => r.id === routeRoomId)) {
      sessionStorage.setItem(LAST_ROOM_KEY, routeRoomId);
      return;
    }

    const saved = sessionStorage.getItem(LAST_ROOM_KEY);
    const savedRoom = saved && rooms.find((r) => r.id === saved);
    goToRoom(savedRoom ? savedRoom.id : rooms[0].id, true);
  }, [rooms, routeRoomId, isFetchingRooms, navigate, goToRoom]);

  useEffect(() => {
    if (!activeRoomId) {
      setRoomMembers([]);
      return;
    }
    const room = rooms.find((r) => r.id === activeRoomId);
    if (!room || !isGroupChat(room)) {
      setRoomMembers([]);
      return;
    }

    const fetchMembers = async () => {
      try {
        const members = await api.listRoomMembers(activeRoomId);
        setRoomMembers(members);
      } catch (err) {
        console.error('Failed to load group members:', err);
      }
    };
    fetchMembers();
  }, [activeRoomId, rooms]);

  const refreshRooms = async () => {
    try {
      const roomsList = await api.listRooms();
      setRooms(roomsList || []);
    } catch (err) {
      console.error('Failed to refresh rooms:', err);
    }
  };

  useUserNotifications(
    token,
    activeRoomId,
    async (notification) => {
      await refreshRooms();
      goToRoom(notification.group_id);
    },
    useCallback((userId: string, status: 'online' | 'offline') => {
      setGlobalOnlineUserIds((prev) => {
        const next = new Set(prev);
        if (status === 'online') {
          next.add(userId);
        } else {
          next.delete(userId);
        }
        return next;
      });
    }, []),
    useCallback((userIds: string[]) => {
      setGlobalOnlineUserIds(new Set(userIds));
    }, []),
    useCallback(
      async (notification: GroupMentionNotification) => {
        await refreshRooms();
        goToRoom(notification.group_id);
      },
      [goToRoom],
    ),
    useCallback(
      async (notification: DirectMessageNotification) => {
        await refreshRooms();
        goToRoom(notification.room_id);
      },
      [goToRoom],
    ),
  );

  const activeRoom = rooms.find((r) => r.id === activeRoomId) ?? null;
  const isCurrentUserAdmin = roomMembers.some((m) => m.id === currentUser?.id && m.is_admin);

  const handleLogout = () => {
    localStorage.removeItem('pulsechat_token');
    localStorage.removeItem('pulsechat_user');
    sessionStorage.removeItem(LAST_ROOM_KEY);
    navigate('/login');
  };

  const handleCreateGroup = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!groupName.trim()) return;

    setIsSavingGroup(true);
    try {
      const memberIds = invitedMembers.map((m) => m.id);
      const newRoom = await api.createRoom(groupName.trim(), memberIds);
      setRooms((prev) => [newRoom, ...prev]);
      goToRoom(newRoom.id);
      setGroupName('');
      setInvitedMembers([]);
      setIsCreatingGroup(false);
      showAlert('Group created successfully!', 'success');
    } catch (err: unknown) {
      showAlert(err instanceof Error ? err.message : 'Failed to create group.', 'error');
    } finally {
      setIsSavingGroup(false);
    }
  };

  const handleStartDM = async (user: UserSearchResult) => {
    try {
      const dmRoom = await api.createDirectRoom(user.id);
      setRooms((prev) => {
        if (prev.some((r) => r.id === dmRoom.id)) return prev;
        return [dmRoom, ...prev];
      });
      goToRoom(dmRoom.id);
      setShowNewDM(false);
    } catch (err: unknown) {
      showAlert(err instanceof Error ? err.message : 'Failed to start direct chat.', 'error');
    }
  };

  const handleInviteMember = async (user: UserSearchResult) => {
    if (!activeRoomId || !isCurrentUserAdmin) return;
    try {
      const members = await api.addRoomMember(activeRoomId, { user_id: user.id });
      setRoomMembers(members);
      await refreshRooms();
      showAlert(`${user.username} has been added!`, 'success');
    } catch (err: unknown) {
      showAlert(err instanceof Error ? err.message : 'Failed to add member.', 'error');
    }
  };

  const handleRemoveMember = async (userId: string) => {
    if (!activeRoomId || !isCurrentUserAdmin) return;
    const confirmed = await showConfirm('Remove this member from the group?');
    if (!confirmed) return;
    try {
      const members = await api.removeRoomMember(activeRoomId, userId);
      setRoomMembers(members);
      await refreshRooms();
      showAlert('Member removed successfully.', 'success');
    } catch (err: unknown) {
      showAlert(err instanceof Error ? err.message : 'Failed to remove member.', 'error');
    }
  };

  const handleRoomSelect = (roomId: string) => {
    if (roomId === activeRoomId) return;
    setGlobalOnlineUserIds(new Set());
    goToRoom(roomId);
  };

  const handleDeleteGroup = async () => {
    if (!activeRoomId) return;
    const room = rooms.find((r) => r.id === activeRoomId);
    if (!room) return;

    const isGroup = isGroupChat(room);
    if (isGroup && !isCurrentUserAdmin) {
      showAlert('Only group admins can delete groups.', 'error');
      return;
    }

    const confirmed = await showConfirm(
      isGroup
        ? 'Delete this group permanently? All messages will be lost.'
        : 'Delete this chat permanently? All messages will be lost.',
    );
    if (!confirmed) return;

    const deletedId = activeRoomId;
    try {
      await api.deleteGroup(deletedId);
      setShowInviteMembers(false);
      const remaining = rooms.filter((r) => r.id !== deletedId);
      setRooms(remaining);
      if (remaining.length > 0) {
        goToRoom(remaining[0].id, true);
      } else {
        sessionStorage.removeItem(LAST_ROOM_KEY);
        navigate('/', { replace: true });
      }
      showAlert(isGroup ? 'Group deleted permanently.' : 'Chat deleted permanently.', 'success');
    } catch (err: unknown) {
      showAlert(err instanceof Error ? err.message : 'Failed to delete chat.', 'error');
    }
  };

  return (
    <div className="h-screen flex bg-gradient-to-br from-slate-50 via-white to-slate-100 dark:from-slate-950 dark:via-slate-900 dark:to-slate-950 text-slate-900 dark:text-slate-100 overflow-hidden transition-colors duration-300">
      <ChatSidebar
        rooms={rooms}
        activeRoomId={activeRoomId}
        currentUser={currentUser}
        isFetchingRooms={isFetchingRooms}
        onlineUserIds={globalOnlineUserIds}
        searchQuery={searchQuery}
        errorMsg={errorMsg}
        isCreatingGroup={isCreatingGroup}
        groupName={groupName}
        invitedMembers={invitedMembers}
        isSavingGroup={isSavingGroup}
        onSearchChange={setSearchQuery}
        onRoomSelect={handleRoomSelect}
        onToggleCreateGroup={() => {
          setIsCreatingGroup(!isCreatingGroup);
          setShowNewDM(false);
        }}
        onShowNewDM={() => {
          setShowNewDM(true);
          setIsCreatingGroup(false);
        }}
        onGroupNameChange={setGroupName}
        onInvitedMembersChange={setInvitedMembers}
        onCreateGroup={handleCreateGroup}
        onCancelCreateGroup={() => {
          setIsCreatingGroup(false);
          setGroupName('');
          setInvitedMembers([]);
        }}
        onNavigateSettings={() => navigate('/settings')}
        onLogout={handleLogout}
      />

      <main className="flex-1 flex min-w-0 overflow-hidden">
        {activeRoom ? (
          <ChatPanel
            key={activeRoom.id}
            room={activeRoom}
            currentUser={currentUser}
            token={token}
            roomMembers={roomMembers}
            globalOnlineUserIds={globalOnlineUserIds}
            onDeleteGroup={handleDeleteGroup}
            onOpenInvite={() => setShowInviteMembers(true)}
            onRemoveMember={handleRemoveMember}
          />
        ) : (
          <div className="flex-1 flex flex-col items-center justify-center text-slate-400 animate-fade-in bg-white/60 dark:bg-slate-950/60">
            <div className="p-4 bg-accent-muted text-accent rounded-3xl mb-4">
              <MessageSquare className="w-10 h-10" />
            </div>
            <h3 className="text-lg font-bold text-slate-700 dark:text-slate-300 mb-1">Welcome to PulseChat</h3>
            <p className="text-sm text-slate-400">Start a direct chat or create a group</p>
          </div>
        )}
      </main>

      {showNewDM && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/40 backdrop-blur-sm animate-fade-in">
          <div className="w-full max-w-md glass rounded-2xl shadow-2xl p-5 animate-scale-in">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-bold text-slate-800 dark:text-slate-200">New Direct Chat</h3>
              <button
                type="button"
                onClick={() => setShowNewDM(false)}
                className="p-1.5 hover:bg-slate-200 dark:hover:bg-slate-700 rounded-lg text-slate-400"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
            <UserSearch
              onSelect={handleStartDM}
              excludeIds={currentUser?.id ? [currentUser.id] : []}
              placeholder="Search user by username or email..."
            />
          </div>
        </div>
      )}

      {showInviteMembers && activeRoom && isGroupChat(activeRoom) && isCurrentUserAdmin && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/40 backdrop-blur-sm animate-fade-in">
          <div className="w-full max-w-md glass rounded-2xl shadow-2xl p-5 animate-scale-in">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-bold text-slate-800 dark:text-slate-200">
                Add members to {activeRoom.name}
              </h3>
              <button
                type="button"
                onClick={() => setShowInviteMembers(false)}
                className="p-1.5 hover:bg-slate-200 dark:hover:bg-slate-700 rounded-lg text-slate-400"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
            <UserSearch
              onSelect={handleInviteMember}
              excludeIds={[
                ...(currentUser?.id ? [currentUser.id] : []),
                ...roomMembers.map((m) => m.id),
              ]}
              placeholder="Search by username or email..."
            />
            <p className="text-[11px] text-slate-400 mt-3 text-center">Select a user to add them to the group</p>
          </div>
        </div>
      )}
    </div>
  );
}
