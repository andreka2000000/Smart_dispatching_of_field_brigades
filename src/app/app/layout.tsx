import { Providers } from '@/app/providers';
import { AppShell } from '@/components/layout/app-shell';
import '@/features/workspace/workspace.css';
import '@/features/workspace/access.css';
import { requireUser } from '@/features/auth/server';
import { AuthProvider } from '@/features/auth/context';
export default async function WorkspaceLayout({ children }: { children: React.ReactNode }) {
  const user = await requireUser();
  return (
    <AuthProvider user={user}>
      <Providers>
        <AppShell>{children}</AppShell>
      </Providers>
    </AuthProvider>
  );
}
