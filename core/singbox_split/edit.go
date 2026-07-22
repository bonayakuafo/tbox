package singbox_split

import "strings"

// MergeList appends entries from adds to base that are not already present
// (trims whitespace, dedupes), returning a new slice.
// Used by rule edit to incrementally add match conditions (e.g. another domain)
// without touching existing entries.
func MergeList(base, adds []string) []string {
	seen := make(map[string]struct{}, len(base)+len(adds))
	result := make([]string, 0, len(base)+len(adds))
	appendUnique := func(list []string) {
		for _, v := range list {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	appendUnique(base)
	appendUnique(adds)
	return result
}

// RemoveList removes entries from base that appear in removes (compared after trimming whitespace).
func RemoveList(base, removes []string) []string {
	if len(removes) == 0 {
		return base
	}
	rm := make(map[string]struct{}, len(removes))
	for _, v := range removes {
		rm[strings.TrimSpace(v)] = struct{}{}
	}
	result := make([]string, 0, len(base))
	for _, v := range base {
		if _, ok := rm[strings.TrimSpace(v)]; ok {
			continue
		}
		result = append(result, v)
	}
	return result
}
