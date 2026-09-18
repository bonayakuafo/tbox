package manage

import (
	"tbox/core"
	"tbox/core/node"
	"tbox/core/protocols"
	"tbox/core/sub"
	"tbox/log"
	"encoding/json"
	"os"
)

type Manage struct {
	Subs     []*sub.Subscirbe   `json:"subs"`
	Index    int                `json:"index"`
	NodeList []*node.Node       `json:"nodes"`
	Filter   []*node.NodeFilter `json:"filter"`
}

var Manager *Manage

// initialize
func init() {
	Manager = NewManage()
	if _, err := os.Stat(core.DataFile); os.IsNotExist(err) {
		Manager.ensureDirectNode()
		Manager.Save()
	} else {
		file, err := os.Open(core.DataFile)
		if err != nil {
			log.Error(err)
			return
		}
		defer file.Close()
		err = json.NewDecoder(file).Decode(Manager)
		if err != nil {
			log.Error(err)
		}
		Manager.NodeForEach(func(i int, n *node.Node) {
			if n != nil {
				n.ParseData()
			}
		})
		Manager.ensureDirectNode()
		Manager.Save()
	}
}

func (m *Manage) ensureDirectNode() {
	for _, n := range m.NodeList {
		if n != nil && n.Protocol != nil && n.Protocol.GetProtocolMode() == protocols.ModeDirect {
			return
		}
	}
	direct := node.NewNodeByData(&protocols.Direct{})
	direct.Serialize2Data()
	m.NodeList = append(m.NodeList, direct)
}

func NewManage() *Manage {
	return &Manage{
		NodeList: make([]*node.Node, 0),
		Subs:     make([]*sub.Subscirbe, 0),
		Index:    0,
		Filter:   make([]*node.NodeFilter, 0),
	}
}

// Save persists the state to disk.
func (m *Manage) Save() {
	err := core.WriteJSON(m, core.DataFile)
	if err != nil {
		log.Error(err)
	}
}
