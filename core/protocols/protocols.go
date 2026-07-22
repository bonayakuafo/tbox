package protocols

import (
	"encoding/json"
	"regexp"
)

type Protocol interface {
	// GetProtocolMode returns the protocol mode.
	GetProtocolMode() Mode
	// GetName returns the remarks/alias.
	GetName() string
	// GetAddr returns the remote address.
	GetAddr() string
	// GetPort returns the remote port.
	GetPort() int
	// GetInfo returns the node information text.
	GetInfo() string
	// GetLink returns the share link.
	GetLink() string
}

// Serialize encodes the protocol as "<mode>: <json>".
func Serialize(p Protocol) string {
	jsonData, _ := json.Marshal(p)
	return string(p.GetProtocolMode()) + ": " + string(jsonData)
}

// Deserialize decodes a "<mode>: <json>" line back into a Protocol.
func Deserialize(text string) Protocol {
	expr := "(^[a-zA-Z0-9]*?): ({.*?}$)"
	r, _ := regexp.Compile(expr)
	result := r.FindStringSubmatch(text)
	if len(result) == 3 {
		jsonText := result[2]
		var data Protocol
		switch result[1] {
		case string(ModeVMess):
			data = new(VMess)
		case string(ModeShadowSocks):
			data = new(ShadowSocks)
		case string(ModeShadowSocksR):
			data = new(ShadowSocksR)
		case string(ModeHTTP):
			data = new(HTTP)
		case string(ModeTrojan):
			data = new(Trojan)
		case string(ModeSocks):
			data = new(Socks)
		case string(ModeVLESS):
			data = new(VLess)
		case string(ModeVMessAEAD):
			data = new(VMessAEAD)
		case string(ModeTUIC):
			data = new(TUIC)
		case string(ModeAnyTLS):
			data = new(AnyTLS)
		case string(ModeHysteria2):
			data = new(Hysteria2)
		}
		err := json.Unmarshal([]byte(jsonText), &data)
		if err != nil {
			return nil
		}
		return data
	}
	return nil
}
