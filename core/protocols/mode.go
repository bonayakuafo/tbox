package protocols

type Mode string

const (
	ModeVMess        Mode = "VMess"
	ModeVMessAEAD    Mode = "VMessAEAD"
	ModeVLESS        Mode = "VLESS"
	ModeTrojan       Mode = "Trojan"
	ModeShadowSocks  Mode = "ShadowSocks"
	ModeShadowSocksR Mode = "ShadowSocksR"
	ModeHTTP         Mode = "HTTP"
	ModeSocks        Mode = "Socks"
	ModeHysteria2    Mode = "Hysteria2"
	ModeTUIC         Mode = "TUIC"
	ModeAnyTLS       Mode = "AnyTLS"
)

func (m Mode) String() string {
	return string(m)
}
