// Package browseropener は OS ごとの標準ブラウザの自動起動ユーティリティを提供します。
package browseropener

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Opener は、ユーザーの既定のWebブラウザで指定URLを開くためのオブジェクトです。
type Opener struct{}

// New は、マルチプラットフォーム（Windows, macOS, Linux）対応のブラウザ起動オブジェクトを生成します。
func New() Opener { return Opener{} }

// Open は、実行中のプラットフォームに対応したOSコマンドを非同期的に実行し、Webブラウザで指定URLを開きます。
func (Opener) Open(url string) error {
	cmdName, args, err := commandForGOOS(runtime.GOOS, url)
	if err != nil {
		return err
	}
	return exec.Command(cmdName, args...).Start()
}

func commandForGOOS(goos, url string) (string, []string, error) {
	switch goos {
	case "windows":
		return "cmd", []string{"/c", "start", "", url}, nil
	case "darwin":
		return "open", []string{url}, nil
	case "linux":
		return "xdg-open", []string{url}, nil
	default:
		return "", nil, fmt.Errorf("unsupported platform %q", goos)
	}
}
