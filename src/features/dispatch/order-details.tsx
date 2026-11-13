'use client';
import { useState } from 'react';
import { Modal } from '@/components/ui/modal';
import { staff, type DemoOrder } from '@/lib/demo-data';
import { useDemo } from '@/lib/demo-context';
export function OrderDetails({ order, onClose }: { order: DemoOrder; onClose: () => void }) {
  const { assign } = useDemo();
  const [message, setMessage] = useState('');
  return (
    <Modal title={order.title} onClose={onClose}>
      <p className="muted">
        {order.id} · {order.kind}
      </p>
      <div className="detail-tags">
        <span className={`badge badge-${order.priority}`}>{order.priority}</span>
        <span className="status-pill">{order.status}</span>
        <span className="sla">SLA: {order.sla || 'выполнена'}</span>
      </div>
      <h3>Адрес и окно визита</h3>
      <p>{order.address}</p>
      <p>Окно визита: {order.slot}</p>
      <h3>Задача</h3>
      <p>Навыки: {order.skills.join(', ')}</p>
      <p>Оборудование: {order.equipment.join(', ') || 'не требуется'}</p>
      <h3>Подбор мастера</h3>
      <p className="muted">
        Демо: проверяются только навыки и статус сети. Маршрут, смену и АО проверит Go API.
      </p>
      <div className="candidate-list">
        {[...staff]
          .sort(
            (a, b) =>
              Number(order.skills.every((s) => b.skills.includes(s))) -
              Number(order.skills.every((s) => a.skills.includes(s))),
          )
          .map((person) => {
            const missing = order.skills.filter((s) => !person.skills.includes(s));
            const eligible = missing.length === 0 && person.status !== 'Не в сети';
            return (
              <div className="candidate" key={person.id}>
                <span className="avatar">{person.initials}</span>
                <div>
                  <strong>{person.name}</strong>
                  <p className="muted">
                    {person.skills.join(' · ')} · {person.status}
                  </p>
                  {missing.length > 0 && (
                    <p className="danger">Не хватает навыков: {missing.join(', ')}</p>
                  )}
                </div>
                <button
                  className="primary-button"
                  disabled={!eligible}
                  onClick={() => {
                    assign(order.id, person.id);
                    setMessage(`Назначено: ${person.name} · демо`);
                  }}
                >
                  Назначить
                </button>
              </div>
            );
          })}
      </div>
      {message && (
        <p role="status" className="success">
          {message}
        </p>
      )}
    </Modal>
  );
}
