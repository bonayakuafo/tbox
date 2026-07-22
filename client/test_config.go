package client

import (
	"tbox/client/config"
	"tbox/core/protocols"
	"tbox/log"
)

// GenTestConfig generates the core config file.
func GenTestConfig(node protocols.Protocol) string {
	cfg, err := config.CreateConfig(CoreName)
	if err != nil {
		log.Error(err)
		return ""
	}
	path, err := cfg.GenConfig(node)
	if err != nil {
		log.Error(err)
		return ""
	}
	return path
}
