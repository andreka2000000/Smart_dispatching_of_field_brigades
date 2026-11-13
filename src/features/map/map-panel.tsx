'use client';
import dynamic from 'next/dynamic';
const DispatchMap = dynamic(() => import('./dispatch-map'), {
  ssr: false,
  loading: () => <div className="map-frame loading-map">Загрузка карты…</div>,
});
export { DispatchMap };
