package cmd

import (
	"tbox/cmd/help"
	"tbox/core/routing"
	"tbox/log"
	"fmt"
	"os"
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/atotto/clipboard"
	"github.com/olekukonko/tablewriter"
)

func InitRouteShell(shell *ishell.Shell) {
	routingCmd := &ishell.Cmd{
		Name:    "routing",
		Aliases: []string{"-h", "--help"},
		Func: func(c *ishell.Context) {
			shell.Process("routing", "help")
		},
	}
	routingCmd.AddCmd(&ishell.Cmd{
		Name: "help",
		Func: func(c *ishell.Context) {
			c.Println(help.Routing)
		},
	})
	// block
	routingCmd.AddCmd(&ishell.Cmd{
		Name: "block",
		Func: func(c *ishell.Context) {
			argMap := FlagsParse(c.Args, map[string]string{
				"a": "add",
				"r": "rm",
				"f": "file",
				"c": "clipboard",
			})
			mode := routing.TypeBlock
			if _, ok := argMap["clipboard"]; ok {
				content, err := clipboard.ReadAll()
				if err != nil {
					log.Error(err)
					return
				}
				content = strings.ReplaceAll(content, "\r\n", "\n")
				content = strings.ReplaceAll(content, "\r", "\n")
				routing.AddRule(mode, strings.Split(content, "\n")...)
			} else if fileArg, ok := argMap["file"]; ok {
				if _, err := os.Stat(fileArg); os.IsNotExist(err) {
					log.Error("open ", fileArg, " : no such file")
					return
				}
				data, _ := os.ReadFile(fileArg)
				content := strings.ReplaceAll(string(data), "\r\n", "\n")
				content = strings.ReplaceAll(content, "\r", "\n")
				count := routing.AddRule(mode, strings.Split(content, "\n")...)
				log.Infof("added %d rules", count)
			} else if data, ok := argMap["add"]; ok {
				routing.AddRule(mode, data)
			} else if key, ok := argMap["rm"]; ok {
				routing.DelRule(mode, key)
			} else if key, ok := argMap["data"]; ok {
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"Index", "Type", "Rule"})
				center := tablewriter.ALIGN_CENTER
				table.SetAlignment(center)
				table.AppendBulk(routing.GetRule(mode, key))
				table.SetCaption(true, fmt.Sprintf("total [ %d ] rules", routing.RuleLen(mode)))
				table.Render()
			} else {
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"Index", "Type", "Rule"})
				center := tablewriter.ALIGN_CENTER
				table.SetAlignment(center)
				table.AppendBulk(routing.GetRule(mode, "0-100"))
				table.SetCaption(true, fmt.Sprintf("total [ %d ] rules", routing.RuleLen(mode)))
				table.Render()
			}
		},
	})
	// proxy
	routingCmd.AddCmd(&ishell.Cmd{
		Name: "proxy",
		Func: func(c *ishell.Context) {
			argMap := FlagsParse(c.Args, map[string]string{
				"a": "add",
				"r": "rm",
				"f": "file",
				"c": "clipboard",
			})
			mode := routing.TypeProxy
			if _, ok := argMap["clipboard"]; ok {
				content, err := clipboard.ReadAll()
				if err != nil {
					log.Error(err)
					return
				}
				content = strings.ReplaceAll(content, "\r\n", "\n")
				content = strings.ReplaceAll(content, "\r", "\n")
				routing.AddRule(mode, strings.Split(content, "\n")...)
			} else if fileArg, ok := argMap["file"]; ok {
				if _, err := os.Stat(fileArg); os.IsNotExist(err) {
					log.Error("open ", fileArg, " : no such file")
					return
				}
				data, _ := os.ReadFile(fileArg)
				content := strings.ReplaceAll(string(data), "\r\n", "\n")
				content = strings.ReplaceAll(content, "\r", "\n")
				count := routing.AddRule(mode, strings.Split(content, "\n")...)
				log.Infof("added %d rules", count)
			} else if data, ok := argMap["add"]; ok {
				routing.AddRule(mode, data)
			} else if key, ok := argMap["rm"]; ok {
				routing.DelRule(mode, key)
			} else if key, ok := argMap["data"]; ok {
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"Index", "Type", "Rule"})
				center := tablewriter.ALIGN_CENTER
				table.SetAlignment(center)
				table.AppendBulk(routing.GetRule(mode, key))
				table.SetCaption(true, fmt.Sprintf("total [ %d ] rules", routing.RuleLen(mode)))
				table.Render()
			} else {
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"Index", "Type", "Rule"})
				center := tablewriter.ALIGN_CENTER
				table.SetAlignment(center)
				table.AppendBulk(routing.GetRule(mode, "0-100"))
				table.SetCaption(true, fmt.Sprintf("total [ %d ] rules", routing.RuleLen(mode)))
				table.Render()
			}
		},
	})
	// direct
	routingCmd.AddCmd(&ishell.Cmd{
		Name: "direct",
		Func: func(c *ishell.Context) {
			argMap := FlagsParse(c.Args, map[string]string{
				"a": "add",
				"r": "rm",
				"f": "file",
				"c": "clipboard",
			})
			mode := routing.TypeDirect
			if _, ok := argMap["clipboard"]; ok {
				content, err := clipboard.ReadAll()
				if err != nil {
					log.Error(err)
					return
				}
				content = strings.ReplaceAll(content, "\r\n", "\n")
				content = strings.ReplaceAll(content, "\r", "\n")
				routing.AddRule(mode, strings.Split(content, "\n")...)
			} else if fileArg, ok := argMap["file"]; ok {
				if _, err := os.Stat(fileArg); os.IsNotExist(err) {
					log.Error("open ", fileArg, " : no such file")
					return
				}
				data, _ := os.ReadFile(fileArg)
				content := strings.ReplaceAll(string(data), "\r\n", "\n")
				content = strings.ReplaceAll(content, "\r", "\n")
				count := routing.AddRule(mode, strings.Split(content, "\n")...)
				log.Infof("added %d rules", count)
			} else if data, ok := argMap["add"]; ok {
				routing.AddRule(mode, data)
			} else if key, ok := argMap["rm"]; ok {
				routing.DelRule(mode, key)
			} else if key, ok := argMap["data"]; ok {
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"Index", "Type", "Rule"})
				center := tablewriter.ALIGN_CENTER
				table.SetAlignment(center)
				table.AppendBulk(routing.GetRule(mode, key))
				table.SetCaption(true, fmt.Sprintf("total [ %d ] rules", routing.RuleLen(mode)))
				table.Render()
			} else {
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"Index", "Type", "Rule"})
				center := tablewriter.ALIGN_CENTER
				table.SetAlignment(center)
				table.AppendBulk(routing.GetRule(mode, "0-100"))
				table.SetCaption(true, fmt.Sprintf("total [ %d ] rules", routing.RuleLen(mode)))
				table.Render()
			}
		},
	})
	shell.AddCmd(routingCmd)
}
