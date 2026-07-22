package protocols

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

// buildVMessLink builds a vmess:// base64(json) link for use across test cases.
func buildVMessLink(m map[string]any) string {
	data, _ := json.Marshal(m)
	return "vmess://" + base64.StdEncoding.EncodeToString(data)
}

func validVMessMap() map[string]any {
	return map[string]any{
		"v":    "2",
		"ps":   "test-node",
		"add":  "example.com",
		"port": "443",
		"id":   "b831381d-6324-4d53-ad4f-8cda48b30811",
		"aid":  "0",
		"scy":  "auto",
		"net":  "ws",
		"type": "none",
		"host": "example.com",
		"path": "/path",
		"tls":  "tls",
	}
}

func TestParseVMessLink(t *testing.T) {
	link := buildVMessLink(validVMessMap())
	p := ParseLink(link)
	if p == nil {
		t.Fatal("valid vmess link parsed to nil")
	}
	v, ok := p.(*VMess)
	if !ok {
		t.Fatalf("expected *VMess, got %T", p)
	}
	if v.GetProtocolMode() != ModeVMess {
		t.Errorf("mode = %s, expected %s", v.GetProtocolMode(), ModeVMess)
	}
	if v.Add != "example.com" || v.Port != 443 {
		t.Errorf("address/port parse error: %s:%d", v.Add, v.Port)
	}
	if v.Id != "b831381d-6324-4d53-ad4f-8cda48b30811" {
		t.Errorf("id parse error: %s", v.Id)
	}
	if v.Ps != "test-node" {
		t.Errorf("remarks parse error: %s", v.Ps)
	}
	if v.Net != "ws" || v.Path != "/path" {
		t.Errorf("transport parse error: net=%s path=%s", v.Net, v.Path)
	}
}

// TestParseVMessLink_MissingRequiredField verifies parse returns nil when required fields are missing.
// The current ParseVMessLink requires all of aid/net/type/host/path/tls to be present.
func TestParseVMessLink_MissingRequiredField(t *testing.T) {
	fields := []string{"ps", "add", "port", "id", "aid", "net", "type", "host", "path", "tls"}
	for _, f := range fields {
		m := validVMessMap()
		delete(m, f)
		if p := ParseLink(buildVMessLink(m)); p != nil {
			t.Errorf("expected nil when field %q missing, got successful parse", f)
		}
	}
}

func TestParseVMessLink_AlpnCleared(t *testing.T) {
	// parse.go intentionally clears alpn (some nodes fail to connect when alpn is set)
	m := validVMessMap()
	m["alpn"] = "h2,http/1.1"
	p := ParseLink(buildVMessLink(m))
	if p == nil {
		t.Fatal("parse returned nil")
	}
	if v := p.(*VMess); v.Alpn != "" {
		t.Errorf("alpn should be cleared, got %q", v.Alpn)
	}
}

func TestParseVLessLink(t *testing.T) {
	link := "vless://b831381d-6324-4d53-ad4f-8cda48b30811@example.com:443?" +
		"security=reality&sni=www.example.com&pbk=abc&sid=00&type=tcp&flow=xtls-rprx-vision#reality-node"
	p := ParseLink(link)
	if p == nil {
		t.Fatal("valid vless link parsed to nil")
	}
	v, ok := p.(*VLess)
	if !ok {
		t.Fatalf("expected *VLess, got %T", p)
	}
	if v.ID != "b831381d-6324-4d53-ad4f-8cda48b30811" {
		t.Errorf("id parse error: %s", v.ID)
	}
	if v.Address != "example.com" || v.Port != 443 {
		t.Errorf("address/port parse error: %s:%d", v.Address, v.Port)
	}
	if v.Remarks != "reality-node" {
		t.Errorf("remarks parse error: %s", v.Remarks)
	}
	if got := v.Get("security"); got != "reality" {
		t.Errorf("security query parse error: %s", got)
	}
	if got := v.Get("flow"); got != "xtls-rprx-vision" {
		t.Errorf("flow query parse error: %s", got)
	}
}

func TestParseTrojanLink(t *testing.T) {
	link := "trojan://password123@example.com:443?security=tls&sni=example.com&type=ws&path=/ws#trojan-node"
	p := ParseLink(link)
	if p == nil {
		t.Fatal("valid trojan link parsed to nil")
	}
	tr, ok := p.(*Trojan)
	if !ok {
		t.Fatalf("expected *Trojan, got %T", p)
	}
	if tr.Password != "password123" {
		t.Errorf("password parse error: %s", tr.Password)
	}
	if tr.Address != "example.com" || tr.Port != 443 {
		t.Errorf("address/port parse error: %s:%d", tr.Address, tr.Port)
	}
	if tr.Sni() != "example.com" {
		t.Errorf("sni parse error: %s", tr.Sni())
	}
	if tr.Remarks != "trojan-node" {
		t.Errorf("remarks parse error: %s", tr.Remarks)
	}
}

func TestParseSSLink_SIP002(t *testing.T) {
	// ss://base64(method:password)@host:port#remarks
	userinfo := base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:password123"))
	link := "ss://" + userinfo + "@example.com:8388#ss-node"
	p := ParseLink(link)
	if p == nil {
		t.Fatal("valid ss(SIP002) link parsed to nil")
	}
	ss, ok := p.(*ShadowSocks)
	if !ok {
		t.Fatalf("expected *ShadowSocks, got %T", p)
	}
	if ss.Method != "aes-256-gcm" || ss.Password != "password123" {
		t.Errorf("method/password parse error: %s / %s", ss.Method, ss.Password)
	}
	if ss.Address != "example.com" || ss.Port != 8388 {
		t.Errorf("address/port parse error: %s:%d", ss.Address, ss.Port)
	}
}

func TestParseSSLink_FullBase64(t *testing.T) {
	// ss://base64(method:password@host:port)#remarks
	raw := base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:password123@example.com:8388"))
	link := "ss://" + raw + "#ss-node"
	p := ParseLink(link)
	if p == nil {
		t.Fatal("valid ss(full base64) link parsed to nil")
	}
	ss := p.(*ShadowSocks)
	if ss.Method != "aes-256-gcm" || ss.Password != "password123" {
		t.Errorf("method/password parse error: %s / %s", ss.Method, ss.Password)
	}
	if ss.Address != "example.com" || ss.Port != 8388 {
		t.Errorf("address/port parse error: %s:%d", ss.Address, ss.Port)
	}
}

func TestParseSSRLink(t *testing.T) {
	// host:port:protocol:method:obfs:base64(password)/?params
	pwd := base64Encode("password123")
	remarks := base64Encode("ssr-node")
	body := "example.com:8388:origin:aes-256-cfb:plain:" + pwd + "/?remarks=" + remarks
	link := "ssr://" + base64EncodeWithEq(body)
	p := ParseLink(link)
	if p == nil {
		t.Fatal("valid ssr link parsed to nil")
	}
	ssr, ok := p.(*ShadowSocksR)
	if !ok {
		t.Fatalf("expected *ShadowSocksR, got %T", p)
	}
	if ssr.Address != "example.com" || ssr.Port != 8388 {
		t.Errorf("address/port parse error: %s:%d", ssr.Address, ssr.Port)
	}
	if ssr.Method != "aes-256-cfb" || ssr.Protocol != "origin" || ssr.Obfs != "plain" {
		t.Errorf("method/protocol/obfs parse error: %s/%s/%s", ssr.Method, ssr.Protocol, ssr.Obfs)
	}
	if ssr.Password != "password123" {
		t.Errorf("password parse error: %s", ssr.Password)
	}
	if ssr.Remarks != "ssr-node" {
		t.Errorf("remarks parse error: %s", ssr.Remarks)
	}
}

func TestParseHysteria2Link(t *testing.T) {
	for _, scheme := range []string{"hysteria2", "hy2"} {
		link := scheme + "://password123@example.com:443?sni=example.com#hy2-node"
		p := ParseLink(link)
		if p == nil {
			t.Fatalf("scheme %s parsed to nil", scheme)
		}
		h, ok := p.(*Hysteria2)
		if !ok {
			t.Fatalf("scheme %s expected *Hysteria2, got %T", scheme, p)
		}
		if h.Password != "password123" || h.Address != "example.com" || h.Port != 443 {
			t.Errorf("scheme %s field parse error: %s@%s:%d", scheme, h.Password, h.Address, h.Port)
		}
		if h.Sni() != "example.com" {
			t.Errorf("scheme %s sni parse error: %s", scheme, h.Sni())
		}
	}
}

func TestParseTUICLink(t *testing.T) {
	link := "tuic://uuid-1234:password123@example.com:443?congestion_control=bbr&sni=example.com&alpn=h3#tuic-node"
	p := ParseLink(link)
	if p == nil {
		t.Fatal("valid tuic link parsed to nil")
	}
	tu, ok := p.(*TUIC)
	if !ok {
		t.Fatalf("expected *TUIC, got %T", p)
	}
	if tu.UUID != "uuid-1234" || tu.Password != "password123" {
		t.Errorf("uuid/password parse error: %s / %s", tu.UUID, tu.Password)
	}
	if tu.Address != "example.com" || tu.Port != 443 {
		t.Errorf("address/port parse error: %s:%d", tu.Address, tu.Port)
	}
	if got := tu.Get("congestion_control"); got != "bbr" {
		t.Errorf("congestion_control parse error: %s", got)
	}
}

func TestParseAnyTLSLink(t *testing.T) {
	link := "anytls://password123@example.com:443?sni=example.com#anytls-node"
	p := ParseLink(link)
	if p == nil {
		t.Fatal("valid anytls link parsed to nil")
	}
	a, ok := p.(*AnyTLS)
	if !ok {
		t.Fatalf("expected *AnyTLS, got %T", p)
	}
	if a.Password != "password123" || a.Address != "example.com" || a.Port != 443 {
		t.Errorf("field parse error: %s@%s:%d", a.Password, a.Address, a.Port)
	}
	if a.Sni() != "example.com" {
		t.Errorf("sni parse error: %s", a.Sni())
	}
}

func TestParseSocksLink(t *testing.T) {
	link := "socks5://user:pass@example.com:1080#socks-node"
	p := ParseLink(link)
	if p == nil {
		t.Fatal("valid socks link parsed to nil")
	}
	s, ok := p.(*Socks)
	if !ok {
		t.Fatalf("expected *Socks, got %T", p)
	}
	if s.Username != "user" || s.Password != "pass" {
		t.Errorf("username/password parse error: %s / %s", s.Username, s.Password)
	}
	if s.Address != "example.com" || s.Port != 1080 {
		t.Errorf("address/port parse error: %s:%d", s.Address, s.Port)
	}
}

func TestParseSocksLink_NoAuth(t *testing.T) {
	link := "socks5://example.com:1080#no-auth"
	p := ParseLink(link)
	if p == nil {
		t.Fatal("auth-less socks link should parse successfully")
	}
	s := p.(*Socks)
	if s.Username != "" || s.Password != "" {
		t.Errorf("should have no auth info: %s / %s", s.Username, s.Password)
	}
}

// TestParseLink_InvalidReturnsNil covers various invalid inputs.
func TestParseLink_InvalidReturnsNil(t *testing.T) {
	cases := []struct {
		name string
		link string
	}{
		{"empty string", ""},
		{"unknown scheme", "unknown://foo@bar:443#x"},
		{"vmess non-base64", "vmess://!!!not-base64!!!"},
		{"vmess empty body", "vmess://"},
		{"vless missing port", "vless://uuid@example.com#x"},
		{"vless missing user", "vless://@example.com:443#x"},
		{"trojan missing password", "trojan://@example.com:443#x"},
		{"trojan non-numeric port", "trojan://pwd@example.com:abc#x"},
		{"tuic missing password", "tuic://uuid@example.com:443#x"},
		{"hysteria2 missing password", "hysteria2://@example.com:443#x"},
		{"anytls missing password", "anytls://@example.com:443#x"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if p := ParseLink(c.link); p != nil {
				t.Errorf("invalid link should return nil, got %T", p)
			}
		})
	}
}

// TestPortBoundary records the current inconsistent port upper-bound checks.
// Note: Trojan/TUIC/Hysteria2/AnyTLS/Socks use port < 65535 (reject 65535),
// while VMess/VLESS/VMessAEAD use port <= 65535 (accept 65535).
// This case pins current behavior; update if the bounds are unified later.
func TestPortBoundary(t *testing.T) {
	if p := ParseLink("trojan://pwd@example.com:65535#x"); p != nil {
		t.Errorf("trojan port 65535 should currently return nil (port < 65535)")
	}
	if p := ParseLink("trojan://pwd@example.com:0#x"); p != nil {
		t.Errorf("trojan port 0 should return nil")
	}
	link := buildVMessLink(func() map[string]any {
		m := validVMessMap()
		m["port"] = "65535"
		return m
	}())
	if p := ParseLink(link); p == nil {
		t.Errorf("vmess port 65535 should currently parse successfully (port <= 65535)")
	}
}

// TestRoundTrip verifies that parse -> GetLink -> parse keeps key fields stable.
// This is regression protection for GetLink / serialization changes (Node
// persistence depends on GetLink round-trip).
func TestRoundTrip(t *testing.T) {
	links := []string{
		buildVMessLink(validVMessMap()),
		"vless://b831381d-6324-4d53-ad4f-8cda48b30811@example.com:443?security=tls&sni=example.com&type=ws&path=/ws#vless-node",
		"trojan://password123@example.com:443?security=tls&sni=example.com#trojan-node",
		"tuic://uuid-1234:password123@example.com:443?congestion_control=bbr&sni=example.com#tuic-node",
		"hysteria2://password123@example.com:443?sni=example.com#hy2-node",
		"anytls://password123@example.com:443?sni=example.com#anytls-node",
		"socks5://user:pass@example.com:1080#socks-node",
	}
	for _, link := range links {
		first := ParseLink(link)
		if first == nil {
			t.Errorf("initial parse failed: %s", link)
			continue
		}
		second := ParseLink(first.GetLink())
		if second == nil {
			t.Errorf("second parse of GetLink() failed, protocol %s, link=%s", first.GetProtocolMode(), first.GetLink())
			continue
		}
		if first.GetProtocolMode() != second.GetProtocolMode() {
			t.Errorf("protocol changed on round-trip: %s -> %s", first.GetProtocolMode(), second.GetProtocolMode())
		}
		if first.GetAddr() != second.GetAddr() || first.GetPort() != second.GetPort() {
			t.Errorf("address/port changed on round-trip: %s:%d -> %s:%d",
				first.GetAddr(), first.GetPort(), second.GetAddr(), second.GetPort())
		}
		if first.GetName() != second.GetName() {
			t.Errorf("remarks changed on round-trip: %q -> %q", first.GetName(), second.GetName())
		}
	}
}
