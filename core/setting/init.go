package setting

import (
	"tbox/core"
	"tbox/core/setting/key"
	"tbox/log"
	"os"

	"github.com/spf13/viper"
)

func init() {
	// create the config file if it does not exist
	if _, err := os.Stat(core.SettingFile); os.IsNotExist(err) {
		file, err := os.Create(core.SettingFile)
		if err != nil {
			log.Error(err)
		} else {
			_ = file.Close()
		}
	}
	viper.SetConfigName("tbox.setting")
	viper.SetConfigType("toml")
	viper.AddConfigPath(core.GetConfigDir())
	// set default values
	viper.SetDefault(key.Socks, 23333)
	viper.SetDefault(key.Http, 0)
	viper.SetDefault(key.UDP, true)
	viper.SetDefault(key.Sniffing, true)
	viper.SetDefault(key.FromLanConn, false)
	viper.SetDefault(key.Mux, false)
	viper.SetDefault(key.AllowInsecure, false)

	viper.SetDefault(key.RoutingStrategy, "IPIfNonMatch") // routing strategy
	viper.SetDefault(key.RoutingBypass, true)             // bypass LAN and mainland

	viper.SetDefault(key.DNSPort, 13500)
	viper.SetDefault(key.DNSForeign, "tcp://9.9.9.9")
	viper.SetDefault(key.DNSDomestic, "119.29.29.29")
	viper.SetDefault(key.DNSBackup, "114.114.114.114")

	viper.SetDefault(key.TestURL, "https://www.cloudflare.com")
	viper.SetDefault(key.TestTimeout, 10)
	viper.SetDefault(key.TestMinTime, 10000)
	viper.SetDefault(key.RunBefore, "")

	viper.SetDefault(key.PID, 0)
	viper.SetDefault(key.ClientCore, "sing-box")
	viper.SetDefault(key.UserAgent, "sing-box")
	viper.SetDefault(key.ChainProxy, "")
	viper.SetDefault(key.TunMode, false)
	viper.SetDefault(key.TunAutoRoute, true)
	viper.SetDefault(key.TunAutoRedirect, false)
	viper.SetDefault(key.SingboxConfigDir, "")
	// read the config file
	// when no file exists there is no config, so the defaults above are used.
	// when a file exists, missing keys fall back to defaults; present keys use the file value.
	err := viper.ReadInConfig()
	if err != nil {
		log.Error(err)
	}
}
