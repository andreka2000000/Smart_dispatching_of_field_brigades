import { describe, expect, it } from 'vitest';
import type { Route } from './types';
import { nextRoadLeg, type RoadRoute } from './road-model';
const timestamp = (time: string) => `2026-10-28T${time}:00+05:00`;
describe('road estimates and model time', () => {
  it('provides a reference ETA before departure and no invented remaining ETA during travel', () => {
    const route = {
      stops: [
        { seq: 1, end: timestamp('09:00') },
        { seq: 2, arrival: timestamp('09:20'), end: timestamp('10:20') },
        { seq: 3, arrival: timestamp('12:00'), end: timestamp('13:00') },
        { seq: 4, arrival: timestamp('13:20'), end: timestamp('13:20') },
      ],
    } as Route;
    const road = {
      legs: [
        { from_seq: 1, to_seq: 2, duration_sec: 900 },
        { from_seq: 3, to_seq: 4, duration_sec: 900 },
      ],
    } as RoadRoute;
    expect(nextRoadLeg(route, road, timestamp('08:30'))?.estimatedArrival).toBe(
      '2026-10-28T04:15:00.000Z',
    );
    expect(nextRoadLeg(route, road, timestamp('09:10'))?.estimatedArrival).toBeUndefined();
    expect(nextRoadLeg(route, road, timestamp('09:10'))?.inTransit).toBe(true);
    expect(nextRoadLeg(route, road, timestamp('10:30'))?.from.seq).toBe(3);
    expect(nextRoadLeg(route, road, timestamp('13:30'))).toBeNull();
  });
});
