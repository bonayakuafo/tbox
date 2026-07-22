package config

import (
	"testing"

	"tbox/core/protocols"
)

// TestXrayVLessXhttp uses a real xhttp share link to verify that the xray
// outbound generates the correct xhttpSettings block (network=xhttp with
// path/host/mode), which is what this adaptation is centered on.
func TestXrayVLessXhttp(t *testing.T) {
	link := "vless://5bd2-s-s-s@cfa.s.com:443?type=xhttp&encryption=none&" +
		"host=usbad.osaip.net&path=%2Fapi%2Fv1%2Fupload&security=tls&fp=chrome&" +
		"insecure=0&sni=usbad.s.net&mode=auto#xhttp-node"
	p := protocols.ParseLink(link)
	if p == nil {
		t.Fatal("xhttp link parse failed")
	}
	vless, ok := p.(*protocols.VLess)
	if !ok {
		t.Fatalf("expected *VLess, got %T", p)
	}

	out := Xray{}.vLessOutbound(vless).(map[string]interface{})
	ss := out["streamSettings"].(map[string]interface{})

	if ss["network"] != "xhttp" {
		t.Errorf("network should be xhttp, got %v", ss["network"])
	}
	xs, ok := ss["xhttpSettings"].(map[string]interface{})
	if !ok {
		t.Fatal("xhttpSettings block should be generated")
	}
	if xs["path"] != "/api/v1/upload" {
		t.Errorf("bad path: %v", xs["path"])
	}
	if xs["host"] != "usbad.osaip.net" {
		t.Errorf("bad host: %v", xs["host"])
	}
	if xs["mode"] != "auto" {
		t.Errorf("bad mode: %v", xs["mode"])
	}

	// tls fingerprint should be written
	tls := ss["tlsSettings"].(map[string]interface{})
	if tls["fingerprint"] != "chrome" {
		t.Errorf("fingerprint should be chrome, got %v", tls["fingerprint"])
	}
	if tls["serverName"] != "usbad.s.net" {
		t.Errorf("bad serverName: %v", tls["serverName"])
	}
}

// TestXhttpSettingsDefaults verifies the defaults for empty path/mode and
// omission of an empty host.
func TestXhttpSettingsDefaults(t *testing.T) {
	s := xhttpSettings("", "", "")
	if s["path"] != "/" {
		t.Errorf("empty path should default to /, got %v", s["path"])
	}
	if s["mode"] != "auto" {
		t.Errorf("empty mode should default to auto, got %v", s["mode"])
	}
	if _, ok := s["host"]; ok {
		t.Errorf("empty host should be omitted, got %v", s["host"])
	}
}

// TestSelectCore verifies the per-node auto core-selection logic.
func TestSelectCore(t *testing.T) {
	// xhttp transport -> xray
	xhttpLink := "vless://id@h.com:443?type=xhttp&security=tls&sni=a.com#x"
	xhttpNode := protocols.ParseLink(xhttpLink)
	if got := SelectCore(xhttpNode, "sing-box"); got != "xray" {
		t.Errorf("xhttp node should pick xray, got %s", got)
	}

	// plain vless (tcp) -> keep the default sing-box
	tcpLink := "vless://id@h.com:443?type=tcp&security=tls&sni=a.com&flow=xtls-rprx-vision#n"
	tcpNode := protocols.ParseLink(tcpLink)
	if got := SelectCore(tcpNode, "sing-box"); got != "sing-box" {
		t.Errorf("tcp node should keep sing-box, got %s", got)
	}

	// Hysteria2 -> force sing-box (even when default is xray)
	hy2 := protocols.ParseLink("hysteria2://pw@h.com:443?sni=a.com#h")
	if got := SelectCore(hy2, "xray"); got != "sing-box" {
		t.Errorf("Hysteria2 should force sing-box, got %s", got)
	}

	// TUIC -> force sing-box
	tuic := protocols.ParseLink("tuic://uuid:pw@h.com:443?sni=a.com#t")
	if got := SelectCore(tuic, "xray"); got != "sing-box" {
		t.Errorf("TUIC should force sing-box, got %s", got)
	}

	// nil node -> returns default
	if got := SelectCore(nil, "xray"); got != "xray" {
		t.Errorf("nil node should return default, got %s", got)
	}
}

// TestDnsBareHost verifies extracting the bare host from various DNS addresses.
func TestDnsBareHost(t *testing.T) {
	cases := map[string]string{
		"tcp://8.8.8.8":       "8.8.8.8",
		"tcp://9.9.9.9:853":   "9.9.9.9",
		"https://1.1.1.1/dns": "1.1.1.1",
		"114.114.114.114":     "114.114.114.114",
		"udp://223.5.5.5":     "223.5.5.5",
		"https://dns.google":  "dns.google",
	}
	for in, want := range cases {
		if got := dnsBareHost(in); got != want {
			t.Errorf("dnsBareHost(%q)=%q, expected %q", in, got, want)
		}
	}
}

// TestDnsRouteIP verifies that only literal IPs are returned; hostnames/DoH
// return empty (to avoid polluting the route's ip field).
func TestDnsRouteIP(t *testing.T) {
	if got := dnsRouteIP("tcp://8.8.8.8"); got != "8.8.8.8" {
		t.Errorf("should return 8.8.8.8, got %q", got)
	}
	if got := dnsRouteIP("https://dns.google"); got != "" {
		t.Errorf("hostname should return empty, got %q", got)
	}
	if got := dnsRouteIP(""); got != "" {
		t.Errorf("empty should return empty, got %q", got)
	}
}
