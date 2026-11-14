'use client';
import Link from 'next/link';
import { useState } from 'react';
import { ArrowUpRight, Menu, X } from 'lucide-react';
import { BrandMark } from './brand';
import s from './landing.module.css';

export function Navigation() {
  const [open, setOpen] = useState(false);
  return (
    <header className={s.header}>
      <div className={s.navInner}>
        <Link href="/" className={s.brand} aria-label="МаксТелеком — главная">
          <BrandMark />
          <span>
            МаксТелеком<span className={s.brandDot}></span>
          </span>
        </Link>
        <nav aria-label="Навигация по главной" className={`${s.navLinks} ${open ? s.navOpen : ''}`}>
          <a href="#features" onClick={() => setOpen(false)}>
            Возможности
          </a>
          <a href="#roles" onClick={() => setOpen(false)}>
            Для кого
          </a>
          <a href="#how-it-works" onClick={() => setOpen(false)}>
            Как это работает
          </a>
          <Link href="/app" className={s.mobileDemo}>
            Открыть сервис <ArrowUpRight size={15} />
          </Link>
        </nav>
        <Link href="/app" className={s.navCta}>
          Открыть сервис <ArrowUpRight size={15} />
        </Link>
        <button
          className={s.menuButton}
          aria-label={open ? 'Закрыть меню' : 'Открыть меню'}
          aria-expanded={open}
          onClick={() => setOpen(!open)}
        >
          {open ? <X size={22} /> : <Menu size={22} />}
        </button>
      </div>
    </header>
  );
}
