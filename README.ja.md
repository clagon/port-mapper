# Porto

[English](README.md) | [日本語](README.ja.md)

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-00ADD8.svg?style=flat&logo=go)](https://go.dev)
[![Node Version](https://img.shields.io/badge/Node-18%2B-339933.svg?style=flat&logo=node.js)](https://nodejs.org)

**Porto** (発音: `/ˈpɔːr.toʊ/` 📣) は、Linux、macOS、Windows 向けの、上質でとてもシンプルなローカル UPnP ポート開放ツールです。

ルーターのポートを一時的に開放するための直感的なインターフェースを備えており、Minecraft などのゲームサーバーに友達を招待したり、ローカルの Web アプリを数秒で共有したりできます。

> [!NOTE]
> Porto は、必要なときだけポートを開放し、使い終わったらすぐに閉じられるようにするためのツールです。ポートを開けたままにしない運用を手助けします。

---

## ✨ 機能

- 🚀 **設定不要の UPnP**: ルーターを自動検出し、ワンクリックでポートをマッピングできます。
- 🎨 **洗練された UI**: 落ち着いた配色と滑らかなアニメーションを備えた、モダンな Svelte + Vite 製 Web インターフェースです。
- 🔒 **プライバシー重視 & ローカル専用**: UI はデフォルトで `127.0.0.1` (localhost) のみにバインドされます。設定がこのマシンの外へ送信されることはありません。
- 🛠️ **一体化した構成**: フロントエンドの全アセットを直接埋め込んだ、単一バイナリの Go バックエンドです。

---

## 🚀 クイックスタート

### 前提条件

Porto をビルドして実行するには、以下が必要です。
- **Go** 1.23 以降
- **Node.js** 18 以降、および **npm** (フロントエンドのビルド用)

---

### 1. フロントエンドのビルド

Svelte ベースの UI をコンパイルし、Go バックエンドのアセットディレクトリへ出力します。

```bash
cd frontend
npm install
npm run build
```

フロントエンドのビルド成果物は `backend/assets/static/` 以下に生成され、Go のコンパイル時に埋め込まれます。

---

### 2. アプリケーションの実行

Go バックエンドをコンパイルして実行します。

```bash
cd backend
go run ./cmd/porto
```

起動すると、Porto は以下の処理を行います。
1. `config.json` を自動的に読み込みます（見つからない場合はデフォルト値が使用されます）。
2. `127.0.0.1:61234` でローカルサーバーを起動します。
3. 既定のブラウザでダッシュボードを自動的に開きます。

※ ブラウザが自動的に開かない場合は、こちらにアクセスしてください。`--listen-addr` でホストやポートを変更している場合は、その設定に合わせた URL を使用してください。
👉 **[http://127.0.0.1:61234](http://127.0.0.1:61234)**

---

## ⚙️ コマンドラインオプション

以下のコマンドラインフラグで、Porto の動作をカスタマイズできます。

| フラグ | 説明 | デフォルト |
| :--- | :--- | :--- |
| `--listen-addr` | ローカル Web UI のホストとポートを指定します | `127.0.0.1:61234` |
| `--config` | `config.json` ファイルの任意のパスを指定します | 実行ファイルと一緒にある `config.json` |
| `--no-browser` | 起動時にブラウザを開かないようにします | `false` (ブラウザを開く) |

例:
```bash
go run ./cmd/porto --listen-addr 127.0.0.1:9090 --no-browser
```

---

## 📖 ユーザーガイド & ドキュメント

Porto の使い方や接続時の注意点については、組み込みのドキュメントをご覧ください。アプリ起動後は、ヘッダーのヘルプアイコンからも開けます。
- [使い方ガイド (Usage Guide)](frontend/docs/usage.md) ── 4 つの簡単なステップで共有を始める方法。
- [Minecraft の設定例 (Minecraft Guide)](frontend/docs/minecraft.md) ── 友達向けに Java 版および Bedrock 版サーバーをセットアップするための実践的なガイド。
- [安全ガイド & FAQ (Security & FAQ)](frontend/docs/security.md) ── Porto の安全面の考え方と、よくある接続問題の解決方法。

---

## 🛠️ バックエンド API

Go バックエンドは、localhost 上で最小限の HTTP API を公開します。

### ヘルスエンドポイント
- **URL**: `GET /api/health`
- **レスポンス**: `{"ok":true}`
