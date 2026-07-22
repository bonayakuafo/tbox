package cmd

import (
	"tbox/cmd/help"
	"tbox/core"
	"tbox/core/manage"
	"tbox/core/node"
	"tbox/core/protocols"
	"tbox/core/sub"
	"tbox/log"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/atotto/clipboard"
	"github.com/olekukonko/tablewriter"
)

func getSubNameByID(subID string) string {
	if subID == "" {
		return ""
	}
	var name string
	manage.Manager.SubForEach(func(i int, s *sub.Subscirbe) {
		if s.ID() == subID {
			name = s.Name
		}
	})
	if name == "" {
		return "(unknown subscription)"
	}
	return name
}

func InitNodeShell(shell *ishell.Shell) {
	nodeCmd := &ishell.Cmd{
		Name: "node",
		Func: func(c *ishell.Context) {
			argMap := FlagsParse(c.Args, map[string]string{
				"d": "desc",
			})
			key := "all"
			if _, ok := argMap["data"]; ok {
				key = argMap["data"]
			}
			_, isDesc := argMap["desc"]
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"index", "protocol", "alias", "address", "port", "test result", "subscription"})
			table.SetAlignment(tablewriter.ALIGN_CENTER)
			center := tablewriter.ALIGN_CENTER
			left := tablewriter.ALIGN_LEFT
			table.SetColumnAlignment([]int{center, center, left, center, center, center, center})
			table.SetColWidth(70)
			indexList := core.IndexList(key, manage.Manager.NodeLen())
			if isDesc {
				indexList = core.Reverse(indexList)
			}
			for _, index := range indexList {
				n := manage.Manager.GetNode(index)
				if n != nil {
					subName := getSubNameByID(n.SubID)
					table.Append([]string{
						strconv.Itoa(index),
						string(n.GetProtocolMode()),
						n.GetName(),
						n.GetAddr(),
						strconv.Itoa(n.GetPort()),
						n.TestResultStr(),
						subName,
					})
				}
			}
			if n := manage.Manager.SelectedNode(); n == nil {
				table.SetCaption(true, fmt.Sprintf("[ %d/%d ] %s",
					manage.Manager.SelectedIndex(),
					manage.Manager.NodeLen(),
					"no node",
				))
			} else {
				table.SetCaption(true, fmt.Sprintf("[ %d/%d ] %s",
					manage.Manager.SelectedIndex(),
					manage.Manager.NodeLen(),
					n.GetName(),
				))
			}
			table.Render()
		},
	}
	// help
	nodeCmd.AddCmd(&ishell.Cmd{
		Name:    "help",
		Aliases: []string{"-h", "--help"},
		Help:    "show help",
		Func: func(c *ishell.Context) {
			c.Println(help.Node)
		},
	})
	// tcping
	nodeCmd.AddCmd(&ishell.Cmd{
		Name: "tcping",
		Func: func(c *ishell.Context) {
			manage.Manager.Tcping()
			_ = shell.Process("node", "-d")
		},
	})
	// info
	nodeCmd.AddCmd(&ishell.Cmd{
		Name: "info",
		Func: func(c *ishell.Context) {
			if len(c.Args) == 1 {
				v, err := strconv.Atoi(c.Args[0])
				if err != nil {
					log.Warn("invalid argument")
					return
				}
				n := manage.Manager.GetNode(v)
				if n == nil {
					log.Warn("no such node")
					return
				}
				n.Show()
			}
		},
	})
	// rm
	nodeCmd.AddCmd(&ishell.Cmd{
		Name: "rm",
		Func: func(c *ishell.Context) {
			if len(c.Args) == 1 {
				indexes := core.IndexList(c.Args[0], manage.Manager.NodeLen())
				if len(indexes) == 0 {
					log.Warn("no matching nodes to delete")
					return
				}
				if !confirm(c, fmt.Sprintf("about to delete %d nodes, confirm?", len(indexes))) {
					log.Info("cancelled")
					return
				}
				manage.Manager.DelNode(c.Args[0])
			}
		},
	})
	// sort
	nodeCmd.AddCmd(&ishell.Cmd{
		Name: "sort",
		Func: func(c *ishell.Context) {
			if len(c.Args) == 1 {
				var mode int
				switch c.Args[0] {
				case "0":
					mode = 0
				case "1":
					mode = 1
				case "2":
					mode = 2
				case "3":
					mode = 3
				case "4":
					mode = 4
				case "5":
					mode = 5
				default:
					return
				}
				manage.Manager.Sort(mode)
				_ = shell.Process("node")
			}
		},
	})
	// export
	nodeCmd.AddCmd(&ishell.Cmd{
		Name: "export",
		Func: func(c *ishell.Context) {
			argMap := FlagsParse(c.Args, map[string]string{
				"c": "clipboard",
			})
			key := "all"
			if _, ok := argMap["data"]; ok {
				key = argMap["data"]
			}
			links := manage.Manager.GetNodeLink(key)
			if _, ok := argMap["clipboard"]; ok {
				err := clipboard.WriteAll(strings.Join(links, "\n"))
				if err != nil {
					log.Error(err)
					return
				}
				c.Println("exported", len(links), "entries to clipboard")
			} else {
				c.Println("========================================================================")
				c.Println(strings.Join(links, "\n"))
				c.Println("========================================================================")
				c.Println("exported", len(links), "entries")
			}
		},
	})
	// find
	nodeCmd.AddCmd(&ishell.Cmd{
		Name: "find",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				return
			}
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"index", "protocol", "alias", "address", "port", "test result", "subscription"})
			table.SetAlignment(tablewriter.ALIGN_CENTER)
			center := tablewriter.ALIGN_CENTER
			left := tablewriter.ALIGN_LEFT
			table.SetColumnAlignment([]int{center, center, left, center, center, center, center})
			table.SetColWidth(70)
			manage.Manager.NodeForEach(func(i int, n *node.Node) {
				if n != nil && strings.Contains(n.GetName(), c.Args[0]) {
					subName := getSubNameByID(n.SubID)
					defer table.Append([]string{
						strconv.Itoa(i),
						string(n.GetProtocolMode()),
						n.GetName(),
						n.GetAddr(),
						strconv.Itoa(n.GetPort()),
						n.TestResultStr(),
						subName,
					})
				}
			})
			if n := manage.Manager.SelectedNode(); n == nil {
				table.SetCaption(true, fmt.Sprintf("[ %d/%d ] %s",
					manage.Manager.SelectedIndex(),
					manage.Manager.NodeLen(),
					"no node",
				))
			} else {
				table.SetCaption(true, fmt.Sprintf("[ %d/%d ] %s",
					manage.Manager.SelectedIndex(),
					manage.Manager.NodeLen(),
					n.GetName(),
				))
			}
			table.Render()
		},
	})
	// add
	nodeCmd.AddCmd(&ishell.Cmd{
		Name: "add",
		Func: func(c *ishell.Context) {
			argMap := FlagsParse(c.Args, map[string]string{
				"l": "link",
				"c": "clipboard",
				"f": "file",
			})
			// import nodes from clipboard
			if _, ok := argMap["clipboard"]; ok {
				content, err := clipboard.ReadAll()
				if err != nil {
					log.Error(err)
					return
				}
				content = strings.ReplaceAll(content, "\r\n", "\n")
				content = strings.ReplaceAll(content, "\r", "\n")
				c.Println("clipboard content:")
				c.Println("========================================================================")
				c.Println(content)
				c.Println("========================================================================")
				if strings.Contains(content, "://") {
					for _, link := range strings.Split(content, "\n") {
						manage.Manager.AddNode(node.NewNode(link, "default"))
					}
				} else {
					for _, link := range sub.Sub2links(content) {
						manage.Manager.AddNode(node.NewNode(link, "default"))
					}
				}
			} else if fileArg, ok := argMap["file"]; ok {
				if _, err := os.Stat(fileArg); os.IsNotExist(err) {
					log.Error("open ", fileArg, " : no such file")
					return
				}
				data, _ := os.ReadFile(fileArg)
				content := strings.ReplaceAll(string(data), "\r\n", "\n")
				content = strings.ReplaceAll(content, "\r", "\n")
				c.Println("file content:")
				c.Println("========================================================================")
				c.Println(content)
				c.Println("========================================================================")
				if strings.Contains(content, "://") {
					for _, link := range strings.Split(content, "\n") {
						manage.Manager.AddNode(node.NewNode(link, "default"))
					}
				} else {
					for _, link := range sub.Sub2links(content) {
						manage.Manager.AddNode(node.NewNode(link, "default"))
					}
				}
			} else if linkArg, ok := argMap["link"]; ok {
				manage.Manager.AddNode(node.NewNode(linkArg, "default"))
			} else if dataArg, ok := argMap["data"]; ok && strings.Contains(dataArg, "://") {
				manage.Manager.AddNode(node.NewNode(dataArg, "default"))
			} else {
				c.ShowPrompt(false)
				defer c.ShowPrompt(true)
				modeList := []string{
					protocols.ModeVMess.String(),
					protocols.ModeVLESS.String(),
					protocols.ModeVMessAEAD.String(),
					protocols.ModeTrojan.String(),
					protocols.ModeShadowSocks.String(),
					protocols.ModeSocks.String(),
					protocols.ModeHysteria2.String(),
					"exit",
				}
				i := c.MultiChoice(modeList, "which protocol to add manually?")
				protocolMode := modeList[i]
				switch protocolMode {
				case protocols.ModeVMessAEAD.String():
					c.Println("========================")
					c.Println(protocolMode)
					c.Println("========================")
					c.Print("alias (remarks): ")
					remarks := c.ReadLine()
					c.Print("address: ")
					address := c.ReadLine()
					c.Print("port: ")
					port, err := strconv.Atoi(c.ReadLine())
					if err != nil || port < 1 || port > 65535 {
						log.Warn("port must be a number between 1 and 65535")
						return
					}
					c.Print("user ID (id): ")
					id := c.ReadLine()
					data := make(map[string]string)
					networkList := []string{
						"tcp",
						"kcp",
						"ws",
						"h2",
						"quic",
						"grpc",
					}
					index := c.MultiChoice(networkList, "transport (network)?")
					network := networkList[index]
					if network != "tcp" {
						data["type"] = network
					}
					switch network {
					case "kcp":
						typeList := []string{
							"none",
							"srtp",
							"utp",
							"wechat-video",
							"dtls",
							"wireguard",
						}
						index := c.MultiChoice(typeList, "disguise type (headerType)?")
						if typeList[index] != "none" {
							data["headerType"] = typeList[index]
						}
						c.Print("KCP seed (seed): ")
						seed := c.ReadLine()
						if seed != "" {
							data["seed"] = seed
						}
					case "ws":
						c.Print("WebSocket path: ")
						path := c.ReadLine()
						if path != "" {
							data["path"] = path
						}
						c.Print("WebSocket host: ")
						host := c.ReadLine()
						if host != "" {
							data["host"] = host
						}
					case "h2":
						c.Print("HTTP/2 path: ")
						path := c.ReadLine()
						if path != "" {
							data["path"] = path
						}
						c.Print("HTTP/2 host: ")
						host := c.ReadLine()
						if host != "" {
							data["host"] = host
						}
					case "quic":
						typeList := []string{
							"none",
							"srtp",
							"utp",
							"wechat-video",
							"dtls",
							"wireguard",
						}
						index := c.MultiChoice(typeList, "disguise type (headerType)?")
						data["headerType"] = typeList[index]
						quicSecurityList := []string{
							"none",
							"aes-128-gcm",
							"chacha20-poly1305",
						}
						index = c.MultiChoice(quicSecurityList, "QUIC encryption (quicSecurity)?")
						quicSecurity := quicSecurityList[index]
						data["quicSecurity"] = quicSecurity
						if quicSecurity != "none" {
							c.Print("encryption key (key), required when quicSecurity is not none: ")
							key := c.ReadLine()
							if key == "" {
								log.Warn("encryption key cannot be empty")
								return
							}
							data["key"] = key
						}
					case "grpc":
						c.Print("gRPC ServiceName: ")
						serviceName := c.ReadLine()
						if serviceName != "" {
							data["serviceName"] = serviceName
						}
						grpcModeList := []string{
							"gun",
							"multi",
						}
						index := c.MultiChoice(grpcModeList, "gRPC transport mode?")
						mode := grpcModeList[index]
						if mode != "gun" {
							data["mode"] = mode
						}
					}
					securityList := []string{
						"",
						"tls",
						"reality",
					}
					security_index := c.MultiChoice(securityList, "transport security?")
					security := securityList[security_index]
					switch security {
					case "":
					case "tls":
						data["security"] = security
						c.Print("SNI: ")
						sni := c.ReadLine()
						if sni != "" {
							data["sni"] = sni
						}
						alpnList := []string{
							"",
							"h2",
							"http/1.1",
							"h2,http/1.1",
						}
						index := c.MultiChoice(alpnList, "Alpn?")
						if alpnList[index] != "" {
							data["alpn"] = alpnList[index]
						}
					case "reality":
						data["security"] = security
						c.Print("SNI: ")
						sni := c.ReadLine()
						if sni != "" {
							data["sni"] = sni
						}
						fpList := []string{
							"",
							"chrome",
							"firefox",
							"safari",
							"ios",
							"android",
							"edge",
							"360",
							"qq",
							"random",
							"randomized",
						}
						index := c.MultiChoice(fpList, "FingerPrint?")
						if fpList[index] != "" {
							data["fp"] = fpList[index]
						}
						c.Print("PublicKey: ")
						data["pbk"] = c.ReadLine()
						c.Print("ShortId: ")
						data["sid"] = c.ReadLine()
						c.Print("SpiderX: ")
						data["spx"] = c.ReadLine()
					}
					vmessAEAD := &protocols.VMessAEAD{
						ID:      id,
						Address: address,
						Port:    port,
						Remarks: remarks,
						Values:  url.Values{},
					}
					for k, v := range data {
						vmessAEAD.Values[k] = []string{v}
					}
					if manage.Manager.AddNode(node.NewNodeByData(vmessAEAD)) {
						c.Println("added")
					}
				case protocols.ModeVLESS.String():
					c.Println("========================")
					c.Println(protocolMode)
					c.Println("========================")
					c.Print("alias (remarks): ")
					remarks := c.ReadLine()
					c.Print("address: ")
					address := c.ReadLine()
					c.Print("port: ")
					port, err := strconv.Atoi(c.ReadLine())
					if err != nil || port < 1 || port > 65535 {
						log.Warn("port must be a number between 1 and 65535")
						return
					}
					c.Print("user ID (id): ")
					id := c.ReadLine()
					data := make(map[string]string)
					flowList := []string{
						"",
						"xtls-rprx-vision",
						"xtls-rprx-vision-udp443",
						"xtls-rprx-origin",
						"xtls-rprx-origin-udp443",
						"xtls-rprx-direct",
						"xtls-rprx-direct-udp443",
						"xtls-rprx-splice",
						"xtls-rprx-splice-udp443",
					}
					index := c.MultiChoice(flowList, "flow control?")
					flow := flowList[index]
					data["flow"] = flow
					networkList := []string{
						"tcp",
						"kcp",
						"ws",
						"h2",
						"quic",
						"grpc",
					}
					index = c.MultiChoice(networkList, "transport (network)?")
					network := networkList[index]
					if network != "tcp" {
						data["type"] = network
					}
					switch network {
					case "tcp":
						typeList := []string{
							"none",
							"http",
						}
						index = c.MultiChoice(typeList, "disguise type (headerType)?")
						if typeList[index] != "none" {
							data["headerType"] = typeList[index]
						}
					case "kcp":
						typeList := []string{
							"none",
							"srtp",
							"utp",
							"wechat-video",
							"dtls",
							"wireguard",
						}
						index = c.MultiChoice(typeList, "disguise type (headerType)?")
						if typeList[index] != "none" {
							data["headerType"] = typeList[index]
						}
						c.Print("KCP seed (seed): ")
						seed := c.ReadLine()
						if seed != "" {
							data["seed"] = seed
						}
					case "ws":
						c.Print("WebSocket path: ")
						path := c.ReadLine()
						if path != "" {
							data["path"] = path
						}
						c.Print("WebSocket host: ")
						host := c.ReadLine()
						if host != "" {
							data["host"] = host
						}
					case "h2":
						c.Print("HTTP/2 path: ")
						path := c.ReadLine()
						if path != "" {
							data["path"] = path
						}
						c.Print("HTTP/2 host: ")
						host := c.ReadLine()
						if host != "" {
							data["host"] = host
						}
					case "quic":
						typeList := []string{
							"none",
							"srtp",
							"utp",
							"wechat-video",
							"dtls",
							"wireguard",
						}
						index = c.MultiChoice(typeList, "disguise type (headerType)?")
						data["headerType"] = typeList[index]
						quicSecurityList := []string{
							"none",
							"aes-128-gcm",
							"chacha20-poly1305",
						}
						index = c.MultiChoice(quicSecurityList, "QUIC encryption (quicSecurity)?")
						quicSecurity := quicSecurityList[index]
						data["quicSecurity"] = quicSecurity
						if quicSecurity != "none" {
							c.Print("encryption key (key), required when quicSecurity is not none: ")
							key := c.ReadLine()
							if key == "" {
								log.Warn("encryption key cannot be empty")
								return
							}
							data["key"] = key
						}
					case "grpc":
						c.Print("gRPC ServiceName: ")
						serviceName := c.ReadLine()
						if serviceName != "" {
							data["serviceName"] = serviceName
						}
						grpcModeList := []string{
							"gun",
							"multi",
						}
						index = c.MultiChoice(grpcModeList, "gRPC transport mode?")
						mode := grpcModeList[index]
						if mode != "gun" {
							data["mode"] = mode
						}
					}

					securityList := []string{
						"",
						"tls",
						"xtls",
						"reality",
					}
					index = c.MultiChoice(securityList, "transport security?")
					security := securityList[index]
					switch security {
					case "":
					case "tls":
						data["security"] = security
						c.Print("SNI: ")
						sni := c.ReadLine()
						if sni != "" {
							data["sni"] = sni
						}
						alpnList := []string{
							"",
							"h2",
							"http/1.1",
							"h2,http/1.1",
						}
						index := c.MultiChoice(alpnList, "Alpn?")
						if alpnList[index] != "" {
							data["alpn"] = alpnList[index]
						}
					case "xtls":
						data["security"] = security
						c.Print("SNI: ")
						sni := c.ReadLine()
						if sni != "" {
							data["sni"] = sni
						}
						alpnList := []string{
							"",
							"h2",
							"http/1.1",
							"h2,http/1.1",
						}
						index := c.MultiChoice(alpnList, "Alpn?")
						if alpnList[index] != "" {
							data["alpn"] = alpnList[index]
						}
					case "reality":
						data["security"] = security
						c.Print("SNI: ")
						sni := c.ReadLine()
						if sni != "" {
							data["sni"] = sni
						}
						fpList := []string{
							"",
							"chrome",
							"firefox",
							"safari",
							"ios",
							"android",
							"edge",
							"360",
							"qq",
							"random",
							"randomized",
						}
						index := c.MultiChoice(fpList, "FingerPrint?")
						if fpList[index] != "" {
							data["fp"] = fpList[index]
						}
						c.Print("PublicKey: ")
						data["pbk"] = c.ReadLine()
						c.Print("ShortId: ")
						data["sid"] = c.ReadLine()
						c.Print("SpiderX: ")
						data["spx"] = c.ReadLine()
					}
					vless := &protocols.VLess{
						ID:      id,
						Address: address,
						Port:    port,
						Remarks: remarks,
						Values:  url.Values{},
					}
					for k, v := range data {
						vless.Values[k] = []string{v}
					}
					if manage.Manager.AddNode(node.NewNodeByData(vless)) {
						c.Println("added")
					}
				case protocols.ModeVMess.String():
					vmess := &protocols.VMess{V: "2"}
					c.Println("========================")
					c.Println(protocolMode)
					c.Println("========================")
					c.Print("alias (remarks): ")
					vmess.Ps = c.ReadLine()
					c.Print("address: ")
					vmess.Add = c.ReadLine()
					c.Print("port: ")
					port, err := strconv.Atoi(c.ReadLine())
					if err != nil || port < 1 || port > 65535 {
						log.Warn("port must be a number between 1 and 65535")
						return
					}
					vmess.Port = port
					c.Print("user ID (id): ")
					vmess.Id = c.ReadLine()
					c.Print("alter ID (alterID): ")
					alterID, err := strconv.Atoi(c.ReadLine())
					if err != nil {
						log.Warn("alter ID must be a number")
						return
					}
					vmess.Aid = alterID
					securityList := []string{
						"auto",
						"aes-128-gcm",
						"chacha20-poly1305",
						"none",
						"zero",
					}
					index := c.MultiChoice(securityList, "encryption (security)?")
					vmess.Scy = securityList[index]
					networkList := []string{
						"tcp",
						"kcp",
						"ws",
						"h2",
						"quic",
						"grpc",
					}
					index = c.MultiChoice(networkList, "transport (network)?")
					vmess.Net = networkList[index]
					switch networkList[index] {
					case "tcp":
						vmess.Type = "none"
					case "kcp":
						typeList := []string{
							"none",
							"srtp",
							"utp",
							"wechat-video",
							"dtls",
							"wireguard",
						}
						index = c.MultiChoice(typeList, "disguise header type (type)?")
						vmess.Type = typeList[index]
						c.Print("mKCP seed (path): ")
						vmess.Path = c.ReadLine()
					case "quic":
						typeList := []string{
							"none",
							"srtp",
							"utp",
							"wechat-video",
							"dtls",
							"wireguard",
						}
						index = c.MultiChoice(typeList, "disguise type (type)?")
						vmess.Type = typeList[index]
						quicSecurityList := []string{
							"none",
							"aes-128-gcm",
							"chacha20-poly1305",
						}
						index = c.MultiChoice(quicSecurityList, "QUIC encryption (host)?")
						vmess.Host = quicSecurityList[index]
						if vmess.Host != "none" {
							c.Print("QUIC encryption key (path): ")
							vmess.Path = c.ReadLine()
						}

					case "ws", "h2":
						c.Print("Host (host): ")
						vmess.Host = c.ReadLine()
						c.Print("Path (path): ")
						vmess.Path = c.ReadLine()
					case "grpc":
						typeList := []string{
							"gun",
							"multi",
						}
						index = c.MultiChoice(typeList, "gRPC transport mode (type)?")
						vmess.Type = typeList[index]
						c.Print("gRPC ServiceName (path): ")
						vmess.Path = c.ReadLine()
					}
					tlsList := []string{
						"",
						"tls",
					}
					index = c.MultiChoice(tlsList, "transport security (tls)?")
					vmess.Tls = tlsList[index]
					if vmess.Tls != "" {
						c.Print("SNI: ")
						vmess.Sni = c.ReadLine()
						alpnList := []string{
							"",
							"h2",
							"http/1.1",
							"h2,http/1.1",
						}
						index := c.MultiChoice(alpnList, "Alpn?")
						vmess.Alpn = alpnList[index]
					}
					c.Println("========================")
					if manage.Manager.AddNode(node.NewNodeByData(vmess)) {
						c.Println("added")
					}
				case protocols.ModeShadowSocks.String():
					c.Println("========================")
					c.Println(protocolMode)
					c.Println("========================")
					c.Print("alias (remarks): ")
					remarks := c.ReadLine()
					c.Print("address: ")
					addr := c.ReadLine()
					c.Print("port: ")
					port, err := strconv.Atoi(c.ReadLine())
					if err != nil || port < 1 || port > 65535 {
						log.Warn("port must be a number between 1 and 65535")
						return
					}
					c.Print("password: ")
					password := c.ReadLine()
					security := []string{
						"aes-256-cfb",
						"aes-128-cfb",
						"chacha20",
						"chacha20-ietf",
						"aes-256-gcm",
						"aes-256-gcm",
						"chacha20-poly1305",
						"chacha20-ietf-poly1305",
					}
					sIndex := c.MultiChoice(security, "encryption (security)?")
					c.Println("========================")
					ss := &protocols.ShadowSocks{
						Remarks:  remarks,
						Password: password,
						Address:  addr,
						Port:     port,
						Method:   security[sIndex],
					}
					if manage.Manager.AddNode(node.NewNodeByData(ss)) {
						c.Println("added")
					}
				case protocols.ModeTrojan.String():
					c.Println("========================")
					c.Println(protocolMode)
					c.Println("========================")
					c.Print("alias (remarks): ")
					remarks := c.ReadLine()
					c.Print("address: ")
					addr := c.ReadLine()
					c.Print("port: ")
					port, err := strconv.Atoi(c.ReadLine())
					if err != nil || port < 1 || port > 65535 {
						log.Warn("port must be a number between 1 and 65535")
						return
					}
					c.Print("password: ")
					password := c.ReadLine()
					c.Print("SNI (optional): ")
					sni := c.ReadLine()
					c.Println("========================")
					trojan := &protocols.Trojan{
						Remarks:  remarks,
						Password: password,
						Address:  addr,
						Port:     port,
					}
					if sni != "" {
						trojan.Values = url.Values{
							"sni": []string{sni},
						}
					}
					if manage.Manager.AddNode(node.NewNodeByData(trojan)) {
						c.Println("added")
					}
				case protocols.ModeHysteria2.String():
					c.Println("========================")
					c.Println(protocolMode)
					c.Println("========================")
					c.Print("alias (remarks): ")
					remarks := c.ReadLine()
					c.Print("address: ")
					addr := c.ReadLine()
					c.Print("port: ")
					port, err := strconv.Atoi(c.ReadLine())
					if err != nil || port < 1 || port > 65535 {
						log.Warn("port must be a number between 1 and 65535")
						return
					}
					c.Print("password: ")
					password := c.ReadLine()
					c.Print("SNI (optional): ")
					sni := c.ReadLine()
					c.Println("========================")
					trojan := &protocols.Trojan{
						Remarks:  remarks,
						Password: password,
						Address:  addr,
						Port:     port,
					}
					if sni != "" {
						trojan.Values = url.Values{
							"sni": []string{sni},
						}
					}
					if manage.Manager.AddNode(node.NewNodeByData(trojan)) {
						c.Println("added")
					}
				case protocols.ModeSocks.String():
					c.Println("========================")
					c.Println(protocolMode)
					c.Println("========================")
					c.Print("alias (remarks): ")
					remarks := c.ReadLine()
					c.Print("address: ")
					addr := c.ReadLine()
					c.Print("port: ")
					port, err := strconv.Atoi(c.ReadLine())
					if err != nil || port < 1 || port > 65535 {
						log.Warn("port must be a number between 1 and 65535")
						return
					}
					c.Print("username (optional): ")
					username := c.ReadLine()
					c.Print("password (optional): ")
					password := c.ReadLine()
					c.Println("========================")
					socks := &protocols.Socks{
						Remarks: remarks,
						Address: addr,
						Port:    port,
					}
					if username != "" && password != "" {
						socks.Username = username
						socks.Password = password
					}
					if manage.Manager.AddNode(node.NewNodeByData(socks)) {
						c.Println("added")
					}
				}
			}
		},
	})
	shell.AddCmd(nodeCmd)
}
