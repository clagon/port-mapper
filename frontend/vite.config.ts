import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [svelte()],
  build: {
    outDir: '../backend/assets/static',
    emptyOutDir: false,
  },
  server: {
    host: '127.0.0.1',
    port: 5173,
  },
});
