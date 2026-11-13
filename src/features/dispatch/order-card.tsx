import { MapPin, Clock } from 'lucide-react';
import type { DemoOrder } from '@/lib/demo-data';
import { staff } from '@/lib/demo-data';
export function OrderCard({
  order,
  selected,
  onSelect,
}: {
  order: DemoOrder;
  selected?: boolean;
  onSelect: () => void;
}) {
  const person = staff.find((s) => s.id === order.staffId);
  return (
    <button
      className={`order-card priority-${order.priority} ${selected ? 'selected' : ''}`}
      onClick={onSelect}
      aria-pressed={selected}
    >
      <div className="card-top">
        <span className="muted truncate">
          {order.id} · {order.kind}
        </span>
        <span className={`badge badge-${order.priority}`}>{order.priority}</span>
      </div>
      <strong className="order-title truncate">{order.title}</strong>
      <span className="address truncate">
        <MapPin size={13} />
        {order.address}
      </span>
      <span className="card-row">
        <span className="muted">Окно {order.slot}</span>
        {order.sla && (
          <span className={order.priority === 'Критичный' ? 'sla danger' : 'sla'}>
            <Clock size={12} />
            SLA: {order.sla}
          </span>
        )}
      </span>
      <span className="card-row">
        <span className="status-pill">{order.status}</span>
        <span className={!person ? 'danger' : 'muted'}>{person?.name || 'Нужен мастер'}</span>
      </span>
    </button>
  );
}
