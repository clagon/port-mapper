import type { HealthResponse, PortMapping, Settings, StatusResponse } from './types';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  });
  if (!res.ok) {
    let errMsg = `${path} failed: ${res.status}`;
    try {
      const rawText = await res.text();
      if (rawText) {
        try {
          const errObj = JSON.parse(rawText);
          if (errObj && typeof errObj === 'object' && 'message' in errObj) {
            errMsg = String(errObj.message);
          } else {
            errMsg = rawText;
          }
        } catch {
          errMsg = rawText;
        }
      }
    } catch {
      errMsg = `${path} failed: ${res.status}`;
    }
    throw new Error(errMsg);
  }
  return (await res.json()) as T;
}

export const api = {
  health: () => request<HealthResponse>('/api/health'),
  status: () => request<StatusResponse>('/api/status'),
  discover: () => request<StatusResponse>('/api/discover', { method: 'POST' }),
  openPort: (mapping: PortMapping) => request<StatusResponse>('/api/ports/open', {
    method: 'POST',
    body: JSON.stringify(mapping),
  }),
  closePort: (mapping: PortMapping) => request<StatusResponse>('/api/ports/close', {
    method: 'POST',
    body: JSON.stringify(mapping),
  }),
  getSettings: () => request<Settings>('/api/settings'),
  saveSettings: (settings: Settings) => request<{ ok: boolean }>('/api/settings', {
    method: 'POST',
    body: JSON.stringify(settings),
  }),
};
