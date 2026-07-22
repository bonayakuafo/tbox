package field

type Field struct {
	Key   string // field name
	Value string // default value
}

func NilStrField(key string) Field {
	return Field{
		Key:   key,
		Value: "",
	}
}

func NoneField(key string) Field {
	return Field{
		Key:   key,
		Value: "none",
	}
}

func NewField(key, value string) Field {
	return Field{
		Key:   key,
		Value: value,
	}
}

var (
	NetworkType     Field = NewField("type", "tcp")        // transport, one of tcp/kcp/ws/http/quic/grpc
	VLessEncryption       = NoneField("encryption")        // VLESS encryption, only value: none
	VMessEncryption       = NewField("encryption", "auto") // VMess encryption, one of auto/aes-128-gcm/chacha20-poly1305/none
	TlsSecurity           = NoneField("security")          // underlying transport TLS type, one of none/tls/xtls/reality

	// TCP
	TCPHeaderType = NoneField("headerType")

	// HTTP/2
	H2Path = NewField("path", "/")
	H2Host = NilStrField("host")

	// WebSocket
	WsPath = NewField("path", "/")
	WsHost = NilStrField("host")

	// mKCP
	MkcpHeaderType = NoneField("headerType") // mKCP obfuscation header, one of none/srtp/utp/wechat-video/dtls/wireguard
	Seed           = NilStrField("seed")     // mKCP seed

	// QUIC
	QuicSecurity   = NoneField("quicSecurity") // QUIC encryption, one of none/aes-128-gcm/chacha20-poly1305
	QuicKey        = NilStrField("key")        // QUIC encryption key when quicSecurity != none
	QuicHeaderType = NoneField("headerType")   // QUIC obfuscation header, same values as mKCP headerType

	// gRPC
	GrpcServiceName = NilStrField("serviceName")
	GrpcMode        = NewField("mode", "gun") // gRPC transport mode, one of gun/multi/guna

	// XHTTP (splithttp), xray core only
	XhttpMode = NewField("mode", "auto") // xhttp transport mode, one of auto/packet-up/stream-up/stream-one

	Security = NoneField("security")
	SNI      = NilStrField("sni")  // TLS SNI
	Alpn     = NilStrField("alpn") // alpn, e.g. h2,http/1.1
	Flow     = NilStrField("flow") // XTLS flow, one of xtls-rprx-direct/xtls-rprx-splice

	FingerPrint = NewField("fp", "chrome") // TLS Client Hello fingerprint
	PublicKey   = NilStrField("pbk")       // REALITY public key
	ShortId     = NilStrField("sid")       // REALITY short id
	SpiderX     = NilStrField("spx")       // REALITY spider path
)
