package main

import (
	"context"
	"github.com/chihqiang/sshlr/conf"
	"github.com/chihqiang/sshlr/pkg/tunnel"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

var (
	files = []string{"config.yaml", "~/.ssh/sshlr.yaml", "/etc/sshlr.yaml"}
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
	file, err := os.ReadFile(filename)
	var cfg conf.Config
	if err := yaml.Unmarshal(file, &cfg); err != nil {
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
	// 优雅退出
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	slog.Info("Shutting down tunnels")
	manager.StopAll()
}
