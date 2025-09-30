package tunnel

import (
	"context"
	"fmt"
	"github.com/chihqiang/sshlr/pkg/sshx"
	"log/slog"
)

type ITunnel interface {
	Start(ctx context.Context) error
	Stop() error
}

// TunnelManager 管理所有本地和远程隧道
type TunnelManager struct {
	sshClients map[string]sshx.Config
	tunnels    []ITunnel
}

// NewTunnelManager 根据配置创建 TunnelManager
func NewTunnelManager(ssh []sshx.Config, local []LocalConfig, remote []RemoteConfig) (*TunnelManager, error) {
	manager := &TunnelManager{sshClients: make(map[string]sshx.Config)}
	if len(ssh) == 0 {
		return nil, fmt.Errorf("no SSH client configuration provided")
	}
	// 构建 SSH 客户端映射
	for _, s := range ssh {
		manager.sshClients[s.Name] = s
	}
	// 构建 LocalTunnel
	for _, l := range local {
		sshCfg, ok := manager.sshClients[l.SSH]
		if !ok {
			return nil, fmt.Errorf("local tunnel SSH client not found: %s", l.SSH)
		}
		manager.tunnels = append(manager.tunnels, NewLocalTunnel(sshCfg, l))
	}
	// 构建 RemoteTunnel
	for _, r := range remote {
		sshCfg, ok := manager.sshClients[r.SSH]
		if !ok {
			return nil, fmt.Errorf("remote tunnel SSH client not found: %s", r.SSH)
		}
		manager.tunnels = append(manager.tunnels, NewRemoteTunnel(sshCfg, r))
	}
	if len(manager.tunnels) == 0 {
		return nil, fmt.Errorf("no valid tunnels could be created")
	}
	return manager, nil
}

// StartAll 启动所有隧道
func (m *TunnelManager) StartAll(ctx context.Context) {
	for _, t := range m.tunnels {
		go func(t ITunnel) {
			if err := t.Start(ctx); err != nil {
				slog.Error("Tunnel start failed", "err", err)
			}
		}(t)
	}
}

// StopAll 停止所有隧道
func (m *TunnelManager) StopAll() {
	for _, t := range m.tunnels {
		if err := t.Stop(); err != nil {
			slog.Warn("Tunnel stop failed", "err", err)
		}
	}
}
