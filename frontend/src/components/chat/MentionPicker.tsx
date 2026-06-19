import { AtSign, Users } from 'lucide-react';
import { RoomMemberResponse } from '../../services/api';

interface MentionPickerProps {
  members: RoomMemberResponse[];
  currentUserId?: string;
  query: string;
  onSelect: (mention: string) => void;
}

export default function MentionPicker({ members, currentUserId, query, onSelect }: MentionPickerProps) {
  const q = query.toLowerCase();
  const filtered = members.filter(
    (m) => m.id !== currentUserId && m.username.toLowerCase().includes(q),
  );
  const showAll = 'all'.includes(q) || q === '';

  if (!showAll && filtered.length === 0) return null;

  return (
    <div className="absolute bottom-full left-0 mb-2 w-56 max-h-48 overflow-y-auto glass rounded-xl shadow-xl border border-slate-200/60 dark:border-slate-700/60 py-1 z-50 animate-scale-in">
      {showAll && (
        <button
          type="button"
          onClick={() => onSelect('all')}
          className="w-full flex items-center gap-2 px-3 py-2 text-sm text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800/60 transition-colors"
        >
          <Users className="w-4 h-4 text-accent shrink-0" />
          <span className="font-medium">@all</span>
          <span className="text-[10px] text-slate-400 ml-auto">Everyone</span>
        </button>
      )}
      {filtered.map((member) => (
        <button
          key={member.id}
          type="button"
          onClick={() => onSelect(member.username)}
          className="w-full flex items-center gap-2 px-3 py-2 text-sm text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800/60 transition-colors"
        >
          <AtSign className="w-4 h-4 text-accent shrink-0" />
          <span className="font-medium truncate">{member.username}</span>
        </button>
      ))}
    </div>
  );
}
