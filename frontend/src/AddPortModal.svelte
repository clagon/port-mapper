<script>
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  let appName = '';
  let portNumber = '';
  let protocol = 'tcp';

  function submit() {
    dispatch('submit', {
      appName,
      portNumber: parseInt(portNumber, 10),
      protocol
    });
  }

  function close() {
    dispatch('close');
  }
</script>

<div class="fixed inset-0 bg-black/50 z-[100] flex items-center justify-center p-4">
  <main class="w-full max-w-md mx-auto relative z-[101]">
    <button class="inline-flex items-center gap-2 text-surface-card hover:text-primary transition-colors duration-200 mb-8 font-body-md group" on:click={close}>
      <span class="material-symbols-outlined group-hover:-translate-x-1 transition-transform duration-200">arrow_back</span>
      ダッシュボードに戻る
    </button>

    <div class="bg-surface-card rounded-3xl p-8 md:p-10 shadow-ambient-low hover:shadow-ambient-hover transition-shadow duration-300 flex flex-col gap-8 relative overflow-hidden">
      <div class="absolute -top-24 -right-24 w-64 h-64 bg-primary-fixed rounded-full blur-3xl opacity-30 pointer-events-none"></div>

      <div class="text-center space-y-2 relative z-10">
        <div class="w-16 h-16 bg-surface-container-low rounded-full flex items-center justify-center mx-auto mb-4">
          <span class="material-symbols-outlined text-primary text-3xl" style="font-variation-settings: 'FILL' 1;">add_link</span>
        </div>
        <h1 class="font-headline-md text-headline-md text-on-surface">新しいポートを追加</h1>
        <p class="font-body-md text-body-md text-secondary">共有したいゲームやアプリの情報を教えてください。</p>
      </div>

      <form class="space-y-6 relative z-10" on:submit|preventDefault={submit}>
        <div class="space-y-3">
          <label class="block font-label-sm text-label-sm text-on-surface-variant" for="appName">アプリ名（またはゲーム名）</label>
          <div class="relative input-glow rounded-xl transition-all duration-200 bg-[#F9F6F3]">
            <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
              <span class="material-symbols-outlined text-secondary">videogame_asset</span>
            </div>
            <input class="block w-full pl-12 pr-4 py-4 bg-transparent border-none rounded-xl font-body-md text-body-md text-on-surface focus:ring-0 placeholder:text-secondary-fixed-dim" id="appName" name="appName" bind:value={appName} placeholder="例: Minecraft サーバー" required type="text"/>
          </div>
        </div>

        <div class="space-y-3">
          <label class="block font-label-sm text-label-sm text-on-surface-variant" for="portNumber">ポート番号は？</label>
          <div class="relative input-glow rounded-xl transition-all duration-200 bg-[#F9F6F3]">
            <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
              <span class="material-symbols-outlined text-secondary">tag</span>
            </div>
            <input class="block w-full pl-12 pr-4 py-4 bg-transparent border-none rounded-xl font-body-md text-body-md text-on-surface focus:ring-0 placeholder:text-secondary-fixed-dim" id="portNumber" name="portNumber" bind:value={portNumber} placeholder="例: 25565" required type="number" min="1" max="65535"/>
          </div>
          <p class="font-label-sm text-label-sm text-text-muted px-1">通常、アプリの説明書や設定画面に記載されています。</p>
        </div>

        <div class="space-y-3 pt-2">
          <div class="block font-label-sm text-label-sm text-on-surface-variant">通信の種類（プロトコル）</div>
          <div class="grid grid-cols-2 gap-4">
            <label class="cursor-pointer relative">
              <input class="peer sr-only" name="protocol" type="radio" value="tcp" bind:group={protocol}/>
              <div class="p-4 rounded-xl border-2 border-surface-variant bg-surface-card text-center transition-all duration-200 peer-checked:border-primary peer-checked:bg-surface-container-low peer-checked:text-primary hover:bg-surface-container-lowest hover:shadow-sm">
                <span class="block font-headline-md-mobile text-headline-md-mobile mb-1">TCP</span>
                <span class="block font-label-sm text-label-sm text-secondary peer-checked:text-primary-container">一般的な通信</span>
              </div>
            </label>
            <label class="cursor-pointer relative">
              <input class="peer sr-only" name="protocol" type="radio" value="udp" bind:group={protocol}/>
              <div class="p-4 rounded-xl border-2 border-surface-variant bg-surface-card text-center transition-all duration-200 peer-checked:border-primary peer-checked:bg-surface-container-low peer-checked:text-primary hover:bg-surface-container-lowest hover:shadow-sm">
                <span class="block font-headline-md-mobile text-headline-md-mobile mb-1">UDP</span>
                <span class="block font-label-sm text-label-sm text-secondary peer-checked:text-primary-container">ゲームや音声</span>
              </div>
            </label>
          </div>
        </div>

        <div class="pt-6">
          <button class="w-full py-4 px-6 bg-primary text-on-primary rounded-full font-headline-md-mobile text-headline-md-mobile hover:bg-surface-tint hover:-translate-y-1 hover:shadow-ambient-hover transition-all duration-200 flex items-center justify-center gap-2 group relative overflow-hidden" type="submit">
            <div class="absolute inset-0 border-2 border-white/20 rounded-full"></div>
            <span>共有を開始する</span>
            <span class="material-symbols-outlined group-hover:translate-x-1 transition-transform duration-200">arrow_forward</span>
          </button>
        </div>
      </form>
    </div>
  </main>
</div>
