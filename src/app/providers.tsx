'use client';
import { useState } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { DemoProvider } from '@/lib/demo-context';
export function Providers({ children }: { children: React.ReactNode }) {
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: 10_000 } } }),
  );
  return (
    <QueryClientProvider client={client}>
      <DemoProvider>{children}</DemoProvider>
    </QueryClientProvider>
  );
}
