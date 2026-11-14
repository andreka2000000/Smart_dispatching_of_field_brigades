import { requireUser } from '@/features/auth/server';
import { RequestsPage } from '@/features/workspace/requests';
export const metadata = { title: 'Обращения абонентов' };
export default async function Page() {
  await requireUser(['DISPATCHER']);
  return <RequestsPage />;
}
