import React, { useState, useEffect } from 'react';
import { useNavigate, Link, useLocation } from 'react-router-dom';
import { MessageSquare, Shield, ArrowRight, Loader2, Sun, Moon } from 'lucide-react';
import { api, formatApiError, getLoginRedirectMessage, SERVER_UNAVAILABLE_MSG } from '../services/api';
import { useTheme } from '../contexts/ThemeContext';

export default function Login() {
  const navigate = useNavigate();
  const location = useLocation();
  const { mode, toggleMode } = useTheme();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    const fromStorage = getLoginRedirectMessage();
    const fromState = (location.state as { serverError?: string } | null)?.serverError;
    if (fromStorage || fromState) {
      setError(fromStorage || fromState || SERVER_UNAVAILABLE_MSG);
      window.history.replaceState({}, document.title);
    }
  }, [location.state]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError('');

    try {
      const response = await api.login(email, password);
      localStorage.setItem('pulsechat_token', response.token);
      localStorage.setItem('pulsechat_user', JSON.stringify(response.user));
      const last = sessionStorage.getItem('pulsechat_last_room');
      navigate(last ? `/chat/${last}` : '/');
    } catch (err: unknown) {
      setError(formatApiError(err, 'Failed to sign in. Please verify your credentials.'));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-50 via-white to-slate-100 dark:from-slate-900 dark:via-slate-950 dark:to-black p-4 relative overflow-hidden transition-colors duration-500">
      {/* Background decorations */}
      <div className="absolute top-1/4 left-1/4 w-96 h-96 rounded-full blur-[120px] pointer-events-none bg-brand-500/5 dark:bg-brand-500/10" />
      <div className="absolute bottom-1/4 right-1/4 w-96 h-96 rounded-full blur-[120px] pointer-events-none bg-indigo-500/5 dark:bg-indigo-500/10" />

      {/* Theme toggle floating button */}
      <button
        onClick={toggleMode}
        className="absolute top-6 right-6 p-2.5 rounded-xl glass hover:scale-105 active:scale-95 transition-all duration-200 z-20"
        title={`Switch to ${mode === 'dark' ? 'light' : 'dark'} mode`}
      >
        {mode === 'dark' ? (
          <Sun className="w-5 h-5 text-amber-400" />
        ) : (
          <Moon className="w-5 h-5 text-slate-600" />
        )}
      </button>

      <div className="w-full max-w-md glass rounded-2xl shadow-2xl dark:shadow-brand-500/5 p-8 relative z-10 animate-scale-in">
        
        {/* Brand Header */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center p-3 bg-accent-muted text-accent rounded-xl mb-4 shadow-accent">
            <MessageSquare className="w-8 h-8" />
          </div>
          <h1 className="text-3xl font-extrabold tracking-tight text-slate-900 dark:text-white">
            Welcome Back
          </h1>
          <p className="text-slate-500 dark:text-slate-400 text-sm mt-2">
            Sign in to pulse your real-time conversations
          </p>
        </div>

        {error && (
          <div className="mb-6 p-3 bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/20 rounded-xl text-rose-600 dark:text-rose-400 text-xs flex items-center gap-2 animate-fade-in">
            <Shield className="w-4 h-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Form */}
        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label className="block text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-2">
              Email Address
            </label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@example.com"
              className="w-full bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 focus:border-accent rounded-xl px-4 py-3 text-sm text-slate-900 dark:text-slate-100 placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none focus:ring-1 ring-accent transition-all duration-200"
              required
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-2">
              Password
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              className="w-full bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 focus:border-accent rounded-xl px-4 py-3 text-sm text-slate-900 dark:text-slate-100 placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none focus:ring-1 ring-accent transition-all duration-200"
              required
            />
          </div>

          <button
            type="submit"
            disabled={isLoading}
            className="w-full bg-accent-gradient text-white rounded-xl py-3 text-sm font-semibold shadow-lg shadow-accent flex items-center justify-center gap-2 transition-all duration-200 active:scale-[0.98] disabled:opacity-50 disabled:pointer-events-none group hover:opacity-90"
          >
            {isLoading ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <>
                <span>Sign In</span>
                <ArrowRight className="w-4 h-4 transition-transform duration-200 group-hover:translate-x-1" />
              </>
            )}
          </button>
        </form>

        {/* Footer */}
        <div className="mt-8 text-center text-xs text-slate-500 dark:text-slate-500">
          Don't have an account?{' '}
          <Link
            to="/register"
            className="text-accent hover:text-accent-light font-semibold transition-colors duration-200"
          >
            Sign up now
          </Link>
        </div>
      </div>
    </div>
  );
}
