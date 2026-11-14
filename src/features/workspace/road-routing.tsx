'use client';
import { useQuery } from '@tanstack/react-query';
import { Navigation, Route as RouteIcon } from 'lucide-react';
import { request } from '@/shared/api/client';
import { config } from '@/shared/config';
import { useWorkspace } from './context';
import { errorText, time } from './model';
import { distanceText, durationText, nextRoadLeg, type RoadRoute } from './road-model';
import type { Route } from './types';
export function useRoadRoute(route?: Route) {
  const d = useWorkspace().data;
  const date = d?.day.date;
  const generation = d?.generation;
  const version = d?.plan?.plan_version;
  const staff = route?.staff_id;
  return useQuery({
    queryKey: ['road-route', date, generation, version, staff],
    enabled: !!(route && date && version),
    queryFn: ({ signal }) =>
      request<RoadRoute>(
        config.apiBaseUrl,
        `/ui/v1/days/${date}/staff/${encodeURIComponent(staff!)}/road-route?generation=${generation}&version=${version}`,
        { signal },
      ),
    staleTime: 15 * 60_000,
    gcTime: 20 * 60_000,
    retry: false,
  });
}
export function RoadPanel({ route }: { route?: Route }) {
  const w = useWorkspace();
  const d = w.data!;
  const query = useRoadRoute(route);
  const road = query.data;
  const next = route && road ? nextRoadLeg(route, road, d.day.now) : null;
  const location = (id: string) =>
    d.reference.service_centers.find((c) => c.service_center_id === id)?.name ||
    d.reference.access_nodes.flatMap((n) => n.buildings).find((b) => b.building_id === id)
      ?.address ||
    id;
  return (
    <section className="live-panel road-panel" aria-label="Дорожный маршрут">
      <div className="live-section-title">
        <div>
          <span className="eyebrow">ПО ДОРОГАМ ГОРОДА</span>
          <h2>Следующий переезд</h2>
        </div>
        <Navigation size={19} />
      </div>
      {!route ? (
        <p className="muted">Сначала постройте план смены.</p>
      ) : query.isPending ? (
        <p role="status" className="muted">
          Строим дорожный маршрут…
        </p>
      ) : query.error ? (
        <div role="status" className="road-unavailable">
          <p>{errorText(query.error)}</p>
          <button
            className="secondary-button"
            onClick={() => {
              void w.refresh();
              void query.refetch();
            }}
          >
            Повторить расчёт дороги
          </button>
        </div>
      ) : (
        road && (
          <>
            {next ? (
              <>
                <p className="road-destination">{location(next.to.location_id)}</p>
                <div className="road-metrics">
                  <strong>
                    {durationText(next.leg.duration_sec)}
                    <small>Оценка времени в пути</small>
                  </strong>
                  <strong>
                    {distanceText(next.leg.distance_m)}
                    <small>По дорожной сети</small>
                  </strong>
                  <strong>
                    {next.estimatedArrival ? time(next.estimatedArrival) : '—'}
                    <small>
                      {next.estimatedArrival
                        ? `При выезде в ${time(next.from.end)}`
                        : 'Остаток пути без GPS неизвестен'}
                    </small>
                  </strong>
                </div>
                <p className="road-origin">
                  От: {location(next.from.location_id)} · прибытие по плану {time(next.to.arrival)}
                </p>
                {next.inTransit && (
                  <p className="road-note">
                    По модельному плану переезд уже начат. Показана оценка всего участка от
                    предыдущей остановки.
                  </p>
                )}
              </>
            ) : (
              <p className="muted">
                {road.legs.length
                  ? 'Все переезды завершены по модельному расписанию.'
                  : 'В этом плане нет дорожных переездов.'}
              </p>
            )}
            <div className="road-footer">
              <span>
                <RouteIcon size={14} /> За смену: {distanceText(road.distance_m)} ·{' '}
                {durationText(road.duration_sec)}
              </span>
              <span className="state-badge violet">Без текущих пробок</span>
            </div>
            <p className="road-note">
              OSRM · OpenStreetMap. Положение мастера не отслеживается. Дорожная оценка справочная;
              расписание проверяется по матрице кейса.
            </p>
          </>
        )
      )}
    </section>
  );
}
