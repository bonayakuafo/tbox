package cmd

import (
	"tbox/client/config"
	"tbox/cmd/help"
	"tbox/core/manage"
	"tbox/core/protocols"
	"tbox/core/setting"
	"tbox/log"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/olekukonko/tablewriter"
)

func showChainProxySetting() {
	raw := setting.ChainProxy()
	if strings.TrimSpace(raw) == "" {
		log.Info("chain proxy: disabled")
		return
	}

	log.Info("chain proxy: ", raw)
	indexes, err := setting.ChainProxyIndexes()
	if err != nil {
		log.Error(err)
		return
	}

	for _, index := range indexes {
		n := manage.Manager.GetNode(index)
		if n == nil {
			log.Error(fmt.Errorf("chain proxy node not found: index %d", index))
			continue
		}
		log.Info(fmt.Sprintf("[%d] %s %s", index, n.GetProtocolMode(), n.GetName()))
	}
}

func InitSettingShell(shell *ishell.Shell) {
	baseSettingCmd := &ishell.Cmd{
		Name: "setting",
		Func: func(c *ishell.Context) {
			// connection settings
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"socks port", "http port", "udp forward", "traffic sniffing", "allow LAN connections", "mux", "allow insecure"})
			table.SetAlignment(tablewriter.ALIGN_CENTER)
			data := []string{
				strconv.Itoa(setting.Socks()),
				strconv.Itoa(setting.Http()),
				strconv.FormatBool(setting.UDP()),
				strconv.FormatBool(setting.Sniffing()),
				strconv.FormatBool(setting.FromLanConn()),
				strconv.FormatBool(setting.Mux()),
				strconv.FormatBool(setting.AllowInsecure()),
			}
			table.Append(data)
			table.Render()

			// DNS and routing settings
			table = tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"DNS port", "foreign DNS", "domestic DNS", "backup domestic DNS", "routing strategy", "bypass LAN and mainland"})
			table.SetAlignment(tablewriter.ALIGN_CENTER)
			data = []string{
				strconv.Itoa(setting.DNSPort()),
				setting.DNSForeign(),
				setting.DNSDomestic(),
				setting.DNSBackup(),
				setting.RoutingStrategy(),
				strconv.FormatBool(setting.RoutingBypass()),
			}
			table.Append(data)
			table.Render()

			table = tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"test URL", "test timeout (s)", "batch test end time (ms)", "run on startup"})
			table.SetAlignment(tablewriter.ALIGN_CENTER)
			data = []string{
				setting.TestUrl(),
				strconv.Itoa(setting.TestTimeout()),
				strconv.Itoa(setting.TestMinTime()),
				setting.RunBefore(),
			}
			table.Append(data)
			table.Render()

			table = tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"client core", "User-Agent"})
			table.SetAlignment(tablewriter.ALIGN_CENTER)
			data = []string{
				setting.ClientCore(),
				setting.UserAgent(),
			}
			table.Append(data)
			table.Render()

			table = tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"chain proxy", "TUN mode", "TUN auto route", "TUN auto redirect", "config dir"})
			table.SetAlignment(tablewriter.ALIGN_CENTER)
			data = []string{
				setting.ChainProxy(),
				strconv.FormatBool(setting.TunMode()),
				strconv.FormatBool(setting.TunAutoRoute()),
				strconv.FormatBool(setting.TunAutoRedirect()),
				setting.SingboxConfigDir(),
			}
			table.Append(data)
			table.Render()
		},
	}
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name:    "help",
		Aliases: []string{"-h", "--help"},
		Func: func(c *ishell.Context) {
			c.Println(help.Setting)
		},
	})

	// local connection settings
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "socks",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				v, err := strconv.Atoi(c.Args[0])
				if err != nil {
					log.Warn("invalid input")
					return
				}
				err = setting.SetSocks(v)
				if err != nil {
					log.Error(err)
					return
				}
			}
			log.Info("socks port: ", setting.Socks())
		},
	})
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "http",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				v, err := strconv.Atoi(c.Args[0])
				if err != nil {
					log.Error("invalid input")
					return
				}
				err = setting.SetHttp(v)
				if err != nil {
					log.Error(err)
					return
				}
			}
			log.Info("http port: ", setting.Http())
		},
	})
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "udp",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				str := strings.ToLower(c.Args[0])
				switch str {
				case "y", "yes", "true", "t":
					setting.SetUDP(true)
				case "n", "no", "false", "f":
					setting.SetUDP(false)
				}
			}
			log.Info("UDP forward: ", setting.UDP())
		},
	})
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "sniffing",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				str := strings.ToLower(c.Args[0])
				switch str {
				case "y", "yes", "true", "t":
					setting.SetSniffing(true)
				case "n", "no", "false", "f":
					setting.SetSniffing(false)
				}
			}
			log.Info("traffic sniffing: ", setting.Sniffing())
		},
	})
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "mux",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				str := strings.ToLower(c.Args[0])
				switch str {
				case "y", "yes", "true", "t":
					setting.SetMux(true)
				case "n", "no", "false", "f":
					setting.SetMux(false)
				}
			}
			log.Info("mux: ", setting.Mux())
		},
	})
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "allow_insecure",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				str := strings.ToLower(c.Args[0])
				switch str {
				case "y", "yes", "true", "t":
					setting.SetAllowInsecure(true)
				case "n", "no", "false", "f":
					setting.SetAllowInsecure(false)
				}
			}
			log.Info("allow insecure connections: ", setting.AllowInsecure())
		},
	})
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "from_lan_conn",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				str := strings.ToLower(c.Args[0])
				switch str {
				case "y", "yes", "true", "t":
					setting.SetFromLanConn(true)
				case "n", "no", "false", "f":
					setting.SetFromLanConn(false)
				}
			}
			log.Info("allow LAN connections: ", setting.FromLanConn())
		},
	})
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "user_agent",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				err := setting.SetUserAgent(c.Args[0])
				if err != nil {
					log.Warn(err)
					return
				}
			}
			log.Info("User-Agent: ", setting.UserAgent())
		},
	})

	// routing
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "routing.strategy",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				switch c.Args[0] {
				case "1", "AsIs":
					setting.SetRoutingStrategy(1)
				case "2", "RoutingStrategy":
					setting.SetRoutingStrategy(2)
				case "3", "IPOnDemand":
					setting.SetRoutingStrategy(3)
				}
			}
			log.Info("routing strategy: ", setting.RoutingStrategy())
		},
	})
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "routing.bypass",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				str := strings.ToLower(c.Args[0])
				switch str {
				case "y", "yes", "true", "t":
					setting.SetRoutingBypass(true)
				case "n", "no", "false", "f":
					setting.SetRoutingBypass(false)
				}
			}
			log.Info("bypass LAN and mainland: ", setting.RoutingBypass())
		},
	})

	// DNS
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "dns.port",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				v, err := strconv.Atoi(c.Args[0])
				if err != nil {
					log.Error("invalid input")
					return
				}
				err = setting.SetDNSPort(v)
				if err != nil {
					log.Error(err)
					return
				}
			}
			log.Info("DNS port: ", setting.DNSPort())
		},
	})
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "dns.foreign",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				err := setting.SetDNSForeign(c.Args[0])
				if err != nil {
					log.Warn(err)
					return
				}
			}
			log.Info("foreign DNS: ", setting.DNSForeign())
		},
	})
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "dns.domestic",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				err := setting.SetDNSDomestic(c.Args[0])
				if err != nil {
					log.Warn(err)
					return
				}
			}
			log.Info("domestic DNS: ", setting.DNSDomestic())
		},
	})
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "dns.backup",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				err := setting.SetDNSBackup(c.Args[0])
				if err != nil {
					log.Warn(err)
					return
				}
			}
			log.Info("backup domestic DNS: ", setting.DNSBackup())
		},
	})

	// external network test settings
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "test.timeout",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				v, err := strconv.Atoi(c.Args[0])
				if err != nil {
					log.Error("invalid input")
					return
				}
				err = setting.SetTestTimeout(v)
				if err != nil {
					log.Error(err)
					return
				}
			}
			log.Info("test timeout (s): ", setting.TestTimeout())
		},
	})
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "test.mintime",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				v, err := strconv.Atoi(c.Args[0])
				if err != nil {
					log.Error("invalid input")
					return
				}
				err = setting.SetTestMinTime(v)
				if err != nil {
					log.Error(err)
					return
				}
			}
			log.Info("batch test end time (ms): ", setting.TestMinTime())
		},
	})
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "test.url",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				err := setting.SetTestUrl(c.Args[0])
				if err != nil {
					log.Warn(err)
					return
				}
			}
			log.Info("test URL: ", setting.TestUrl())
		},
	})
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "run_before",
		Func: func(c *ishell.Context) {
			argMap := FlagsParse(c.Args, map[string]string{
				"c": "close",
			})
			if _, ok := argMap["close"]; ok {
				err := setting.SetRunBefore("")
				if err != nil {
					log.Warn(err)
					return
				}
			} else if _, ok := argMap["data"]; ok {
				err := setting.SetRunBefore(argMap["data"])
				if err != nil {
					log.Warn(err)
					return
				}
			}
			log.Info("run on startup: ", setting.RunBefore())
		},
	})

	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "chain_proxy",
		Func: func(c *ishell.Context) {
			if len(c.Args) == 0 {
				showChainProxySetting()
				return
			}

			raw := strings.TrimSpace(c.Args[0])
			if strings.EqualFold(raw, "off") {
				err := setting.SetChainProxy("")
				if err != nil {
					log.Error(err)
					return
				}
				showChainProxySetting()
				return
			}

			indexes, err := setting.ParseChainProxyIndexes(raw)
			if err != nil {
				log.Error(err)
				return
			}

			for _, index := range indexes {
				n := manage.Manager.GetNode(index)
				if n == nil {
					log.Error(fmt.Errorf("chain proxy node not found: index %d", index))
					return
				}
				// xhttp nodes must be carried by xray and cannot be a hop in a
				// sing-box chain proxy; reject here to avoid a config that
				// would fail at runtime.
				if config.NodeUsesXhttp(n.Protocol) {
					log.Error(fmt.Errorf("index %d is an xhttp node and cannot be added to the chain proxy", index))
					return
				}
				// SSR nodes are handled by the external ssr converter; the
				// current sing-box has dropped SSR outbound, so they cannot
				// be a hop in a sing-box chain proxy either.
				if n.Protocol.GetProtocolMode() == protocols.ModeShadowSocksR {
					log.Error(fmt.Errorf("index %d is an SSR node and cannot be added to the chain proxy", index))
					return
				}
			}

			err = setting.SetChainProxy(raw)
			if err != nil {
				log.Error(err)
				return
			}

			showChainProxySetting()
		},
	})
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "tun_mode",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				str := strings.ToLower(c.Args[0])
				switch str {
				case "y", "yes", "true", "t":
					setting.SetTunMode(true)
				case "n", "no", "false", "f":
					setting.SetTunMode(false)
				}
			}
			log.Info("TUN mode: ", setting.TunMode())
		},
	})
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "tun_auto_route",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				str := strings.ToLower(c.Args[0])
				switch str {
				case "y", "yes", "true", "t":
					setting.SetTunAutoRoute(true)
				case "n", "no", "false", "f":
					setting.SetTunAutoRoute(false)
				}
			}
			log.Info("TUN auto route: ", setting.TunAutoRoute())
		},
	})
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "tun_auto_redirect",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				str := strings.ToLower(c.Args[0])
				switch str {
				case "y", "yes", "true", "t":
					setting.SetTunAutoRedirect(true)
				case "n", "no", "false", "f":
					setting.SetTunAutoRedirect(false)
				}
			}
			log.Info("TUN auto redirect: ", setting.TunAutoRedirect())
		},
	})
	baseSettingCmd.AddCmd(&ishell.Cmd{
		Name: "config_dir",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				dir := strings.TrimSpace(c.Args[0])
				err := setting.SetSingboxConfigDir(dir)
				if err != nil {
					log.Error(err)
					return
				}
			}
			log.Info("sing-box config dir: ", setting.SingboxConfigDir())
		},
	})
	enrichSettingCompleters(baseSettingCmd)
	shell.AddCmd(baseSettingCmd)
}

// settingMeta describes a one-line help string and completion values for a
// setting subcommand.
type settingMeta struct {
	help    string   // one-line description, shown on tab / setting help
	options []string // candidate values for the first argument (empty = freeform)
}

// yesNo holds candidate values for boolean-style switches.
var yesNo = []string{"y", "n"}

// settingMetas provides help/completion metadata for each setting subcommand.
// Keys must match the Name used above in AddCmd exactly.
var settingMetas = map[string]settingMeta{
	"socks":             {"set socks listen port", nil},
	"http":              {"set http listen port (0 disables; shared with socks under sing-box)", nil},
	"udp":               {"enable UDP forward", yesNo},
	"sniffing":          {"enable traffic sniffing (domain-based routing)", yesNo},
	"mux":               {"enable multiplexing", yesNo},
	"allow_insecure":    {"allow insecure TLS connections", yesNo},
	"from_lan_conn":     {"allow connections from the LAN", yesNo},
	"user_agent":        {"set User-Agent used for subscription requests", nil},
	"routing.strategy":  {"routing strategy 1=AsIs 2=IPIfNonMatch 3=IPOnDemand", []string{"1", "2", "3"}},
	"routing.bypass":    {"bypass LAN and mainland traffic directly", yesNo},
	"dns.port":          {"set DNS listen port", nil},
	"dns.foreign":       {"set foreign DNS (supports tcp:// https:// etc.)", nil},
	"dns.domestic":      {"set domestic DNS", nil},
	"dns.backup":        {"set backup domestic DNS", nil},
	"test.timeout":      {"set external test timeout (seconds)", nil},
	"test.mintime":      {"set batch-test end time (milliseconds)", nil},
	"test.url":          {"set external test URL", nil},
	"run_before":        {"command to run when the program starts (-c to clear)", []string{"-c"}},
	"chain_proxy":       {"set chain proxy node indexes (off to disable)", []string{"off"}},
	"tun_mode":          {"enable TUN mode (Linux only, sing-box)", yesNo},
	"tun_auto_route":    {"whether TUN configures routes automatically", yesNo},
	"tun_auto_redirect": {"whether TUN auto-redirects (nftables)", yesNo},
	"config_dir":        {"set sing-box extra config directory", nil},
}

// enrichSettingCompleters adds one-line help and value completion to the
// already-registered setting subcommands without changing their execution
// logic. This way tab lists the subcommands with descriptions, and typing a
// subcommand then pressing tab suggests candidate values (y/n, core names,
// etc.) for convenience.
func enrichSettingCompleters(baseSettingCmd *ishell.Cmd) {
	for _, sub := range baseSettingCmd.Children() {
		meta, ok := settingMetas[sub.Name]
		if !ok {
			continue
		}
		if sub.Help == "" {
			sub.Help = meta.help
		}
		if len(meta.options) > 0 {
			options := meta.options
			// Only offer completions for the first argument.
			sub.Completer = func(args []string) []string {
				if len(args) > 1 {
					return nil
				}
				return options
			}
		}
	}
}
