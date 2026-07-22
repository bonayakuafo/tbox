package key

const (
	Socks         = "socks"
	Http          = "http"
	UDP           = "udp"
	Sniffing      = "sniffing"
	FromLanConn   = "from_lan_conn"
	Mux           = "mux"
	AllowInsecure = "allow_insecure"
	UserAgent     = "user_agent"

	RoutingStrategy = "routing.strategy"
	RoutingBypass   = "routing.bypass" // bypass LAN and mainland

	DNSPort     = "dns.port"
	DNSDomestic = "dns.domestic" // domestic dns
	DNSForeign  = "dns.foreign"  // foreign dns
	DNSBackup   = "dns.backup"   // backup dns

	TestURL     = "test.url"
	TestTimeout = "test.timeout"
	TestMinTime = "test.mintime"

	RunBefore = "run_before"
	PID       = "pid"        // core process id
	BridgePID = "bridge_pid" // xray converter process id in TUN+xray bridge mode

	ClientCore = "client.core"

	ChainProxy       = "singbox.chain_proxy"
	TunMode          = "singbox.tun_mode"
	TunAutoRoute     = "singbox.tun_auto_route"
	TunAutoRedirect  = "singbox.tun_auto_redirect"
	SingboxConfigDir = "singbox.config_dir"
)
