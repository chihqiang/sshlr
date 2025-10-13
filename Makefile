# 尝试获取最近的 tag，如果没有 tag 就返回空
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null)

# 获取当前 commit 的短 hash
GIT_COMMIT := $(shell git rev-parse --short HEAD)

# 将 config.yaml 内容 Base64 编码，用于嵌入到二进制中
CONFIG_BASE64=$(shell cat config.yaml | base64)

# ------------------------------
# 版本号逻辑
# ------------------------------
# 如果存在 tag → 使用 tag 作为版本号
# 如果没有 tag → 使用 commit hash + -dev 作为开发版本号
version := $(if $(GIT_TAG),$(GIT_TAG),$(GIT_COMMIT)-main)

# ------------------------------
# 构建输出文件名
# ------------------------------
OUTPUT := sshlr

# Go 项目入口文件
MAIN := cmd/sshlr/main.go

# ------------------------------
# 构建目标本地二进制
# ------------------------------
build:
	@echo "🔧 Building $(OUTPUT) with version $(version)..."
	GO111MODULE=on CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=$(version)" -o $(OUTPUT) $(MAIN)
	@echo "✅ Build complete: $(OUTPUT)"


build_config:
	@echo "🔧 Building $(OUTPUT) with embedded config (version=$(version))..."
	GO111MODULE=on CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=$(version) -X main.configBase64Content=$(CONFIG_BASE64)" -o $(OUTPUT) $(MAIN)
	@echo "✅ Build complete: $(OUTPUT)"

# ------------------------------
# 安装目标
# ------------------------------
install: build
	@echo "📦 Installing $(OUTPUT) to /usr/local/bin ..."
	@rm -f /usr/local/bin/$(OUTPUT)
	@install -m 0755 $(OUTPUT) /usr/local/bin/
	@echo "✅ Installed: /usr/local/bin/$(OUTPUT)"
