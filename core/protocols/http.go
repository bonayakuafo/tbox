package protocols

import "fmt"

type HTTP struct {
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Remarks  string `json:"remarks"`
}

func (h *HTTP) GetProtocolMode() Mode { return ModeHTTP }

func (h *HTTP) GetName() string { return h.Remarks }

func (h *HTTP) GetAddr() string { return h.Address }

func (h *HTTP) GetPort() int { return h.Port }

func (h *HTTP) GetInfo() string { return fmt.Sprintf("%s|%s|%d", h.Remarks, h.Address, h.Port) }

func (h *HTTP) GetLink() string { return "" }

func (h *HTTP) Check() *HTTP {
	if h.Address != "" && h.Port > 0 && h.Port < 65535 && h.Remarks != "" {
		if h.Username == "" && h.Password == "" {
			return h
		}
		if h.Username != "" && h.Password != "" {
			return h
		}
	}
	return nil
}
