# tbox

tbox is an interactive Shell CLI tool for managing and testing proxy nodes. It supports multiple proxy protocols and can generate configuration and start the sing-box or xray core.

## Features

- **Supported protocols**: VMess, VLESS, Trojan, Shadowsocks, ShadowsocksR, Socks, Hysteria2, TUIC, AnyTLS
- **Interactive shell**: TUI interface based on [ishell](https://github.com/abiosoft/ishell)
- **Automatic core management**: auto-downloads the sing-box core when missing (supports Linux/macOS/Windows)
- **Node testing**: supports TCPing latency testing
- **Subscription management**: supports importing and updating subscription links
- **Multi-core support**: supports sing-box (default) and xray, automatically selected at runtime based on node protocol/transport
- **Chain proxy**: supports configuring multi-hop proxy chains by node index
- **Traffic split**: supports adding, deleting, modifying, and viewing custom routing rules via the `rule route` command
- **DNS split resolution**: supports assigning different DNS servers to different domains/rule-sets via the `rule dns` command
- **Background residency**: the core started by `run` is detached from the CLI lifecycle; the proxy keeps running after exiting tbox (including Ctrl+C), making it suitable for use with systemd auto-start
- **Subscription remark viewing**: the default `sub` list does not show a remark column; view them separately via `sub remark`

## Quick Start

### Build

Requires Go 1.19 or higher:

```bash
go build -o tbox tbox.go
```

Or use the cross-compile script:

```bash
python build.py -d
```

### Run

```bash
./tbox
```

On first run, the sing-box core is downloaded automatically (if not found).

## Usage Guide

### Basic commands

```bash
# Enter the interactive shell
./tbox

# Add a node
node add -link <share link>

# View the node list
node -d

# Run a specific node
run <index>

# Test node latency
node tcping

# View traffic split rules
rule route ls

# Set DNS split for a specific domain
rule dns add --server domestic --domain-suffix qq.com

# Stop the proxy
stop

# View subscription remarks separately
sub remark
```

### Full command reference

For detailed usage of all commands, see:

- **[docs/commands.md](docs/commands.md)** — complete command reference manual (node, sub, setting, rule route, rule dns, run, etc.)
- **[docs/advanced.md](docs/advanced.md)** — advanced feature documentation (chain proxy, traffic split, DNS split resolution, TUN mode, User-Agent, auto-download, background residency and systemd deployment, etc.)

### Background running and auto-start (systemd)

The core (sing-box/xray) started by `run` runs independently, detached from the tbox CLI's process group; the proxy stays resident in the background after exiting tbox (including pressing `Ctrl+C`). Core logs are written to `<TBOX_HOME>/tbox.core.log`. To fully stop it, run `stop` (which also cleans up any unmanaged leftover cores).

With systemd you can achieve auto-start on boot and run the last-selected node. Create a user-level unit at `~/.config/systemd/user/tbox.service`:

```ini
[Unit]
Description=tbox proxy
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/path/to/tbox
ExecStart=/path/to/tbox/tbox run
ExecStop=/path/to/tbox/tbox stop

[Install]
WantedBy=default.target
```

Enable and start:

```bash
systemctl --user daemon-reload
systemctl --user enable --now tbox.service

# View status and logs
systemctl --user status tbox.service
tail -f /path/to/tbox/tbox.core.log

# Stop / restart
systemctl --user stop tbox.service
systemctl --user restart tbox.service
```

Notes:
- `Type=oneshot` + `RemainAfterExit=yes` matches tbox's behavior of "a one-shot command that exits after starting the core, with the core staying resident", so systemd won't wrongly conclude the service has stopped.
- `ExecStart=tbox run` runs the last-selected node; first select a node in the interactive shell with `run <index>`, and systemd will reuse that selection afterward.
- To start on boot (without logging in), use a system-level unit and `loginctl enable-linger <username>`, or move it to `/etc/systemd/system/` and configure `User=`.
- TUN mode requires `CAP_NET_ADMIN`; refer to the TUN section in [docs/advanced.md](docs/advanced.md) below to grant the capability to the core binary.

### Supported share link formats

| Protocol | Link format example |
|------|-------------|
| VMess | `vmess://eyJhZGQiOi...` |
| VLESS | `vless://uuid@server:port?...#name` |
| Trojan | `trojan://password@server:port?...#name` |
| Shadowsocks | `ss://method:password@server:port#name` |
| Socks5 | `socks5://user:password@server:port#name` |
| Hysteria2 | `hysteria2://password@server:port?...#name` / `hy2://password@server:port?...#name` |
| TUIC | `tuic://uuid:password@server:port?congestion_control=bbr...#name` |
| AnyTLS | `anytls://password@server:port?sni=...#name` |

### Environment variables

| Variable | Description |
|------|------|
| `TBOX_HOME` | Config directory path (default: the program's directory) |
| `CORE_HOME` | Core program search path |
| `CORE_LOCATION_ASSET` | geoip.dat/geosite.dat asset file path (xray only) |

### Directory structure

```
<TBOX_HOME>/
├── tbox.data.json     # node and subscription data
├── tbox.setting.toml  # config file
├── tbox.routing.json  # routing rules
├── tbox.rules.json    # sing-box split rules (rule route / rule dns)
├── tbox.core.log      # unified core log (lines prefixed with [sing-box] / [xray])
├── sing-box/           # sing-box-specific directory
│   ├── sing-box        # sing-box binary
│   ├── config.json     # generated sing-box config
│   └── cache.db        # rule-set cache
└── xray/               # xray-specific directory (created automatically when xray is used)
    ├── xray            # xray binary
    ├── config.json     # generated xray config
    ├── geoip.dat       # released with the release
    └── geosite.dat     # released with the release
```

> Legacy unprefixed filenames (`data.json`, `setting.toml`, `routing.json`, `core_access.log`, `singbox_split_rules.json`) are automatically migrated to the `tbox.*` naming on first run.

## Core Differences

### sing-box (default)
- Does not require geoip.dat/geosite.dat files
- Uses remote rule-sets with automatic download
- Supports newer protocols such as TUIC and AnyTLS
- Config and cache stored in the `sing-box/` subdirectory
- Auto-downloaded from GitHub when missing
- HTTP proxy shares a port with SOCKS (mixed inbound)

### xray
- Requires geoip.dat and geosite.dat files (released to `xray/` alongside the binary on auto-download)
- Asset files must be placed in the core directory or the location specified by `CORE_LOCATION_ASSET`
- Does not support the TUIC and AnyTLS protocols

## Configuration

Example `setting.toml` config file:

```toml
socks = 23333              # SOCKS5 listen port
http = 0                   # HTTP listen port (0 means disabled)
udp = true                 # enable UDP
sniffing = true            # enable traffic sniffing
allow_insecure = false     # allow insecure TLS

[dns]
port = 13500
domestic = "119.29.29.29"
foreign = "tcp://9.9.9.9"
backup = "114.114.114.114"

[routing]
strategy = "IPIfNonMatch"
bypass = true              # bypass LAN and mainland China

[client]
core = "sing-box"          # default core: sing-box, xray (actually chosen per node at runtime)

[singbox]
chain_proxy = "1,3,5"      # chain proxy node indices, supports 1,3,5 or 1-3,5; off disables it
# request split and DNS split rules are managed directly via the rule command
# rule route add --target direct --geosite cn
# rule dns add --server foreign --domain-suffix google.com
tun_mode = false           # enable TUN mode (Linux only)
tun_auto_route = true      # TUN auto-route
tun_auto_redirect = false  # TUN auto-redirect (nftables)
```

DNS split rules, like traffic split rules, are managed directly via the `rule` command; for example, use `rule dns add --server domestic --domain-suffix qq.com` to assign a resolution server to a specific domain.

The subscription list no longer shows a remark column by default; to view remarks separately, use `sub remark`.

For detailed command examples of chain proxy, custom traffic split, and DNS split resolution, refer to [docs/commands.md](docs/commands.md) and [docs/advanced.md](docs/advanced.md).

For more advanced configuration, refer to [docs/advanced.md](docs/advanced.md).

## Protocol Support Status

| Protocol | sing-box | xray |
|------|----------|------|
| VMess | ✅ | ✅ |
| VLESS | ✅ | ✅ |
| Trojan | ✅ | ✅ |
| Shadowsocks | ✅ | ✅ |
| ShadowsocksR | ❌ | ❌ |
| Socks | ✅ | ✅ |
| Hysteria2 | ✅ | ❌ (automatically uses sing-box) |
| TUIC | ✅ | ❌ (automatically uses sing-box) |
| AnyTLS | ✅ | ❌ (automatically uses sing-box) |

> ShadowsocksR: newer sing-box has removed the SSR outbound, and xray doesn't support it either, so SSR nodes are carried by the standalone
> [ssr converter](https://github.com/bonayakuafo/ssr): sing-box acts as the frontend (inbound/routing/DNS/split), and
> the SSR outbound is handed off to the ssr converter over an internal socks. Auto-downloaded from GitHub when missing.

## Development

### Project structure

```
.
├── tbox.go              # entrypoint file
├── cmd/                  # shell command implementations
├── client/               # core client management
│   ├── config/           # config generation (sing-box/xray)
│   ├── singbox_download.go # sing-box auto-download
│   └── xray_download.go  # xray auto-download
├── core/                 # core business logic
│   ├── manage/           # node/subscription management
│   ├── node/             # node model
│   ├── protocols/        # protocol parsing
│   ├── routing/          # routing rules
│   └── setting/          # config management
└── log/                  # logging utilities
```

### Adding a new protocol

1. Add the Mode constant in `core/protocols/mode.go`
2. Create the protocol struct file in `core/protocols/`
3. Add the Parse function in `core/protocols/parse.go`
4. Add Deserialize support in `core/protocols/protocols.go`
5. Add outbound generation in `client/config/singbox.go`

## License

MIT License

## Acknowledgements

- [Txray](https://github.com/hsernos/Txray) — most of this project's code is based on this repository; special thanks to the original author.
- [sing-box](https://github.com/SagerNet/sing-box)
- [Xray-core](https://github.com/XTLS/Xray-core)
- [v2ray-core](https://github.com/v2fly/v2ray-core)
- [ishell](https://github.com/abiosoft/ishell)
