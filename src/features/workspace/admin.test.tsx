import type { ReactNode } from 'react';
import { renderToString } from 'react-dom/server';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { expect, it, vi } from 'vitest';
import { LiveAdmin } from './admin';
const { snapshot } = vi.hoisted(() => ({ snapshot: vi.fn() }));
vi.mock('./context', () => ({
  useWorkspace: () => ({ data: snapshot() }),
  WorkspaceGate: ({ children }: { children: ReactNode }) => children,
}));
vi.mock('@/components/layout/page-header', () => ({ PageHeader: () => null }));
vi.mock('./accounts', () => ({ AccountsPanel: () => null }));
vi.mock('./new-shift', () => ({ NewShift: () => null }));
it.each([undefined, [], ['ST-001']])(
  'renders the admin overview with absent_staff_ids=%j',
  (absent) => {
    // The Go response omits an empty absent_staff_ids slice (omitempty).
    snapshot.mockReturnValue({
      day: {
        date: '2026-10-29',
        now: '2026-10-29T08:30:00+05:00',
        orders: [],
        ...(absent === undefined ? {} : { absent_staff_ids: absent }),
      },
      reference: { staff: [{ staff_id: 'ST-001' }, { staff_id: 'ST-002' }], service_centers: [] },
      plan: null,
    });
    const html = renderToString(
      <QueryClientProvider client={new QueryClient()}>
        <LiveAdmin />
      </QueryClientProvider>,
    );
    expect(html).toContain('Всё, что держит сервис в порядке.');
    expect(html).toContain('Команда на смене');
    expect(html).toContain(`<strong>${2 - (absent?.length ?? 0)}</strong>`);
  },
);
