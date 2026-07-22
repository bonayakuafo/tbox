package singbox_split

import (
	"tbox/core/setting"
	"fmt"
	"strconv"
	"strings"
)

func ValidateRule(r *Rule, nodeCount int) error {
	if r == nil {
		return fmt.Errorf("rule must not be nil")
	}

	if err := validateMatch(&r.Match); err != nil {
		return err
	}

	if _, _, err := parseTarget(r.Target, nodeCount, true); err != nil {
		return err
	}

	if r.Action != "" && r.Action != "route" && r.Action != "resolve" {
		return fmt.Errorf("action must be route or resolve")
	}

	return nil
}

func ValidateDNSRule(r *DNSRule, nodeCount int) error {
	if r == nil {
		return fmt.Errorf("DNS rule must not be nil")
	}

	if err := validateMatch(&r.Match); err != nil {
		return err
	}

	_, available, err := resolveDNSServerAlias(r.Server)
	if err != nil {
		return err
	}
	if !available {
		return fmt.Errorf("DNS server alias %s is not configured", r.Server)
	}

	if strings.TrimSpace(r.Detour) != "" {
		if _, _, err := parseDNSDetour(r.Detour, nodeCount, true); err != nil {
			return err
		}
	}

	return nil
}

func ValidateRuleSet(rs *RuleSet, nodeCount int) error {
	if rs == nil {
		return nil
	}

	for i, rule := range rs.Rules {
		if err := ValidateRule(rule, nodeCount); err != nil {
			return fmt.Errorf("rule #%d validation failed: %w", i+1, err)
		}
	}

	for i, rule := range rs.DNSRules {
		if err := ValidateDNSRule(rule, nodeCount); err != nil {
			return fmt.Errorf("DNS rule #%d validation failed: %w", i+1, err)
		}
	}

	return nil
}

func validateMatch(m *Match) error {
	if m == nil || !hasMatcher(*m) {
		return fmt.Errorf("rule needs at least one match condition")
	}

	for _, qt := range m.QueryType {
		if err := validateQueryType(qt); err != nil {
			return err
		}
	}

	return nil
}

func hasMatcher(m Match) bool {
	return len(m.Inbound) > 0 ||
		len(m.IPCIDR) > 0 ||
		len(m.Domain) > 0 ||
		len(m.DomainSuffix) > 0 ||
		len(m.DomainKeyword) > 0 ||
		len(m.Geosite) > 0 ||
		len(m.RuleSet) > 0 ||
		len(m.QueryType) > 0 ||
		len(m.MatchOutbound) > 0
}

func validateQueryType(qt string) error {
	switch strings.ToUpper(qt) {
	case "A", "AAAA", "CNAME", "MX", "NS", "PTR", "TXT", "SOA", "SRV", "HTTPS":
		return nil
	default:
		return fmt.Errorf("unsupported DNS query type: %s", qt)
	}
}

func resolveDNSServerAlias(alias string) (string, bool, error) {
	switch alias {
	case "local":
		return "local", true, nil
	case "domestic":
		if setting.DNSDomestic() == "" {
			return "", false, nil
		}
		return "domestic_1", true, nil
	case "backup":
		if setting.DNSBackup() == "" {
			return "", false, nil
		}
		return "domestic_2", true, nil
	case "foreign":
		if setting.DNSForeign() == "" {
			return "", false, nil
		}
		return "foreign", true, nil
	default:
		return "", false, fmt.Errorf("unknown DNS server alias: %s", alias)
	}
}

func parseDNSDetour(target string, nodeCount int, checkNodeCount bool) (string, int, error) {
	target = strings.TrimSpace(target)

	switch target {
	case "proxy":
		return "proxy", 0, nil
	case "direct":
		return "direct-out", 0, nil
	case "block":
		return "block", 0, nil
	}

	if !strings.HasPrefix(target, "node:") {
		return "", 0, fmt.Errorf("detour must be proxy, direct, block, or node:<index>")
	}

	indexText := strings.TrimPrefix(target, "node:")
	index, err := strconv.Atoi(indexText)
	if err != nil || index <= 0 {
		return "", 0, fmt.Errorf("detour node target format error, must be node:<positive integer>")
	}

	if checkNodeCount && index > nodeCount {
		return "", 0, fmt.Errorf("detour node target out of range, max allowed is node:%d", nodeCount)
	}

	return fmt.Sprintf("split-node-%d", index), index, nil
}

func parseTarget(target string, nodeCount int, checkNodeCount bool) (string, int, error) {
	switch target {
	case "proxy":
		return "proxy", 0, nil
	case "direct":
		return "direct-out", 0, nil
	case "block":
		return "block", 0, nil
	}

	if !strings.HasPrefix(target, "node:") {
		return "", 0, fmt.Errorf("target must be proxy, direct, block, or node:<index>")
	}

	indexText := strings.TrimPrefix(target, "node:")
	index, err := strconv.Atoi(indexText)
	if err != nil || index <= 0 {
		return "", 0, fmt.Errorf("node target format error, must be node:<positive integer>")
	}

	if checkNodeCount && index > nodeCount {
		return "", 0, fmt.Errorf("node target out of range, max allowed is node:%d", nodeCount)
	}

	return fmt.Sprintf("split-node-%d", index), index, nil
}
