import { useState, useEffect, useRef } from 'react';
import { Loader2, Search, UserPlus, X } from 'lucide-react';
import { api, UserSearchResult, getFileURL } from '../services/api';

interface UserSearchProps {
  onSelect: (user: UserSearchResult) => void;
  placeholder?: string;
  excludeIds?: string[];
  selectedUsers?: UserSearchResult[];
  onRemoveSelected?: (userId: string) => void;
  showSelected?: boolean;
}

export default function UserSearch({
  onSelect,
  placeholder = 'Search by username or email...',
  excludeIds = [],
  selectedUsers = [],
  onRemoveSelected,
  showSelected = false,
}: UserSearchProps) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<UserSearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const debounceRef = useRef<number | null>(null);

  const excludeIdsKey = JSON.stringify(excludeIds);
  const selectedUsersKey = JSON.stringify(selectedUsers);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);

    if (query.trim().length < 1) {
      setResults([]);
      return;
    }

    debounceRef.current = window.setTimeout(async () => {
      setIsSearching(true);
      try {
        const users = await api.searchUsers(query.trim());
        const parsedExclude = JSON.parse(excludeIdsKey) as string[];
        const parsedSelected = JSON.parse(selectedUsersKey) as UserSearchResult[];
        const filtered = users.filter(
          (u) => !parsedExclude.includes(u.id) && !parsedSelected.some((s) => s.id === u.id)
        );
        setResults(filtered);
      } catch {
        setResults([]);
      } finally {
        setIsSearching(false);
      }
    }, 300);

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [query, excludeIdsKey, selectedUsersKey]);

  const handleSelect = (user: UserSearchResult) => {
    onSelect(user);
    setQuery('');
    setResults([]);
  };

  return (
    <div className="space-y-2">
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={placeholder}
          className="w-full bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 focus:border-accent rounded-xl pl-9 pr-9 py-2.5 text-sm text-slate-700 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-accent/20 transition-all"
        />
        {isSearching && (
          <Loader2 className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400 animate-spin" />
        )}
      </div>

      {showSelected && selectedUsers.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {selectedUsers.map((user) => (
            <span
              key={user.id}
              className="inline-flex items-center gap-1.5 px-2.5 py-1 bg-accent-muted text-accent rounded-full text-xs font-medium"
            >
              {user.username}
              {onRemoveSelected && (
                <button
                  type="button"
                  onClick={() => onRemoveSelected(user.id)}
                  className="hover:text-rose-500 transition-colors"
                >
                  <X className="w-3 h-3" />
                </button>
              )}
            </span>
          ))}
        </div>
      )}

      {results.length > 0 && (
        <div className="border border-slate-200 dark:border-slate-700 rounded-xl overflow-hidden bg-white dark:bg-slate-900 shadow-lg max-h-48 overflow-y-auto">
          {results.map((user) => (
            <button
              key={user.id}
              type="button"
              onClick={() => handleSelect(user)}
              className="w-full flex items-center gap-3 px-3 py-2.5 hover:bg-slate-50 dark:hover:bg-slate-800/60 transition-colors text-left"
            >
              {user.avatar_url ? (
                <img
                  src={getFileURL(user.avatar_url)}
                  alt={user.username}
                  className="w-8 h-8 rounded-full object-cover border border-slate-200 dark:border-slate-700"
                />
              ) : (
                <div className="w-8 h-8 rounded-full bg-accent-muted text-accent flex items-center justify-center text-xs font-bold uppercase">
                  {user.username.slice(0, 2)}
                </div>
              )}
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium text-slate-800 dark:text-slate-200 truncate">{user.username}</p>
                <p className="text-xs text-slate-400 truncate">{user.email}</p>
              </div>
              <UserPlus className="w-4 h-4 text-accent shrink-0" />
            </button>
          ))}
        </div>
      )}

      {query.trim().length >= 1 && !isSearching && results.length === 0 && (
        <p className="text-xs text-slate-400 px-1">No users found</p>
      )}
    </div>
  );
}
