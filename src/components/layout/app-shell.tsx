'use client';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { LayoutDashboard, Smartphone, SlidersHorizontal, Route } from 'lucide-react';
import { config } from '@/shared/config';
const links = [
  { href: '/app/dispatch', label: 'Диспетчерская', icon: LayoutDashboard },
  { href: '/app/technician', label: 'Приложение мастера', icon: Smartphone },
  { href: '/app/admin', label: 'Админ-панель', icon: SlidersHorizontal },
];
export function AppShell({ children }: { children: React.ReactNode }) {
  const path = usePathname();
  return (
    <div className="app-shell">
      <aside className="sidebar">
        <Link href="/" className="brand">
          <span className="brand-symbol">
            <Route size={23} />
          </span>
          <span>
            Магистраль<small>Выездные бригады</small>
          </span>
        </Link>
        <nav aria-label="Основная навигация">
          {links.map(({ href, label, icon: Icon }) => (
            <Link
              key={href}
              href={href}
              aria-current={path === href ? 'page' : undefined}
              className={path === href ? 'nav-link active' : 'nav-link'}
            >
              <Icon size={18} />
              {label}
            </Link>
          ))}
        </nav>
        <div className="demo-notice">
          <strong>{config.dataMode === 'mock' ? 'Демо-режим' : 'Подключён Go API'}</strong>
          <p>
            {config.dataMode === 'mock'
              ? 'Данные тестовые. Изменения сохраняются до перезагрузки страницы.'
              : 'Назначения проверяются сервером.'}
          </p>
        </div>
      </aside>
      <main className="main">{children}</main>
    </div>
  );
}
