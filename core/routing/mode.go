package routing

import (
	"regexp"
	"strings"
)

type Mode string

const (
	ModeIP     Mode = "IP"
	ModeDomain      = "Domain"
)

// GetRuleMode reports whether the string is an IP rule or a domain rule.
// Returns IP|Domain.
func GetRuleMode(str string) Mode {
	if strings.HasPrefix(str, "geoip:") {
		return ModeIP
	}
	if strings.Contains(str, "ip.dat:") {
		return ModeIP
	}
	pattern := `(?:^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})(?:/(?:[1-9]|[1-2][0-9]|3[0-2]){1})?$`
	re, _ := regexp.Compile(pattern)
	if re.MatchString(str) {
		return ModeIP
	}
	return ModeDomain
}
