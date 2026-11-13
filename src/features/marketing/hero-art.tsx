'use client';
import { useState } from 'react';
import { Pause, Play, Check, ArrowUpRight } from 'lucide-react';
import s from './landing.module.css';

export function HeroArt() {
  const [paused, setPaused] = useState(false);
  return (
    <div className={s.heroArt} data-paused={paused}>
      <div className={s.artGlow} />
      <div className={s.artCoordinates}>56°50′ N &nbsp; 60°36′ E</div>
      <svg
        className={s.cityArt}
        viewBox="0 0 660 590"
        fill="none"
        role="img"
        aria-labelledby="hero-art-title"
      >
        <title id="hero-art-title">
          Светящийся диспетчерский узел соединяет заявки и бригады на карте города
        </title>
        <defs>
          <linearGradient
            id="floor"
            x1="100"
            y1="220"
            x2="600"
            y2="550"
            gradientUnits="userSpaceOnUse"
          >
            <stop stopColor="#21143a" />
            <stop offset="1" stopColor="#09090f" />
          </linearGradient>
          <linearGradient
            id="core-top"
            x1="240"
            y1="140"
            x2="410"
            y2="250"
            gradientUnits="userSpaceOnUse"
          >
            <stop stopColor="#c8abff" />
            <stop offset=".5" stopColor="#7940dc" />
            <stop offset="1" stopColor="#4e247e" />
          </linearGradient>
          <linearGradient
            id="core-side"
            x1="240"
            y1="210"
            x2="320"
            y2="350"
            gradientUnits="userSpaceOnUse"
          >
            <stop stopColor="#6b35c0" />
            <stop offset="1" stopColor="#1f103c" />
          </linearGradient>
          <linearGradient
            id="core-right"
            x1="330"
            y1="230"
            x2="400"
            y2="320"
            gradientUnits="userSpaceOnUse"
          >
            <stop stopColor="#4c227e" />
            <stop offset="1" stopColor="#150c25" />
          </linearGradient>
          <linearGradient
            id="beam"
            x1="330"
            y1="140"
            x2="330"
            y2="380"
            gradientUnits="userSpaceOnUse"
          >
            <stop stopColor="#a678ff" stopOpacity="0" />
            <stop offset="1" stopColor="#a678ff" stopOpacity=".25" />
          </linearGradient>
          <radialGradient id="ground-glow">
            <stop stopColor="#9956ff" stopOpacity=".42" />
            <stop offset="1" stopColor="#9956ff" stopOpacity="0" />
          </radialGradient>
          <filter id="route-glow">
            <feGaussianBlur stdDeviation="4" />
          </filter>
          <pattern
            id="city-grid"
            width="60"
            height="32"
            patternUnits="userSpaceOnUse"
            patternTransform="matrix(1 .5 -1 .5 330 0)"
          >
            <path d="M60 0H0V32" stroke="#7b52aa" strokeOpacity=".22" />
          </pattern>
        </defs>
        <ellipse cx="330" cy="405" rx="290" ry="145" fill="url(#ground-glow)" />
        <path
          d="m45 335 285-152 285 152v15L330 506 45 350Z"
          fill="#0e0b16"
          stroke="#5c3682"
          strokeOpacity=".45"
        />
        <path
          d="m45 335 285-152 285 152-285 155Z"
          fill="url(#floor)"
          stroke="#6c4296"
          strokeOpacity=".65"
        />
        <path d="m45 335 285-152 285 152-285 155Z" fill="url(#city-grid)" />
        <g stroke="#4b345f" strokeWidth="1">
          <path d="m90 335 240-128 241 128M140 335l190-102 192 102M190 335l140-75 142 75M140 387l190-102 190 102M190 414l140-75 140 75M237 440l93-50 93 50" />
          <path d="m95 310 285 155M145 284l285 155M195 257l285 155M245 230l285 155M285 208 570 363M95 363l285-155M145 390l285-155M195 416l285-155M245 443l285-155M285 465l285-155" />
        </g>
        <g stroke="#a16bff" strokeWidth="2" strokeLinejoin="round">
          <path d="m139 335 95-51 95 51 96-51 96 51" />
          <path d="m234 386 95-51 95 51 48-25" />
          <path d="m329 335 1 108" />
        </g>
        <g stroke="#a16bff" strokeWidth="5" filter="url(#route-glow)" opacity=".35">
          <path d="m139 335 95-51 95 51 96-51 96 51" />
          <path d="m234 386 95-51 95 51 48-25" />
        </g>
        <g fill="#c9a5ff" stroke="#332044" strokeWidth="6">
          {[
            [139, 335],
            [234, 284],
            [425, 284],
            [521, 335],
            [234, 386],
            [472, 361],
            [330, 443],
          ].map(([x, y]) => (
            <circle key={`${x}-${y}`} cx={x} cy={y} r="5" />
          ))}
        </g>
        {!paused && (
          <g className={s.movingSignals} fill="#fff">
            <circle r="3">
              <animateMotion
                dur="5s"
                repeatCount="indefinite"
                path="M139 335 234 284 329 335 425 284 521 335"
              />
            </circle>
            <circle r="3">
              <animateMotion
                dur="4s"
                repeatCount="indefinite"
                path="M472 361 424 386 329 335 234 386"
              />
            </circle>
          </g>
        )}
        <g opacity=".8">
          <path d="m120 275 28-15 28 15v40l-28 15-28-15Z" fill="#130e20" stroke="#514062" />
          <path d="m120 275 28 15 28-15m-28 15v40" stroke="#514062" />
          <path d="m493 311 27-14 27 14v48l-27 15-27-15Z" fill="#130e20" stroke="#514062" />
          <path d="m493 311 27 15 27-15m-27 15v48" stroke="#514062" />
          <path d="m363 413 23-12 23 12v24l-23 12-23-12Z" fill="#171022" stroke="#514062" />
          <path d="m363 413 23 12 23-12m-23 12v24" stroke="#514062" />
        </g>
        <path d="m237 176 93 48 93-48v161l-93 49-93-49Z" fill="url(#beam)" />
        <g className={s.floatingCore}>
          <path
            d="m237 208 93 49 93-49v91l-93 49-93-49Z"
            fill="url(#core-side)"
            stroke="#a273df"
            strokeWidth="1.3"
          />
          <path d="m330 257 93-49v91l-93 49Z" fill="url(#core-right)" />
          <path
            d="m237 208 93-49 93 49-93 49Z"
            fill="url(#core-top)"
            stroke="#d7b5ff"
            strokeWidth="1.4"
          />
          <path d="m247 208 83-43 83 43-83 44Z" stroke="#e4cfff" strokeOpacity=".4" />
          <path d="m330 257 0 91M237 280l93 49 93-49" stroke="#a273df" strokeOpacity=".35" />
          <path
            d="m278 211 50-24 43 22m-66-11 22 29 25-10"
            stroke="#f0dfff"
            strokeWidth="3.5"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
          <g fill="#fff1ff">
            <ellipse cx="278" cy="211" rx="5" ry="3" />
            <ellipse cx="328" cy="187" rx="5" ry="3" />
            <ellipse cx="371" cy="209" rx="5" ry="3" />
            <ellipse cx="327" cy="227" rx="5" ry="3" />
          </g>
          <path
            d="m262 251 39 21v5l-39-21ZM262 265l21 11v5l-21-11Z"
            fill="#c8a9ff"
            fillOpacity=".6"
          />
        </g>
        <g stroke="#8c5cbf" strokeOpacity=".35">
          <path d="M330 140V93M220 200l-53-28M441 200l54-29" strokeDasharray="3 5" />
          <path d="M330 93h62" />
        </g>
        <text x="402" y="97" fill="#a38abf" fontSize="10" fontFamily="monospace">
          СИСТЕМА АКТИВНА
        </text>
      </svg>
      <div className={s.artCardTop}>
        <span className={s.artCheck}>
          <Check size={14} />
        </span>
        <div>
          <strong>Мастер назначен</strong>
          <small>Прибытие в согласованное окно</small>
        </div>
        <ArrowUpRight size={14} />
      </div>
      <div className={s.artCardBottom}>
        <span className={s.signalDot} />
        <span>Заявки. Люди. Маршруты.</span>
        <span className={s.artCardBars}>
          <i />
          <i />
          <i />
          <i />
        </span>
      </div>
      <div className={s.artBottom}>
        <span>Иллюстрация работы системы</span>
        <button
          aria-label={paused ? 'Включить анимацию' : 'Приостановить анимацию'}
          aria-pressed={paused}
          onClick={() => setPaused(!paused)}
        >
          {paused ? <Play size={12} /> : <Pause size={12} />}
        </button>
      </div>
    </div>
  );
}
