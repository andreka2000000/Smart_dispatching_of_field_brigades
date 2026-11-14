import { requireUser } from '@/features/auth/server';
import { RequestsPage } from '@/features/workspace/requests';
export const metadata = { title: 'Личный кабинет' };
export default async function Page() {
  await requireUser(['SUBSCRIBER']);
  return <RequestsPage />;
}
