package node

import (
	"tbox/core/protocols"
	"fmt"
	"net"
	"strings"
	"time"
)

type Node struct {
	protocols.Protocol `json:"-"`
	SubID              string  `json:"sub_id"`
	Data               string  `json:"data"`
	TestResult         float64 `json:"-"`
}

func (n *Node) TestResultStr() string {
	if n.TestResult == 0 {
		return ""
	} else if n.TestResult >= 99998 {
		return "-1ms"
	} else {
		return fmt.Sprintf("%.4vms", n.TestResult)
	}
}

// NewNode creates a Node from a share link.
func NewNode(link, id string) *Node {
	if data := protocols.ParseLink(link); data != nil {
		return &Node{Protocol: data, SubID: id}
	}
	return nil
}

func NewNodeByData(protocol protocols.Protocol) *Node {
	return &Node{Protocol: protocol}
}

// ParseData deserializes Data back into a Protocol.
func (n *Node) ParseData() {
	n.Protocol = protocols.ParseLink(n.Data)
}

// Serialize2Data serializes the Protocol into Data.
func (n *Node) Serialize2Data() {
	n.Data = n.GetLink()
}

func (n *Node) Tcping() {
	count := 3
	var sum float64
	var timeout time.Duration = 3 * time.Second
	isTimeout := false
	for i := 0; i < count; i++ {
		start := time.Now()
		d := net.Dialer{Timeout: timeout}
		conn, err := d.Dial("tcp", fmt.Sprintf("%s:%d", n.GetAddr(), n.GetPort()))
		if err != nil {
			isTimeout = true
			break
		}
		conn.Close()
		elapsed := time.Since(start)
		sum += float64(elapsed.Nanoseconds()) / 1e6
	}
	if isTimeout {
		n.TestResult = 99999
	} else {
		n.TestResult = sum / float64(count)
	}
}

func (n *Node) Show() {
	ShowTopBottomSepLine('=', strings.Split(n.GetInfo(), "\n")...)
}
