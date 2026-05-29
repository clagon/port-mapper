package server

import (
	"bytes"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/clagon/port-mapper/backend/assets"
	"github.com/labstack/echo/v4"
)

// assetsFS はフロントエンドの静的ファイルを格納した組み込みファイルシステムへの参照であり、テスト時にモックに置換可能です。
var assetsFS fs.FS = assets.FS

// staticHandler は、埋め込まれたフロントエンドSPA静的ファイルを配信し、存在しないルートへのリクエスト時には `index.html` にフォールバックするSPA用ルーターハンドラーを提供します。
func staticHandler() echo.HandlerFunc {
	sub, err := fs.Sub(assetsFS, "static")
	if err != nil {
		return func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusNotFound)
		}
	}

	return func(c echo.Context) error {
		name := strings.TrimPrefix(c.Request().URL.Path, "/")
		if name == "" || (!strings.Contains(path.Base(name), ".") && c.Request().Method == http.MethodGet) {
			name = "index.html"
		}
		if strings.Contains(name, "..") {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		data, err := fs.ReadFile(sub, path.Clean(name))
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		http.ServeContent(c.Response().Writer, c.Request(), path.Base(name), time.Time{}, bytes.NewReader(data))
		return nil
	}
}
