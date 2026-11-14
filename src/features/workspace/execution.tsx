'use client';
import { useMutation } from '@tanstack/react-query';
import { useWorkspace } from './context';
import { request } from '@/shared/api/client';
import { ordersView, time } from './model';
export function ExecutionPanel() {
  const w = useWorkspace();
  const d = w.data!;
  const orders = ordersView(d).filter((o) => o.stop);
  const save = useMutation({
    mutationFn: (input: { id: string; action: string; report: string }) =>
      request('', `/ui/v1/days/${d.day.date}/orders/${encodeURIComponent(input.id)}/execution`, {
        method: 'POST',
        body: JSON.stringify({ action: input.action, report: input.report }),
      }),
    onSuccess: () => w.refresh(),
  });
  return (
    <section className="live-panel">
      <span className="eyebrow">ФАКТИЧЕСКОЕ ВЫПОЛНЕНИЕ</span>
      <h2>Мои задания</h2>
      <p className="muted">
        Подтверждайте начало и результат работы. Модельное время смены: {time(d.day.now)}.
      </p>
      {orders.map((o) => {
        const fact = d.facts?.find((f) => f.order_id === o.source.order_id);
        const started = fact?.status === 'IN_PROGRESS';
        const done = fact?.status === 'COMPLETED';
        const ready = Date.parse(d.day.now) >= Date.parse(o.stop!.start);
        return (
          <form
            className="request-decision live-form"
            key={o.source.order_id}
            onSubmit={(e) => {
              e.preventDefault();
              const f = new FormData(e.currentTarget);
              save.mutate({
                id: o.source.order_id,
                action: started ? 'COMPLETE' : 'START',
                report: String(f.get('report') || ''),
              });
            }}
          >
            <strong>{o.source.address}</strong>
            <small>
              {o.source.order_id} · {time(o.stop!.start)}–{time(o.stop!.end)} · {o.status}
            </small>
            {done ? (
              <p>{fact.report}</p>
            ) : (
              <>
                {started && (
                  <label>
                    Результат работы
                    <textarea name="report" required minLength={3} maxLength={2000} rows={2} />
                  </label>
                )}
                <button className="primary-button" disabled={save.isPending || !ready}>
                  {started ? 'Подтвердить выполнение' : 'Начать работу'}
                </button>
                {!ready && (
                  <small className="muted">
                    Доступно после начала визита по модельному времени.
                  </small>
                )}
              </>
            )}
          </form>
        );
      })}
      {!orders.length && <p className="muted">Назначенных заданий пока нет.</p>}
      {save.error && (
        <p className="form-error" role="alert">
          {save.error.message}
        </p>
      )}
    </section>
  );
}
