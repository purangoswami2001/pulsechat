import { RoomResponse } from '../../services/api';

export const isGroupChat = (room: RoomResponse) => room.type === 'group' || room.type === 'private';
export const isDirectChat = (room: RoomResponse) => room.type === 'direct';

export const getRoomLabel = (room: RoomResponse) => {
  if (room.type === 'direct') return room.display_name || room.name;
  return room.name;
};

export const getDateLabel = (dateStr: string) => {
  const date = new Date(dateStr);
  const today = new Date();
  const yesterday = new Date(today);
  yesterday.setDate(yesterday.getDate() - 1);

  if (date.toDateString() === today.toDateString()) return 'Today';
  if (date.toDateString() === yesterday.toDateString()) return 'Yesterday';
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
};
