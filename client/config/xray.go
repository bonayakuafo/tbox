package config

import (
	"tbox/core"
	"tbox/core/protocols"
	"tbox/core/protocols/field"
	"tbox/core/routing"
	"tbox/core/setting"
	"tbox/log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// dnsBareHost strips the scheme prefix (tcp://, https://, etc.) and port from
// a DNS address, returning the bare host. xray's dokodemo-door address and
// DNS server address only accept bare host/IP.
func dnsBareHost(addr string) string {
	s := strings.TrimSpace(addr)
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	// strip any DoH path
	if i := strings.Index(s, "/"); i >= 0 {
		s = s[:i]
	}
	// strip the port (handling IPv6 [::]:port)
	if host, _, err := net.SplitHostPort(s); err == nil {
		s = host
	}
	return strings.Trim(s, "[]")
}

// xrayAllowInsecure returns the value for xray tlsSettings.allowInsecure.
// From xray 26.x onwards allowInsecure=true has been removed (migrated to
// pinnedPeerCertSha256, which requires a cert hash and can't be derived from
// a share link automatically); allowInsecure:true in the config causes a hard
// load failure. So we always return false (verify certs). If the user's
// global setting or the link requested skipping verification, log a warning
// and suggest switching to the sing-box core.
func xrayAllowInsecure(want bool) bool {
	if want {
		log.Warn("the xray core has removed allowInsecure; connecting in secure (verify-cert) mode. If this node needs to skip certificate verification (e.g. self-signed cert), switch to the sing-box core")
	}
	return false
}

// dnsRouteIP returns the bare IP only when the DNS address parses to a valid
// IP; otherwise returns an empty string. Used for the ip field in xray route
// rules (which only accepts IP/CIDR, not domains or schemes).
func dnsRouteIP(addr string) string {
	host := dnsBareHost(addr)
	if net.ParseIP(host) != nil {
		return host
	}
	return ""
}

type Xray struct{}

// GenConfig generates the core config file.
func (x Xray) GenConfig(node protocols.Protocol) (string, error) {
	coreDir := core.GetCoreDir("xray")
	if err := os.MkdirAll(coreDir, 0755); err != nil {
		log.Error(err)
		return "", err
	}
	path := filepath.Join(coreDir, "config.json")
	var conf = map[string]interface{}{
		"log":       x.logConfig(),
		"inbounds":  x.InboundsConfig(),
		"outbounds": x.OutboundConfig(node),
		"policy":    x.policyConfig(),
		"dns":       x.dnsConfig(),
		"routing":   x.routingConfig(),
	}
	err := core.WriteJSON(conf, path)
	if err != nil {
		log.Error(err)
		return "", err
	}
	return path, nil
}

// GenBridgeConfig generates a minimal xray config for bridge mode: xray
// becomes a pure protocol converter. Only a socks inbound listening on
// 127.0.0.1:bridgePort and the proxy outbound corresponding to the node
// (plus freedom/block fallbacks). No TUN / routing / DNS / split — those all
// stay in sing-box, whose proxy outbound hands traffic over via an internal
// socks so xhttp etc. get converted here.
func (x Xray) GenBridgeConfig(node protocols.Protocol, bridgePort int) (string, error) {
	coreDir := core.GetCoreDir("xray")
	if err := os.MkdirAll(coreDir, 0755); err != nil {
		log.Error(err)
		return "", err
	}
	path := filepath.Join(coreDir, "config.json")
	inbounds := []interface{}{
		map[string]interface{}{
			"tag":      "socks-in",
			"listen":   "127.0.0.1",
			"port":     bridgePort,
			"protocol": "socks",
			"settings": map[string]interface{}{
				"auth": "noauth",
				"udp":  setting.UDP(),
			},
		},
	}
	var conf = map[string]interface{}{
		"log":       x.logConfig(),
		"inbounds":  inbounds,
		"outbounds": x.OutboundConfig(node),
	}
	if err := core.WriteJSON(conf, path); err != nil {
		log.Error(err)
		return "", err
	}
	return path, nil
}

// log
// We don't set access/error log file paths; log goes to stdout/stderr, and
// tbox redirects the core's stdout/stderr into tbox.core.log (see
// client.launchCore). At warning level, per-connection access logs aren't
// produced, keeping the log file from ballooning.
func (x Xray) logConfig() interface{} {
	return map[string]string{
		"loglevel": "warning",
	}
}

// inbounds
func (x Xray) InboundsConfig() interface{} {
	listen := "127.0.0.1"
	if setting.FromLanConn() {
		listen = "0.0.0.0"
	}
	data := []interface{}{
		map[string]interface{}{
			"tag":      "proxy",
			"port":     setting.Socks(),
			"listen":   listen,
			"protocol": "socks",
			"sniffing": map[string]interface{}{
				"enabled": setting.Sniffing(),
				"destOverride": []string{
					"http",
					"tls",
				},
			},
			"settings": map[string]interface{}{
				"auth":      "noauth",
				"udp":       setting.UDP(),
				"userLevel": 0,
			},
		},
	}
	if setting.Http() > 0 {
		data = append(data, map[string]interface{}{
			"tag":      "http",
			"port":     setting.Http(),
			"listen":   listen,
			"protocol": "http",
			"settings": map[string]interface{}{
				"userLevel": 0,
			},
		})
	}
	if setting.DNSPort() > 0 {
		data = append(data, map[string]interface{}{
			"tag":      "dns-in",
			"port":     setting.DNSPort(),
			"listen":   listen,
			"protocol": "dokodemo-door",
			"settings": map[string]interface{}{
				"userLevel": 0,
				"address":   dnsBareHost(setting.DNSForeign()),
				"network":   "tcp,udp",
				"port":      53,
			},
		})
	}
	return data
}

// local policy
func (x Xray) policyConfig() interface{} {
	return map[string]interface{}{
		"levels": map[string]interface{}{
			"0": map[string]interface{}{
				"handshake":    4,
				"connIdle":     300,
				"uplinkOnly":   1,
				"downlinkOnly": 1,
				"bufferSize":   10240,
			},
		},
		"system": map[string]interface{}{
			"statsInboundUplink":   true,
			"statsInboundDownlink": true,
		},
	}
}

// DNS
func (x Xray) dnsConfig() interface{} {
	servers := make([]interface{}, 0)
	if setting.DNSDomestic() != "" {
		servers = append(servers, map[string]interface{}{
			"address": dnsBareHost(setting.DNSDomestic()),
			"port":    53,
			"domains": []interface{}{
				"geosite:cn",
			},
			"expectIPs": []interface{}{
				"geoip:cn",
			},
		})
	}
	if setting.DNSBackup() != "" {
		servers = append(servers, map[string]interface{}{
			"address": dnsBareHost(setting.DNSBackup()),
			"port":    53,
			"domains": []interface{}{
				"geosite:cn",
			},
			"expectIPs": []interface{}{
				"geoip:cn",
			},
		})
	}
	if setting.DNSForeign() != "" {
		servers = append(servers, map[string]interface{}{
			"address": dnsBareHost(setting.DNSForeign()),
			"port":    53,
			"domains": []interface{}{
				"geosite:geolocation-!cn",
				"geosite:speedtest",
			},
		})
	}
	return map[string]interface{}{
		"hosts": map[string]interface{}{
			"domain:googleapis.cn": "googleapis.com",
		},
		"servers": servers,
	}
}

// routing
func (x Xray) routingConfig() interface{} {
	rules := make([]interface{}, 0)
	if setting.DNSPort() != 0 {
		rules = append(rules, map[string]interface{}{
			"type": "field",
			"inboundTag": []interface{}{
				"dns-in",
			},
			"outboundTag": "dns-out",
		})
	}
	if foreignIP := dnsRouteIP(setting.DNSForeign()); foreignIP != "" {
		rules = append(rules, map[string]interface{}{
			"type":        "field",
			"port":        53,
			"outboundTag": "proxy",
			"ip": []string{
				foreignIP,
			},
		})
	}
	{
		var ip []string
		if v := dnsRouteIP(setting.DNSDomestic()); v != "" {
			ip = append(ip, v)
		}
		if v := dnsRouteIP(setting.DNSBackup()); v != "" {
			ip = append(ip, v)
		}
		if len(ip) > 0 {
			rules = append(rules, map[string]interface{}{
				"type":        "field",
				"port":        53,
				"outboundTag": "direct",
				"ip":          ip,
			})
		}
	}
	ips, domains := routing.GetRulesGroupData(routing.TypeBlock)
	if len(ips) != 0 {
		rules = append(rules, map[string]interface{}{
			"type":        "field",
			"outboundTag": "block",
			"ip":          ips,
		})
	}
	if len(domains) != 0 {
		rules = append(rules, map[string]interface{}{
			"type":        "field",
			"outboundTag": "block",
			"domain":      domains,
		})
	}
	ips, domains = routing.GetRulesGroupData(routing.TypeDirect)
	if len(ips) != 0 {
		rules = append(rules, map[string]interface{}{
			"type":        "field",
			"outboundTag": "direct",
			"ip":          ips,
		})
	}
	if len(domains) != 0 {
		rules = append(rules, map[string]interface{}{
			"type":        "field",
			"outboundTag": "direct",
			"domain":      domains,
		})
	}
	ips, domains = routing.GetRulesGroupData(routing.TypeProxy)
	if len(ips) != 0 {
		rules = append(rules, map[string]interface{}{
			"type":        "field",
			"outboundTag": "proxy",
			"ip":          ips,
		})
	}
	if len(domains) != 0 {
		rules = append(rules, map[string]interface{}{
			"type":        "field",
			"outboundTag": "proxy",
			"domain":      domains,
		})
	}

	if setting.RoutingBypass() {
		rules = append(rules, map[string]interface{}{
			"type":        "field",
			"outboundTag": "direct",
			"ip": []string{
				"geoip:private",
				"geoip:cn",
			},
		})
		rules = append(rules, map[string]interface{}{
			"type":        "field",
			"outboundTag": "direct",
			"domain": []string{
				"geosite:cn",
			},
		})
	}
	return map[string]interface{}{
		"domainStrategy": setting.RoutingStrategy(),
		"rules":          rules,
	}
}

// outbound
func (x Xray) OutboundConfig(n protocols.Protocol) interface{} {
	out := make([]interface{}, 0)
	switch n.GetProtocolMode() {
	case protocols.ModeTrojan:
		t := n.(*protocols.Trojan)
		out = append(out, x.trojanOutbound(t))
	case protocols.ModeShadowSocks:
		ss := n.(*protocols.ShadowSocks)
		out = append(out, x.shadowsocksOutbound(ss))
	case protocols.ModeVMess:
		v := n.(*protocols.VMess)
		out = append(out, x.vMessOutbound(v))
	case protocols.ModeSocks:
		v := n.(*protocols.Socks)
		out = append(out, x.socksOutbound(v))
	case protocols.ModeVLESS:
		v := n.(*protocols.VLess)
		out = append(out, x.vLessOutbound(v))
	case protocols.ModeVMessAEAD:
		v := n.(*protocols.VMessAEAD)
		out = append(out, x.vMessAEADOutbound(v))
	case protocols.ModeTUIC, protocols.ModeAnyTLS:
		return out
	}
	out = append(out, map[string]interface{}{
		"tag":      "direct",
		"protocol": "freedom",
		"settings": map[string]interface{}{},
	})
	out = append(out, map[string]interface{}{
		"tag":      "block",
		"protocol": "blackhole",
		"settings": map[string]interface{}{
			"response": map[string]interface{}{
				"type": "http",
			},
		},
	})
	out = append(out, map[string]interface{}{
		"tag":      "dns-out",
		"protocol": "dns",
	})
	return out
}

// Shadowsocks
func (x Xray) shadowsocksOutbound(ss *protocols.ShadowSocks) interface{} {
	return map[string]interface{}{
		"tag":      "proxy",
		"protocol": "shadowsocks",
		"settings": map[string]interface{}{
			"servers": []interface{}{
				map[string]interface{}{
					"address":  ss.Address,
					"port":     ss.Port,
					"password": ss.Password,
					"method":   ss.Method,
					"level":    0,
				},
			},
		},
		"streamSettings": map[string]interface{}{
			"network": "tcp",
		},
	}
}

// Trojan
func (x Xray) trojanOutbound(trojan *protocols.Trojan) interface{} {
	streamSettings := map[string]interface{}{
		"network":  "tcp",
		"security": "tls",
	}
	if trojan.Sni() != "" {
		streamSettings["tlsSettings"] = map[string]interface{}{
			"allowInsecure": xrayAllowInsecure(setting.AllowInsecure()),
			"serverName":    trojan.Sni(),
		}
	}
	return map[string]interface{}{
		"tag":      "proxy",
		"protocol": "trojan",
		"settings": map[string]interface{}{
			"servers": []interface{}{
				map[string]interface{}{
					"address":  trojan.Address,
					"port":     trojan.Port,
					"password": trojan.Password,
					"level":    0,
				},
			},
		},
		"streamSettings": streamSettings,
	}
}

// VMess
func (x Xray) vMessOutbound(vmess *protocols.VMess) interface{} {
	mux := setting.Mux()
	streamSettings := map[string]interface{}{
		"network":  vmess.Net,
		"security": vmess.Tls,
	}
	if vmess.Tls == "tls" {
		tlsSettings := map[string]interface{}{
			"allowInsecure": xrayAllowInsecure(setting.AllowInsecure()),
		}
		if vmess.Sni != "" {
			tlsSettings["serverName"] = vmess.Sni
		}
		if vmess.Alpn != "" {
			tlsSettings["alpn"] = strings.Split(vmess.Alpn, ",")
		}
		streamSettings["tlsSettings"] = tlsSettings
	}
	switch vmess.Net {
	case "tcp":
		streamSettings["tcpSettings"] = map[string]interface{}{
			"header": map[string]interface{}{
				"type": vmess.Type,
			},
		}
	case "kcp":
		kcpSettings := map[string]interface{}{
			"mtu":              1350,
			"tti":              50,
			"uplinkCapacity":   12,
			"downlinkCapacity": 100,
			"congestion":       false,
			"readBufferSize":   2,
			"writeBufferSize":  2,
			"header": map[string]interface{}{
				"type": vmess.Type,
			},
		}
		if vmess.Type != "none" {
			kcpSettings["seed"] = vmess.Path
		}
		streamSettings["kcpSettings"] = kcpSettings
	case "ws":
		streamSettings["wsSettings"] = map[string]interface{}{
			"path": vmess.Path,
			"headers": map[string]interface{}{
				"Host": vmess.Host,
			},
		}
	case "h2":
		mux = false
		host := make([]string, 0)
		for _, line := range strings.Split(vmess.Host, ",") {
			line = strings.TrimSpace(line)
			if line != "" {
				host = append(host, line)
			}
		}
		streamSettings["httpSettings"] = map[string]interface{}{
			"path": vmess.Path,
			"host": host,
		}
	case "quic":
		quicSettings := map[string]interface{}{
			"security": vmess.Host,
			"header": map[string]interface{}{
				"type": vmess.Type,
			},
		}
		if vmess.Host != "none" {
			quicSettings["key"] = vmess.Path
		}
		streamSettings["quicSettings"] = quicSettings
	case "grpc":
		streamSettings["grpcSettings"] = map[string]interface{}{
			"serviceName": vmess.Path,
			"multiMode":   vmess.Type == "multi",
		}
	case "xhttp":
		streamSettings["xhttpSettings"] = xhttpSettings(vmess.Path, vmess.Host, "auto")
	}
	return map[string]interface{}{
		"tag":      "proxy",
		"protocol": "vmess",
		"settings": map[string]interface{}{
			"vnext": []interface{}{
				map[string]interface{}{
					"address": vmess.Add,
					"port":    vmess.Port,
					"users": []interface{}{
						map[string]interface{}{
							"id":       vmess.Id,
							"alterId":  vmess.Aid,
							"security": vmess.Scy,
							"level":    0,
						},
					},
				},
			},
		},
		"streamSettings": streamSettings,
		"mux": map[string]interface{}{
			"enabled": mux,
		},
	}
}

// socks
func (x Xray) socksOutbound(socks *protocols.Socks) interface{} {
	user := map[string]interface{}{
		"address": socks.Address,
		"port":    socks.Port,
	}
	if socks.Username != "" || socks.Password != "" {
		user["users"] = []interface{}{
			map[string]interface{}{
				"user": socks.Username,
				"pass": socks.Password,
			},
		}
	}
	return map[string]interface{}{
		"tag":      "proxy",
		"protocol": "socks",
		"settings": map[string]interface{}{
			"servers": []interface{}{
				user,
			},
		},
		"streamSettings": map[string]interface{}{
			"network": "tcp",
			"tcpSettings": map[string]interface{}{
				"header": map[string]interface{}{
					"type": "none",
				},
			},
		},
		"mux": map[string]interface{}{
			"enabled": false,
		},
	}
}

// VLESS
func (x Xray) vLessOutbound(vless *protocols.VLess) interface{} {
	mux := setting.Mux()
	security := vless.GetValue(field.Security)
	network := vless.GetValue(field.NetworkType)
	user := map[string]interface{}{
		"id":         vless.ID,
		"flow":       vless.GetValue(field.Flow),
		"encryption": vless.GetValue(field.VLessEncryption),
		"level":      0,
	}
	streamSettings := map[string]interface{}{
		"network":  network,
		"security": security,
	}
	switch security {
	case "tls":
		tlsSettings := map[string]interface{}{
			"allowInsecure": xrayAllowInsecure(resolveInsecure(vless.Values)),
		}
		sni := vless.GetHostValue(field.SNI)
		alpn := vless.GetValue(field.Alpn)
		if sni != "" {
			tlsSettings["serverName"] = sni
		}
		if alpn != "" {
			tlsSettings["alpn"] = strings.Split(alpn, ",")
		}
		if fp := vless.GetValue(field.FingerPrint); fp != "" {
			tlsSettings["fingerprint"] = fp
		}
		streamSettings["tlsSettings"] = tlsSettings
	case "xtls":
		xtlsSettings := map[string]interface{}{
			"allowInsecure": xrayAllowInsecure(setting.AllowInsecure()),
		}
		sni := vless.GetHostValue(field.SNI)
		alpn := vless.GetValue(field.Alpn)
		if sni != "" {
			xtlsSettings["serverName"] = sni
		}
		if alpn != "" {
			xtlsSettings["alpn"] = strings.Split(alpn, ",")
		}
		streamSettings["xtlsSettings"] = xtlsSettings
		mux = false
	case "reality":
		realitySettings := map[string]interface{}{
			"show":        false,
			"fingerprint": vless.GetValue(field.FingerPrint),
			"serverName":  vless.GetHostValue(field.SNI),
			"publicKey":   vless.GetValue(field.PublicKey),
			"shortId":     vless.GetValue(field.ShortId),
			"spiderX":     vless.GetValue(field.SpiderX),
		}
		streamSettings["realitySettings"] = realitySettings
		mux = false
	}
	switch network {
	case "tcp":
		streamSettings["tcpSettings"] = map[string]interface{}{
			"header": map[string]interface{}{
				"type": vless.GetValue(field.TCPHeaderType),
			},
		}
	case "kcp":
		kcpSettings := map[string]interface{}{
			"mtu":              1350,
			"tti":              50,
			"uplinkCapacity":   12,
			"downlinkCapacity": 100,
			"congestion":       false,
			"readBufferSize":   2,
			"writeBufferSize":  2,
			"header": map[string]interface{}{
				"type": vless.GetValue(field.MkcpHeaderType),
			},
		}
		if vless.Has(field.Seed.Key) {
			kcpSettings["seed"] = vless.GetValue(field.Seed)
		}
		streamSettings["kcpSettings"] = kcpSettings
	case "h2":
		mux = false
		host := make([]string, 0)
		for _, line := range strings.Split(vless.GetHostValue(field.H2Host), ",") {
			line = strings.TrimSpace(line)
			if line != "" {
				host = append(host, line)
			}
		}
		streamSettings["httpSettings"] = map[string]interface{}{
			"path": vless.GetValue(field.H2Path),
			"host": host,
		}
	case "ws":
		streamSettings["wsSettings"] = map[string]interface{}{
			"path": vless.GetValue(field.WsPath),
			"headers": map[string]interface{}{
				"Host": vless.GetValue(field.WsHost),
			},
		}
	case "quic":
		quicSettings := map[string]interface{}{
			"security": vless.GetValue(field.QuicSecurity),
			"header": map[string]interface{}{
				"type": vless.GetValue(field.QuicHeaderType),
			},
		}
		if vless.GetValue(field.QuicSecurity) != "none" {
			quicSettings["key"] = vless.GetValue(field.QuicKey)
		}
		streamSettings["quicSettings"] = quicSettings
	case "grpc":
		streamSettings["grpcSettings"] = map[string]interface{}{
			"serviceName": vless.GetValue(field.GrpcServiceName),
			"multiMode":   vless.GetValue(field.GrpcMode) == "multi",
		}
	case "xhttp":
		streamSettings["xhttpSettings"] = xhttpSettings(
			vless.GetValue(field.WsPath),
			vless.GetValue(field.WsHost),
			vless.GetValue(field.XhttpMode),
		)
	}
	return map[string]interface{}{
		"tag":      "proxy",
		"protocol": "vless",
		"settings": map[string]interface{}{
			"vnext": []interface{}{
				map[string]interface{}{
					"address": vless.Address,
					"port":    vless.Port,
					"users": []interface{}{
						user,
					},
				},
			},
		},
		"streamSettings": streamSettings,
		"mux": map[string]interface{}{
			"enabled": mux,
		},
	}
}

// VMessAEAD
func (x Xray) vMessAEADOutbound(vmess *protocols.VMessAEAD) interface{} {
	mux := setting.Mux()
	security := vmess.GetValue(field.Security)
	network := vmess.GetValue(field.NetworkType)
	streamSettings := map[string]interface{}{
		"network":  network,
		"security": security,
	}
	switch security {
	case "tls":
		tlsSettings := map[string]interface{}{
			"allowInsecure": xrayAllowInsecure(setting.AllowInsecure()),
		}
		sni := vmess.GetHostValue(field.SNI)
		alpn := vmess.GetValue(field.Alpn)
		if sni != "" {
			tlsSettings["serverName"] = sni
		}
		if alpn != "" {
			tlsSettings["alpn"] = strings.Split(alpn, ",")
		}
		streamSettings["tlsSettings"] = tlsSettings
	case "reality":
		realitySettings := map[string]interface{}{
			"show":        false,
			"fingerprint": vmess.GetValue(field.FingerPrint),
			"serverName":  vmess.GetHostValue(field.SNI),
			"publicKey":   vmess.GetValue(field.PublicKey),
			"shortId":     vmess.GetValue(field.ShortId),
			"spiderX":     vmess.GetValue(field.SpiderX),
		}
		streamSettings["realitySettings"] = realitySettings
		mux = false
	}
	switch network {
	case "tcp":
		streamSettings["tcpSettings"] = map[string]interface{}{
			"header": map[string]interface{}{
				"type": vmess.GetValue(field.TCPHeaderType),
			},
		}
	case "kcp":
		kcpSettings := map[string]interface{}{
			"mtu":              1350,
			"tti":              50,
			"uplinkCapacity":   12,
			"downlinkCapacity": 100,
			"congestion":       false,
			"readBufferSize":   2,
			"writeBufferSize":  2,
			"header": map[string]interface{}{
				"type": vmess.GetValue(field.MkcpHeaderType),
			},
		}
		if vmess.Has(field.Seed.Key) {
			kcpSettings["seed"] = vmess.GetValue(field.Seed)
		}
		streamSettings["kcpSettings"] = kcpSettings
	case "h2":
		mux = false
		host := make([]string, 0)
		for _, line := range strings.Split(vmess.GetHostValue(field.H2Host), ",") {
			line = strings.TrimSpace(line)
			if line != "" {
				host = append(host, line)
			}
		}
		streamSettings["httpSettings"] = map[string]interface{}{
			"path": vmess.GetValue(field.H2Path),
			"host": host,
		}
	case "ws":
		streamSettings["wsSettings"] = map[string]interface{}{
			"path": vmess.GetValue(field.WsPath),
			"headers": map[string]interface{}{
				"Host": vmess.GetValue(field.WsHost),
			},
		}
	case "quic":
		quicSettings := map[string]interface{}{
			"security": vmess.GetValue(field.QuicSecurity),
			"header": map[string]interface{}{
				"type": vmess.GetValue(field.QuicHeaderType),
			},
		}
		if vmess.GetValue(field.QuicSecurity) != "none" {
			quicSettings["key"] = vmess.GetValue(field.QuicKey)
		}
		streamSettings["quicSettings"] = quicSettings
	case "grpc":
		streamSettings["grpcSettings"] = map[string]interface{}{
			"serviceName": vmess.GetValue(field.GrpcServiceName),
			"multiMode":   vmess.GetValue(field.GrpcMode) == "multi",
		}
	case "xhttp":
		streamSettings["xhttpSettings"] = xhttpSettings(
			vmess.GetValue(field.WsPath),
			vmess.GetValue(field.WsHost),
			vmess.GetValue(field.XhttpMode),
		)
	}
	return map[string]interface{}{
		"tag":      "proxy",
		"protocol": "vmess",
		"settings": map[string]interface{}{
			"vnext": []interface{}{
				map[string]interface{}{
					"address": vmess.Address,
					"port":    vmess.Port,
					"users": []interface{}{
						map[string]interface{}{
							"id":       vmess.ID,
							"security": vmess.GetValue(field.VMessEncryption),
							"level":    0,
						},
					},
				},
			},
		},
		"streamSettings": streamSettings,
		"mux": map[string]interface{}{
			"enabled": mux,
		},
	}
}

// xhttpSettings generates the xray xhttp (splithttp) transport config.
// Empty path defaults to "/", empty mode defaults to "auto", empty host is omitted.
func xhttpSettings(path, host, mode string) map[string]interface{} {
	if path == "" {
		path = "/"
	}
	if mode == "" {
		mode = "auto"
	}
	settings := map[string]interface{}{
		"path": path,
		"mode": mode,
	}
	if host != "" {
		settings["host"] = host
	}
	return settings
}
