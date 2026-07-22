package manage

import "tbox/core/node"

func (m *Manage) SelectedNode() *node.Node {
	return m.GetNode(m.SelectedIndex())
}

func (m *Manage) SelectedIndex() int {
	return m.Index
}

func (m *Manage) SetSelectedIndex(index int) int {
	m.Index = index
	return m.Index
}

func (m *Manage) SetSelectedIndexByNode(n *node.Node) int {
	m.SetSelectedIndex(1)
	if n != nil {
		m.NodeForEach(func(i int, n1 *node.Node) {
			if n == n1 {
				m.SetSelectedIndex(i)
				return
			}
		})
	}
	return m.Index
}

// SetSelectedIndexByLink restores the selected node by matching its share link content.
// Updating a subscription deletes and rebuilds nodes so the original pointer becomes stale;
// matching by link content is the only reliable option.
// Selects the first node if no match is found.
func (m *Manage) SetSelectedIndexByLink(link string) int {
	m.SetSelectedIndex(1)
	if link != "" {
		m.NodeForEach(func(i int, n *node.Node) {
			if n != nil && n.GetLink() == link {
				m.SetSelectedIndex(i)
				return
			}
		})
	}
	return m.Index
}
