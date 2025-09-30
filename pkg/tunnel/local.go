package tunnel

import (
	"context"
	"github.com/chihqiang/sshlr/pkg/sshx"
	"golang.org/x/crypto/ssh"
	"io"
	"log/slog"
	"net"
	"time"
)

type LocalConfig struct {
	SSH           string `yaml:"ssh" mapstructure:"ssh" json:"ssh"`                                  // 对应 Ssh.Name 的值，用于指定使用的 SSH 连接
	Local         string `yaml:"local" mapstructure:"local" json:"local"`                            // 本地监听地址与端口（例如 "127.0.0.1:8080"）
	Remote        string `yaml:"remote" mapstructure:"remote" json:"remote"`                         // 远程目标地址与端口（例如 "example.com:80"）
	RetryInterval int    `yaml:"retry_interval" mapstructure:"retry_interval" json:"retry_interval"` // 连接失败后的重试间隔（秒）
}

// LocalTunnel 本地端口转发实现（ssh -L）
type LocalTunnel struct {
	SSHConfig     sshx.Config // SSH 配置
	LocalAddr     string      // 本地监听地址，如 "127.0.0.1:8080"
	RemoteAddr    string      // 远程目标地址，如 "example.com:80"
	retry         int         // 当前重试次数
	retryInterval time.Duration
	client        *ssh.Client
	listener      net.Listener
	cancelFunc    context.CancelFunc
}

// NewLocalTunnel 创建 LocalTunnel
func NewLocalTunnel(cfg sshx.Config, config LocalConfig) *LocalTunnel {
	if config.RetryInterval == 0 {
		config.RetryInterval = 5
	}
	return &LocalTunnel{
		SSHConfig:     cfg,
		LocalAddr:     config.Local,
		RemoteAddr:    config.Remote,
		retryInterval: time.Duration(config.RetryInterval) * time.Second,
	}
}

// Start 启动本地端口转发
func (t *LocalTunnel) Start(ctx context.Context) error {
	var err error
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		t.listener, err = net.Listen("tcp", t.LocalAddr)
		if err != nil {
			return err
		}
		slog.Info("[local] started", "local", t.LocalAddr, "remote", t.RemoteAddr)
		ctxTunnel, cancel := context.WithCancel(ctx)
		t.cancelFunc = cancel
		go t.acceptLoop(ctxTunnel)
		return nil
	}
}

// Stop 停止本地端口转发
func (t *LocalTunnel) Stop() error {
	if t.cancelFunc != nil {
		t.cancelFunc()
	}
	if t.listener != nil {
		t.listener.Close()
	}
	if t.client != nil {
		t.client.Close()
	}
	slog.Info("[local] tunnel stopped", "local", t.LocalAddr, "remote", t.RemoteAddr)
	return nil
}

// acceptLoop 接收本地连接并转发到远程
func (t *LocalTunnel) acceptLoop(ctx context.Context) {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				slog.Warn("[local] Accept failed", "local", t.LocalAddr, "err", err)
				continue
			}
		}
		go t.handleConn(conn)
	}
}

// handleConn 处理单个本地连接
func (t *LocalTunnel) handleConn(localConn net.Conn) {
	defer localConn.Close()
	slog.Info("[local] Access", "from", localConn.RemoteAddr(), "to", t.RemoteAddr)
	var (
		err error
	)
	for {
		t.client, err = sshx.Open(t.SSHConfig)
		if err == nil {
			break
		}
		slog.Warn("[local] SSH connect failed, retrying...", "err", err)
		time.Sleep(t.retryInterval)
	}

	remoteConn, err := t.client.Dial("tcp", t.RemoteAddr)
	if err != nil {
		slog.Warn("[local] Remote dial failed", "local", localConn.RemoteAddr(), "remote", t.RemoteAddr, "err", err)
		return
	}
	defer remoteConn.Close()
	done := make(chan struct{}, 2)
	go func() {
		io.Copy(remoteConn, localConn)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(localConn, remoteConn)
		done <- struct{}{}
	}()
	<-done
	slog.Debug("[local] Connection closed", "local", localConn.RemoteAddr(), "remote", t.RemoteAddr)
}
