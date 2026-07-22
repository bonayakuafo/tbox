package cmd

import (
	"tbox/client"
	"tbox/cmd/help"

	"github.com/abiosoft/ishell"
)

func InitShell(shell *ishell.Shell) {
	shell.AddCmd(&ishell.Cmd{
		Name:    "version",
		Aliases: []string{"-v", "--version"},
		Help:    "program version",
		Func: func(c *ishell.Context) {
			c.Printf("%s version \"%s\"\n", c.Get("name"), c.Get("version"))
		},
	})
	shell.AddCmd(&ishell.Cmd{
		Name:    "help",
		Aliases: []string{"-h", "--help"},
		Help:    "help information",
		Func: func(c *ishell.Context) {
			c.Println(help.Help)
		},
	})
	InitSettingShell(shell)
	InitRuleShell(shell)
	InitNodeShell(shell)
	InitSubscribeShell(shell)
	InitFilterShell(shell)
	InitRouteShell(shell)
	InitServiceShell(shell)
	InitAliasShell(shell)
	shell.AddCmd(&ishell.Cmd{
		Name: "log",
		Func: func(c *ishell.Context) {
			client.ShowLog()
		},
	})
}

// argument parser
func FlagsParse(args []string, keys map[string]string) map[string]string {
	resultMap := make(map[string]string)
	key := "data"
	for _, x := range args {
		if len(x) >= 2 {
			if x[:2] == "--" {
				key = x[2:]
				resultMap[key] = ""
			} else if x[:1] == "-" {
				if x[1] >= 48 && x[1] <= 57 {
					resultMap[key] = x
				} else if len(x) == 2 {
					d, ok := keys[x[1:]]
					if ok {
						key = d
					} else {
						key = x[1:]
					}
					resultMap[key] = ""
				}
			} else {
				resultMap[key] = x
			}
		} else if len(x) > 0 {
			resultMap[key] = x
		}
	}
	return resultMap
}
