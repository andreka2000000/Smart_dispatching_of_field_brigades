'use client';
import dynamic from 'next/dynamic';
export const LiveMap = dynamic(() => import('./live-map'), {
  ssr: false,
  loading: () => <div className="live-map-frame loading-map">Загружаем карту…</div>,
});
