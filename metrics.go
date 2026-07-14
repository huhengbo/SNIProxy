package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

// metrics 轻量 Prometheus 文本指标，无额外依赖。
type metrics struct {
	startUnix int64

	connectionsActive   atomic.Int64
	connectionsTotal    atomic.Int64
	connectionsRejected atomic.Int64
	handshakeErrors     atomic.Int64
	deniedHosts         atomic.Int64
	dialErrors          atomic.Int64
	writeErrors         atomic.Int64
	bytesClientToBackend atomic.Int64
	bytesBackendToClient atomic.Int64
	reloadsTotal        atomic.Int64
	reloadErrors        atomic.Int64
}

func newMetrics() *metrics {
	return &metrics{startUnix: time.Now().Unix()}
}

func (m *metrics) handler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	now := time.Now().Unix()
	uptime := now - m.startUnix
	if uptime < 0 {
		uptime = 0
	}

	_, _ = fmt.Fprintf(w, "# HELP sniproxy_up Always 1 when process is running.\n")
	_, _ = fmt.Fprintf(w, "# TYPE sniproxy_up gauge\n")
	_, _ = fmt.Fprintf(w, "sniproxy_up 1\n")

	_, _ = fmt.Fprintf(w, "# HELP sniproxy_uptime_seconds Process uptime in seconds.\n")
	_, _ = fmt.Fprintf(w, "# TYPE sniproxy_uptime_seconds gauge\n")
	_, _ = fmt.Fprintf(w, "sniproxy_uptime_seconds %d\n", uptime)

	writeGauge := func(name, help string, v int64) {
		_, _ = fmt.Fprintf(w, "# HELP %s %s\n", name, help)
		_, _ = fmt.Fprintf(w, "# TYPE %s gauge\n", name)
		_, _ = fmt.Fprintf(w, "%s %d\n", name, v)
	}
	writeCounter := func(name, help string, v int64) {
		_, _ = fmt.Fprintf(w, "# HELP %s %s\n", name, help)
		_, _ = fmt.Fprintf(w, "# TYPE %s counter\n", name)
		_, _ = fmt.Fprintf(w, "%s %d\n", name, v)
	}

	writeGauge("sniproxy_connections_active", "Currently active proxy connections.", m.connectionsActive.Load())
	writeCounter("sniproxy_connections_total", "Total accepted connections.", m.connectionsTotal.Load())
	writeCounter("sniproxy_connections_rejected_total", "Connections rejected (limit or policy).", m.connectionsRejected.Load())
	writeCounter("sniproxy_handshake_errors_total", "Client header/handshake parse errors.", m.handshakeErrors.Load())
	writeCounter("sniproxy_denied_hosts_total", "Connections denied by host rules.", m.deniedHosts.Load())
	writeCounter("sniproxy_dial_errors_total", "Backend dial failures.", m.dialErrors.Load())
	writeCounter("sniproxy_write_errors_total", "Failures writing first packet to backend.", m.writeErrors.Load())
	writeCounter("sniproxy_bytes_client_to_backend_total", "Bytes forwarded client -> backend.", m.bytesClientToBackend.Load())
	writeCounter("sniproxy_bytes_backend_to_client_total", "Bytes forwarded backend -> client.", m.bytesBackendToClient.Load())
	writeCounter("sniproxy_reloads_total", "Successful config reloads.", m.reloadsTotal.Load())
	writeCounter("sniproxy_reload_errors_total", "Failed config reloads.", m.reloadErrors.Load())
}

func (m *metrics) healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func startMetricsServer(addr string, m *metrics) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", m.handler)
	mux.HandleFunc("/healthz", m.healthHandler)
	mux.HandleFunc("/readyz", m.healthHandler)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		serviceLogger(fmt.Sprintf("指标服务监听: %s (/metrics /healthz)", addr), 0, false)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serviceLogger(fmt.Sprintf("指标服务异常: %v", err), 31, false)
		}
	}()
	return srv
}
