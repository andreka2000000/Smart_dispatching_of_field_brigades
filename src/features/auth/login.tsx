'use client';
import { useState } from 'react';
import Link from 'next/link';
import { ArrowRight, LockKeyhole, ShieldCheck } from 'lucide-react';
import { BrandMark } from '@/features/marketing/brand';
import { request } from '@/shared/api/client';
import { roleHome, type User } from './types';
import './login.css';
export function Login() {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  return (
    <main className="login-page">
      <section className="login-story">
        <Link href="/" className="login-brand">
          <BrandMark size={34} />
          МаксТелеком
        </Link>
        <div>
          <span className="eyebrow">ОДНА СИСТЕМА. СВОЯ ЗАДАЧА У КАЖДОГО.</span>
          <h1>
            Порядок там,
            <br />
            где раньше
            <br />
            <em>был хаос.</em>
          </h1>
          <p>
            Общая картина для диспетчера.
            <br />
            Чёткий маршрут для исполнителя.
            <br />
            Понятный сервис для абонента.
          </p>
        </div>
        <span className="login-foot">
          <ShieldCheck size={16} />
          Персональный доступ к вашему рабочему пространству
        </span>
      </section>
      <section className="login-card">
        <span className="login-lock">
          <LockKeyhole size={23} />
        </span>
        <span className="eyebrow">С ВОЗВРАЩЕНИЕМ</span>
        <h2>Войти в систему</h2>
        <p>
          Используйте свой аккаунт. Рабочее пространство откроется в соответствии с вашими правами.
        </p>
        <form
          onSubmit={async (event) => {
            event.preventDefault();
            setBusy(true);
            setError('');
            const fields = new FormData(event.currentTarget);
            try {
              const user = await request<User>('', '/auth/v1/session', {
                method: 'POST',
                body: JSON.stringify({
                  username: String(fields.get('username')),
                  password: String(fields.get('password')),
                }),
              });
              const channel = new BroadcastChannel('maxtelecom-auth');
              channel.postMessage('changed');
              channel.close();
              window.location.replace(roleHome[user.role]);
            } catch (err) {
              setError(err instanceof Error ? err.message : 'Не удалось войти.');
              setBusy(false);
            }
          }}
        >
          <label>
            Логин
            <input
              name="username"
              autoComplete="username"
              required
              autoFocus
              maxLength={64}
              placeholder="Ваш логин"
            />
          </label>
          <label>
            Пароль
            <input
              name="password"
              type="password"
              autoComplete="current-password"
              required
              maxLength={256}
              placeholder="Ваш пароль"
            />
          </label>
          {error && (
            <p className="form-error" role="alert">
              {error}
            </p>
          )}
          <button className="primary-button" disabled={busy}>
            {busy ? 'Проверяем данные…' : 'Войти'}
            <ArrowRight size={16} />
          </button>
        </form>
        <small>Нет аккаунта? Обратитесь к диспетчеру МаксТелеком.</small>
        <Link href="/" className="login-back">
          Вернуться на главную
        </Link>
      </section>
    </main>
  );
}
