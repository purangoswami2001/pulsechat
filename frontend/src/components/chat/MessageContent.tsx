import { RoomMemberResponse } from '../../services/api';

interface MessageContentProps {
  content: string;
  members?: RoomMemberResponse[];
  isMe?: boolean;
}

const MENTION_REGEX = /(@all|@[a-zA-Z0-9_]+)/gi;

export default function MessageContent({ content, members = [], isMe = false }: MessageContentProps) {
  const memberNames = new Set(members.map((m) => m.username.toLowerCase()));

  const parts = content.split(MENTION_REGEX);

  return (
    <p>
      {parts.map((part, i) => {
        if (!part.startsWith('@')) {
          return <span key={i}>{part}</span>;
        }

        const label = part.slice(1);
        const isAll = label.toLowerCase() === 'all';
        const isKnown = isAll || memberNames.has(label.toLowerCase());

        if (!isKnown) {
          return <span key={i}>{part}</span>;
        }

        return (
          <span
            key={i}
            className={`font-semibold rounded px-0.5 ${
              isMe ? 'text-white/95 bg-white/20' : 'text-accent bg-accent-muted'
            }`}
          >
            {isAll ? '@all' : `@${label}`}
          </span>
        );
      })}
    </p>
  );
}
