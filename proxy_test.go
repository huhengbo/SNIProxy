package main

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestProxyHTTPForward(t *testing.T) {
	backendLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer backendLn.Close()
	backendPort := backendLn.Addr().(*net.TCPAddr).Port

	var gotReq []byte
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c, err := backendLn.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
		buf := make([]byte, 4096)
		n, _ := c.Read(buf)
		gotReq = append([]byte(nil), buf[:n]...)
		_, _ = c.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok"))
	}()

	cfg := &configModel{
		Rules: []Rule{
			{Host: "example.com", Backend: "127.0.0.1", Port: backendPort},
		},
		HeaderTimeout: duration(2 * time.Second),
		IdleTimeout:   duration(2 * time.Second),
		DialTimeout:   duration(2 * time.Second),
		MaxConns:      100,
	}
	cfg.applyDefaults()

	dialer, err := buildDialer(false, "", cfg.DialTimeout.Duration())
	if err != nil {
		t.Fatal(err)
	}
	m := newMetrics()
	srv := newProxyServer(cfg, dialer, m, "")

	proxyLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer proxyLn.Close()

	go func() {
		c, err := proxyLn.Accept()
		if err != nil {
			return
		}
		m.connectionsActive.Add(1)
		defer m.connectionsActive.Add(-1)
		srv.serve(c)
	}()

	client, err := net.Dial("tcp", proxyLn.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	req := []byte("GET /hello HTTP/1.1\r\nHost: example.com\r\nConnection: close\r\n\r\n")
	if _, err := client.Write(req); err != nil {
		t.Fatal(err)
	}
	_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
	resp, _ := io.ReadAll(client)
	if !bytes.Contains(resp, []byte("200 OK")) {
		t.Fatalf("unexpected response: %q", resp)
	}

	wg.Wait()
	if !bytes.Contains(gotReq, []byte("GET /hello")) {
		t.Fatalf("backend did not receive request: %q", gotReq)
	}
	if m.bytesClientToBackend.Load() == 0 {
		t.Fatal("expected client->backend bytes metric")
	}
}

func TestProxyRejectUnmatchedHost(t *testing.T) {
	cfg := &configModel{
		Rules:         []Rule{{Host: "allowed.com"}},
		HeaderTimeout: duration(2 * time.Second),
		IdleTimeout:   duration(2 * time.Second),
		DialTimeout:   duration(2 * time.Second),
		MaxConns:      100,
	}
	dialer, err := buildDialer(false, "", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	m := newMetrics()
	srv := newProxyServer(cfg, dialer, m, "")

	proxyLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer proxyLn.Close()

	done := make(chan struct{})
	go func() {
		c, err := proxyLn.Accept()
		if err != nil {
			return
		}
		srv.serve(c)
		close(done)
	}()

	client, err := net.Dial("tcp", proxyLn.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	_, _ = client.Write([]byte("GET / HTTP/1.1\r\nHost: denied.com\r\n\r\n"))
	_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 64)
	_, _ = client.Read(buf)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not finish")
	}
	if m.deniedHosts.Load() != 1 {
		t.Fatalf("deniedHosts=%d", m.deniedHosts.Load())
	}
}

func TestProxyEmptyReadNoPanic(t *testing.T) {
	cfg := &configModel{
		AllowAllHosts: true,
		HeaderTimeout: duration(500 * time.Millisecond),
		IdleTimeout:   duration(time.Second),
		DialTimeout:   duration(time.Second),
		MaxConns:      10,
	}
	dialer, _ := buildDialer(false, "", time.Second)
	m := newMetrics()
	srv := newProxyServer(cfg, dialer, m, "")

	c1, c2 := net.Pipe()
	defer c1.Close()

	done := make(chan struct{})
	go func() {
		srv.serve(c2)
		close(done)
	}()
	_ = c1.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("serve hung or panicked")
	}
	if m.handshakeErrors.Load() == 0 {
		t.Fatal("expected handshake error metric")
	}
}

func TestMetricsEndpoint(t *testing.T) {
	m := newMetrics()
	m.connectionsTotal.Add(3)
	m.deniedHosts.Add(1)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	srv := startMetricsServer(addr, m)
	defer func() {
		_ = srv.Close()
	}()
	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get("http://" + addr + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("sniproxy_connections_total 3")) {
		t.Fatalf("metrics body: %s", body)
	}
	if !bytes.Contains(body, []byte("sniproxy_denied_hosts_total 1")) {
		t.Fatalf("metrics body: %s", body)
	}

	hz, err := http.Get("http://" + addr + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer hz.Body.Close()
	if hz.StatusCode != 200 {
		t.Fatalf("healthz=%d", hz.StatusCode)
	}
}
