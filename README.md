# sshlr

## Project Introduction

sshlr is a lightweight SSH port forwarding tool written in Go, providing more flexible port forwarding management capabilities than the native SSH command. It can manage multiple local and remote port forwarding tunnels simultaneously, and supports features like configuration hot reloading and automatic retry.

## Features

- **Multiple types of port forwarding**: Supports local port forwarding (similar to `ssh -L`) and remote port forwarding (similar to `ssh -R`)
- **Multi-connection management**: Can configure multiple SSH connections and multiple forwarding tunnels at the same time
- **Multiple authentication methods**: Supports password authentication and private key authentication (including keys with passphrases)
- **Configuration hot reloading**: Automatically restarts tunnels after configuration file changes, no manual intervention required
- **Automatic retry mechanism**: Automatically retries after connection failures to improve stability
- **Graceful shutdown**: Supports signal handling to ensure proper resource release
- **Flexible configuration locations**: Supports placing configuration files in multiple locations

## Installation Guide

### Prerequisites
- Install Go 1.23 or higher

### Method 1: Build the binary locally

```bash
# Clone the repository
git clone https://github.com/chihqiang/sshlr.git
cd sshlr
# Compile the project
go build -o sshlr cmd/sshlr/main.go
# Move to system path (optional)
sudo mv sshlr /usr/local/bin/
```

### Method 2: Use `go install`

```bash
# Install the latest version to your Go bin directory (~/.go/bin)
go install github.com/chihqiang/sshlr/cmd/sshlr@latest
```

## Usage

### Basic Usage

1. First, create a configuration file (choose one of the following locations):
   - `config.yaml` in the current directory
   - `.ssh/sshlr.yaml` in the user's home directory
   - `/etc/sshlr.yaml` in the system directory

2. Run the program:

```bash
sshlr
```

### Configuration File Details

The configuration file uses YAML format and mainly contains three parts: `ssh`, `local`, and `remote`.

#### Complete Configuration Example

```yaml
# SSH client connection configuration: supports multiple remote servers
ssh:
  - name: serverA                # Unique identifier, referenced in tunnel configurations
    user: root                   # SSH login username
    host: 192.168.1.100          # SSH host address (can be IP or domain)
    port: 22                     # SSH port (default 22, customizable)
    # Password authentication (choose either password or private key)
    password: "your_password"
    # Private key authentication (optional, omit password if using key)
    # private_key: ~/.ssh/id_rsa           # Path to private key file (supports ~ for home directory)
    # private_key: |                       # Or paste PEM formatted private key directly
    #   -----BEGIN RSA PRIVATE KEY-----
    #   ...
    #   -----END RSA PRIVATE KEY-----
    # passphrase: "your_key_passphrase"     # Passphrase for encrypted private key (if any)
    # Connection timeout in seconds, default 10
    # timeout: 10

# Forward tunnel configuration: local port forwarding to remote server (equivalent to ssh -L)
local:
  - ssh: serverA                 # Which SSH client to use (matches name above)
    local: 127.0.0.1:8888        # Local listen address (access this port to forward)
    remote: 127.0.0.1:80         # Forward target address (remote service port, e.g., nginx)
    # Retry interval in seconds after connection failure, default 5
    # retry_interval: 5

# Reverse tunnel configuration: remote port forwarding to local service (equivalent to ssh -R)
remote:
  - ssh: serverA
    local: 127.0.0.1:8080        # Local service address (service running on your machine)
    remote: 0.0.0.0:8088         # Remote server listen port (accessible externally)
    # Retry interval in seconds after connection failure, default 5
    # retry_interval: 5
```

#### Configuration Field Description

**SSH Connection Configuration** (`ssh`):
- `name`: Unique identifier for the SSH connection, referenced in tunnel configurations
- `user`: SSH login username
- `host`: SSH server address (IP or domain)
- `port`: SSH server port (default 22)
- `password`: Password authentication (choose either password or private key)
- `private_key`: Path to private key file or PEM formatted private key content
- `passphrase`: Passphrase for the private key (if the key is encrypted)
- `timeout`: Connection timeout in seconds (default 10)

**Local Port Forwarding Configuration** (`local`):
- `ssh`: Name of the referenced SSH connection (must match `name` in `ssh` configuration)
- `local`: Local listen address and port (e.g., `127.0.0.1:8888`)
- `remote`: Remote target address and port (e.g., `127.0.0.1:80`)
- `retry_interval`: Retry interval in seconds after connection failure (default 5)

**Remote Port Forwarding Configuration** (`remote`):
- `ssh`: Name of the referenced SSH connection (must match `name` in `ssh` configuration)
- `local`: Local target address and port (e.g., `127.0.0.1:8080`)
- `remote`: Remote listen address and port (e.g., `0.0.0.0:8088`)
- `retry_interval`: Retry interval in seconds after connection failure (default 5)

## Usage Scenarios

### Scenario 1: Access Remote Internal Network Services

Use local port forwarding to map internal network services on a remote server to your local machine:

```yaml
ssh:
  - name: remote-server
    user: ubuntu
    host: example.com
    private_key: ~/.ssh/id_rsa

local:
  - ssh: remote-server
    local: 127.0.0.1:3306
    remote: 10.0.0.5:3306  # MySQL service in the remote internal network
```

With this configuration, accessing `127.0.0.1:3306` locally is equivalent to accessing `10.0.0.5:3306` in the remote internal network.

### Scenario 2: Expose Local Services to the Public Network

Use remote port forwarding to expose a local development web service to a public server:

```yaml
ssh:
  - name: public-server
    user: root
    host: public.example.com
    private_key: ~/.ssh/id_rsa

remote:
  - ssh: public-server
    local: 127.0.0.1:8080  # Local development web service
    remote: 0.0.0.0:80     # Port 80 on the public server
```

With this configuration, users accessing `public.example.com:80` will be forwarded to your local `127.0.0.1:8080`.
