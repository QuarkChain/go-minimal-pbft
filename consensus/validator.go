package consensus

import (
	"github.com/ethereum/go-ethereum/common"
)

// Volatile state for each Validator
// NOTE: The ProposerPriority is not included in Validator.Hash();
// make sure to update that method if changes are made here
type Validator struct {
	Address common.Address `json:"address"`
	PubKey  PubKey         `json:"pub_key"`

	ProposerPriority int64 `json:"proposer_priority"`
}

// Creates a new copy of the validator so we can mutate ProposerPriority.
// Panics if the validator is nil.
func (v *Validator) Copy() *Validator {
	vCopy := *v
	return &vCopy
}

func (v *Validator) VotingPower() int64 {
	return 1
}
