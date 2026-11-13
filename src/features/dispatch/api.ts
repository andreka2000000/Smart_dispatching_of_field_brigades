import { config } from '../../shared/config';
import { request } from '../../shared/api/client';
import { demoPlan } from '../../mocks/plan';
import type { Plan, Event, EventResult } from '../../entities/plan/types';

const dayPath = (date: string) => `/api/v1/days/${encodeURIComponent(date)}`;
export const dispatchApi = {
  getPlan(date: string, signal?: AbortSignal): Promise<Plan> {
    if (config.dataMode === 'mock') {
      return Promise.resolve({ ...demoPlan, date, as_of: `${date}T08:30:00+05:00` });
    }
    return request(config.apiBaseUrl, `${dayPath(date)}/plan`, { signal });
  },
  buildPlan(date: string, timeLimitSec = 300, signal?: AbortSignal): Promise<Plan> {
    if (config.dataMode === 'mock')
      return Promise.reject(new Error('Расчёт доступен после подключения Go API.'));
    return request(config.apiBaseUrl, `${dayPath(date)}/plan`, {
      method: 'POST',
      body: JSON.stringify({ time_limit_sec: timeLimitSec }),
      signal,
    });
  },
  postEvent(date: string, event: Event, signal?: AbortSignal): Promise<EventResult> {
    if (config.dataMode === 'mock')
      return Promise.reject(new Error('События доступны после подключения Go API.'));
    return request(config.apiBaseUrl, `${dayPath(date)}/events`, {
      method: 'POST',
      body: JSON.stringify(event),
      signal,
    });
  },
};
