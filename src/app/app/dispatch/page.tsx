import type { Metadata } from 'next';
import { LiveDispatcher } from '@/features/workspace/dispatcher';
import { requireUser } from '@/features/auth/server';
export const metadata: Metadata = { title: 'Диспетчерская' };
export default async function Page() {
  await requireUser(['DISPATCHER']);
  return <LiveDispatcher />;
}
