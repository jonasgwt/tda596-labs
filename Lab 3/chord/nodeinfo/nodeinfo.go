package nodeinfo

import (
	"math/big"
)

// NodeInfo contains basic details about a node in the Chord ring.
type NodeInfo struct {
	ID      *big.Int
	Address string
}
