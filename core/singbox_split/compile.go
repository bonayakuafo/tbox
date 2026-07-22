package singbox_split

import "sort"

func CompileRules(rs *RuleSet) ([]map[string]interface{}, []int, error) {
	if rs == nil {
		return make([]map[string]interface{}, 0), make([]int, 0), nil
	}

	rules := make([]map[string]interface{}, 0, len(rs.Rules))
	nodeIndexSet := make(map[int]struct{})

	for _, rule := range rs.Rules {
		if rule == nil {
			return nil, nil, ValidateRule(rule, 0)
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
		if len(rule.Match.RuleSet) > 0 {
			compiled["rule_set"] = rule.Match.RuleSet
		} else if len(rule.Match.Geosite) > 0 {
			compiled["rule_set"] = rule.Match.Geosite
		}
		targetOutbound, nodeIndex, err := parseTarget(rule.Target, 0, false)
		if err != nil {
			return nil, nil, err
		}

		action := rule.Action
		if action == "" {
			action = "route"
		}
		compiled["action"] = action
		if action != "resolve" {
			compiled["outbound"] = targetOutbound
		}
		rules = append(rules, compiled)

		if nodeIndex > 0 {
			nodeIndexSet[nodeIndex] = struct{}{}
		}
	}

	nodeIndexes := make([]int, 0, len(nodeIndexSet))
	for index := range nodeIndexSet {
		nodeIndexes = append(nodeIndexes, index)
	}
	sort.Ints(nodeIndexes)

	return rules, nodeIndexes, nil
}
