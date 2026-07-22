package protocols

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

type VMess struct {
	V          string `json:"v"`
	Ps         string `json:"ps"`
	Add        string `json:"add"`
	Port       int    `json:"port"`
	Id         string `json:"id"`
	Scy        string `json:"scy"`
	Aid        int    `json:"aid"`
	Net        string `json:"net"`
	Type       string `json:"type"`
	Host       string `json:"host"`
	Path       string `json:"path"`
	Tls        string `json:"tls"`
	Sni        string `json:"sni"`
	Alpn       string `json:"alpn"`
	Fp         string `json:"fp"`
	VerifyCert bool   `json:"verifyCert"`
	Class      int    `json:"class"`
}

func (v *VMess) GetProtocolMode() Mode {
	return ModeVMess
}

func (v *VMess) GetName() string {
	return v.Ps
}

func (v *VMess) GetAddr() string {
	return v.Add
}

func (v *VMess) GetPort() int {
	return v.Port
}

func (v *VMess) GetInfo() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "Remarks", v.Ps))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "Address", v.Add))
	buf.WriteString(fmt.Sprintf("%12s: %d\n", "Port", v.Port))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "UserID", v.Id))
	buf.WriteString(fmt.Sprintf("%12s: %d\n", "AlterID", v.Aid))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "Encryption", v.Scy))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "HeaderType", v.Type))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "FakeHost", v.Host))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "Transport", v.Net))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "path", v.Path))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "Security", v.Tls))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "ConfigVer", v.V))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "SNI", v.Sni))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "Alpn", v.Alpn))
	buf.WriteString(fmt.Sprintf("%12s: %s\n", "Fingerprint", v.Fp))
	buf.WriteString(fmt.Sprintf("%12s: %v\n", "VerifyCert", v.VerifyCert))
	buf.WriteString(fmt.Sprintf("%12s: %d\n", "Class", v.Class))
	buf.WriteString(fmt.Sprintf("%12s: %s", "Protocol", v.GetProtocolMode()))
	return buf.String()
}

func (v *VMess) GetLink() string {
	data := map[string]string{
		"v":    v.V,
		"ps":   v.Ps,
		"add":  v.Add,
		"port": strconv.Itoa(v.Port),
		"id":   v.Id,
		"aid":  strconv.Itoa(v.Aid),
		"scy":  v.Scy,
		"net":  v.Net,
		"type": v.Type,
		"host": v.Host,
		"path": v.Path,
		"tls":  v.Tls,
		"sni":  v.Sni,
		"alpn": v.Alpn,
	}
	jsonData, _ := json.Marshal(data)
	return "vmess://" + base64EncodeWithEq(string(jsonData))
}

func (v *VMess) Check() *VMess {
	if v.Add != "" && v.Port > 0 && v.Port <= 65535 && v.Ps != "" && v.Id != "" && v.Net != "" {
		return v
	}
	return nil
}
