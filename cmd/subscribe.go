package cmd

import (
	"tbox/cmd/help"
	"tbox/core/manage"
	"tbox/core/node"
	"tbox/core/setting"
	"tbox/core/sub"
	"tbox/log"
	"os"
	"strconv"
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/olekukonko/tablewriter"
)

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen/2-1] + "..." + s[len(s)-maxLen/2+2:]
}

// subNamesInNodes returns subscription aliases actually present in the current node list (deduped),
// used for update-node tab completion: only suggests subscriptions with nodes, avoiding empty ones.
func subNamesInNodes() []string {
	usedIDs := make(map[string]struct{})
	manage.Manager.NodeForEach(func(i int, n *node.Node) {
		if n != nil && n.SubID != "" {
			usedIDs[n.SubID] = struct{}{}
		}
	})
	names := make([]string, 0, len(usedIDs))
	manage.Manager.SubForEach(func(i int, s *sub.Subscirbe) {
		if _, ok := usedIDs[s.ID()]; ok && s.Name != "" {
			names = append(names, s.Name)
		}
	})
	return names
}

// resolveSubKey resolves the update-node target argument into a subscription index key.
// pure numbers/ranges are returned as-is (handled by IndexList); aliases are resolved to their index.
// unmatched aliases are returned as-is and handled downstream as an index.
func resolveSubKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	matched := ""
	manage.Manager.SubForEach(func(i int, s *sub.Subscirbe) {
		if s.Name == raw {
			matched = strconv.Itoa(i)
		}
	})
	if matched != "" {
		return matched
	}
	return raw
}

func InitSubscribeShell(shell *ishell.Shell) {
	subCmd := &ishell.Cmd{
		Name: "sub",
		Func: func(c *ishell.Context) {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Index", "Alias", "Subscription URL", "User-Agent", "Enabled"})
			table.SetAlignment(tablewriter.ALIGN_CENTER)
			manage.Manager.SubForEach(func(i int, subscirbe *sub.Subscirbe) {
				ua := subscirbe.UserAgent
				if ua == "" {
					ua = "(default)"
				} else {
					ua = truncateString(ua, 20)
				}
				table.Append([]string{
					strconv.Itoa(i),
					subscirbe.Name,
					subscirbe.Url,
					ua,
					strconv.FormatBool(subscirbe.Using),
				})
			})
			table.Render()
			log.Info("View subscription notes: sub remark")
		},
	}
	// help
	subCmd.AddCmd(&ishell.Cmd{
		Name:    "help",
		Aliases: []string{"-h", "--help"},
		Func: func(c *ishell.Context) {
			c.Println(help.Sub)
		},
	})
	subCmd.AddCmd(&ishell.Cmd{
		Name: "remark",
		Func: func(c *ishell.Context) {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Index", "Alias", "Note"})
			table.SetAlignment(tablewriter.ALIGN_CENTER)
			manage.Manager.SubForEach(func(i int, subscirbe *sub.Subscirbe) {
				remark := subscirbe.Remark
				if remark == "" {
					remark = "-"
				}
				table.Append([]string{
					strconv.Itoa(i),
					subscirbe.Name,
					remark,
				})
			})
			table.Render()
		},
	})
	// add
	subCmd.AddCmd(&ishell.Cmd{
		Name: "add",
		Func: func(c *ishell.Context) {
			argMap := FlagsParse(c.Args, map[string]string{
				"r": "remarks",
				"a": "ua",
				"m": "remark",
			})
			if len(c.Args) >= 1 {
				if sublink, ok := argMap["data"]; ok {
					remarksArg := argMap["remarks"]
					if remarksArg == "" {
						remarksArg = "remarks"
					}
					s := sub.NewSubscirbe(sublink, remarksArg)
					s.UserAgent = argMap["ua"]
					s.Remark = argMap["remark"]
					manage.Manager.AddSubscirbe(s)
					_ = shell.Process("sub")
				} else {
					log.Warn("a subscription link is required")
				}
			} else if len(c.Args) == 0 {
				log.Warn("a subscription link is required")
			}
		},
	})
	// update-node
	subCmd.AddCmd(&ishell.Cmd{
		Name: "update-node",
		// tab completion: list subscription aliases actually present in the current nodes, to help pick an update target.
		Completer: func(args []string) []string {
			return subNamesInNodes()
		},
		Func: func(c *ishell.Context) {
			argMap := FlagsParse(c.Args, map[string]string{
				"s": "socks",
				"h": "http",
				"a": "addr",
			})
			opt := sub.UpdataOption{}
			// key accepts a subscription index or an alias (from completion); aliases are resolved to their index.
			opt.Key = resolveSubKey(argMap["data"])
			if socks, ok := argMap["socks"]; ok {
				if v, err := strconv.Atoi(socks); err == nil {
					if 0 < v && v <= 65535 {
						opt.Port = v
					}
				}
				opt.ProxyMode = sub.SOCKS
			} else if http, ok := argMap["http"]; ok {
				if v, err := strconv.Atoi(http); err == nil {
					if 0 < v && v <= 65535 {
						opt.Port = v
					}
				}
				opt.ProxyMode = sub.HTTP
			}
			if address, ok := argMap["addr"]; ok {
				opt.Addr = address
			}
			opt.UserAgent = setting.UserAgent()
			manage.Manager.UpdataNode(opt)
		},
	})
	// rm
	subCmd.AddCmd(&ishell.Cmd{
		Name:    "rm",
		Aliases: []string{"del"},
		Func: func(c *ishell.Context) {
			if len(c.Args) == 1 {
				if !confirm(c, "Confirm removal of subscription "+c.Args[0]+" and its nodes?") {
					log.Info("cancelled")
					return
				}
				manage.Manager.DelSub(c.Args[0])
				_ = shell.Process("sub")
			}
		},
	})
	// mv
	subCmd.AddCmd(&ishell.Cmd{
		Name:    "mv",
		Aliases: []string{"set"},
		Func: func(c *ishell.Context) {
			argMap := FlagsParse(c.Args, map[string]string{
				"r": "remarks",
				"u": "url",
				"a": "ua",
				"m": "remark",
			})
			if key, ok := argMap["data"]; ok {
				url := argMap["url"]
				remarks := argMap["remarks"]
				userAgent := argMap["ua"]
				remark := argMap["remark"]

				using := ""
				if value, ok := argMap["using"]; ok {
					using = value
				}
				manage.Manager.SetSub(key, using, url, remarks, userAgent, remark)
				_ = shell.Process("sub")
			}
		},
	})
	shell.AddCmd(subCmd)
}
