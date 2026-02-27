import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(...inputs));
}

export function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: 'EUR',
  }).format(amount);
}

export function formatDate(date: Date | string): string {
  return new Intl.DateTimeFormat('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  }).format(new Date(date));
}

export function formatPhoneNumber(phone: string): string {
  const cleaned = phone.replace(/\\D/g, '');
  if (cleaned.startsWith('49')) {
    return '+49 ' + cleaned.slice(2, 4) + ' ' + cleaned.slice(4, 7) + ' ' + cleaned.slice(7);
  }
  return phone;
}