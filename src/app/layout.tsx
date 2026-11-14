import type { Metadata } from 'next';
import localFont from 'next/font/local';
import './globals.css';

const manrope = localFont({
  src: [
    { path: '../../public/fonts/manrope-regular.ttf', weight: '400', style: 'normal' },
    { path: '../../public/fonts/manrope-bold.ttf', weight: '700', style: 'normal' },
  ],
  variable: '--font-manrope',
  display: 'swap',
});

export const metadata: Metadata = {
  title: {
    default: 'МаксТелеком — порядок там, где раньше был хаос',
    template: '%s · МаксТелеком',
  },
  description:
    'Заявки, команда и маршруты в одной системе. МаксТелеком помогает организовать работу выездных бригад и сделать сервис понятнее для каждого.',
};
export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="ru" className={manrope.variable}>
      <body>{children}</body>
    </html>
  );
}
