package singbox_split

import (
	"sort"
	"strings"
)

func CompileDNSRules(rs *RuleSet, nodeCount int) ([]map[string]interface{}, []int, map[string]string, error) {
	if rs == nil {
		return make([]map[string]interface{}, 0), make([]int, 0), make(map[string]string), nil
	}

	rules := make([]map[string]interface{}, 0, len(rs.DNSRules))
	nodeIndexSet := make(map[int]struct{})
	serverDetours := make(map[string]string)

	for _, rule := range rs.DNSRules {
		if rule == nil {
			return nil, nil, nil, ValidateDNSRule(rule, nodeCount)
		}

		server, available, err := resolveDNSServerAlias(rule.Server)
		if err != nil {
			return nil, nil, nil, err
		}
		if !available {
			continue
		}

		compiled := make(map[string]interface{})
		if len(rule.Match.Inbound) > 0 {
			compiled["inbound"] = rule.Match.Inbound
		}
		if len(rule.Match.IPCIDR) > 0 {
			compiled["ip_cidr"] = rule.Match.IPCIDR
		}
		if len(rule.Match.Domain) > 0 {
			compiled["domain"] = rule.Match.Domain
		}
		if len(rule.Match.DomainSuffix) > 0 {
			compiled["domain_suffix"] = rule.Match.DomainSuffix
		}
		if len(rule.Match.DomainKeyword) > 0 {
			compiled["domain_keyword"] = rule.Match.DomainKeyword
		}
		if len(rule.Match.RuleSet) > 0 {
			compiled["rule_set"] = rule.Match.RuleSet
		} else if len(rule.Match.Geosite) > 0 {
			compiled["rule_set"] = rule.Match.Geosite
		}
		if len(rule.Match.QueryType) > 0 {
			compiled["query_type"] = rule.Match.QueryType
		}

		compiled["action"] = "route"
		compiled["server"] = server
		rules = append(rules, compiled)

		detour := strings.TrimSpace(rule.Detour)
		if detour == "" {
			if rule.Server == "foreign" {
				serverDetours[server] = "proxy"
			}
		} else {
			detourName, nodeIndex, err := parseDNSDetour(detour, nodeCount, false)
			if err != nil {
				return nil, nil, nil, err
			}
			if nodeIndex > 0 {
				nodeIndexSet[nodeIndex] = struct{}{}
			}
			serverDetours[server] = detourName
		}
	}

	nodeIndexes := make([]int, 0, len(nodeIndexSet))
	for index := range nodeIndexSet {
		nodeIndexes = append(nodeIndexes, index)
	}
	sort.Ints(nodeIndexes)

	return rules, nodeIndexes, serverDetours, nil
}
