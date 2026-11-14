import { redirect } from 'next/navigation';
import { requireUser } from '@/features/auth/server';
import { roleHome } from '@/features/auth/types';
export default async function Page() {
  const user = await requireUser();
  redirect(roleHome[user.role]);
}
