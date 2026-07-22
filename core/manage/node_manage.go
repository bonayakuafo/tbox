package manage

import (
	"tbox/core"
	"tbox/core/node"
	"tbox/log"
	"sync"
)

// NodeLen returns the number of nodes.
func (m *Manage) NodeLen() int {
	return len(m.NodeList)
}

// GetNode returns a node by 1-based index.
func (m *Manage) GetNode(index int) *node.Node {
	return m.getNode(index - 1)
}

// getNode returns a node by 0-based index.
func (m *Manage) getNode(i int) *node.Node {
	if i < m.NodeLen() && i >= 0 {
		return m.NodeList[i]
	}
	return nil
}

func (m *Manage) addNode(n *node.Node) bool {
	if n == nil {
		return false
	}
	if f := m.IsCanFilter(n); f != nil {
		log.Infof("rule [%s] filtered node ==> %s", f.String(), n.GetName())
		return false
	}
	n.Serialize2Data()
	m.NodeList = append(m.NodeList, n)
	return true
}

func (m *Manage) AddNode(n *node.Node) bool {
	ok := false
	if ok = m.addNode(n); ok {
		m.Save()
	}
	return ok
}

func (m *Manage) NodeForEach(funC func(int, *node.Node)) {
	for i, n := range m.NodeList {
		funC(i+1, n)
	}
}

func (m *Manage) Tcping() {
	var wg sync.WaitGroup
	m.NodeForEach(func(i int, n *node.Node) {
		wg.Add(1)
		go func(n *node.Node) {
			defer wg.Done()
			n.Tcping()
		}(n)
	})
	wg.Wait()
	defer m.Save()
	m.NodeSort(func(n1 *node.Node, n2 *node.Node) bool {
		return n1.TestResult < n2.TestResult
	})
	m.SetSelectedIndex(1)
}

func (m *Manage) NodeSort(less func(*node.Node, *node.Node) bool) {
	if m.NodeLen() <= 1 {
		return
	}
	for i := 1; i < m.NodeLen(); i++ {
		preIndex := i - 1
		current := m.getNode(i)
		for preIndex >= 0 && !less(m.getNode(preIndex), current) {
			m.NodeList[preIndex+1] = m.NodeList[preIndex]
			preIndex -= 1
		}
		m.NodeList[preIndex+1] = current
	}
}

func (m *Manage) Sort(mode int) {
	selectedNode := m.SelectedNode()
	switch mode {
	case 0:
		for i := 0; i < m.NodeLen()/2; i++ {
			j := m.NodeLen() - i - 1
			m.NodeList[i], m.NodeList[j] = m.NodeList[j], m.NodeList[i]
		}
	case 1:
		m.NodeSort(func(n1 *node.Node, n2 *node.Node) bool {
			return n1.GetProtocolMode() < n2.Protocol.GetProtocolMode()
		})
	case 2:
		m.NodeSort(func(n1 *node.Node, n2 *node.Node) bool {
			return n1.GetName() < n2.GetName()
		})
	case 3:
		m.NodeSort(func(n1 *node.Node, n2 *node.Node) bool {
			return n1.GetAddr() < n2.GetAddr()
		})
	case 4:
		m.NodeSort(func(n1 *node.Node, n2 *node.Node) bool {
			return n1.GetPort() < n2.GetPort()
		})
	case 5:
		m.NodeSort(func(n1 *node.Node, n2 *node.Node) bool {
			return n1.TestResult < n2.TestResult
		})
	default:
		return
	}
	defer m.Save()
	m.SetSelectedIndexByNode(selectedNode)
}

func (m *Manage) DelNode(key string) {
	indexList := core.IndexList(key, m.NodeLen())
	if len(indexList) == 0 {
		return
	}
	defer m.Save()
	selectedNode := m.SelectedNode()
	newNodeList := make([]*node.Node, 0)
	m.NodeForEach(func(i int, n *node.Node) {
		if !HasIn(i, indexList) {
			newNodeList = append(newNodeList, n)
		}
	})
	m.NodeList = newNodeList
	m.SetSelectedIndexByNode(selectedNode)
}

func (m *Manage) DelNodeById(id string) {
	defer m.Save()
	selectedNode := m.SelectedNode()
	newNodeList := make([]*node.Node, 0)
	m.NodeForEach(func(i int, n *node.Node) {
		if n.SubID != id {
			newNodeList = append(newNodeList, n)
		}
	})
	m.NodeList = newNodeList
	m.SetSelectedIndexByNode(selectedNode)
}

func (m *Manage) GetNodeLink(key string) []string {
	links := make([]string, 0)
	for _, index := range core.IndexList(key, m.NodeLen()) {
		links = append(links, m.GetNode(index).GetLink())
	}
	return links
}
