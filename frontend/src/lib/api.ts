import type { HealthResponse, Settings, StatusResponse } from './types';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  });
  if (!res.ok) {
    throw new Error(`${path} failed: ${res.status}`);
  }
  return (await res.json()) as T;
}

export const api = {
  health: () => request<HealthResponse>('/api/health'),
  status: () => request<StatusResponse>('/api/status'),
  discover: () => request<{ ok: boolean }>('/api/discover', { method: 'POST' }),
  openPort: () => request<{ ok: boolean }>('/api/ports/open', { method: 'POST' }),
  closePort: () => request<{ ok: boolean }>('/api/ports/close', { method: 'POST' }),
  getSettings: () => request<Settings>('/api/settings'),
  saveSettings: (settings: Settings) => request<{ ok: boolean }>('/api/settings', {
    method: 'POST',
    body: JSON.stringify(settings),
  }),
};
