'use client';
import { useMemo, useState } from 'react';
import {
  Activity,
  ArrowUpRight,
  ClipboardList,
  MapPin,
  Plus,
  Route as RouteIcon,
  Search,
  Users,
  Waypoints,
} from 'lucide-react';
import { PageHeader } from '@/components/layout/page-header';
import { OperationNotice, useWorkspace, WorkspaceGate } from './context';
import {
  countText,
  eventNames,
  initials,
  locationPoints,
  ordersView,
  staffName,
  time,
} from './model';
import { LiveMap } from './map';
import { AlertsPanel, OrderDetail, Timeline } from './panels';
import { EventDialog, NewOrderDialog } from './forms';
import { RoadPanel } from './road-routing';
export function LiveDispatcher() {
  return (
    <>
      <PageHeader title="Диспетчерская" subtitle="Вся смена — в одной картине" />
      <WorkspaceGate>
        <DispatcherContent />
      </WorkspaceGate>
    </>
  );
}
function DispatcherContent() {
  const w = useWorkspace();
  const d = w.data!;
  const orders = useMemo(() => ordersView(d), [d]);
  const points = useMemo(() => locationPoints(d), [d]);
  const [search, setSearch] = useState('');
  const [filter, setFilter] = useState('Все');
  const [staffId, setStaffId] = useState('');
  const [selected, setSelected] = useState<string | null>(null);
  const [dialog, setDialog] = useState<'event' | 'new' | null>(null);
  const selectedOrder = orders.find((o) => o.source.order_id === selected);
  const visible = orders.filter(
    (o) =>
      (!staffId || o.staffId === staffId) &&
      `${o.source.order_id} ${o.source.address} ${o.title}`
        .toLowerCase()
        .includes(search.toLowerCase()) &&
      (filter === 'Все' ||
        (filter === 'Без назначения' && o.status === 'Без назначения') ||
        (filter === 'ПГП' && !!o.groupId) ||
        (filter === 'В работе' &&
          ['Назначена', 'На объекте', 'Окно визита идёт', 'Ожидает подтверждения'].includes(
            o.status,
          )) ||
        (filter === 'Завершены' && ['Выполнена', 'Отменена'].includes(o.status))),
  );
  const route = d.plan?.routes.find((r) => r.staff_id === (staffId || selectedOrder?.staffId));
  const active = d.plan?.alerts.filter((a) => a.status === 'ACTIVE') || [];
  const optimization = d.plan?.optimization;
  const metrics = [
    {
      label: 'Заявок в смене',
      value: d.day.orders.length,
      note: 'Все обращения дня',
      Icon: ClipboardList,
      tone: 'violet',
    },
    {
      label: 'Без назначения',
      value: d.plan?.unassigned.length ?? d.day.orders.length,
      note: 'Требуют внимания',
      Icon: Users,
      tone: 'amber',
    },
    {
      label: 'Активных рисков',
      value: active.filter((a) => a.type === 'LATE_RISK').length,
      note: 'Прогноз до вмешательства',
      Icon: Activity,
      tone: 'red',
    },
    {
      label: 'Групповых проблем',
      value: d.plan?.pgp_groups.length || 0,
      note: `${d.plan?.summary?.pgp_orders || 0} обращений в ПГП`,
      Icon: Waypoints,
      tone: 'violet',
    },
  ];
  return (
    <div className="page-content live-content">
      <div className="shift-toolbar">
        <div className="shift-state">
          <span className="status-light" />
          <span>
            Смена {time(d.day.now)}
            <small>
              {d.plan
                ? `Версия ${d.plan.plan_version} · ${countText(d.event_count, 'событие', 'события', 'событий')}`
                : 'Первоначальный план ещё не построен'}
            </small>
          </span>
        </div>
        <div className="shift-actions">
          <button
            className="secondary-button"
            disabled={w.busy || d.event_count > 0}
            title={d.event_count ? 'После событий план обновляется через изменения' : undefined}
            onClick={() => {
              w.event.reset();
              void w.build.mutateAsync().catch(() => {});
            }}
          >
            <RouteIcon size={15} />
            {w.build.isPending ? 'Рассчитываем…' : 'Построить план'}
          </button>
          <button
            className="secondary-button"
            disabled={w.busy || !d.plan}
            onClick={() => {
              w.event.reset();
              setDialog('event');
            }}
          >
            Добавить событие
          </button>
          <button
            className="primary-button"
            disabled={w.busy || !d.plan}
            onClick={() => {
              w.event.reset();
              setDialog('new');
            }}
          >
            <Plus size={16} />
            Новая заявка
          </button>
        </div>
      </div>
      <OperationNotice />
      {optimization && (
        <p className="plan-hint" role="status">
          {optimization.status === 'improved'
            ? `План улучшен: ${optimization.selected.visits} визитов вместо ${optimization.baseline.visits}. Переезды: ${Math.round(optimization.baseline.travel_sec / 60)} → ${Math.round(optimization.selected.travel_sec / 60)} мин.`
            : optimization.status === 'unavailable'
              ? 'Оптимизатор временно недоступен. План построен резервным алгоритмом.'
              : optimization.status === 'short_budget'
                ? 'План построен в быстром режиме.'
                : 'Сохранён базовый план: оптимизатор не нашёл улучшения.'}
        </p>
      )}
      {d.event_count > 0 && (
        <p className="plan-hint">
          Первоначальный план построен. Дальнейшие изменения пересчитывают его через события.
        </p>
      )}
      <section className="live-metrics" aria-label="Сводка смены">
        {metrics.map(({ label, value, note, Icon, tone }) => (
          <div className="live-metric" key={label}>
            <div className="metric-top">
              <span>{label}</span>
              <Icon size={17} className={tone} />
            </div>
            <strong>{value.toString().padStart(2, '0')}</strong>
            <small>{note}</small>
          </div>
        ))}
      </section>
      <div className="live-dispatch-grid">
        <section className="live-panel jobs-panel">
          <div className="live-section-title">
            <div>
              <span className="eyebrow">ОБРАЩЕНИЯ</span>
              <h2>
                Заявки <span className="muted">{visible.length}</span>
              </h2>
            </div>
            <ArrowUpRight size={17} />
          </div>
          <label className="live-search">
            <Search size={16} />
            <input
              aria-label="Поиск заявок"
              placeholder="Номер, адрес или услуга"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </label>
          <div className="filter-chips">
            {['Все', 'Без назначения', 'В работе', 'ПГП', 'Завершены'].map((x) => (
              <button aria-pressed={filter === x} key={x} onClick={() => setFilter(x)}>
                {x}
              </button>
            ))}
          </div>
          <div className="live-orders">
            {visible.map((o) => (
              <button
                className={`live-order ${selected === o.source.order_id ? 'selected' : ''}`}
                key={o.source.order_id}
                onClick={() => setSelected(o.source.order_id)}
                aria-pressed={selected === o.source.order_id}
              >
                <div>
                  <small>{o.source.order_id}</small>
                  <span className={`state-badge ${o.tone}`}>{o.status}</span>
                </div>
                <strong>{o.title}</strong>
                <p>
                  <MapPin size={13} />
                  {o.source.address}
                </p>
                <footer>
                  <span>
                    {time(o.source.slot.start)}–{time(o.source.slot.end)}
                  </span>
                  <span>
                    {d.reference.staff.find((s) => s.staff_id === o.staffId)?.display_name ||
                      o.staffId ||
                      (o.status === 'Отменена'
                        ? 'Выезд отменён'
                        : o.groupId
                          ? 'Станционная диагностика'
                          : 'Нужен мастер')}
                  </span>
                </footer>
              </button>
            ))}
            {!visible.length && (
              <div className="gentle-empty">По выбранным условиям заявок нет.</div>
            )}
          </div>
        </section>
        <section className="map-panel">
          <div className="map-title">
            <span className="eyebrow">СВЯЗАНЫ ОДНИМ МАРШРУТОМ</span>
            <h2>Карта выездов</h2>
          </div>
          <LiveMap
            orders={visible}
            points={points}
            route={route}
            selected={selected}
            onSelect={setSelected}
          />
        </section>
        <section className="live-panel team-panel">
          <div className="live-section-title">
            <div>
              <span className="eyebrow">ЛЮДИ</span>
              <h2>Команда</h2>
            </div>
            <Users size={17} />
          </div>
          {staffId && (
            <button className="text-button" onClick={() => setStaffId('')}>
              Все сотрудники
            </button>
          )}
          <div className="live-team">
            {d.reference.staff.map((s) => {
              const r = d.plan?.routes.find((r) => r.staff_id === s.staff_id);
              const absent = d.day.absent_staff_ids?.includes(s.staff_id);
              const state = absent
                ? 'Не на смене'
                : r?.status === 'UNAVAILABLE'
                  ? 'Недоступен'
                  : r?.stops.some(
                        (x) =>
                          x.type === 'VISIT' &&
                          Date.parse(x.start) <= Date.parse(d.day.now) &&
                          Date.parse(x.end) > Date.parse(d.day.now),
                      )
                    ? 'На объекте'
                    : 'На смене';
              return (
                <button
                  className={`live-person ${staffId === s.staff_id ? 'selected' : ''}`}
                  key={s.staff_id}
                  onClick={() => setStaffId(staffId === s.staff_id ? '' : s.staff_id)}
                  aria-pressed={staffId === s.staff_id}
                >
                  <span className="avatar">{initials(staffName(s))}</span>
                  <div>
                    <strong>{staffName(s)}</strong>
                    <small>
                      {state} · {countText(r?.totals?.visits || 0, 'визит', 'визита', 'визитов')}
                    </small>
                    <progress
                      max="100"
                      value={Math.min(100, (r?.totals?.utilization || 0) * 100)}
                      aria-label={`Загрузка ${staffName(s)}`}
                    />
                  </div>
                </button>
              );
            })}
          </div>
          <div className="live-journal">
            <span className="eyebrow">ЛЕНТА СМЕНЫ</span>
            {d.events
              .slice(-5)
              .reverse()
              .map((e) => (
                <div key={e.event_id}>
                  <i />
                  <p>
                    {eventNames[e.type]}
                    <small>{time(e.timestamp)}</small>
                  </p>
                </div>
              ))}
            {!d.events.length && <p className="muted">Изменений пока нет.</p>}
          </div>
        </section>
      </div>
      {route && <RoadPanel route={route} />}
      <Timeline onSelect={setSelected} />
      <div className="lower-grid">
        <AlertsPanel onSelect={setSelected} />
        <section className="live-panel">
          <div className="live-section-title">
            <div>
              <span className="eyebrow">ОБЩАЯ ПРИЧИНА</span>
              <h2>Групповые проблемы</h2>
            </div>
            <Waypoints size={18} />
          </div>
          {d.plan?.pgp_groups.map((g) => (
            <div className="pgp-card" key={g.group_id}>
              <span className="state-badge violet">{g.order_ids.length} обращений</span>
              <h3>
                {d.reference.access_nodes.find((n) => n.access_node_id === g.access_node_id)
                  ?.name || g.access_node_id}
              </h3>
              <p>Станционная диагностика · обнаружено {time(g.detected_at)}</p>
              <div>
                {g.order_ids.map((id) => (
                  <button className="text-button" key={id} onClick={() => setSelected(id)}>
                    {id}
                    <ArrowUpRight size={11} />
                  </button>
                ))}
              </div>
            </div>
          ))}
          {!d.plan?.pgp_groups.length && (
            <div className="gentle-empty">Обращения не указывают на групповую проблему.</div>
          )}
        </section>
      </div>
      {selectedOrder && !dialog && (
        <OrderDetail
          order={selectedOrder}
          onClose={() => setSelected(null)}
          onEvent={() => {
            w.event.reset();
            setDialog('event');
          }}
        />
      )}
      {dialog === 'event' && (
        <EventDialog
          orderId={selectedOrder?.source.order_id}
          staffId={selectedOrder?.staffId || staffId}
          kind={selectedOrder ? 'ORDER_CANCELLED' : 'TIME_TICK'}
          onClose={() => setDialog(null)}
        />
      )}
      {dialog === 'new' && <NewOrderDialog onClose={() => setDialog(null)} />}
    </div>
  );
}
