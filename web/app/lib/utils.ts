import type { ClassValue } from "clsx"
import { clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function errorMessage(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === 'string') return e;
  return 'An unknown error occurred';
}

export function formatShortId(value?: string | null, length = 8): string {
  const normalized = value?.trim();
  if (!normalized) return "--";
  if (normalized.length <= length) return normalized;
  return normalized.slice(0, length);
}

export async function copyToClipboard(value: string): Promise<boolean> {
  if (!value?.trim()) return false;
  if (typeof navigator === "undefined" || !navigator.clipboard?.writeText) {
    return false;
  }

  try {
    await navigator.clipboard.writeText(value);
    return true;
  } catch {
    return false;
  }
}
