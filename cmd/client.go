package cmd

import (
	"tbox/client"
	"tbox/core"
	"tbox/core/manage"
	"tbox/log"
	"strconv"

	"github.com/abiosoft/ishell"
)

func InitServiceShell(shell *ishell.Shell) {
	// start or restart the service
	shell.AddCmd(&ishell.Cmd{
		Name: "run",
		Help: "start or restart the service",
		Func: func(c *ishell.Context) {
			if len(c.Args) == 1 {
				client.Start(c.Args[0])
			} else {
				client.Start(strconv.Itoa(manage.Manager.SelectedIndex()))
			}

		},
	})
	// stop the service
	shell.AddCmd(&ishell.Cmd{
		Name: "stop",
		Help: "stop the service",
		Func: func(c *ishell.Context) {
			client.StopAll()
		},
	})
	// test nodes (supports ranges/multi-select): test / test 3 / test 1-5 / test 1,3,5
	// press Ctrl+C during a speed test to interrupt and return without exiting the program.
	shell.AddCmd(&ishell.Cmd{
		Name: "test",
		Func: func(c *ishell.Context) {
			key := "all"
			if len(c.Args) == 1 {
				key = c.Args[0]
			}
			indexList := core.IndexList(key, manage.Manager.NodeLen())
			if len(indexList) == 0 {
				log.Warn("no nodes selected")
				return
			}
			client.Connecting(indexList)
		},
	})
}
