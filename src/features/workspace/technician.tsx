'use client';
import { useMemo, useState } from 'react';
import { Clock3, MapPin, Route as RouteIcon } from 'lucide-react';
import { PageHeader } from '@/components/layout/page-header';
import { OperationNotice, useWorkspace, WorkspaceGate } from './context';
import { countText, locationPoints, ordersView, profileNames, staffName, time } from './model';
import { LiveMap } from './map';
import { OrderDetail, RouteStops, Timeline } from './panels';
import { EventDialog } from './forms';
import { RoadPanel } from './road-routing';
import { useIdentity } from '@/features/auth/context';
import { ExecutionPanel } from './execution';
export function LiveTechnician() {
  return (
    <>
      <PageHeader title="Маршрут мастера" subtitle="Адрес, время и следующий шаг" />
      <WorkspaceGate>
        <TechnicianContent />
      </WorkspaceGate>
    </>
  );
}
function TechnicianContent() {
  const w = useWorkspace();
  const d = w.data!;
  const user = useIdentity();
  const [selected, setSelected] = useState<string | null>(null);
  const [event, setEvent] = useState(false);
  const staffId = user.staff_id;
  const person = d.reference.staff.find((s) => s.staff_id === staffId)!;
  const route = d.plan?.routes.find((r) => r.staff_id === staffId);
  const all = useMemo(() => ordersView(d), [d]);
  const orders = all.filter((o) => o.staffId === staffId);
  const points = useMemo(() => locationPoints(d), [d]);
  const currentStop = route?.stops.find(
    (s) => s.type === 'VISIT' && Date.parse(s.end) > Date.parse(d.day.now),
  );
  const current = orders.find((o) => o.source.order_id === currentStop?.order_id);
  const detail = all.find((o) => o.source.order_id === selected);
  if (!person)
    return (
      <div className="page-content">
        <section className="live-panel state-panel">
          <h2>Профиль сотрудника не найден в этой смене</h2>
          <p>Попросите диспетчера проверить привязку аккаунта к справочнику.</p>
        </section>
      </div>
    );
  return (
    <div className="page-content live-content">
      <div className="shift-toolbar">
        <span className="eyebrow">МОЙ МАРШРУТ · {staffId}</span>
        <span className="subtle-chip">
          {time(d.day.now)} · версия {d.plan?.plan_version || '—'}
        </span>
      </div>
      <OperationNotice />
      <section className="technician-hero live-panel">
        <span className="eyebrow">НА СВЯЗИ С КОМАНДОЙ</span>
        <h2>{staffName(person)}</h2>
        <p>
          {person.profiles.map((p) => profileNames[p]).join(' · ')} ·{' '}
          {person.technologies.join(' / ')}
        </p>
        <div className="technician-stats">
          <span>
            <RouteIcon size={17} />
            {countText(route?.totals?.visits || 0, 'визит', 'визита', 'визитов')}
          </span>
          <span>
            <Clock3 size={17} />
            {person.shift.start}–{person.shift.end}
          </span>
          <span>
            <MapPin size={17} />
            {
              d.reference.service_centers.find(
                (c) => c.service_center_id === person.service_center_id,
              )?.name
            }
          </span>
        </div>
      </section>
      <div className="technician-grid">
        <div className="technician-route-column">
          <RoadPanel route={route} />
          <section className="live-panel">
            <div className="live-section-title">
              <div>
                <span className="eyebrow">БЛИЖАЙШИЙ ВИЗИТ</span>
                <h2>{current?.title || 'Нет предстоящих визитов'}</h2>
              </div>
            </div>
            {current && (
              <>
                <p className="next-address">{current.source.address}</p>
                <span className="state-badge violet">
                  {time(current.stop!.start)}–{time(current.stop!.end)}
                </span>
                <p className="equipment-note">
                  Взять с собой:{' '}
                  {Object.entries(current.source.required_equipment)
                    .filter(([, n]) => n > 0)
                    .map(([k, n]) => `${k} × ${n}`)
                    .join(' · ') || 'Дополнительное оборудование не требуется'}
                </p>
                <button
                  className="primary-button"
                  onClick={() => setSelected(current.source.order_id)}
                >
                  Детали визита
                </button>
              </>
            )}
            <h3>Остановки смены</h3>
            <RouteStops route={route} onSelect={setSelected} />
            <button
              className="secondary-button"
              disabled={w.busy || !d.plan}
              onClick={() => {
                w.event.reset();
                setEvent(true);
              }}
            >
              Сообщить об изменении
            </button>
          </section>
        </div>
        <LiveMap
          orders={orders}
          points={points}
          route={route}
          selected={selected}
          onSelect={setSelected}
        />
      </div>
      <ExecutionPanel />
      <section id="schedule">
        <Timeline staffId={staffId} onSelect={setSelected} />
      </section>
      {detail && !event && (
        <OrderDetail
          order={detail}
          onClose={() => setSelected(null)}
          onEvent={() => {
            w.event.reset();
            setEvent(true);
          }}
        />
      )}
      {event && (
        <EventDialog
          kind={detail ? 'VISIT_DELAYED' : 'STAFF_UNAVAILABLE'}
          orderId={detail?.source.order_id}
          staffId={staffId}
          onClose={() => setEvent(false)}
        />
      )}
    </div>
  );
}
