# Porto

[English](README.md) | [日本語](README.ja.md)

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-00ADD8.svg?style=flat&logo=go)](https://go.dev)
[![Node Version](https://img.shields.io/badge/Node-18%2B-339933.svg?style=flat&logo=node.js)](https://nodejs.org)

**Porto** (pronounced: `/ˈpɔːr.toʊ/` 📣) は、Linux、macOS、Windows 向けに設計された、高品質で非常にシンプルかつ安全なローカル UPnP ポート開放ツールです。

ルーターのポートを一時的に開放するための美しく直感的なインターフェースを提供し、Minecraft などのゲームサーバーに友達を招待したり、ローカルの Web アプリを数秒で共有したりできるようにします。

> [!NOTE]
> Porto は、ローカルネットワークの親切な守護者として機能します。必要なときにだけポートを開放し、使い終わったらすぐに閉じることで、持続的な外部の脅威からコンピューターを安全に保ちます。

---

## ✨ 機能

- 🚀 **設定不要の UPnP**: ルーターを自動的に検出し、ワンクリックでポートをマッピングします。
- 🎨 **高品質なデザインの UI**: ダークモードとスムーズなアニメーションを備えた、洗練されたモダンな Svelte + Vite Web インターフェース。
- 🔒 **プライバシーファースト & ローカル専用**: UI はデフォルトで `127.0.0.1` (localhost) に厳密にバインドされます。設定がマシン外に出ることはありません。
- 🛠️ **シームレスな統合**: フロントエンドのすべてのアセットが直接埋め込まれた、単一バイナリの Go バックエンド。

---

## 🚀 クイックスタート

### 前提条件

Porto をビルドして実行するには、以下が必要です:
- **Go** 1.23 以降
- **Node.js** 18+ および **npm** (フロントエンドのビルド用)

---

### 1. フロントエンドのビルド

Svelte ベースの UI をコンパイルし、Go バックエンドのアセットディレクトリに出力します:

```bash
cd frontend
npm install
npm run build
```

フロントエンドのビルド出力は `backend/assets/static/` の下に生成され、Go のコンパイル時に埋め込まれます（Git では意図的に無視され、`.gitkeep` のみが残ります）。

---

### 2. アプリケーションの実行

Go バックエンドをコンパイルして実行します:

```bash
cd backend
go run ./cmd/port-mapper
```

起動時、Porto は以下の処理を行います:
1. `config.json` を自動的に読み込みます（見つからない場合は安全なデフォルト値を使用します）。
2. ローカルサーバーを `127.0.0.1:8080` で開始します。
3. デフォルトのブラウザでダッシュボードを自動的に開こうとします。

ブラウザが自動的に開かない場合は、こちらにアクセスしてください:
👉 **[http://127.0.0.1:8080](http://127.0.0.1:8080)**

---

## ⚙️ コマンドラインオプション

以下のコマンドラインフラグを使用して、Porto の動作をカスタマイズできます:

| フラグ | 説明 | デフォルト |
| :--- | :--- | :--- |
| `--listen-addr` | ローカル Web UI のホストとポートを指定します | `127.0.0.1:8080` |
| `--config` | `config.json` ファイルのカスタムパスを指定します | バイナリの横にある `config.json` |
| `--no-browser` | 起動時にブラウザを開かないようにします | `false` (ブラウザを開く) |

例:
```bash
go run ./cmd/port-mapper --listen-addr 127.0.0.1:9090 --no-browser
```

---

## 📖 ユーザーガイド & ドキュメント

Porto の使い方やネットワークを安全に保つ方法について詳しくは、組み込みのドキュメントをご覧ください:
- [使い方ガイド (Usage Guide)](frontend/docs/usage.md) ── 4つの簡単なステップで共有を開始する方法。
- [Minecraftの設定例 (Minecraft Guide)](frontend/docs/minecraft.md) ── 友達のために Java および Bedrock サーバーをセットアップする実践的なガイド。
- [安全ガイド & FAQ (Security & FAQ)](frontend/docs/security.md) ── Porto が安全である理由と、一般的な接続問題の解決方法。

---

## 🛠️ バックエンド API

Go バックエンドは、localhost 上で最小限の HTTP API を公開しています。

### ヘルスエンドポイント
- **URL**: `GET /api/health`
- **レスポンス**: `{"ok":true}`
