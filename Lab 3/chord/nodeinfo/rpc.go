package nodeinfo

import (
	"chord/helpers"
	"fmt"
	"math/big"
	"net/rpc"
)

// FindSuccessorRPC is the RPC wrapper for FindSuccessor.
func (n *NodeInfo) FindSuccessorRPC(id *big.Int, successor *NodeInfo) error {
	client, err := rpc.Dial("tcp", n.Address)
	if err != nil { return fmt.Errorf("failed to connect to successor: %w", err) }
	defer client.Close()
	return client.Call("Node.FindSuccessor", id, successor)
}

// NotifyRPC is the RPC wrapper for Notify.
func (n *NodeInfo) NotifyRPC(candidate *NodeInfo, _ *struct{}) error {
	client, err := helpers.Dial(n.Address)
	if err != nil { return fmt.Errorf("failed to connect to successor: %w", err) }
	defer client.Close()
	return client.Call("Node.Notify", candidate, nil)
}

// GetPredecessorRPC is the RPC wrapper for GetPredecessor.
func (n *NodeInfo) GetPredecessorRPC(predecessor *NodeInfo) error {
	client, err := helpers.Dial(n.Address)
	if err != nil { return fmt.Errorf("failed to connect to successor: %w", err) }
	defer client.Close()
	err = client.Call("Node.GetPredecessor", struct{}{}, predecessor)
	if err != nil { return fmt.Errorf("failed to get predecessor: %w", err) }
	return nil
}