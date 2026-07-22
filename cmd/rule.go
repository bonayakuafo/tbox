package cmd

import (
	"tbox/cmd/help"
	"tbox/core/manage"
	"tbox/core/singbox_split"
	"tbox/log"
	"fmt"
	"strconv"
	"strings"

	"github.com/abiosoft/ishell"
)

// The rule command manages sing-box split routing rules with two subcommand groups:
//   rule route  —— traffic routing rules
//   rule dns    —— DNS routing rules

// ---------- shared helpers ----------

func loadRuleSet() (*singbox_split.RuleSet, bool) {
	rs, err := singbox_split.Load()
	if err != nil {
		log.Error(err)
		return nil, false
	}
	return rs, true
}

func saveRuleSet(rs *singbox_split.RuleSet) bool {
	if err := singbox_split.Save(rs); err != nil {
		log.Error(err)
		return false
	}
	return true
}

func parseRuleIndex(indexText string) (int, bool) {
	index, err := strconv.Atoi(strings.TrimSpace(indexText))
	if err != nil || index <= 0 {
		log.Warn("rule index must be a positive integer")
		return 0, false
	}
	return index, true
}

// parseNodeTarget parses a target of the form node:N and validates that the node exists.
// A non node: prefix returns isNode=false (leaving it to later Validate handling).
func parseNodeTarget(target string) (index int, isNode bool, err error) {
	if !strings.HasPrefix(target, "node:") {
		return 0, false, nil
	}
	idx, convErr := strconv.Atoi(strings.TrimPrefix(target, "node:"))
	if convErr != nil || idx <= 0 {
		return 0, true, fmt.Errorf("invalid node target format, must be node:<positive integer>")
	}
	return idx, true, nil
}

func matchFromArgs(argMap map[string]string) singbox_split.Match {
	return singbox_split.Match{
		Inbound:       splitListArgs(argMap["inbound"]),
		IPCIDR:        splitListArgs(argMap["ip-cidr"]),
		Domain:        splitListArgs(argMap["domain"]),
		DomainSuffix:  splitListArgs(argMap["domain-suffix"]),
		DomainKeyword: splitListArgs(argMap["domain-keyword"]),
		Geosite:       splitListArgs(argMap["geosite"]),
		RuleSet:       splitListArgs(argMap["rule-set"]),
		QueryType:     splitListArgs(argMap["query-type"]),
		MatchOutbound: splitListArgs(argMap["match-outbound"]),
	}
}

// editMatch incrementally adds/removes match fields on an existing Match via --add-xxx / --rm-xxx.
func editMatch(m *singbox_split.Match, argMap map[string]string) {
	type field struct {
		list *[]string
		name string
	}
	fields := []field{
		{&m.Inbound, "inbound"},
		{&m.IPCIDR, "ip-cidr"},
		{&m.Domain, "domain"},
		{&m.DomainSuffix, "domain-suffix"},
		{&m.DomainKeyword, "domain-keyword"},
		{&m.Geosite, "geosite"},
		{&m.RuleSet, "rule-set"},
		{&m.QueryType, "query-type"},
		{&m.MatchOutbound, "match-outbound"},
	}
	for _, f := range fields {
		if v, ok := argMap["add-"+f.name]; ok {
			*f.list = singbox_split.MergeList(*f.list, splitListArgs(v))
		}
		if v, ok := argMap["rm-"+f.name]; ok {
			*f.list = singbox_split.RemoveList(*f.list, splitListArgs(v))
		}
	}
}

func formatMatch(m singbox_split.Match) string {
	parts := make([]string, 0, 9)
	add := func(name string, list []string) {
		if len(list) > 0 {
			parts = append(parts, name+":"+strings.Join(list, ","))
		}
	}
	add("inbound", m.Inbound)
	add("ip_cidr", m.IPCIDR)
	add("domain", m.Domain)
	add("domain_suffix", m.DomainSuffix)
	add("domain_keyword", m.DomainKeyword)
	add("geosite", m.Geosite)
	add("rule_set", m.RuleSet)
	add("query_type", m.QueryType)
	add("match_outbound", m.MatchOutbound)
	return strings.Join(parts, "; ")
}

// ---------- route (traffic routing rules) ----------

func showRouteSummary() {
	rs, ok := loadRuleSet()
	if !ok {
		return
	}
	log.Info("route rule count: ", len(rs.Rules))
}

func listRouteRules() {
	rs, ok := loadRuleSet()
	if !ok {
		return
	}
	if len(rs.Rules) == 0 {
		log.Info("no route rules configured")
		return
	}
	for i, rule := range rs.Rules {
		if rule == nil {
			log.Warn(fmt.Sprintf("[%d] empty rule", i+1))
			continue
		}
		actionDesc := ""
		if rule.Action == "resolve" {
			actionDesc = " [resolve]"
		}
		log.Info(fmt.Sprintf("[%d] %s -> %s%s", i+1, formatMatch(rule.Match), rule.Target, actionDesc))
	}
}

func addRouteRule(c *ishell.Context) {
	argMap := FlagsParse(c.Args, map[string]string{})
	target := strings.TrimSpace(argMap["target"])
	if target == "" {
		log.Warn("use --target to specify the routing target")
		return
	}
	action := strings.TrimSpace(argMap["action"])
	if action == "" {
		action = "route"
	}
	rule := &singbox_split.Rule{
		Target: target,
		Action: action,
		Match:  matchFromArgs(argMap),
	}
	if index, isNode, err := parseNodeTarget(target); err != nil {
		log.Error(err)
		return
	} else if isNode && manage.Manager.GetNode(index) == nil {
		log.Error(fmt.Errorf("node target does not exist: index %d", index))
		return
	}
	if err := singbox_split.ValidateRule(rule, manage.Manager.NodeLen()); err != nil {
		log.Error(err)
		return
	}
	rs, ok := loadRuleSet()
	if !ok {
		return
	}
	rs.Rules = append(rs.Rules, rule)
	if !saveRuleSet(rs) {
		return
	}
	log.Info("route rule added: ", formatMatch(rule.Match), " -> ", rule.Target)
	showRouteSummary()
}

func editRouteRule(c *ishell.Context) {
	if len(c.Args) < 1 {
		log.Warn("please provide the rule index to edit")
		return
	}
	index, ok := parseRuleIndex(c.Args[0])
	if !ok {
		return
	}
	rs, ok := loadRuleSet()
	if !ok {
		return
	}
	if index > len(rs.Rules) {
		log.Warn("rule index out of range")
		return
	}
	rule := rs.Rules[index-1]
	if rule == nil {
		log.Warn("this rule is empty")
		return
	}
	argMap := FlagsParse(c.Args[1:], map[string]string{})
	editMatch(&rule.Match, argMap)
	if v, ok := argMap["target"]; ok {
		target := strings.TrimSpace(v)
		if idx, isNode, err := parseNodeTarget(target); err != nil {
			log.Error(err)
			return
		} else if isNode && manage.Manager.GetNode(idx) == nil {
			log.Error(fmt.Errorf("node target does not exist: index %d", idx))
			return
		}
		rule.Target = target
	}
	if v, ok := argMap["action"]; ok {
		rule.Action = strings.TrimSpace(v)
	}
	if err := singbox_split.ValidateRule(rule, manage.Manager.NodeLen()); err != nil {
		log.Error(err)
		return
	}
	if !saveRuleSet(rs) {
		return
	}
	log.Info("route rule updated: ", formatMatch(rule.Match), " -> ", rule.Target)
}

func removeRouteRule(indexText string) {
	index, ok := parseRuleIndex(indexText)
	if !ok {
		return
	}
	rs, ok := loadRuleSet()
	if !ok {
		return
	}
	if index > len(rs.Rules) {
		log.Warn("rule index out of range")
		return
	}
	removed := rs.Rules[index-1]
	rs.Rules = append(rs.Rules[:index-1], rs.Rules[index:]...)
	if !saveRuleSet(rs) {
		return
	}
	log.Info("route rule removed: ", formatMatch(removed.Match), " -> ", removed.Target)
	showRouteSummary()
}

func clearRouteRules(c *ishell.Context) {
	if !confirm(c, "Clear all route rules?") {
		log.Info("clear route rules canceled")
		return
	}
	rs, ok := loadRuleSet()
	if !ok {
		return
	}
	rs.Rules = make([]*singbox_split.Rule, 0)
	if !saveRuleSet(rs) {
		return
	}
	log.Info("all route rules cleared")
}

// ---------- dns (DNS routing rules) ----------

func formatDNS(rule *singbox_split.DNSRule) string {
	if rule == nil {
		return "empty rule"
	}
	detour := strings.TrimSpace(rule.Detour)
	if detour == "" {
		if rule.Server == "foreign" {
			detour = "proxy"
		} else {
			detour = "direct"
		}
	}
	return fmt.Sprintf("%s -> %s (%s)", formatMatch(rule.Match), rule.Server, detour)
}

func showDNSSummary() {
	rs, ok := loadRuleSet()
	if !ok {
		return
	}
	log.Info("DNS rule count: ", len(rs.DNSRules))
}

func listDNSRules() {
	rs, ok := loadRuleSet()
	if !ok {
		return
	}
	if len(rs.DNSRules) == 0 {
		log.Info("no DNS rules configured")
		return
	}
	for i, rule := range rs.DNSRules {
		if rule == nil {
			log.Warn(fmt.Sprintf("[%d] empty rule", i+1))
			continue
		}
		log.Info(fmt.Sprintf("[%d] %s", i+1, formatDNS(rule)))
	}
}

func addDNSRule(c *ishell.Context) {
	argMap := FlagsParse(c.Args, map[string]string{})
	server := strings.TrimSpace(argMap["server"])
	if server == "" {
		log.Warn("use --server to specify the DNS server alias")
		return
	}
	detour := strings.TrimSpace(argMap["detour"])
	if index, isNode, err := parseNodeTarget(detour); err != nil {
		log.Error(err)
		return
	} else if isNode && manage.Manager.GetNode(index) == nil {
		log.Error(fmt.Errorf("detour node does not exist: index %d", index))
		return
	}
	rule := &singbox_split.DNSRule{
		Server: server,
		Detour: detour,
		Match:  matchFromArgs(argMap),
	}
	if err := singbox_split.ValidateDNSRule(rule, manage.Manager.NodeLen()); err != nil {
		log.Error(err)
		return
	}
	rs, ok := loadRuleSet()
	if !ok {
		return
	}
	rs.DNSRules = append(rs.DNSRules, rule)
	if !saveRuleSet(rs) {
		return
	}
	log.Info("DNS rule added: ", formatDNS(rule))
	showDNSSummary()
}

func editDNSRule(c *ishell.Context) {
	if len(c.Args) < 1 {
		log.Warn("please provide the rule index to edit")
		return
	}
	index, ok := parseRuleIndex(c.Args[0])
	if !ok {
		return
	}
	rs, ok := loadRuleSet()
	if !ok {
		return
	}
	if index > len(rs.DNSRules) {
		log.Warn("rule index out of range")
		return
	}
	rule := rs.DNSRules[index-1]
	if rule == nil {
		log.Warn("this rule is empty")
		return
	}
	argMap := FlagsParse(c.Args[1:], map[string]string{})
	editMatch(&rule.Match, argMap)
	if v, ok := argMap["server"]; ok {
		rule.Server = strings.TrimSpace(v)
	}
	if v, ok := argMap["detour"]; ok {
		detour := strings.TrimSpace(v)
		if idx, isNode, err := parseNodeTarget(detour); err != nil {
			log.Error(err)
			return
		} else if isNode && manage.Manager.GetNode(idx) == nil {
			log.Error(fmt.Errorf("detour node does not exist: index %d", idx))
			return
		}
		rule.Detour = detour
	}
	if err := singbox_split.ValidateDNSRule(rule, manage.Manager.NodeLen()); err != nil {
		log.Error(err)
		return
	}
	if !saveRuleSet(rs) {
		return
	}
	log.Info("DNS rule updated: ", formatDNS(rule))
}

func removeDNSRule(indexText string) {
	index, ok := parseRuleIndex(indexText)
	if !ok {
		return
	}
	rs, ok := loadRuleSet()
	if !ok {
		return
	}
	if index > len(rs.DNSRules) {
		log.Warn("rule index out of range")
		return
	}
	removed := rs.DNSRules[index-1]
	rs.DNSRules = append(rs.DNSRules[:index-1], rs.DNSRules[index:]...)
	if !saveRuleSet(rs) {
		return
	}
	log.Info("DNS rule removed: ", formatDNS(removed))
	showDNSSummary()
}

func clearDNSRules(c *ishell.Context) {
	if !confirm(c, "Clear all DNS rules?") {
		log.Info("clear DNS rules canceled")
		return
	}
	rs, ok := loadRuleSet()
	if !ok {
		return
	}
	rs.DNSRules = make([]*singbox_split.DNSRule, 0)
	if !saveRuleSet(rs) {
		return
	}
	log.Info("all DNS rules cleared")
}

// ---------- tab completion ----------

var routeAddFlags = []string{
	"--target", "--action",
	"--domain", "--domain-suffix", "--ip-cidr",
	"--geosite", "--match-outbound", "--inbound",
}

var routeEditFlags = []string{
	"--target", "--action",
	"--add-domain", "--rm-domain",
	"--add-domain-suffix", "--rm-domain-suffix",
	"--add-ip-cidr", "--rm-ip-cidr",
	"--add-geosite", "--rm-geosite",
	"--add-match-outbound", "--rm-match-outbound",
	"--add-inbound", "--rm-inbound",
}

var dnsAddFlags = []string{
	"--server", "--detour",
	"--domain", "--domain-suffix", "--domain-keyword",
	"--rule-set", "--ip-cidr", "--query-type", "--inbound",
}

var dnsEditFlags = []string{
	"--server", "--detour",
	"--add-domain", "--rm-domain",
	"--add-domain-suffix", "--rm-domain-suffix",
	"--add-domain-keyword", "--rm-domain-keyword",
	"--add-rule-set", "--rm-rule-set",
	"--add-ip-cidr", "--rm-ip-cidr",
	"--add-query-type", "--rm-query-type",
	"--add-inbound", "--rm-inbound",
}

// flagCompleter completes --xxx flag names that have not been used yet.
func flagCompleter(flags []string) func([]string) []string {
	return func(args []string) []string {
		used := make(map[string]struct{}, len(args))
		for _, a := range args {
			used[a] = struct{}{}
		}
		result := make([]string, 0, len(flags))
		for _, f := range flags {
			if _, ok := used[f]; !ok {
				result = append(result, f)
			}
		}
		return result
	}
}

// ---------- registration ----------

func InitRuleShell(shell *ishell.Shell) {
	ruleCmd := &ishell.Cmd{
		Name: "rule",
		Help: "manage sing-box split routing rules (route/dns)",
		Func: func(c *ishell.Context) {
			c.Println(help.Rule)
		},
	}
	ruleCmd.AddCmd(&ishell.Cmd{
		Name:    "help",
		Aliases: []string{"-h", "--help"},
		Func: func(c *ishell.Context) {
			c.Println(help.Rule)
		},
	})
	ruleCmd.AddCmd(buildRouteCmd())
	ruleCmd.AddCmd(buildDNSCmd())
	shell.AddCmd(ruleCmd)
}

func buildRouteCmd() *ishell.Cmd {
	routeCmd := &ishell.Cmd{
		Name: "route",
		Help: "traffic routing rules",
		Func: func(c *ishell.Context) {
			showRouteSummary()
		},
	}
	routeCmd.AddCmd(&ishell.Cmd{
		Name: "ls",
		Func: func(c *ishell.Context) { listRouteRules() },
	})
	routeCmd.AddCmd(&ishell.Cmd{
		Name:      "add",
		Func:      func(c *ishell.Context) { addRouteRule(c) },
		Completer: flagCompleter(routeAddFlags),
	})
	routeCmd.AddCmd(&ishell.Cmd{
		Name:      "edit",
		Func:      func(c *ishell.Context) { editRouteRule(c) },
		Completer: flagCompleter(routeEditFlags),
	})
	routeCmd.AddCmd(&ishell.Cmd{
		Name: "rm",
		Func: func(c *ishell.Context) {
			if len(c.Args) < 1 {
				log.Warn("please provide the rule index to remove")
				return
			}
			removeRouteRule(c.Args[0])
		},
	})
	routeCmd.AddCmd(&ishell.Cmd{
		Name: "clear",
		Func: func(c *ishell.Context) { clearRouteRules(c) },
	})
	return routeCmd
}

func buildDNSCmd() *ishell.Cmd {
	dnsCmd := &ishell.Cmd{
		Name: "dns",
		Help: "DNS routing rules",
		Func: func(c *ishell.Context) {
			showDNSSummary()
		},
	}
	dnsCmd.AddCmd(&ishell.Cmd{
		Name: "ls",
		Func: func(c *ishell.Context) { listDNSRules() },
	})
	dnsCmd.AddCmd(&ishell.Cmd{
		Name:      "add",
		Func:      func(c *ishell.Context) { addDNSRule(c) },
		Completer: flagCompleter(dnsAddFlags),
	})
	dnsCmd.AddCmd(&ishell.Cmd{
		Name:      "edit",
		Func:      func(c *ishell.Context) { editDNSRule(c) },
		Completer: flagCompleter(dnsEditFlags),
	})
	dnsCmd.AddCmd(&ishell.Cmd{
		Name: "rm",
		Func: func(c *ishell.Context) {
			if len(c.Args) < 1 {
				log.Warn("please provide the rule index to remove")
				return
			}
			removeDNSRule(c.Args[0])
		},
	})
	dnsCmd.AddCmd(&ishell.Cmd{
		Name: "clear",
		Func: func(c *ishell.Context) { clearDNSRules(c) },
	})
	return dnsCmd
}
