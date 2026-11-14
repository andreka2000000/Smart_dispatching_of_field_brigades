'use client';
import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Plus, ShieldCheck, UserRound } from 'lucide-react';
import { request } from '@/shared/api/client';
import { useIdentity } from '@/features/auth/context';
import { roleNames, type Role, type User } from '@/features/auth/types';
import { useWorkspace } from './context';
import { staffName } from './model';
export function AccountsPanel() {
  const identity = useIdentity();
  const client = useQueryClient();
  const data = useWorkspace().data!;
  const [open, setOpen] = useState(false);
  const [role, setRole] = useState<Role>('TECHNICIAN');
  const users = useQuery({
    queryKey: ['accounts', identity.id],
    queryFn: () => request<User[]>('', '/ui/v1/users'),
  });
  const create = useMutation({
    mutationFn: (input: object) =>
      request('', '/ui/v1/users', { method: 'POST', body: JSON.stringify(input) }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['accounts'] });
      setOpen(false);
    },
  });
  const access = useMutation({
    mutationFn: (u: User) =>
      request('', `/ui/v1/users/${encodeURIComponent(u.id)}`, {
        method: 'PATCH',
        body: JSON.stringify({ disabled: !u.disabled }),
      }),
    onSuccess: () => void client.invalidateQueries({ queryKey: ['accounts'] }),
  });
  return (
    <>
      <div className="live-section-title">
        <div>
          <span className="eyebrow">АККАУНТЫ И ПОЛНОМОЧИЯ</span>
          <h2>Каждому — свой доступ</h2>
        </div>
        <button
          className="primary-button"
          onClick={() => {
            create.reset();
            setOpen(!open);
          }}
        >
          <Plus size={15} />
          Добавить аккаунт
        </button>
      </div>
      <div className="access-role-cards">
        {(['DISPATCHER', 'TECHNICIAN', 'SUBSCRIBER'] as Role[]).map((r) => (
          <article key={r}>
            <ShieldCheck size={18} />
            <strong>{roleNames[r]}</strong>
            <span>
              {users.data?.filter((u) => u.role === r && !u.disabled).length ?? '—'} активных
            </span>
            <small>
              {r === 'DISPATCHER'
                ? 'Вся смена, обращения, команда и управление доступом'
                : r === 'TECHNICIAN'
                  ? 'Свой маршрут, задания и сообщения об изменениях'
                  : 'Создание и просмотр своих обращений'}
            </small>
          </article>
        ))}
      </div>
      {open && (
        <form
          className="account-form live-form"
          onSubmit={(event) => {
            event.preventDefault();
            const fields = new FormData(event.currentTarget);
            create.mutate({
              name: fields.get('name'),
              username: fields.get('username'),
              password: fields.get('password'),
              role,
              staff_id: role === 'TECHNICIAN' ? fields.get('staff') : '',
            });
          }}
        >
          <label>
            Имя
            <input name="name" required minLength={2} maxLength={100} autoComplete="off" />
          </label>
          <label>
            Логин
            <input
              name="username"
              required
              pattern="[a-z0-9][a-z0-9._-]{2,63}"
              autoComplete="off"
              placeholder="Например, master.ivanov"
            />
          </label>
          <label>
            Роль
            <select value={role} onChange={(e) => setRole(e.target.value as Role)}>
              {Object.entries(roleNames).map(([value, name]) => (
                <option key={value} value={value}>
                  {name}
                </option>
              ))}
            </select>
          </label>
          <label>
            Первоначальный пароль
            <input
              type="password"
              name="password"
              required
              minLength={12}
              maxLength={256}
              autoComplete="new-password"
            />
          </label>
          {role === 'TECHNICIAN' && (
            <label>
              Сотрудник в справочнике
              <select name="staff" required>
                <option value="">Выберите сотрудника</option>
                {data.reference.staff
                  .filter((s) => !users.data?.some((u) => !u.disabled && u.staff_id === s.staff_id))
                  .map((s) => (
                    <option key={s.staff_id} value={s.staff_id}>
                      {staffName(s)} · {s.staff_id}
                    </option>
                  ))}
              </select>
            </label>
          )}
          <button className="primary-button" disabled={create.isPending}>
            {create.isPending ? 'Создаём…' : 'Создать аккаунт'}
          </button>
        </form>
      )}
      {(create.error || access.error || users.error) && (
        <p role="alert" className="form-error">
          {(create.error || access.error || users.error)?.message}
        </p>
      )}
      {users.isPending ? (
        <p role="status">Загружаем аккаунты…</p>
      ) : (
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Пользователь</th>
                <th>Роль</th>
                <th>Сотрудник</th>
                <th>Доступ</th>
                <th>
                  <span className="sr-only">Действие</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {users.data?.map((u) => (
                <tr key={u.id}>
                  <td>
                    <span className="table-person">
                      <UserRound size={17} />
                      <span>
                        {u.name}
                        <small>
                          {u.username}
                          {u.id === identity.id ? ' · это вы' : ''}
                        </small>
                      </span>
                    </span>
                  </td>
                  <td>{roleNames[u.role]}</td>
                  <td>{u.staff_id || '—'}</td>
                  <td>
                    <span className={`state-badge ${u.disabled ? 'amber' : 'green'}`}>
                      {u.disabled ? 'Отключён' : 'Активен'}
                    </span>
                  </td>
                  <td>
                    <button
                      className="text-action"
                      disabled={u.id === identity.id || access.isPending}
                      onClick={() => access.mutate(u)}
                    >
                      {u.disabled ? 'Восстановить' : 'Отключить'}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}
