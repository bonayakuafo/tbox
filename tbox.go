package main

import (
	"tbox/cmd"
	"tbox/core"
	"tbox/core/setting"
	"tbox/core/setting/key"
	"tbox/log"
	"os"
	"path/filepath"

	"github.com/abiosoft/ishell"
	"github.com/spf13/viper"
)

// version is injected at build time via -ldflags "-X main.version=<tag>".
var version = "dev"

const name = "tbox"

func init() {
	// initialize the logger
	absPath := filepath.Join(core.GetConfigDir(), "info.log")
	log.Init(
		log.GetConsoleZapcore(log.INFO),
		log.GetFileZapcore(absPath, log.INFO, 5),
	)
}

func beforeOfRun(shell *ishell.Shell) {
	cmd := viper.GetString(key.RunBefore)
	if cmd != "" {
		for _, line := range setting.NewAlias("", cmd).GetCmd() {
			shell.Process(line...)
		}
		shell.Print("\n>>> ")
	}
}

func main() {
	shell := ishell.New()
	cmd.InitShell(shell)
	shell.Set("version", version)
	shell.Set("name", name)
	if len(os.Args) > 1 {
		_ = shell.Process(os.Args[1:]...)
	} else {
		go beforeOfRun(shell)
		shell.Printf("%s - %s Shell Client - %s\n", name, viper.GetString(key.ClientCore), version)
		shell.Run()
	}
}
