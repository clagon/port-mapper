package main

import "testing"

func TestParseArgsDefaults(t *testing.T) {
	opts, err := parseArgs(nil)
	if err != nil {
		t.Fatalf("parseArgs() error = %v", err)
	}
	if opts.ListenAddr != "" {
		t.Fatalf("ListenAddr = %q, want empty", opts.ListenAddr)
	}
	if opts.ConfigPath != "" {
		t.Fatalf("ConfigPath = %q, want empty", opts.ConfigPath)
	}
	if !opts.OpenBrowser {
		t.Fatal("OpenBrowser = false, want true")
	}
}

func TestParseArgsOverrides(t *testing.T) {
	opts, err := parseArgs([]string{"--listen-addr", "127.0.0.1:9090", "--config", "/tmp/port-mapper.json", "--no-browser"})
	if err != nil {
		t.Fatalf("parseArgs() error = %v", err)
	}
	if opts.ListenAddr != "127.0.0.1:9090" {
		t.Fatalf("ListenAddr = %q, want %q", opts.ListenAddr, "127.0.0.1:9090")
	}
	if opts.ConfigPath != "/tmp/port-mapper.json" {
		t.Fatalf("ConfigPath = %q, want %q", opts.ConfigPath, "/tmp/port-mapper.json")
	}
	if opts.OpenBrowser {
		t.Fatal("OpenBrowser = true, want false")
	}
}

func TestParseArgsRejectsUnknownFlag(t *testing.T) {
	if _, err := parseArgs([]string{"--does-not-exist"}); err == nil {
		t.Fatal("parseArgs() error = nil, want error")
	}
}
