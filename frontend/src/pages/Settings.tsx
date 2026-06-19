import React, { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  ArrowLeft,
  Camera,
  Trash2,
  Loader2,
  Sun,
  Moon,
  Save,
  LogOut,
  User,
  Palette,
  Shield,
  MessageSquare,
} from 'lucide-react';

import { api, ProfileResponse, getFileURL } from '../services/api';
import { useTheme, CHAT_THEMES, ChatTheme } from '../contexts/ThemeContext';

type SettingsTab = 'profile' | 'appearance' | 'account';

const NAV_ITEMS: { id: SettingsTab; label: string; icon: React.ElementType; description: string }[] = [
  { id: 'profile', label: 'Profile', icon: User, description: 'Name, email & avatar' },
  { id: 'appearance', label: 'Appearance', icon: Palette, description: 'Theme & colors' },
  { id: 'account', label: 'Account', icon: MessageSquare, description: 'Session & sign out' },
];

export default function Settings() {
  const navigate = useNavigate();
  const { mode, toggleMode, chatTheme, setChatTheme } = useTheme();

  const [activeTab, setActiveTab] = useState<SettingsTab>('profile');
  const [profile, setProfile] = useState<ProfileResponse | null>(null);
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [isUploadingAvatar, setIsUploadingAvatar] = useState(false);
  const [successMsg, setSuccessMsg] = useState('');
  const [errorMsg, setErrorMsg] = useState('');

  const avatarInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const loadProfile = async () => {
      try {
        const data = await api.getProfile();
        setProfile(data);
        setUsername(data.username);
        setEmail(data.email);
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : '';
        if (msg.includes('401') || msg.includes('Unauthorized')) {
          localStorage.removeItem('pulsechat_token');
          localStorage.removeItem('pulsechat_user');
          navigate('/login');
          return;
        }
        setErrorMsg('Failed to load profile');
      } finally {
        setIsLoading(false);
      }
    };
    loadProfile();
  }, [navigate]);

  const showSuccess = (msg: string) => {
    setSuccessMsg(msg);
    setTimeout(() => setSuccessMsg(''), 3000);
  };

  const handleSaveProfile = async () => {
    if (!username.trim() || !email.trim()) {
      setErrorMsg('Username and email are required');
      return;
    }
    setIsSaving(true);
    setErrorMsg('');
    try {
      const updated = await api.updateProfile(username.trim(), email.trim());
      setProfile(updated);
      const stored = localStorage.getItem('pulsechat_user');
      if (stored) {
        const user = JSON.parse(stored);
        user.username = updated.username;
        user.email = updated.email;
        localStorage.setItem('pulsechat_user', JSON.stringify(user));
      }
      showSuccess('Profile updated successfully');
    } catch (err: unknown) {
      setErrorMsg(err instanceof Error ? err.message : 'Failed to update profile');
    } finally {
      setIsSaving(false);
    }
  };

  const handleAvatarUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (!file.type.startsWith('image/')) {
      setErrorMsg('Only image files are allowed');
      return;
    }
    if (file.size > 2 * 1024 * 1024) {
      setErrorMsg('Avatar must be under 2MB');
      return;
    }

    setIsUploadingAvatar(true);
    setErrorMsg('');
    try {
      const updated = await api.uploadAvatar(file);
      setProfile(updated);
      showSuccess('Avatar updated');
    } catch (err: unknown) {
      setErrorMsg(err instanceof Error ? err.message : 'Failed to upload avatar');
    } finally {
      setIsUploadingAvatar(false);
      if (avatarInputRef.current) avatarInputRef.current.value = '';
    }
  };

  const handleRemoveAvatar = async () => {
    setIsUploadingAvatar(true);
    setErrorMsg('');
    try {
      const updated = await api.removeAvatar();
      setProfile(updated);
      showSuccess('Avatar removed');
    } catch (err: unknown) {
      setErrorMsg(err instanceof Error ? err.message : 'Failed to remove avatar');
    } finally {
      setIsUploadingAvatar(false);
    }
  };

  const handleLogout = () => {
    localStorage.removeItem('pulsechat_token');
    localStorage.removeItem('pulsechat_user');
    navigate('/login');
  };

  if (isLoading) {
    return (
      <div className="h-screen flex items-center justify-center bg-slate-50 dark:bg-slate-950 transition-colors">
        <Loader2 className="w-8 h-8 animate-spin text-accent" />
      </div>
    );
  }

  const avatarURL = profile?.avatar_url ? getFileURL(profile.avatar_url) : '';
  const initials = profile?.username?.slice(0, 2).toUpperCase() || 'PC';

  return (
    <div className="h-screen flex bg-slate-50 dark:bg-slate-950 text-slate-900 dark:text-slate-100 overflow-hidden transition-colors duration-300">
      {/* Sidebar navigation */}
      <aside className="w-72 border-r border-slate-200/80 dark:border-slate-800/80 glass flex flex-col shrink-0">
        <div className="p-5 border-b border-slate-200 dark:border-slate-800">
          <button
            onClick={() => {
              const last = sessionStorage.getItem('pulsechat_last_room');
              navigate(last ? `/chat/${last}` : '/');
            }}
            className="flex items-center gap-2 text-slate-500 hover:text-accent transition-colors mb-4 text-sm font-medium"
          >
            <ArrowLeft className="w-4 h-4" />
            Back to chat
          </button>
          <h1 className="text-xl font-bold text-slate-900 dark:text-white">Settings</h1>
          <p className="text-xs text-slate-500 mt-1">Manage your account preferences</p>
        </div>

        <nav className="flex-1 p-3 space-y-1">
          {NAV_ITEMS.map(({ id, label, icon: Icon, description }) => (
            <button
              key={id}
              onClick={() => setActiveTab(id)}
              className={`w-full flex items-start gap-3 px-3 py-3 rounded-xl text-left transition-all duration-200 ${
                activeTab === id
                  ? 'bg-accent-muted text-accent shadow-sm'
                  : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800/50'
              }`}
            >
              <Icon className={`w-5 h-5 mt-0.5 shrink-0 ${activeTab === id ? 'text-accent' : 'opacity-60'}`} />
              <div>
                <div className={`text-sm font-semibold ${activeTab === id ? 'text-accent' : 'text-slate-800 dark:text-slate-200'}`}>
                  {label}
                </div>
                <div className="text-[11px] text-slate-400 mt-0.5">{description}</div>
              </div>
            </button>
          ))}
        </nav>

        <div className="p-4 border-t border-slate-200 dark:border-slate-800">
          <div className="flex items-center gap-3">
            {avatarURL ? (
              <img src={avatarURL} alt="" className="w-10 h-10 rounded-xl object-cover border border-slate-200 dark:border-slate-700" />
            ) : (
              <div className="w-10 h-10 rounded-xl bg-accent-gradient text-white flex items-center justify-center text-sm font-bold">
                {initials}
              </div>
            )}
            <div className="min-w-0">
              <p className="text-sm font-semibold truncate">{profile?.username}</p>
              <p className="text-[11px] text-slate-400 truncate">{profile?.email}</p>
            </div>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto">
        <div className="max-w-2xl mx-auto px-8 py-10">
          {successMsg && (
            <div className="mb-6 p-3 bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-200 dark:border-emerald-500/20 rounded-xl text-emerald-600 dark:text-emerald-400 text-sm flex items-center gap-2 animate-fade-in">
              <span>✓</span>
              <span>{successMsg}</span>
            </div>
          )}
          {errorMsg && (
            <div className="mb-6 p-3 bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/20 rounded-xl text-rose-600 dark:text-rose-400 text-sm flex items-center gap-2 animate-fade-in">
              <Shield className="w-4 h-4 shrink-0" />
              <span>{errorMsg}</span>
            </div>
          )}

          {activeTab === 'profile' && (
            <section className="animate-fade-in">
              <div className="mb-8">
                <h2 className="text-2xl font-bold text-slate-900 dark:text-white">Profile</h2>
                <p className="text-sm text-slate-500 mt-1">Update your personal information and photo</p>
              </div>

              <div className="glass rounded-2xl p-6 space-y-6">
                <div className="flex items-center gap-5">
                  <div className="relative group">
                    {avatarURL ? (
                      <img
                        src={avatarURL}
                        alt="Avatar"
                        className="w-24 h-24 rounded-2xl object-cover border-2 border-slate-200 dark:border-slate-700 shadow-md"
                      />
                    ) : (
                      <div className="w-24 h-24 rounded-2xl bg-accent-gradient text-white flex items-center justify-center text-3xl font-bold shadow-md">
                        {initials}
                      </div>
                    )}
                    <button
                      onClick={() => avatarInputRef.current?.click()}
                      disabled={isUploadingAvatar}
                      className="absolute inset-0 rounded-2xl bg-black/40 opacity-0 group-hover:opacity-100 flex items-center justify-center transition-opacity cursor-pointer"
                    >
                      {isUploadingAvatar ? (
                        <Loader2 className="w-6 h-6 text-white animate-spin" />
                      ) : (
                        <Camera className="w-6 h-6 text-white" />
                      )}
                    </button>
                    <input ref={avatarInputRef} type="file" accept="image/*" onChange={handleAvatarUpload} className="hidden" />
                  </div>
                  <div className="space-y-2">
                    <button
                      onClick={() => avatarInputRef.current?.click()}
                      disabled={isUploadingAvatar}
                      className="text-sm text-accent hover:text-accent-light font-medium transition-colors"
                    >
                      Change photo
                    </button>
                    {avatarURL && (
                      <button
                        onClick={handleRemoveAvatar}
                        disabled={isUploadingAvatar}
                        className="flex items-center gap-1 text-sm text-rose-500 hover:text-rose-400 font-medium transition-colors"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                        Remove
                      </button>
                    )}
                  </div>
                </div>

                <div className="space-y-4">
                  <div>
                    <label className="block text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-2">
                      Username
                    </label>
                    <input
                      type="text"
                      value={username}
                      onChange={(e) => setUsername(e.target.value)}
                      className="w-full bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 focus:border-accent rounded-xl px-4 py-3 text-sm focus:outline-none focus:ring-1 ring-accent transition-all"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-2">
                      Email
                    </label>
                    <input
                      type="email"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      className="w-full bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 focus:border-accent rounded-xl px-4 py-3 text-sm focus:outline-none focus:ring-1 ring-accent transition-all"
                    />
                  </div>
                  <button
                    onClick={handleSaveProfile}
                    disabled={isSaving}
                    className="bg-accent-gradient text-white px-6 py-2.5 rounded-xl text-sm font-semibold flex items-center gap-2 hover:opacity-90 transition-opacity disabled:opacity-50"
                  >
                    {isSaving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                    Save Changes
                  </button>
                </div>
              </div>
            </section>
          )}

          {activeTab === 'appearance' && (
            <section className="animate-fade-in">
              <div className="mb-8">
                <h2 className="text-2xl font-bold text-slate-900 dark:text-white">Appearance</h2>
                <p className="text-sm text-slate-500 mt-1">Customize how PulseChat looks and feels</p>
              </div>

              <div className="glass rounded-2xl p-6 space-y-6">
                <div className="flex items-center justify-between py-2">
                  <div>
                    <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300">Theme Mode</h3>
                    <p className="text-xs text-slate-400 mt-0.5">Switch between dark and light mode</p>
                  </div>
                  <button
                    onClick={toggleMode}
                    className="flex items-center gap-2 px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                  >
                    {mode === 'dark' ? (
                      <>
                        <Moon className="w-4 h-4 text-indigo-400" />
                        <span className="text-sm font-medium">Dark</span>
                      </>
                    ) : (
                      <>
                        <Sun className="w-4 h-4 text-amber-500" />
                        <span className="text-sm font-medium">Light</span>
                      </>
                    )}
                  </button>
                </div>

                <div className="pt-2 border-t border-slate-200/50 dark:border-slate-700/30">
                  <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-1">Chat Theme</h3>
                  <p className="text-xs text-slate-400 mb-4">Choose your accent color for chat bubbles and highlights</p>
                  <div className="grid grid-cols-5 gap-3">
                    {CHAT_THEMES.map((theme) => (
                      <button
                        key={theme.id}
                        onClick={() => setChatTheme(theme.id as ChatTheme)}
                        className={`flex flex-col items-center gap-2 p-3 rounded-xl border-2 transition-all duration-200 ${
                          chatTheme === theme.id
                            ? 'border-accent bg-accent-muted scale-105'
                            : 'border-slate-200 dark:border-slate-700 hover:border-slate-300 dark:hover:border-slate-600'
                        }`}
                      >
                        <div
                          className="w-8 h-8 rounded-full shadow-md border-2 border-white dark:border-slate-600"
                          style={{ backgroundColor: theme.color }}
                        />
                        <span className="text-[11px] font-medium text-slate-600 dark:text-slate-400">{theme.label}</span>
                      </button>
                    ))}
                  </div>
                </div>
              </div>
            </section>
          )}

          {activeTab === 'account' && (
            <section className="animate-fade-in">
              <div className="mb-8">
                <h2 className="text-2xl font-bold text-slate-900 dark:text-white">Account</h2>
                <p className="text-sm text-slate-500 mt-1">Your account details and session</p>
              </div>

              <div className="glass rounded-2xl p-6 space-y-4">
                <div className="flex items-center justify-between py-3 border-b border-slate-200/50 dark:border-slate-700/30">
                  <span className="text-sm text-slate-500">Member since</span>
                  <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
                    {profile
                      ? new Date(profile.created_at).toLocaleDateString(undefined, {
                          year: 'numeric',
                          month: 'long',
                          day: 'numeric',
                        })
                      : '—'}
                  </span>
                </div>
                <div className="flex items-center justify-between py-3 border-b border-slate-200/50 dark:border-slate-700/30">
                  <span className="text-sm text-slate-500">User ID</span>
                  <span className="text-xs font-mono text-slate-400 bg-slate-100 dark:bg-slate-800 px-2 py-1 rounded-lg">
                    {profile?.id}
                  </span>
                </div>
                <div className="flex items-center justify-between py-3">
                  <span className="text-sm text-slate-500">Email</span>
                  <span className="text-sm font-medium text-slate-700 dark:text-slate-300">{profile?.email}</span>
                </div>

                <div className="pt-4 mt-2 border-t border-slate-200/50 dark:border-slate-700/30">
                  <button
                    onClick={handleLogout}
                    className="flex items-center gap-2 px-4 py-2.5 rounded-xl text-rose-500 hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-500/10 text-sm font-semibold transition-colors"
                  >
                    <LogOut className="w-4 h-4" />
                    Sign Out
                  </button>
                </div>
              </div>
            </section>
          )}
        </div>
      </main>
    </div>
  );
}
