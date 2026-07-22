package config

import (
	"tbox/core/protocols"
	"tbox/core/protocols/field"
	"tbox/core/setting"
	"errors"
)

type Config interface {
	// GenConfig returns the config info for the core.
	GenConfig(protocols.Protocol) (string, error)
}

var Releases = map[string]string{
	"sing-box": "https://github.com/SagerNet/sing-box/releases",
	"xray":     "https://github.com/XTLS/Xray-core/releases",
	// TODO: the ssr-client repo has not published a release yet; swap in the real URL once available.
	"ssr": "https://github.com/bonayakuafo/ssr/releases",
}

func CreateConfig(serviceName string) (config Config, err error) {
	if serviceName == "sing-box" {
		return SingBox{}, nil
	}
	if serviceName == "xray" {
		return Xray{}, nil
	}

	return nil, errors.New("unsupported core")
}

// singboxOnlyModes marks protocols that xray does not support outbound and
// therefore must be served by sing-box. (SSR is not outbound in either core;
// it is served by the external ssr converter — see client/config/bridge.go.)
var singboxOnlyModes = map[protocols.Mode]bool{
	protocols.ModeHysteria2: true,
	protocols.ModeTUIC:      true,
	protocols.ModeAnyTLS:    true,
}

// SelectCore picks the core actually used based on the node's protocol and
// transport. It defaults to preferred (usually sing-box) and auto-switches
// when a capability is missing:
//   - xhttp transport: sing-box does not support it, use xray
//   - Hysteria2/TUIC/AnyTLS: xray does not support them, use sing-box
//   - SSR: newer sing-box has dropped SSR outbound; always use the external ssr converter
func SelectCore(node protocols.Protocol, preferred string) string {
	if node == nil {
		return preferred
	}
	if node.GetProtocolMode() == protocols.ModeShadowSocksR {
		return "ssr"
	}
	if singboxOnlyModes[node.GetProtocolMode()] {
		return "sing-box"
	}
	if nodeUsesXhttp(node) {
		return "xray"
	}
	return preferred
}

// NodeUsesXhttp reports whether the node uses xhttp (splithttp) transport
// (only xray supports it). Exported so the command layer can validate (e.g.
// chain-proxy forbids xhttp nodes).
func NodeUsesXhttp(node protocols.Protocol) bool {
	return nodeUsesXhttp(node)
}

// nodeUsesXhttp reports whether the node uses xhttp (splithttp) transport
// (only xray supports it).
func nodeUsesXhttp(node protocols.Protocol) bool {
	switch n := node.(type) {
	case *protocols.VLess:
		return n.GetValue(field.NetworkType) == "xhttp"
	case *protocols.VMessAEAD:
		return n.GetValue(field.NetworkType) == "xhttp"
	case *protocols.VMess:
		return n.Net == "xhttp"
	}
	return false
}

// bridgeBasePort is the default internal port between sing-box and the
// converter in bridge mode. It listens only on 127.0.0.1 and is not exposed,
// independent of the user-configurable socks/http/dns ports.
const bridgeBasePort = 47890

// BridgePort returns the bridge internal port; when it collides with
// socks/http/dns ports it is incremented to avoid clashes.
func BridgePort() int {
	used := map[int]bool{
		setting.Socks():   true,
		setting.Http():    true,
		setting.DNSPort(): true,
	}
	port := bridgeBasePort
	for used[port] {
		port++
	}
	return port
}
