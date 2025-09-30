package conf

import (
	"errors"
	"github.com/chihqiang/sshlr/pkg/sshx"
	"github.com/chihqiang/sshlr/pkg/tunnel"
	"os"
	"path/filepath"
	"strings"
)

// Config 定义整体配置结构
type Config struct {
	Ssh    []sshx.Config         `yaml:"ssh" mapstructure:"ssh" json:"ssh"`          // SSH 客户端配置列表
	Local  []tunnel.LocalConfig  `yaml:"local" mapstructure:"local" json:"local"`    // 本地端口转发配置（ssh -L）
	Remote []tunnel.RemoteConfig `yaml:"remote" mapstructure:"remote" json:"remote"` // 远程端口转发配置（ssh -R）
}

// GetConfigPath 返回第一个存在的配置文件路径（必须是文件），支持 ~ 展开
func GetConfigPath(paths ...string) (string, error) {
	homeDir, _ := os.UserHomeDir() // 获取用户主目录
	for _, p := range paths {
		if strings.HasPrefix(p, "~") {
			p = filepath.Join(homeDir, p[1:])
		}
		info, err := os.Stat(p)
		if err == nil && !info.IsDir() {
			return p, nil
		}
	}
	return "", errors.New("no configuration file found in the provided paths")
}
