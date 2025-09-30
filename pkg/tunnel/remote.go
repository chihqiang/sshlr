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

type RemoteConfig struct {
	SSH           string `yaml:"ssh" mapstructure:"ssh" json:"ssh"`                                  // 对应 Ssh.Name 的值，用于指定使用的 SSH 连接
	Local         string `yaml:"local" mapstructure:"local" json:"local"`                            // 本地目标地址（例如 "127.0.0.1:8080"）
	Remote        string `yaml:"remote" mapstructure:"remote" json:"remote"`                         // 远程监听地址（例如 "0.0.0.0:9000"）
	RetryInterval int    `yaml:"retry_interval" mapstructure:"retry_interval" json:"retry_interval"` // 连接失败后的重试间隔（秒）
}

// RemoteTunnel 远程端口转发实现（ssh -R）
type RemoteTunnel struct {
	SSHConfig     sshx.Config
	LocalAddr     string // 本地目标地址，如 "127.0.0.1:3306"
	RemoteAddr    string // 远程监听地址，如 "0.0.0.0:9000"
	retryInterval time.Duration

	client     *ssh.Client
	listener   net.Listener
	cancelFunc context.CancelFunc
}

// NewRemoteTunnel 创建 RemoteTunnel
func NewRemoteTunnel(cfg sshx.Config, config RemoteConfig) *RemoteTunnel {
	if config.RetryInterval == 0 {
		config.RetryInterval = 5
	}
	return &RemoteTunnel{
		SSHConfig:     cfg,
		LocalAddr:     config.Local,
		RemoteAddr:    config.Remote,
		retryInterval: time.Duration(config.RetryInterval) * time.Second,
	}
}

// Start 启动远程端口转发
func (t *RemoteTunnel) Start(ctx context.Context) error {
	var err error
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		t.client, err = sshx.Open(t.SSHConfig)
		if err != nil {
			slog.Warn("[remote] SSH connect failed, retrying", "remote", t.RemoteAddr, "err", err)
			time.Sleep(t.retryInterval)
			continue
		}

		// 远程监听
		t.listener, err = t.client.Listen("tcp", t.RemoteAddr)
		if err != nil {
			t.client.Close()
			slog.Warn("[remote] Failed to listen on remote address", "remote", t.RemoteAddr, "err", err)
			time.Sleep(t.retryInterval)
			continue
		}

		slog.Info("[remote] started", "local", t.LocalAddr, "remote", t.RemoteAddr)
		ctxTunnel, cancel := context.WithCancel(ctx)
		t.cancelFunc = cancel
		go t.acceptLoop(ctxTunnel)
		return nil
	}
}

// Stop 停止远程端口转发
func (t *RemoteTunnel) Stop() error {
	if t.cancelFunc != nil {
		t.cancelFunc()
	}
	if t.listener != nil {
		t.listener.Close()
	}
	if t.client != nil {
		t.client.Close()
	}
	slog.Info("[remote] stopped", "local", t.LocalAddr, "remote", t.RemoteAddr)
	return nil
}

// acceptLoop 接收远程连接并转发到本地
func (t *RemoteTunnel) acceptLoop(ctx context.Context) {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				slog.Warn("[remote] Accept failed", "remote", t.RemoteAddr, "err", err)
				continue
			}
		}
		go t.handleConn(conn)
	}
}

// handleConn 处理单个远程连接
func (t *RemoteTunnel) handleConn(remoteConn net.Conn) {
	defer remoteConn.Close()
	// 访问记录日志
	slog.Info("[remote] Access", "from", remoteConn.RemoteAddr(), "to", t.LocalAddr)

	localConn, err := net.Dial("tcp", t.LocalAddr)
	if err != nil {
		slog.Warn("[remote] Local dial failed", "local", t.LocalAddr, "remote", t.RemoteAddr, "err", err)
		return
	}
	defer localConn.Close()
	done := make(chan struct{}, 2)
	go func() {
		io.Copy(localConn, remoteConn)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(remoteConn, localConn)
		done <- struct{}{}
	}()

	<-done
	slog.Debug("[remote] Connection closed", "local", t.LocalAddr, "remote", remoteConn.RemoteAddr())
}
