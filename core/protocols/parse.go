package protocols

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// ParseLink parses a share link.
func ParseLink(link string) Protocol {
	u, err := url.Parse(link)
	if err != nil {
		return nil
	}
	switch u.Scheme {
	case "vmess":
		if obj := ParseVMessLink(link); obj != nil {
			return obj
		}
		if obj := ParseVMessAEADLink(link); obj != nil {
			return obj
		}
	case "vless":
		if obj := ParseVLessLink(link); obj != nil {
			return obj
		}
	case "ss":
		if obj := ParseSSLink(link); obj != nil {
			return obj
		}
	case "ssr":
		if obj := ParseSSRLink(link); obj != nil {
			return obj
		}
	case "trojan":
		if obj := ParseTrojanLink(link); obj != nil {
			return obj
		}
	case "hysteria2", "hy2":
		if obj := ParseHysteria2Link(link); obj != nil {
			return obj
		}
	case "socks", "socks5":
		if obj := ParseSocksLink(link); obj != nil {
			return obj
		}
	case "http":
		if obj := ParseHTTPLink(link); obj != nil {
			return obj
		}
	case "tuic":
		if obj := ParseTUICLink(link); obj != nil {
			return obj
		}
	case "anytls":
		if obj := ParseAnyTLSLink(link); obj != nil {
			return obj
		}
	}
	return nil
}

func ParseVMessAEADLink(link string) *VMessAEAD {
	u, err := url.Parse(link)
	if err != nil {
		return nil
	}
	if u.Scheme != "vmess" {
		return nil
	}
	vless := new(VMessAEAD)
	vless.Address = u.Hostname()
	vless.Port, err = strconv.Atoi(u.Port())
	if err != nil {
		return nil
	}
	if u.User == nil {
		return nil
	}
	vless.ID = u.User.Username()
	vless.Remarks = u.Fragment
	vless.Values = u.Query()
	if vless.Remarks == "" {
		vless.Remarks = u.Host
	}
	return vless.Check()
}

func ParseVLessLink(link string) *VLess {
	u, err := url.Parse(link)
	if err != nil {
		return nil
	}
	if u.Scheme != "vless" {
		return nil
	}
	vless := new(VLess)
	vless.Address = u.Hostname()
	vless.Port, err = strconv.Atoi(u.Port())
	if err != nil {
		return nil
	}
	if u.User == nil {
		return nil
	}
	vless.ID = u.User.Username()
	vless.Remarks = u.Fragment
	vless.Values = u.Query()
	if vless.Remarks == "" {
		vless.Remarks = u.Host
	}
	return vless.Check()
}

func ParseSocksLink(link string) *Socks {
	u, err := url.Parse(link)
	if err != nil {
		return nil
	}
	if u.Scheme != "socks" && u.Scheme != "socks5" {
		return nil
	}
	socks := new(Socks)

	socks.Address = u.Hostname()
	socks.Port, err = strconv.Atoi(u.Port())
	if err != nil {
		return nil
	}
	if u.User != nil {
		socks.Username = u.User.Username()
		socks.Password, _ = u.User.Password()
	}
	socks.Remarks = u.Fragment
	if socks.Remarks == "" {
		socks.Remarks = u.Host
	}
	return socks.Check()
}

func ParseHTTPLink(link string) *HTTP {
	u, err := url.Parse(link)
	if err != nil {
		return nil
	}
	if u.Scheme != "http" {
		return nil
	}
	http := new(HTTP)
	http.Address = u.Hostname()
	http.Port, err = strconv.Atoi(u.Port())
	if err != nil {
		return nil
	}
	if u.User != nil {
		http.Username = u.User.Username()
		http.Password, _ = u.User.Password()
	}
	http.Remarks = u.Fragment
	if http.Remarks == "" {
		http.Remarks = u.Host
	}
	return http.Check()
}

func ParseVMessLink(link string) *VMess {
	vmess := new(VMess)
	if strings.ToLower(link[:8]) == "vmess://" {
		link = link[8:]
	} else {
		return nil
	}
	if len(link) == 0 {
		return nil
	}
	jsonStr := base64Decode(link)
	if jsonStr == "" {
		return nil
	}
	var mapResult map[string]any
	err := json.Unmarshal([]byte(jsonStr), &mapResult)
	if err != nil {
		return nil
	}
	if version, ok := mapResult["v"]; ok {
		vmess.V = fmt.Sprintf("%v", version)
	}
	if ps, ok := mapResult["ps"]; ok {
		vmess.Ps = fmt.Sprintf("%v", ps)
	} else {
		return nil
	}
	if addr, ok := mapResult["add"]; ok {
		vmess.Add = fmt.Sprintf("%v", addr)
	} else {
		return nil
	}
	if scy, ok := mapResult["scy"]; ok {
		vmess.Scy = fmt.Sprintf("%v", scy)
	} else {
		vmess.Scy = "auto"
	}
	if port, ok := mapResult["port"]; ok {
		value, err := strconv.Atoi(fmt.Sprintf("%v", port))
		if err == nil {
			vmess.Port = value
		} else {
			return nil
		}
	} else {
		return nil
	}

	if id, ok := mapResult["id"]; ok {
		vmess.Id = fmt.Sprintf("%v", id)
	} else {
		return nil
	}
	if aid, ok := mapResult["aid"]; ok {
		if value, err := strconv.Atoi(fmt.Sprintf("%v", aid)); err == nil {
			vmess.Aid = value
		} else {
			return nil
		}
	} else {
		return nil
	}
	if net, ok := mapResult["net"]; ok {
		vmess.Net = fmt.Sprintf("%v", net)
	} else {
		return nil
	}
	if type1, ok := mapResult["type"]; ok {
		vmess.Type = fmt.Sprintf("%v", type1)
	} else {
		return nil
	}
	if host, ok := mapResult["host"]; ok {
		vmess.Host = fmt.Sprintf("%v", host)
	} else {
		return nil
	}
	if path, ok := mapResult["path"]; ok {
		vmess.Path = fmt.Sprintf("%v", path)
	} else {
		return nil
	}
	if tls, ok := mapResult["tls"]; ok {
		vmess.Tls = fmt.Sprintf("%v", tls)
	} else {
		return nil
	}
	if sni, ok := mapResult["sni"]; ok {
		vmess.Sni = fmt.Sprintf("%v", sni)
	}
	if _, ok := mapResult["alpn"]; ok {
		// Some nodes fail to connect when alpn is set, so this is intentionally left commented out
		// vmess.Alpn = fmt.Sprintf("%v", alpn)
		vmess.Alpn = ""
	}
	return vmess.Check()
}

func ParseTrojanLink(link string) *Trojan {
	u, err := url.Parse(link)
	if err != nil {
		return nil
	}
	if u.Scheme != "trojan" {
		return nil
	}
	trojan := new(Trojan)
	if u.User == nil {
		return nil
	}
	trojan.Password = u.User.Username()
	trojan.Address = u.Hostname()
	trojan.Port, err = strconv.Atoi(u.Port())
	if err != nil {
		return nil
	}
	trojan.Remarks = u.Fragment
	if trojan.Remarks == "" {
		trojan.Remarks = u.Host
	}
	trojan.Values = u.Query()
	return trojan.Check()
}

func ParseSSRLink(link string) *ShadowSocksR {
	ssr := new(ShadowSocksR)
	if strings.ToLower(link[:6]) == "ssr://" {
		link = link[6:]
	} else {
		return nil
	}
	link = base64Decode(link)
	if link == "" {
		return nil
	}
	expr := `^([a-zA-Z0-9-\.]*):([0-9]{1,5}):([a-z0-9_]*):([a-z0-9-]*):([a-z0-9_\.]*):([a-zA-Z0-9-_=]*)(/.(.*$))?`
	r, _ := regexp.Compile(expr)
	result := r.FindStringSubmatch(link)
	if len(result) != 9 {
		return nil
	}
	ssr.Address = result[1]
	port, err := strconv.Atoi(result[2])
	if err != nil {
		return nil
	}
	ssr.Port = port
	if ssr.Port < 0 || ssr.Port > 65535 {
		return nil
	}
	ssr.Protocol = result[3]
	ssr.Method = result[4]
	ssr.Obfs = result[5]
	ssr.Password = base64Decode(result[6])
	if ssr.Password == "" {
		return nil
	}
	for _, str := range strings.Split(result[8], "&") {
		if strings.HasPrefix(str, "obfsparam=") {
			ssr.ObfsParam = base64Decode(str[10:])
		} else if strings.HasPrefix(str, "protoparam=") {
			ssr.ProtoParam = base64Decode(str[11:])
		} else if strings.HasPrefix(str, "remarks=") {
			ssr.Remarks = base64Decode(str[8:])
		} else if strings.HasPrefix(str, "group=") {
			ssr.Group = base64Decode(str[6:])
		}
	}
	if ssr.Remarks == "" {
		ssr.Remarks = ssr.Address + ":" + strconv.Itoa(ssr.Port)
	}
	return ssr
}

func ParseSSLink(link string) *ShadowSocks {
	u, err := url.Parse(link)
	if err != nil {
		return nil
	}
	if u.Scheme != "ss" {
		return nil
	}
	ss := new(ShadowSocks)
	ss.Remarks = u.Fragment
	if u.User == nil {
		u, err = url.Parse("ss://" + base64Decode(u.Host))
		if err != nil {
			return nil
		}
		ss.Address = u.Hostname()
		ss.Port, err = strconv.Atoi(u.Port())
		if err != nil {
			return nil
		}
		ss.Method = u.User.Username()
		ss.Password, _ = u.User.Password()
		if ss.Remarks == "" {
			ss.Remarks = u.Host
		}
	} else {
		ss.Address = u.Hostname()
		ss.Port, err = strconv.Atoi(u.Port())
		if err != nil {
			return nil
		}
		if ss.Remarks == "" {
			ss.Remarks = u.Host
		}
		result := strings.SplitN(base64Decode(u.User.Username()), ":", 2)
		ss.Method = result[0]
		ss.Password = result[1]
		ss.Values = u.Query()
	}
	return ss.Check()
}

func ParseHysteria2Link(link string) *Hysteria2 {
	u, err := url.Parse(link)
	if err != nil {
		return nil
	}
	if u.Scheme != "hysteria2" && u.Scheme != "hy2" {
		return nil
	}
	hysteria2 := new(Hysteria2)
	if u.User == nil {
		return nil
	}
	hysteria2.Password = u.User.Username()
	hysteria2.Address = u.Hostname()
	hysteria2.Port, err = strconv.Atoi(u.Port())
	if err != nil {
		return nil
	}
	hysteria2.Remarks = u.Fragment
	if hysteria2.Remarks == "" {
		hysteria2.Remarks = u.Host
	}
	hysteria2.Values = u.Query()
	return hysteria2.Check()
}

func ParseTUICLink(link string) *TUIC {
	u, err := url.Parse(link)
	if err != nil {
		return nil
	}
	if u.Scheme != "tuic" {
		return nil
	}
	tuic := new(TUIC)
	if u.User == nil {
		return nil
	}
	tuic.UUID = u.User.Username()
	tuic.Password, _ = u.User.Password()
	tuic.Address = u.Hostname()
	tuic.Port, err = strconv.Atoi(u.Port())
	if err != nil {
		return nil
	}
	tuic.Remarks = u.Fragment
	if tuic.Remarks == "" {
		tuic.Remarks = u.Host
	}
	tuic.Values = u.Query()
	return tuic.Check()
}

func ParseAnyTLSLink(link string) *AnyTLS {
	u, err := url.Parse(link)
	if err != nil {
		return nil
	}
	if u.Scheme != "anytls" {
		return nil
	}
	anytls := new(AnyTLS)
	if u.User == nil {
		return nil
	}
	anytls.Password = u.User.Username()
	anytls.Address = u.Hostname()
	anytls.Port, err = strconv.Atoi(u.Port())
	if err != nil {
		return nil
	}
	anytls.Remarks = u.Fragment
	if anytls.Remarks == "" {
		anytls.Remarks = u.Host
	}
	anytls.Values = u.Query()
	return anytls.Check()
}
