import type { components } from '@/shared/api/schema';
export type Reference = components['schemas']['ReferenceData'];
export type WorkOrder = components['schemas']['Order'];
export type Staff = components['schemas']['Staff'];
export type PlanQuality = {
  weighted_visits: number;
  visits: number;
  travel_sec: number;
  utilization_cv: number;
};
// Optional UI extension; the organizer's generated contract stays unchanged.
export type Plan = components['schemas']['Plan'] & {
  optimization?: {
    engine: 'pyvrp_go' | 'go_baseline';
    status: string;
    baseline: PlanQuality;
    selected: PlanQuality;
  };
};
export type Route = components['schemas']['Route'];
export type Stop = components['schemas']['Stop'];
export type Alert = components['schemas']['Alert'];
export type Event = components['schemas']['Event'];
export type EventResult = components['schemas']['EventResult'];
export type Day = components['schemas']['DayInput'];
export type DaySummary = { date: string; as_of: string; plan_version: number; event_count: number };
export type Snapshot = {
  facts?: {
    order_id: string;
    staff_id: string;
    status: 'IN_PROGRESS' | 'COMPLETED';
    started_at: string;
    completed_at?: string;
    report: string;
  }[];
  day: Day;
  reference: Reference;
  plan: Plan | null;
  events: Event[];
  generation: number;
  event_count: number;
};
