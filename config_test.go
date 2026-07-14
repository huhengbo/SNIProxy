package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHostMatches(t *testing.T) {
	tests := []struct {
		name, host, pattern string
		want                bool
	}{
		{"exact", "example.com", "example.com", true},
		{"subdomain", "a.example.com", "example.com", true},
		{"deep subdomain", "a.b.example.com", "example.com", true},
		{"no false substring", "evil-example.com", "example.com", false},
		{"no prefix glue", "notexample.com", "example.com", false},
		{"wildcard base", "example.com", "*.example.com", true},
		{"wildcard sub", "a.example.com", "*.example.com", true},
		{"wildcard deep", "a.b.example.com", "*.example.com", true},
		{"wildcard reject", "evil-example.com", "*.example.com", false},
		{"case insensitive", "Example.COM", "example.com", true},
		{"trailing dot", "example.com.", "example.com", true},
		{"empty", "", "example.com", false},
		{"other domain", "google.com", "example.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostMatches(tt.host, tt.pattern); got != tt.want {
				t.Fatalf("hostMatches(%q, %q)=%v want %v", tt.host, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Example.Com", "example.com"},
		{"example.com.", "example.com"},
		{"example.com:8080", "example.com"},
		{"[::1]:443", "::1"},
	}
	for _, tt := range tests {
		if got := normalizeHost(tt.in); got != tt.want {
			t.Fatalf("normalizeHost(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestResolveTarget(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		listenPort int
		rule       *Rule
		want       string
	}{
		{
			name:       "default listen port",
			serverName: "example.com",
			listenPort: 443,
			rule:       nil,
			want:       "example.com:443",
		},
		{
			name:       "rule backend host only",
			serverName: "cdn.example.com",
			listenPort: 443,
			rule:       &Rule{Host: "*.example.com", Backend: "10.0.0.5"},
			want:       "10.0.0.5:443",
		},
		{
			name:       "rule backend with port",
			serverName: "api.example.com",
			listenPort: 443,
			rule:       &Rule{Host: "api.example.com", Backend: "127.0.0.1:8443"},
			want:       "127.0.0.1:8443",
		},
		{
			name:       "rule port override",
			serverName: "api.example.com",
			listenPort: 443,
			rule:       &Rule{Host: "api.example.com", Backend: "127.0.0.1", Port: 8443},
			want:       "127.0.0.1:8443",
		},
		{
			name:       "sni with port stripped",
			serverName: "example.com:443",
			listenPort: 80,
			rule:       nil,
			want:       "example.com:80",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveTarget(tt.serverName, tt.listenPort, tt.rule)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestFindRuleFirstMatchOnly(t *testing.T) {
	cfg := &configModel{
		Rules: []Rule{
			{Host: "example.com", Backend: "1.1.1.1"},
			{Host: "a.example.com", Backend: "2.2.2.2"},
		},
	}
	rule, ok := cfg.findRule("a.example.com")
	if !ok {
		t.Fatal("expected match")
	}
	if rule.Backend != "1.1.1.1" {
		t.Fatalf("expected first rule, got backend %q", rule.Backend)
	}
}

func TestLoadConfigObjectRules(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
listen:
  - ":443"
rules:
  - host: example.com
  - host: "*.cdn.example.com"
    backend: 10.0.0.5:8443
dial_timeout: 3s
idle_timeout: 30
metrics_addr: "127.0.0.1:9100"
socks5:
  enable: false
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Rules) != 2 {
		t.Fatalf("rules=%d", len(cfg.Rules))
	}
	if cfg.Rules[0].Host != "example.com" {
		t.Fatalf("rule host=%q", cfg.Rules[0].Host)
	}
	if cfg.Rules[1].Backend != "10.0.0.5:8443" {
		t.Fatalf("backend=%q", cfg.Rules[1].Backend)
	}
	if cfg.DialTimeout.Duration() != 3*time.Second {
		t.Fatalf("dial_timeout=%v", cfg.DialTimeout.Duration())
	}
	if cfg.IdleTimeout.Duration() != 30*time.Second {
		t.Fatalf("idle_timeout=%v", cfg.IdleTimeout.Duration())
	}
	if cfg.MetricsAddr != "127.0.0.1:9100" {
		t.Fatalf("metrics_addr=%q", cfg.MetricsAddr)
	}
}

func TestLoadConfigRejectStringRules(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	// 旧版字符串 rules 不再支持
	content := `
listen: [":443"]
rules:
  - example.com
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(path); err == nil {
		t.Fatal("expected error for string rules")
	}
}

func TestLoadConfigRejectEmptyRules(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("listen: [\":443\"]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(path); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadConfigSocksRequiresAddr(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
allow_all_hosts: true
socks5:
  enable: true
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(path); err == nil {
		t.Fatal("expected socks5.addr error")
	}
}

func TestHostFromHTTPHost(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"example.com", "example.com"},
		{"example.com:8080", "example.com"},
		{"[::1]:80", "::1"},
	}
	for _, tt := range tests {
		if got := hostFromHTTPHost(tt.in); got != tt.want {
			t.Fatalf("hostFromHTTPHost(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestSameListen(t *testing.T) {
	if !sameListen([]string{":80", ":443"}, []string{":443", ":80"}) {
		t.Fatal("expected equal ignoring order")
	}
	if sameListen([]string{":80"}, []string{":443"}) {
		t.Fatal("expected not equal")
	}
}

func TestReloadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	v1 := `
listen: [":1443"]
rules:
  - host: a.com
`
	v2 := `
listen: [":1443"]
rules:
  - host: b.com
    backend: 127.0.0.1:9
`
	if err := os.WriteFile(path, []byte(v1), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	dialer, err := buildDialer(false, "", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	m := newMetrics()
	srv := newProxyServer(cfg, dialer, m, path)

	if err := os.WriteFile(path, []byte(v2), 0644); err != nil {
		t.Fatal(err)
	}
	if err := srv.reload(); err != nil {
		t.Fatal(err)
	}
	cfg2, _ := srv.snapshot()
	if cfg2.Rules[0].Host != "b.com" {
		t.Fatalf("reload host=%q", cfg2.Rules[0].Host)
	}
	if m.reloadsTotal.Load() != 1 {
		t.Fatalf("reloads=%d", m.reloadsTotal.Load())
	}
}
