<script>
  import { onMount } from 'svelte';
  import { api } from './lib/api';
  import { busy, settings, status } from './lib/stores';
  import { validateSettings } from './lib/validate';

  let error = '';
  let form = {
    listen_addr: '127.0.0.1:8080',
    auto_discover: true,
  };

  async function refresh() {
    error = '';
    busy.set(true);
    try {
      const [s, cfg] = await Promise.all([api.status(), api.getSettings()]);
      status.set(s);
      settings.set(cfg);
      form = cfg;
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      busy.set(false);
    }
  }

  async function discover() {
    error = '';
    busy.set(true);
    try {
      await api.discover();
      await refresh();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      busy.set(false);
    }
  }

  async function save() {
    const errors = validateSettings(form);
    if (errors.length > 0) {
      error = errors.join(', ');
      return;
    }
    error = '';
    busy.set(true);
    try {
      await api.saveSettings(form);
      await refresh();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      busy.set(false);
    }
  }

  onMount(refresh);
</script>

<main>
  <div class="grid">
    <section class="card">
      <div class="row" style="justify-content: space-between; align-items: center;">
        <div>
          <h1>port-mapper</h1>
          <p class="muted">Local-only UPnP port mapper UI</p>
        </div>
        <div class="row">
          <button on:click={refresh}>Refresh</button>
          <button on:click={discover}>Discover</button>
        </div>
      </div>
      {#if error}
        <p class="warn">{error}</p>
      {/if}
    </section>

    <section class="card">
      <h2>Status</h2>
      {#if $status}
        <p class={$status.discovered ? 'ok' : 'muted'}>
          {$status.discovered ? 'Router discovered' : 'No router selected yet'}
        </p>
        <pre>{JSON.stringify($status, null, 2)}</pre>
      {/if}
    </section>

    <section class="card">
      <h2>Settings</h2>
      <div class="grid">
        <label>
          <div class="muted">Listen address</div>
          <input bind:value={form.listen_addr} />
        </label>
        <label>
          <input type="checkbox" bind:checked={form.auto_discover} /> Auto discover on startup
        </label>
        <div class="row">
          <button on:click={save}>Save settings</button>
        </div>
      </div>
    </section>
  </div>
</main>
