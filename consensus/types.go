package consensus

import "github.com/ethereum/go-ethereum/common"

type Block struct {
	LastBlockID     common.Hash
	Height          uint64
	TimeMs          uint64
	ProposerAddress common.Address
	LastCommit      *Commit
}

func (b *Block) Hash() common.Hash {
	return common.Hash{}
}

// Commit contains the evidence that a block was committed by a set of validators.
// NOTE: Commit is empty for height 1, but never nil.
type Commit struct {
	// NOTE: The signatures are in order of address to preserve the bonded
	// ValidatorSet order.
	// Any peer with a block can gossip signatures by index with a peer without
	// recalculating the active ValidatorSet.
	Height     uint64      `json:"height"`
	Round      int32       `json:"round"`
	BlockID    common.Hash `json:"block_id"`
	Signatures []CommitSig `json:"signatures"`

	// Memoized in first call to corresponding method.
	// NOTE: can't memoize in constructor because constructor isn't used for
	// unmarshaling.
	// hash     common.Hash
	// bitArray *bits.BitArray
}
