# Advanced Feature Guide

This document details the advanced feature configuration of tbox.

## 1. Chain Proxy (sing-box)

Chain proxy now supports multi-hop relays configured by **node index**, no manual outbound tag entry required. You can chain one or more existing nodes in front of the currently running node to form a multi-hop proxy chain.

**Command format**:
```bash
setting chain_proxy [index-expr|off]
```

**Supported index expressions**:
- Single index: `1`
- Comma-separated: `1,3,5`
- Mixed range: `1-3,5`
- Disable chain proxy: `off`

**Multi-hop chain diagram**:
```text
local
  │
  ├─> node 1
  │     │
  │     └─> node 3
  │            │
  │            └─> node 5
  │                   │
  │                   └─> selected runtime node
  │                              │
  └─────────────────────────────> destination
```

The corresponding traffic path is:
```
local → node 1 → node 3 → node 5 → current running node → destination site
```

**Common examples**:
```bash
# Relay through nodes 1, 3, and 5 hop by hop, then out via the current running node
setting chain_proxy 1,3,5

# Range syntax, equivalent to 1,2,3,5
setting chain_proxy 1-3,5

# Disable chain proxy
setting chain_proxy off
```

**Inspecting the effect**:
- After running `setting chain_proxy 1,3,5`, viewing the settings displays the parsed chain nodes
- The output includes the node protocol and name for confirming the final chain order
- The currently running node is still decided by `run {index}`; `chain_proxy` only prepends relay nodes in front of it

**Applicable scenarios**:
- You need to first enter an ingress node, then relay through nodes in different regions
- You want to stack a pre-exit landing proxy in front of the exit node
- You want to quickly switch between different multi-hop combinations instead of manually editing the sing-box config

---

## 2. Request Split (sing-box)

Request split is now **rule-driven**. You can add, remove, view, and modify split rules directly from the CLI, rather than only doing simple on/off control.

**Command overview**:
```bash
rule route ls
rule route add [flags]
rule route edit {rule-index} [flags]
rule route rm {rule-index}
rule route clear
```

> sing-box only.

### 2.1 View rules

```bash
rule route ls
```

Lists all current custom rules for confirming match conditions, target outbound, and order.

### 2.2 Add a rule

```bash
rule route add --target {proxy|direct|block|node:<index>} --action {route|resolve} [match flags]
```

**Available match conditions** (combine as needed):
- `--inbound`: match inbound tags
- `--ip-cidr`: match IP CIDR ranges
- `--domain`: match full domains
- `--domain-suffix`: match domain suffixes
- `--geosite`: match geosite categories, e.g. `cn`, `netflix`
- `--match-outbound`: match existing outbound tags

**Target explanation**:
- `proxy`: use the currently running node
- `direct`: direct connection
- `block`: block the traffic
- `node:<index>`: use a specific node outbound, e.g. `node:5`

**Action explanation** (`--action`):
- `route` (default): route directly to the target outbound
- `resolve`: perform domain resolution first, then route. When using `resolve`, no `outbound` is generated; use it when the routing decision needs to happen after DNS resolution.

**Practical examples**:
```bash
# Domestic traffic direct
rule route add --target direct --geosite cn

# YouTube goes through node 5, ignoring the current running node
rule route add --target node:5 --domain-suffix youtube.com

# Block specific domains
rule route add --target block --domain ads.example.com

# Route traffic from a given inbound tag directly to the proxy
rule route add --target proxy --inbound tun-in

# Resolve the domain first, then route (no outbound specified)
rule route add --action resolve --domain-suffix google.com
```

### 2.3 Deleting and clearing rules

```bash
rule route rm 2
rule route clear
```

- `rm` deletes a single rule by rule index
- `clear` clears all custom split rules

### 2.4 Concrete effect example

Here is a common combination:

```bash
rule route add --target direct --geosite cn
rule route add --target node:5 --domain-suffix youtube.com
rule route add --target block --domain ads.example.com
rule route ls
```

This set of rules achieves:
- Domestic sites go direct, reducing proxy traffic usage
- `youtube.com` and its subdomains always go through node 5
- `ads.example.com` is blocked outright

If the same request matches multiple rules, rule order generally takes precedence, so put more specific rules first and more general rules later.

### 2.5 Domain matching after DNS resolution

Some sites perform DNS resolution first and then connect using the resolved IP. Without domain reverse-lookup, `domain` / `domain_suffix` / `geosite` rules only see the IP and cannot hit on the original domain.

When generating the sing-box config, tbox enables DNS `reverse_mapping` and runs `http` / `tls` / `quic` sniff at the very beginning of the route rules. As long as the application's DNS queries go through tbox/sing-box, sing-box records the "domain → IP" mapping and reverse-looks up the domain when a subsequent IP connection enters the routing stage. For HTTPS sites, even if the connection target is already an IP, the domain can usually be recovered via TLS SNI sniff, so request-split domain rules continue to take effect.

---

## 3. DNS Split Resolution

DNS split resolution sends **different types of domain queries** to different DNS servers. This lets domestic domains be handed to a domestic DNS first, improving hit rate and resolution speed, while foreign domains are sent to a foreign DNS, reducing pollution and incorrect resolution.

It complements request split:
- Request split decides "which outbound the traffic finally uses"
- DNS split decides "which DNS server resolves the domain first"

**Command overview**:
```bash
rule dns
rule dns ls
rule dns add --server {local|domestic|backup|foreign} [match flags] [--detour <target>]
rule dns edit {rule-index} [match flags]
rule dns rm {rule-index}
rule dns clear
```

> sing-box only. `edit` supports incremental add/remove on the existing rule (`--add-xxx` / `--rm-xxx`) and overwriting `--server` / `--detour`.

### 3.1 How it works

tbox compiles the custom DNS split rules into sing-box's `dns.rules`. Each rule, once matched, performs `action: route` and sends the query to the corresponding DNS server tag.

**server alias mapping**:
- `local` → `local`
- `domestic` → `domestic_1`
- `backup` → `domestic_2`
- `foreign` → `foreign`

Where:
- `domestic_1` corresponds to the domestic DNS configured by `setting dns.domestic`
- `domestic_2` corresponds to the backup domestic DNS configured by `setting dns.backup`
- `foreign` corresponds to the foreign DNS configured by `setting dns.foreign`
- `local` is the local fallback resolver

**detour behavior**:
- You can specify via `--detour` which outbound node a given DNS server uses to send queries
- `foreign` uses the `proxy` outbound by default
- `domestic` / `backup` / `local` have no default detour
- The default is overridden only when `--detour` is used explicitly
- Note: detour ultimately applies at the **DNS server** level, not to an individual rule

### 3.2 Available match conditions

DNS split supports the following match flags (combine as needed):
- `--domain`: match full domains
- `--domain-suffix`: match domain suffixes
- `--domain-keyword`: match domain keywords
- `--rule-set`: match sing-box rule-set tags
- `--ip-cidr`: match IP CIDR ranges
- `--query-type`: match DNS query types, e.g. `A`, `AAAA`, `HTTPS`
- `--inbound`: match inbound tags

### 3.3 Rule priority

DNS split priority follows the order "**custom rules first, built-in default rules last**":

1. First match the custom rules you added via `rule dns add`
2. If no hit and `dns.domestic` is configured, the built-in default rule sends `geoip-cn` / `geosite-cn` to `domestic_1`
3. If still no hit and `dns.foreign` is configured, the default sends general ingress (such as `mixed-in`) to `foreign`
4. If the corresponding alias is not configured, it eventually falls back to `local`

So put more specific custom DNS rules first and let the built-in default rules catch the more general cases.

### 3.4 Common examples

**Example 1: Route common domestic domains to the domestic DNS**

```bash
# First configure the DNS servers
setting dns.domestic 119.29.29.29
setting dns.backup 114.114.114.114
setting dns.foreign tcp://9.9.9.9

# Route the given domestic domain suffixes to the domestic DNS
rule dns add --server domestic --domain-suffix qq.com,baidu.com,bilibili.com
```

Good for handing domains that are clearly domestic sites directly to the domestic DNS, reducing false negatives.

**Example 2: Route foreign domains to the foreign DNS**

```bash
rule dns add --server foreign --domain-suffix google.com,youtube.com,github.com
```

Resolution of these domains prioritizes the foreign DNS, which usually pairs better with proxy nodes.

**Example 3: Directly reuse sing-box rule-sets**

```bash
rule dns add --server domestic --rule-set geosite-cn,geoip-cn
rule dns add --server foreign --query-type HTTPS
rule dns ls
```

This style is good for directly leveraging existing rule-set categories or separately controlling the resolver by query type.

**Example 4: Route DNS queries through a specific node**

```bash
# Route the foreign DNS server's queries through node 2
rule dns add --server foreign --domain-suffix example.com --detour node:2
```

If you want a DNS server not to use its default outbound (for example, `foreign` defaults to `proxy` but you want it through a specific node), use `--detour` to override.

### 3.5 Usage suggestions

- Domestic-first scenario: configure `dns.domestic` / `dns.backup` first, then add `domestic` rules for common domestic domains
- Foreign-service scenario: add `foreign` rules for domains like `google.com`, `youtube.com`, `github.com`
- On rule conflict: keep more specific domain rules first in the list

---

## 4. TUN Mode (Linux only)

TUN mode creates a virtual network interface and transparently pipes system-level traffic into sing-box.

**Configuration**:
```bash
setting tun_mode y
setting tun_auto_route y
setting tun_auto_redirect y
```

**Permission setup** (required once):
```bash
# Ubuntu/Debian
sudo apt install libcap2-bin
sudo setcap cap_net_admin,cap_net_bind_service=ep /path/to/sing-box

# CentOS/RHEL
sudo yum install libcap
sudo setcap cap_net_admin,cap_net_bind_service=ep /path/to/sing-box
```

**TUN mode effects**:
- No need to manually configure a SOCKS5 proxy for each application
- System-wide traffic goes through sing-box automatically
- Transparent proxy for both TCP and UDP

**Permission check on startup**:
If sing-box lacks `CAP_NET_ADMIN`, tbox prints:
```
[warn] the current sing-box binary lacks CAP_NET_ADMIN capability; TUN mode may not work correctly
[warn] please run: sudo setcap cap_net_admin,cap_net_bind_service=ep /path/to/sing-box
```

---

## 5. Subscription User-Agent

**Set an independent User-Agent for a single subscription**:
```bash
# specify the UA when adding a subscription
sub add https://example.com/sub -a sing-box

# modify the UA of an existing subscription
sub mv 1 -a sing-box
```

**Global default UA**:
The default is `sing-box`; change it via:
```bash
setting user_agent sing-box
```

**Priority**:
Subscription-level UA > Global UA > Default `sing-box`

---

## 6. Auto-Download Core (sing-box / xray)

When the required core is missing, tbox auto-downloads the latest release from GitHub. Both sing-box and xray are supported.

**Download flow**:
1. Try to fetch the latest release via the GitHub API
2. If the API is rate-limited, fall back to a pinned version (sing-box `v1.13.7`, xray `v1.8.24`)
3. Pick the corresponding asset for the system architecture
4. Auto-extract and place it in the core's dedicated directory (`<TBOX_HOME>/sing-box/` or `<TBOX_HOME>/xray/`)
5. Set the Unix executable bit

The xray release archive contains `geoip.dat` and `geosite.dat`; they are extracted alongside into the `xray/` directory, no manual download needed.

**Supported architectures**:
- Linux: amd64, arm64, 386, arm
- macOS: amd64, arm64
- Windows: amd64, arm64, 386 (xray has no windows/arm)

### 6.1 Mirror acceleration in mainland China

When downloading a core, tbox detects (via a public geo API) whether the outbound IP is in mainland China. If so, it automatically prefixes the GitHub download URL with the mirror `https://ghfast.top/` for acceleration.

Manual control is available via the `TBOX_GH_MIRROR` environment variable:

| Value | Behavior |
|---|---|
| unset | auto-detect; only enables the mirror on mainland China networks |
| `off` / `0` / `false` / `no` | force-disable the mirror, connect to GitHub directly |
| other value (must end with `/`) | use as a custom mirror prefix, skip IP detection |

If the geo API is unreachable, it silently falls back to a direct connection and doesn't block the download.

### 6.2 Per-node auto core selection

tbox auto-picks the core that can actually run each node based on its protocol and transport; no need to switch the default core manually:

- **xhttp transport** (VLESS/VMess) → automatically uses **xray** (sing-box doesn't support xhttp)
- **Hysteria2 / TUIC / AnyTLS** → automatically uses **sing-box** (the xray core doesn't support them)
- **ShadowsocksR** → carried by the standalone **ssr converter** (the new sing-box has dropped the SSR outbound, and xray doesn't support it either)
- Other protocols → use the default core (sing-box)

If the auto-selected core has not been downloaded, it is fetched via the flow above.

**SSR / bridge mode**: SSR nodes cannot be an outbound of sing-box/xray, so tbox starts a "sing-box + ssr" dual-core bridge — sing-box handles inbound/routing/DNS/split, and its `proxy` outbound is forwarded through an internal socks port (default `127.0.0.1:47890`, auto-shifted on conflict) to the ssr converter for protocol conversion. Similarly, xhttp nodes start a "sing-box + xray" bridge when TUN is enabled (xray has no TUN capability). Under bridge mode, `stop` shuts down both cores together.

**Split targets using xhttp/ssr**: when the main node uses a common protocol but some traffic should go through an xhttp/ssr node, use `rule route add --target node:<xhttp-or-ssr-node> ...` (DNS split's `--detour node:N` works the same way). tbox starts the converter for that node and forwards the matched split traffic to it via the internal socks port; the main outbound is still carried by sing-box natively, and the two don't interfere with each other.

> Constraint: at any given runtime, **at most one** xhttp/ssr node actually carries an outbound. It can be both the main node and a split target (they must be the same node), but if the main node uses xhttp1 while a split points to another xhttp2 (or if two different xhttp/ssr nodes appear in the split), `run` errors out and refuses to start.

---

## 7. Directory Structure

```
<TBOX_HOME>/
├── tbox.data.json     # node and subscription data
├── tbox.setting.toml  # user configuration
├── tbox.routing.json  # routing rules
├── tbox.rules.json    # sing-box split rules (rule route / rule dns)
├── tbox.core.log      # unified core log
├── sing-box/           # sing-box dedicated directory
│   ├── sing-box        # sing-box binary
│   ├── config.json     # generated sing-box config
│   └── cache.db        # rule-set cache
└── xray/               # xray dedicated directory (created on demand when xray is used)
    ├── xray            # xray binary
    ├── config.json     # generated xray config
    ├── geoip.dat       # released alongside the binary
    └── geosite.dat     # released alongside the binary
```

> Note: stdout/stderr from both cores is written directly to `tbox.core.log` (the cores no longer keep their own log file). A separator line `==== [core-name] run @ time ====` is written before each start, making it easy to tell different cores and different launches apart. The cores hold the log file handle directly, so they can keep writing to the log after tbox exits. If the log exceeds 5MB, it is auto-cleared before the next start.

> On first run, old un-prefixed file names (`data.json`, `setting.toml`, `routing.json`, `core_access.log`, `singbox_split_rules.json`) are auto-migrated to the `tbox.*` naming above.

---

## 8. Background Running and systemd Deployment

### 8.1 Core-resident mechanism

The core (sing-box/xray) started by `run` is intentionally decoupled from the tbox CLI, and the proxy keeps running in the background after tbox exits:

- **Process-group isolation**: the core is started in an independent process group via `Setpgid`. When you press `Ctrl+C` in the terminal, `SIGINT` only reaches the foreground process group that tbox belongs to and does not affect the core.
- **Direct log writing**: the core's output is written directly to `tbox.core.log`, not through a tbox pipe. So when tbox exits and the read end of its pipe closes, the core is not passively terminated by `SIGPIPE` from writing logs.
- **Leftover cleanup**: `run` / `node tcping` (speed test) / `stop` clean up any leftover cores not managed by tbox at their respective entry points (matched exactly by the absolute path of the core binary), avoiding contention for the same socks port or TUN interface with the core about to start.

Stopping:
- `stop` command: stops the core that tbox recorded and cleans up leftover cores.
- Direct `kill <pid>`: the core's PID is recorded in the settings and can also be found via `ps`/`pgrep`.

### 8.2 systemd user unit

Leveraging the core-resident behavior, you can manage tbox with systemd to auto-start on boot and run the previously selected node. Create `~/.config/systemd/user/tbox.service`:

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

Key points:
- `Type=oneshot` + `RemainAfterExit=yes`: tbox is a "one-shot command that starts the core and then exits" model; the core is the resident process. `oneshot` makes systemd keep the service active after `ExecStart` returns, instead of mistaking "started then exited" as a failure and restarting repeatedly.
- `ExecStart=tbox run` (no index) runs the **previously selected node**. First pick the node in the interactive shell with `run <index>`; systemd will then reuse that selection.
- `ExecStop=tbox stop` shuts down the core and cleans up leftovers.

Enabling and management:

```bash
systemctl --user daemon-reload
systemctl --user enable --now tbox.service

# status and logs
systemctl --user status tbox.service
tail -f /path/to/tbox/tbox.core.log

# stop / restart
systemctl --user stop tbox.service
systemctl --user restart tbox.service
```

### 8.3 Extra notes

- **Start before login**: user-level units start by default only after the user logs in. To run at boot (without needing a login), run `loginctl enable-linger <username>`, or switch to a system-level unit (place it in `/etc/systemd/system/` and set `User=`).
- **TUN mode**: the core binary needs `CAP_NET_ADMIN`. Refer to the TUN section in this document and run `sudo setcap cap_net_admin,cap_net_bind_service=ep <core-path>`, otherwise launching TUN via systemd as a regular user will fail.
- **TBOX_HOME**: systemd's working directory and `PATH` may differ from a login shell. It's recommended to set `WorkingDirectory` in the unit, and if needed use `Environment=TBOX_HOME=/path/to/tbox` to explicitly specify the config directory, and `Environment=CORE_HOME=...` to specify the core search path.
