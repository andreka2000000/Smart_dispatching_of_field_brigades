'use client';
import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { CalendarPlus } from 'lucide-react';
import { request } from '@/shared/api/client';
import { Modal } from '@/components/ui/modal';
import { useWorkspace } from './context';
export function NewShift() {
  const w = useWorkspace();
  const router = useRouter();
  const client = useQueryClient();
  const [open, setOpen] = useState(false);
  const save = useMutation({
    mutationFn: (date: string) =>
      request<{ date: string }>('', '/ui/v1/days', {
        method: 'POST',
        body: JSON.stringify({ date }),
      }),
    onSuccess: async (day) => {
      w.setDate(day.date);
      await client.invalidateQueries({ queryKey: ['workspace-days'] });
      setOpen(false);
      router.push('/app/dispatch');
    },
  });
  const next = w.date
    ? new Date(Date.parse(w.date + 'T00:00:00Z') + 86400000).toISOString().slice(0, 10)
    : '';
  return (
    <>
      <button
        className="secondary-button"
        onClick={() => {
          save.reset();
          setOpen(true);
        }}
      >
        <CalendarPlus size={16} />
        Новая смена
      </button>
      {open && (
        <Modal title="Создать рабочую смену" onClose={() => setOpen(false)}>
          <form
            className="form-stack live-form"
            onSubmit={(e) => {
              e.preventDefault();
              save.mutate(String(new FormData(e.currentTarget).get('date')));
            }}
          >
            <p>
              Новая смена использует текущих сотрудников, сервисные центры и дорожную матрицу.
              Обращения в неё добавляются после согласования.
            </p>
            <label>
              Дата смены
              <input name="date" type="date" required defaultValue={next} />
            </label>
            <p className="muted">
              Начальное модельное время — 08:30. После создания нажмите «Построить план» в
              диспетчерской. Существующая смена на выбранную дату не будет заменена.
            </p>
            {save.error && (
              <p className="form-error" role="alert">
                {save.error.message}
              </p>
            )}
            <button className="primary-button" disabled={save.isPending}>
              {save.isPending ? 'Создаём…' : 'Создать смену'}
            </button>
          </form>
        </Modal>
      )}
    </>
  );
}
