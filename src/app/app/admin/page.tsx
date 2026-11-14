import { LiveAdmin } from '@/features/workspace/admin';
import { requireUser } from '@/features/auth/server';
export default async function Page() {
  await requireUser(['DISPATCHER']);
  return <LiveAdmin />;
}
