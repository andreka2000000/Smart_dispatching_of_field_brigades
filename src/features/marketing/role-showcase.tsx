'use client';
import Link from 'next/link';
import { useState } from 'react';
import {
  ArrowUpRight,
  UserRound,
  RadioTower,
  MapPin,
  Check,
  ShieldCheck,
  Smartphone,
  LayoutDashboard,
  CalendarDays,
  Clock3,
} from 'lucide-react';
import s from './landing.module.css';
const roles = [
  {
    id: 'subscriber',
    label: 'Абонент',
    Icon: UserRound,
    title: 'Сервис, в котором всё понятно.',
    description:
      'От заявки до визита мастера. Выберите услугу, укажите удобное время и следите за статусом обращения.',
    points: ['Удобное окно визита', 'Свои заявки в одном месте', 'Понятный статус обращения'],
  },
  {
    id: 'dispatcher',
    label: 'Диспетчер',
    Icon: LayoutDashboard,
    title: 'Весь рабочий день перед глазами.',
    description:
      'Заявки, доступность команды и маршруты на одном экране. Легко увидеть, где требуется ваше внимание.',
    points: ['Общая карта выездов', 'Расписание каждой бригады', 'Приоритеты и риски опозданий'],
  },
  {
    id: 'technician',
    label: 'Мастер',
    Icon: Smartphone,
    title: 'Меньше вопросов. Больше ясности.',
    description:
      'Следующий адрес, время визита и нужное оборудование. Всё, что помогает спокойно делать свою работу.',
    points: ['Понятная очередь на день', 'Детали предстоящего визита', 'Обновление статуса работы'],
  },
  {
    id: 'admin',
    label: 'Администратор',
    Icon: ShieldCheck,
    title: 'Порядок начинается с настроек.',
    description:
      'Сотрудники, доступы и правила работы собраны вместе. У каждой роли — своё рабочее пространство.',
    points: ['Разделение ролей', 'Единый список сотрудников', 'Настройки сервисной команды'],
  },
];
export function RoleShowcase() {
  const [active, setActive] = useState('subscriber');
  const [slot, setSlot] = useState('14:00–16:00');
  const role = roles.find((r) => r.id === active)!;
  return (
    <>
      <div className={s.roleTabs} role="tablist" aria-label="Роли в системе">
        {roles.map(({ id, label, Icon }) => (
          <button
            key={id}
            id={`tab-${id}`}
            role="tab"
            aria-selected={active === id}
            aria-controls="role-panel"
            tabIndex={active === id ? 0 : -1}
            onClick={() => setActive(id)}
            onKeyDown={(event) => {
              const index = roles.findIndex((item) => item.id === id);
              let nextIndex: number;
              if (event.key === 'ArrowRight') nextIndex = (index + 1) % roles.length;
              else if (event.key === 'ArrowLeft')
                nextIndex = (index - 1 + roles.length) % roles.length;
              else if (event.key === 'Home') nextIndex = 0;
              else if (event.key === 'End') nextIndex = roles.length - 1;
              else return;
              event.preventDefault();
              const nextId = roles[nextIndex].id;
              setActive(nextId);
              event.currentTarget.parentElement
                ?.querySelector<HTMLButtonElement>(`#tab-${nextId}`)
                ?.focus();
            }}
          >
            <Icon size={16} />
            {label}
          </button>
        ))}
      </div>
      <div
        id="role-panel"
        role="tabpanel"
        aria-labelledby={`tab-${active}`}
        className={s.rolePanel}
      >
        <div className={s.roleCopy}>
          <span className={s.eyebrow}>СВОЯ РОЛЬ. ОБЩАЯ КАРТИНА.</span>
          <h3>{role.title}</h3>
          <p>{role.description}</p>
          <ul>
            {role.points.map((point) => (
              <li key={point}>
                <Check size={14} />
                {point}
              </li>
            ))}
          </ul>
          {active === 'subscriber' ? (
            <a href="#how-it-works" className={s.textLink}>
              Как устроен сервис <ArrowUpRight size={15} />
            </a>
          ) : (
            <Link
              href={`/app/${active === 'dispatcher' ? 'dispatch' : active}`}
              className={s.textLink}
            >
              Открыть демо <ArrowUpRight size={15} />
            </Link>
          )}
        </div>
        <div className={s.roleVisual}>
          <div className={s.roleWindow}>
            <div className={s.roleWindowTop}>
              <span>
                <i />
                <i />
                <i />
              </span>
              <small>{active === 'subscriber' ? 'Личный кабинет' : role.label}</small>
              <span className={s.windowLock}>
                <ShieldCheck size={12} />
              </span>
            </div>
            {active === 'subscriber' ? (
              <div className={s.appointment}>
                <span className={s.mockLabel}>
                  <RadioTower size={15} /> ПОДКЛЮЧЕНИЕ ИНТЕРНЕТА
                </span>
                <h4>
                  Будем в удобное
                  <br />
                  для вас время.
                </h4>
                <div className={s.appointmentDate}>
                  <CalendarDays size={17} />
                  <div>
                    Пятница, 13 ноября<small>Выберите пример окна визита</small>
                  </div>
                </div>
                <div className={s.slotPicker} aria-label="Пример выбора окна">
                  {['12:00–14:00', '14:00–16:00', '16:00–18:00'].map((time) => (
                    <button key={time} aria-pressed={slot === time} onClick={() => setSlot(time)}>
                      {time}
                    </button>
                  ))}
                </div>
                <div className={s.appointmentStatus}>
                  <span>
                    <Check size={15} />
                  </span>
                  <div>
                    Окно выбрано<small>{slot} · пример интерфейса</small>
                  </div>
                </div>
              </div>
            ) : active === 'dispatcher' ? (
              <div className={s.dispatchPreview}>
                <div className={s.mockLabel}>
                  <LayoutDashboard size={14} /> ПЛАН СМЕНЫ
                </div>
                <div className={s.previewStat}>
                  <strong>Всё на своих местах</strong>
                  <span>Согласованные окна и маршруты</span>
                </div>
                {['Алексей Смирнов', 'Марина Волкова', 'Дмитрий Козлов'].map((name, i) => (
                  <div key={name} className={s.timelineRow}>
                    <span>{name}</span>
                    <div>
                      <i style={{ left: `${i * 10}%`, width: '27%' }} />
                      <i style={{ left: `${38 + i * 7}%`, width: '33%' }} />
                    </div>
                  </div>
                ))}
                <div className={s.timelineHours}>
                  <span>09:00</span>
                  <span>12:00</span>
                  <span>15:00</span>
                  <span>18:00</span>
                </div>
              </div>
            ) : active === 'technician' ? (
              <div className={s.technicianPreview}>
                <span className={s.mockLabel}>
                  <Smartphone size={14} /> СЛЕДУЮЩИЙ ВИЗИТ
                </span>
                <span className={s.previewBadge}>Назначена</span>
                <h4>
                  Подключение
                  <br />
                  оптоволокна
                </h4>
                <p>
                  <MapPin size={14} /> ул. Учителей, 18
                </p>
                <p>
                  <Clock3 size={14} /> 14:00–16:00
                </p>
                <div className={s.equipment}>
                  <span>Взять с собой</span>
                  <strong>ONT · роутер · патч-корд</strong>
                </div>
                <span className={s.previewAction}>
                  Всё готово к выезду <Check size={14} />
                </span>
              </div>
            ) : (
              <div className={s.adminPreview}>
                <span className={s.mockLabel}>
                  <ShieldCheck size={14} /> ДОСТУП К СИСТЕМЕ
                </span>
                <h4>
                  Нужные права.
                  <br />
                  Каждому.
                </h4>
                {['Диспетчер', 'Мастер', 'Абонент'].map((name, i) => (
                  <div className={s.accessRow} key={name}>
                    <span className={s.accessAvatar}>{['НО', 'АС', 'ЕК'][i]}</span>
                    <strong>{name}</strong>
                    <span>
                      Доступ настроен <Check size={12} />
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
          <p className={s.visualNote}>Интерактивный макет · без отправки данных</p>
        </div>
      </div>
    </>
  );
}
