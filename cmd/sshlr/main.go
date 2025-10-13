package main

import (
	"context"
	"encoding/base64"
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
	//cat config.yaml| base64
	configBase64Content string
)

func init() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
}

func loadConfig() (*conf.Config, error) {
	var data []byte
	filename, err := conf.GetConfigPath(files...)
	if err == nil {
		// 优先读取外部文件
		data, err = os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
	} else {
		// 没有外部文件，使用嵌入 Base64 配置
		data, err = base64.StdEncoding.DecodeString(configBase64Content)
		if err != nil {
			return nil, err
		}
	}
	var cfg conf.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("load Config err", "err", err)
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
