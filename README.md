# Porto

[English](README.md) | [日本語](README.ja.md)

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-00ADD8.svg?style=flat&logo=go)](https://go.dev)
[![Node Version](https://img.shields.io/badge/Node-18%2B-339933.svg?style=flat&logo=node.js)](https://nodejs.org)

**Porto** (pronounced: `/ˈpɔːr.toʊ/` 📣) is a premium, extremely simple, and secure local UPnP port-opening tool designed for Linux, macOS, and Windows.

It provides a beautiful and intuitive interface to temporarily open ports on your router, allowing you to invite friends to join your gaming servers (e.g., Minecraft) or share local web apps in seconds.

> [!NOTE]
> Porto acts as a friendly guardian of your local network: it helps you open ports only when needed, and close them as soon as you are done, keeping your computer safe from persistent external threats.

---

## ✨ Features

- 🚀 **Zero Configuration UPnP**: Automatically discovers your router and maps ports with a single click.
- 🎨 **Premium Aesthetic UI**: Sleek, modern Svelte + Vite web interface with dark mode and smooth animations.
- 🔒 **Privacy First & Local-Only**: The UI binds strictly to `127.0.0.1` (localhost) by default. Your configurations never leave your machine.
- 🛠️ **Seamless Integration**: Single-binary Go backend with the entire frontend assets embedded directly inside it.

---

## 🚀 Quick Start

### Prerequisites

To build and run Porto, you will need:
- **Go** 1.23 or newer
- **Node.js** 18+ and **npm** (for building the frontend)

---

### 1. Build the Frontend

Compile the Svelte-based UI into the Go backend's asset directory:

```bash
cd frontend
npm install
npm run build
```

The frontend build output is generated under `backend/assets/static/` and is embedded during Go compilation (intentionally ignored by Git, leaving only `.gitkeep`).

---

### 2. Run the Application

Compile and run the Go backend:

```bash
cd backend
go run ./cmd/port-mapper
```

Upon startup, Porto will:
1. Automatically load `config.json` (or use safe defaults if missing).
2. Start the local server on `127.0.0.1:8080`.
3. Try to open the dashboard automatically in your default browser.

If your browser does not open automatically, visit:
👉 **[http://127.0.0.1:8080](http://127.0.0.1:8080)**

---

## ⚙️ Command-Line Options

You can customize Porto's behavior using the following command-line flags:

| Flag | Description | Default |
| :--- | :--- | :--- |
| `--listen-addr` | Specify the host and port for the local web UI | `127.0.0.1:8080` |
| `--config` | Point to a custom path for the `config.json` file | `config.json` beside the binary |
| `--no-browser` | Prevent the app from opening the browser on startup | `false` (will open browser) |

Example:
```bash
go run ./cmd/port-mapper --listen-addr 127.0.0.1:9090 --no-browser
```

---

## 📖 User Guides & Documentation

To learn more about how to use Porto and keep your network safe, check out the built-in docs:
- [使い方ガイド (Usage Guide)](frontend/docs/usage.md) ── How to start sharing in 4 simple steps.
- [Minecraftの設定例 (Minecraft Guide)](frontend/docs/minecraft.md) ── Practical guide to set up Java and Bedrock servers for friends.
- [安全ガイド & FAQ (Security & FAQ)](frontend/docs/security.md) ── Learn why Porto is safe and solve common connection issues.

---

## 🛠️ Backend API

The Go backend exposes a minimal HTTP API on localhost.

### Health Endpoint
- **URL**: `GET /api/health`
- **Response**: `{"ok":true}`


