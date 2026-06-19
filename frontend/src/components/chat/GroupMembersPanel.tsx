import { Users, X, UserPlus, Shield, UserMinus } from 'lucide-react';
import { RoomMemberResponse, getFileURL } from '../../services/api';

interface GroupMembersPanelProps {
  members: RoomMemberResponse[];
  currentUserId?: string;
  isAdmin: boolean;
  onlineUserIds: Set<string>;
  onClose: () => void;
  onAddMembers: () => void;
  onRemoveMember: (userId: string) => void;
}

export default function GroupMembersPanel({
  members,
  currentUserId,
  isAdmin,
  onlineUserIds,
  onClose,
  onAddMembers,
  onRemoveMember,
}: GroupMembersPanelProps) {
  return (
    <aside className="w-72 shrink-0 border-l border-slate-200/80 dark:border-slate-800/80 glass flex flex-col h-full animate-slide-in-right">
      <div className="p-4 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between shrink-0">
        <div className="flex items-center gap-2">
          <Users className="w-4 h-4 text-accent" />
          <span className="text-xs font-semibold uppercase tracking-widest text-slate-500 dark:text-slate-400">
            Members
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-[11px] font-bold text-accent bg-accent-muted px-2 py-0.5 rounded-full">
            {members.length}
          </span>
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 hover:bg-slate-200 dark:hover:bg-slate-700 rounded-lg text-slate-400"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      </div>

      {isAdmin && (
        <div className="px-3 py-2 border-b border-slate-200/60 dark:border-slate-800/60 shrink-0">
          <button
            type="button"
            onClick={onAddMembers}
            className="w-full flex items-center justify-center gap-2 px-3 py-2 text-xs font-medium text-accent bg-accent-muted rounded-lg hover:opacity-90 transition-opacity"
          >
            <UserPlus className="w-4 h-4" />
            Add members
          </button>
        </div>
      )}

      <div className="flex-1 overflow-y-auto p-3 space-y-1 min-h-0">
        {members.length === 0 ? (
          <div className="text-center py-8 text-slate-400 text-xs">Loading members...</div>
        ) : (
          members.map((member, idx) => {
            const isMe = member.id === currentUserId;
            const isOnline = onlineUserIds.has(member.id);
            const showOnlineDot = isMe || isOnline;
            const canRemove = isAdmin && !member.is_admin && !isMe;

            const statusLabel = isMe ? null : isOnline ? 'Online' : 'Offline';

            return (
              <div
                key={member.id}
                className="flex items-center gap-2.5 px-2.5 py-2 rounded-xl text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800/40 transition-all duration-200 group"
                style={{ animationDelay: `${idx * 50}ms` }}
              >
                <div className="relative">
                  {member.avatar_url ? (
                    <img
                      src={getFileURL(member.avatar_url)}
                      alt={member.username}
                      className="w-8 h-8 rounded-full object-cover border border-slate-200 dark:border-slate-700"
                    />
                  ) : (
                    <div className="w-8 h-8 rounded-full bg-accent-muted text-accent flex items-center justify-center font-bold text-[10px] uppercase">
                      {member.username.slice(0, 2)}
                    </div>
                  )}
                  {showOnlineDot && (
                    <div className="absolute -bottom-0.5 -right-0.5 w-3 h-3 bg-emerald-500 rounded-full border-2 border-white dark:border-slate-900" />
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <span className="truncate text-xs font-medium flex items-center gap-1">
                    {isMe ? `${member.username} (You)` : member.username}
                    {member.is_admin && (
                      <span title="Group admin">
                        <Shield className="w-3 h-3 text-amber-500 shrink-0" />
                      </span>
                    )}
                  </span>
                  {statusLabel && (
                    <span
                      className={`text-[10px] font-medium ${
                        isOnline ? 'text-emerald-500' : 'text-slate-400'
                      }`}
                    >
                      {statusLabel}
                    </span>
                  )}
                </div>
                {canRemove && (
                  <button
                    type="button"
                    onClick={() => onRemoveMember(member.id)}
                    className="p-1.5 rounded-lg text-slate-400 hover:text-rose-500 hover:bg-rose-50 dark:hover:bg-rose-500/10 opacity-0 group-hover:opacity-100 transition-all"
                    title="Remove member"
                  >
                    <UserMinus className="w-3.5 h-3.5" />
                  </button>
                )}
              </div>
            );
          })
        )}
      </div>
    </aside>
  );
}
