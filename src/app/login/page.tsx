import { redirect } from 'next/navigation';
import { currentUser } from '@/features/auth/server';
import { roleHome } from '@/features/auth/types';
import { Login } from '@/features/auth/login';
export const metadata = { title: 'Вход' };
export default async function Page() {
  const user = await currentUser();
  if (user) redirect(roleHome[user.role]);
  return <Login />;
}
