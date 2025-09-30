package main

import (
	"context"
	"github.com/chihqiang/sshlr/conf"
	"github.com/chihqiang/sshlr/pkg/tunnel"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	files = []string{"config.yaml", "~/.ssh/sshlr.yaml", "/etc/sshlr.yaml"}
	mu    sync.Mutex
)

func init() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
}

func main() {
	// 获取配置文件路径
	filename, err := conf.GetConfigPath(files...)
	if err != nil {
		slog.Error("No configuration file found", "err", err)
		return
	}
	viper.SetConfigFile(filename)
	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		slog.Error("Failed to read config file", "err", err)
		return
	}
	var cfg conf.Config
	// 解析 YAML 配置到结构体
	if err := viper.Unmarshal(&cfg); err != nil {
		slog.Error("Failed to parse YAML config", "err", err)
		return
	}

	// 创建 TunnelManager
	manager, err := tunnel.NewTunnelManager(cfg.Ssh, cfg.Local, cfg.Remote)
	if err != nil {
		slog.Error("Failed to create TunnelManager", "err", err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// 启动所有隧道
	manager.StartAll(ctx)
	slog.Info("All tunnels started")
	// 监听配置变化（防抖处理）
	viper.WatchConfig()
	var (
		debounceTimer *time.Timer
		debounceDelay = 500 * time.Millisecond
	)
	viper.OnConfigChange(func(e fsnotify.Event) {
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(debounceDelay, func() {
			mu.Lock()
			defer mu.Unlock()
			slog.Info("Config file changed, reloading", "file", e.Name)

			// 停止并重启隧道
			manager.StopAll()
			manager.StartAll(ctx)
		})
	})

	// 优雅退出
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	slog.Info("Shutting down tunnels")
	manager.StopAll()
}
