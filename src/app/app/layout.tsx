import { Providers } from '@/app/providers';
import { AppShell } from '@/components/layout/app-shell';
export default function WorkspaceLayout({ children }: { children: React.ReactNode }) {
  return (
    <Providers>
      <AppShell>{children}</AppShell>
    </Providers>
  );
}
