package setting

import (
	"tbox/core/setting/key"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

func Socks() int {
	return viper.GetInt(key.Socks)
}

func SetSocks(port int) error {
	if port < 1 || port > 65535 {
		return errors.New("socks port must be in 1~65535")
	}
	viper.Set(key.Socks, port)
	return viper.WriteConfig()
}

func Http() int {
	return viper.GetInt(key.Http)
}

func SetHttp(port int) error {
	if port < 0 || port > 65535 {
		return errors.New("http port must be in 0~65535")
	}
	viper.Set(key.Http, port)
	return viper.WriteConfig()
}

func UDP() bool {
	return viper.GetBool(key.UDP)
}

func SetUDP(status bool) error {
	viper.Set(key.UDP, status)
	return viper.WriteConfig()
}

func Sniffing() bool {
	return viper.GetBool(key.Sniffing)
}

func SetSniffing(status bool) error {
	viper.Set(key.Sniffing, status)
	return viper.WriteConfig()
}

func FromLanConn() bool {
	return viper.GetBool(key.FromLanConn)
}

func SetFromLanConn(status bool) error {
	viper.Set(key.FromLanConn, status)
	return viper.WriteConfig()
}

func Mux() bool {
	return viper.GetBool(key.Mux)
}

func SetMux(status bool) error {
	viper.Set(key.Mux, status)
	return viper.WriteConfig()
}

func Pid() int {
	return viper.GetInt(key.PID)
}

func SetPid(pid int) error {
	viper.Set(key.PID, pid)
	return viper.WriteConfig()
}

// BridgePid is the process id of the xray protocol converter in the TUN+xray
// bridge mode. That mode runs sing-box (TUN/routing/DNS) together with xray
// (only for socks->xhttp conversion); sing-box's pid stays in Pid while
// xray's is recorded here so Stop can clean up both.
func BridgePid() int {
	return viper.GetInt(key.BridgePID)
}

func SetBridgePid(pid int) error {
	viper.Set(key.BridgePID, pid)
	return viper.WriteConfig()
}

func AllowInsecure() bool {
	return viper.GetBool(key.AllowInsecure)
}

func SetAllowInsecure(status bool) error {
	viper.Set(key.AllowInsecure, status)
	return viper.WriteConfig()
}

func UserAgent() string {
	return viper.GetString(key.UserAgent)
}

func SetUserAgent(userAgent string) error {
	viper.Set(key.UserAgent, userAgent)
	return viper.WriteConfig()
}

func ChainProxy() string {
	return viper.GetString(key.ChainProxy)
}

func SetChainProxy(tag string) error {
	return SetChainProxyIndexes(tag)
}

func ChainProxyIndexes() ([]int, error) {
	return parseChainProxyIndexes(ChainProxy())
}

func ParseChainProxyIndexes(raw string) ([]int, error) {
	return parseChainProxyIndexes(raw)
}

func SetChainProxyIndexes(raw string) error {
	raw = strings.TrimSpace(raw)
	_, err := parseChainProxyIndexes(raw)
	if err != nil {
		return err
	}
	viper.Set(key.ChainProxy, raw)
	return viper.WriteConfig()
}

func parseChainProxyIndexes(raw string) ([]int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []int{}, nil
	}

	tokens := strings.Split(raw, ",")
	indexes := make([]int, 0)
	seen := make(map[int]struct{})

	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			return nil, errors.New("chain proxy format error: empty index")
		}

		if strings.Contains(token, "-") {
			parts := strings.Split(token, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("chain proxy format error: invalid range %q", token)
			}

			start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil || start <= 0 {
				return nil, fmt.Errorf("chain proxy format error: invalid index %q", token)
			}
			end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil || end <= 0 {
				return nil, fmt.Errorf("chain proxy format error: invalid index %q", token)
			}
			if start > end {
				return nil, fmt.Errorf("chain proxy format error: range start greater than end %q", token)
			}

			for i := start; i <= end; i++ {
				if _, ok := seen[i]; ok {
					return nil, fmt.Errorf("chain proxy format error: index %d duplicated", i)
				}
				seen[i] = struct{}{}
				indexes = append(indexes, i)
			}
			continue
		}

		index, err := strconv.Atoi(token)
		if err != nil || index <= 0 {
			return nil, fmt.Errorf("chain proxy format error: invalid index %q", token)
		}
		if _, ok := seen[index]; ok {
			return nil, fmt.Errorf("chain proxy format error: index %d duplicated", index)
		}
		seen[index] = struct{}{}
		indexes = append(indexes, index)
	}

	return indexes, nil
}

func TunMode() bool {
	return viper.GetBool(key.TunMode)
}

func SetTunMode(status bool) error {
	viper.Set(key.TunMode, status)
	return viper.WriteConfig()
}

func TunAutoRoute() bool {
	return viper.GetBool(key.TunAutoRoute)
}

func SetTunAutoRoute(status bool) error {
	viper.Set(key.TunAutoRoute, status)
	return viper.WriteConfig()
}

func TunAutoRedirect() bool {
	return viper.GetBool(key.TunAutoRedirect)
}

func SetTunAutoRedirect(status bool) error {
	viper.Set(key.TunAutoRedirect, status)
	return viper.WriteConfig()
}

func SingboxConfigDir() string {
	return viper.GetString(key.SingboxConfigDir)
}

func SetSingboxConfigDir(dir string) error {
	viper.Set(key.SingboxConfigDir, dir)
	return viper.WriteConfig()
}
