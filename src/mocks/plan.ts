import type { Plan } from '../entities/plan/types';

// Empty, deterministic contract fixture. No fabricated optimizer result or live GPS.
export const demoDate = '2026-10-26';
export const demoPlan: Plan = {
  date: demoDate,
  plan_version: 1,
  as_of: '2026-10-26T08:30:00+05:00',
  routes: [],
  unassigned: [],
  pgp_groups: [],
  cancelled: [],
  alerts: [],
};
