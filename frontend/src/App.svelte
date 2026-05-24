<script>
  import { onMount } from 'svelte';
  import { api } from './lib/api';
  import { busy, settings, status } from './lib/stores';
  import { validateSettings } from './lib/validate';

  import Dashboard from './Dashboard.svelte';
  import AddPortModal from './AddPortModal.svelte';

  let error = '';
  let form = {
    listen_addr: '127.0.0.1:8080',
    auto_discover: true,
  };

  let isAddModalOpen = false;

  async function refresh() {
    error = '';
    busy.set(true);
    try {
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

  async function runAction(action) {
    error = '';
    busy.set(true);
    try {
      await action();
      await refresh();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      busy.set(false);
    }
  }

  async function discover() {
    await runAction(() => api.discover());
  }

  async function openPort(portData) {
    await runAction(() => api.openPort());
    isAddModalOpen = false;
  }

  async function closePort(event) {
    const port = event.detail;
    await runAction(() => api.closePort());
  }

  async function save() {
    const errors = validateSettings(form);
    if (errors.length > 0) {
      error = errors.join(', ');
      return;
    }
    await runAction(() => api.saveSettings(form));
  }

  function handleAddPortSubmit(event) {
    openPort(event.detail);
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
  refresh={refresh}
  on:addPort={() => isAddModalOpen = true}
  on:closePort={closePort}
  on:settings={discover}
/>

{#if isAddModalOpen}
  <AddPortModal
    on:close={() => isAddModalOpen = false}
    on:submit={handleAddPortSubmit}
  />
{/if}
