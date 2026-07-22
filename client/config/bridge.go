package config

import (
	"tbox/core/manage"
	"tbox/core/protocols"
	"tbox/core/setting"
	"tbox/core/singbox_split"
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

// converterTag is the sing-box socks outbound tag pointing at the converter
// (xray/ssr) in bridge mode. In shape 2 (main node normal + split target is
// the converter node) split rules use this tag to reference the converter;
// in shape 1 (main node itself is the converter node) the proxy outbound
// itself is the bridge, and converterTag is equivalent to proxy.
const converterTag = "converter-out"

// BridgePlan describes the bridge decision for one run. The converter node is
// unique per run: it may be the main (running) node, or appear only in
// split/DNS targets, but it must always be the same node.
type BridgePlan struct {
	// Enabled indicates this run needs the "sing-box + converter" bridge.
	Enabled bool
	// ConverterCore is the converter core name ("xray" or "ssr").
	ConverterCore string
	// NodeIndex is the 1-based index of the converter node (used when
	// generating the converter config).
	NodeIndex int
	// Node is the converter node's protocol object.
	Node protocols.Protocol
	// MainIsConverter is true when the main (running) node is itself the
	// converter node (shape 1); false when the main node is a normal protocol
	// and the converter only serves split/DNS targets (shape 2).
	MainIsConverter bool
}

// converterCandidate is a node that needs a converter (xhttp or ssr).
type converterCandidate struct {
	index     int
	converter string // "xray" | "ssr"
}

// nodeConverter returns the converter core name required by this node, or an
// empty string for normal nodes:
//   - SSR node             -> "ssr" (newer sing-box has removed SSR outbound)
//   - xhttp transport node -> "xray" (sing-box does not support xhttp)
func nodeConverter(node protocols.Protocol) string {
	if node == nil {
		return ""
	}
	if node.GetProtocolMode() == protocols.ModeShadowSocksR {
		return "ssr"
	}
	if nodeUsesXhttp(node) {
		return "xray"
	}
	return ""
}

// splitTargetNodeIndex parses the 1-based node index out of a "node:N"
// target; returns 0 for any other form.
func splitTargetNodeIndex(target string) int {
	target = strings.TrimSpace(target)
	if !strings.HasPrefix(target, "node:") {
		return 0
	}
	index, err := strconv.Atoi(strings.TrimPrefix(target, "node:"))
	if err != nil || index <= 0 {
		return 0
	}
	return index
}

// ResolveBridgePlan scans the main (running) node, all split --target node:N,
// and all DNS --detour node:N, collects the xhttp/ssr nodes and deduplicates
// them, yielding the unique converter node.
//
// Returns:
//   - 0 converter candidates            -> BridgePlan{Enabled:false}, nil
//   - exactly 1 unique node             -> populated BridgePlan, nil
//   - >=2 distinct converter nodes      -> nil plan, error (forbidden: split
//     targets must reference the same node as main / each other)
//
// Special case: when xhttp is the main node (not a split target), bridging is
// only required under TUN (otherwise we just run xray solo). This function
// only handles the "uniqueness" check; whether we actually bridge depends on
// the main-node trigger conditions and whether split converter targets exist,
// per the logic below.
func ResolveBridgePlan(mainNode protocols.Protocol, preferred string) (*BridgePlan, error) {
	candidates := make([]converterCandidate, 0)
	seen := make(map[int]struct{})

	addCandidate := func(index int, node protocols.Protocol) {
		conv := nodeConverter(node)
		if conv == "" {
			return
		}
		if _, ok := seen[index]; ok {
			return
		}
		seen[index] = struct{}{}
		candidates = append(candidates, converterCandidate{index: index, converter: conv})
	}

	// Main node: use the currently running node's actual index.
	mainIndex := manage.Manager.SelectedIndex()
	mainIsConverterNode := nodeConverter(mainNode) != ""
	if mainIsConverterNode {
		addCandidate(mainIndex, mainNode)
	}

	// Split / DNS target nodes.
	if rs, err := singbox_split.Load(); err == nil && rs != nil {
		for _, rule := range rs.Rules {
			if rule == nil {
				continue
			}
			if idx := splitTargetNodeIndex(rule.Target); idx > 0 {
				addCandidate(idx, nodeProtocolAt(idx))
			}
		}
		for _, rule := range rs.DNSRules {
			if rule == nil {
				continue
			}
			if idx := splitTargetNodeIndex(rule.Detour); idx > 0 {
				addCandidate(idx, nodeProtocolAt(idx))
			}
		}
	}

	if len(candidates) == 0 {
		return &BridgePlan{Enabled: false}, nil
	}
	if len(candidates) > 1 {
		return nil, fmt.Errorf("detected multiple distinct xhttp/ssr nodes (indexes %s); the main node and split targets must use the same converter node",
			formatCandidateIndexes(candidates))
	}

	c := candidates[0]
	node := c.index
	mainIsConverter := mainIsConverterNode && node == mainIndex

	// Whether we actually need the bridge:
	//   - ssr: always bridge (sing-box cannot outbound SSR).
	//   - xray: always bridge when it is a split target (main core is sing-box);
	//     only when it is the main node and TUN is disabled can we run xray solo without a bridge.
	needBridge := true
	if c.converter == "xray" && mainIsConverter && len(candidates) == 1 {
		// xhttp main node, and no other split converter targets: bridge only under TUN.
		needBridge = runtime.GOOS == "linux" && setting.TunMode()
	}
	if !needBridge {
		return &BridgePlan{Enabled: false}, nil
	}

	plan := &BridgePlan{
		Enabled:         true,
		ConverterCore:   c.converter,
		NodeIndex:       node,
		Node:            nodeProtocolAt(node),
		MainIsConverter: mainIsConverter,
	}
	return plan, nil
}

// nodeProtocolAt returns the protocol object at the given 1-based index, or nil if not found.
func nodeProtocolAt(index int) protocols.Protocol {
	n := manage.Manager.GetNode(index)
	if n == nil {
		return nil
	}
	return n.Protocol
}

func formatCandidateIndexes(cands []converterCandidate) string {
	parts := make([]string, 0, len(cands))
	for _, c := range cands {
		parts = append(parts, strconv.Itoa(c.index))
	}
	return strings.Join(parts, ", ")
}
