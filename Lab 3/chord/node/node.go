package node

import (
	"chord/nodeinfo"
	"math/big"
	"strconv"
	"strings"
	"sync"
)

// Constants
const M = 8 // SHA-1 output size in bits
const R = 2 // Number of successors to store for each node

// Node represents a Chord node.
type Node struct {
	ID           *big.Int
	Address      string
	Successor    *nodeinfo.NodeInfo
	SuccessorList []*nodeinfo.NodeInfo
	Predecessor  *nodeinfo.NodeInfo
	FingerTable  []*nodeinfo.NodeInfo
	Files        map[string]string
	mutex        sync.Mutex
}

func NewNode(address string) *Node {
	// !! ACTUAL CODE UNCOMMENT THIS LATER !!
	// hash := sha1.Sum([]byte(address))
	// !! ACTUAL CODE UNCOMMENT THIS LATER !!

	// !! TEST CODE !!
	// split by : and take the last part
	parts := strings.Split(address, ":")
	port := parts[len(parts)-1]
	// convert to int
	portInt, _ := strconv.Atoi(port)
	portInt = portInt % 11
	// convert to big int
	portIntBig := big.NewInt(int64(portInt))
	// !! TEST CODE !!

	node := &Node{
		ID:          portIntBig,
		Address:     address,
		FingerTable: make([]*nodeinfo.NodeInfo, M),
		Files:       make(map[string]string),
	}
	node.Successor = &nodeinfo.NodeInfo{ID: node.ID, Address: address}
	node.Predecessor = nil
	return node
}

func (n *Node) CreateRing() {
	n.Successor = &nodeinfo.NodeInfo{ID: n.ID, Address: n.Address}
	n.Predecessor = nil
}