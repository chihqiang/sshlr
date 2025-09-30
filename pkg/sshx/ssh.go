package sshx

import (
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config 定义 SSH 客户端连接配置
type Config struct {
	Name       string `yaml:"name" mapstructure:"name" json:"name"`                      // 唯一标识，用于 Local/Remote 配置中引用
	User       string `yaml:"user" mapstructure:"user" json:"user"`                      // SSH 登录用户名
	Host       string `yaml:"host" mapstructure:"host" json:"host"`                      // SSH 主机地址（IP 或域名）
	Port       int    `yaml:"port" mapstructure:"port" json:"port"`                      // SSH 端口（默认 22）
	Password   string `yaml:"password" mapstructure:"password" json:"password"`          // 用户密码（可选，若使用密钥认证则留空）
	PrivateKey string `yaml:"private_key" mapstructure:"private_key" json:"private_key"` // 私钥文件路径或 PEM 内容（可选）
	Passphrase string `yaml:"passphrase" mapstructure:"passphrase" json:"passphrase"`    // 私钥解密密码（可选）
	Timeout    int    `yaml:"timeout" mapstructure:"timeout" json:"timeout"`             // 连接超时时间（秒），默认 10
}

// Ping 测试 SSH 是否可连接
func Ping(cfg Config) bool {
	client, err := Open(cfg)
	if err != nil {
		return false
	}
	defer client.Close()
	return true
}

// Open 建立 SSH 连接并返回客户端实例
func Open(cfg Config) (*ssh.Client, error) {
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	authMethods, err := loadSigner(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10
	}
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		&ssh.ClientConfig{
			User:            cfg.User,
			Auth:            authMethods,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 跳过主机密钥校验（生产环境可替换为 KnownHosts）
			Timeout:         time.Duration(cfg.Timeout) * time.Second,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial ssh %s@%s:%d: %w", cfg.User, cfg.Host, cfg.Port, err)
	}
	return client, nil
}

// loadSigner 根据配置加载认证方式（私钥或密码）
func loadSigner(cfg Config) ([]ssh.AuthMethod, error) {
	var authMethods []ssh.AuthMethod

	// 优先使用私钥
	if cfg.PrivateKey != "" {
		var (
			pemBytes []byte
			err      error
			keyPath  string
		)

		// 如果不是 PEM 格式，则认为是文件路径
		if !isPrivateKeyPEM(cfg.PrivateKey) {
			keyPath = cfg.PrivateKey
			// 支持 ~ 表示用户家目录
			if strings.HasPrefix(keyPath, "~") {
				home, err := os.UserHomeDir()
				if err != nil {
					return nil, fmt.Errorf("failed to get user home dir: %w", err)
				}
				keyPath = filepath.Join(home, keyPath[1:])
			}
			pemBytes, err = os.ReadFile(keyPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read private key file: %w", err)
			}
		} else {
			pemBytes = []byte(cfg.PrivateKey)
		}
		// 解析私钥（支持带 passphrase）
		var signer ssh.Signer
		if cfg.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(cfg.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(pemBytes)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))

	} else if cfg.Password != "" {
		// 其次使用密码认证
		authMethods = append(authMethods, ssh.Password(cfg.Password))
	} else {
		return nil, fmt.Errorf("no ssh authentication method configured (private_key or password required)")
	}

	return authMethods, nil
}

// isPrivateKeyPEM 判断字符串是否为 PEM 格式的私钥
func isPrivateKeyPEM(s string) bool {
	block, _ := pem.Decode([]byte(strings.TrimSpace(s)))
	return block != nil && block.Type != ""
}
