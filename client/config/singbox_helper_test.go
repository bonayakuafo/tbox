package config

import (
	"net/url"
	"testing"

	"tbox/core/protocols"
)

// TestVLessOutbound_InsecureFromLink locks in that insecure=1 from the link
// takes effect; previously VLESS only read the global setting, ignoring the
// link parameter and causing TLS handshake failures for self-signed nodes.
func TestVLessOutbound_InsecureFromLink(t *testing.T) {
	vless := &protocols.VLess{
		ID:      "5ed6c433-6bec-4762-84d0-000000000000",
		Address: "1.1.1.1",
		Port:    50133,
		Remarks: "hk",
		Values: url.Values{
			"security": []string{"tls"},
			"type":     []string{"tcp"},
			"flow":     []string{"xtls-rprx-vision"},
			"sni":      []string{"d1.example.com"},
			"insecure": []string{"1"},
		},
	}
	out := SingBox{}.outboundForProtocol(vless, "proxy")
	tls, ok := out["tls"].(map[string]interface{})
	if !ok {
		t.Fatal("tls node should produce a tls block")
	}
	if tls["insecure"] != true {
		t.Errorf("insecure should be true when link has insecure=1, got %v", tls["insecure"])
	}
	if tls["server_name"] != "d1.example.com" {
		t.Errorf("unexpected server_name: %v", tls["server_name"])
	}
	if out["flow"] != "xtls-rprx-vision" {
		t.Errorf("unexpected flow: %v", out["flow"])
	}
}

func TestVLessOutbound_SecureByDefault(t *testing.T) {
	vless := &protocols.VLess{
		ID:      "uuid",
		Address: "1.1.1.1",
		Port:    443,
		Remarks: "n",
		Values: url.Values{
			"security": []string{"tls"},
			"type":     []string{"tcp"},
			"sni":      []string{"d1.example.com"},
		},
	}
	out := SingBox{}.outboundForProtocol(vless, "proxy")
	tls := out["tls"].(map[string]interface{})
	if tls["insecure"] != false {
		t.Errorf("insecure should be false when not set, got %v", tls["insecure"])
	}
}

func TestBuildTLS_Basic(t *testing.T) {
	tls := buildTLS(tlsOptions{
		ServerName: "example.com",
		Alpn:       "h2,http/1.1",
		Insecure:   true,
	})
	if tls["enabled"] != true {
		t.Error("enabled should be true")
	}
	if tls["insecure"] != true {
		t.Error("insecure should be true")
	}
	if tls["server_name"] != "example.com" {
		t.Errorf("unexpected server_name: %v", tls["server_name"])
	}
	alpn, ok := tls["alpn"].([]string)
	if !ok || len(alpn) != 2 || alpn[0] != "h2" || alpn[1] != "http/1.1" {
		t.Errorf("bad alpn parse: %v", tls["alpn"])
	}
	if _, ok := tls["reality"]; ok {
		t.Error("non-reality should not include reality block")
	}
}

func TestBuildTLS_OmitsEmpty(t *testing.T) {
	tls := buildTLS(tlsOptions{Insecure: false})
	if _, ok := tls["server_name"]; ok {
		t.Error("empty SNI should not emit server_name")
	}
	if _, ok := tls["alpn"]; ok {
		t.Error("empty alpn should not emit alpn")
	}
	if tls["insecure"] != false {
		t.Error("insecure should be explicitly false")
	}
}

func TestBuildTLS_Reality(t *testing.T) {
	tls := buildTLS(tlsOptions{
		ServerName: "example.com",
		Reality:    true,
		PublicKey:  "PUBKEY",
		ShortID:    "sid1",
	})
	reality, ok := tls["reality"].(map[string]interface{})
	if !ok {
		t.Fatal("should include reality block")
	}
	if reality["public_key"] != "PUBKEY" || reality["short_id"] != "sid1" {
		t.Errorf("bad reality fields: %v", reality)
	}
	utls, ok := tls["utls"].(map[string]interface{})
	if !ok {
		t.Fatal("reality should carry a utls block")
	}
	if utls["fingerprint"] != "chrome" {
		t.Errorf("empty fingerprint should default to chrome, got %v", utls["fingerprint"])
	}
}

func TestBuildStreamTransport(t *testing.T) {
	ws := buildStreamTransport(transportOptions{Network: "ws", Path: "/p", Host: "h.com"})
	if ws["type"] != "ws" || ws["path"] != "/p" {
		t.Errorf("bad ws transport: %v", ws)
	}
	if headers, ok := ws["headers"].(map[string]interface{}); !ok || headers["Host"] != "h.com" {
		t.Errorf("bad ws Host header: %v", ws["headers"])
	}

	h2 := buildStreamTransport(transportOptions{Network: "h2", Path: "/", Host: "a.com,b.com"})
	if h2["type"] != "http" {
		t.Errorf("h2 should map to http, got %v", h2["type"])
	}
	if hosts, ok := h2["host"].([]string); !ok || len(hosts) != 2 {
		t.Errorf("bad h2 host list: %v", h2["host"])
	}

	grpc := buildStreamTransport(transportOptions{Network: "grpc", ServiceName: "svc"})
	if grpc["type"] != "grpc" || grpc["service_name"] != "svc" {
		t.Errorf("bad grpc transport: %v", grpc)
	}

	if buildStreamTransport(transportOptions{Network: "tcp"}) != nil {
		t.Error("tcp should return nil")
	}
	if buildStreamTransport(transportOptions{Network: "unknown"}) != nil {
		t.Error("unknown network should return nil")
	}
}

// TestVMessAEADOutbound_SingBoxFormat is the core regression guard for this
// fix: confirms that the VMessAEAD outbound is generated in flat sing-box
// format, not the old xray format.
func TestVMessAEADOutbound_SingBoxFormat(t *testing.T) {
	vmess := &protocols.VMessAEAD{
		ID:      "b831381d-6324-4d53-ad4f-8cda48b30811",
		Address: "example.com",
		Port:    443,
		Remarks: "aead-node",
		Values: url.Values{
			"security": []string{"tls"},
			"type":     []string{"ws"},
			"path":     []string{"/ws"},
			"host":     []string{"example.com"},
			"sni":      []string{"example.com"},
		},
	}
	out := SingBox{}.outboundForProtocol(vmess, "proxy")
	if out == nil {
		t.Fatal("VMessAEAD outbound returned nil")
	}

	// Key fields of the flat sing-box format
	if out["type"] != "vmess" {
		t.Errorf("type should be vmess, got %v", out["type"])
	}
	if out["server"] != "example.com" || out["server_port"] != 443 {
		t.Errorf("bad server/server_port: %v:%v", out["server"], out["server_port"])
	}
	if out["uuid"] != "b831381d-6324-4d53-ad4f-8cda48b30811" {
		t.Errorf("bad uuid: %v", out["uuid"])
	}

	// The old xray-format fields must not reappear
	for _, xrayKey := range []string{"protocol", "settings", "streamSettings"} {
		if _, ok := out[xrayKey]; ok {
			t.Errorf("xray-format field %q should not be present", xrayKey)
		}
	}

	// tls block should be produced by buildTLS
	tls, ok := out["tls"].(map[string]interface{})
	if !ok || tls["enabled"] != true || tls["server_name"] != "example.com" {
		t.Errorf("bad tls block: %v", out["tls"])
	}

	// transport should be the sing-box ws structure
	tr, ok := out["transport"].(map[string]interface{})
	if !ok || tr["type"] != "ws" || tr["path"] != "/ws" {
		t.Errorf("bad transport block: %v", out["transport"])
	}
}

func TestVMessAEADOutbound_Reality(t *testing.T) {
	vmess := &protocols.VMessAEAD{
		ID:      "uuid",
		Address: "example.com",
		Port:    443,
		Remarks: "r",
		Values: url.Values{
			"security": []string{"reality"},
			"type":     []string{"tcp"},
			"pbk":      []string{"PUBKEY"},
			"sid":      []string{"sid1"},
			"sni":      []string{"example.com"},
		},
	}
	out := SingBox{}.outboundForProtocol(vmess, "proxy")
	tls, ok := out["tls"].(map[string]interface{})
	if !ok {
		t.Fatal("reality node should carry a tls block")
	}
	if _, ok := tls["reality"].(map[string]interface{}); !ok {
		t.Errorf("should include reality block: %v", tls)
	}
	// tcp without special headers should not emit transport
	if _, ok := out["transport"]; ok {
		t.Errorf("tcp network should not emit transport: %v", out["transport"])
	}
}

// TestHysteria2Outbound_Obfs verifies that a salamander obfuscation block is
// generated when the link has obfs; and no obfs block otherwise.
func TestHysteria2Outbound_Obfs(t *testing.T) {
	withObfs := &protocols.Hysteria2{
		Password: "pw",
		Address:  "example.com",
		Port:     443,
		Remarks:  "h",
		Values: url.Values{
			"obfs":          []string{"salamander"},
			"obfs-password": []string{"o-pw"},
		},
	}
	out := SingBox{}.hysteria2Outbound(withObfs)
	obfs, ok := out["obfs"].(map[string]interface{})
	if !ok {
		t.Fatal("obfs block should be generated when link has obfs param")
	}
	if obfs["type"] != "salamander" || obfs["password"] != "o-pw" {
		t.Errorf("bad obfs fields: %v", obfs)
	}

	noObfs := &protocols.Hysteria2{
		Password: "pw", Address: "example.com", Port: 443, Remarks: "h",
		Values: url.Values{},
	}
	outNoObfs := SingBox{}.hysteria2Outbound(noObfs)
	if _, ok := outNoObfs["obfs"]; ok {
		t.Error("no obfs param should not emit obfs block")
	}
}

// TestHysteria2Insecure verifies that the link insecure parameter takes
// precedence over the global setting.
// TestHysteria2Outbound_Insecure verifies that the hysteria2 outbound
// recognizes the various parameter names for skipping cert verification
// (insecure / allowInsecure) and propagates them to tls.insecure.
func TestHysteria2Outbound_Insecure(t *testing.T) {
	cases := []struct {
		query string
		want  bool
	}{
		{"insecure=1", true},
		{"insecure=true", true},
		{"insecure=yes", true},
		{"insecure=0", false},
		{"allowInsecure=true", true},
		{"allowInsecure=1", true},
		{"", false},
	}
	for _, c := range cases {
		u, _ := url.Parse("hysteria2://pw@example.com:443?" + c.query)
		h := &protocols.Hysteria2{
			Password: "pw", Address: "example.com", Port: 443, Remarks: "h",
			Values: u.Query(),
		}
		out := SingBox{}.hysteria2Outbound(h)
		tls := out["tls"].(map[string]interface{})
		if tls["insecure"] != c.want {
			t.Errorf("query=%q: expected insecure=%v, got %v", c.query, c.want, tls["insecure"])
		}
	}
}

// TestHysteria2Outbound_AlpnDefault verifies that alpn defaults to h3 and can
// be overridden by the link.
func TestHysteria2Outbound_AlpnDefault(t *testing.T) {
	def := &protocols.Hysteria2{
		Password: "pw", Address: "example.com", Port: 443, Remarks: "h",
		Values: url.Values{},
	}
	tls := SingBox{}.hysteria2Outbound(def)["tls"].(map[string]interface{})
	if alpn, ok := tls["alpn"].([]string); !ok || len(alpn) != 1 || alpn[0] != "h3" {
		t.Errorf("default alpn should be [h3], got %v", tls["alpn"])
	}

	custom := &protocols.Hysteria2{
		Password: "pw", Address: "example.com", Port: 443, Remarks: "h",
		Values: url.Values{"alpn": []string{"h3,h2"}},
	}
	tls = SingBox{}.hysteria2Outbound(custom)["tls"].(map[string]interface{})
	if alpn, ok := tls["alpn"].([]string); !ok || len(alpn) != 2 {
		t.Errorf("custom alpn should be [h3 h2], got %v", tls["alpn"])
	}
}

// TestResolveInsecure_ParamNames locks in that multiple parameter names all
// trigger skipping cert verification.
func TestResolveInsecure_ParamNames(t *testing.T) {
	for _, key := range []string{"insecure", "allowInsecure", "allow_insecure", "skip-cert-verify"} {
		v := url.Values{key: []string{"1"}}
		if !resolveInsecure(v) {
			t.Errorf("param %q=1 should trigger insecure", key)
		}
	}
	if resolveInsecure(url.Values{}) {
		t.Error("no params and global disabled should not be insecure")
	}
	if resolveInsecure(url.Values{"insecure": []string{"0"}}) {
		t.Error("insecure=0 should not trigger")
	}
}

// TestTUICOutbound_InsecureFromLink locks in that TUIC reads the link insecure param.
func TestTUICOutbound_InsecureFromLink(t *testing.T) {
	tuic := &protocols.TUIC{
		UUID: "u", Password: "p", Address: "1.1.1.1", Port: 443, Remarks: "t",
		Values: url.Values{"insecure": []string{"1"}, "sni": []string{"a.com"}},
	}
	out := SingBox{}.tuicOutbound(tuic)
	tls := out["tls"].(map[string]interface{})
	if tls["insecure"] != true {
		t.Errorf("TUIC should read insecure=true from link, got %v", tls["insecure"])
	}
}

// TestAnyTLSOutbound_InsecureFromLink locks in that AnyTLS reads the link insecure param.
func TestAnyTLSOutbound_InsecureFromLink(t *testing.T) {
	anytls := &protocols.AnyTLS{
		Password: "p", Address: "1.1.1.1", Port: 443, Remarks: "a",
		Values: url.Values{"allowInsecure": []string{"true"}},
	}
	out := SingBox{}.anytlsOutbound(anytls)
	tls := out["tls"].(map[string]interface{})
	if tls["insecure"] != true {
		t.Errorf("AnyTLS should read insecure=true from link, got %v", tls["insecure"])
	}
}

// TestOutboundConfig_BridgeMainProxy shape 1: the main node is itself the
// converter; the proxy outbound should be replaced with a socks outbound
// pointing at bridgePort, delegating the main outbound to the converter.
func TestOutboundConfig_BridgeMainProxy(t *testing.T) {
	vless := &protocols.VLess{
		ID: "uuid", Address: "1.1.1.1", Port: 443, Remarks: "n",
		Values: url.Values{"type": []string{"xhttp"}, "security": []string{"tls"}},
	}
	plan := &BridgePlan{Enabled: true, ConverterCore: "xray", NodeIndex: 1, MainIsConverter: true}
	out := SingBox{}.outboundConfig(vless, nil, plan, 47890).([]interface{})
	if len(out) == 0 {
		t.Fatal("outbound list is empty")
	}
	proxy := out[0].(map[string]interface{})
	if proxy["tag"] != "proxy" {
		t.Fatalf("first outbound should be proxy, got %v", proxy["tag"])
	}
	if proxy["type"] != "socks" {
		t.Errorf("shape 1 proxy outbound type should be socks, got %v", proxy["type"])
	}
	if proxy["server"] != "127.0.0.1" || proxy["server_port"] != 47890 {
		t.Errorf("proxy should point at 127.0.0.1:47890, got %v:%v", proxy["server"], proxy["server_port"])
	}
}

// TestOutboundConfig_BridgeSplitConverter shape 2: main node is a normal
// protocol, converter serves only split targets. proxy should keep the main
// node's native outbound; an additional converter-out socks outbound is generated.
func TestOutboundConfig_BridgeSplitConverter(t *testing.T) {
	main := &protocols.VLess{
		ID: "uuid", Address: "1.1.1.1", Port: 443, Remarks: "main",
		Values: url.Values{"type": []string{"tcp"}, "security": []string{"tls"}},
	}
	plan := &BridgePlan{Enabled: true, ConverterCore: "xray", NodeIndex: 2, MainIsConverter: false}
	out := SingBox{}.outboundConfig(main, nil, plan, 47890).([]interface{})
	proxy := out[0].(map[string]interface{})
	if proxy["type"] != "vless" {
		t.Errorf("shape 2 proxy should be the main node's native vless outbound, got %v", proxy["type"])
	}
	// find converter-out
	var conv map[string]interface{}
	for _, o := range out {
		m := o.(map[string]interface{})
		if m["tag"] == converterTag {
			conv = m
			break
		}
	}
	if conv == nil {
		t.Fatal("shape 2 should generate the converter-out outbound")
	}
	if conv["type"] != "socks" || conv["server_port"] != 47890 {
		t.Errorf("converter-out should be socks:47890, got %v:%v", conv["type"], conv["server_port"])
	}
}

// TestOutboundConfig_NonBridgeUnchanged locks in that the non-bridge path
// (plan=nil) still generates the node's native protocol outbound (vless).
func TestOutboundConfig_NonBridgeUnchanged(t *testing.T) {
	vless := &protocols.VLess{
		ID: "uuid", Address: "1.1.1.1", Port: 443, Remarks: "n",
		Values: url.Values{"type": []string{"tcp"}, "security": []string{"tls"}},
	}
	out := SingBox{}.outboundConfig(vless, nil, nil, 0).([]interface{})
	proxy := out[0].(map[string]interface{})
	if proxy["type"] != "vless" {
		t.Errorf("non-bridge mode proxy should be vless outbound, got %v", proxy["type"])
	}
}
