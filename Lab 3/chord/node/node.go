package node

import (
	"chord/nodeinfo"
	"crypto/sha1"
	"math/big"
	"sync"
)

// Constants
const M = 16 // SHA-1 output size in bits
const R = 2 // Number of successors to store for each node

// Node represents a Chord node.
type Node struct {
	ID           *big.Int
	Address      string
	Successor    *nodeinfo.NodeInfo
	Predecessor  *nodeinfo.NodeInfo
	FingerTable  []*nodeinfo.NodeInfo
	Files        map[string]string
	mutex        sync.Mutex
}

func NewNode(address string) *Node {
	hash := sha1.Sum([]byte(address))
	maxID := new(big.Int).Exp(big.NewInt(2), big.NewInt(M), nil)

	node := &Node{
		ID:          new(big.Int).Mod(new(big.Int).SetBytes(hash[:]), maxID),
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