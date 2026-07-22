package setting

import (
	"tbox/core/setting/key"
	"errors"

	"github.com/spf13/viper"
)

func ClientCore() string {
	return viper.GetString(key.ClientCore)
}

func SetClientCore(coreType int) error {
	switch coreType {
	case 1:
		viper.Set(key.ClientCore, "sing-box")
	case 2:
		viper.Set(key.ClientCore, "xray")
	default:
		return errors.New("no matching client core")
	}
	return viper.WriteConfig()
}
