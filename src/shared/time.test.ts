import { expect, it } from 'vitest';
import { formatModelTime } from './time';
it('uses model time in Yekaterinburg regardless of supplied offset', () => {
  expect(formatModelTime('2026-10-26T03:30:00Z')).toBe('08:30');
  expect(formatModelTime('2026-10-26T08:30:00+05:00')).toBe('08:30');
});
