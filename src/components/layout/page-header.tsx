'use client';
import { useState } from 'react';
import { Bell } from 'lucide-react';
export function PageHeader({
  title,
  subtitle,
  children,
}: {
  title: string;
  subtitle: string;
  children?: React.ReactNode;
}) {
  const [notifications, setNotifications] = useState(false);
  return (
    <header className="page-header">
      <div className="heading-block">
        <h1>{title}</h1>
        <p>{subtitle}</p>
      </div>
      <div className="header-controls">
        {children}
        <button
          className="icon-button notification-button"
          aria-label="Уведомления"
          aria-expanded={notifications}
          onClick={() => setNotifications(!notifications)}
        >
          <Bell size={19} />
          <i />
        </button>
        <div className="user">
          <span className="avatar">НО</span>
          <span>
            Надежда Орлова<small>Диспетчер</small>
          </span>
        </div>
        {notifications && (
          <div className="notifications" role="status">
            <strong>Уведомления</strong>
            <p>Новые события отображаются в ленте диспетчерской.</p>
          </div>
        )}
      </div>
    </header>
  );
}
