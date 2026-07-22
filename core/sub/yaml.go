package sub

import (
	"tbox/log"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type clashSubscription struct {
	Proxies []clashProxy `yaml:"proxies"`
}

type clashProxy struct {
	Name              string           `yaml:"name"`
	Type              string           `yaml:"type"`
	Server            string           `yaml:"server"`
	Port              int              `yaml:"port"`
	UUID              string           `yaml:"uuid"`
	AlterID           int              `yaml:"alterId"`
	Cipher            string           `yaml:"cipher"`
	Password          string           `yaml:"password"`
	TLS               bool             `yaml:"tls"`
	Network           string           `yaml:"network"`
	Host              string           `yaml:"host"`
	Path              string           `yaml:"path"`
	SNI               string           `yaml:"sni"`
	ServerName        string           `yaml:"servername"`
	Flow              string           `yaml:"flow"`
	Security          string           `yaml:"security"`
	ALPN              []string         `yaml:"alpn"`
	SkipCertVerify    bool             `yaml:"skip-cert-verify"`
	ClientFingerprint string           `yaml:"client-fingerprint"`
	UDP               bool             `yaml:"udp"`
	WSOpts            clashWSOpts      `yaml:"ws-opts"`
	HTTPOpts          clashHTTPOpts    `yaml:"http-opts"`
	H2Opts            clashHTTPOpts    `yaml:"h2-opts"`
	GrpcOpts          clashGrpcOpts    `yaml:"grpc-opts"`
	RealityOpts       clashRealityOpts `yaml:"reality-opts"`
}

type clashWSOpts struct {
	Path    string            `yaml:"path"`
	Headers map[string]string `yaml:"headers"`
}

type clashHTTPOpts struct {
	Path string   `yaml:"path"`
	Host []string `yaml:"host"`
}

type clashGrpcOpts struct {
	GrpcServiceName string `yaml:"grpc-service-name"`
}

type clashRealityOpts struct {
	PublicKey string `yaml:"public-key"`
	ShortID   string `yaml:"short-id"`
	SpiderX   string `yaml:"spider-x"`
}

func clashProxiesToLinks(data []byte) ([]string, error) {
	var sub clashSubscription
	if err := yaml.Unmarshal(data, &sub); err != nil {
		return nil, err
	}
	if len(sub.Proxies) == 0 {
		return nil, fmt.Errorf("not clash yaml subscription")
	}

	links := make([]string, 0, len(sub.Proxies))
	for _, proxy := range sub.Proxies {
		link, err := clashProxyToLink(proxy)
		if err != nil {
			log.Warnf("skipped YAML node [%s]: %v", proxyDisplayName(proxy), err)
			continue
		}
		if link != "" {
			links = append(links, link)
		}
	}
	return links, nil
}

func clashProxyToLink(proxy clashProxy) (string, error) {
	if proxy.Server == "" || proxy.Port <= 0 || proxy.Port > 65535 {
		return "", fmt.Errorf("invalid server or port")
	}

	switch strings.ToLower(proxy.Type) {
	case "vmess":
		return clashVMessToLink(proxy)
	case "ss":
		return clashSSToLink(proxy)
	case "trojan":
		return clashTrojanToLink(proxy)
	case "vless":
		return clashVLESSToLink(proxy)
	default:
		return "", fmt.Errorf("unsupported type: %s", proxy.Type)
	}
}

func clashVMessToLink(proxy clashProxy) (string, error) {
	if proxy.UUID == "" {
		return "", fmt.Errorf("vmess missing uuid")
	}

	host, path := clashHostPath(proxy)
	security := clashSecurity(proxy)
	data := map[string]string{
		"v":    "2",
		"ps":   proxyDisplayName(proxy),
		"add":  proxy.Server,
		"port": strconv.Itoa(proxy.Port),
		"id":   proxy.UUID,
		"aid":  strconv.Itoa(proxy.AlterID),
		"scy":  firstNonEmpty(proxy.Cipher, "auto"),
		"net":  firstNonEmpty(proxy.Network, "tcp"),
		"type": clashHeaderType(),
		"host": host,
		"path": path,
		"tls":  security,
	}
	if sni := clashSNI(proxy); sni != "" {
		data["sni"] = sni
	}
	if len(proxy.ALPN) > 0 {
		data["alpn"] = strings.Join(proxy.ALPN, ",")
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return "vmess://" + base64.StdEncoding.EncodeToString(jsonData), nil
}

func clashSSToLink(proxy clashProxy) (string, error) {
	if proxy.Cipher == "" {
		return "", fmt.Errorf("ss missing cipher")
	}
	if proxy.Cipher != "none" && proxy.Password == "" {
		return "", fmt.Errorf("ss missing password")
	}

	userinfo := base64.StdEncoding.EncodeToString([]byte(proxy.Cipher + ":" + proxy.Password))
	return fmt.Sprintf("ss://%s@%s:%d#%s", userinfo, proxy.Server, proxy.Port, url.QueryEscape(proxyDisplayName(proxy))), nil
}

func clashTrojanToLink(proxy clashProxy) (string, error) {
	if proxy.Password == "" {
		return "", fmt.Errorf("trojan missing password")
	}

	values := url.Values{}
	if sni := clashSNI(proxy); sni != "" {
		values.Set("sni", sni)
	}
	if len(proxy.ALPN) > 0 {
		values.Set("alpn", strings.Join(proxy.ALPN, ","))
	}
	if proxy.SkipCertVerify {
		values.Set("allowInsecure", "1")
	}

	u := url.URL{
		Scheme:   "trojan",
		User:     url.User(proxy.Password),
		Host:     fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
		RawQuery: values.Encode(),
		Fragment: proxyDisplayName(proxy),
	}
	return u.String(), nil
}

func clashVLESSToLink(proxy clashProxy) (string, error) {
	if proxy.UUID == "" {
		return "", fmt.Errorf("vless missing uuid")
	}

	host, path := clashHostPath(proxy)
	values := url.Values{}
	values.Set("encryption", "none")
	values.Set("type", firstNonEmpty(proxy.Network, "tcp"))

	if security := clashSecurity(proxy); security != "" {
		values.Set("security", security)
	}
	if sni := clashSNI(proxy); sni != "" {
		values.Set("sni", sni)
	}
	if proxy.Flow != "" {
		values.Set("flow", proxy.Flow)
	}
	if len(proxy.ALPN) > 0 {
		values.Set("alpn", strings.Join(proxy.ALPN, ","))
	}
	if proxy.ClientFingerprint != "" {
		values.Set("fp", proxy.ClientFingerprint)
	}
	if proxy.SkipCertVerify {
		values.Set("allowInsecure", "1")
	}
	if proxy.RealityOpts.PublicKey != "" {
		values.Set("pbk", proxy.RealityOpts.PublicKey)
	}
	if proxy.RealityOpts.ShortID != "" {
		values.Set("sid", proxy.RealityOpts.ShortID)
	}
	if proxy.RealityOpts.SpiderX != "" {
		values.Set("spx", proxy.RealityOpts.SpiderX)
	}

	switch values.Get("type") {
	case "ws", "http", "h2":
		if path != "" {
			values.Set("path", path)
		}
		if host != "" {
			values.Set("host", host)
		}
	case "grpc":
		if proxy.GrpcOpts.GrpcServiceName != "" {
			values.Set("serviceName", proxy.GrpcOpts.GrpcServiceName)
		}
	default:
		if headerType := clashHeaderType(); headerType != "none" {
			values.Set("headerType", headerType)
		}
	}

	u := url.URL{
		Scheme:   "vless",
		User:     url.User(proxy.UUID),
		Host:     fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
		RawQuery: values.Encode(),
		Fragment: proxyDisplayName(proxy),
	}
	return u.String(), nil
}

func clashHostPath(proxy clashProxy) (string, string) {
	host := proxy.Host
	path := proxy.Path

	switch firstNonEmpty(proxy.Network, "tcp") {
	case "ws":
		if proxy.WSOpts.Path != "" {
			path = proxy.WSOpts.Path
		}
		if wsHost := proxy.WSOpts.Headers["Host"]; wsHost != "" {
			host = wsHost
		} else if wsHost := proxy.WSOpts.Headers["host"]; wsHost != "" {
			host = wsHost
		}
	case "http":
		if proxy.HTTPOpts.Path != "" {
			path = proxy.HTTPOpts.Path
		}
		if len(proxy.HTTPOpts.Host) > 0 {
			host = strings.Join(proxy.HTTPOpts.Host, ",")
		}
	case "h2":
		if proxy.H2Opts.Path != "" {
			path = proxy.H2Opts.Path
		}
		if len(proxy.H2Opts.Host) > 0 {
			host = strings.Join(proxy.H2Opts.Host, ",")
		}
	}

	if path == "" && (proxy.Network == "ws" || proxy.Network == "http" || proxy.Network == "h2") {
		path = "/"
	}
	return host, path
}

func clashHeaderType() string {
	return "none"
}

func clashSNI(proxy clashProxy) string {
	return firstNonEmpty(proxy.ServerName, proxy.SNI)
}

func clashSecurity(proxy clashProxy) string {
	security := strings.ToLower(proxy.Security)
	if security != "" && security != "none" {
		return security
	}
	if proxy.RealityOpts.PublicKey != "" {
		return "reality"
	}
	if proxy.TLS {
		return "tls"
	}
	return ""
}

func proxyDisplayName(proxy clashProxy) string {
	if proxy.Name != "" {
		return proxy.Name
	}
	return fmt.Sprintf("%s:%d", proxy.Server, proxy.Port)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
