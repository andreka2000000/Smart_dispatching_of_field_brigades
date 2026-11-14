import { readFileSync } from 'node:fs';
import { describe, it, expect } from 'vitest';
import type { Snapshot } from './types';
import { ordersView, locationPoints, fromLocal, localInput } from './model';
const fixture = (name: string) => JSON.parse(readFileSync(new URL(`./__fixtures__/${name}.json`, import.meta.url), 'utf8'));
function data(): Snapshot {
  return {
    reference: fixture('reference'),
    day: fixture('day'),
    generation: 1,
    event_count: 0,
    events: [],
    plan: null,
  };
}
describe('live workspace presentation', () => {
  it('joins the Go plan with reference data and uses model time', () => {
    const d = data();
    d.plan = {
      date: d.day.date,
      as_of: d.day.now,
      plan_version: 1,
      routes: [
        {
          staff_id: 'ST-001',
          service_center_id: 'SC-01',
          status: 'ACTIVE',
          stops: [
            {
              seq: 1,
              type: 'VISIT',
              location_id: 'B-00412',
              order_id: d.day.orders[0].order_id,
              arrival: '2026-10-26T09:18:00+05:00',
              start: '2026-10-26T09:18:00+05:00',
              end: '2026-10-26T10:18:00+05:00',
            },
          ],
        },
      ],
      unassigned: [],
      pgp_groups: [],
      cancelled: [],
      alerts: [],
    };
    expect(ordersView(d)[0].status).toBe('Назначена');
    d.day.now = '2026-10-26T09:30:00+05:00';
    expect(ordersView(d)[0].status).toBe('Окно визита идёт');
    d.day.now = '2026-10-26T10:18:00+05:00';
    expect(ordersView(d)[0].status).toBe('Ожидает подтверждения');
    d.facts = [
      {
        order_id: d.day.orders[0].order_id,
        staff_id: 'ST-001',
        status: 'COMPLETED',
        started_at: '2026-10-26T09:18:00+05:00',
        completed_at: d.day.now,
        report: 'Работа выполнена',
      },
    ];
    expect(ordersView(d)[0].status).toBe('Выполнена');
    expect(ordersView(d)[0].title).toContain('неисправности');
    expect(locationPoints(d)['B-00412']).toEqual([60.6531, 56.8712]);
  });
  it('keeps PGP and cancelled orders separate from jobs awaiting staff', () => {
    const d = data();
    d.plan = {
      date: d.day.date,
      as_of: d.day.now,
      plan_version: 1,
      routes: [],
      unassigned: [],
      pgp_groups: [
        {
          group_id: 'g',
          access_node_id: 'AN-0031',
          order_ids: [d.day.orders[0].order_id],
          detected_at: d.day.now,
          action: 'TRANSFER_TO_STATION',
        },
      ],
      cancelled: [{ order_id: d.day.orders[1].order_id, cancelled_at: d.day.now }],
      alerts: [],
    };
    const orders = ordersView(d);
    expect(orders[0].status).toBe('ПГП');
    expect(orders[1].status).toBe('Отменена');
  });
  it('round-trips event time in the case timezone independently of local time', () => {
    const value = '2026-10-26T03:45:12Z';
    expect(localInput(value)).toBe('2026-10-26T08:45:12');
    expect(new Date(fromLocal(localInput(value))).toISOString()).toBe('2026-10-26T03:45:12.000Z');
  });
});
