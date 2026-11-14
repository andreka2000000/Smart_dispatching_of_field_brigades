// The target is server-only and resolved at runtime (also inside Docker).
export async function proxy(request: Request) {
  const url = new URL(request.url);
  const target = (process.env.API_PROXY_TARGET || 'http://127.0.0.1:8080').replace(/\/$/, '');
  try {
    const upstream = await fetch(`${target}${url.pathname}${url.search}`, {
      method: request.method,
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
        Cookie: request.headers.get('Cookie') || '',
        ...(request.headers.has('Origin') ? { Origin: request.headers.get('Origin')! } : {}),
        ...(request.headers.has('Sec-Fetch-Site')
          ? { 'Sec-Fetch-Site': request.headers.get('Sec-Fetch-Site')! }
          : {}),
      },
      body: ['GET', 'HEAD'].includes(request.method) ? undefined : await request.arrayBuffer(),
      cache: 'no-store',
      signal: AbortSignal.any([request.signal, AbortSignal.timeout(330_000)]),
    });
    const headers = new Headers({
      'Content-Type': 'application/json; charset=utf-8',
      'Cache-Control': 'no-store',
    });
    for (const value of upstream.headers.getSetCookie()) headers.append('Set-Cookie', value);
    return new Response(upstream.body, {
      status: upstream.status,
      headers,
    });
  } catch {
    return Response.json(
      {
        error: {
          code: 'BACKEND_UNAVAILABLE',
          message: 'Сервис временно недоступен. Проверьте запуск Docker и повторите запрос.',
        },
      },
      { status: 502 },
    );
  }
}
