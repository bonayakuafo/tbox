package protocols

import (
	"tbox/core/protocols/field"
	"bytes"
	"fmt"
	"net/url"
)

type VLess struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	Port    int    `json:"port"`
	Remarks string `json:"remarks"`
	url.Values
}

// GetProtocolMode returns the protocol mode.
func (v *VLess) GetProtocolMode() Mode {
	return ModeVLESS
}

// GetName returns the remarks/alias.
func (v *VLess) GetName() string {
	return v.Remarks
}

// GetAddr returns the remote address.
func (v *VLess) GetAddr() string {
	return v.Address
}

// GetPort returns the remote port.
func (v *VLess) GetPort() int {
	return v.Port
}

// GetInfo returns the node information text.
func (v *VLess) GetInfo() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "Remarks", v.Remarks))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "Address", v.Address))
	buf.WriteString(fmt.Sprintf("%12s: %d\n", "Port", v.Port))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "UserID", v.ID))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "Flow", v.GetValue(field.Flow)))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "Encryption", v.GetValue(field.VLessEncryption)))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "Transport", v.GetValue(field.NetworkType)))
	switch v.GetValue(field.NetworkType) {
	case "tcp":
		buf.WriteString(fmt.Sprintf("%12s: %s\n", "HeaderType", v.GetValue(field.TCPHeaderType)))
	case "kcp":
		buf.WriteString(fmt.Sprintf("%12s: %s\n", "HeaderType", v.GetValue(field.MkcpHeaderType)))
		if v.GetValue(field.Seed) != "" {
			buf.WriteString(fmt.Sprintf("%12s: %s\n", "KCPSeed", v.GetValue(field.Seed)))
		}
	case "ws":
		buf.WriteString(fmt.Sprintf("%12s: %s\n", "Path", v.GetValue(field.WsPath)))
		buf.WriteString(fmt.Sprintf("%12s: %s\n", "Host", v.GetValue(field.WsHost)))
	case "h2":
		buf.WriteString(fmt.Sprintf("%12s: %s\n", "Path", v.GetValue(field.WsPath)))
		buf.WriteString(fmt.Sprintf("%12s: %s\n", "Host", v.GetHostValue(field.WsHost)))
	case "quic":
		buf.WriteString(fmt.Sprintf("%12s: %s\n", "HeaderType", v.GetValue(field.QuicHeaderType)))
		buf.WriteString(fmt.Sprintf("%12s: %s\n", "QUICSecurity", v.GetValue(field.QuicSecurity)))
		if v.GetValue(field.QuicSecurity) != "none" {
			buf.WriteString(fmt.Sprintf("%12s: %s\n", "QUICKey", v.GetValue(field.QuicKey)))
		}
	case "grpc":
		buf.WriteString(fmt.Sprintf("%12s: %s\n", "ServiceName", v.GetValue(field.GrpcServiceName)))
		buf.WriteString(fmt.Sprintf("%12s: %s\n", "GrpcMode", v.GetValue(field.GrpcMode)))
	}
	if v.GetValue(field.Security) == "reality" {
		buf.WriteString(fmt.Sprintf("%12s: %s\n", "FingerPrint", v.GetValue(field.FingerPrint)))
		buf.WriteString(fmt.Sprintf("%12s: %s\n", "RealityPub", v.GetValue(field.PublicKey)))
		buf.WriteString(fmt.Sprintf("%12s: %s\n", "RealityID", v.GetValue(field.ShortId)))
		buf.WriteString(fmt.Sprintf("%12s: %s\n", "RealitySpx", v.GetValue(field.SpiderX)))
	}
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "Security", v.GetValue(field.Security)))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "SNI", v.GetValue(field.SNI)))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "Alpn", v.GetValue(field.Alpn)))
	buf.WriteString(fmt.Sprintf("%12s: %s", "Protocol", v.GetProtocolMode()))

	return buf.String()
}

// GetLink returns the share link.
func (v *VLess) GetLink() string {
	u := url.URL{
		Scheme:   "vless",
		User:     url.User(v.ID),
		Host:     fmt.Sprintf("%s:%d", v.GetAddr(), v.GetPort()),
		RawQuery: v.Values.Encode(),
		Fragment: v.Remarks,
	}
	return u.String()
}

func (v *VLess) GetValue(field field.Field) string {
	if v.Has(field.Key) {
		return v.Get(field.Key)
	}
	return field.Value
}

// GetHostValue returns the H2 Host / SNI value, defaulting to Address if unset.
func (v *VLess) GetHostValue(field field.Field) string {
	if v.Has(field.Key) {
		return v.Get(field.Key)
	}
	return v.Address
}

func (v *VLess) Check() *VLess {
	if v.ID != "" && v.Port > 0 && v.Port <= 65535 && v.Address != "" && v.Remarks != "" {
		return v
	}
	return nil
}
