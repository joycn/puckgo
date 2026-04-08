# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Puckgo is a DNS-based transparent proxy for home routers. It intercepts DNS queries and proxies TCP connections, with support for AES-encrypted SOCKS5 upstream and domain-based split-tunneling (proxy vs direct).

## Build & Development

This is a **pre-Go-modules (GOPATH-based)** project. Import path: `github.com/joycn/puckgo`.

```bash
# Build for current platform
go build -ldflags "-w -s"

# Cross-compile targets (see Makefile)
make macos          # darwin/amd64
make linux_x86_64   # linux/amd64
make linux_mipsle   # linux/mipsle (router hardware)

# Run tests
go test -race ./...

# Lint
go vet ./...
```

## Architecture

### Three Proxy Modes

Configured via `config.Mode` (`config/config.go`):

| Mode | Reception | Dialer | Use Case |
|------|-----------|--------|----------|
| `transparent` | `Transparent` (Linux TPROXY + DNS cache lookup) | `CryptoDialer` or `NormalDialer` | Linux router, intercepts all traffic via iptables |
| `local` | SOCKS5 server (`socks.Server`) | `Pac` (split-tunnel: proxy or direct per domain) | Local SOCKS5 client with access list routing |
| `server` | `CryptoReception` or `NormalReception` | `DirectDialer` | Remote endpoint that decrypts and forwards |

### Core Interfaces

**`network.Dialer`** (`network/network.go`) — connects to upstream:
```go
type Dialer interface {
    Dial(addr *socks.AddrSpec) (net.Conn, error)
}
```

**`conn.Reception`** (`conn/conn.go`) — extracts target address from incoming connection:
```go
type Reception interface {
    Recept(c net.Conn) (*socks.AddrSpec, net.Conn, error)
}
```

The `Proxy` struct (`proxy/proxy.go`) embeds both interfaces. `handleConn` accepts a connection, calls `Recept()` to get the target, calls `Dial()` to connect upstream, then does bidirectional `io.Copy`.

### Package Responsibilities

- **cmd/** — CLI (cobra/viper), config parsing, entry point (`start.go` calls `proxy.NewProxy` + `StartProxy`)
- **config/** — Config structs, mode constants, defaults
- **proxy/** — Core proxy engine: listener, connection handling, mode-specific setup (`mode.go`, `mode_linux.go`, `mode_darwin.go`)
- **conn/** — Connection strategies: `CryptoDialer/Reception` (AES-OFB encrypted SOCKS5), `DirectDialer`, `NormalDialer/Reception`, `Pac` (domain-based split-tunnel), `Transparent` (Linux TPROXY)
- **dnsforward/** — DNS forwarder: intercepts queries, resolves via specified server through proxy, caches IP-to-domain mappings, manages ipset entries
- **network/** — Low-level networking: `Dialer` interface, transparent socket options, Linux ip rule/route setup
- **iptables/** — iptables/TPROXY rule management for transparent proxy

### Platform-Specific Code

- `*_linux.go` files contain TPROXY, netlink, ipset, iptables, and transparent socket logic
- `mode_darwin.go` only supports `SocksLocalMode`

### Data Flow

```
Client → [iptables TPROXY / SOCKS5] → Proxy.Listener
  → Reception.Recept() → Dialer.Dial() → bidirectional io.Copy

Transparent mode DNS:
  DNS query → DNSForwarder → resolver (via proxy) → ipset + cache update
  TCP connection → TPROXY → DNS cache lookup for domain → upstream proxy
```

### Key Custom Dependencies

- `github.com/joycn/socks` — SOCKS5 protocol (same author)
- `github.com/joycn/datasource` — Access list / domain matching (same author)
- `github.com/joycn/ttlcache` — TTL cache for DNS-to-IP mappings (same author)

### Configuration

CLI flags (`--config`, `--accesslist`, `--listen`, `--timeout`, `--loglevel`) backed by Viper with config file at `$HOME/.puckgo.yaml`. Key defaults: proxy listens on `:1200`, SOCKS5 on `:1080`, DNS on `0.0.0.0:53`, upstream DNS `8.8.8.8:53`, timeout 300s.
