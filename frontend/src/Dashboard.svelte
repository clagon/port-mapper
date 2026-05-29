<script>
  import { createEventDispatcher } from 'svelte';
  export let status;
  export let refresh;
  export let busy = false;

  const dispatch = createEventDispatcher();

  $: portCount = Array.isArray(status?.ports) ? status.ports.length : 0;
</script>

<!-- TopNavBar -->
<nav class="bg-background dark:bg-background docked full-width top-0 text-primary dark:text-primary-fixed-dim z-50">
  <div class="flex justify-between items-center px-margin-page py-6 w-full max-w-7xl mx-auto">
    <!-- Brand -->
    <div class="text-headline-md font-headline-md font-bold text-primary dark:text-primary-fixed-dim flex items-center gap-2">
      <span class="material-symbols-outlined" style="font-variation-settings: 'FILL' 1;">router</span>
      PortMapper
    </div>
    <!-- Navigation Links -->
    <div class="hidden md:flex items-center gap-8">
      <a class="text-primary dark:text-primary-fixed-dim font-bold border-b-2 border-primary dark:border-primary-fixed-dim pb-1 hover:text-primary-container dark:hover:text-primary-fixed transition-colors duration-200" href="#">Dashboard</a>
      <a class="text-secondary dark:text-secondary-fixed-dim font-body-md hover:text-primary-container dark:hover:text-primary-fixed transition-colors duration-200" href="#">Logs</a>
    </div>
    <!-- Trailing Actions -->
    <div class="flex items-center gap-4">
      <span class="hidden md:inline-block font-label-sm text-label-sm text-status-active bg-surface-container px-3 py-1 rounded-full">Status: Active</span>
      <button aria-label="Settings" class="text-secondary hover:text-primary-container transition-colors duration-200 focus-ring-soft rounded-full p-1" on:click={() => dispatch('settings')}>
        <span class="material-symbols-outlined">settings</span>
      </button>
      <button aria-label="Help" class="text-secondary hover:text-primary-container transition-colors duration-200 focus-ring-soft rounded-full p-1">
        <span class="material-symbols-outlined">help</span>
      </button>
    </div>
  </div>
</nav>

<!-- Main Content Canvas -->
<main class="flex-grow w-full max-w-7xl mx-auto px-margin-page py-8 flex flex-col gap-gutter">
  <!-- Header -->
  <header class="flex justify-between items-end mb-4">
    <div>
      <h1 class="font-display-lg text-display-lg text-text-main mb-2">おはようございます</h1>
      <p class="font-body-md text-text-muted">ネットワークは現在安全で、最適化されています。</p>
    </div>
    <button 
      class="bg-surface-container-low text-primary px-4 py-2 rounded-full font-label-sm hover:bg-surface-container transition-colors duration-200 flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed" 
      on:click={refresh}
      disabled={busy}
    >
      {#if busy}
        <span class="material-symbols-outlined text-sm animate-spin">sync</span>
        更新中...
      {:else}
        更新
      {/if}
    </button>
  </header>

  <!-- Top Bento Grid: Hero & Main Action -->
  <div class="grid grid-cols-1 md:grid-cols-12 gap-gutter">
    <!-- Hero Status Card (Spans 8 cols) -->
    <div class="md:col-span-8 bg-surface-card rounded-xl p-card-padding shadow-ambient-card flex flex-col justify-between relative overflow-hidden group">
      <div class="absolute -right-20 -top-20 w-64 h-64 bg-surface-container rounded-full opacity-50 transition-transform duration-500 group-hover:scale-110 pointer-events-none"></div>

      <div class="relative z-10 flex flex-col h-full gap-stack-gap">
        <div class="flex items-center gap-3">
          <span class="material-symbols-outlined text-status-active text-3xl">verified_user</span>
          <h2 class="font-headline-md text-headline-md text-text-main">システムステータス</h2>
        </div>

        <div class="mt-4">
          <div class="inline-flex items-center gap-2 bg-surface-container-low px-4 py-2 rounded-full border border-surface-dim mb-6">
            {#if busy}
              <div class="w-3 h-3 rounded-full bg-primary animate-pulse"></div>
              <span class="font-label-sm text-label-sm text-text-main">ルーター探索中...</span>
            {:else}
              <div class="w-3 h-3 rounded-full {status?.discovered ? 'bg-status-active animate-pulse' : 'bg-status-warning'}"></div>
              <span class="font-label-sm text-label-sm text-text-main">{status?.discovered ? 'システム準備完了' : 'ルーター未検出'}</span>
            {/if}
          </div>
        </div>

        <div class="flex flex-col md:flex-row gap-8 mt-auto pt-4 border-t border-surface-variant">
          <div>
            <p class="font-label-sm text-label-sm text-text-muted mb-1">ローカル IP</p>
            <p class="font-body-lg text-body-lg font-semibold text-text-main">{status?.local_ip || '---.---.---.---'}</p>
          </div>
          <div>
            <p class="font-label-sm text-label-sm text-text-muted mb-1">パブリック IP</p>
            <p class="font-body-lg text-body-lg font-semibold text-text-main">{status?.external_ip || '---.---.---.---'}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Main Action Card (Spans 4 cols) -->
    <button class="md:col-span-4 bg-primary-container text-on-primary-container rounded-xl p-card-padding shadow-ambient-card shadow-ambient-button flex flex-col items-center justify-center gap-4 hover:bg-primary transition-colors duration-300 focus-ring-soft text-center group" on:click={() => dispatch('addPort')}>
      <div class="w-16 h-16 rounded-full bg-surface-card flex items-center justify-center text-primary-container group-hover:scale-110 transition-transform duration-300 shadow-sm">
        <span class="material-symbols-outlined text-4xl" style="font-variation-settings: 'wght' 600;">add</span>
      </div>
      <div>
        <h3 class="font-headline-md text-headline-md mb-1">ポートを開放する</h3>
        <p class="font-label-sm text-label-sm opacity-80">新しいルールの追加</p>
      </div>
    </button>
  </div>

  <!-- Section: Active Mappings -->
  <section class="mt-8">
    <div class="flex justify-between items-center mb-6">
      <h2 class="font-headline-md text-headline-md text-text-main">アクティブなポート</h2>
      <span class="font-label-sm text-label-sm text-primary px-2 py-1">
        {portCount} 個
      </span>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-gutter">
      {#if portCount > 0}
        {#each status.ports as port, index}
          <div class="bg-surface-card rounded-xl p-card-padding shadow-ambient-card flex flex-col gap-4 border border-transparent hover:border-surface-variant transition-colors">
            <div class="flex justify-between items-start">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-full bg-surface-container flex items-center justify-center text-primary">
                  <span class="material-symbols-outlined" style="font-variation-settings: 'FILL' 1;">
                    {port.protocol === 'UDP' ? 'sports_esports' : 'public'}
                  </span>
                </div>
                <div>
                  <h3 class="font-body-md text-body-md font-semibold text-text-main">{port.description || `Mapping ${index + 1}`}</h3>
                  <p class="font-label-sm text-label-sm text-text-muted">{port.protocol}</p>
                </div>
              </div>
              <div class="w-2 h-2 rounded-full bg-status-active"></div>
            </div>

            <div class="bg-background-page rounded-lg p-3 flex justify-between items-center mt-2">
              <span class="font-label-sm text-label-sm text-text-muted">ポート番号</span>
              <span class="font-body-md text-body-md font-bold text-text-main tracking-wider">{port.external_port}</span>
            </div>

            <div class="mt-auto pt-2 flex gap-2">
              <button class="flex-1 bg-surface-card border-2 border-surface-variant text-error font-label-sm text-label-sm rounded-full py-2 hover:bg-error-container hover:border-error-container transition-all focus-ring-soft" on:click={() => dispatch('closePort', port)}>
                停止
              </button>
            </div>
          </div>
        {/each}
      {/if}

      <button class="bg-transparent border-2 border-dashed border-outline-variant rounded-xl p-card-padding flex flex-col items-center justify-center gap-3 hover:border-primary-container hover:bg-surface-container-lowest transition-all focus-ring-soft text-text-muted hover:text-primary-container group min-h-[220px]" on:click={() => dispatch('addPort')}>
        <span class="material-symbols-outlined text-3xl opacity-50 group-hover:opacity-100 transition-opacity">add_circle</span>
        <span class="font-body-md text-body-md font-semibold">新しいポートを追加</span>
      </button>
    </div>
  </section>
</main>

<footer class="bg-transparent full-width bottom-0 mt-auto">
  <div class="flex flex-col md:flex-row justify-between items-center px-margin-page py-8 w-full max-w-7xl mx-auto gap-stack-gap border-t border-surface-variant">
    <div class="font-label-sm text-label-sm font-bold text-primary opacity-80 hover:opacity-100 transition-opacity">
      PortMapper
    </div>
    <div class="font-label-sm text-label-sm text-secondary dark:text-secondary-fixed-dim">
      © 2024 PortMapper • Your Helpful Network Guardian
    </div>
    <div class="flex gap-4">
      <a class="font-label-sm text-label-sm text-secondary dark:text-secondary-fixed-dim hover:text-primary transition-colors opacity-80 hover:opacity-100" href="#">Documentation</a>
      <a class="font-label-sm text-label-sm text-secondary dark:text-secondary-fixed-dim hover:text-primary transition-colors opacity-80 hover:opacity-100" href="#">System Health</a>
      <a class="font-label-sm text-label-sm text-secondary dark:text-secondary-fixed-dim hover:text-primary transition-colors opacity-80 hover:opacity-100" href="#">Support</a>
    </div>
  </div>
</footer>
