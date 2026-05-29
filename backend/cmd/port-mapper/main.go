// Package main は Porto アプリケーションの起動用エントリポイントを提供します。
package main

import (
	"flag"
	"io"
	"log"
	"log/slog"
	"os"

	"github.com/clagon/port-mapper/backend/internal/app"
	"github.com/clagon/port-mapper/backend/internal/browseropener"
)

// cliOptions はコマンドライン引数から解析された起動オプションを表します。
type cliOptions struct {
	ListenAddr  string
	ConfigPath  string
	OpenBrowser bool
}

// parseArgs はコマンドライン引数を解析し起動オプションにマッピングします。
func parseArgs(args []string) (cliOptions, error) {
	fs := flag.NewFlagSet("port-mapper", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	opts := cliOptions{OpenBrowser: true}
	fs.StringVar(&opts.ListenAddr, "listen-addr", "", "listen address for the local API")
	fs.StringVar(&opts.ConfigPath, "config", "", "path to config.json")
	noBrowser := fs.Bool("no-browser", false, "do not open the browser automatically")

	if err := fs.Parse(args); err != nil {
		return cliOptions{}, err
	}
	if *noBrowser {
		opts.OpenBrowser = false
	}
	return opts, nil
}

// main はアプリケーションのエントリポイントであり、フラグ解析、DI（依存性の注入）、および初期起動処理を実行します。
func main() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	a, err := app.New(app.AppOptions{
		ListenAddr:    opts.ListenAddr,
		ConfigPath:    opts.ConfigPath,
		OpenBrowser:   opts.OpenBrowser,
		BrowserOpener: browseropener.New(),
		Logger:        logger,
	})
	if err != nil {
		log.Fatal(err)
	}

	logger.Info("port-mapper ready",
		"config_path", a.ConfigPath(),
		"listen_addr", a.Addr(),
		"browser_open", opts.OpenBrowser,
	)

	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
