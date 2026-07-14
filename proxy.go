package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

type proxyServer struct {
	mu         sync.RWMutex
	cfg        *configModel
	dialer     proxy.Dialer
	metrics    *metrics
	configPath string
}

func newProxyServer(cfg *configModel, dialer proxy.Dialer, metrics *metrics, configPath string) *proxyServer {
	return &proxyServer{
		cfg:        cfg,
		dialer:     dialer,
		metrics:    metrics,
		configPath: configPath,
	}
}

func (p *proxyServer) snapshot() (*configModel, proxy.Dialer) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cfg, p.dialer
}

// reload 热加载配置。listen 变更不会生效，需重启进程。
func (p *proxyServer) reload() error {
	newCfg, err := loadConfig(p.configPath)
	if err != nil {
		p.metrics.reloadErrors.Add(1)
		return err
	}
	newDialer, err := buildDialer(newCfg.Socks5.Enable, newCfg.Socks5.Addr, newCfg.DialTimeout.Duration())
	if err != nil {
		p.metrics.reloadErrors.Add(1)
		return err
	}

	p.mu.Lock()
	oldListen := append([]string(nil), p.cfg.Listen...)
	p.cfg = newCfg
	p.dialer = newDialer
	p.mu.Unlock()

	p.metrics.reloadsTotal.Add(1)
	serviceLogger("配置热加载成功", 32, false)
	newCfg.logSummary()
	if !sameListen(oldListen, newCfg.Listen) {
		serviceLogger("警告: listen 变更不会热更新，需重启进程后生效", 33, false)
	}
	return nil
}

func (p *proxyServer) listenAndServe(ctx context.Context, listenAddr string) error {
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("监听失败 %s: %w", listenAddr, err)
	}
	serviceLogger(fmt.Sprintf("开始监听: %v", ln.Addr()), 0, false)

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	var wg sync.WaitGroup
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			serviceLogger(fmt.Sprintf("接受连接请求时出错: %v", err), 31, false)
			continue
		}

		cfg, _ := p.snapshot()
		if p.metrics.connectionsActive.Load() >= int64(cfg.MaxConns) {
			p.metrics.connectionsRejected.Add(1)
			serviceLogger("连接数已达上限，拒绝新连接", 31, false)
			_ = conn.Close()
			continue
		}

		p.metrics.connectionsTotal.Add(1)
		p.metrics.connectionsActive.Add(1)
		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			defer p.metrics.connectionsActive.Add(-1)
			p.serve(c)
		}(conn)
	}
	wg.Wait()
	return nil
}

func (p *proxyServer) serve(c net.Conn) {
	defer c.Close()

	cfg, dialer := p.snapshot()

	raddr := c.RemoteAddr().String()
	if ta, ok := c.RemoteAddr().(*net.TCPAddr); ok {
		raddr = ta.String()
	}
	serviceLogger("连接来自: "+raddr, 32, false)

	localPort := 0
	if la, ok := c.LocalAddr().(*net.TCPAddr); ok {
		localPort = la.Port
	}

	header, serverName, err := p.readClientHeader(c, cfg)
	if err != nil {
		p.metrics.handshakeErrors.Add(1)
		serviceLogger(fmt.Sprintf("读取客户端首包失败: %v", err), 31, true)
		return
	}
	if serverName == "" {
		p.metrics.handshakeErrors.Add(1)
		serviceLogger("未找到目标域名, 忽略...", 31, true)
		return
	}

	var (
		dst  string
		rule *Rule
	)
	if cfg.AllowAllHosts {
		dst, err = resolveTarget(serverName, localPort, nil)
	} else {
		var ok bool
		rule, ok = cfg.findRule(serverName)
		if !ok {
			p.metrics.deniedHosts.Add(1)
			serviceLogger(fmt.Sprintf("域名未匹配规则, 拒绝: %s", serverName), 31, true)
			return
		}
		dst, err = resolveTarget(serverName, localPort, rule)
	}
	if err != nil {
		p.metrics.handshakeErrors.Add(1)
		serviceLogger(fmt.Sprintf("解析目标地址失败: %v", err), 31, false)
		return
	}

	serviceLogger(fmt.Sprintf("转发目标: %s", dst), 32, false)
	p.forward(c, header, dst, raddr, cfg, dialer)
}

func (p *proxyServer) readClientHeader(c net.Conn, cfg *configModel) (header []byte, serverName string, err error) {
	_ = c.SetReadDeadline(time.Now().Add(cfg.HeaderTimeout.Duration()))
	defer func() { _ = c.SetReadDeadline(time.Time{}) }()

	buf := make([]byte, 0, 2048)
	tmp := make([]byte, 2048)

	for {
		n, readErr := c.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if len(buf) > maxClientHeaderSize {
			return nil, "", fmt.Errorf("首包过大")
		}
		if len(buf) == 0 {
			if readErr != nil {
				return nil, "", readErr
			}
			continue
		}

		if buf[0] != 0x16 {
			if bytes.Contains(buf, []byte("\r\n\r\n")) || errors.Is(readErr, io.EOF) {
				host, hostErr := extractHTTPHost(buf)
				if hostErr != nil {
					return nil, "", fmt.Errorf("解析 HTTP Host: %w", hostErr)
				}
				return buf, host, nil
			}
			if readErr != nil {
				return nil, "", readErr
			}
			continue
		}

		if isCompleteTLSRecord(buf) {
			name, sniErr := getSNIServerName(buf)
			if sniErr != nil {
				return nil, "", sniErr
			}
			return buf, name, nil
		}
		if readErr != nil {
			return nil, "", fmt.Errorf("TLS 握手不完整: %w", readErr)
		}
	}
}

func (p *proxyServer) forward(conn net.Conn, data []byte, dst, raddr string, cfg *configModel, dialer proxy.Dialer) {
	backend, err := dialer.Dial("tcp", dst)
	if err != nil {
		p.metrics.dialErrors.Add(1)
		serviceLogger(fmt.Sprintf("无法连接到后端 %s: %v", dst, err), 31, false)
		return
	}
	defer backend.Close()

	if _, err = backend.Write(data); err != nil {
		p.metrics.writeErrors.Add(1)
		serviceLogger(fmt.Sprintf("无法传输到后端: %v", err), 31, false)
		return
	}
	// 首包已计入 client->backend
	p.metrics.bytesClientToBackend.Add(int64(len(data)))

	_ = conn.SetDeadline(time.Time{})
	_ = backend.SetDeadline(time.Time{})

	var once sync.Once
	done := make(chan struct{})
	finish := func() {
		once.Do(func() {
			_ = conn.Close()
			_ = backend.Close()
			close(done)
		})
	}

	go p.pipe(backend, conn, false, raddr, dst, cfg.IdleTimeout.Duration(), finish)
	go p.pipe(conn, backend, true, raddr, dst, cfg.IdleTimeout.Duration(), finish)
	<-done
}

func (p *proxyServer) pipe(dst io.Writer, src net.Conn, toClient bool, raddr, dsts string, idle time.Duration, finish func()) {
	defer finish()

	buf := make([]byte, 32*1024)
	var written int64
	for {
		if idle > 0 {
			_ = src.SetReadDeadline(time.Now().Add(idle))
		}
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				break
			}
			if nr != nw {
				break
			}
		}
		if er != nil {
			break
		}
	}

	if toClient {
		p.metrics.bytesBackendToClient.Add(written)
		serviceLogger(fmt.Sprintf("[%v] -> [%v] %d bytes", dsts, raddr, written), 33, true)
	} else {
		p.metrics.bytesClientToBackend.Add(written)
		serviceLogger(fmt.Sprintf("[%v] -> [%v] %d bytes", raddr, dsts, written), 33, true)
	}
}
