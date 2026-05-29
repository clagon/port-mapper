<script>
  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';
  import { api } from './lib/api';
  import { busy, blocking, settings, status } from './lib/stores';
  import { validateSettings } from './lib/validate';

  import Dashboard from './Dashboard.svelte';
  import AddPortModal from './AddPortModal.svelte';
  import SettingsModal from './SettingsModal.svelte';

  let error = '';
  let form = {
    listen_addr: '127.0.0.1:8080',
    auto_discover: true,
  };

  let isAddModalOpen = false;
  let isSettingsModalOpen = false;

  async function refresh() {
    error = '';
    busy.set(true);
    try {
      await api.discover();
      const [nextStatus, nextSettings] = await Promise.all([api.status(), api.getSettings()]);
      status.set(nextStatus);
      settings.set(nextSettings);
      form = nextSettings;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      busy.set(false);
    }
  }

  async function runAction(action, message = '処理を実行中...') {
    error = '';
    busy.set(true);
    blocking.set(message);
    try {
      await action();
      await refresh();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      busy.set(false);
      blocking.set(false);
    }
  }

  async function openPort(portData) {
    const mapping = {
      protocol: portData.protocol,
      external_port: portData.portNumber,
      internal_port: portData.portNumber,
      internal_ip: '',
      description: portData.appName,
      lease_duration_seconds: 0
    };
    await runAction(() => api.openPort(mapping), 'ポートを開放しています...');
    isAddModalOpen = false;
  }

  async function closePort(event) {
    const port = event.detail;
    await runAction(() => api.closePort({ external_port: port.external_port, protocol: port.protocol }), 'ポートを閉鎖しています...');
  }

  async function save() {
    const errors = validateSettings(form);
    if (errors.length > 0) {
      error = errors.join(', ');
      return;
    }
    await runAction(() => api.saveSettings(form), '設定を保存しています...');
  }

  function handleAddPortSubmit(event) {
    openPort(event.detail);
  }

  async function handleSettingsSave(event) {
    const newSettings = event.detail;
    form = newSettings;
    await save();
    isSettingsModalOpen = false;
  }

  onMount(refresh);
</script>

{#if error}
  <div class="fixed top-4 left-1/2 -translate-x-1/2 bg-error text-on-error px-6 py-3 rounded-xl shadow-ambient-hover z-[200] max-w-md w-full text-center font-body-md" role="alert">
    {error}
    <button class="absolute top-1/2 right-4 -translate-y-1/2 text-on-error opacity-80 hover:opacity-100" on:click={() => error = ''}>
      <span class="material-symbols-outlined text-sm">close</span>
    </button>
  </div>
{/if}

<Dashboard
  status={$status}
  busy={$busy}
  refresh={refresh}
  on:addPort={() => isAddModalOpen = true}
  on:closePort={closePort}
  on:settings={() => isSettingsModalOpen = true}
/>

{#if isAddModalOpen}
  <AddPortModal
    on:close={() => isAddModalOpen = false}
    on:submit={handleAddPortSubmit}
  />
{/if}

{#if isSettingsModalOpen}
  <SettingsModal
    listenAddr={form.listen_addr}
    autoDiscover={form.auto_discover}
    on:close={() => isSettingsModalOpen = false}
    on:save={handleSettingsSave}
  />
{/if}

{#if $busy && !$blocking}
  <div class="fixed top-0 left-0 right-0 h-[3px] bg-surface-container-low overflow-hidden z-[200]" transition:fade={{ duration: 150 }}>
    <div class="h-full bg-gradient-to-r from-primary to-primary-container rounded-r-full animate-progress-infinite"></div>
  </div>
{/if}

{#if $blocking}
  <div class="fixed inset-0 bg-background/60 backdrop-blur-md z-[300] flex items-center justify-center flex-col gap-6" transition:fade={{ duration: 200 }}>
    <div class="relative w-16 h-16">
      <div class="absolute inset-0 rounded-full border-4 border-primary/20 animate-ping opacity-75"></div>
      <div class="absolute inset-0 rounded-full border-4 border-t-primary border-r-primary border-b-transparent border-l-transparent animate-spin"></div>
    </div>
    <div class="text-center space-y-2">
      <h3 class="font-headline-sm text-headline-sm text-text-main animate-pulse">{$blocking}</h3>
      <p class="font-label-sm text-label-sm text-text-muted">ネットワークの設定を更新しています。少々お待ちください。</p>
    </div>
  </div>
{/if}
