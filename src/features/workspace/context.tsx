'use client';
import { createContext, useContext, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { request } from '@/shared/api/client';
import { config } from '@/shared/config';
import { dispatchApi } from '@/features/dispatch/api';
import type { DaySummary, Event, EventResult, Snapshot } from './types';
import { errorText } from './model';
import { useIdentity } from '@/features/auth/context';
function useWorkspaceState() {
  const user = useIdentity();
  const qc = useQueryClient();
  const [chosen, setChosen] = useState('');
  const days = useQuery({
    queryKey: ['workspace-days', user.id],
    queryFn: ({ signal }) => request<DaySummary[]>(config.apiBaseUrl, '/ui/v1/days', { signal }),
    refetchInterval: 15_000,
  });
  const date = chosen || days.data?.[0]?.date || '';
  const snapshot = useQuery({
    queryKey: ['workspace', date, user.id],
    queryFn: ({ signal }) =>
      request<Snapshot>(config.apiBaseUrl, `/ui/v1/days/${date}/workspace`, { signal }),
    enabled: !!date,
    refetchInterval: 10_000,
  });
  const refresh = async () => {
    await Promise.all([
      qc.invalidateQueries({ queryKey: ['workspace', date] }),
      qc.invalidateQueries({ queryKey: ['workspace-days'] }),
    ]);
  };
  const event = useMutation({
    mutationFn: (body: Event) => dispatchApi.postEvent(date, body),
    onSettled: refresh,
  });
  const build = useMutation({
    mutationFn: () => dispatchApi.buildPlan(date, 30),
    onSettled: refresh,
  });
  return {
    days,
    date,
    setDate: (value: string) => {
      event.reset();
      build.reset();
      setChosen(value);
    },
    snapshot,
    data: snapshot.data,
    refresh,
    event,
    build,
    busy: event.isPending || build.isPending,
  };
}
type WorkspaceContext = ReturnType<typeof useWorkspaceState>;
const Context = createContext<WorkspaceContext | null>(null);
export function WorkspaceProvider({ children }: { children: React.ReactNode }) {
  const state = useWorkspaceState();
  return <Context.Provider value={state}>{children}</Context.Provider>;
}
export function useOptionalWorkspace() {
  return useContext(Context);
}
export function useWorkspace() {
  const value = useContext(Context);
  if (!value) throw new Error('WorkspaceProvider is missing');
  return value;
}
export function WorkspaceGate({ children }: { children: React.ReactNode }) {
  const w = useWorkspace();
  const error = w.days.error || w.snapshot.error;
  if (!w.data) {
    return (
      <div className="page-content">
        <section className="live-panel state-panel" role={error ? 'alert' : 'status'}>
          <span className="eyebrow">МАКСТЕЛЕКОМ / РАБОЧЕЕ ПРОСТРАНСТВО</span>
          <h2>
            {error
              ? 'Не удалось загрузить смену'
              : w.days.isPending || (w.snapshot.isPending && !!w.date)
                ? 'Загружаем смену…'
                : 'Нет загруженных смен'}
          </h2>
          <p>
            {error
              ? errorText(error)
              : 'После загрузки данных здесь появятся заявки, команда и расписание.'}
          </p>
          <button className="secondary-button" onClick={() => void w.refresh()}>
            Обновить данные
          </button>
        </section>
      </div>
    );
  }
  return (
    <>
      {error && (
        <div role="alert" className="connection-warning">
          Связь прервана. На экране последняя загруженная версия. {errorText(error)}
        </div>
      )}
      {children}
    </>
  );
}
export function OperationNotice() {
  const w = useWorkspace();
  const error = w.event.error || w.build.error;
  const result = w.event.data as EventResult | undefined;
  if (error)
    return (
      <div role="alert" className="operation-notice error">
        {errorText(error)}{' '}
        <button
          onClick={() => {
            w.event.reset();
            w.build.reset();
          }}
        >
          Скрыть
        </button>
      </div>
    );
  if (w.busy)
    return (
      <div role="status" className="operation-notice">
        Сохраняем изменения и обновляем план…
      </div>
    );
  if (result)
    return (
      <div role="status" className="operation-notice success">
        {result.duplicate ? 'Событие уже было обработано' : 'Изменение принято'} · версия{' '}
        {result.plan_version}
      </div>
    );
  return null;
}
