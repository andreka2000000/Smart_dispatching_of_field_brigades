import 'server-only';
import { cookies } from 'next/headers';
import { redirect } from 'next/navigation';
import type { Role, User } from './types';
export async function currentUser(): Promise<User | null> {
  const session = (await cookies()).get('mt_session')?.value;
  if (!session) return null;
  try {
    const target = (process.env.API_PROXY_TARGET || 'http://127.0.0.1:8080').replace(/\/$/, '');
    const response = await fetch(`${target}/auth/v1/session`, {
      headers: { Cookie: `mt_session=${session}` },
      cache: 'no-store',
      signal: AbortSignal.timeout(5000),
    });
    return response.ok ? ((await response.json()) as User) : null;
  } catch {
    return null;
  }
}
export async function requireUser(roles?: Role[]): Promise<User> {
  const user = await currentUser();
  if (!user) redirect('/login');
  if (roles && !roles.includes(user.role)) redirect('/app/forbidden');
  return user;
}
