'use client';
import { useState } from 'react';
import { LiveTechnician } from '@/features/workspace/technician';
import { Check, MapPin, WifiOff } from 'lucide-react';
import { PageHeader } from '@/components/layout/page-header';
import { useDemo } from '@/lib/demo-context';
import { DispatchMap } from '@/features/map/map-panel';
import { config } from '@/shared/config';
export function Technician() {
  return config.dataMode === 'api' ? <LiveTechnician /> : <DemoTechnician />;
}
function DemoTechnician() {
  const { orders, updateStatus } = useDemo();
  const [selected, setSelected] = useState<string | null>(null);
  const [offline, setOffline] = useState(false);
  const queue = orders.filter((o) => o.staffId === 'ST-002' && o.status !== 'Выполнена');
  const current = queue.find((o) => o.id === selected) || queue[0];
  if (config.dataMode !== 'mock')
    return (
      <>
        <PageHeader title="Приложение мастера" subtitle="Подключение Go API" />
        <div className="page-content panel">
          Для мобильного интерфейса требуется endpoint текущего сотрудника. Демо-данные отключены.
        </div>
      </>
    );
  return (
    <>
      <PageHeader
        title="Приложение мастера"
        subtitle="Дмитрий Козлов · Renault Kangoo М 508 ОР 77 · пт, 13 ноября, 11:20"
      />
      <div className="page-content">
        <div className="technician-toolbar">
          <span>
            Задач на сегодня: <strong>{queue.length}</strong>
          </span>
          <button
            className="secondary-button"
            role="switch"
            aria-checked={offline}
            onClick={() => setOffline(!offline)}
          >
            <WifiOff size={15} />
            {offline ? 'Нет связи · демо' : 'В сети'}
          </button>
        </div>
        <div className="technician-grid">
          <section>
            <h2>Карта маршрута</h2>
            <DispatchMap orders={queue} onSelect={setSelected} />
            <p className="map-note">Демонстрационная позиция мастера</p>
          </section>
          <section className="panel">
            <h2>Текущая заявка</h2>
            {current ? (
              <>
                <span className={`badge badge-${current.priority}`}>{current.priority}</span>
                <span className="status-pill">{current.status}</span>
                <p className="muted">
                  {current.id} · {current.kind}
                </p>
                <h3>{current.title}</h3>
                <p className="address">
                  <MapPin size={14} />
                  {current.address}
                </p>
                <p>Окно {current.slot}</p>
                <p className="muted">
                  Взять: {current.equipment.join(', ') || 'оборудование не требуется'}
                </p>
                <button
                  className="primary-button"
                  disabled={offline}
                  onClick={() =>
                    updateStatus(
                      current.id,
                      current.status === 'Назначена'
                        ? 'В пути'
                        : current.status === 'В пути'
                          ? 'На объекте'
                          : 'Выполнена',
                    )
                  }
                >
                  <Check size={16} />
                  {current.status === 'Назначена'
                    ? 'Принять заявку'
                    : current.status === 'В пути'
                      ? 'Прибыл на объект'
                      : 'Завершить работу'}
                </button>
                <p className="muted">Изменение статуса действует только в демо.</p>
              </>
            ) : (
              <p>Все заявки выполнены.</p>
            )}
          </section>
        </div>
        <section>
          <h2>Очередь на день</h2>
          <div className="queue">
            {queue.map((o) => (
              <button
                className={`queue-item ${current?.id === o.id ? 'selected' : ''}`}
                key={o.id}
                onClick={() => setSelected(o.id)}
              >
                <strong>{o.title}</strong>
                <span>
                  {o.slot} · {o.address}
                </span>
                <span className="status-pill">{o.status}</span>
              </button>
            ))}
          </div>
        </section>
      </div>
    </>
  );
}
