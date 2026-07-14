package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Rule 单条转发规则（仅对象格式）。
type Rule struct {
	// Host 匹配模式：example.com 或 *.example.com
	Host string `yaml:"host"`
	// Backend 目标主机；可写 IP/域名，或 host:port。空则使用客户端 SNI/Host。
	Backend string `yaml:"backend,omitempty"`
	// Port 覆盖目标端口；0 表示使用本地监听端口。若 Backend 已含端口且 Port 为 0 则用 Backend 端口。
	Port int `yaml:"port,omitempty"`
}

// Socks5Config 前置 SOCKS5 代理。
type Socks5Config struct {
	Enable bool   `yaml:"enable"`
	Addr   string `yaml:"addr"`
}

// duration 支持 YAML 中的 "10s"/"5m" 或整数秒。
type duration time.Duration

func (d *duration) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		if value.Tag == "!!int" || value.Tag == "!!float" {
			n, err := strconv.Atoi(value.Value)
			if err != nil {
				return fmt.Errorf("无效的时长 %q: %w", value.Value, err)
			}
			*d = duration(time.Duration(n) * time.Second)
			return nil
		}
		s := value.Value
		if s == "" {
			return nil
		}
		if n, err := strconv.Atoi(s); err == nil {
			*d = duration(time.Duration(n) * time.Second)
			return nil
		}
		parsed, err := time.ParseDuration(s)
		if err != nil {
			return fmt.Errorf("无效的时长 %q: %w", s, err)
		}
		*d = duration(parsed)
		return nil
	default:
		return fmt.Errorf("无效的时长节点类型 %v", value.Kind)
	}
}

func (d duration) Duration() time.Duration { return time.Duration(d) }

type configModel struct {
	Listen        []string     `yaml:"listen"`
	Rules         []Rule       `yaml:"rules"`
	Socks5        Socks5Config `yaml:"socks5"`
	AllowAllHosts bool         `yaml:"allow_all_hosts"`
	MetricsAddr   string       `yaml:"metrics_addr"`
	DialTimeout   duration     `yaml:"dial_timeout"`
	IdleTimeout   duration     `yaml:"idle_timeout"`
	HeaderTimeout duration     `yaml:"header_timeout"`
	MaxConns      int          `yaml:"max_conns"`
}

func loadConfig(path string) (*configModel, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("配置文件读取失败: %w", err)
	}
	var cfg configModel
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("配置文件解析失败: %w", err)
	}
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *configModel) applyDefaults() {
	if c.DialTimeout <= 0 {
		c.DialTimeout = duration(10 * time.Second)
	}
	if c.IdleTimeout <= 0 {
		c.IdleTimeout = duration(5 * time.Minute)
	}
	if c.HeaderTimeout <= 0 {
		c.HeaderTimeout = duration(5 * time.Second)
	}
	if c.MaxConns <= 0 {
		c.MaxConns = 10000
	}
	if len(c.Listen) == 0 {
		c.Listen = []string{":443"}
	}
}

func (c *configModel) validate() error {
	if len(c.Rules) == 0 && !c.AllowAllHosts {
		return fmt.Errorf("rules 不能为空（除非 allow_all_hosts 为 true）")
	}
	for i, rule := range c.Rules {
		if strings.TrimSpace(rule.Host) == "" {
			return fmt.Errorf("rules[%d].host 不能为空", i)
		}
		if strings.HasPrefix(rule.Host, "*.") && len(rule.Host) < 4 {
			return fmt.Errorf("rules[%d].host 通配符非法: %q", i, rule.Host)
		}
		if rule.Port < 0 || rule.Port > 65535 {
			return fmt.Errorf("rules[%d].port 非法: %d", i, rule.Port)
		}
		if rule.Backend != "" {
			if _, _, err := net.SplitHostPort(rule.Backend); err != nil {
				// 允许纯 host / IP（无端口）
				if strings.Contains(rule.Backend, "://") {
					return fmt.Errorf("rules[%d].backend 不应包含协议 scheme: %q", i, rule.Backend)
				}
			}
		}
	}
	if c.Socks5.Enable {
		if strings.TrimSpace(c.Socks5.Addr) == "" {
			return fmt.Errorf("socks5.enable 为 true 时必须配置 socks5.addr")
		}
		if _, _, err := net.SplitHostPort(c.Socks5.Addr); err != nil {
			return fmt.Errorf("socks5.addr 格式非法（需要 host:port）: %w", err)
		}
	}
	if c.MetricsAddr != "" {
		if _, _, err := net.SplitHostPort(c.MetricsAddr); err != nil {
			return fmt.Errorf("metrics_addr 格式非法（需要 host:port）: %w", err)
		}
	}
	for i, addr := range c.Listen {
		if strings.TrimSpace(addr) == "" {
			return fmt.Errorf("listen[%d] 不能为空", i)
		}
	}
	return nil
}

func (c *configModel) logSummary() {
	for _, rule := range c.Rules {
		extra := ""
		if rule.Backend != "" || rule.Port != 0 {
			extra = fmt.Sprintf(" -> backend=%q port=%d", rule.Backend, rule.Port)
		}
		serviceLogger(fmt.Sprintf("加载规则: %s%s", rule.Host, extra), 32, false)
	}
	serviceLogger(fmt.Sprintf("前置代理: %v", c.Socks5.Enable), 32, false)
	serviceLogger(fmt.Sprintf("任意域名: %v", c.AllowAllHosts), 32, false)
	serviceLogger(fmt.Sprintf("连接上限: %d, 拨号超时: %s, 空闲超时: %s, 首包超时: %s",
		c.MaxConns, c.DialTimeout.Duration(), c.IdleTimeout.Duration(), c.HeaderTimeout.Duration()), 32, true)
	if c.MetricsAddr != "" {
		serviceLogger(fmt.Sprintf("指标地址: %s", c.MetricsAddr), 32, false)
	}
}

// sameListen 比较监听地址集合是否一致（忽略顺序）。
func sameListen(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]int, len(a))
	for _, x := range a {
		set[x]++
	}
	for _, x := range b {
		set[x]--
		if set[x] < 0 {
			return false
		}
	}
	return true
}

// normalizeHost 去掉端口、尾点、转小写。
func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.ToLower(host)
	host = strings.TrimSuffix(host, ".")
	if h, _, err := net.SplitHostPort(host); err == nil {
		return strings.ToLower(strings.TrimSuffix(h, "."))
	}
	return host
}

// hostMatches 域名匹配：精确、后缀边界，或 *.example.com 通配。
func hostMatches(serverName, pattern string) bool {
	host := normalizeHost(serverName)
	pat := normalizeHost(pattern)
	if host == "" || pat == "" {
		return false
	}
	if strings.HasPrefix(pat, "*.") {
		base := pat[2:]
		if base == "" {
			return false
		}
		return host == base || strings.HasSuffix(host, "."+base)
	}
	return host == pat || strings.HasSuffix(host, "."+pat)
}

// findRule 返回第一条匹配规则；无匹配返回 nil, false。
func (c *configModel) findRule(serverName string) (*Rule, bool) {
	for i := range c.Rules {
		if hostMatches(serverName, c.Rules[i].Host) {
			return &c.Rules[i], true
		}
	}
	return nil, false
}

// resolveTarget 根据 SNI/Host、监听端口与规则计算后端地址 host:port。
func resolveTarget(serverName string, listenPort int, rule *Rule) (string, error) {
	hostOnly := normalizeHost(serverName)
	if hostOnly == "" {
		return "", fmt.Errorf("空主机名")
	}

	if rule == nil {
		return net.JoinHostPort(hostOnly, strconv.Itoa(listenPort)), nil
	}

	backend := strings.TrimSpace(rule.Backend)
	if backend == "" {
		backend = hostOnly
	}

	if h, p, err := net.SplitHostPort(backend); err == nil {
		if rule.Port != 0 {
			return net.JoinHostPort(h, strconv.Itoa(rule.Port)), nil
		}
		return net.JoinHostPort(h, p), nil
	}

	port := listenPort
	if rule.Port != 0 {
		port = rule.Port
	}
	return net.JoinHostPort(backend, strconv.Itoa(port)), nil
}
