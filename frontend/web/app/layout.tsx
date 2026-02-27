import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import './globals.css';
import { Providers } from '@/components/providers';

const inter = Inter({ subsets: ['latin'] });

export const metadata: Metadata = {
  title: 'GruenFahrt - Nachhaltige Mitfahrgelegenheiten in Deutschland',
  description: 'Die umweltfreundliche Fahrt-Sharing-Plattform fuer Deutschland. Sicher, guenstig und nachhaltig unterwegs.',
  keywords: 'Mitfahrgelegenheit, Fahrt-Sharing, Deutschland, nachhaltig, umweltfreundlich',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="de">
      <body className={inter.className}>
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}