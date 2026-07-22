package protocols

import (
	"bytes"
	"fmt"
	"net/url"
)

type AnyTLS struct {
	url.Values
	Password string `json:"password"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Remarks  string `json:"remarks"`
}

func (a *AnyTLS) GetProtocolMode() Mode {
	return ModeAnyTLS
}

func (a *AnyTLS) GetName() string {
	return a.Remarks
}

func (a *AnyTLS) GetAddr() string {
	return a.Address
}

func (a *AnyTLS) GetPort() int {
	return a.Port
}

func (a *AnyTLS) GetInfo() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%8s: %s\n", "Remarks", a.Remarks))
	buf.WriteString(fmt.Sprintf("%8s: %s\n", "Address", a.Address))
	buf.WriteString(fmt.Sprintf("%8s: %d\n", "Port", a.Port))
	buf.WriteString(fmt.Sprintf("%8s: %s\n", "Password", a.Password))
	buf.WriteString(fmt.Sprintf("%8s: %s\n", "SNI", a.Sni()))
	buf.WriteString(fmt.Sprintf("%8s: %s", "Protocol", a.GetProtocolMode()))
	return buf.String()
}

func (a *AnyTLS) GetLink() string {
	u := &url.URL{
		Scheme:   "anytls",
		Host:     fmt.Sprintf("%s:%d", a.Address, a.Port),
		Fragment: a.Remarks,
		User:     url.User(a.Password),
		RawQuery: a.Values.Encode(),
	}
	return u.String()
}

func (a *AnyTLS) Sni() string {
	if a.Has("sni") {
		return a.Get("sni")
	}
	return ""
}

func (a *AnyTLS) Check() *AnyTLS {
	if a.Password != "" && a.Address != "" && a.Port > 0 && a.Port < 65535 && a.Remarks != "" {
		return a
	}
	return nil
}
