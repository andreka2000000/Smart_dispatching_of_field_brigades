import type { Metadata } from 'next';
import { Providers } from './providers';
import { AppShell } from '@/components/layout/app-shell';
import './globals.css';

export const metadata: Metadata = {
  title: 'Магистраль — умная диспетчеризация выездных бригад',
  description: 'Рабочее место диспетчера и приложение выездного мастера',
};
export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="ru">
      <body>
        <Providers>
          <AppShell>{children}</AppShell>
        </Providers>
      </body>
    </html>
  );
}
