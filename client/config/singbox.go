package config

import (
	"tbox/core"
	"tbox/core/manage"
	"tbox/core/protocols"
	"tbox/core/protocols/field"
	"tbox/core/routing"
	"tbox/core/setting"
	"tbox/core/setting/key"
	"tbox/core/singbox_split"
	"tbox/log"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type SingBox struct{}

// RuleSetMirrorPrefix is the mirror prefix for remote rule-set download URLs;
// empty by default (direct GitHub). When direct download timeouts cause
// startup to fail, the client sets it to a mirror prefix and regenerates the
// config to retry once; on success (rule-set is cached in cache.db and won't
// be downloaded again), it is cleared so we don't use the mirror long-term.
// In-process only, not persisted.
var RuleSetMirrorPrefix string

// rulesetURL prepends the current mirror prefix (if any) to a remote rule-set URL.
func rulesetURL(raw string) string {
	if RuleSetMirrorPrefix == "" {
		return raw
	}
	return RuleSetMirrorPrefix + raw
}

type singBoxSplitData struct {
	routeRules       []interface{}
	dnsRules         []interface{}
	outbounds        []interface{}
	dnsServerDetours map[string]string
}

// GenConfig generates the core config file.
func (s SingBox) GenConfig(node protocols.Protocol) (string, error) {
	coreDir := core.GetCoreDir("sing-box")
	if err := os.MkdirAll(coreDir, 0755); err != nil {
		log.Error(err)
		return "", err
	}
	return s.genConfig(node, nil, 0)
}

// GenBridgeConfig generates the sing-box config for bridge mode: same as the
// regular config, except the main/split outbounds are rewritten per plan to
// point at 127.0.0.1:bridgePort socks (handed off to the converter). TUN /
// routing / DNS all remain in sing-box. plan describes the converter node and
// the shape (see BridgePlan).
func (s SingBox) GenBridgeConfig(node protocols.Protocol, plan *BridgePlan, bridgePort int) (string, error) {
	coreDir := core.GetCoreDir("sing-box")
	if err := os.MkdirAll(coreDir, 0755); err != nil {
		log.Error(err)
		return "", err
	}
	return s.genConfig(node, plan, bridgePort)
}

func (s SingBox) genConfig(node protocols.Protocol, plan *BridgePlan, bridgePort int) (string, error) {
	coreDir := core.GetCoreDir("sing-box")
	splitData := s.loadSplitData(plan)
	path := filepath.Join(coreDir, "config.json")
	var conf = map[string]interface{}{
		"log":          s.logConfig(),
		"inbounds":     s.inboundsConfig(),
		"outbounds":    s.outboundConfig(node, splitData.outbounds, plan, bridgePort),
		"dns":          s.dnsConfig(splitData.dnsRules, splitData.dnsServerDetours),
		"route":        s.routingConfig(splitData.routeRules, bridgePort, node != nil && node.GetProtocolMode() == protocols.ModeDirect),
		"experimental": s.experimentalConfig(),
	}
	err := core.WriteJSON(conf, path)
	if err != nil {
		log.Error(err)
		return "", err
	}
	return path, nil
}

func (s SingBox) loadSplitData(plan *BridgePlan) singBoxSplitData {
	data := singBoxSplitData{
		routeRules:       make([]interface{}, 0),
		dnsRules:         make([]interface{}, 0),
		outbounds:        make([]interface{}, 0),
		dnsServerDetours: make(map[string]string),
	}

	ruleSet, err := singbox_split.Load()
	if err != nil {
		log.Warn(err)
		return data
	}

	// Under bridge mode, the converter node's split-node-N outbound is
	// replaced by the bridge socks outbound; we no longer emit an independent
	// protocol outbound (sing-box can't outbound xhttp/ssr). In shape 1 (main
	// node is the converter) the bridge tag is proxy; in shape 2 (converter
	// serves only split targets) it is converter-out.
	converterIndex := 0
	bridgeTag := converterTag
	if plan != nil && plan.Enabled {
		converterIndex = plan.NodeIndex
		if plan.MainIsConverter {
			bridgeTag = "proxy"
		}
	}

	addedOutbounds := make(map[int]struct{})
	for i, rule := range ruleSet.Rules {
		if rule == nil {
			log.Warnf("skipping empty split rule: %d", i+1)
			continue
		}

		routeRules, indexes, err := singbox_split.CompileRules(&singbox_split.RuleSet{Rules: []*singbox_split.Rule{rule}})
		if err != nil {
			log.Warnf("failed to compile split route rule, skipping rule %d: %v", i+1, err)
			continue
		}

		// Filter out the converter node index: it does not emit an independent
		// outbound; converter-out handles it instead.
		nativeIndexes := excludeIndex(indexes, converterIndex)
		if !s.ensureSplitOutbounds(nativeIndexes, addedOutbounds, &data.outbounds) {
			log.Warnf("split rule references invalid node, skipping rule %d", i+1)
			continue
		}

		for _, compiledRule := range routeRules {
			if converterIndex > 0 {
				rewriteSplitOutbound(compiledRule, converterIndex, bridgeTag)
			}
			data.routeRules = append(data.routeRules, compiledRule)
		}
	}

	dnsRules, dnsNodeIndexes, dnsServerDetours, err := singbox_split.CompileDNSRules(ruleSet, manage.Manager.NodeLen())
	if err != nil {
		log.Warnf("failed to compile DNS split rules: %v", err)
	} else {
		nativeDNSIndexes := excludeIndex(dnsNodeIndexes, converterIndex)
		if !s.ensureSplitOutbounds(nativeDNSIndexes, addedOutbounds, &data.outbounds) {
			log.Warn("DNS split rules reference invalid node, skipping DNS split rules")
		} else {
			for _, dnsRule := range dnsRules {
				data.dnsRules = append(data.dnsRules, dnsRule)
			}
			if converterIndex > 0 {
				rewriteDetours(dnsServerDetours, converterIndex, bridgeTag)
			}
			data.dnsServerDetours = dnsServerDetours
		}
	}

	return data
}

// excludeIndex returns the index slice with target removed (returns unchanged when target <= 0).
func excludeIndex(indexes []int, target int) []int {
	if target <= 0 {
		return indexes
	}
	out := make([]int, 0, len(indexes))
	for _, i := range indexes {
		if i != target {
			out = append(out, i)
		}
	}
	return out
}

// rewriteSplitOutbound rewrites a route rule whose outbound targets the
// converter node (split-node-N) to the bridge outbound tag (converter-out for
// shape 2, proxy for shape 1).
func rewriteSplitOutbound(rule map[string]interface{}, converterIndex int, bridgeTag string) {
	if rule["outbound"] == fmt.Sprintf("split-node-%d", converterIndex) {
		rule["outbound"] = bridgeTag
	}
}

// rewriteDetours rewrites detour targets in the DNS detour map that point at
// the converter node to the bridge outbound tag.
func rewriteDetours(detours map[string]string, converterIndex int, bridgeTag string) {
	target := fmt.Sprintf("split-node-%d", converterIndex)
	for k, v := range detours {
		if v == target {
			detours[k] = bridgeTag
		}
	}
}

func (s SingBox) ensureSplitOutbounds(indexes []int, added map[int]struct{}, outbounds *[]interface{}) bool {
	for _, index := range uniqueIndexes(indexes) {
		if _, ok := added[index]; ok {
			continue
		}

		node := manage.Manager.GetNode(index)
		if node == nil || node.Protocol == nil {
			log.Warnf("split node does not exist or its protocol is nil: %d", index)
			return false
		}

		outbound := s.outboundForProtocol(node.Protocol, fmt.Sprintf("split-node-%d", index))
		if outbound == nil {
			log.Warnf("split node's protocol has no sing-box outbound support: %d", index)
			return false
		}

		delete(outbound, "detour")
		*outbounds = append(*outbounds, outbound)
		added[index] = struct{}{}
	}

	return true
}

func uniqueIndexes(indexes []int) []int {
	result := make([]int, 0, len(indexes))
	seen := make(map[int]struct{})
	for _, index := range indexes {
		if _, ok := seen[index]; ok {
			continue
		}
		seen[index] = struct{}{}
		result = append(result, index)
	}
	return result
}

// log
// We don't set output; log goes to stdout, and tbox redirects the core's
// stdout/stderr into tbox.core.log (see client.launchCore).
func (s SingBox) logConfig() interface{} {
	return map[string]interface{}{
		"level":     "info",
		"timestamp": true,
	}
}

// inbounds
func (s SingBox) inboundsConfig() interface{} {
	listen := "127.0.0.1"
	if setting.FromLanConn() {
		listen = "0.0.0.0"
	}
	data := []interface{}{
		map[string]interface{}{
			"type":        "mixed",
			"tag":         "mixed-in",
			"listen_port": setting.Socks(),
			"listen":      listen,
		},
	}

	if runtime.GOOS == "linux" && viper.GetBool(key.TunMode) {
		tunInbound := map[string]interface{}{
			"type":           "tun",
			"tag":            "tun-in",
			"interface_name": "tun0",
			"address": []string{
				"172.19.0.1/30",
				"fdfe:dcba:9876::1/126",
			},
			"mtu":          9000,
			"auto_route":   viper.GetBool(key.TunAutoRoute),
			"strict_route": true,
			"stack":        "system",
		}
		if viper.GetBool(key.TunAutoRedirect) {
			tunInbound["auto_redirect"] = true
		}
		data = append(data, tunInbound)
	}

	return data
}

// experimental config
func (s SingBox) experimentalConfig() interface{} {
	return map[string]interface{}{
		"cache_file": map[string]interface{}{
			"enabled": true,
			"path":    filepath.Join(core.GetCoreDir("sing-box"), "cache.db"),
		},
	}
}

// DNS
func (s SingBox) dnsConfig(splitRules []interface{}, serverDetours map[string]string) interface{} {
	servers := make([]interface{}, 0)
	rules := make([]interface{}, 0)

	// local fallback resolver
	servers = append(servers, map[string]interface{}{
		"type": "local",
		"tag":  "local",
	})
	rules = append(rules, splitRules...)

	// domestic-backup config
	if setting.DNSBackup() != "" {
		server := s.buildDNSServer("domestic_2", setting.DNSBackup())
		if d, ok := serverDetours["domestic_2"]; ok && d != "" {
			server["detour"] = d
		}
		servers = append(servers, server)
	}

	// normal domestic config
	if setting.DNSDomestic() != "" {
		server := s.buildDNSServer("domestic_1", setting.DNSDomestic())
		if d, ok := serverDetours["domestic_1"]; ok && d != "" {
			server["detour"] = d
		}
		servers = append(servers, server)
		rules = append(rules, map[string]interface{}{
			"rule_set": []string{
				"geoip-cn",
				"geosite-cn",
			},
			"server": "domestic_1",
		})
	}

	// default rules go through the foreign proxy
	if setting.DNSForeign() != "" {
		foreignServer := s.buildDNSServer("foreign", setting.DNSForeign())
		if d, ok := serverDetours["foreign"]; ok && d != "" {
			foreignServer["detour"] = d
		} else {
			foreignServer["detour"] = "proxy"
		}
		servers = append(servers, foreignServer)
		rules = append(rules, map[string]interface{}{
			"inbound": []string{
				"mixed-in",
			},
			"server": "foreign",
		})
	}

	return map[string]interface{}{
		"servers":  servers,
		"rules":    rules,
		"strategy": "prefer_ipv4",
		// Matches sniff rules and honors the sniffing switch: records
		// domain->IP mapping so the route stage can reverse-lookup domains for IP connections.
		"reverse_mapping": setting.Sniffing(),
	}
}

func (s SingBox) buildDNSServer(tag, address string) map[string]interface{} {
	server := map[string]interface{}{
		"tag": tag,
	}
	rawAddress := strings.TrimSpace(address)
	parsedURL, err := url.Parse(rawAddress)
	if err == nil && parsedURL.Scheme != "" {
		scheme := strings.ToLower(parsedURL.Scheme)
		defaultPort := 53
		switch scheme {
		case "https", "h3":
			defaultPort = 443
		case "tls", "quic":
			defaultPort = 853
		case "tcp":
			defaultPort = 53
		default:
			scheme = "udp"
		}

		host, port := splitDNSHostPort(parsedURL.Host, defaultPort)
		server["type"] = scheme
		server["server"] = host
		server["server_port"] = port
		if (scheme == "https" || scheme == "h3") && parsedURL.Path != "" {
			server["path"] = parsedURL.Path
		}
		return server
	}

	host, port := splitDNSHostPort(rawAddress, 53)
	server["type"] = "udp"
	server["server"] = host
	server["server_port"] = port
	return server
}

func splitDNSHostPort(address string, defaultPort int) (string, int) {
	host := strings.TrimSpace(address)
	if host == "" {
		return host, defaultPort
	}

	if parsedHost, parsedPort, err := net.SplitHostPort(host); err == nil {
		if port, convErr := strconv.Atoi(parsedPort); convErr == nil {
			return strings.Trim(parsedHost, "[]"), port
		}
		return strings.Trim(parsedHost, "[]"), defaultPort
	}

	return strings.Trim(host, "[]"), defaultPort
}

// routing
func (s SingBox) routingConfig(splitRules []interface{}, bridgePort int, direct bool) interface{} {
	rules := make([]interface{}, 0)
	ruleSet := make([]interface{}, 0)

	ruleSet = append(ruleSet, map[string]interface{}{
		"tag":             "geosite-cn",
		"type":            "remote",
		"format":          "binary",
		"url":             rulesetURL("https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs"),
		"download_detour": "direct-out",
	}, map[string]interface{}{
		"tag":             "geoip-cn",
		"type":            "remote",
		"format":          "binary",
		"url":             rulesetURL("https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs"),
		"download_detour": "direct-out",
	})

	s.appendOutboundRules(&rules, routing.TypeBlock, "block")
	s.appendOutboundRules(&rules, routing.TypeDirect, "direct-out")
	s.appendOutboundRules(&rules, routing.TypeProxy, "proxy")

	inboundTags := []string{"mixed-in"}
	if runtime.GOOS == "linux" && viper.GetBool(key.TunMode) {
		inboundTags = append(inboundTags, "tun-in")
	}
	// Matches xray's inbound sniffing and honors the sniffing switch.
	// sniff + DNS reverse_mapping restores domains at the route stage,
	// ensuring domain / domain-suffix / geosite split rules still apply to
	// IP connections after DNS resolution.
	if setting.Sniffing() {
		rules = append([]interface{}{
			map[string]interface{}{
				"inbound": inboundTags,
				"action":  "sniff",
				"sniffer": []string{
					"http",
					"tls",
					"quic",
				},
				"timeout": "1s",
			},
		}, rules...)
	}

	// Bridge mode: the converter's (xray/ssr) outbound traffic to the real
	// server also gets captured by TUN's auto_route back into sing-box; if it
	// then goes via proxy we get a loop (converter -> sing-box -> converter)
	// and a timeout. Direct those processes' traffic out to escape the
	// tunnel. Must come before the proxy fallback rule. Under non-TUN the
	// rule doesn't fire (the converter outbound doesn't traverse sing-box),
	// so it has no side effect.
	if bridgePort > 0 {
		rules = append(rules, map[string]interface{}{
			"process_name": []string{"xray", "ssr"},
			"outbound":     "direct-out",
		})
	}

	rules = append(rules, splitRules...)

	// bypassed traffic all goes direct
	if setting.RoutingBypass() {
		rules = append(rules, map[string]interface{}{
			"rule_set": []string{
				"geoip-cn",
				"geosite-cn",
			},
			"outbound": "direct-out",
		})
	}

	final := "proxy"
	if direct {
		final = "direct-out"
	}
	rules = append(rules, map[string]interface{}{
		"inbound":  inboundTags,
		"outbound": final,
	})
	return map[string]interface{}{
		"rule_set":                ruleSet,
		"rules":                   rules,
		"default_domain_resolver": "local",
		"final":                   final,
		"auto_detect_interface":   true,
	}
}

func (s SingBox) appendOutboundRules(rules *[]interface{}, groupType routing.Type, outbound string) {
	ips, domains := routing.GetRulesGroupData(groupType)
	if len(ips) > 0 {
		*rules = append(*rules, map[string]interface{}{
			"outbound": outbound,
			"ip_cidr":  ips,
		})
	}
	if len(domains) > 0 {
		*rules = append(*rules, map[string]interface{}{
			"outbound":      outbound,
			"domain_suffix": domains,
		})
	}
}

// outbound
func (s SingBox) outboundForProtocol(n protocols.Protocol, tag string) map[string]interface{} {
	var outbound map[string]interface{}
	switch n.GetProtocolMode() {
	case protocols.ModeTrojan:
		t := n.(*protocols.Trojan)
		outbound = s.trojanOutbound(t)
	case protocols.ModeHysteria2:
		t := n.(*protocols.Hysteria2)
		outbound = s.hysteria2Outbound(t)
	case protocols.ModeVMess:
		v := n.(*protocols.VMess)
		outbound = s.vMessOutbound(v)
	case protocols.ModeShadowSocks:
		ss := n.(*protocols.ShadowSocks)
		outbound = s.shadowsocksOutbound(ss)
	case protocols.ModeShadowSocksR:
		ssr := n.(*protocols.ShadowSocksR)
		outbound = s.shadowsocksrOutbound(ssr)
	case protocols.ModeHTTP:
		v := n.(*protocols.HTTP)
		outbound = s.httpOutbound(v)
	case protocols.ModeSocks:
		v := n.(*protocols.Socks)
		outbound = s.socksOutbound(v)
	case protocols.ModeVLESS:
		v := n.(*protocols.VLess)
		outbound = s.vLessOutbound(v)
	case protocols.ModeVMessAEAD:
		v := n.(*protocols.VMessAEAD)
		outbound = s.vMessAEADOutbound(v)
	case protocols.ModeTUIC:
		t := n.(*protocols.TUIC)
		outbound = s.tuicOutbound(t)
	case protocols.ModeAnyTLS:
		a := n.(*protocols.AnyTLS)
		outbound = s.anytlsOutbound(a)
	}
	if outbound != nil {
		outbound["tag"] = tag
	}
	return outbound
}

// outboundConfig generates the sing-box outbound list. Under bridge mode
// (plan.Enabled):
//   - shape 1 (plan.MainIsConverter): the main node is itself the converter
//     node; proxy is directly the socks bridge pointing at 127.0.0.1:bridgePort
//     (chain proxy does not apply here — it is ignored with a warning).
//   - shape 2 (main node is normal, converter serves only split targets):
//     proxy is still the main node's native outbound; an additional socks
//     outbound with tag=converter-out is added so split rules can reference the converter.
//
// Non-bridge mode uses the original logic: main node outbound + chain proxy.
func (s SingBox) outboundConfig(n protocols.Protocol, splitOutbounds []interface{}, plan *BridgePlan, bridgePort int) interface{} {
	out := make([]interface{}, 0)
	bridging := plan != nil && plan.Enabled && bridgePort > 0

	socksBridge := func(tag string) map[string]interface{} {
		return map[string]interface{}{
			"tag":         tag,
			"type":        "socks",
			"server":      "127.0.0.1",
			"server_port": bridgePort,
			"version":     "5",
		}
	}

	if bridging && plan.MainIsConverter {
		// shape 1: proxy is the bridge itself.
		if indexes, err := setting.ChainProxyIndexes(); err == nil && len(indexes) > 0 {
			log.Warn("under bridge mode the current node is served by the converter; chain proxy has no effect and was ignored")
		}
		out = append(out, socksBridge("proxy"))
	} else {
		// main node's native outbound (plus chain proxy).
		proxyOutbound := s.outboundForProtocol(n, "proxy")
		if proxyOutbound != nil {
			chainOutbounds, detourTag := s.chainOutbounds()
			if detourTag != "" {
				proxyOutbound["detour"] = detourTag
			}
			out = append(out, proxyOutbound)
			out = append(out, chainOutbounds...)
		}
		// shape 2: additionally attach a converter-out for split targets to reference.
		if bridging {
			out = append(out, socksBridge(converterTag))
		}
	}
	out = append(out, splitOutbounds...)

	out = append(out, map[string]interface{}{
		"tag":  "direct-out",
		"type": "direct",
	})
	out = append(out, map[string]interface{}{
		"tag":  "block",
		"type": "block",
	})
	return out
}

func (s SingBox) chainOutbounds() ([]interface{}, string) {
	indexes, err := setting.ChainProxyIndexes()
	if err != nil {
		log.Warn(err)
		return nil, ""
	}
	if len(indexes) == 0 {
		return nil, ""
	}

	runtimeIndex := manage.Manager.SelectedIndex()
	for _, index := range indexes {
		if index == runtimeIndex {
			log.Warnf("chain proxy must not include the currently running node: %d", index)
			return nil, ""
		}
	}

	chainOutbounds := make([]interface{}, 0, len(indexes))
	tags := make([]string, 0, len(indexes))
	for _, index := range indexes {
		node := manage.Manager.GetNode(index)
		if node == nil || node.Protocol == nil {
			log.Warnf("chain proxy node does not exist or protocol is nil: %d", index)
			return nil, ""
		}
		if nodeUsesXhttp(node.Protocol) {
			log.Warnf("chain proxy does not support xhttp-transport nodes (only xray supports xhttp, cannot chain via sing-box): %d", index)
			return nil, ""
		}
		if node.Protocol.GetProtocolMode() == protocols.ModeShadowSocksR {
			log.Warnf("chain proxy does not support SSR nodes (newer sing-box has removed SSR outbound; served by the external ssr converter): %d", index)
			return nil, ""
		}
		tag := fmt.Sprintf("chain-node-%d", index)
		outbound := s.outboundForProtocol(node.Protocol, tag)
		if outbound == nil {
			log.Warnf("chain proxy node's protocol has no sing-box outbound support: %d", index)
			return nil, ""
		}
		chainOutbounds = append(chainOutbounds, outbound)
		tags = append(tags, tag)
	}

	for i := len(chainOutbounds) - 1; i > 0; i-- {
		chainOutbounds[i].(map[string]interface{})["detour"] = tags[i-1]
	}

	return chainOutbounds, tags[len(tags)-1]
}

// Shadowsocks
func (s SingBox) shadowsocksOutbound(ss *protocols.ShadowSocks) map[string]interface{} {
	return map[string]interface{}{
		"tag":         "proxy",
		"type":        "shadowsocks",
		"server":      ss.Address,
		"server_port": ss.Port,
		"method":      ss.Method,
		"password":    ss.Password,
	}
}

// Shadowsocksr
func (s SingBox) shadowsocksrOutbound(ssr *protocols.ShadowSocksR) map[string]interface{} {
	return map[string]interface{}{
		"tag":            "proxy",
		"type":           "shadowsocksr",
		"server":         ssr.Address,
		"server_port":    ssr.Port,
		"method":         ssr.Method,
		"password":       ssr.Password,
		"obfs":           ssr.Obfs,
		"obfs_param":     ssr.ObfsParam,
		"protocol":       ssr.Protocol,
		"protocol_param": ssr.ProtoParam,
	}
}

// Trojan
func (s SingBox) trojanOutbound(trojan *protocols.Trojan) map[string]interface{} {
	queryParams := trojan.Values
	outbound := map[string]interface{}{
		"tag":         "proxy",
		"type":        "trojan",
		"server":      trojan.Address,
		"server_port": trojan.Port,
		"password":    trojan.Password,
	}

	// tls: the trojan protocol uses TLS by default; standard share links
	// usually omit the security parameter. Only an explicit security=none
	// disables it. Previously we only accepted security=tls, which caused
	// almost every trojan link's generated config to miss the tls block and
	// fail to connect.
	if queryParams.Get("security") != "none" {
		// If SNI is empty, fall back to the server address, matching other clients.
		sni := trojan.Sni()
		if sni == "" {
			sni = trojan.Address
		}
		outbound["tls"] = buildTLS(tlsOptions{
			ServerName: sni,
			Alpn:       queryParams.Get("alpn"),
			Insecure:   resolveInsecure(queryParams),
		})
	}

	// multiplex
	if setting.Mux() {
		outbound["multiplex"] = map[string]interface{}{
			"enabled": setting.Mux(),
		}
	}

	// transport
	if queryParams.Get("type") == "ws" {
		if transport := buildStreamTransport(transportOptions{
			Network: "ws",
			Path:    queryParams.Get("path"),
			Host:    queryParams.Get("host"),
		}); transport != nil {
			outbound["transport"] = transport
		}
	}

	return outbound
}

// TUIC
func (s SingBox) tuicOutbound(tuic *protocols.TUIC) map[string]interface{} {
	queryParams := tuic.Values
	outbound := map[string]interface{}{
		"tag":         "proxy",
		"type":        "tuic",
		"server":      tuic.Address,
		"server_port": tuic.Port,
		"uuid":        tuic.UUID,
		"password":    tuic.Password,
	}

	if queryParams.Has("congestion_control") && queryParams.Get("congestion_control") != "" {
		outbound["congestion_control"] = queryParams.Get("congestion_control")
	}
	if queryParams.Has("udp_relay_mode") && queryParams.Get("udp_relay_mode") != "" {
		outbound["udp_relay_mode"] = queryParams.Get("udp_relay_mode")
	}

	outbound["tls"] = buildTLS(tlsOptions{
		ServerName: tuic.Sni(),
		Alpn:       queryParams.Get("alpn"),
		Insecure:   resolveInsecure(queryParams),
	})

	return outbound
}

// AnyTLS
func (s SingBox) anytlsOutbound(anytls *protocols.AnyTLS) map[string]interface{} {
	queryParams := anytls.Values
	outbound := map[string]interface{}{
		"tag":         "proxy",
		"type":        "anytls",
		"server":      anytls.Address,
		"server_port": anytls.Port,
		"password":    anytls.Password,
	}

	outbound["tls"] = buildTLS(tlsOptions{
		ServerName: anytls.Sni(),
		Alpn:       queryParams.Get("alpn"),
		Insecure:   resolveInsecure(queryParams),
	})

	return outbound
}

// Hysteria2
func (s SingBox) hysteria2Outbound(hysteria2 *protocols.Hysteria2) map[string]interface{} {
	outbound := map[string]interface{}{
		"tag":         "proxy",
		"type":        "hysteria2",
		"server":      hysteria2.Address,
		"server_port": hysteria2.Port,
		"password":    hysteria2.Password,
	}

	// obfs (salamander). Only enabled when the link has obfs.
	if obfsType := strings.TrimSpace(hysteria2.Get("obfs")); obfsType != "" {
		obfs := map[string]interface{}{
			"type": obfsType,
		}
		if pwd := hysteria2.Get("obfs-password"); pwd != "" {
			obfs["password"] = pwd
		}
		outbound["obfs"] = obfs
	}

	// tls. alpn defaults to h3 but can be overridden by the link's alpn.
	alpn := "h3"
	if v := strings.TrimSpace(hysteria2.Get("alpn")); v != "" {
		alpn = v
	}
	outbound["tls"] = buildTLS(tlsOptions{
		ServerName: hysteria2.Sni(),
		Alpn:       alpn,
		// Accepts insecure / allowInsecure / skip-cert-verify naming;
		// defaults to the global allow_insecure setting.
		Insecure: resolveInsecure(hysteria2.Values),
	})

	return outbound
}

// VMess
func (s SingBox) vMessOutbound(vmess *protocols.VMess) map[string]interface{} {
	outbound := map[string]interface{}{
		"tag":         "proxy",
		"type":        "vmess",
		"server":      vmess.Add,
		"server_port": vmess.Port,
		"uuid":        vmess.Id,
		"security":    vmess.Scy,
		"alter_id":    vmess.Aid,
	}

	// tls
	if vmess.Tls == "tls" {
		outbound["tls"] = buildTLS(tlsOptions{
			ServerName: vmess.Sni,
			Alpn:       vmess.Alpn,
			Insecure:   setting.AllowInsecure(),
		})
	}

	// multiplex
	if setting.Mux() {
		outbound["multiplex"] = map[string]interface{}{
			"enabled": setting.Mux(),
		}
	}

	// transport
	if transport := buildStreamTransport(transportOptions{
		Network:     vmess.Net,
		Path:        vmess.Path,
		Host:        vmess.Host,
		ServiceName: vmess.Path,
	}); transport != nil {
		outbound["transport"] = transport
	}

	return outbound
}

// socks
func (s SingBox) socksOutbound(socks *protocols.Socks) map[string]interface{} {
	outbound := map[string]interface{}{
		"tag":         "proxy",
		"type":        "socks",
		"server":      socks.Address,
		"server_port": socks.Port,
		"version":     "5",
	}
	if socks.Username != "" {
		outbound["username"] = socks.Username
	}
	if socks.Password != "" {
		outbound["password"] = socks.Password
	}
	return outbound
}

func (s SingBox) httpOutbound(http *protocols.HTTP) map[string]interface{} {
	outbound := map[string]interface{}{
		"tag":         "proxy",
		"type":        "http",
		"server":      http.Address,
		"server_port": http.Port,
	}
	if http.Username != "" {
		outbound["username"] = http.Username
	}
	if http.Password != "" {
		outbound["password"] = http.Password
	}
	return outbound
}

// VLESS
func (s SingBox) vLessOutbound(vless *protocols.VLess) map[string]interface{} {
	security := vless.GetValue(field.Security)
	network := vless.GetValue(field.NetworkType)

	outbound := map[string]interface{}{
		"tag":         "proxy",
		"type":        "vless",
		"server":      vless.Address,
		"server_port": vless.Port,
		"uuid":        vless.ID,
	}

	flow := vless.GetValue(field.Flow)
	if flow != "" {
		outbound["flow"] = flow
	}

	if security == "tls" || security == "xtls" || security == "reality" {
		// insecure first reads the link's insecure / allowInsecure params and
		// then ORs in the global setting. Previously we only consulted the
		// global setting, ignoring explicit insecure=1 in the link, which
		// caused TLS handshake failures for self-signed / mismatched-cert nodes.
		insecure := resolveInsecure(vless.Values)
		outbound["tls"] = buildTLS(tlsOptions{
			ServerName:  vless.GetHostValue(field.SNI),
			Alpn:        vless.GetValue(field.Alpn),
			Insecure:    insecure,
			Reality:     security == "reality",
			PublicKey:   vless.GetValue(field.PublicKey),
			ShortID:     vless.GetValue(field.ShortId),
			Fingerprint: vless.GetValue(field.FingerPrint),
		})
	}

	switch network {
	case "tcp":
		headerType := vless.GetValue(field.TCPHeaderType)
		if headerType != "" && headerType != "none" {
			outbound["transport"] = map[string]interface{}{
				"type": "tcp",
				"header": map[string]interface{}{
					"type": headerType,
				},
			}
		}
	case "h2":
		outbound["transport"] = buildStreamTransport(transportOptions{
			Network: "h2",
			Path:    vless.GetValue(field.H2Path),
			Host:    vless.GetHostValue(field.H2Host),
		})
	case "ws":
		outbound["transport"] = buildStreamTransport(transportOptions{
			Network: "ws",
			Path:    vless.GetValue(field.WsPath),
			Host:    vless.GetValue(field.WsHost),
		})
	case "grpc":
		outbound["transport"] = buildStreamTransport(transportOptions{
			Network:     "grpc",
			ServiceName: vless.GetValue(field.GrpcServiceName),
		})
	}

	if setting.Mux() {
		outbound["multiplex"] = map[string]interface{}{
			"enabled": true,
		}
	}

	return outbound
}

// VMessAEAD
// Generates a flat sing-box vmess outbound config. Previously this function
// mistakenly emitted the xray format (protocol/settings.vnext/streamSettings),
// which sing-box could not parse and led to node startup failures.
func (s SingBox) vMessAEADOutbound(vmess *protocols.VMessAEAD) map[string]interface{} {
	security := vmess.GetValue(field.VMessEncryption)
	if security == "" {
		security = "auto"
	}
	outbound := map[string]interface{}{
		"tag":         "proxy",
		"type":        "vmess",
		"server":      vmess.Address,
		"server_port": vmess.Port,
		"uuid":        vmess.ID,
		"security":    security,
		"alter_id":    0,
	}

	mux := setting.Mux()
	switch vmess.GetValue(field.Security) {
	case "tls":
		outbound["tls"] = buildTLS(tlsOptions{
			ServerName: vmess.GetHostValue(field.SNI),
			Alpn:       vmess.GetValue(field.Alpn),
			Insecure:   setting.AllowInsecure(),
		})
	case "reality":
		outbound["tls"] = buildTLS(tlsOptions{
			ServerName:  vmess.GetHostValue(field.SNI),
			Alpn:        vmess.GetValue(field.Alpn),
			Insecure:    setting.AllowInsecure(),
			Reality:     true,
			PublicKey:   vmess.GetValue(field.PublicKey),
			ShortID:     vmess.GetValue(field.ShortId),
			Fingerprint: vmess.GetValue(field.FingerPrint),
		})
		mux = false
	}

	network := vmess.GetValue(field.NetworkType)
	path := vmess.GetValue(field.WsPath)
	host := vmess.GetValue(field.WsHost)
	if network == "h2" {
		path = vmess.GetValue(field.H2Path)
		host = vmess.GetHostValue(field.H2Host)
		mux = false
	}
	if transport := buildStreamTransport(transportOptions{
		Network:     network,
		Path:        path,
		Host:        host,
		ServiceName: vmess.GetValue(field.GrpcServiceName),
	}); transport != nil {
		outbound["transport"] = transport
	}

	if mux {
		outbound["multiplex"] = map[string]interface{}{
			"enabled": true,
		}
	}

	return outbound
}
