import React from 'react';
import {
  MessageSquare,
  Users,
  LogOut,
  Plus,
  Loader2,
  AlertCircle,
  Search,
  Settings,
} from 'lucide-react';
import { RoomResponse, UserResponse, UserSearchResult, getFileURL } from '../../services/api';
import UserSearch from '../UserSearch';
import UserAvatar from './UserAvatar';
import { getRoomLabel, isDirectChat, isGroupChat } from './utils';

interface ChatSidebarProps {
  rooms: RoomResponse[];
  activeRoomId: string | null;
  currentUser: UserResponse | null;
  isFetchingRooms: boolean;
  onlineUserIds: Set<string>;
  searchQuery: string;
  errorMsg: string;
  isCreatingGroup: boolean;
  groupName: string;
  invitedMembers: UserSearchResult[];
  isSavingGroup: boolean;
  onSearchChange: (query: string) => void;
  onRoomSelect: (roomId: string) => void;
  onToggleCreateGroup: () => void;
  onShowNewDM: () => void;
  onGroupNameChange: (name: string) => void;
  onInvitedMembersChange: (members: UserSearchResult[]) => void;
  onCreateGroup: (e: React.FormEvent) => void;
  onCancelCreateGroup: () => void;
  onNavigateSettings: () => void;
  onLogout: () => void;
}

export default function ChatSidebar({
  rooms,
  activeRoomId,
  currentUser,
  isFetchingRooms,
  onlineUserIds,
  searchQuery,
  errorMsg,
  isCreatingGroup,
  groupName,
  invitedMembers,
  isSavingGroup,
  onSearchChange,
  onRoomSelect,
  onToggleCreateGroup,
  onShowNewDM,
  onGroupNameChange,
  onInvitedMembersChange,
  onCreateGroup,
  onCancelCreateGroup,
  onNavigateSettings,
  onLogout,
}: ChatSidebarProps) {
  const matchesSearch = (room: RoomResponse) => {
    const label = getRoomLabel(room).toLowerCase();
    return label.includes(searchQuery.toLowerCase()) || room.name.toLowerCase().includes(searchQuery.toLowerCase());
  };

  const directMessages = rooms.filter((r) => isDirectChat(r) && matchesSearch(r));
  const groups = rooms.filter((r) => isGroupChat(r) && matchesSearch(r));

  const renderChatButton = (room: RoomResponse) => {
    const label = getRoomLabel(room);
    const isDirect = isDirectChat(room);
    const isOnline = isDirect && room.other_user_id ? onlineUserIds.has(room.other_user_id) : false;

    return (
      <button
        key={room.id}
        type="button"
        onClick={() => onRoomSelect(room.id)}
        className={`w-full flex items-center gap-2.5 px-3 py-2.5 rounded-xl text-sm transition-all duration-200 group ${
          room.id === activeRoomId
            ? 'bg-accent-muted text-accent font-medium shadow-sm'
            : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800/50 hover:text-slate-900 dark:hover:text-slate-200'
        }`}
      >
        {isDirect ? (
          <UserAvatar avatarUrl={room.other_user_avatar_url} name={label} size="sm" online={isOnline} />
        ) : (
          <Users
            className={`w-4 h-4 shrink-0 transition-colors ${
              room.id === activeRoomId ? 'text-accent' : 'opacity-40 group-hover:opacity-70'
            }`}
          />
        )}
        <span className="truncate">{label}</span>
      </button>
    );
  };

  return (
    <aside className="w-72 border-r border-slate-200/80 dark:border-slate-800/80 glass flex flex-col shrink-0 transition-colors duration-300">
      <div className="p-4 border-b border-slate-200 dark:border-slate-800 flex items-center">
        <div className="flex items-center gap-2.5">
          <div className="p-2 bg-accent-muted text-accent rounded-xl">
            <MessageSquare className="w-5 h-5" />
          </div>
          <span className="font-bold tracking-tight text-lg text-slate-900 dark:text-white">PulseChat</span>
        </div>
      </div>

      <div className="p-3">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder="Search chats..."
            className="w-full bg-slate-100 dark:bg-slate-800/50 border border-transparent focus:border-accent rounded-lg pl-9 pr-3 py-2 text-xs text-slate-700 dark:text-slate-300 placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none transition-all duration-200"
          />
        </div>
      </div>

      <div className="flex-1 overflow-y-auto min-h-0">
        {isFetchingRooms && (
          <div className="flex items-center justify-center py-6 text-slate-400 gap-2">
            <Loader2 className="w-4 h-4 animate-spin" />
            <span className="text-xs">Loading...</span>
          </div>
        )}

        {!isFetchingRooms && searchQuery && directMessages.length === 0 && groups.length === 0 && (
          <div className="text-center py-8 text-slate-400 text-xs px-4">No chats match your search</div>
        )}

        <div className="px-4 pt-1 pb-2 flex items-center justify-between">
          <span className="text-[11px] font-semibold uppercase tracking-widest text-slate-400 dark:text-slate-500">
            Direct
          </span>
          <button
            type="button"
            onClick={onShowNewDM}
            className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-400 hover:text-accent rounded-lg transition-all duration-200"
            title="New direct chat"
          >
            <Plus className="w-4 h-4" />
          </button>
        </div>
        <div className="px-2 pb-2 space-y-0.5">
          {directMessages.map(renderChatButton)}
          {!isFetchingRooms && directMessages.length === 0 && !searchQuery && (
            <p className="px-3 py-1 text-[11px] text-slate-400">No direct chats yet</p>
          )}
        </div>

        <div className="px-4 py-2 flex items-center justify-between border-t border-slate-200/60 dark:border-slate-800/60 mt-1">
          <span className="text-[11px] font-semibold uppercase tracking-widest text-slate-400 dark:text-slate-500">
            Groups
          </span>
          <button
            type="button"
            onClick={onToggleCreateGroup}
            className="p-1.5 hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-400 hover:text-accent rounded-lg transition-all duration-200"
            title="Create group"
          >
            <Plus className="w-4 h-4" />
          </button>
        </div>

        {isCreatingGroup && (
          <form onSubmit={onCreateGroup} className="px-3 mb-2 animate-fade-in-up">
            <div className="p-3 bg-slate-100 dark:bg-slate-800/50 rounded-xl space-y-3">
              <input
                type="text"
                value={groupName}
                onChange={(e) => onGroupNameChange(e.target.value)}
                placeholder="Group name"
                className="w-full bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 focus:border-accent rounded-lg px-3 py-2 text-xs text-slate-700 dark:text-slate-200 focus:outline-none transition-colors"
                autoFocus
                required
              />
              <UserSearch
                onSelect={(user) => onInvitedMembersChange([...invitedMembers, user])}
                excludeIds={currentUser?.id ? [currentUser.id] : []}
                selectedUsers={invitedMembers}
                onRemoveSelected={(id) => onInvitedMembersChange(invitedMembers.filter((u) => u.id !== id))}
                showSelected
                placeholder="Add members by username or email..."
              />
              <div className="flex justify-end gap-2">
                <button
                  type="button"
                  onClick={onCancelCreateGroup}
                  className="px-3 py-1.5 text-xs text-slate-500 hover:text-slate-700 dark:hover:text-white rounded-lg hover:bg-slate-200 dark:hover:bg-slate-700 transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isSavingGroup}
                  className="px-3 py-1.5 text-xs bg-accent-gradient text-white rounded-lg font-semibold hover:opacity-90 transition-opacity disabled:opacity-50"
                >
                  {isSavingGroup ? 'Creating...' : 'Create Group'}
                </button>
              </div>
            </div>
          </form>
        )}

        <div className="px-2 space-y-0.5">
          {groups.map(renderChatButton)}
          {!isFetchingRooms && groups.length === 0 && !searchQuery && !isCreatingGroup && (
            <p className="px-3 py-1 text-[11px] text-slate-400">No groups yet</p>
          )}
        </div>

        {errorMsg && (
          <div className="mx-3 my-3 p-2.5 bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/20 rounded-xl text-rose-600 dark:text-rose-400 text-xs flex items-center gap-1.5 animate-fade-in">
            <AlertCircle className="w-3.5 h-3.5 shrink-0" />
            <span>{errorMsg}</span>
          </div>
        )}
      </div>

      <div className="p-3 border-t border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-900/60 transition-colors shrink-0">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2.5 overflow-hidden">
            {currentUser?.avatar_url ? (
              <img
                src={getFileURL(currentUser.avatar_url)}
                alt={currentUser.username}
                className="w-9 h-9 rounded-full object-cover shrink-0 shadow-sm border border-slate-200 dark:border-slate-700"
              />
            ) : (
              <div className="w-9 h-9 rounded-full bg-accent-gradient text-white flex items-center justify-center font-bold text-sm shrink-0 uppercase shadow-sm">
                {currentUser?.username.slice(0, 2) || 'PC'}
              </div>
            )}
            <div className="overflow-hidden">
              <h4 className="text-sm font-semibold text-slate-800 dark:text-slate-200 truncate">
                {currentUser?.username || 'Me'}
              </h4>
              <p className="text-[10px] text-emerald-500 font-medium flex items-center gap-1">
                <span className="w-1.5 h-1.5 bg-emerald-500 rounded-full inline-block" />
                Online
              </p>
            </div>
          </div>
          <div className="flex items-center gap-1">
            <button
              onClick={onNavigateSettings}
              className="p-2 text-slate-400 hover:text-accent hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-all duration-200"
              title="Settings"
            >
              <Settings className="w-4 h-4" />
            </button>
            <button
              onClick={onLogout}
              className="p-2 text-slate-400 hover:text-rose-500 hover:bg-rose-50 dark:hover:bg-rose-500/10 rounded-lg transition-all duration-200"
              title="Sign Out"
            >
              <LogOut className="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>
    </aside>
  );
}
