package protocols

import (
	"bytes"
	"fmt"
	"net/url"
)

type Trojan struct {
	url.Values
	Password string `json:"password"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Remarks  string `json:"remarks"`
}

func (t *Trojan) GetProtocolMode() Mode {
	return ModeTrojan
}

func (t *Trojan) GetName() string {
	return t.Remarks
}
func (t *Trojan) GetAddr() string {
	return t.Address
}

func (t *Trojan) GetPort() int {
	return t.Port
}

func (t *Trojan) GetInfo() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%8s: %s\n", "Remarks", t.Remarks))
	buf.WriteString(fmt.Sprintf("%8s: %s\n", "Address", t.Address))
	buf.WriteString(fmt.Sprintf("%8s: %d\n", "Port", t.Port))
	buf.WriteString(fmt.Sprintf("%8s: %s\n", "Password", t.Password))
	buf.WriteString(fmt.Sprintf("%8s: %s\n", "SNI", t.Sni()))
	buf.WriteString(fmt.Sprintf("%8s: %s", "Protocol", t.GetProtocolMode()))
	return buf.String()
}

func (t *Trojan) GetLink() string {
	u := &url.URL{
		Scheme:   "trojan",
		Host:     fmt.Sprintf("%s:%d", t.Address, t.Port),
		Fragment: t.Remarks,
		User:     url.User(t.Password),
		RawQuery: t.Values.Encode(),
	}
	return u.String()
}

func (t *Trojan) Sni() string {
	if t.Has("sni") {
		return t.Get("sni")
	}
	return ""
}

func (t *Trojan) Check() *Trojan {
	if t.Password != "" && t.Address != "" && t.Port > 0 && t.Port < 65535 && t.Remarks != "" {
		return t
	}
	return nil
}
