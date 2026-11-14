import { LiveTechnician } from '@/features/workspace/technician';
import { requireUser } from '@/features/auth/server';
export default async function Page() {
  await requireUser(['TECHNICIAN']);
  return <LiveTechnician />;
}
