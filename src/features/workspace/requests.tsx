'use client';
import { useState, useRef } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  ArrowRight,
  Inbox,
  MapPin,
  MessageSquare,
  Phone,
  Plus,
  Send,
  CheckCircle2,
} from 'lucide-react';
import { PageHeader } from '@/components/layout/page-header';
import { useIdentity } from '@/features/auth/context';
import { request } from '@/shared/api/client';
import { ApprovalDialog } from './approval';
import { time } from './model';
type ServiceRequest = {
  id: string;
  owner_name: string;
  service: string;
  address: string;
  phone: string;
  description: string;
  preferred_date: string;
  status: 'NEW' | 'IN_REVIEW' | 'APPROVED' | 'REJECTED' | 'CLOSED';
  order?: {
    id: string;
    date: string;
    state: string;
    slot?: { start: string; end: string };
    staff_name?: string;
    staff_id?: string;
    planned_start?: string;
    reason?: string;
    report?: string;
  };
  decision_note: string;
  created_at: string;
  updated_at: string;
};
const services: Record<string, string> = {
  INTERNET: 'Подключение интернета',
  REPAIR: 'Проверка и ремонт',
  TV: 'Телевидение',
};
const statuses = {
  NEW: 'Поступило',
  IN_REVIEW: 'На рассмотрении',
  APPROVED: 'Одобрено',
  REJECTED: 'Отклонено',
  CLOSED: 'Решено без выезда',
};
const workflow: Record<string, string> = {
  SCHEDULED: 'Исполнитель назначен',
  UNASSIGNED: 'Ожидает назначения',
  IN_PROGRESS: 'Работа начата',
  COMPLETED: 'Работа выполнена',
  CANCELLED: 'Наряд отменён',
  STATION: 'Передано станционной службе',
  REQUIRES_REVIEW: 'Нужно уточнение диспетчера',
};
const created = (value: string) =>
  new Intl.DateTimeFormat('ru-RU', {
    day: 'numeric',
    month: 'long',
    hour: '2-digit',
    minute: '2-digit',
    timeZone: 'Asia/Yekaterinburg',
  }).format(new Date(value));
export function RequestsPage() {
  const user = useIdentity();
  const dispatcher = user.role === 'DISPATCHER';
  const client = useQueryClient();
  const [service, setService] = useState('INTERNET');
  const [filter, setFilter] = useState('ALL');
  const [composing, setComposing] = useState(false);
  const submission = useRef({ signature: '', key: '' });
  const requests = useQuery({
    queryKey: ['service-requests', user.id],
    queryFn: () => request<ServiceRequest[]>('', '/ui/v1/requests'),
    refetchInterval: 15_000,
  });
  const refresh = () => client.invalidateQueries({ queryKey: ['service-requests'] });
  const create = useMutation({
    mutationFn: (input: object) =>
      request<{ id: string }>('', '/ui/v1/requests', {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    onSuccess: () => {
      submission.current = { signature: '', key: '' };
      void refresh();
      setComposing(false);
    },
  });
  const list =
    requests.data?.filter(
      (r) =>
        filter === 'ALL' ||
        r.status === filter ||
        (filter === 'COMPLETED' && r.order?.state === 'COMPLETED'),
    ) || [];
  return (
    <>
      <PageHeader
        title={dispatcher ? 'Обращения абонентов' : 'Мой МаксТелеком'}
        subtitle={
          dispatcher
            ? 'От первого обращения — к согласованной услуге'
            : 'Ваши услуги начинаются с простого обращения'
        }
      />
      <div className="page-content live-content requests-content">
        <section className="request-welcome live-panel">
          <div>
            <span className="eyebrow">
              {dispatcher ? 'ПРИЁМ И СОПРОВОЖДЕНИЕ' : 'ЛИЧНЫЙ КАБИНЕТ'}
            </span>
            <h2>
              {dispatcher ? 'Ни одно обращение не потеряется.' : `Здравствуйте, ${user.name}.`}
            </h2>
            <p>
              {dispatcher
                ? 'Проверьте детали, свяжитесь с абонентом и зафиксируйте следующий шаг.'
                : 'Оставьте заявку на услугу и следите за ответом диспетчера в одном месте.'}
            </p>
          </div>
          {!dispatcher && (
            <button
              className="primary-button"
              onClick={() => {
                create.reset();
                setComposing(!composing);
              }}
            >
              <Plus size={17} />
              Заказать услугу
            </button>
          )}
        </section>
        {!dispatcher && composing && (
          <section className="live-panel request-compose" id="new-request">
            <span className="eyebrow">НОВОЕ ОБРАЩЕНИЕ</span>
            <h2>Чем мы можем помочь?</h2>
            <div className="request-service-options">
              {Object.entries(services).map(([value, label]) => (
                <button
                  key={value}
                  aria-pressed={service === value}
                  onClick={() => setService(value)}
                >
                  <MessageSquare size={18} />
                  {label}
                  <ArrowRight size={15} />
                </button>
              ))}
            </div>
            <form
              className="account-form live-form"
              onSubmit={(event) => {
                event.preventDefault();
                const data = new FormData(event.currentTarget);
                const payload = {
                  service,
                  address: data.get('address'),
                  phone: data.get('phone'),
                  preferred_date: data.get('date'),
                  description: data.get('description'),
                };
                const signature = JSON.stringify(payload);
                if (signature !== submission.current.signature)
                  submission.current = { signature, key: crypto.randomUUID() };
                create.mutate({ ...payload, client_key: submission.current.key });
              }}
            >
              <label>
                Адрес услуги
                <input
                  name="address"
                  required
                  minLength={5}
                  maxLength={300}
                  placeholder="Город, улица, дом, квартира"
                  autoComplete="street-address"
                />
              </label>
              <label>
                Контактный телефон
                <input
                  name="phone"
                  type="tel"
                  autoComplete="tel"
                  required
                  minLength={7}
                  maxLength={40}
                  placeholder="+7 …"
                />
              </label>
              <label>
                Желаемая дата
                <input name="date" type="date" />
              </label>
              <label className="form-wide">
                Опишите задачу
                <textarea
                  name="description"
                  maxLength={2000}
                  rows={3}
                  placeholder="Что нужно подключить или что перестало работать?"
                />
              </label>
              <p className="muted form-wide">
                Диспетчер уточнит детали и согласует время визита. Желаемая дата пока не является
                подтверждённым окном.
              </p>
              <button className="primary-button" disabled={create.isPending}>
                <Send size={15} />
                {create.isPending ? 'Отправляем…' : 'Отправить обращение'}
              </button>
            </form>
          </section>
        )}
        {create.isSuccess && (
          <p role="status" className="operation-notice success">
            <CheckCircle2 size={16} />
            Обращение {create.data.id} принято. Оно уже доступно диспетчеру.
          </p>
        )}
        {(requests.error || create.error) && (
          <p role="alert" className="operation-notice error">
            {(requests.error || create.error)?.message}
          </p>
        )}
        <div className="request-list-heading">
          <h2>
            {dispatcher ? 'Входящие обращения' : 'Мои обращения'}
            <span className="muted"> {requests.data?.length || 0}</span>
          </h2>
          <div className="filter-chips">
            {[
              ['ALL', 'Все'],
              ['NEW', 'Новые'],
              ['IN_REVIEW', 'На рассмотрении'],
              ['APPROVED', 'Одобренные'],
              ['REJECTED', 'Отклонённые'],
              ['COMPLETED', 'Выполненные'],
              ['CLOSED', 'Без выезда'],
            ].map(([id, name]) => (
              <button key={id} aria-pressed={filter === id} onClick={() => setFilter(id)}>
                {name}
              </button>
            ))}
          </div>
        </div>
        {requests.isPending ? (
          <p role="status">Загружаем обращения…</p>
        ) : !list.length ? (
          <section className="live-panel request-empty">
            <Inbox size={36} />
            <h2>
              {dispatcher
                ? 'Здесь появятся обращения абонентов'
                : 'Всё начинается с вашего обращения'}
            </h2>
            <p>
              {dispatcher
                ? 'Новые заявки из личных кабинетов поступают сюда автоматически.'
                : 'Выберите услугу — мы поможем с подключением или решением проблемы.'}
            </p>
            {!dispatcher && (
              <button className="secondary-button" onClick={() => setComposing(true)}>
                Оставить первое обращение
                <ArrowRight size={15} />
              </button>
            )}
          </section>
        ) : (
          <div className="request-cards">
            {list.map((r) => (
              <RequestCard key={r.id} value={r} dispatcher={dispatcher} onSaved={refresh} />
            ))}
          </div>
        )}
      </div>
    </>
  );
}
function RequestCard({
  value: r,
  dispatcher,
  onSaved,
}: {
  value: ServiceRequest;
  dispatcher: boolean;
  onSaved: () => Promise<unknown>;
}) {
  const [approving, setApproving] = useState(false);
  const update = useMutation({
    mutationFn: (input: { status: string; decision_note: string }) =>
      request('', `/ui/v1/requests/${encodeURIComponent(r.id)}`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      }),
    onSuccess: onSaved,
  });
  return (
    <article className="live-panel request-card">
      <div className="request-card-top">
        <span className="eyebrow">{r.id}</span>
        <span
          className={`state-badge ${r.order?.state === 'COMPLETED' || r.status === 'CLOSED' ? 'green' : r.status === 'NEW' ? 'violet' : 'amber'}`}
        >
          {r.order ? workflow[r.order.state] || statuses[r.status] : statuses[r.status]}
        </span>
      </div>
      <h3>{services[r.service]}</h3>
      <p className="request-address">
        <MapPin size={16} />
        {r.address}
      </p>
      {dispatcher && (
        <p className="request-contact">
          <Phone size={14} />
          {r.phone}
          <span>· {r.owner_name}</span>
        </p>
      )}
      {r.description && <p className="request-description">{r.description}</p>}
      {r.preferred_date && <p className="muted">Желаемая дата: {r.preferred_date}</p>}
      {r.decision_note && (
        <div className="request-reply">
          <MessageSquare size={16} />
          <div>
            <strong>Ответ диспетчера</strong>
            <p>{r.decision_note}</p>
          </div>
        </div>
      )}
      <small className="request-created">Поступило {created(r.created_at)}</small>
      {r.order && (
        <div className="request-reply">
          <div>
            <strong>{workflow[r.order.state] || r.order.state}</strong>
            <p>Наряд {r.order.id}</p>
            {r.order.slot && (
              <p>
                Согласовано: {r.order.date}, {time(r.order.slot.start)}–{time(r.order.slot.end)}
              </p>
            )}
            {r.order.staff_name && (
              <p>
                Исполнитель: {r.order.staff_name}
                {dispatcher && r.order.staff_id ? ` · ${r.order.staff_id}` : ''}
              </p>
            )}
            {r.order.planned_start && <p>Начало по плану: {time(r.order.planned_start)}</p>}
            {r.order.reason && <p>{r.order.reason}</p>}
            {r.order.report && <p>Результат: {r.order.report}</p>}
          </div>
        </div>
      )}
      {dispatcher && ['NEW', 'IN_REVIEW'].includes(r.status) && (
        <button
          className="primary-button"
          style={{ marginTop: 20 }}
          onClick={() => setApproving(true)}
        >
          Одобрить и распределить
        </button>
      )}
      {approving && (
        <ApprovalDialog
          id={r.id}
          service={r.service}
          address={r.address}
          preferredDate={r.preferred_date}
          onClose={() => setApproving(false)}
        />
      )}
      {dispatcher && ['NEW', 'IN_REVIEW'].includes(r.status) && (
        <form
          className="request-decision live-form"
          onSubmit={(event) => {
            event.preventDefault();
            const fields = new FormData(event.currentTarget);
            const submitter = (event.nativeEvent as SubmitEvent)
              .submitter as HTMLButtonElement | null;
            update.mutate({
              status: submitter?.value === 'REJECTED' ? 'REJECTED' : 'CLOSED',
              decision_note: String(fields.get('note')),
            });
          }}
        >
          <label>
            Ответ абоненту
            <textarea
              name="note"
              minLength={3}
              maxLength={2000}
              required
              rows={2}
              defaultValue={r.decision_note}
              placeholder="Согласованные действия и дальнейшие шаги"
            />
          </label>
          <div>
            {r.status === 'NEW' && (
              <button
                className="secondary-button"
                type="button"
                disabled={update.isPending}
                onClick={() =>
                  update.mutate({ status: 'IN_REVIEW', decision_note: r.decision_note })
                }
              >
                Взять на рассмотрение
              </button>
            )}
            <button className="primary-button" disabled={update.isPending}>
              Решить без выезда
            </button>
            <button className="secondary-button" value="REJECTED" disabled={update.isPending}>
              Отклонить с объяснением
            </button>
          </div>
          <small className="muted">
            Обработка обращения не означает выполнения выездного наряда.
          </small>
        </form>
      )}
      {update.error && (
        <p role="alert" className="form-error">
          {update.error.message}
        </p>
      )}
    </article>
  );
}
