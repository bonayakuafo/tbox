# PROJECT KNOWLEDGE BASE

**Generated:** 2026-04-13
**Commit:** 5c4aeaa
**Branch:** master

## OVERVIEW
tbox is an interactive shell CLI (Go 1.19) for managing and testing proxy nodes. It wraps sing-box/xray cores with an ishell-based TUI.

## STRUCTURE
```
.
├── tbox.go           # main entrypoint; bootstraps ishell
├── cmd/               # shell command packages (not main binaries)
├── core/              # domain logic: protocols, nodes, routing, settings
├── client/            # core runner: config generation, process control
├── log/               # zap-based logging wrapper
└── build.py           # Python cross-compilation script
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Add a shell command | `cmd/*.go` | Register in `cmd/shell.go` via `Init*Shell` |
| Add a proxy protocol | `core/protocols/` | Implement `Protocol` interface; register in `protocols.go` + `parse.go` |
| Node data / filters / subs | `core/manage/`, `core/node/`, `core/sub/` | `Manage` singleton auto-loads `data.json` |
| Config generation per core | `client/config/` | `Config` interface: `GenConfig(protocols.Protocol) string` |
| Settings / viper defaults | `core/setting/` | `init.go` sets defaults; `key/` holds key constants |
| Build / release | `build.py` | Cross-compiles for multiple OS/arch combos |

## CODE MAP
| Symbol | Type | Location | Role |
|--------|------|----------|------|
| `main` | func | `tbox.go` | Entrypoint |
| `InitShell` | func | `cmd/shell.go` | Registers all shell commands |
| `Protocol` | interface | `core/protocols/protocols.go` | All proxy protocols implement this |
| `Manage` | struct | `core/manage/manage.go` | Singleton state manager |
| `Config` | interface | `client/config/service.go` | Abstraction over sing-box/xray configs |
| `Start` / `Stop` | func | `client/client.go` | Starts/stops the proxy core process |

## CONVENTIONS
- Module path is bare `tbox` (no VCS prefix like `github.com/...`)
- `cmd/` contains multiple non-main packages that register ishell commands; not the standard Go `cmd/<binary>/main.go` pattern
- Heavy use of `init()` for singleton setup: `tbox.go`, `core/manage/manage.go`, `core/setting/init.go`, `log/log.go`, etc.
- Errors are often silently dropped (`jsonData, _ := json.Marshal(p)`)
- **English only**: all code, comments, commit messages, UI strings, log/error messages, and docs MUST be written in English .

## ANTI-PATTERNS (THIS PROJECT)
- Do not override built-in shell commands via alias (`cmd/alias.go` prevents this)
- QUIC security key must not be empty when security is not `"none"`
- ALPN is intentionally commented out for some VMess nodes due to connection issues (`core/protocols/parse.go`)
- Test timeout / min-time values must be `>= 0`

## UNIQUE STYLES
- Interactive shell TUI rather than traditional CLI flags/subcommands
- `build.py` (Python) handles cross-compilation instead of Makefile / `go build` scripts
- `core/manage/manage.go` uses a global `Manager` variable loaded from JSON at init time
- Config directory resolved via `TBOX_HOME` env var, falling back to executable directory

## COMMANDS
```bash
# Run interactive shell
go run tbox.go

# Cross-compile all targets
python build.py -d
```

## NOTES
- No formal `*_test.go` unit tests exist; "test" in filenames refers to node connectivity testing, not software tests
- No CI/GitHub Actions or Makefile present
- `client/config/singbox.go` and `client/config/xray.go` contain `panic(err)` on template failures
