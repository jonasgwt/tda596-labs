package helpers

import (
	"fmt"
	"math/big"
	"net/rpc"
)

// between checks if a given ID lies between two other IDs in the circular ID space.
func Between(start, id, end *big.Int) bool {
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