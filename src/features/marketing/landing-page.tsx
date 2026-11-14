import Link from 'next/link';
import {
  ArrowRight,
  ArrowUpRight,
  Check,
  ChevronRight,
  Clock3,
  MapPin,
  RadioTower,
  Route,
  ShieldCheck,
  UserRound,
  Waypoints,
} from 'lucide-react';
import { Navigation } from './navigation';
import { HeroArt } from './hero-art';
import { RoleShowcase } from './role-showcase';
import { BrandMark } from './brand';
import s from './landing.module.css';

function RouteIllustration() {
  return (
    <svg viewBox="0 0 600 260" fill="none" className={s.routeIllustration} aria-hidden="true">
      <defs>
        <pattern id="mini-grid" width="40" height="40" patternUnits="userSpaceOnUse">
          <path d="M40 0H0V40" stroke="#482b63" strokeOpacity=".3" />
        </pattern>
        <linearGradient
          id="mini-line"
          x1="100"
          y1="0"
          x2="500"
          y2="240"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#533275" />
          <stop offset=".5" stopColor="#ba8aff" />
          <stop offset="1" stopColor="#7549aa" />
        </linearGradient>
      </defs>
      <rect width="600" height="260" fill="url(#mini-grid)" />
      <g stroke="#2f223e" strokeWidth="10">
        <path d="M0 80h600M100 0v260M260 0v260M440 0v260M0 210h600" />
      </g>
      <path
        className={s.routeTrace}
        d="M100 210v-90q0-40 40-40h80q40 0 40 40v50q0 40 40 40h100q40 0 40-40V60"
        stroke="url(#mini-line)"
        strokeWidth="3"
        strokeLinecap="round"
      />
      <g fill="#1b102b" stroke="#c99cff" strokeWidth="2">
        {[
          [100, 210],
          [180, 80],
          [260, 155],
          [370, 210],
          [440, 60],
        ].map(([x, y], i) => (
          <g key={i}>
            <circle cx={x} cy={y} r="13" />
            <text
              x={x}
              y={y + 4}
              fill="#e2c9ff"
              stroke="none"
              textAnchor="middle"
              fontSize="10"
              fontFamily="sans-serif"
            >
              {i + 1}
            </text>
          </g>
        ))}
      </g>
      <rect x="315" y="83" width="158" height="41" rx="9" fill="#181021" stroke="#69438a" />
      <circle cx="335" cy="104" r="4" fill="#b286e9" />
      <text x="349" y="108" fill="#d9c9e9" fontSize="11" fontFamily="sans-serif">
        Маршрут построен
      </text>
    </svg>
  );
}

export function LandingPage() {
  return (
    <div className={s.page}>
      <a href="#main-content" className={s.skipLink}>
        К содержанию
      </a>
      <Navigation />
      <main id="main-content">
        <section className={s.hero} aria-labelledby="hero-title">
          <div className={s.heroGlow} />
          <div className={s.container + ' ' + s.heroGrid}>
            <div className={s.heroCopy}>
              <a href="#features" className={s.heroPill}>
                <span /> Умная диспетчеризация <ChevronRight size={13} />
              </a>
              <h1 id="hero-title">
                Порядок там,
                <br />
                <em>где раньше был хаос.</em>
              </h1>
              <p>
                Соединяем заявки, команду и маршруты.
                <br />
                Чтобы сложный рабочий день
                <br className={s.desktopBreak} /> складывался в понятный план.
              </p>
              <div className={s.heroActions}>
                <Link href="/app" className={s.primary}>
                  Открыть сервис <ArrowUpRight size={17} />
                </Link>
                <a href="#how-it-works" className={s.secondary}>
                  Как это работает <ArrowRight size={16} />
                </a>
              </div>
              <div className={s.heroFootnote}>
                <span />
                <span>Один сервис. Вся команда. Общий маршрут.</span>
              </div>
            </div>
            <HeroArt />
          </div>
          <div className={s.container + ' ' + s.heroBottomLine}>
            <span>СЕРВИС, КОТОРЫЙ ДВИЖЕТСЯ ВМЕСТЕ С ВАМИ</span>
            <a href="#features">
              Узнать больше <ArrowRight size={13} />
            </a>
          </div>
        </section>

        <div className={s.container}>
          <div className={s.capabilityStrip}>
            <span>
              <RadioTower size={18} /> Подключение услуг
            </span>
            <span>
              <Route size={18} /> Выездное обслуживание
            </span>
            <span>
              <ShieldCheck size={18} /> Восстановление связи
            </span>
            <span>
              <Waypoints size={18} /> Управление бригадами
            </span>
          </div>
        </div>

        <section id="features" className={s.section}>
          <div className={s.container}>
            <div className={s.sectionIntro}>
              <span className={s.eyebrow}>01 / ВОЗМОЖНОСТИ</span>
              <h2>
                Сложный день.
                <br />
                <span>Понятная система.</span>
              </h2>
              <p>
                Меньше переключений между таблицами и звонками.
                <br />
                Больше внимания тому, что действительно важно.
              </p>
            </div>
            <div className={s.bentoGrid}>
              <article className={s.routeCard}>
                <div className={s.cardIcon}>
                  <Route size={21} />
                </div>
                <span className={s.cardIndex}>01</span>
                <h3>
                  У каждого выезда —<br />
                  свой маршрут.
                </h3>
                <p>
                  Адреса, время визита и порядок остановок
                  <br />в одной картине рабочего дня.
                </p>
                <RouteIllustration />
                <div className={s.routeCardFooter}>
                  <span>
                    <MapPin size={13} /> От заявки до нужной двери
                  </span>
                  <ArrowUpRight size={16} />
                </div>
              </article>
              <article className={s.teamCard}>
                <div className={s.cardIcon}>
                  <UserRound size={20} />
                </div>
                <span className={s.cardIndex}>02</span>
                <h3>Команда в одном ритме.</h3>
                <p>
                  Кто на объекте, кто в пути и кто готов
                  <br />к следующей задаче.
                </p>
                <div className={s.teamRows}>
                  {[
                    { name: 'Алексей Смирнов', letters: 'АС', state: 'На объекте', width: '78%' },
                    { name: 'Марина Волкова', letters: 'МВ', state: 'В пути', width: '54%' },
                    { name: 'Дмитрий Козлов', letters: 'ДК', state: 'Свободен', width: '30%' },
                  ].map((person) => (
                    <div key={person.name}>
                      <span className={s.teamAvatar}>{person.letters}</span>
                      <div>
                        <strong>
                          {person.name}
                          <small>{person.state}</small>
                        </strong>
                        <span className={s.loadTrack}>
                          <i style={{ width: person.width }} />
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </article>
              <article className={s.riskCard}>
                <div>
                  <span className={s.cardIcon}>
                    <Clock3 size={20} />
                  </span>
                  <span className={s.cardIndex}>03</span>
                  <h3>
                    Увидеть раньше.
                    <br />
                    Успеть вовремя.
                  </h3>
                  <p>
                    Важные изменения не теряются.
                    <br />
                    Риски — в центре внимания.
                  </p>
                </div>
                <div className={s.riskIllustration} aria-hidden="true">
                  <div className={s.riskRing} />
                  <Clock3 size={32} />
                  <span>
                    <i /> Время под контролем
                  </span>
                </div>
              </article>
            </div>
          </div>
        </section>

        <section id="roles" className={s.section + ' ' + s.rolesSection}>
          <div className={s.container}>
            <div className={s.sectionIntro}>
              <span className={s.eyebrow}>02 / ЛЮДИ И РОЛИ</span>
              <h2>
                Разные задачи.
                <br />
                <span>Одна команда.</span>
              </h2>
              <p>
                Абоненту — ясность. Команде — удобные инструменты.
                <br />
                Каждому — своё пространство для работы.
              </p>
            </div>
            <RoleShowcase />
          </div>
        </section>

        <section id="how-it-works" className={s.section + ' ' + s.processSection}>
          <div className={s.container}>
            <div className={s.processHeading}>
              <div>
                <span className={s.eyebrow}>03 / КАК ЭТО РАБОТАЕТ</span>
                <h2>
                  От обращения
                  <br />
                  до результата.
                </h2>
              </div>
              <p>
                За хорошим сервисом — понятный процесс.
                <br />
                На каждом этапе.
              </p>
            </div>
            <div className={s.steps}>
              {[
                {
                  number: '01',
                  Icon: MapPin,
                  title: 'Всё начинается с заявки',
                  text: 'Услуга, адрес и удобное время. Вся необходимая информация — в одном обращении.',
                },
                {
                  number: '02',
                  Icon: Waypoints,
                  title: 'Складывается план',
                  text: 'Диспетчер видит задачи и команду. У каждого мастера — своя последовательность визитов.',
                },
                {
                  number: '03',
                  Icon: Check,
                  title: 'Команда выходит на маршрут',
                  text: 'Мастер знает следующий шаг, а абонент понимает, что происходит с его заявкой.',
                },
              ].map(({ number, Icon, title, text }) => (
                <article key={number}>
                  <div className={s.stepTop}>
                    <span>{number}</span>
                    <Icon size={20} />
                  </div>
                  <h3>{title}</h3>
                  <p>{text}</p>
                </article>
              ))}
            </div>
          </div>
        </section>

        <section className={s.faqSection}>
          <div className={s.container + ' ' + s.faqGrid}>
            <div>
              <span className={s.eyebrow}>ДЕТАЛИ</span>
              <h2>
                Если остались
                <br />
                вопросы.
              </h2>
            </div>
            <div className={s.faqList}>
              {[
                {
                  q: 'Кому подойдёт МаксТелеком?',
                  a: 'Командам, которые подключают услуги, восстанавливают связь и обслуживают объекты на выезде. Абонент, диспетчер, мастер и администратор получают своё представление общего процесса.',
                },
                {
                  q: 'Что есть в рабочем пространстве?',
                  a: 'Рабочее место диспетчера: список заявок, карту, команду и карточки назначений. В нём отображаются заявки выбранной смены. Для знакомства с сервисом доступен демонстрационный набор.',
                },
                {
                  q: 'Что увидит абонент?',
                  a: 'Свой путь от обращения до визита мастера: выбор услуги и удобного окна, а затем — статус заявки. На этой странице можно попробовать пример выбора времени; полноценный личный кабинет — следующий этап развития продукта.',
                },
              ].map((item) => (
                <details key={item.q}>
                  <summary>
                    {item.q}
                    <PlusMark />
                  </summary>
                  <p>{item.a}</p>
                </details>
              ))}
            </div>
          </div>
        </section>

        <section className={s.finalSection}>
          <div className={s.container}>
            <div className={s.finalCard}>
              <div className={s.finalGlow} />
              <div className={s.finalMark}>
                <BrandMark size={38} />
              </div>
              <span className={s.eyebrow}>МЕНЬШЕ ХАОСА. БОЛЬШЕ СВЯЗИ.</span>
              <h2>
                Хороший сервис
                <br />
                начинается с порядка.
              </h2>
              <p>
                Посмотрите, как может выглядеть
                <br />
                ваш следующий рабочий день.
              </p>
              <Link href="/app" className={s.primary}>
                Открыть сервис <ArrowUpRight size={17} />
              </Link>
              <small>Единое рабочее пространство команды</small>
            </div>
          </div>
        </section>
      </main>
      <footer className={s.footer}>
        <div className={s.container}>
          <div className={s.footerTop}>
            <Link href="/" className={s.brand}>
              <BrandMark />
              <span>
                МаксТелеком<span className={s.brandDot}></span>
              </span>
            </Link>
            <p>
              Когда люди, задачи и маршруты
              <br />
              работают вместе.
            </p>
            <nav aria-label="Навигация в подвале">
              <a href="#features">Возможности</a>
              <a href="#roles">Для кого</a>
              <Link href="/app">
                Демо <ArrowUpRight size={12} />
              </Link>
            </nav>
          </div>
          <div className={s.footerBottom}>
            <span>© 2026 МаксТелеком</span>
            <span>
              <i /> Создано для движения вперёд
            </span>
            <a href="#main-content">Наверх ↑</a>
          </div>
        </div>
      </footer>
    </div>
  );
}

function PlusMark() {
  return (
    <span className={s.plusMark} aria-hidden="true">
      <i />
      <i />
    </span>
  );
}
