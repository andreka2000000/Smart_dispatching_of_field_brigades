import { afterEach, expect, it, vi } from 'vitest';
import { proxy } from './proxy';
afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});
it('forwards the session and CSRF origin and preserves login cookie attributes', async () => {
  vi.stubEnv('API_PROXY_TARGET', 'http://api:8080');
  const upstream = vi
    .fn()
    .mockResolvedValue(
      new Response('{"role":"TECHNICIAN"}', {
        headers: {
          'Set-Cookie': 'mt_session=opaque; Path=/; Max-Age=28800; HttpOnly; SameSite=Strict',
        },
      }),
    );
  vi.stubGlobal('fetch', upstream);
  const result = await proxy(
    new Request('http://127.0.0.1:3001/auth/v1/session', {
      method: 'POST',
      headers: {
        Cookie: 'mt_session=previous',
        Origin: 'http://127.0.0.1:3001',
        'Sec-Fetch-Site': 'same-origin',
        'X-User-Role': 'DISPATCHER',
      },
      body: '{"username":"master","password":"test-only-password"}',
    }),
  );
  const [url, options] = upstream.mock.calls[0];
  expect(url).toBe('http://api:8080/auth/v1/session');
  expect(options.headers.Cookie).toBe('mt_session=previous');
  expect(options.headers.Origin).toBe('http://127.0.0.1:3001');
  expect(options.headers['X-User-Role']).toBeUndefined();
  expect(result.headers.get('Set-Cookie')).toContain('HttpOnly; SameSite=Strict');
  expect(result.headers.get('Cache-Control')).toBe('no-store');
});
it('preserves denied PATCH requests instead of reporting success', async () => {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue(new Response('{"error":{"code":"FORBIDDEN"}}', { status: 403 })),
  );
  const result = await proxy(
    new Request('http://127.0.0.1:3001/ui/v1/users/other', {
      method: 'PATCH',
      body: '{"disabled":false}',
    }),
  );
  expect(result.status).toBe(403);
  expect((await result.json()).error.code).toBe('FORBIDDEN');
});
it('returns a useful error if the backend is unreachable', async () => {
  vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('internal details')));
  const result = await proxy(new Request('http://127.0.0.1:3001/auth/v1/session'));
  expect(result.status).toBe(502);
  expect((await result.json()).error.code).toBe('BACKEND_UNAVAILABLE');
});
