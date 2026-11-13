import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiError, request } from './client';
afterEach(() => vi.unstubAllGlobals());
describe('organizer API boundary', () => {
  it('preserves backend error code and message', async () => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(
          new Response(
            JSON.stringify({ error: { code: 'PLAN_NOT_FOUND', message: 'День не загружен' } }),
            { status: 404 },
          ),
        ),
    );
    await expect(request('', '/api/v1/days/2026-10-26/plan')).rejects.toMatchObject({
      status: 404,
      code: 'PLAN_NOT_FOUND',
      message: 'День не загружен',
    });
  });
  it('reports non-JSON proxy failures as an API error', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response('<html>Bad gateway</html>', { status: 502 })),
    );
    await expect(request('', '/api/v1/health')).rejects.toBeInstanceOf(ApiError);
  });
});
