'use client';
import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { PageHeader } from '@/components/layout/page-header';
import { request } from '@/shared/api/client';
import { config } from '@/shared/config';
import { useWorkspace, WorkspaceGate } from './context';
import { countText, initials, profileNames, staffName, time } from './model';
import { AccountsPanel } from './accounts';
import { NewShift } from './new-shift';
import { Activity, Building2, ClipboardCheck, Users } from 'lucide-react';
export function LiveAdmin() {
  return (
    <>
      <PageHeader
        title="Центр управления"
        subtitle="Команда, инфраструктура и доступ — под вашим контролем"
      >
        <NewShift />
      </PageHeader>
      <WorkspaceGate>
        <AdminContent />
      </WorkspaceGate>
    </>
  );
}
function AdminContent() {
  const d = useWorkspace().data!;
  const [tab, setTab] = useState('Обзор');
  const [search, setSearch] = useState('');
  const health = useQuery({
    queryKey: ['health'],
    queryFn: ({ signal }) =>
      request<{ status: string; team: string; version: string }>(
        config.apiBaseUrl,
        '/api/v1/health',
        { signal },
      ),
    refetchInterval: 15_000,
  });
  return (
    <div className="page-content live-content">
      <div className="admin-intro">
        <span className="eyebrow">МАКСТЕЛЕКОМ / КОНТРОЛЬНАЯ ТОЧКА</span>
        <h2>Всё, что держит сервис в порядке.</h2>
        <p>Единое пространство диспетчера: от команды и ресурсов до персональных прав доступа.</p>
      </div>
      <div className="control-metrics">
        {[
          {
            label: 'Команда на смене',
            value: d.reference.staff.length - (d.day.absent_staff_ids?.length ?? 0),
            note: `из ${d.reference.staff.length} сотрудников`,
            Icon: Users,
          },
          {
            label: 'Визитов в плане',
            value: d.plan?.summary?.visits_planned ?? 0,
            note: `${d.day.orders.length} заявок в дне`,
            Icon: ClipboardCheck,
          },
          {
            label: 'Сервисных центров',
            value: d.reference.service_centers.length,
            note: 'выезд и пополнение оборудования',
            Icon: Building2,
          },
          {
            label: 'Сервис планирования',
            value: health.data ? 'На связи' : 'Проверяем',
            note: health.data ? `версия ${health.data.version}` : 'ожидаем подтверждения API',
            Icon: Activity,
          },
        ].map(({ label, value, note, Icon }) => (
          <article key={label}>
            <div>
              <span>{label}</span>
              <Icon size={18} />
            </div>
            <strong>{value}</strong>
            <small>{note}</small>
          </article>
        ))}
      </div>
      <div className="filter-chips admin-filters">
        {['Обзор', 'Команда', 'Нормативы', 'Сервисные центры', 'Доступы', 'Платформа'].map((x) => (
          <button key={x} aria-pressed={tab === x} onClick={() => setTab(x)}>
            {x}
          </button>
        ))}
      </div>
      <section className="live-panel admin-panel">
        {tab === 'Доступы' && <AccountsPanel />}
        {tab === 'Обзор' && (
          <>
            <div className="live-section-title">
              <div>
                <span className="eyebrow">КОМАНДА НА КАРТЕ ГОРОДА</span>
                <h2>Опорные точки сервиса</h2>
              </div>
              <span className="subtle-chip">Текущая смена · {time(d.day.now)}</span>
            </div>
            <div className="control-centers">
              {d.reference.service_centers.map((center) => {
                const staff = d.reference.staff.filter(
                  (s) => s.service_center_id === center.service_center_id,
                );
                const ids = new Set(staff.map((s) => s.staff_id));
                const visits =
                  d.plan?.routes
                    .filter((r) => ids.has(r.staff_id))
                    .reduce((n, r) => n + (r.totals?.visits || 0), 0) || 0;
                return (
                  <article key={center.service_center_id}>
                    <Building2 size={20} />
                    <span className="eyebrow">{center.service_center_id}</span>
                    <h3>{center.name}</h3>
                    <p>{center.address}</p>
                    <div className="center-numbers">
                      <span>
                        <strong>{staff.length}</strong>сотрудников
                      </span>
                      <span>
                        <strong>{visits}</strong>визитов
                      </span>
                    </div>
                    <span
                      className="center-distribution"
                      aria-label={`Доля визитов: ${Math.round((visits / Math.max(1, d.plan?.summary?.visits_planned || 1)) * 100)}%`}
                    >
                      <i
                        style={{
                          width: `${(visits / Math.max(1, d.plan?.summary?.visits_planned || 1)) * 100}%`,
                        }}
                      />
                    </span>
                    <small>Доля назначенных визитов смены</small>
                  </article>
                );
              })}
            </div>
            <div className="control-note">
              <strong>Диспетчер управляет и операциями, и доступом.</strong>
              <p>
                Аккаунт исполнителя связан с конкретным сотрудником. Абоненту доступны только его
                обращения. Права проверяются сервером при каждом запросе.
              </p>
            </div>
          </>
        )}
        {tab === 'Команда' && (
          <>
            <div className="live-section-title">
              <h2>Команда МаксТелеком</h2>
              <span className="subtle-chip">
                {countText(d.reference.staff.length, 'сотрудник', 'сотрудника', 'сотрудников')}
              </span>
            </div>
            <label className="team-search">
              Найти сотрудника
              <input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Имя, код или технология"
              />
            </label>
            <div className="table-scroll">
              <table>
                <thead>
                  <tr>
                    <th>Сотрудник</th>
                    <th>Специализация</th>
                    <th>Технологии</th>
                    <th>Смена</th>
                    <th>Допуск к высоте</th>
                  </tr>
                </thead>
                <tbody>
                  {d.reference.staff
                    .filter((s) =>
                      `${staffName(s)} ${s.staff_id} ${s.technologies.join(' ')}`
                        .toLowerCase()
                        .includes(search.toLowerCase()),
                    )
                    .map((s) => (
                      <tr key={s.staff_id}>
                        <td>
                          <span className="table-person">
                            <span className="avatar">{initials(staffName(s))}</span>
                            <span>
                              {staffName(s)}
                              <small>{s.staff_id}</small>
                            </span>
                          </span>
                        </td>
                        <td>{s.profiles.map((p) => profileNames[p]).join(', ')}</td>
                        <td>{s.technologies.join(' · ')}</td>
                        <td>
                          {s.shift.start}–{s.shift.end}
                        </td>
                        <td>{s.height_permit ? 'Есть' : 'Нет'}</td>
                      </tr>
                    ))}
                </tbody>
              </table>
            </div>
          </>
        )}
        {tab === 'Нормативы' && (
          <>
            <h2>Типы работ и сроки</h2>
            <div className="table-scroll">
              <table>
                <thead>
                  <tr>
                    <th>Услуга</th>
                    <th>Длительность</th>
                    <th>Окно начала</th>
                    <th>Контрольный срок</th>
                    <th>Участие в ПГП</th>
                  </tr>
                </thead>
                <tbody>
                  {d.reference.work_types.map((t) => (
                    <tr key={t.code}>
                      <td>
                        {t.name}
                        <small>{t.code}</small>
                      </td>
                      <td>{t.duration_min} мин</td>
                      <td>
                        {t.max_start_delay_min == null
                          ? 'В пределах слота'
                          : `+${t.max_start_delay_min} мин`}
                      </td>
                      <td>
                        {t.deadline_kind === 'SLOT'
                          ? 'По окну визита'
                          : `${t.deadline_value} ${t.deadline_kind === 'HOURS' ? 'ч' : 'раб. дн.'}`}
                      </td>
                      <td>{t.pgp_relevant ? 'Да' : 'Нет'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </>
        )}
        {tab === 'Сервисные центры' && (
          <>
            <h2>Точки начала маршрутов и пополнения</h2>
            <div className="center-cards">
              {d.reference.service_centers.map((c) => (
                <article key={c.service_center_id}>
                  <span className="eyebrow">{c.service_center_id}</span>
                  <h3>{c.name}</h3>
                  <p>{c.address}</p>
                  <small>
                    {c.opening_hours.open}–{c.opening_hours.close} · пополнение{' '}
                    {c.restock_duration_min} мин
                  </small>
                </article>
              ))}
            </div>
          </>
        )}
        {tab === 'Платформа' && (
          <>
            <h2>Общее состояние</h2>
            <div className="service-rows">
              <div>
                <span>Сервис планирования</span>
                <span className={`state-badge ${health.data ? 'green' : 'amber'}`}>
                  {health.data ? 'На связи' : 'Нет подтверждения'}
                </span>
              </div>
              <div>
                <span>Версия</span>
                <strong>{health.data?.version || '—'}</strong>
              </div>
              <div>
                <span>Данные</span>
                <strong>PostgreSQL · поколение {d.generation}</strong>
              </div>
              <div>
                <span>Модельное время</span>
                <strong>{time(d.day.now)}</strong>
              </div>
              <div>
                <span>События дня</span>
                <strong>{d.event_count}</strong>
              </div>
              <div>
                <span>Подложка карты</span>
                <strong>OpenStreetMap</strong>
              </div>
            </div>
            <p className="muted role-view-note">
              Персональные сессии и права проверяются Go API. Управление аккаунтами доступно в
              разделе «Доступы».
            </p>
          </>
        )}
      </section>
    </div>
  );
}
