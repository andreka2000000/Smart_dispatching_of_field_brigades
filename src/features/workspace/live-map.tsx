'use client';
import { useEffect, useRef, useState } from 'react';
import {
  Map as LibreMap,
  Marker,
  NavigationControl,
  LngLatBounds,
  setWorkerUrl,
  type GeoJSONSource,
} from 'maplibre-gl';
import type { Route } from './types';
import type { ViewOrder } from './model';
import { useRoadRoute } from './road-routing';
import { distanceText, durationText } from './road-model';
setWorkerUrl(new URL('maplibre-gl/dist/maplibre-gl-worker.mjs', import.meta.url).toString());
export default function LiveMap({
  orders,
  points,
  route,
  onSelect,
  selected,
}: {
  orders: ViewOrder[];
  points: Record<string, [number, number]>;
  route?: Route;
  onSelect: (id: string) => void;
  selected?: string | null;
}) {
  const node = useRef<HTMLDivElement>(null);
  const map = useRef<LibreMap | null>(null);
  const callback = useRef(onSelect);
  const fitted = useRef('');
  const [ready, setReady] = useState(false);
  const [error, setError] = useState(false);
  const road = useRoadRoute(route);
  useEffect(() => {
    callback.current = onSelect;
  }, [onSelect]);
  useEffect(() => {
    if (!node.current) return;
    const m = new LibreMap({
      container: node.current,
      center: [60.62, 56.85],
      zoom: 11,
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
        layers: [
          {
            id: 'base',
            type: 'raster',
            source: 'osm',
            paint: {
              'raster-saturation': -0.8,
              'raster-brightness-max': 0.5,
              'raster-brightness-min': 0.03,
            },
          },
        ],
      },
    });
    map.current = m;
    const observer = new ResizeObserver(() => m.resize());
    observer.observe(node.current);
    m.addControl(new NavigationControl({ showCompass: false }), 'top-right');
    m.on('style.load', () => setReady(true));
    m.on('error', () => setError(true));
    return () => {
      m.remove();
      observer.disconnect();
      map.current = null;
    };
  }, []);
  useEffect(() => {
    const m = map.current;
    if (!m || !ready) return;
    const markers: Marker[] = [];
    const grouped = new Map<string, ViewOrder[]>();
    for (const o of orders) {
      const key = `${o.source.lon},${o.source.lat}`;
      grouped.set(key, [...(grouped.get(key) || []), o]);
    }
    for (const items of grouped.values()) {
      const focus = items.find((o) => o.source.order_id === selected) || items[0];
      const el = document.createElement('button');
      el.type = 'button';
      el.className = `live-marker ${focus.tone} ${items.some((o) => o.source.order_id === selected) ? 'focused' : ''}`;
      el.textContent = items.length > 1 ? String(items.length) : '•';
      el.setAttribute(
        'aria-label',
        `Заявка ${focus.source.order_id}: ${focus.source.address}${items.length > 1 ? `, по адресу ${items.length} заявок` : ''}`,
      );
      el.addEventListener('click', () => callback.current(focus.source.order_id));
      markers.push(
        new Marker({ element: el }).setLngLat([focus.source.lon, focus.source.lat]).addTo(m),
      );
    }
    const coords =
      route?.stops.map((s) => points[s.location_id]).filter((p): p is [number, number] => !!p) ||
      [];
    const geometry = {
      type: 'FeatureCollection' as const,
      features:
        road.data && road.data.geometry.coordinates.length > 1
          ? [
              {
                type: 'Feature' as const,
                properties: {},
                geometry: road.data.geometry,
              },
            ]
          : [],
    };
    const source = m.getSource('route') as GeoJSONSource | undefined;
    if (source) source.setData(geometry);
    else {
      m.addSource('route', {
        type: 'geojson',
        data: geometry,
        attribution: '<a href="https://project-osrm.org/">OSRM</a>',
      });
      m.addLayer({
        id: 'route-halo',
        type: 'line',
        source: 'route',
        layout: { 'line-join': 'round', 'line-cap': 'round' },
        paint: { 'line-color': '#1b092e', 'line-width': 8, 'line-opacity': 0.85 },
      });
      m.addLayer({
        id: 'route-line',
        type: 'line',
        source: 'route',
        layout: { 'line-join': 'round', 'line-cap': 'round' },
        paint: {
          'line-color': '#bc91ff',
          'line-width': 4,
          'line-opacity': 0.95,
        },
      });
    }
    const selectedSeq = route?.stops.find((s) => s.order_id === selected)?.seq;
    const selectedLeg = road.data?.legs.find((l) => l.to_seq === selectedSeq);
    const selectedGeometry = {
      type: 'FeatureCollection' as const,
      features:
        selectedLeg && selectedLeg.geometry.coordinates.length > 1
          ? [{ type: 'Feature' as const, properties: {}, geometry: selectedLeg.geometry }]
          : [],
    };
    const selectedSource = m.getSource('route-selected') as GeoJSONSource | undefined;
    if (selectedSource) selectedSource.setData(selectedGeometry);
    else {
      m.addSource('route-selected', { type: 'geojson', data: selectedGeometry });
      m.addLayer({
        id: 'route-selected-line',
        type: 'line',
        source: 'route-selected',
        layout: { 'line-join': 'round', 'line-cap': 'round' },
        paint: { 'line-color': '#e4c9ff', 'line-width': 5 },
      });
    }
    const centers = new Set(
      route?.stops
        .filter((s) => ['DEPOT_START', 'DEPOT_END', 'RESTOCK'].includes(s.type))
        .map((s) => s.location_id) || [],
    );
    for (const id of centers) {
      const position = points[id];
      if (!position) continue;
      const el = document.createElement('div');
      el.className = 'live-depot-marker';
      el.textContent = 'СЦ';
      el.setAttribute('role', 'img');
      el.setAttribute('aria-label', `Сервисный центр ${id}`);
      markers.push(new Marker({ element: el }).setLngLat(position).addTo(m));
    }
    const fitPoints = road.data?.geometry.coordinates.length
      ? road.data.geometry.coordinates
      : coords.length
        ? coords
        : orders.map((o) => [o.source.lon, o.source.lat] as [number, number]);
    const signature = JSON.stringify(fitPoints);
    if (fitPoints.length && fitted.current !== signature) {
      fitted.current = signature;
      const bounds = new LngLatBounds();
      fitPoints.forEach((p) => bounds.extend(p));
      m.fitBounds(bounds, { padding: 65, maxZoom: 13, duration: 500 });
    }
    return () => markers.forEach((m) => m.remove());
  }, [orders, points, route, selected, ready, road.data]);
  return (
    <div className="live-map-frame">
      <div
        ref={node}
        className="live-map-canvas"
        role="region"
        aria-label="Карта заявок МаксТелеком"
      />
      <div className="live-map-label">
        <span className="status-light" />
        {route ? 'Автомобильный маршрут' : 'География заявок'}
        <small>
          {route
            ? road.data
              ? `${distanceText(road.data.distance_m)} · ${durationText(road.data.duration_sec)} · без текущих пробок`
              : road.error
                ? 'Дорожный маршрут недоступен'
                : 'Строим путь по дорогам…'
            : 'Выберите сотрудника для дорожного маршрута'}
        </small>
      </div>
      {route && road.error && (
        <div className="live-map-error" role="status">
          Не удалось загрузить дорожный путь. Сохранённый план доступен в расписании.
        </div>
      )}
      {error && (
        <div className="live-map-error" role="status">
          Подложка временно недоступна. Адреса доступны в списке.
        </div>
      )}
    </div>
  );
}
