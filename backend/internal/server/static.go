package server

import (
	"bytes"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/clagon/port-mapper/backend/assets"
)

var assetsFS = assets.FS

func staticHandler() http.Handler {
	sub, err := fs.Sub(assetsFS, "static")
	if err != nil {
		return http.NotFoundHandler()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" || (!strings.Contains(path.Base(name), ".") && r.Method == http.MethodGet) {
			name = "index.html"
		}

		if strings.Contains(name, "..") {
			http.NotFound(w, r)
			return
		}

		data, err := fs.ReadFile(sub, path.Clean(name))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		http.ServeContent(w, r, path.Base(name), time.Time{}, bytes.NewReader(data))
	})
}
