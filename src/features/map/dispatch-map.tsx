'use client';
import { useEffect, useRef, useState } from 'react';
import { Map as LibreMap, Marker, NavigationControl, setWorkerUrl } from 'maplibre-gl';
import type { DemoOrder } from '@/lib/demo-data';
import { staff } from '@/lib/demo-data';

setWorkerUrl(new URL('maplibre-gl/dist/maplibre-gl-worker.mjs', import.meta.url).toString());
const colors = { Критичный: '#ef3340', Высокий: '#f97316', Обычный: '#3b82f6', Низкий: '#94a3b8' };
export default function DispatchMap({
  orders,
  onSelect,
}: {
  orders: DemoOrder[];
  onSelect: (id: string) => void;
}) {
  const container = useRef<HTMLDivElement>(null);
  const [error, setError] = useState(false);
  const callback = useRef(onSelect);
  useEffect(() => {
    callback.current = onSelect;
  }, [onSelect]);
  useEffect(() => {
    if (!container.current) return;
    const map = new LibreMap({
      container: container.current,
      center: [37.6, 55.77],
      zoom: 9.5,
      style: {
        version: 8,
        sources: {
          osm: {
            type: 'raster',
            tiles: ['https://tile.openstreetmap.org/{z}/{x}/{y}.png'],
            tileSize: 256,
            attribution:
              '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
          },
        },
        layers: [{ id: 'osm', type: 'raster', source: 'osm' }],
      },
    });
    map.addControl(new NavigationControl({ showCompass: false }), 'top-right');
    map.on('error', () => setError(true));
    const markers: Marker[] = [];
    for (const order of orders.filter((o) => o.status !== 'Выполнена')) {
      const el = document.createElement('button');
      el.className = 'map-order-marker';
      el.style.background = colors[order.priority];
      el.setAttribute('aria-label', `${order.id}: ${order.title}`);
      el.title = order.id;
      el.addEventListener('click', () => callback.current(order.id));
      markers.push(new Marker({ element: el }).setLngLat(order.coordinates).addTo(map));
    }
    for (const person of staff) {
      const el = document.createElement('span');
      el.className = 'map-staff-marker';
      el.textContent = person.initials;
      el.title = `${person.name} — демонстрационная позиция`;
      markers.push(new Marker({ element: el }).setLngLat(person.coordinates).addTo(map));
    }
    return () => {
      markers.forEach((m) => m.remove());
      map.remove();
    };
  }, [orders]);
  return (
    <div className="map-frame">
      <div
        ref={container}
        className="map-canvas"
        role="region"
        aria-label="Карта заявок и мастеров"
      />
      <div className="map-legend">
        {Object.entries(colors).map(([label, color]) => (
          <span key={label}>
            <i style={{ background: color }} />
            {label}
          </span>
        ))}
        <span>
          <i style={{ background: '#9ca3af', borderRadius: 2 }} />
          Мастер
        </span>
      </div>
      {error && (
        <span className="map-error" role="status">
          Подложка карты временно недоступна
        </span>
      )}
    </div>
  );
}
