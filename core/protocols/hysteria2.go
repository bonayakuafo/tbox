package protocols

import (
	"bytes"
	"fmt"
	"net/url"
)

type Hysteria2 struct {
	url.Values
	Password string `json:"password"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Remarks  string `json:"remarks"`
}

func (t *Hysteria2) GetProtocolMode() Mode {
	return ModeHysteria2
}

func (t *Hysteria2) GetName() string {
	return t.Remarks
}
func (t *Hysteria2) GetAddr() string {
	return t.Address
}

func (t *Hysteria2) GetPort() int {
	return t.Port
}

func (t *Hysteria2) GetInfo() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%8s: %s\n", "Remarks", t.Remarks))
	buf.WriteString(fmt.Sprintf("%8s: %s\n", "Address", t.Address))
	buf.WriteString(fmt.Sprintf("%8s: %d\n", "Port", t.Port))
	buf.WriteString(fmt.Sprintf("%8s: %s\n", "Password", t.Password))
	buf.WriteString(fmt.Sprintf("%8s: %s\n", "SNI", t.Sni()))
	buf.WriteString(fmt.Sprintf("%8s: %s", "Protocol", t.GetProtocolMode()))
	return buf.String()
}

func (t *Hysteria2) GetLink() string {
	u := &url.URL{
		Scheme:   "hysteria2",
		Host:     fmt.Sprintf("%s:%d", t.Address, t.Port),
		Fragment: t.Remarks,
		User:     url.User(t.Password),
		RawQuery: t.Values.Encode(),
	}
	return u.String()
}

func (t *Hysteria2) Sni() string {
	if t.Has("sni") {
		return t.Get("sni")
	}
	return ""
}

func (t *Hysteria2) Check() *Hysteria2 {
	if t.Password != "" && t.Address != "" && t.Port > 0 && t.Port < 65535 && t.Remarks != "" {
		return t
	}
	return nil
}
