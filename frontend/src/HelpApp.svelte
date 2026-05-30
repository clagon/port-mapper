<script lang="ts">
  import { marked } from 'marked';
  import usageMd from '../docs/usage.md?raw';
  import minecraftMd from '../docs/minecraft.md?raw';
  import securityMd from '../docs/security.md?raw';

  const docs = [
    { id: 'usage', title: '使い方ガイド', subtitle: 'Usage Guide', content: usageMd, icon: 'menu_book' },
    { id: 'minecraft', title: 'Minecraftの設定例', subtitle: 'Minecraft Server', content: minecraftMd, icon: 'sports_esports' },
    { id: 'security', title: '安全ガイド & FAQ', subtitle: 'Security & FAQ', content: securityMd, icon: 'shield' }
  ];

  let activeDocId = 'usage';

  $: activeDoc = docs.find(d => d.id === activeDocId) || docs[0];
  $: htmlContent = marked(activeDoc.content);
</script>

<div class="min-h-screen bg-background-page dark:bg-background-page flex flex-col font-body-md text-text-main antialiased">
  <!-- Top Bar -->
  <nav class="bg-background dark:bg-background border-b border-surface-variant/40 sticky top-0 z-50 shadow-sm">
    <div class="flex items-center justify-between px-6 py-4 max-w-7xl mx-auto w-full">
      <div class="flex items-center gap-3">
        <span class="material-symbols-outlined text-primary text-2xl" style="font-variation-settings: 'FILL' 1;">router</span>
        <span class="font-headline-md text-xl font-bold tracking-tight text-primary">Porto</span>
        <span class="h-4 w-[1px] bg-surface-variant"></span>
        <span class="font-label-sm text-xs text-text-muted uppercase tracking-wider">ヘルプ & ドキュメント</span>
      </div>
      <button class="text-secondary hover:text-text-main font-label-sm text-xs bg-surface-container-low px-4 py-2 rounded-full border border-surface-variant hover:bg-surface-container transition-all" on:click={() => window.close()}>
        ウィンドウを閉じる
      </button>
    </div>
  </nav>

  <!-- Container -->
  <div class="flex-grow max-w-7xl w-full mx-auto px-6 py-8 flex flex-col md:flex-row gap-8">
    <!-- Sidebar Navigation -->
    <aside class="w-full md:w-64 flex-shrink-0 flex flex-col gap-4">
      <div class="bg-surface-card rounded-2xl p-4 shadow-ambient-low border border-surface-variant/30 flex flex-col gap-2">
        <h2 class="font-label-sm text-xs text-text-muted uppercase tracking-wider px-3 mb-2">ドキュメント</h2>
        {#each docs as doc (doc.id)}
          <button
            class="w-full flex items-center gap-3 px-4 py-3 rounded-xl transition-all duration-200 text-left font-body-md {activeDocId === doc.id ? 'bg-primary-container text-on-primary-container font-semibold shadow-sm' : 'text-secondary hover:text-text-main hover:bg-surface-container-low'}"
            on:click={() => activeDocId = doc.id}
          >
            <span class="material-symbols-outlined text-xl {activeDocId === doc.id ? 'text-primary' : 'text-secondary'}">
              {doc.icon}
            </span>
            <div>
              <span class="block text-sm leading-none">{doc.title}</span>
              <span class="block text-[10px] opacity-70 mt-0.5 tracking-wide">{doc.subtitle}</span>
            </div>
          </button>
        {/each}
      </div>

      <!-- Quick Tips Card -->
      <div class="bg-gradient-to-br from-primary-container/20 to-surface-card rounded-2xl p-6 border border-primary-container/10 flex flex-col gap-3">
        <div class="w-8 h-8 rounded-full bg-primary-container flex items-center justify-center text-primary">
          <span class="material-symbols-outlined text-sm">lightbulb</span>
        </div>
        <h3 class="font-headline-md text-sm text-on-primary-container">ワンポイント</h3>
        <p class="font-label-sm text-[11px] text-secondary leading-relaxed">
          ポートマッピングは、ルーター側のUPnP機能が有効である必要があります。検出されない場合はルーターの設定画面をご確認ください。
        </p>
      </div>
    </aside>

    <!-- Main Content Panel -->
    <main class="flex-grow bg-surface-card rounded-3xl p-8 md:p-12 shadow-ambient-card border border-surface-variant/30 relative overflow-hidden">
      <!-- Glow background accent -->
      <div class="absolute -top-32 -right-32 w-80 h-80 bg-primary-fixed rounded-full blur-3xl opacity-10 pointer-events-none"></div>

      <!-- Document Header -->
      <div class="border-b border-surface-variant/40 pb-6 mb-8 relative z-10">
        <div class="flex items-center gap-2 text-primary mb-2">
          <span class="material-symbols-outlined text-lg">{activeDoc.icon}</span>
          <span class="font-label-sm text-xs uppercase tracking-wider">{activeDoc.subtitle}</span>
        </div>
        <h1 class="font-display-lg text-3xl text-text-main font-extrabold tracking-tight">{activeDoc.title}</h1>
      </div>

      <!-- Markdown Body -->
      <article class="markdown-body relative z-10">
        {@html htmlContent}
      </article>
    </main>
  </div>

  <!-- Footer -->
  <footer class="bg-surface-card border-t border-surface-variant/40 py-6 mt-auto">
    <div class="max-w-7xl mx-auto px-6 flex justify-between items-center text-[11px] text-text-muted">
      <span>&copy; 2024-2026 Porto. All rights reserved.</span>
      <span>Local-only & Private</span>
    </div>
  </footer>
</div>

<style>
  /* Local layout adjustments if needed */
  aside button {
    cursor: pointer;
    border: none;
  }
</style>
