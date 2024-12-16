package node

import (
	"chord/helpers"
	"chord/nodeinfo"
	"crypto/sha1"
	"fmt"
	"math/big"
	"net/rpc"
	"os"
)

// Join allows a node to join an existing Chord ring.
// the node that wants to join is the one that is calling this function, and the remoteAddress is the address of the node that is already in the ring
func (n *Node) Join(remoteAddress string) error {
	client, _ := helpers.Dial(remoteAddress)

	// find the successor of the current node
	var successor nodeinfo.NodeInfo
	err := client.Call("Node.FindSuccessor", n.ID, &successor)
	if err != nil {
		return fmt.Errorf("failed to find successor: %w", err)
	}
	n.mutex.Lock()
	n.Successor = &successor
	n.mutex.Unlock()

	// notify the successor that the current node has joined the ring, and update the predecessor of the successor
	client, _ = helpers.Dial(successor.Address)
	err = client.Call("Node.Notify", &nodeinfo.NodeInfo{ID: n.ID, Address: n.Address}, nil)
	if err != nil { return fmt.Errorf("failed to notify successor: %w", err) }
	defer client.Close()

	return nil
}

// Stabilize periodically verifies the successor and updates pointers.
func (n *Node) Stabilize() error {
	n.mutex.Lock()
	successor := n.Successor
	n.mutex.Unlock()

	var predecessor nodeinfo.NodeInfo
	err := successor.GetPredecessorRPC(&predecessor)
	if err != nil {
		return fmt.Errorf("error getting predecessor: %w", err)
	}

	n.mutex.Lock()
	if predecessor.ID != nil && helpers.Between(n.ID, predecessor.ID, n.Successor.ID) {
		n.Successor = &predecessor
	}
	n.mutex.Unlock()

	successor = n.Successor
	if successor != nil && successor.ID.Cmp(n.ID) != 0 {
		successor.NotifyRPC(&nodeinfo.NodeInfo{ID: n.ID, Address: n.Address}, nil)
	}
	return nil
}

// FindSuccessor finds the successor node for a given ID.
func (n *Node) FindSuccessor(id *big.Int, successor *nodeinfo.NodeInfo) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	// If the current node is the only node in the ring
	if n.Successor.ID == n.ID {
		*successor = *n.Successor
		return nil
	}
	
	if helpers.Between(n.ID, id, n.Successor.ID) {
		*successor = *n.Successor
		return nil
	}

	// Avoid recursion if Successor is nil
	if n.Successor == nil { return fmt.Errorf("successor is nil") }

	return n.Successor.FindSuccessorRPC(id, successor)
}

// Notify is called to update predecessor pointers.
func (n *Node) Notify(candidate *nodeinfo.NodeInfo, _ *struct{}) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.Predecessor == nil || helpers.Between(n.Predecessor.ID, candidate.ID, n.ID) {
		n.Predecessor = candidate
	}
	return nil
}

// FixFingers periodically updates the finger table.
func (n *Node) FixFingers() error {
	for i := 0; i < M; i++ {
		offset := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(i)), nil)
		nextID := new(big.Int).Add(n.ID, offset)
		nextID.Mod(nextID, big.NewInt(0).Exp(big.NewInt(2), big.NewInt(M), nil))
		var successor nodeinfo.NodeInfo
		err := n.FindSuccessor(nextID, &successor)
		if err != nil { return fmt.Errorf("failed to find successor: %w", err) }
		n.mutex.Lock()
		n.FingerTable[i] = &successor
		n.mutex.Unlock()
	}
	return nil
}

// GetPredecessor is called to get the predecessor of the current node
func (n *Node) GetPredecessor(_ *struct{}, predecessor *nodeinfo.NodeInfo) error {
	if n.Predecessor == nil { return nil }
	*predecessor = *n.Predecessor
	return nil
}

// CheckPredecessor verifies that the predecessor is alive.
func (n *Node) CheckPredecessor() {
	n.mutex.Lock()
	predecessor := n.Predecessor
	n.mutex.Unlock()

	if predecessor != nil {
		client, err := rpc.Dial("tcp", predecessor.Address)
		if err != nil {
			n.mutex.Lock()
			n.Predecessor = nil
			n.mutex.Unlock()
			return
		}
		defer client.Close()
	}
}

// Lookup hashes a file name and finds its successor.
func (n *Node) Lookup(fileName string) (*nodeinfo.NodeInfo, error) {
	hash := sha1.Sum([]byte(fileName))
	key := new(big.Int).SetBytes(hash[:])
	var successor nodeinfo.NodeInfo
	err := n.FindSuccessor(key, &successor)
	if err != nil { return nil, err }
	return &successor, nil
}

// StoreFile stores a file in the Chord ring.
func (n *Node) StoreFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil { return err }
	successor, err := n.Lookup(filePath)
	if err != nil { return err }
	client, err := rpc.Dial("tcp", successor.Address)
	if err != nil { return err }
	defer client.Close()

	var ack bool
	return client.Call("Node.ReceiveFile", map[string]string{filePath: string(content)}, &ack)
}

// ReceiveFile RPC to accept file storage.
func (n *Node) ReceiveFile(file map[string]string, ack *bool) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	for key, content := range file {
		n.Files[key] = content
	}
	*ack = true
	return nil
}

// PrintState prints the state of the current node.
func (n *Node) PrintState() {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	fmt.Printf("Node ID: %s\n", n.ID.Text(16))
	fmt.Printf("Address: %s\n", n.Address)
	fmt.Printf("Successor ID: %s\n", n.Successor.ID.Text(16))
	if n.Predecessor != nil {
		fmt.Printf("Predecessor ID: %s\n", n.Predecessor.ID.Text(16))
	} else {
		fmt.Println("Predecessor: nil")
	}
	fmt.Println("Finger Table:")
	for i, finger := range n.FingerTable {
		if finger != nil {
			fmt.Printf("%d: %s (%s)\n", i, finger.ID.Text(16), finger.Address)
		}
	}
}
