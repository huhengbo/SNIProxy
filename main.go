package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	version string // 编译时写入版本号

	ConfigFilePath string
	LogFilePath    string
	EnableDebug    bool
)

func parseFlags() {
	var printVersion bool
	help := `
SNIProxy ` + version + `
https://github.com/huhengbo/SNIProxy

参数：
    -c config.yaml
        配置文件 (默认 config.yaml)
    -l sni.log
        日志文件 (默认 无)
    -d
        调试模式 (默认 关)
    -v
        程序版本
    -h
        帮助说明

信号：
    SIGHUP
        热加载配置（listen 变更需重启）
    SIGINT / SIGTERM
        优雅退出
`
	flag.StringVar(&ConfigFilePath, "c", "config.yaml", "配置文件")
	flag.StringVar(&LogFilePath, "l", "", "日志文件")
	flag.BoolVar(&EnableDebug, "d", false, "调试模式")
	flag.BoolVar(&printVersion, "v", false, "程序版本")
	flag.Usage = func() { fmt.Print(help) }
	flag.Parse()
	if printVersion {
		fmt.Printf("SNIProxy %s\n", version)
		os.Exit(0)
	}
}

func main() {
	parseFlags()

	cfg, err := loadConfig(ConfigFilePath)
	if err != nil {
		serviceLogger(err.Error(), 31, false)
		os.Exit(1)
	}

	if err := initLogger(LogFilePath, EnableDebug); err != nil {
		serviceLogger(err.Error(), 31, false)
		os.Exit(1)
	}
	defer closeLogger()

	serviceLogger(fmt.Sprintf("调试模式: %v", EnableDebug), 32, false)
	cfg.logSummary()

	dialer, err := buildDialer(cfg.Socks5.Enable, cfg.Socks5.Addr, cfg.DialTimeout.Duration())
	if err != nil {
		serviceLogger(err.Error(), 31, false)
		os.Exit(1)
	}

	m := newMetrics()
	var metricsSrv *http.Server
	if cfg.MetricsAddr != "" {
		metricsSrv = startMetricsServer(cfg.MetricsAddr, m)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := newProxyServer(cfg, dialer, m, ConfigFilePath)

	// SIGHUP 热加载
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-hup:
				if err := srv.reload(); err != nil {
					serviceLogger(fmt.Sprintf("配置热加载失败: %v", err), 31, false)
				}
			}
		}
	}()

	var wg sync.WaitGroup
	for _, addr := range cfg.Listen {
		addr := addr
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := srv.listenAndServe(ctx, addr); err != nil && ctx.Err() == nil {
				serviceLogger(err.Error(), 31, false)
				stop()
			}
		}()
	}

	<-ctx.Done()
	fmt.Printf("\n接收到退出信号, 正在关闭...\n")

	if metricsSrv != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = metricsSrv.Shutdown(shutdownCtx)
		cancel()
	}

	wg.Wait()
	serviceLogger("已退出", 0, false)
}
