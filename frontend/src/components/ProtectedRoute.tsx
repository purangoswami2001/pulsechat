import React, { useEffect, useState } from 'react';
import { Navigate } from 'react-router-dom';
import { Loader2 } from 'lucide-react';
import { api, redirectToLoginUnavailable } from '../services/api';

interface ProtectedRouteProps {
  children: React.ReactNode;
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ children }) => {
  const token = localStorage.getItem('pulsechat_token');
  const [checking, setChecking] = useState(!!token);
  const [serverOk, setServerOk] = useState(true);

  useEffect(() => {
    if (!token) return;

    let cancelled = false;
    api
      .pingServer()
      .then(() => {
        if (!cancelled) setServerOk(true);
      })
      .catch(() => {
        if (cancelled) return;
        setServerOk(false);
        redirectToLoginUnavailable();
      })
      .finally(() => {
        if (!cancelled) setChecking(false);
      });

    return () => {
      cancelled = true;
    };
  }, [token]);

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  if (checking) {
    return (
      <div className="h-screen flex items-center justify-center bg-slate-50 dark:bg-slate-950">
        <Loader2 className="w-8 h-8 animate-spin text-accent" />
      </div>
    );
  }

  if (!serverOk) {
    return null;
  }

  return <>{children}</>;
};

export default ProtectedRoute;
