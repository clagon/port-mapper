package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
)

// generateToken は、暗号論的に安全なランダムなバイト列を生成し、16進数（hex）エンコードされた文字列としてトークンを返します。
// n が 0 以下の場合は、デフォルトとして16バイト（32文字の16進数）のサイズを使用します。
func generateToken(n int) (string, error) {
	if n <= 0 {
		n = 16
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// isBearerToken は、HTTPリクエストから提供されたAuthorizationヘッダーの値が、期待されるBearerトークン形式と一致するかを安全に検証します。
func isBearerToken(expected, provided string) bool {
	expected = strings.TrimSpace(expected)
	provided = strings.TrimSpace(provided)
	if expected == "" || !strings.HasPrefix(provided, "Bearer ") {
		return false
	}
	got := strings.TrimSpace(strings.TrimPrefix(provided, "Bearer "))
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

// browserTokenMiddleware は、ローカルブラウザからの変更系APIリクエストだけを受け付けるための防御を追加します。
func browserTokenMiddleware(token, listenAddr string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			if !isBearerToken(token, req.Header.Get(echo.HeaderAuthorization)) {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid browser token")
			}
			if !isAllowedFetchSite(req.Header.Get("Sec-Fetch-Site")) {
				return echo.NewHTTPError(http.StatusForbidden, "cross-site request is not allowed")
			}
			if !isAllowedOrigin(req, listenAddr) {
				return echo.NewHTTPError(http.StatusForbidden, "cross-origin request is not allowed")
			}
			if !strings.HasPrefix(req.Header.Get(echo.HeaderContentType), echo.MIMEApplicationJSON) {
				return echo.NewHTTPError(http.StatusUnsupportedMediaType, "application/json is required")
			}
			return next(c)
		}
	}
}

func isAllowedFetchSite(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "none", "same-origin", "same-site":
		return true
	default:
		return false
	}
}

func isAllowedOrigin(req *http.Request, listenAddr string) bool {
	origin := strings.TrimSpace(req.Header.Get(echo.HeaderOrigin))
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	expectedScheme := "http"
	if req.TLS != nil {
		expectedScheme = "https"
	}
	return strings.EqualFold(u.Scheme, expectedScheme) && isAllowedOriginHost(u, listenAddr)
}

func isAllowedOriginHost(origin *url.URL, listenAddr string) bool {
	_, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return false
	}
	originHost := strings.ToLower(origin.Hostname())
	originPort := origin.Port()
	if originPort == "" {
		originPort = defaultPortForScheme(origin.Scheme)
	}
	if originPort != port {
		return false
	}
	switch originHost {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func defaultPortForScheme(scheme string) string {
	switch strings.ToLower(scheme) {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}
