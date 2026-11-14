'use client';
import { useState } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { WorkspaceProvider } from '@/features/workspace/context';
import { useIdentity } from '@/features/auth/context';
export function Providers({ children }: { children: React.ReactNode }) {
  const user = useIdentity();
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: 10_000 } } }),
  );
  return (
    <QueryClientProvider client={client}>
      {user.role === 'SUBSCRIBER' ? children : <WorkspaceProvider>{children}</WorkspaceProvider>}
    </QueryClientProvider>
  );
}
