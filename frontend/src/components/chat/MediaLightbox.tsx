import { X } from 'lucide-react';
import { useEffect } from 'react';
import { getFileURL } from '../../services/api';

interface MediaLightboxProps {
  url: string;
  type: 'image' | 'video';
  onClose: () => void;
}

export default function MediaLightbox({ url, type, onClose }: MediaLightboxProps) {
  const fullUrl = getFileURL(url);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  return (
    <div
      className="fixed inset-0 z-[100] flex items-center justify-center bg-black/80 backdrop-blur-sm animate-fade-in p-4"
      onClick={onClose}
    >
      <button
        type="button"
        onClick={onClose}
        className="absolute top-4 right-4 p-2.5 rounded-full bg-black/50 hover:bg-black/70 text-white transition-colors z-10"
        aria-label="Close"
      >
        <X className="w-6 h-6" />
      </button>

      <div
        className="relative max-w-[95vw] max-h-[90vh] flex items-center justify-center"
        onClick={(e) => e.stopPropagation()}
      >
        {type === 'image' ? (
          <img
            src={fullUrl}
            alt="Attachment preview"
            className="max-w-full max-h-[90vh] rounded-xl object-contain shadow-2xl"
          />
        ) : (
          <video
            src={fullUrl}
            controls
            autoPlay
            className="max-w-full max-h-[90vh] rounded-xl shadow-2xl bg-black"
          />
        )}
      </div>
    </div>
  );
}
