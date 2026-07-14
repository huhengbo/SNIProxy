package main

import (
	"crypto/tls"
	"net"
	"testing"
	"time"
)

// captureClientHello 通过真实 TLS 握手捕获 ClientHello 字节。
func captureClientHello(t *testing.T, serverName string) []byte {
	t.Helper()
	client, server := net.Pipe()
	defer client.Close()

	helloCh := make(chan []byte, 1)
	go func() {
		defer server.Close()
		buf := make([]byte, 16*1024)
		_ = server.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, err := server.Read(buf)
		if err != nil || n == 0 {
			helloCh <- nil
			return
		}
		helloCh <- append([]byte(nil), buf[:n]...)
	}()

	go func() {
		cfg := &tls.Config{
			ServerName:         serverName,
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
		}
		_ = tls.Client(client, cfg).Handshake()
	}()

	hello := <-helloCh
	if len(hello) == 0 {
		t.Fatal("failed to capture ClientHello")
	}
	return hello
}

func TestGetSNIServerName(t *testing.T) {
	hello := captureClientHello(t, "www.example.com")
	name, err := getSNIServerName(hello)
	if err != nil {
		t.Fatal(err)
	}
	if name != "www.example.com" {
		t.Fatalf("sni=%q", name)
	}
}

func TestIsCompleteTLSRecord(t *testing.T) {
	hello := captureClientHello(t, "a.example.com")
	if !isCompleteTLSRecord(hello) {
		t.Fatal("expected complete record")
	}
	if isCompleteTLSRecord(hello[:3]) {
		t.Fatal("short buffer should be incomplete")
	}
	if isCompleteTLSRecord(hello[:len(hello)-1]) {
		t.Fatal("truncated record should be incomplete")
	}
}

func TestExtractHTTPHost(t *testing.T) {
	raw := []byte("GET / HTTP/1.1\r\nHost: example.com:8080\r\nUser-Agent: test\r\n\r\n")
	host, err := extractHTTPHost(raw)
	if err != nil {
		t.Fatal(err)
	}
	if host != "example.com" {
		t.Fatalf("host=%q", host)
	}
}

func TestGetSNIServerNameRejectNonTLS(t *testing.T) {
	if _, err := getSNIServerName([]byte("GET / HTTP/1.1\r\n")); err == nil {
		t.Fatal("expected error for non-TLS")
	}
}
