import { useState } from 'react';

const EMOJI_CATEGORIES = [
  {
    name: 'Smileys',
    icon: '😊',
    emojis: [
      '😀','😃','😄','😁','😆','😅','🤣','😂','🙂','😊',
      '😇','🥰','😍','🤩','😘','😗','😚','😙','🥲','😋',
      '😛','😜','🤪','😝','🤑','🤗','🤭','🤫','🤔','🫡',
      '🤐','🤨','😐','😑','😶','🫥','😏','😒','🙄','😬',
      '😮‍💨','🤥','😌','😔','😪','🤤','😴','😷','🤒','🤕',
      '🤢','🤮','🥴','😵','🤯','🥳','🥸','😎','🤓','🧐',
    ],
  },
  {
    name: 'Gestures',
    icon: '👋',
    emojis: [
      '👋','🤚','🖐️','✋','🖖','🫱','🫲','🫳','🫴','👌',
      '🤌','🤏','✌️','🤞','🫰','🤟','🤘','🤙','👈','👉',
      '👆','🖕','👇','☝️','🫵','👍','👎','✊','👊','🤛',
      '🤜','👏','🙌','🫶','👐','🤲','🤝','🙏','💪','🦾',
    ],
  },
  {
    name: 'Hearts',
    icon: '❤️',
    emojis: [
      '❤️','🧡','💛','💚','💙','💜','🖤','🤍','🤎','💔',
      '❤️‍🔥','❤️‍🩹','💕','💞','💓','💗','💖','💘','💝','💟',
      '♥️','💋','🫂','✨','⭐','🌟','💫','🔥','💥','💢',
    ],
  },
  {
    name: 'Animals',
    icon: '🐱',
    emojis: [
      '🐶','🐱','🐭','🐹','🐰','🦊','🐻','🐼','🐻‍❄️','🐨',
      '🐯','🦁','🐮','🐷','🐸','🐵','🙈','🙉','🙊','🐔',
      '🐧','🐦','🐤','🦆','🦅','🦉','🐝','🐛','🦋','🐌',
    ],
  },
  {
    name: 'Food',
    icon: '🍕',
    emojis: [
      '🍎','🍐','🍊','🍋','🍌','🍉','🍇','🍓','🫐','🍒',
      '🍑','🥭','🍍','🥥','🥝','🍅','🥑','🍕','🍔','🍟',
      '🌭','🍿','🧁','🍰','🎂','🍩','🍪','🍫','☕','🍵',
    ],
  },
  {
    name: 'Objects',
    icon: '💡',
    emojis: [
      '💡','🔦','🕯️','💰','💎','🔑','🗝️','🔒','📱','💻',
      '⌨️','🖥️','📷','📸','🎥','📽️','🎬','📺','📻','🎵',
      '🎶','🎤','🎧','🎸','🎹','🎺','🎷','🥁','📚','📖',
    ],
  },
];

interface EmojiPickerProps {
  onEmojiSelect: (emoji: string) => void;
  onClose: () => void;
}

export default function EmojiPicker({ onEmojiSelect, onClose }: EmojiPickerProps) {
  const [activeCategory, setActiveCategory] = useState(0);

  const currentCategory = EMOJI_CATEGORIES[activeCategory];

  return (
    <div className="absolute bottom-full right-0 mb-2 w-80 glass rounded-2xl shadow-2xl z-50 animate-scale-in overflow-hidden border border-slate-200/60 dark:border-slate-700/60">
      {/* Header */}
      <div className="flex items-center justify-between p-3 border-b border-slate-200/50 dark:border-slate-700/50 bg-slate-50/50 dark:bg-slate-800/30">
        <span className="text-xs font-semibold text-slate-600 dark:text-slate-300 uppercase tracking-wider">Emojis</span>
        <button
          type="button"
          onClick={onClose}
          className="p-1 hover:bg-slate-200 dark:hover:bg-slate-700 rounded-md transition-colors text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
        >
          ✕
        </button>
      </div>

      {/* Category tabs */}
      <div className="flex gap-0.5 px-2 py-1.5 border-b border-slate-200/30 dark:border-slate-700/30 overflow-x-auto">
        {EMOJI_CATEGORIES.map((cat, idx) => (
          <button
            key={cat.name}
            onClick={() => setActiveCategory(idx)}
            className={`p-1.5 rounded-lg text-lg transition-all duration-150 shrink-0 ${
              idx === activeCategory
                ? 'bg-accent-muted scale-110'
                : 'hover:bg-slate-100 dark:hover:bg-slate-800/50'
            }`}
            title={cat.name}
          >
            {cat.icon}
          </button>
        ))}
      </div>

      {/* Category label */}
      <div className="px-3 pt-2 pb-1">
        <span className="text-[10px] font-semibold text-slate-400 uppercase tracking-widest">
          {currentCategory.name}
        </span>
      </div>

      {/* Emoji grid */}
      <div className="p-2 grid grid-cols-8 gap-0.5 max-h-48 overflow-y-auto">
        {currentCategory.emojis.map((emoji, idx) => (
          <button
            key={`${emoji}-${idx}`}
            type="button"
            onClick={() => onEmojiSelect(emoji)}
            className="p-1.5 text-xl hover:bg-accent-muted rounded-lg transition-all duration-100 hover:scale-125 active:scale-100"
          >
            {emoji}
          </button>
        ))}
      </div>

      {/* Footer hint */}
      <div className="px-3 py-2 border-t border-slate-200/30 dark:border-slate-700/30 bg-slate-50/30 dark:bg-slate-800/20">
        <p className="text-[10px] text-slate-400 text-center">
          Tap emojis to add to your message — click Send when ready
        </p>
      </div>
    </div>
  );
}
