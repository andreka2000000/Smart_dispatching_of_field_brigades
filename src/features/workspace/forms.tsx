'use client';
import { useRef, useState } from 'react';
import { Modal } from '@/components/ui/modal';
import { useWorkspace } from './context';
import { eventNames, errorText, fromLocal, localInput, staffName, time } from './model';
import type { Event, WorkOrder } from './types';
import { useIdentity } from '@/features/auth/context';
function useEventSender() {
  const w = useWorkspace();
  const previous = useRef({ signature: '', id: '' });
  return async (payload: Omit<Event, 'event_id'>) => {
    const signature = JSON.stringify(payload);
    if (signature !== previous.current.signature)
      previous.current = { signature, id: crypto.randomUUID() };
    await w.event.mutateAsync({ ...payload, event_id: previous.current.id } as Event);
  };
}
export function EventDialog({
  onClose,
  orderId,
  staffId,
  kind = 'TIME_TICK',
}: {
  onClose: () => void;
  orderId?: string;
  staffId?: string;
  kind?: string;
}) {
  const w = useWorkspace();
  const d = w.data!;
  const user = useIdentity();
  const send = useEventSender();
  const [type, setType] = useState(kind);
  const [order, setOrder] = useState(orderId || d.day.orders[0]?.order_id || '');
  const [staff, setStaff] = useState(staffId || d.reference.staff[0]?.staff_id || '');
  const [at, setAt] = useState(localInput(d.day.now));
  const [permanent, setPermanent] = useState(true);
  const [error, setError] = useState('');
  const stop = d.plan?.routes.flatMap((r) => r.stops).find((s) => s.order_id === order);
  const needsOrder = ['ORDER_CANCELLED', 'VISIT_DELAYED', 'ORDER_TRANSFERRED'].includes(type);
  const needsStaff = ['VISIT_DELAYED', 'ORDER_TRANSFERRED', 'STAFF_UNAVAILABLE'].includes(type);
  return (
    <Modal title="Изменение в течение смены" onClose={onClose}>
      <form
        className="form-stack live-form"
        onSubmit={async (e) => {
          e.preventDefault();
          setError('');
          const f = new FormData(e.currentTarget);
          const payload: Record<string, unknown> = {
            type,
            timestamp: fromLocal(String(f.get('timestamp'))),
          };
          if (needsOrder) payload.order_id = order;
          if (needsStaff) payload.staff_id = staff;
          if (['ORDER_CANCELLED', 'STAFF_UNAVAILABLE'].includes(type))
            payload.reason = String(f.get('reason'));
          if (type === 'VISIT_DELAYED') payload.expected_end = fromLocal(String(f.get('end')));
          if (type === 'ORDER_TRANSFERRED') {
            payload.actual_end = fromLocal(String(f.get('end')));
            payload.transferred_to = String(f.get('target'));
          }
          if (type === 'STAFF_UNAVAILABLE')
            payload.until = permanent ? null : fromLocal(String(f.get('end')));
          try {
            await send(payload as Omit<Event, 'event_id'>);
            onClose();
          } catch (err) {
            setError(errorText(err));
          }
        }}
      >
        <p className="muted">
          Сейчас в смене {time(d.day.now)}. Изменение будет проверено и сохранено вместе с новой
          версией плана.
        </p>
        <label>
          Тип изменения
          <select value={type} onChange={(e) => setType(e.target.value)}>
            {Object.entries(eventNames)
              .filter(([id]) =>
                user.role === 'TECHNICIAN'
                  ? ['VISIT_DELAYED', 'ORDER_TRANSFERRED', 'STAFF_UNAVAILABLE'].includes(id)
                  : id !== 'NEW_ORDER',
              )
              .map(([id, label]) => (
                <option value={id} key={id}>
                  {label}
                </option>
              ))}
          </select>
        </label>
        <label>
          Время изменения
          <input
            type="datetime-local"
            name="timestamp"
            step="1"
            required
            value={at}
            readOnly={user.role === 'TECHNICIAN'}
            onChange={(e) => setAt(e.target.value)}
          />
        </label>
        {needsOrder && (
          <label>
            Заявка
            <select
              value={order}
              required
              onChange={(e) => {
                setOrder(e.target.value);
                const route = d.plan?.routes.find((r) =>
                  r.stops.some((s) => s.order_id === e.target.value),
                );
                if (route) setStaff(route.staff_id);
              }}
            >
              {d.day.orders.map((o) => (
                <option key={o.order_id} value={o.order_id}>
                  {o.order_id} · {o.address}
                </option>
              ))}
            </select>
          </label>
        )}
        {needsStaff && (
          <label>
            Сотрудник
            <select
              value={staff}
              required
              disabled={user.role === 'TECHNICIAN'}
              onChange={(e) => setStaff(e.target.value)}
            >
              {d.reference.staff.map((s) => (
                <option value={s.staff_id} key={s.staff_id}>
                  {staffName(s)}
                </option>
              ))}
            </select>
          </label>
        )}
        {type === 'ORDER_CANCELLED' && (
          <label>
            Причина
            <select name="reason">
              <option value="CUSTOMER_REQUEST">Отказ абонента</option>
              <option value="NOT_CONFIRMED">Визит не подтверждён</option>
            </select>
          </label>
        )}
        {type === 'STAFF_UNAVAILABLE' && (
          <>
            <label>
              Причина
              <select name="reason">
                <option value="VEHICLE_BREAKDOWN">Неисправность автомобиля</option>
                <option value="ILLNESS">Болезнь</option>
                <option value="OTHER">Другая причина</option>
              </select>
            </label>
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={permanent}
                onChange={(e) => setPermanent(e.target.checked)}
              />
              До конца смены
            </label>
          </>
        )}
        {(type === 'VISIT_DELAYED' ||
          type === 'ORDER_TRANSFERRED' ||
          (type === 'STAFF_UNAVAILABLE' && !permanent)) && (
          <label>
            {type === 'VISIT_DELAYED'
              ? 'Ожидаемое окончание визита'
              : type === 'ORDER_TRANSFERRED'
                ? 'Фактическое окончание визита'
                : 'Доступен с'}
            <input
              key={type + order + at}
              type="datetime-local"
              step="1"
              name="end"
              required
              defaultValue={
                type === 'ORDER_TRANSFERRED'
                  ? at
                  : localInput(
                      new Date(
                        Date.parse(type === 'VISIT_DELAYED' && stop ? stop.end : fromLocal(at)) +
                          30 * 60_000,
                      ).toISOString(),
                    )
              }
            />
          </label>
        )}
        {type === 'ORDER_TRANSFERRED' && (
          <label>
            Подразделение
            <select name="target">
              <option value="STATION_EQUIPMENT">Станционное оборудование</option>
              <option value="TRUNK_CABLE">Магистральный кабель</option>
            </select>
          </label>
        )}
        {error && (
          <p role="alert" className="form-error">
            {error}
          </p>
        )}
        <button className="primary-button" disabled={w.busy} type="submit">
          {w.busy ? 'Сохраняем…' : 'Применить изменение'}
        </button>
      </form>
    </Modal>
  );
}
export function NewOrderDialog({ onClose }: { onClose: () => void }) {
  const w = useWorkspace();
  const d = w.data!;
  const send = useEventSender();
  const [work, setWork] = useState(d.reference.work_types[0].code);
  const choices = d.reference.access_nodes.flatMap((n) =>
    n.buildings.map((b) => ({ node: n, building: b, key: n.access_node_id + '|' + b.building_id })),
  );
  const [place, setPlace] = useState(choices[0]?.key || '');
  const [at, setAt] = useState(localInput(d.day.now));
  const [error, setError] = useState('');
  const [id] = useState(() => {
    const prefix = 'ORD-' + d.day.date.replaceAll('-', '') + '-';
    const max = Math.max(
      0,
      ...d.day.orders
        .filter((o) => o.order_id.startsWith(prefix))
        .map((o) => Number(o.order_id.slice(-4))),
    );
    return prefix + String(max + 1).padStart(4, '0');
  });
  const type = d.reference.work_types.find((x) => x.code === work)!;
  const selected = choices.find((c) => c.key === place);
  const defaultDeadline =
    type.deadline_kind === 'HOURS'
      ? localInput(
          new Date(
            Date.parse(fromLocal(at)) + (type.deadline_value || 24) * 3600_000,
          ).toISOString(),
        )
      : type.deadline_kind === 'SLOT'
        ? `${d.day.date}T18:00:00`
        : '';
  return (
    <Modal title="Новая заявка на выезд" onClose={onClose}>
      <form
        className="form-stack live-form"
        onSubmit={async (e) => {
          e.preventDefault();
          setError('');
          if (!selected) return;
          const f = new FormData(e.currentTarget);
          const registeredAt = fromLocal(String(f.get('registered_at')));
          const start = fromLocal(String(f.get('start')));
          const end = fromLocal(String(f.get('finish')));
          const latest =
            type.max_start_delay_min == null
              ? end
              : new Date(
                  Math.min(Date.parse(end), Date.parse(start) + type.max_start_delay_min * 60_000),
                ).toISOString();
          const b = selected.building;
          const order: WorkOrder = {
            order_id: id,
            work_type_code: work,
            technology: selected.node.technology,
            building_id: b.building_id,
            address: b.address,
            lat: b.lat,
            lon: b.lon,
            building_type: b.building_type,
            requires_height: f.get('height') === 'on',
            access_node_id: selected.node.access_node_id,
            registered_at: registeredAt,
            slot: { start, end },
            latest_start: latest,
            deadline: fromLocal(String(f.get('deadline'))),
            required_equipment: {
              ONT: Number(f.get('ONT')),
              ROUTER: Number(f.get('ROUTER')),
              STB: Number(f.get('STB')),
            },
            is_repeat: f.get('repeat') === 'on',
            previous_order_id: null,
          };
          try {
            await send({ type: 'NEW_ORDER', timestamp: registeredAt, order } as Omit<
              Event,
              'event_id'
            >);
            onClose();
          } catch (err) {
            setError(errorText(err));
          }
        }}
      >
        <label>
          Вид работ
          <select value={work} onChange={(e) => setWork(e.target.value as typeof work)}>
            {d.reference.work_types.map((t) => (
              <option key={t.code} value={t.code}>
                {t.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          Адрес и узел подключения
          <select value={place} required onChange={(e) => setPlace(e.target.value)}>
            {choices.map((c) => (
              <option key={c.key} value={c.key}>
                {c.building.address} · {c.node.technology}
              </option>
            ))}
          </select>
        </label>
        <label>
          Время регистрации
          <input
            type="datetime-local"
            name="registered_at"
            step="1"
            required
            value={at}
            onChange={(e) => setAt(e.target.value)}
          />
        </label>
        <div className="form-columns">
          <label>
            Начало окна
            <input
              name="start"
              type="datetime-local"
              required
              defaultValue={`${d.day.date}T14:00`}
            />
          </label>
          <label>
            Конец окна
            <input
              name="finish"
              type="datetime-local"
              required
              defaultValue={`${d.day.date}T16:00`}
            />
          </label>
        </div>
        <label>
          Контрольный срок
          <input
            key={work}
            name="deadline"
            type="datetime-local"
            required
            defaultValue={defaultDeadline}
          />
          <small className="muted">Для срока в рабочих днях укажите дату из обращения.</small>
        </label>
        <div className="form-columns equipment-inputs">
          {['ONT', 'ROUTER', 'STB'].map((k) => (
            <label key={k}>
              {k}
              <input type="number" min="0" max="100" name={k} defaultValue={0} required />
            </label>
          ))}
        </div>
        <label className="checkbox-label">
          <input
            key={place}
            type="checkbox"
            name="height"
            defaultChecked={selected?.building.overhead_line}
          />
          Нужен допуск к работам на высоте
        </label>
        <label className="checkbox-label">
          <input type="checkbox" name="repeat" />
          Повторное обращение
        </label>
        {error && (
          <p role="alert" className="form-error">
            {error}
          </p>
        )}
        <button className="primary-button" disabled={w.busy} type="submit">
          {w.busy ? 'Сохраняем и планируем…' : 'Создать и распределить'}
        </button>
      </form>
    </Modal>
  );
}
