import type { Settings } from './types';

export function validateSettings(settings: Settings): string[] {
  const errors: string[] = [];
  if (!settings.listen_addr.trim()) {
    errors.push('listen_addr is required');
  }
  if (!settings.listen_addr.startsWith('127.0.0.1')) {
    errors.push('listen_addr must stay on localhost');
  }
  return errors;
}
