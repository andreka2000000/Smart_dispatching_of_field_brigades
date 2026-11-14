'use client';
import { useState } from 'react';
import { LiveDispatcher } from '@/features/workspace/dispatcher';
import {
  Search,
  Plus,
  ClipboardList,
  TriangleAlert,
  Users,
  CircleCheck,
  SlidersHorizontal,
} from 'lucide-react';
import { PageHeader } from '@/components/layout/page-header';
import { Modal } from '@/components/ui/modal';
import { useDemo } from '@/lib/demo-context';
import { staff, type DemoOrder } from '@/lib/demo-data';
import { config } from '@/shared/config';
import { OrderCard } from './order-card';
import { OrderDetails } from './order-details';
import { DispatchMap } from '@/features/map/map-panel';

export function Dispatcher() {
  return config.dataMode === 'api' ? <LiveDispatcher /> : <DemoDispatcher />;
}
function DemoDispatcher() {
  const { orders, activity, addOrder } = useDemo();
  const [search, setSearch] = useState('');
  const [region, setRegion] = useState('Все регионы');
  const [tab, setTab] = useState('Все');
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [newOrder, setNewOrder] = useState(false);
  const [selectedStaff, setSelectedStaff] = useState<string | null>(null);
  const selected = orders.find((o) => o.id === selectedId);
  const visible = orders.filter(
    (o) =>
      (!selectedStaff || o.staffId === selectedStaff) &&
      (region === 'Все регионы' || region === 'Москва') &&
      `${o.id} ${o.title} ${o.address}`.toLowerCase().includes(search.toLowerCase()) &&
      (tab === 'Все' ||
        (tab === 'Новые'
          ? o.status === 'Новая'
          : tab === 'Готово'
            ? o.status === 'Выполнена'
            : o.status !== 'Новая' && o.status !== 'Выполнена')),
  );
  const metrics = [
    {
      label: 'Ждут назначения',
      value: orders.filter((o) => o.status === 'Новая').length,
      note: `из ${orders.filter((o) => o.status !== 'Выполнена').length} открытых`,
      Icon: ClipboardList,
      tone: 'neutral',
    },
    {
      label: 'SLA под угрозой',
      value: orders.filter((o) => o.priority === 'Критичный' && o.status !== 'Выполнена').length,
      note: 'менее 30 минут или просрочено',
      Icon: TriangleAlert,
      tone: 'red',
    },
    {
      label: 'Мастера свободны',
      value: staff.filter((s) => s.status === 'Свободен').length,
      note: '7 в сети из 8',
      Icon: Users,
      tone: 'green',
    },
    {
      label: 'Выполнено сегодня',
      value: orders.filter((o) => o.status === 'Выполнена').length,
      note: 'тестовые данные',
      Icon: CircleCheck,
      tone: 'blue',
    },
  ];
  return (
    <>
      <PageHeader
        title="Диспетчерская"
        subtitle="Назначайте мастеров и следите за SLA · пт, 13 ноября, 11:20"
      >
        <label className="search-box">
          <Search size={16} />
          <input
            aria-label="Поиск по заявкам"
            placeholder="Поиск по № заявки, адресу, клиенту"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </label>
        <select aria-label="Регион" value={region} onChange={(e) => setRegion(e.target.value)}>
          {['Все регионы', 'Москва', 'Московская область', 'Санкт-Петербург'].map((r) => (
            <option key={r}>{r}</option>
          ))}
        </select>
      </PageHeader>
      <div className="page-content">
        <div className="toolbar">
          <span className="sr-only">Сводка</span>
          {selectedStaff && (
            <button className="secondary-button" onClick={() => setSelectedStaff(null)}>
              Сбросить фильтр мастера
            </button>
          )}
          <button className="primary-button" onClick={() => setNewOrder(true)}>
            <Plus size={16} />
            Новая заявка
          </button>
        </div>
        <section className="summary-grid" aria-label="Сводка">
          {metrics.map(({ label, value, note, Icon, tone }) => (
            <div className="metric-card" key={label}>
              <span className={`metric-icon ${tone}`}>
                <Icon size={23} />
              </span>
              <div>
                <strong className="metric-value">{value}</strong>
                <span className="metric-label">{label}</span>
                <small className="muted truncate">{note}</small>
              </div>
            </div>
          ))}
        </section>
        <div className="dispatch-grid">
          <section aria-label="Заявки">
            <h2>
              Заявки <span className="muted">({visible.length})</span>
            </h2>
            <div className="tabs" role="tablist" aria-label="Статус заявок">
              {['Все', 'Новые', 'В работе', 'Готово'].map((t) => (
                <button key={t} role="tab" aria-selected={tab === t} onClick={() => setTab(t)}>
                  {t}
                </button>
              ))}
            </div>
            <div className="order-list">
              {visible.map((order) => (
                <OrderCard
                  key={order.id}
                  order={order}
                  selected={selectedId === order.id}
                  onSelect={() => setSelectedId(order.id)}
                />
              ))}
              {visible.length === 0 && <div className="empty-state">Заявки не найдены</div>}
            </div>
          </section>
          <section aria-label="Карта">
            <h2>Карта</h2>
            <DispatchMap orders={visible} onSelect={setSelectedId} />
            <p className="map-note">Тестовые позиции · без GPS-трекинга</p>
          </section>
          <section className="team-column" aria-label="Мастера и события">
            <h2>Мастера</h2>
            <div className="staff-list">
              {staff.map((person) => (
                <button
                  key={person.id}
                  className={`staff-card ${selectedStaff === person.id ? 'selected' : ''}`}
                  aria-pressed={selectedStaff === person.id}
                  onClick={() => setSelectedStaff(selectedStaff === person.id ? null : person.id)}
                >
                  <span className="avatar">{person.initials}</span>
                  <span className="staff-copy">
                    <span className="staff-top">
                      <strong>{person.name}</strong>
                      <span className="staff-status">
                        <i
                          className={`dot ${person.status === 'Свободен' ? 'green-dot' : person.status === 'Не в сети' ? 'gray-dot' : 'orange-dot'}`}
                        />
                        {person.status}
                      </span>
                    </span>
                    <small className="muted truncate">
                      {person.skills.join(' · ')}
                      {person.available && ` · свободен с ${person.available}`}
                    </small>
                    <progress
                      max="100"
                      value={person.load}
                      aria-label={`Загрузка ${person.name}`}
                      className={person.load >= 65 ? 'busy' : ''}
                    />
                  </span>
                </button>
              ))}
            </div>
            <div className="activity">
              <h2>Лента событий</h2>
              <ul>
                {activity.map((event, i) => (
                  <li key={`${event.text}-${i}`}>
                    <span className="activity-dot" />
                    <span>
                      {event.text}
                      <small>{event.time}</small>
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          </section>
        </div>
      </div>
      {selected && <OrderDetails order={selected} onClose={() => setSelectedId(null)} />}
      {newOrder && (
        <Modal title="Новая заявка" onClose={() => setNewOrder(false)}>
          <form
            className="form-stack"
            onSubmit={(e) => {
              e.preventDefault();
              const form = new FormData(e.currentTarget);
              const order: DemoOrder = {
                id: `ДЕМО-${orders.length + 1}`,
                title: String(form.get('title')),
                address: String(form.get('address')),
                kind: 'Нет интернета',
                priority: 'Обычный',
                status: 'Новая',
                slot: '14:00–16:00',
                sla: '4 ч',
                skills: ['Оптика'],
                equipment: ['ONT'],
                coordinates: [37.6, 55.75],
              };
              addOrder(order);
              setNewOrder(false);
            }}
          >
            <label>
              Название
              <input name="title" required maxLength={140} />
            </label>
            <label>
              Адрес
              <input name="address" required maxLength={240} />
            </label>
            <p className="muted">
              Тестовая заявка с окном 14:00–16:00. Адрес не геокодируется; позиция на карте
              условная.
            </p>
            <button className="primary-button" type="submit">
              <SlidersHorizontal size={16} />
              Создать в демо
            </button>
          </form>
        </Modal>
      )}
    </>
  );
}
