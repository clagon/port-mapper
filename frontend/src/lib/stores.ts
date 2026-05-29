import { writable } from 'svelte/store';
import type { Settings, StatusResponse } from './types';

export const status = writable<StatusResponse | null>(null);
export const settings = writable<Settings | null>(null);
export const busy = writable(false);
export const blocking = writable<string | boolean>(false);
