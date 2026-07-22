# Command Reference Manual

This document details the usage of all tbox commands.

## Table of Contents

- [Node Commands (node)](#node-commands-node)
- [Subscription Commands (sub)](#subscription-commands-sub)
- [Setting Commands (setting)](#setting-commands-setting)
- [Request Split Commands (rule route)](#request-split-commands-rule-route)
- [DNS Split Commands (rule dns)](#dns-split-commands-rule-dns)
- [Run Commands (run)](#run-commands-run)
- [Other Commands](#other-commands)

---

## Node Commands (node)

### View node list
```
node [index-expr] [-d]
```
**Parameters**:
- `index-expr` — optional, supports a single index, a range (e.g. 1-10), or comma-separated values (e.g. 1,3,5); defaults to 'all'
- `-d, --desc` — view in descending order

**Examples**:
```bash
node           # view all nodes
node 1-10      # view the first 10 nodes
node -d        # view in descending order
```

### Test node latency
```
node tcping
```
Runs a TCP latency test on all nodes; the results are shown in the node list.

### Sort nodes
```
node sort {0|1|2|3|4|5}
```
**Sort modes**:
- `0` — reverse the current order
- `1` — sort by protocol
- `2` — sort by alias
- `3` — sort by address
- `4` — sort by port
- `5` — sort by test result

### View node details
```
node info {index}
```
Displays the full information of a single node.

### Delete nodes
```
node rm {index-expr}
```
**Examples**:
```bash
node rm 1          # delete node 1
node rm 1,3,5      # delete nodes 1, 3, and 5
node rm 1-10       # delete the first 10 nodes
```

### Find nodes
```
node find {keyword}
```
Finds nodes by alias.

### Add nodes
```
node add [flags]
```
**Flags**:
- `-l, --link {link}` — import a single node from a link
- `-f, --file {path}` — import from a node-link file or subscription file
- `-c, --clipboard` — import by reading from the clipboard

**Examples**:
```bash
node add -l vless://xxx@server:port#name
node add -f /path/to/nodes.txt
node add -c
```

### Export nodes
```
node export [index-expr] [flags]
```
**Flags**:
- `-c, --clipboard` — export to the clipboard

**Examples**:
```bash
node export        # export all nodes
node export 1-5 -c # export the first 5 nodes to the clipboard
```

---

## Subscription Commands (sub)

### View subscription list
```
sub
```
Displays all subscription info, including index, subscription URL, User-Agent, and enabled status.

### View subscription remarks
```
sub remark
```
Displays the remarks of all subscriptions, useful for viewing them separately when the default list omits the remark column.

### Add a subscription
```
sub add {subscription-url} [flags]
```
**Flags**:
- `-r, --remarks {alias}` — set the subscription alias
- `-a, --ua {User-Agent}` — set an independent User-Agent for this subscription

**Examples**:
```bash
sub add https://example.com/sub -r my-subscription -a sing-box
```

### Modify a subscription
```
sub mv {index} [flags]
```
**Flags**:
- `-u, --url {subscription-url}` — modify the subscription URL
- `-r, --remarks {alias}` — modify the alias
- `-a, --ua {User-Agent}` — modify the User-Agent
- `--using {y|n}` — set whether it is enabled

**Examples**:
```bash
sub mv 1 -r new-name
sub mv 1 -a Clash/v1.2.3
sub mv 1 --using n
```

### Delete a subscription
```
sub rm {index}
```

### Update subscription nodes
```
sub update-node [index-expr] [flags]
```
**Flags**:
- `-s, --socks5 [port]` — update via a SOCKS5 proxy
- `-h, --http [port]` — update via an HTTP proxy
- `-a, --addr {address}` — modify the proxy address

**Examples**:
```bash
sub update-node        # update all enabled subscriptions
sub update-node 1      # update subscription 1
sub update-node -s     # update via SOCKS5 proxy
```

---

## Setting Commands (setting)

### View all settings
```
setting
```

### Basic connection settings

#### SOCKS port
```
setting socks [port]
```
Default 23333.

#### HTTP port
```
setting http [port]
```
0 disables the HTTP listener.

#### UDP forwarding
```
setting udp [y|n]
```

#### Traffic address sniffing
```
setting sniffing [y|n]
```

#### Allow LAN connections
```
setting from_lan_conn [y|n]
```

#### Multiplexing
```
setting mux [y|n]
```
Recommended to disable when downloading or watching video.

#### Allow insecure connections
```
setting allow_insecure [y|n]
```

#### Global User-Agent
```
setting user_agent [ua]
```
Defaults to `sing-box`.

### DNS settings

```
setting dns.port [port]
setting dns.foreign [dns]
setting dns.domestic [dns]
setting dns.backup [dns]
```

### Routing settings

```
setting routing.strategy {1|2|3}
```
- `1` — AsIs
- `2` — IPIfNonMatch
- `3` — IPOnDemand

```
setting routing.bypass [y|n]
```
Whether to bypass the LAN and mainland China.

### Test settings

```
setting test.url [url]
setting test.timeout [seconds]
setting test.mintime [milliseconds]
```

### Startup commands

```
setting run_before [command-group] [-c]
```
- `-c, --close` — do not run any command at startup

**Examples**:
```bash
setting run_before "sub update-node | node tcping"
setting run_before -c
```

### Client core (auto-selected, no manual setting needed)

tbox uses sing-box as the default core and automatically picks the core that can actually run each node based on its protocol and transport — no manual switching required:

| Node case | Core actually used | Reason |
|---|---|---|
| xhttp transport (VLESS/VMess) | xray | sing-box doesn't support xhttp |
| Hysteria2 / TUIC / AnyTLS | sing-box | the xray core doesn't support these protocols |
| ShadowsocksR | sing-box + ssr bridge | neither sing-box nor xray provides an SSR outbound; carried by the standalone ssr converter |
| Other protocols | sing-box (default) | supported by both cores |

SSR nodes always run in a "sing-box + ssr" dual-core bridge; xhttp nodes run in a "sing-box + xray" bridge when TUN is enabled (see section 8 of [advanced.md](advanced.md)).

If the selected core is missing, it is auto-downloaded from GitHub (sing-box, xray, and ssr all support auto-download; xray also releases geoip.dat / geosite.dat alongside the binary). On mainland China networks, a mirror is used automatically for acceleration, controllable via the `TBOX_GH_MIRROR` environment variable (see [advanced.md](advanced.md)).

### sing-box advanced settings

#### Chain proxy
```
setting chain_proxy [index-expr|off]
```
Set one or more upstream relay chains for the currently running node; `off` disables it.

**Index expression**:
- Single index: `1`
- Comma-separated: `1,3,5`
- Range: `1-3,5`

**Traffic path**:
```
local → chain node 1 → chain node 3 → chain node 5 → current running node → destination site
```

**Examples**:
```bash
setting chain_proxy 1,3,5
setting chain_proxy 1-3,5
setting chain_proxy off
```

Viewing the setting shows the parsed node list, including protocol, node name, and order, making it easy to confirm the final multi-hop path.

#### TUN mode (Linux only)
```
setting tun_mode [y|n]
setting tun_auto_route [y|n]
setting tun_auto_redirect [y|n]
```

#### Config directory (sing-box)
```
setting config_dir [path]
```
Sets the sing-box working directory. When a non-empty path is set, sing-box starts with the `-C` argument pointing at that directory as its working directory.

**Effect**:
- Specifies sing-box's working directory, affecting where cache, log, and other files are stored
- Empty by default, using the default directory

**Examples**:
```bash
setting config_dir /var/lib/sing-box
setting config_dir ~/.config/sing-box
setting config_dir ""  # clear the setting, restore default
```

---

## Request Split Commands (rule route)

> sing-box only.

```
rule route                view the number of request split rules
rule route ls             list all request split rules
rule route add [flags]    add a rule
rule route edit {index} [flags]  modify a rule (incremental add/remove)
rule route rm {rule-index}   delete a rule
rule route clear          clear all rules
```
Manages sing-box's custom request split rules. Rules are matched in order; once a rule hits, traffic is forwarded to the specified target.

**add Flags**:
- `--target {proxy|direct|block|node:<index>}` — the outbound target after the rule hits
- `--action {route|resolve}` — the rule action, defaults to `route`
  - `route` — route directly to the target outbound
  - `resolve` — perform domain resolution first, then route (does not specify an `outbound`)
- `--inbound {tag[,tag...]}` — match inbound tags
- `--ip-cidr {cidr[,cidr...]}` — match IP CIDR ranges
- `--domain {domain[,domain...]}` — match full domains
- `--domain-suffix {suffix[,suffix...]}` — match domain suffixes
- `--geosite {name[,name...]}` — match geosite categories, e.g. `cn`, `netflix`
- `--match-outbound {tag[,tag...]}` — match existing outbound tags

**target explanation**:
- `proxy` — the currently running node
- `direct` — direct connection
- `block` — block
- `node:<index>` — specify a node outbound, e.g. `node:5`

**edit Flags** (incremental add/remove on the existing rule, no need to rebuild the whole rule):
- `--add-domain / --rm-domain`, `--add-domain-suffix / --rm-domain-suffix`
- `--add-geosite / --rm-geosite`, `--add-ip-cidr / --rm-ip-cidr`
- `--add-match-outbound / --rm-match-outbound`, `--add-inbound / --rm-inbound`
- `--target / --action` — directly overwrite the target/action

**Examples**:
```bash
rule route ls
rule route add --target direct --geosite cn
rule route add --target node:5 --domain-suffix youtube.com
rule route add --action resolve --domain-suffix google.com
rule route edit 1 --add-domain a.com,b.com --rm-domain c.com
rule route rm 2
rule route clear
```

**Domain matching after DNS resolution**: tbox enables `reverse_mapping` in sing-box's DNS config, and performs `http` / `tls` / `quic` sniff before routing. When an application queries DNS first and subsequent connections only appear as IP requests, sing-box tries to recover the original domain via DNS cache or TLS SNI, so `--domain`, `--domain-suffix`, and `--geosite` rules still hit. DNS reverse mapping requires the app's DNS queries to go through tbox/sing-box; HTTPS sites can usually also be recovered via TLS SNI sniff.

---

## DNS Split Commands (rule dns)

> sing-box only.

```
rule dns                  view the DNS split configuration and rule count
rule dns ls               list all DNS split rules
rule dns add --server {local|domestic|backup|foreign} [flags] [--detour <target>]
rule dns edit {index} [flags]  modify a rule (incremental add/remove)
rule dns rm {rule-index}     delete a DNS rule
rule dns clear            clear all DNS split rules
```
Manages sing-box's custom DNS split rules. Rules are matched in order; once a rule hits, the query is sent to the specified DNS server.

**server alias mapping**:
- `local` — local resolver, mapped to sing-box DNS server tag `local`
- `domestic` — domestic DNS, mapped to `domestic_1`
- `backup` — backup domestic DNS, mapped to `domestic_2`
- `foreign` — foreign DNS, mapped to `foreign`

**add Flags**:
- `--server {local|domestic|backup|foreign}` — the DNS server alias to use after the rule hits
- `--domain {domain[,domain...]}` — match full domains
- `--domain-suffix {suffix[,suffix...]}` — match domain suffixes
- `--domain-keyword {keyword[,keyword...]}` — match domain keywords
- `--rule-set {name[,name...]}` — match sing-box rule-set tags, e.g. `geosite-cn`, `geoip-cn`
- `--ip-cidr {cidr[,cidr...]}` — match IP CIDR ranges
- `--query-type {type[,type...]}` — match DNS query types, e.g. `A`, `AAAA`, `HTTPS`
- `--inbound {tag[,tag...]}` — match inbound tags
- `--detour {proxy|direct|block|node:index}` — specify the outbound node for this DNS server (optional)

**edit Flags** (incremental add/remove on the existing rule):
- `--add-domain / --rm-domain`, `--add-domain-suffix / --rm-domain-suffix`
- `--add-domain-keyword / --rm-domain-keyword`, `--add-rule-set / --rm-rule-set`
- `--add-ip-cidr / --rm-ip-cidr`, `--add-query-type / --rm-query-type`
- `--add-inbound / --rm-inbound`
- `--server / --detour` — directly overwrite the server/outbound

**detour explanation**:
- detour ultimately applies to the **DNS server**, not to an individual rule
- `foreign` uses `proxy` by default; other servers have no default detour
- The default is overridden only when `--detour` is explicitly specified

**Examples**:
```bash
# view DNS config and the count of custom DNS rules
rule dns

# view the current DNS split rules
rule dns ls

# domestic domains use the domestic DNS
rule dns add --server domestic --domain-suffix qq.com,baidu.com

# foreign domains use the foreign DNS
rule dns add --server foreign --domain-suffix google.com,youtube.com

# geosite / geoip rule-sets use the domestic DNS
rule dns add --server domestic --rule-set geosite-cn,geoip-cn

# specific query type uses the foreign DNS
rule dns add --server foreign --query-type HTTPS

# route a DNS server's outbound through a specific node
rule dns add --server foreign --domain-suffix example.com --detour node:2

# modify rule 1: add a domain and switch to the foreign DNS
rule dns edit 1 --add-domain a.com --server foreign

# delete DNS rule 2
rule dns rm 2

# clear all custom DNS rules
rule dns clear
```

---

## Run Commands (run)

### Run a specific node
```
run {index}
```
Starts the proxy core and runs the specified node. The core stays resident in the background in an independent process group; the proxy keeps running after exiting tbox (including `Ctrl+C`). Core logs are written to `<TBOX_HOME>/tbox.core.log`. Before starting, any leftover cores not managed by tbox (such as those started directly by systemd, or leftovers from the previous run) are cleaned up to avoid port/TUN conflicts.

**Examples**:
```bash
run 1    # run node 1
run 5    # run node 5
```

For how to configure auto-start with systemd, see [README](../README.md#background-running-and-auto-start-systemd).

---

## Other Commands

### Stop the proxy
```
stop
```
Stops the currently running proxy core and cleans up any leftover cores not managed by tbox (such as sing-box started directly by systemd), ensuring the proxy is fully shut down.

### View help
```
help [command-name]
```
**Examples**:
```bash
help       # view all commands
help node  # view help for the node command
help sub   # view help for the sub command
```

### Exit the program
```
exit
```
Or press `Ctrl+D`.
