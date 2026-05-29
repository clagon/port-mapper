// Package assets は、フロントエンドのビルド済みSPA静的アセットをGoバイナリに埋め込むためのパッケージです。
package assets

import "embed"

// FS はフロントエンドの静的アセット（HTML, CSS, JS等）を保持する埋め込みファイルシステムです。
//go:embed static/**
var FS embed.FS
