package singbox_split

import (
	"tbox/core"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var RuleFile = filepath.Join(core.GetConfigDir(), "tbox.rules.json")

// init migrates the legacy rules file singbox_split_rules.json to tbox.rules.json.
// Renames only when the new file is absent and the old file exists (one-shot,
// lossless upgrade).
func init() {
	oldPath := filepath.Join(core.GetConfigDir(), "singbox_split_rules.json")
	if _, err := os.Stat(RuleFile); err == nil {
		return
	}
	if _, err := os.Stat(oldPath); err != nil {
		return
	}
	_ = os.Rename(oldPath, RuleFile)
}

func Load() (*RuleSet, error) {
	if _, err := os.Stat(RuleFile); os.IsNotExist(err) {
		return NewRuleSet(), nil
	}

	file, err := os.Open(RuleFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	rs := NewRuleSet()
	if err = json.NewDecoder(file).Decode(rs); err != nil {
		return nil, err
	}
	if rs.Rules == nil {
		rs.Rules = make([]*Rule, 0)
	}
	if rs.DNSRules == nil {
		rs.DNSRules = make([]*DNSRule, 0)
	}
	for _, rule := range rs.Rules {
		if rule == nil {
			continue
		}
		if len(rule.Match.Geosite) > 0 && len(rule.Match.RuleSet) == 0 {
			rule.Match.RuleSet = append([]string{}, rule.Match.Geosite...)
		}
	}
	for _, rule := range rs.DNSRules {
		if rule == nil {
			continue
		}
		if len(rule.Match.Geosite) > 0 && len(rule.Match.RuleSet) == 0 {
			rule.Match.RuleSet = append([]string{}, rule.Match.Geosite...)
		}
	}
	return rs, nil
}

func Save(rs *RuleSet) error {
	if rs == nil {
		rs = NewRuleSet()
	}
	if rs.Rules == nil {
		rs.Rules = make([]*Rule, 0)
	}
	if rs.DNSRules == nil {
		rs.DNSRules = make([]*DNSRule, 0)
	}

	tempFile, err := os.CreateTemp(filepath.Dir(RuleFile), filepath.Base(RuleFile)+".*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer func() {
		tempFile.Close()
		os.Remove(tempPath)
	}()

	encoder := json.NewEncoder(tempFile)
	encoder.SetIndent("", "\t")
	if err = encoder.Encode(rs); err != nil {
		return err
	}
	if err = tempFile.Sync(); err != nil {
		return err
	}
	if err = tempFile.Close(); err != nil {
		return err
	}
	if err = os.Rename(tempPath, RuleFile); err != nil {
		return fmt.Errorf("failed to save rules file: %w", err)
	}
	return nil
}

func Init() *RuleSet {
	rs, err := Load()
	if err != nil {
		rs = NewRuleSet()
	}
	if rs.Rules == nil {
		rs.Rules = make([]*Rule, 0)
	}
	if rs.DNSRules == nil {
		rs.DNSRules = make([]*DNSRule, 0)
	}
	return rs
}
