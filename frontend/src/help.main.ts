import HelpApp from './HelpApp.svelte';
import './app.css';
import '@fontsource/material-symbols-outlined';

const app = new HelpApp({
  target: document.getElementById('app') as HTMLElement,
});

export default app;
