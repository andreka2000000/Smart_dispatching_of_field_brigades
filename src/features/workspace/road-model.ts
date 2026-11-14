import type { Route } from './types';
export type RoadGeometry = { type: 'LineString'; coordinates: [number, number][] };
export type RoadLeg = {
  from_seq: number;
  to_seq: number;
  duration_sec: number;
  distance_m: number;
  geometry: RoadGeometry;
};
export type RoadRoute = {
  staff_id: string;
  generation: number;
  plan_version: number;
  provider: 'OSRM';
  traffic: 'unavailable';
  duration_sec: number;
  distance_m: number;
  geometry: RoadGeometry;
  legs: RoadLeg[];
};
export function durationText(seconds: number) {
  const minutes = Math.ceil(seconds / 60);
  if (minutes === 0) return '0 мин';
  return minutes < 60 ? `${minutes} мин` : `${Math.floor(minutes / 60)} ч ${minutes % 60} мин`;
}
export function distanceText(metres: number) {
  return `${(metres / 1000).toLocaleString('ru-RU', { maximumFractionDigits: 1 })} км`;
}
// No GPS is assumed. An ETA is shown only before the planned departure.
export function nextRoadLeg(route: Route, road: RoadRoute, now: string) {
  const leg = road.legs.find((l) => {
    const target = route.stops.find((s) => s.seq === l.to_seq);
    return target && Date.parse(target.arrival) > Date.parse(now);
  });
  if (!leg) return null;
  const from = route.stops.find((s) => s.seq === leg.from_seq)!;
  const to = route.stops.find((s) => s.seq === leg.to_seq)!;
  const inTransit = Date.parse(from.end) < Date.parse(now);
  const estimatedArrival = inTransit
    ? undefined
    : new Date(Date.parse(from.end) + leg.duration_sec * 1000).toISOString();
  return { leg, from, to, inTransit, estimatedArrival };
}
