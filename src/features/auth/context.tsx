'use client';
import { createContext, useContext, useEffect } from 'react';
import type { User } from './types';
import { roleHome } from './types';
const Identity = createContext<User | null>(null);
export function AuthProvider({ user, children }: { user: User; children: React.ReactNode }) {
  useEffect(() => {
    let alive = true;
    const check = async () => {
      try {
        const response = await fetch('/auth/v1/session', { cache: 'no-store' });
        if (alive && response.status === 401) window.location.replace('/login');
        else if (alive && response.ok) {
          const next = (await response.json()) as User;
          if (next.id !== user.id || next.role !== user.role)
            window.location.replace(roleHome[next.role]);
        }
      } catch {
        /* Keep the offline view; API still enforces session expiry. */
      }
    };
    const interval = setInterval(() => void check(), 60_000);
    const channel = new BroadcastChannel('maxtelecom-auth');
    channel.onmessage = () => window.location.replace('/app');
    const focused = () => void check();
    window.addEventListener('focus', focused);
    return () => {
      alive = false;
      clearInterval(interval);
      channel.close();
      window.removeEventListener('focus', focused);
    };
  }, [user.id, user.role]);
  return <Identity.Provider value={user}>{children}</Identity.Provider>;
}
export function useIdentity() {
  const user = useContext(Identity);
  if (!user) throw new Error('AuthProvider is missing');
  return user;
}
