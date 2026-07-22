# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

tbox is an interactive Shell CLI (Go 1.19) for managing and testing proxy nodes. It parses share links into protocol structs, generates config for a proxy core (sing-box by default, or xray), and starts/stops that core as a child process. The UI is an [ishell](https://github.com/abiosoft/ishell) REPL, not a flags/subcommands CLI.

## Commands

```bash
# Run the interactive shell (dev)
go run tbox.go

# Build a local binary
go build -o tbox tbox.go

# Cross-compile all OS/arch targets into build/ (uses -trimpath -ldflags="-s -w")
python build.py -d

# Interactively pick one OS/arch to build
python build.py
```

There are no Go unit tests, no Makefile, and no CI. "test" in the app (`node tcping`, `TestNode`) means node connectivity/latency testing, not software tests. `build.py` is the only build system.

## Architecture

Flow: share link → `core/protocols` parse → `core/manage.Manager` (persisted state) → `client/config` generates core-specific JSON → `client.Start`/`Stop` runs the core binary.

- **`tbox.go`** — entrypoint; builds the ishell instance and calls `cmd.InitShell`. If args are passed they run as a one-shot command; otherwise the REPL starts.
- **`cmd/`** — one file per command group (`node`, `subscribe`, `setting`, `rule`, `routing`, `filter`, `alias`, ...). Each exposes an `Init*Shell(shell)` registered in `cmd/shell.go:InitShell`. This is NOT the standard `cmd/<binary>/main.go` layout — these are command packages, not binaries. Flags are parsed by the custom `cmd.FlagsParse`, not `flag`/`pflag`. The `rule` command (`cmd/rule.go`) has two subgroups: `rule route` (traffic split) and `rule dns` (DNS split); the old top-level `traffic_split`/`dns_split` names are kept as compatibility aliases that forward to `rule route`/`rule dns`.
- **`core/protocols/`** — all protocols implement the `Protocol` interface (`protocols.go`). Modes are constants in `mode.go`. `parse.go` parses share links; `Deserialize` (in `protocols.go`) reconstructs a struct from stored `mode: {json}` lines. Supported: VMess, VMessAEAD, VLESS, Trojan, Shadowsocks, ShadowsocksR, Socks, HTTP, Hysteria2, TUIC, AnyTLS.
- **`core/manage/`** — `Manager` is a global singleton (`manage.go`) loaded from `tbox.data.json` in `init()` and written back via `Save()`. Holds nodes, subscriptions, filters, and selected index. `DelNode` deletes directly (no recycle bin).
- **`client/config/`** — `Config` interface with `GenConfig(protocols.Protocol) (string, error)`. `CreateConfig(coreName)` returns `SingBox{}`, `Xray{}`, or `V2ray{}`. Each writes the core's config file and returns its path. `singbox.go` and `xray.go` can `panic` on template failures.
- **`client/`** — `check.go` resolves `CoreName`/`CorePath` (global vars) from settings; `client.go` `Start`/`run`/`Stop` manage the core subprocess (guarded by `cmdMu`), read the first ~20 lines of output to detect startup failure within 300ms, and store the PID in settings. `singbox_download.go` auto-downloads the sing-box binary from GitHub when missing.
- **`core/singbox_split/`** — compiles `rule route`/`rule dns` rules into sing-box route/DNS config fragments. Rules persist in `tbox.rules.json` (`store.go`); `edit.go` has `MergeList`/`RemoveList` helpers for incremental match-field edits.
- **`core/setting/`** — viper-backed settings. `init.go` sets ALL defaults and creates `tbox.setting.toml` if absent; `key/` holds the setting-key string constants. Read via typed helpers (`setting.Socks()`, `setting.ClientCore()`, ...).
- **`log/`** — zap + lumberjack wrapper, initialized in `init()`.

## Conventions & gotchas

- **English only**: all code, comments, commit messages, UI strings, log messages, error messages, and documentation MUST be written in English.
- Module path is bare `tbox` (no `github.com/...` prefix); imports look like `tbox/core/...`.
- Heavy reliance on package `init()` for singleton setup (`manage`, `setting`, `log`, `tbox.go`). Import order can matter because of this.
- Errors are frequently dropped intentionally (e.g. `json.Marshal(p)` ignoring err in `Serialize`).
- Config location resolves from `TBOX_HOME` env var, else the executable's directory (`core/filepath.go:GetConfigDir`). Files use a `tbox.` prefix: `tbox.data.json`, `tbox.setting.toml`, `tbox.routing.json`, `tbox.core.log`, `tbox.rules.json` (sing-box split rules) live there; sing-box config/cache live in `<TBOX_HOME>/sing-box/`. `core/filepath.go` and `core/singbox_split/store.go` `init()` migrate the old unprefixed filenames (`data.json`, `setting.toml`, `routing.json`, `core_access.log`, `singbox_split_rules.json`) on first run.
- TUIC, AnyTLS, Hysteria2 only work on sing-box; VLESS/VMess with xhttp transport only work on xray; ShadowsocksR works on neither current core (new sing-box dropped SSR outbound) and is handled by a third `ssr` converter binary. `config.SelectCore` picks the core per node (see `client/config/service.go`). xray requires `geoip.dat`/`geosite.dat` (released alongside the binary on auto-download, or `CORE_LOCATION_ASSET`); sing-box and ssr do not.
- Bridge mode (`config.UseBridge`): when a node's outbound must run on a converter that can't front traffic itself, sing-box handles inbound/route/DNS/split and forwards its `proxy` outbound over an internal socks port (`config.BridgePort`, default 47890) to a converter core. Triggers: SSR nodes always (converter `ssr`, launched with `-url <ssr link> -local 127.0.0.1:<port>`); xhttp/xray nodes only under TUN (converter `xray`, launched with a JSON bridge config). `client.launchBridge` starts converter then sing-box; both stay resident; `Stop` kills both PIDs (`PID`+`BridgePID`). Under TUN a `process_name: [xray, ssr] → direct-out` route rule prevents the converter's own outbound from looping back through the tunnel. SSR/xhttp nodes are rejected from chain-proxy (can't be a sing-box chain hop).
- On Linux with sing-box TUN mode, the binary needs `CAP_NET_ADMIN` (`setcap cap_net_admin,cap_net_bind_service=ep`).
- Do not override built-in shell commands via alias — `cmd/alias.go` guards against this.

## Adding a protocol

1. Add the `Mode` constant in `core/protocols/mode.go`.
2. Create the protocol struct file in `core/protocols/` implementing `Protocol`.
3. Add link parsing in `core/protocols/parse.go`.
4. Add the `Deserialize` case in `core/protocols/protocols.go`.
5. Add outbound generation in `client/config/singbox.go` (and xray if applicable).

## Docs

- `docs/commands.md` — full command reference.
- `docs/advanced.md` — chain proxy, traffic/DNS split, TUN mode, auto-download.
- `AGENTS.md` — existing knowledge-base summary (overlaps with this file).
