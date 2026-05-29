// Package auth は Porto のHTTP APIセキュリティのためのトークン生成および検証ユーティリティを提供します。
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// GenerateToken は、暗号論的に安全なランダムなバイト列を生成し、16進数（hex）エンコードされた文字列としてトークンを返します。
// n が 0 以下の場合は、デフォルトとして16バイト（32文字の16進数）のサイズを使用します。
func GenerateToken(n int) (string, error) {
	if n <= 0 {
		n = 16
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// IsBearerToken は、HTTPリクエストから提供されたAuthorizationヘッダーの値が、期待されるBearerトークン形式と一致するかを安全に検証します。
func IsBearerToken(expected, provided string) bool {
	return strings.TrimSpace(expected) != "" && strings.TrimSpace(provided) == "Bearer "+strings.TrimSpace(expected)
}
