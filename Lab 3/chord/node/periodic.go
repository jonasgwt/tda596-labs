package node

import (
	"fmt"
	"time"
)

// StartStabilize periodically stabilizes the node.
func (n *Node) StartStabilize(interval int) {
	for {
		err := n.Stabilize()
		if err != nil {
			fmt.Printf("Error stabilizing: %v\n", err)
		}
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
}

// StartFixFingers periodically updates the finger table.
func (n *Node) StartFixFingers(interval int) {
	for {
		err := n.FixFingers()
		if err != nil {
			fmt.Printf("Error fixing fingers: %v\n", err)
		}
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
}

// StartCheckPredecessor periodically checks if the predecessor is alive.
func (n *Node) StartCheckPredecessor(interval int) {
	for {
		n.CheckPredecessor()
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
}
