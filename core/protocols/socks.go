package protocols

import (
	"bytes"
	"fmt"
	"net/url"
)

type Socks struct {
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Remarks  string `json:"remarks"`
}

// GetProtocolMode returns the protocol mode.
func (s *Socks) GetProtocolMode() Mode {
	return ModeSocks
}

// GetName returns the remarks/alias.
func (s *Socks) GetName() string {
	return s.Remarks
}

// GetAddr returns the remote address.
func (s *Socks) GetAddr() string {
	return s.Address
}

// GetPort returns the remote port.
func (s *Socks) GetPort() int {
	return s.Port
}

// GetInfo returns the node information text.
func (s *Socks) GetInfo() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%8s: %s\n", "Remarks", s.Remarks)
	fmt.Fprintf(&buf, "%8s: %s\n", "Address", s.Address)
	fmt.Fprintf(&buf, "%8s: %d\n", "Port", s.Port)
	fmt.Fprintf(&buf, "%8s: %s\n", "Username", s.Username)
	fmt.Fprintf(&buf, "%8s: %s\n", "Password", s.Password)
	fmt.Fprintf(&buf, "%8s: %s", "Protocol", s.GetProtocolMode())
	return buf.String()
}

// GetLink returns the share link.
func (s *Socks) GetLink() string {
	u := url.URL{
		Scheme:   "socks5",
		Host:     fmt.Sprintf("%s:%d", s.Address, s.Port),
		Fragment: s.Remarks,
	}
	if s.Username != "" && s.Password != "" {
		u.User = url.UserPassword(s.Username, s.Password)
	}
	return u.String()
}

func (s *Socks) Check() *Socks {
	if s.Address != "" && s.Port > 0 && s.Port < 65535 && s.Remarks != "" {
		if s.Username == "" && s.Password == "" {
			return s
		}
		if s.Username != "" && s.Password != "" {
			return s
		}
	}
	return nil
}
