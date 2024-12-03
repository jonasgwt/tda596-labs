package main

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"math/big"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const M = 8 // SHA-1 output size in bits
const R = 2 // Number of successors to store for each node

////////////////////////////////////////////////////////////
//Type Definitions
////////////////////////////////////////////////////////////

// Node represents a Chord node.
type Node struct {
	ID          *big.Int
	Address     string
	Successor   *NodeInfo
	SuccessorList []*NodeInfo
	Predecessor *NodeInfo
	FingerTable []*NodeInfo
	Files       map[string]string // Map of hashed keys to file data
	mutex       sync.Mutex
}

type NodeInfo struct {
	ID *big.Int
	Address string
}

////////////////////////////////////////////////////////////
// Constructor Functions
////////////////////////////////////////////////////////////

// NewNode initializes a new Chord node.
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
		FingerTable: make([]*NodeInfo, M),
		Files:       make(map[string]string),
	}
	node.Successor = &NodeInfo{ID: node.ID, Address: address}
	node.Predecessor = nil
	return node
}

////////////////////////////////////////////////////////////
// Helper Functions
////////////////////////////////////////////////////////////

// between checks if a given ID lies between two other IDs in the circular ID space.
func between(start, id, end *big.Int) bool {
	if start.Cmp(end) < 0 {
		// Normal range
		return start.Cmp(id) < 0 && id.Cmp(end) <= 0
	}
	// Wrapped around the ring
	return start.Cmp(id) < 0 || id.Cmp(end) <= 0
}

// Dial a node and return the client
func Dial(address string) (*rpc.Client, error) {
	client, err := rpc.Dial("tcp", address)
	if err != nil { return nil, fmt.Errorf("failed to connect to node: %w", err) }
	return client, nil
}


////////////////////////////////////////////////////////////
// RPC Methods
////////////////////////////////////////////////////////////

// Join allows a node to join an existing Chord ring.
// the node that wants to join is the one that is calling this function, and the remoteAddress is the address of the node that is already in the ring
func (n *Node) Join(remoteAddress string) error {
	client, _ := Dial(remoteAddress)

	// TODO: what happens if the node that is calling this function dies here?

	// find the successor of the current node
	var successor NodeInfo
	err := client.Call("Node.FindSuccessor", n.ID, &successor)
	if err != nil {
		return fmt.Errorf("failed to find successor: %w", err)
	}
	n.mutex.Lock()
	n.Successor = &successor
	n.mutex.Unlock()

	// notify the successor that the current node has joined the ring, and update the predecessor of the successor
	client, _ = Dial(successor.Address)
	err = client.Call("Node.Notify", &NodeInfo{ID: n.ID, Address: n.Address}, nil)
	if err != nil { return fmt.Errorf("failed to notify successor: %w", err) }
	defer client.Close()

	return nil
}

// FindSuccessor finds the successor node for a given ID.
func (n *Node) FindSuccessor(id *big.Int, successor *NodeInfo) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	// If the current node is the only node in the ring
	if n.Successor.ID == n.ID {
		*successor = *n.Successor
		return nil
	}
	
	if between(n.ID, id, n.Successor.ID) {
		*successor = *n.Successor
		return nil
	}

	// Avoid recursion if Successor is nil
	if n.Successor == nil { return fmt.Errorf("successor is nil") }

	return n.Successor.FindSuccessorRPC(id, successor)
}

// FindSuccessorRPC is the RPC wrapper for FindSuccessor.
func (n *NodeInfo) FindSuccessorRPC(id *big.Int, successor *NodeInfo) error {
	client, err := rpc.Dial("tcp", n.Address)
	if err != nil { return fmt.Errorf("failed to connect to successor: %w", err) }
	defer client.Close()
	return client.Call("Node.FindSuccessor", id, successor)
}

// Stabilize periodically verifies the successor and updates pointers.
func (n *Node) Stabilize() error {
	n.mutex.Lock()
	successor := n.Successor
	n.mutex.Unlock()

	var predecessor NodeInfo
	err := successor.GetPredecessorRPC(&predecessor)
	if err != nil {
		return fmt.Errorf("error getting predecessor: %w", err)
	}

	n.mutex.Lock()
	if predecessor.ID != nil && between(n.ID, predecessor.ID, n.Successor.ID) {
		n.Successor = &predecessor
	}
	n.mutex.Unlock()

	successor = n.Successor
	if successor != nil && successor.ID.Cmp(n.ID) != 0 {
		successor.NotifyRPC(&NodeInfo{ID: n.ID, Address: n.Address}, nil)
	}
	return nil
}

// Notify is called to update predecessor pointers.
func (n *Node) Notify(candidate *NodeInfo, _ *struct{}) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.Predecessor == nil || between(n.Predecessor.ID, candidate.ID, n.ID) {
		n.Predecessor = candidate
	}
	return nil
}

// NotifyRPC is the RPC wrapper for Notify.
func (n *NodeInfo) NotifyRPC(candidate *NodeInfo, _ *struct{}) error {
	client, err := Dial(n.Address)
	if err != nil { return fmt.Errorf("failed to connect to successor: %w", err) }
	defer client.Close()
	return client.Call("Node.Notify", candidate, nil)
}

// FixFingers periodically updates the finger table.
func (n *Node) FixFingers() error {
	for i := 0; i < M; i++ {
		offset := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(i)), nil)
		nextID := new(big.Int).Add(n.ID, offset)
		nextID.Mod(nextID, big.NewInt(0).Exp(big.NewInt(2), big.NewInt(M), nil))
		var successor NodeInfo
		err := n.FindSuccessor(nextID, &successor)
		if err != nil { return fmt.Errorf("failed to find successor: %w", err) }
		n.mutex.Lock()
		n.FingerTable[i] = &successor
		n.mutex.Unlock()
	}
	return nil
}

// GetPredecessor is called to get the predecessor of the current node
func (n *Node) GetPredecessor(_ *struct{}, predecessor *NodeInfo) error {
	if n.Predecessor == nil { return nil }
	*predecessor = *n.Predecessor
	return nil
}

// GetPredecessorRPC is the RPC wrapper for GetPredecessor.
func (n *NodeInfo) GetPredecessorRPC(predecessor *NodeInfo) error {
	client, _ := Dial(n.Address)
	defer client.Close()
	err := client.Call("Node.GetPredecessor", struct{}{}, predecessor)
	if err != nil { return fmt.Errorf("failed to get predecessor: %w", err) }
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
func (n *Node) Lookup(fileName string) (*NodeInfo, error) {
	hash := sha1.Sum([]byte(fileName))
	key := new(big.Int).SetBytes(hash[:])
	var successor NodeInfo
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

	// !! TEST CODE !!
	fmt.Printf("Files: %v\n", n.Files)
	for key, value := range n.Files {
		fmt.Printf("File: %s, Content: %s\n", key, value)
	}
	// !! TEST CODE !!
}

// Main function for running the Chord node.
func main() {
	address := flag.String("a", "", "The IP address that the Chord client will bind to and advertise to other nodes")
	port := flag.Int("p", 0, "The port that the Chord client will bind to and listen on")
	joinAddress := flag.String("ja", "", "The IP address of the machine running a Chord node to join")
	joinPort := flag.Int("jp", 0, "The port of the Chord node to join")
	ts := flag.Int("ts", 3000, "The time in milliseconds between invocations of 'stabilize'")
	tff := flag.Int("tff", 1000, "The time in milliseconds between invocations of 'fix fingers'")
	tcp := flag.Int("tcp", 3000, "The time in milliseconds between invocations of 'check predecessor'")
	flag.Parse()

	if *address == "" || *port == 0 {
		fmt.Println("Error: Both -a and -p must be specified.")
		flag.Usage()
		os.Exit(1)
	}

	nodeAddress := fmt.Sprintf("%s:%d", *address, *port)
	myNode := NewNode(nodeAddress)

	// Start RPC server
	rpc.Register(myNode)
	listener, err := net.Listen("tcp", nodeAddress)
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}
	defer listener.Close()
	go rpc.Accept(listener)

	// Join an existing ring if specified
	if *joinAddress != "" && *joinPort != 0 {
		remoteAddress := fmt.Sprintf("%s:%d", *joinAddress, *joinPort)
		if err := myNode.Join(remoteAddress); err != nil {
			fmt.Printf("Join failed: %v\n", err)
			return
		}
	} else {
		fmt.Println("Creating a new Chord ring.")
		myNode.Successor = &NodeInfo{ID: myNode.ID, Address: myNode.Address}
		myNode.Predecessor = nil
	}

	// Start periodic tasks
	go func() {
		for {
			err := myNode.Stabilize()
			if err != nil {
				fmt.Printf("Error stabilizing: %v\n", err)
			}
			time.Sleep(time.Duration(*ts) * time.Millisecond)
		}
	}()

	go func() {
		for {
			err := myNode.FixFingers()
			if err != nil {
				fmt.Printf("Error fixing fingers: %v\n", err)
			}
			time.Sleep(time.Duration(*tff) * time.Millisecond)
		}
	}()

	go func() {
		for {
			myNode.CheckPredecessor()
			time.Sleep(time.Duration(*tcp) * time.Millisecond)
		}
	}()

	// !! TEST CODE !!
	// go func() {
	// 	for {
	// 		myNode.PrintState()
	// 		time.Sleep(time.Duration(1000) * time.Millisecond)
	// 	}
	// }()
	// !! TEST CODE !!

	// Command loop
	fmt.Println("Chord node is operational. Use commands: 'Lookup <file>', 'StoreFile <file>', 'PrintState'.")
	for {
		var cmd, param string
		fmt.Print("> ")
		fmt.Scanln(&cmd, &param)

		switch cmd {
		case "Lookup":
			if param == "" {
				fmt.Println("Error: File name is required.")
				continue
			}
			successor, err := myNode.Lookup(param)
			if err != nil {
				fmt.Printf("Lookup failed: %v\n", err)
			} else {
				fmt.Printf("File '%s' is managed by node: %s (%s)\n", param, successor.ID.Text(16), successor.Address)
			}
		case "StoreFile":
			if param == "" {
				fmt.Println("Error: File path is required.")
				continue
			}
			if err := myNode.StoreFile(param); err != nil {
				fmt.Printf("StoreFile failed: %v\n", err)
			} else {
				fmt.Printf("File '%s' stored successfully.\n", param)
			}
		case "PrintState":
			myNode.PrintState()
		default:
			fmt.Println("Unknown command. Available commands: 'Lookup <file>', 'StoreFile <file>', 'PrintState'.")
		}
	}
}

