<script>
  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';
  import { api } from './lib/api';
  import { busy, blocking, settings, status } from './lib/stores';
  import { validateSettings } from './lib/validate';
  import { parseError } from './lib/error';

  import Dashboard from './Dashboard.svelte';
  import AddPortModal from './AddPortModal.svelte';
  import SettingsModal from './SettingsModal.svelte';

  let error = '';
  let isErrorDetailsOpen = false;
  let form = {
    listen_addr: '127.0.0.1:8080',
    auto_discover: true,
  };

  $: friendlyError = parseError(error);
  $: if (!error) isErrorDetailsOpen = false;

  let isAddModalOpen = false;
  let isSettingsModalOpen = false;
  let editingPort = null;

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

  async function handleAddPortSubmit(event) {
    const portData = event.detail;
    if (editingPort) {
      await runAction(async () => {
        await api.closePort({
          external_port: editingPort.external_port,
          protocol: editingPort.protocol
        });
        const mapping = {
          protocol: portData.protocol,
          external_port: portData.portNumber,
          internal_port: portData.portNumber,
          internal_ip: '',
          description: portData.appName,
          lease_duration_seconds: 0
        };
        await api.openPort(mapping);
      }, 'ポートの設定を変更しています...');
      editingPort = null;
    } else {
      await openPort(portData);
    }
    isAddModalOpen = false;
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
  <div class="fixed top-6 left-1/2 -translate-x-1/2 bg-surface-card text-on-surface rounded-2xl shadow-ambient-hover z-[200] max-w-md w-[90%] p-6 border-l-4 border-error flex flex-col gap-4 font-body-md" role="alert" transition:fade={{ duration: 150 }}>
    <div class="flex items-start gap-4">
      <div class="w-10 h-10 rounded-full bg-error-container text-error flex items-center justify-center flex-shrink-0">
        <span class="material-symbols-outlined text-error" style="font-variation-settings: 'FILL' 1;">warning</span>
      </div>
      <div class="flex-grow space-y-1">
        <h3 class="font-headline-sm text-headline-sm text-error">{friendlyError.title}</h3>
        <p class="font-body-md text-body-md text-secondary leading-relaxed">{friendlyError.message}</p>
      </div>
      <button class="text-secondary hover:text-text-main flex-shrink-0 p-0 w-8 h-8 flex items-center justify-center rounded-full bg-transparent border-none hover:bg-surface-container transition-all" on:click={() => error = ''} aria-label="エラーを閉じる">
        <span class="material-symbols-outlined text-sm">close</span>
      </button>
    </div>

    <!-- 詳細アコーディオン -->
    {#if friendlyError.details}
      <div class="border-t border-surface-variant pt-3 mt-1">
        <button class="w-full text-left flex justify-between items-center text-label-sm font-label-sm text-secondary hover:text-text-main transition-colors" on:click={() => isErrorDetailsOpen = !isErrorDetailsOpen}>
          <span>詳細なエラー情報</span>
          <span class="material-symbols-outlined text-sm transition-transform duration-200" style="transform: rotate({isErrorDetailsOpen ? 180 : 0}deg)">expand_more</span>
        </button>
        {#if isErrorDetailsOpen}
          <div class="mt-3 bg-surface-container-low rounded-lg p-3 text-label-sm font-label-sm font-mono break-all text-secondary overflow-y-auto max-h-32 select-all selection:bg-primary/20">
            {friendlyError.details}
          </div>
        {/if}
      </div>
    {/if}
  </div>
{/if}

<Dashboard
  status={$status}
  busy={$busy}
  refresh={refresh}
  on:addPort={() => { editingPort = null; isAddModalOpen = true; }}
  on:editPort={(event) => { editingPort = event.detail; isAddModalOpen = true; }}
  on:closePort={closePort}
  on:settings={() => isSettingsModalOpen = true}
/>

{#if isAddModalOpen}
  <AddPortModal
    port={editingPort}
    on:close={() => { isAddModalOpen = false; editingPort = null; }}
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
