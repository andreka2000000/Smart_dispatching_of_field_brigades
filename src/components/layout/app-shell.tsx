'use client';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import {
  LayoutDashboard,
  Route,
  Settings2,
  Inbox,
  LogOut,
  PanelLeftClose,
  PanelLeftOpen,
  ArrowUpRight,
  ShieldCheck,
} from 'lucide-react';
import { BrandMark } from '@/features/marketing/brand';
import { useIdentity } from '@/features/auth/context';
import { roleNames, type Role } from '@/features/auth/types';
import { request } from '@/shared/api/client';
const navigation = {
  DISPATCHER: [
    { href: '/app/dispatch', label: 'Диспетчерская', icon: LayoutDashboard },
    { href: '/app/requests', label: 'Обращения абонентов', icon: Inbox },
    { href: '/app/admin', label: 'Центр управления', icon: Settings2 },
  ],
  TECHNICIAN: [{ href: '/app/technician', label: 'Мой маршрут', icon: Route }],
  SUBSCRIBER: [{ href: '/app/subscriber', label: 'Мои обращения', icon: Inbox }],
} satisfies Record<Role, { href: string; label: string; icon: typeof Inbox }[]>;
export function AppShell({ children }: { children: React.ReactNode }) {
  const user = useIdentity();
  const path = usePathname();
  const query = useQueryClient();
  const [collapsed, setCollapsed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const compact = collapsed || user.role === 'TECHNICIAN';
  return (
    <div className={`app-shell role-${user.role.toLowerCase()} ${compact ? 'shell-compact' : ''}`}>
      <a className="workspace-skip" href="#workspace-main">
        К содержимому
      </a>
      <aside className="sidebar">
        <Link href="/" className="brand" title="МаксТелеком — на главную">
          <span className="brand-symbol">
            <BrandMark size={26} />
          </span>
          <span className="sidebar-label">
            МаксТелеком<small>Порядок там, где раньше был хаос</small>
          </span>
        </Link>
        <div className="sidebar-section sidebar-label">
          {user.role === 'DISPATCHER'
            ? 'УПРАВЛЕНИЕ СЕРВИСОМ'
            : user.role === 'TECHNICIAN'
              ? 'МОЯ СМЕНА'
              : 'ЛИЧНЫЙ КАБИНЕТ'}
        </div>
        <nav aria-label="Рабочее пространство">
          {navigation[user.role].map(({ href, label, icon: Icon }) => (
            <Link
              key={href}
              href={href}
              aria-label={label}
              title={compact ? label : undefined}
              className={`nav-link ${path === href ? 'active' : ''}`}
              aria-current={path === href ? 'page' : undefined}
            >
              <Icon size={19} />
              <span className={compact ? 'sr-only' : 'sidebar-label'}>{label}</span>
            </Link>
          ))}
        </nav>
        <div className="sidebar-bottom">
          {user.role !== 'TECHNICIAN' && (
            <button
              className="sidebar-collapse"
              onClick={() => setCollapsed(!collapsed)}
              title={compact ? 'Развернуть меню' : 'Свернуть меню'}
              aria-label={compact ? 'Развернуть меню' : 'Свернуть меню'}
            >
              {compact ? <PanelLeftOpen size={18} /> : <PanelLeftClose size={18} />}
              <span className="sidebar-label">Свернуть меню</span>
            </button>
          )}
          <div className="sidebar-account" title={`${user.name} · ${roleNames[user.role]}`}>
            <span className="account-avatar">{user.name.slice(0, 1)}</span>
            <div className="sidebar-label">
              <strong>{user.name}</strong>
              <small>{roleNames[user.role]}</small>
            </div>
          </div>
          <span className="sidebar-session sidebar-label">
            <ShieldCheck size={13} />
            Персональный доступ
          </span>
          <button
            className="sidebar-logout"
            title="Выйти из аккаунта"
            disabled={busy}
            onClick={async () => {
              setBusy(true);
              setError('');
              try {
                await request('', '/auth/v1/session', { method: 'DELETE' });
                query.clear();
                const channel = new BroadcastChannel('maxtelecom-auth');
                channel.postMessage('changed');
                channel.close();
                window.location.replace('/login');
              } catch {
                setError('Не удалось выйти. Повторите.');
                setBusy(false);
              }
            }}
          >
            <LogOut size={17} />
            <span className={compact ? 'sr-only' : 'sidebar-label'}>
              {busy ? 'Выходим…' : 'Выйти'}
            </span>
          </button>
          {error && (
            <p className="form-error" role="alert">
              {error}
            </p>
          )}
          <Link href="/" className="sidebar-home sidebar-label">
            На главную
            <ArrowUpRight size={13} />
          </Link>
        </div>
      </aside>
      <main className="main" id="workspace-main" tabIndex={-1}>
        {children}
      </main>
    </div>
  );
}
