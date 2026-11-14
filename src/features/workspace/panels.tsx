'use client';
import { useState } from 'react';
import { Clock3, MapPin, ArrowUpRight } from 'lucide-react';
import { Modal } from '@/components/ui/modal';
import { useWorkspace } from './context';
import { countText, initials, staffName, stopNames, time, type ViewOrder } from './model';
import type { Route } from './types';
export function Timeline({
  onSelect,
  staffId,
}: {
  onSelect: (id: string) => void;
  staffId?: string;
}) {
  const d = useWorkspace().data!;
  const routes = d.plan?.routes.filter((r) => !staffId || r.staff_id === staffId) || [];
  const start = Math.min(
    ...d.reference.staff.map((s) => Date.parse(`${d.day.date}T${s.shift.start}:00+05:00`)),
  );
  const end = Math.max(
    ...d.reference.staff.map((s) => Date.parse(`${d.day.date}T${s.shift.end}:00+05:00`)),
  );
  const span = end - start;
  return (
    <section className="live-panel timeline-panel">
      <div className="live-section-title">
        <div>
          <span className="eyebrow">ПЛАН НА ДЕНЬ</span>
          <h2>У каждого — свой маршрут</h2>
        </div>
        <span className="subtle-chip">
          {countText(routes.length, 'сотрудник', 'сотрудника', 'сотрудников')}
        </span>
      </div>
      <div className="timeline-scroll">
        <div className="timeline-inner">
          <div className="timeline-axis">
            <span>Сотрудник</span>
            <div>
              {Array.from({ length: 5 }, (_, i) => (
                <span key={i}>{time(new Date(start + (span * i) / 4).toISOString())}</span>
              ))}
            </div>
          </div>
          {routes.map((r) => {
            const person = d.reference.staff.find((s) => s.staff_id === r.staff_id)!;
            return (
              <div className="gantt-row" key={r.staff_id}>
                <div className="gantt-person">
                  <span className="avatar">{initials(staffName(person))}</span>
                  <span>
                    {staffName(person)}
                    <small>
                      {person.shift.start}–{person.shift.end}
                    </small>
                  </span>
                </div>
                <div className="gantt-track">
                  {r.stops.map((s) => (
                    <button
                      key={s.seq}
                      className={`gantt-stop stop-${s.type}`}
                      style={{
                        left: `${((Date.parse(s.start) - start) / span) * 100}%`,
                        width: `${Math.max(1, ((Date.parse(s.end) - Date.parse(s.start)) / span) * 100)}%`,
                      }}
                      title={`${stopNames[s.type]} ${time(s.start)}–${time(s.end)}${s.order_id ? ' · ' + s.order_id : ''}`}
                      aria-label={`${staffName(person)}: ${stopNames[s.type]} ${time(s.start)}${s.order_id ? ' ' + s.order_id : ''}`}
                      onClick={() => s.order_id && onSelect(s.order_id)}
                    >
                      {s.type === 'VISIT' ? time(s.start) : s.type === 'BREAK' ? 'Перерыв' : ''}
                    </button>
                  ))}
                  {r.stops.length === 0 && (
                    <span className="gantt-empty">
                      {r.status === 'UNAVAILABLE' ? 'Недоступен' : 'Без визитов'}
                    </span>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </div>
      <div className="timeline-legend">
        <span>
          <i className="legend-visit" />
          Визит
        </span>
        <span>
          <i className="legend-break" />
          Перерыв
        </span>
        <span>
          <i className="legend-stock" />
          Пополнение
        </span>
        <span>Время Екатеринбурга · UTC+5</span>
      </div>
    </section>
  );
}
export function AlertsPanel({ onSelect }: { onSelect: (id: string) => void }) {
  const d = useWorkspace().data!;
  const [all, setAll] = useState(false);
  const alerts = d.plan?.alerts.filter((a) => all || a.status === 'ACTIVE') || [];
  return (
    <section className="live-panel">
      <div className="live-section-title">
        <div>
          <span className="eyebrow">В ЦЕНТРЕ ВНИМАНИЯ</span>
          <h2>Предупреждения</h2>
        </div>
        <button className="text-button" onClick={() => setAll(!all)}>
          {all ? 'Активные' : 'Вся история'}
        </button>
      </div>
      <div className="insights-list">
        {alerts.map((a) => (
          <div className="insight-row" key={a.alert_id}>
            <div className="insight-head">
              <span
                className={`state-badge ${a.status === 'RESOLVED' ? 'green' : a.type === 'LATE_RISK' ? 'amber' : 'violet'}`}
              >
                {a.type === 'LATE_RISK'
                  ? 'Риск опоздания'
                  : a.type === 'PGP_DETECTED'
                    ? 'Групповая проблема'
                    : 'Без назначения'}
              </span>
              <small>{time(a.created_at)}</small>
            </div>
            <p>{a.message}</p>
            {a.forecast_start && (
              <small>
                Прогноз до вмешательства: {time(a.forecast_start)} · опоздание {a.lateness_min} мин
              </small>
            )}
            {a.order_id && (
              <button className="text-button" onClick={() => onSelect(a.order_id!)}>
                {a.order_id}
                <ArrowUpRight size={12} />
              </button>
            )}
            <small className="muted">
              {a.status === 'RESOLVED' ? 'Закрыто ' + time(a.resolved_at!) : 'Активно'}
              {a.applied ? ' · действие применено' : ''}
            </small>
          </div>
        ))}
        {!alerts.length && (
          <div className="gentle-empty">Всё спокойно. Активных предупреждений нет.</div>
        )}
      </div>
    </section>
  );
}
export function OrderDetail({
  order,
  onClose,
  onEvent,
}: {
  order: ViewOrder;
  onClose: () => void;
  onEvent: () => void;
}) {
  const d = useWorkspace().data!;
  const person = d.reference.staff.find((s) => s.staff_id === order.staffId);
  return (
    <Modal title={order.source.order_id} onClose={onClose}>
      <div className="detail-content">
        <span className={`state-badge ${order.tone}`}>{order.status}</span>
        <h3>{order.title}</h3>
        <p>
          <MapPin size={15} />
          {order.source.address}
        </p>
        <p>
          <Clock3 size={15} />
          Окно {time(order.source.slot.start)}–{time(order.source.slot.end)}
        </p>
        <dl className="detail-grid">
          <div>
            <dt>Исполнитель</dt>
            <dd>
              {person
                ? staffName(person)
                : order.groupId
                  ? 'Станционная диагностика'
                  : 'Не назначен'}
            </dd>
          </div>
          <div>
            <dt>Технология</dt>
            <dd>{order.source.technology}</dd>
          </div>
          <div>
            <dt>Последнее начало</dt>
            <dd>{time(order.source.latest_start)}</dd>
          </div>
          <div>
            <dt>Контрольный срок</dt>
            <dd>
              {new Intl.DateTimeFormat('ru-RU', {
                day: 'numeric',
                month: 'short',
                hour: '2-digit',
                minute: '2-digit',
                timeZone: 'Asia/Yekaterinburg',
              }).format(new Date(order.source.deadline))}
            </dd>
          </div>
          {order.stop && (
            <>
              <div>
                <dt>Плановое начало</dt>
                <dd>{time(order.stop.start)}</dd>
              </div>
              <div>
                <dt>Плановое окончание</dt>
                <dd>{time(order.stop.end)}</dd>
              </div>
            </>
          )}
        </dl>
        <p className="muted">
          Оборудование:{' '}
          {Object.entries(order.source.required_equipment)
            .map(([k, v]) => `${k}: ${v}`)
            .join(' · ')}
        </p>
        {order.reason && <div className="detail-reason">{order.reason}</div>}
        <button
          className="secondary-button"
          onClick={onEvent}
          disabled={order.status === 'Выполнена'}
          title={order.status === 'Выполнена' ? 'Выполнение уже подтверждено' : undefined}
        >
          Сообщить об изменении
        </button>
      </div>
    </Modal>
  );
}
export function RouteStops({
  route,
  onSelect,
}: {
  route: Route | undefined;
  onSelect: (id: string) => void;
}) {
  const d = useWorkspace().data!;
  return (
    <div className="route-stops">
      {route?.stops.map((s) => {
        const order = d.day.orders.find((o) => o.order_id === s.order_id);
        return (
          <button
            className={`route-stop-card ${s.type === 'VISIT' ? '' : 'quiet-stop'}`}
            key={s.seq}
            onClick={() => s.order_id && onSelect(s.order_id)}
          >
            <span className="stop-seq">{s.seq}</span>
            <div>
              <strong>
                {stopNames[s.type]} · {time(s.start)}–{time(s.end)}
              </strong>
              <p>
                {order?.address ||
                  d.reference.service_centers.find((c) => c.service_center_id === s.location_id)
                    ?.name ||
                  s.location_id}
              </p>
              {order && (
                <small>
                  {order.order_id} · {order.technology}
                </small>
              )}
            </div>
          </button>
        );
      })}
      {!route?.stops.length && (
        <div className="gentle-empty">На этом маршруте пока нет остановок.</div>
      )}
    </div>
  );
}
