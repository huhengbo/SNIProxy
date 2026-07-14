package main

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/proxy"
)

// buildDialer 在启动时构建 Dialer；SOCKS5 失败则返回错误，绝不静默降级直连。
func buildDialer(enableSocks bool, socksAddr string, dialTimeout time.Duration) (proxy.Dialer, error) {
	base := &net.Dialer{Timeout: dialTimeout}
	if !enableSocks {
		return base, nil
	}
	d, err := proxy.SOCKS5("tcp", socksAddr, nil, base)
	if err != nil {
		return nil, fmt.Errorf("创建 SOCKS5 拨号器失败 (%s): %w", socksAddr, err)
	}
	return d, nil
}
