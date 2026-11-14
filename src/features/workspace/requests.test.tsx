import { renderToString } from 'react-dom/server';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { expect, it, vi } from 'vitest';
import { RequestsPage } from './requests';
const { user } = vi.hoisted(() => ({
  user: { id: 'test-owner', name: 'Абонент', role: 'SUBSCRIBER' },
}));
vi.mock('@/features/auth/context', () => ({ useIdentity: () => user }));
vi.mock('@/components/layout/page-header', () => ({ PageHeader: () => null }));
it.each([
  'SCHEDULED',
  'UNASSIGNED',
  'IN_PROGRESS',
  'COMPLETED',
  'CANCELLED',
  'STATION',
  'REQUIRES_REVIEW',
])('renders the subscriber lifecycle %s without requiring internal route data', (state) => {
  const client = new QueryClient({ defaultOptions: { queries: { staleTime: Infinity } } });
  client.setQueryData(
    ['service-requests', user.id],
    [
      {
        id: 'REQ-test',
        owner_name: 'Абонент',
        service: 'REPAIR',
        address: 'Учебный адрес 1',
        phone: '+79990000000',
        description: 'Проверить связь',
        preferred_date: '',
        status: 'APPROVED',
        decision_note: 'Окно согласовано',
        created_at: '2026-11-14T08:00:00Z',
        updated_at: '2026-11-14T08:00:00Z',
        order: {
          id: 'ORD-20261026-0002',
          date: '2026-10-26',
          state,
          ...(state === 'COMPLETED' ? { report: 'Связь восстановлена' } : {}),
        },
      },
    ],
  );
  const html = renderToString(
    <QueryClientProvider client={client}>
      <RequestsPage />
    </QueryClientProvider>,
  );
  expect(html).toContain('ORD-20261026-0002');
  expect(html).not.toContain('undefined');
  if (state === 'COMPLETED') expect(html).toContain('Связь восстановлена');
  expect(html).not.toContain('Одобрить и распределить');
});
