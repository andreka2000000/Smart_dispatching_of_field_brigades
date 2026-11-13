import type { Metadata } from 'next';
import { Dispatcher } from '@/features/dispatch/dispatcher';
export const metadata: Metadata = { title: 'Диспетчерская' };
export default function Page() {
  return <Dispatcher />;
}
