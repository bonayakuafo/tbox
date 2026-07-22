package config

import (
	"net/url"
	"strings"

	"tbox/core/setting"
)

// tlsOptions describes the parameters needed to generate a sing-box tls config.
type tlsOptions struct {
	ServerName string
	Alpn       string // comma-separated, may be empty
	Insecure   bool
	// reality
	Reality     bool
	PublicKey   string
	ShortID     string
	Fingerprint string
}

// buildTLS generates a sing-box tls config block from tlsOptions.
// insecure is always written explicitly as a bool (insecure:false is
// equivalent to omitting it).
func buildTLS(opts tlsOptions) map[string]interface{} {
	tls := map[string]interface{}{
		"enabled":  true,
		"insecure": opts.Insecure,
	}
	if opts.ServerName != "" {
		tls["server_name"] = opts.ServerName
	}
	if alpnList := splitCSV(opts.Alpn); len(alpnList) > 0 {
		tls["alpn"] = alpnList
	}
	if opts.Reality {
		reality := map[string]interface{}{
			"enabled":    true,
			"public_key": opts.PublicKey,
		}
		if opts.ShortID != "" {
			reality["short_id"] = opts.ShortID
		}
		tls["reality"] = reality

		fingerprint := opts.Fingerprint
		if fingerprint == "" {
			fingerprint = "chrome"
		}
		tls["utls"] = map[string]interface{}{
			"enabled":     true,
			"fingerprint": fingerprint,
		}
	}
	return tls
}

// transportOptions describes the parameters needed to generate a sing-box transport config.
type transportOptions struct {
	Network     string
	Path        string
	Host        string
	ServiceName string
}

// buildStreamTransport generates a sing-box transport config from the transport
// type; returns nil for tcp/unknown types.
func buildStreamTransport(opts transportOptions) map[string]interface{} {
	switch opts.Network {
	case "ws":
		transport := map[string]interface{}{
			"type": "ws",
		}
		if opts.Path != "" {
			transport["path"] = opts.Path
		}
		if opts.Host != "" {
			transport["headers"] = map[string]interface{}{
				"Host": opts.Host,
			}
		}
		return transport
	case "h2":
		transport := map[string]interface{}{
			"type": "http",
		}
		if opts.Path != "" {
			transport["path"] = opts.Path
		}
		if hostList := splitCSV(opts.Host); len(hostList) > 0 {
			transport["host"] = hostList
		}
		return transport
	case "grpc":
		return map[string]interface{}{
			"type":         "grpc",
			"service_name": opts.ServiceName,
		}
	case "quic":
		return map[string]interface{}{
			"type": "quic",
		}
	default:
		return nil
	}
}

// isTruthyParam reports whether a boolean-ish query-string param is true (1/true/yes).
func isTruthyParam(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes":
		return true
	}
	return false
}

// resolveInsecure decides whether to skip TLS certificate verification:
// if the share link explicitly sets insecure / allowInsecure / skip-cert-verify
// to a truthy value, skip; otherwise fall back to the global allow_insecure
// setting. Different protocols name the parameter differently in their share
// links; this handles all common spellings.
func resolveInsecure(values url.Values) bool {
	for _, key := range []string{"insecure", "allowInsecure", "allow_insecure", "skip-cert-verify"} {
		if values.Has(key) && isTruthyParam(values.Get(key)) {
			return true
		}
	}
	return setting.AllowInsecure()
}

// splitCSV splits a comma-separated string, trims whitespace, and drops empty items.
func splitCSV(s string) []string {
	result := make([]string, 0)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
