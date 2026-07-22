package protocols

import (
	"bytes"
	"fmt"
	"net/url"
)

type TUIC struct {
	url.Values
	UUID     string `json:"uuid"`
	Password string `json:"password"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Remarks  string `json:"remarks"`
}

func (t *TUIC) GetProtocolMode() Mode {
	return ModeTUIC
}

func (t *TUIC) GetName() string {
	return t.Remarks
}

func (t *TUIC) GetAddr() string {
	return t.Address
}

func (t *TUIC) GetPort() int {
	return t.Port
}

func (t *TUIC) GetInfo() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%8s: %s\n", "Remarks", t.Remarks))
	buf.WriteString(fmt.Sprintf("%8s: %s\n", "Address", t.Address))
	buf.WriteString(fmt.Sprintf("%8s: %d\n", "Port", t.Port))
	buf.WriteString(fmt.Sprintf("%8s: %s\n", "UUID", t.UUID))
	buf.WriteString(fmt.Sprintf("%8s: %s\n", "Password", t.Password))
	buf.WriteString(fmt.Sprintf("%8s: %s\n", "SNI", t.Sni()))
	buf.WriteString(fmt.Sprintf("%8s: %s", "Protocol", t.GetProtocolMode()))
	return buf.String()
}

func (t *TUIC) GetLink() string {
	u := &url.URL{
		Scheme:   "tuic",
		Host:     fmt.Sprintf("%s:%d", t.Address, t.Port),
		Fragment: t.Remarks,
		User:     url.UserPassword(t.UUID, t.Password),
		RawQuery: t.Values.Encode(),
	}
	return u.String()
}

func (t *TUIC) Sni() string {
	if t.Has("sni") {
		return t.Get("sni")
	}
	return ""
}

func (t *TUIC) Check() *TUIC {
	if t.UUID != "" && t.Password != "" && t.Address != "" && t.Port > 0 && t.Port < 65535 && t.Remarks != "" {
		return t
	}
	return nil
}
