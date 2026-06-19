// Fetch base URL from environment variables
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

export const LOGIN_SERVER_ERROR_KEY = 'pulsechat_server_error';
export const SERVER_UNAVAILABLE_MSG = 'Unable to handle request right now. Please try again.';

const REQUEST_TIMEOUT_MS = 15000;

export class ApiError extends Error {
  readonly status?: number;
  readonly isUnavailable: boolean;

  constructor(message: string, status?: number, isUnavailable = false) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.isUnavailable = isUnavailable;
  }
}

function clearAuthSession() {
  localStorage.removeItem('pulsechat_token');
  localStorage.removeItem('pulsechat_user');
}

export function redirectToLoginUnavailable(message = SERVER_UNAVAILABLE_MSG) {
  clearAuthSession();
  sessionStorage.setItem(LOGIN_SERVER_ERROR_KEY, message);
  const path = window.location.pathname;
  if (path !== '/login' && path !== '/register') {
    window.location.href = '/login';
  }
}

function isNetworkFailure(err: unknown): boolean {
  if (err instanceof ApiError) return err.isUnavailable;
  if (err instanceof TypeError) return true;
  if (err instanceof DOMException && err.name === 'AbortError') return true;
  return false;
}

function isPublicAuthPath(endpoint: string): boolean {
  return (
    endpoint.startsWith('/auth/login') ||
    endpoint.startsWith('/auth/register') ||
    endpoint === '/health'
  );
}

function shouldRedirectOnFailure(endpoint: string): boolean {
  const token = localStorage.getItem('pulsechat_token');
  if (!token) return false;
  if (isPublicAuthPath(endpoint)) return false;
  const path = window.location.pathname;
  return path !== '/login' && path !== '/register';
}

async function fetchWithTimeout(url: string, options: RequestInit): Promise<Response> {
  const controller = new AbortController();
  const timeoutId = window.setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  try {
    return await fetch(url, { ...options, signal: controller.signal });
  } finally {
    window.clearTimeout(timeoutId);
  }
}

function parseErrorMessage(response: Response, errorText: string): string {
  let errorMessage = `HTTP Error ${response.status}`;
  try {
    const errorJson = JSON.parse(errorText);
    errorMessage = errorJson.message || errorJson.error || errorMessage;
  } catch {
    if (errorText) errorMessage = errorText;
  }
  return errorMessage;
}

function handleRequestFailure(err: unknown, endpoint: string): never {
  if (err instanceof ApiError) {
    if (err.isUnavailable && shouldRedirectOnFailure(endpoint)) {
      redirectToLoginUnavailable(err.message);
    }
    throw err;
  }

  if (isNetworkFailure(err)) {
    const apiErr = new ApiError(SERVER_UNAVAILABLE_MSG, undefined, true);
    if (shouldRedirectOnFailure(endpoint)) {
      redirectToLoginUnavailable(apiErr.message);
    }
    throw apiErr;
  }

  throw err;
}

// API helper utility to perform fetch requests with token injection
async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem('pulsechat_token');
  const headers = new Headers(options.headers || {});

  headers.set('Content-Type', 'application/json');
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  try {
    const response = await fetchWithTimeout(`${API_BASE_URL}${endpoint}`, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const errorText = await response.text();
      const errorMessage = parseErrorMessage(response, errorText);

      if (response.status >= 502 && response.status <= 504) {
        throw new ApiError(SERVER_UNAVAILABLE_MSG, response.status, true);
      }

      if (response.status === 401 && shouldRedirectOnFailure(endpoint)) {
        redirectToLoginUnavailable('Session expired. Please sign in again.');
        throw new ApiError(errorMessage, 401);
      }

      throw new ApiError(errorMessage, response.status);
    }

    if (response.status === 204) {
      return {} as T;
    }

    return response.json();
  } catch (err) {
    handleRequestFailure(err, endpoint);
  }
}

// Multipart upload helper (no Content-Type header — browser sets it with boundary)
async function uploadFile(endpoint: string, formData: FormData): Promise<any> {
  const token = localStorage.getItem('pulsechat_token');
  const headers: Record<string, string> = {};
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  try {
    const response = await fetchWithTimeout(`${API_BASE_URL}${endpoint}`, {
      method: 'POST',
      headers,
      body: formData,
    });

    if (!response.ok) {
      const errorText = await response.text();
      const errorMessage = parseErrorMessage(response, errorText);

      if (response.status >= 502 && response.status <= 504) {
        throw new ApiError(SERVER_UNAVAILABLE_MSG, response.status, true);
      }

      if (response.status === 401 && shouldRedirectOnFailure(endpoint)) {
        redirectToLoginUnavailable('Session expired. Please sign in again.');
        throw new ApiError(errorMessage, 401);
      }

      throw new ApiError(errorMessage, response.status);
    }

    return response.json();
  } catch (err) {
    handleRequestFailure(err, endpoint);
  }
}

export interface UserResponse {
  id: string;
  username: string;
  email: string;
  avatar_url?: string;
  created_at: string;
}

export interface AuthResponse {
  token: string;
  user: UserResponse;
}

export interface ProfileResponse {
  id: string;
  username: string;
  email: string;
  avatar_url: string;
  created_at: string;
}

export interface RoomResponse {
  id: string;
  name: string;
  type: string;
  created_at: string;
  display_name?: string;
  other_user_id?: string;
  other_user_avatar_url?: string;
  member_count?: number;
}

export interface UserSearchResult {
  id: string;
  username: string;
  email: string;
  avatar_url?: string;
}

export interface RoomMemberResponse {
  id: string;
  username: string;
  email: string;
  avatar_url?: string;
  joined_at: string;
  is_admin?: boolean;
}

export interface OnlineUserResponse {
  user_id: string;
  username: string;
  avatar_url?: string;
}

export interface MessageResponse {
  id: string;
  room_id: string;
  sender_id: string;
  sender_name?: string;
  sender_avatar_url?: string;
  content: string;
  attachment_url?: string;
  attachment_type?: string;
  created_at: string;
}

export interface UploadResponse {
  url: string;
  type: string;
  name: string;
}

export const api = {
  pingServer: (): Promise<{ status: string }> => {
    return request<{ status: string }>('/health');
  },

  // Authentication APIs
  register: (username: string, email: string, password: string): Promise<AuthResponse> => {
    return request<AuthResponse>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, email, password }),
    });
  },

  login: (email: string, password: string): Promise<AuthResponse> => {
    return request<AuthResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
  },

  // Profile APIs
  getProfile: (): Promise<ProfileResponse> => {
    return request<ProfileResponse>('/auth/profile');
  },

  updateProfile: (username: string, email: string): Promise<ProfileResponse> => {
    return request<ProfileResponse>('/auth/profile', {
      method: 'PUT',
      body: JSON.stringify({ username, email }),
    });
  },

  uploadAvatar: (file: File): Promise<ProfileResponse> => {
    const formData = new FormData();
    formData.append('avatar', file);
    return uploadFile('/auth/avatar', formData);
  },

  removeAvatar: (): Promise<ProfileResponse> => {
    return request<ProfileResponse>('/auth/avatar', {
      method: 'DELETE',
    });
  },

  // Room APIs
  listRooms: (): Promise<RoomResponse[]> => {
    return request<RoomResponse[]>('/rooms');
  },

  createRoom: (name: string, memberIds: string[] = []): Promise<RoomResponse> => {
    return request<RoomResponse>('/rooms', {
      method: 'POST',
      body: JSON.stringify({ name, type: 'group', member_ids: memberIds }),
    });
  },

  createDirectRoom: (userId: string): Promise<RoomResponse> => {
    return request<RoomResponse>('/rooms/direct', {
      method: 'POST',
      body: JSON.stringify({ user_id: userId }),
    });
  },

  listRoomMembers: (roomID: string): Promise<RoomMemberResponse[]> => {
    return request<RoomMemberResponse[]>(`/rooms/${roomID}/members`);
  },

  addRoomMember: (roomID: string, payload: { user_id?: string; username?: string }): Promise<RoomMemberResponse[]> => {
    return request<RoomMemberResponse[]>(`/rooms/${roomID}/members`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  },

  removeRoomMember: (roomID: string, userID: string): Promise<RoomMemberResponse[]> => {
    return request<RoomMemberResponse[]>(`/rooms/${roomID}/members/${userID}`, {
      method: 'DELETE',
    });
  },

  deleteGroup: async (roomID: string): Promise<void> => {
    try {
      await request<void>(`/rooms/${roomID}`, { method: 'DELETE' });
    } catch (err: unknown) {
      const msg = err instanceof ApiError ? err.message : err instanceof Error ? err.message : '';
      if (msg.includes('404') || msg.toLowerCase().includes('not found')) {
        await request<void>(`/rooms/${roomID}/delete`, { method: 'POST' });
        return;
      }
      throw err;
    }
  },

  // User search
  searchUsers: (query: string): Promise<UserSearchResult[]> => {
    return request<UserSearchResult[]>(`/users/search?q=${encodeURIComponent(query)}`);
  },

  // Message APIs
  listMessages: (roomID: string, limit = 50, offset = 0): Promise<MessageResponse[]> => {
    return request<MessageResponse[]>(`/rooms/${roomID}/messages?limit=${limit}&offset=${offset}`);
  },

  // Presence APIs
  listPresence: (roomID: string): Promise<OnlineUserResponse[]> => {
    return request<OnlineUserResponse[]>(`/rooms/${roomID}/presence`);
  },

  // File Upload API
  upload: (file: File): Promise<UploadResponse> => {
    const formData = new FormData();
    formData.append('file', file);
    return uploadFile('/upload', formData);
  },
};

// Helper to get the full URL for a file path
export function getFileURL(path: string): string {
  if (!path) return '';
  if (path.startsWith('http')) return path;
  return `${API_BASE_URL}${path}`;
}

export function getLoginRedirectMessage(): string | null {
  const msg = sessionStorage.getItem(LOGIN_SERVER_ERROR_KEY);
  if (msg) {
    sessionStorage.removeItem(LOGIN_SERVER_ERROR_KEY);
  }
  return msg;
}

export function formatApiError(err: unknown, fallback: string): string {
  if (err instanceof ApiError) return err.message;
  if (err instanceof Error && err.message) return err.message;
  return fallback;
}
