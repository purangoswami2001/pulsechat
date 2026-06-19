import { getFileURL } from '../../services/api';

const SIZE_CLASSES = {
  sm: 'w-8 h-8 text-[10px]',
  md: 'w-10 h-10 text-xs',
  lg: 'w-12 h-12 text-sm',
} as const;

interface UserAvatarProps {
  avatarUrl?: string;
  name: string;
  size?: keyof typeof SIZE_CLASSES;
  online?: boolean;
  className?: string;
}

export default function UserAvatar({ avatarUrl, name, size = 'md', online, className = '' }: UserAvatarProps) {
  const sizeClass = SIZE_CLASSES[size];
  const initials = name.slice(0, 2).toUpperCase() || '?';
  const dotSize = size === 'sm' ? 'w-2.5 h-2.5' : 'w-3 h-3';

  return (
    <div className={`relative shrink-0 ${className}`}>
      {avatarUrl ? (
        <img
          src={getFileURL(avatarUrl)}
          alt={name}
          className={`${sizeClass} rounded-full object-cover border border-slate-200 dark:border-slate-700`}
        />
      ) : (
        <div
          className={`${sizeClass} rounded-full bg-accent-muted text-accent flex items-center justify-center font-bold uppercase`}
        >
          {initials}
        </div>
      )}
      {online && (
        <div
          className={`absolute -bottom-0.5 -right-0.5 ${dotSize} bg-emerald-500 rounded-full border-2 border-white dark:border-slate-900`}
        />
      )}
    </div>
  );
}
