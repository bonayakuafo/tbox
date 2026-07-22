package singbox_split

type RuleSet struct {
	Rules    []*Rule    `json:"rules"`
	DNSRules []*DNSRule `json:"dns_rules,omitempty"`
}

type Rule struct {
	Match  Match  `json:"match"`
	Target string `json:"target"`
	Action string `json:"action,omitempty"`
}

type DNSRule struct {
	Match  Match  `json:"match"`
	Server string `json:"server"`
	Detour string `json:"detour,omitempty"`
}

type Match struct {
	Inbound       []string `json:"inbound,omitempty"`
	IPCIDR        []string `json:"ip_cidr,omitempty"`
	Domain        []string `json:"domain,omitempty"`
	DomainSuffix  []string `json:"domain_suffix,omitempty"`
	DomainKeyword []string `json:"domain_keyword,omitempty"`
	Geosite       []string `json:"geosite,omitempty"`
	RuleSet       []string `json:"rule_set,omitempty"`
	QueryType     []string `json:"query_type,omitempty"`
	MatchOutbound []string `json:"match_outbound,omitempty"`
}

func NewRuleSet() *RuleSet {
	return &RuleSet{
		Rules:    make([]*Rule, 0),
		DNSRules: make([]*DNSRule, 0),
	}
}
