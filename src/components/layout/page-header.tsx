'use client';
import { useState } from 'react';
import { Bell, CalendarDays, RefreshCw } from 'lucide-react';
import { useOptionalWorkspace } from '@/features/workspace/context';
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
  const w = useOptionalWorkspace();
  const active = w?.data?.plan?.alerts.filter((a) => a.status === 'ACTIVE') || [];
  return (
    <header className="page-header">
      <div className="heading-block">
        <h1>{title}</h1>
        <p>{subtitle}</p>
      </div>
      <div className="header-controls">
        {children}
        {w && (
          <>
            <label className="day-picker">
              <CalendarDays size={15} />
              <span className="sr-only">День смены</span>
              <select
                aria-label="День смены"
                value={w.date}
                disabled={w.busy || !w.days.data?.length}
                onChange={(e) => w.setDate(e.target.value)}
              >
                {!w.days.data?.length && <option value="">Нет смен</option>}
                {w.days.data?.map((d) => (
                  <option key={d.date} value={d.date}>
                    {new Intl.DateTimeFormat('ru-RU', {
                      day: 'numeric',
                      month: 'long',
                      year: 'numeric',
                      timeZone: 'UTC',
                    }).format(new Date(d.date + 'T00:00:00Z'))}
                  </option>
                ))}
              </select>
            </label>
            <button
              className="icon-button"
              aria-label="Обновить смену"
              disabled={w.busy}
              onClick={() => void w.refresh()}
            >
              <RefreshCw size={17} />
            </button>
          </>
        )}
        {w && (
          <button
            className="icon-button notification-button"
            aria-label="Уведомления"
            aria-expanded={notifications}
            onClick={() => setNotifications(!notifications)}
          >
            <Bell size={19} />
            {active.length > 0 && <i />}
          </button>
        )}
        {w && notifications && (
          <div className="notifications" role="status">
            <strong>Уведомления</strong>
            {active.length ? (
              active.slice(0, 5).map((a) => <p key={a.alert_id}>{a.message}</p>)
            ) : (
              <p>Активных предупреждений нет.</p>
            )}
          </div>
        )}
      </div>
    </header>
  );
}
