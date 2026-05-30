<script>
  import { createEventDispatcher } from 'svelte';

  export let listenAddr = '127.0.0.1:61234';
  export let autoDiscover = true;

  const dispatch = createEventDispatcher();

  let localAddr = listenAddr;
  let localAutoDiscover = autoDiscover;

  function submit() {
    dispatch('save', {
      listen_addr: localAddr,
      auto_discover: localAutoDiscover,
    });
  }

  function close() {
    dispatch('close');
  }
</script>

<div class="fixed inset-0 bg-black/50 z-[100] flex items-center justify-center p-4">
  <main class="w-full max-w-md mx-auto relative z-[101]">
    <button class="inline-flex items-center gap-2 text-white/80 hover:text-white bg-white/5 hover:bg-white/10 border border-white/10 px-4 py-2 rounded-full transition-all duration-200 mb-8 font-body-md group focus:outline-none focus:ring-2 focus:ring-white/20" on:click={close}>
      <span class="material-symbols-outlined group-hover:-translate-x-1 transition-transform duration-200 text-lg">arrow_back</span>
      ダッシュボードに戻る
    </button>

    <div class="bg-surface-card rounded-3xl p-8 md:p-10 shadow-ambient-low hover:shadow-ambient-hover transition-shadow duration-300 flex flex-col gap-8 relative overflow-hidden">
      <div class="absolute -top-24 -right-24 w-64 h-64 bg-primary-fixed rounded-full blur-3xl opacity-30 pointer-events-none"></div>

      <div class="text-center space-y-2 relative z-10">
        <div class="w-16 h-16 bg-surface-container-low rounded-full flex items-center justify-center mx-auto mb-4">
          <span class="material-symbols-outlined text-primary text-3xl" style="font-variation-settings: 'FILL' 1;">settings</span>
        </div>
        <h1 class="font-headline-md text-headline-md text-on-surface">設定</h1>
        <p class="font-body-md text-body-md text-secondary">ローカル設定を変更します。変更は config.json に保存されます。</p>
      </div>

      <form class="space-y-6 relative z-10" on:submit|preventDefault={submit}>
        <!-- Listen address -->
        <div class="space-y-3">
          <label class="block font-label-sm text-label-sm text-on-surface-variant" for="listenAddr">
            リッスンアドレス
          </label>
          <div class="relative input-glow rounded-xl transition-all duration-200 bg-[#F9F6F3]">
            <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
              <span class="material-symbols-outlined text-secondary">lan</span>
            </div>
            <input
              class="block w-full pl-12 pr-4 py-4 bg-transparent border-none rounded-xl font-body-md text-body-md text-on-surface focus:ring-0 placeholder:text-secondary-fixed-dim"
              id="listenAddr"
              name="listenAddr"
              bind:value={localAddr}
              placeholder="127.0.0.1:61234"
              required
              type="text"
            />
          </div>
          <p class="font-label-sm text-label-sm text-text-muted px-1">localhostのままにしておくとUIがプライベートに保たれます。</p>
        </div>

        <!-- Auto discover -->
        <div class="space-y-3">
          <div class="block font-label-sm text-label-sm text-on-surface-variant">起動時に自動検出</div>
          <label class="cursor-pointer flex items-center gap-4 p-4 rounded-xl border-2 transition-all duration-200 bg-[#F9F6F3] {localAutoDiscover ? 'border-primary bg-surface-container-low' : 'border-surface-variant'}">
            <div class="relative">
              <input class="sr-only peer" type="checkbox" bind:checked={localAutoDiscover} id="autoDiscover" />
              <div class="w-12 h-6 rounded-full transition-colors duration-200 {localAutoDiscover ? 'bg-primary' : 'bg-surface-variant'}"></div>
              <div class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform duration-200 {localAutoDiscover ? 'translate-x-6' : 'translate-x-0'}"></div>
            </div>
            <div>
              <span class="block font-body-md text-body-md text-on-surface">起動時にルーターを自動検出する</span>
              <span class="block font-label-sm text-label-sm text-secondary">手動で選択したい場合はオフにしてください。</span>
            </div>
          </label>
        </div>

        <div class="pt-6">
          <button
            class="w-full py-4 px-6 bg-primary text-on-primary rounded-full font-headline-md-mobile text-headline-md-mobile hover:bg-surface-tint hover:-translate-y-1 hover:shadow-ambient-hover transition-all duration-200 flex items-center justify-center gap-2 group relative overflow-hidden"
            type="submit"
          >
            <div class="absolute inset-0 border-2 border-white/20 rounded-full"></div>
            <span>設定を保存する</span>
            <span class="material-symbols-outlined group-hover:translate-x-1 transition-transform duration-200">save</span>
          </button>
        </div>
      </form>
    </div>
  </main>
</div>
