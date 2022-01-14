package consensus

import (
	"github.com/ethereum/go-ethereum/common"
)

var proposerReptition = int64(8)

type ValidatorSet struct {
	// NOTE: persisted via reflect, must be exported.
	Validators []*Validator `json:"validators"`
	Proposer   *Validator   `json:"proposer"`
}

// HasAddress returns true if address given is in the validator set, false -
// otherwise.
func (vals *ValidatorSet) HasAddress(address common.Address) bool {
	for _, val := range vals.Validators {
		if val.Address == address {
			return true
		}
	}
	return false
}

// GetByAddress returns an index of the validator with address and validator
// itself (copy) if found. Otherwise, -1 and nil are returned.
func (vals *ValidatorSet) GetByAddress(address common.Address) (index int32, val *Validator) {
	for idx, val := range vals.Validators {
		if val.Address == address {
			return int32(idx), val.Copy()
		}
	}
	return -1, nil
}

// GetByIndex returns the validator's address and validator itself (copy) by
// index.
// It returns nil values if index is less than 0 or greater or equal to
// len(ValidatorSet.Validators).
func (vals *ValidatorSet) GetByIndex(index int32) (address common.Address, val *Validator) {
	if index < 0 || int(index) >= len(vals.Validators) {
		return common.Address{}, nil
	}
	val = vals.Validators[index]
	return val.Address, val.Copy()
}

// Size returns the length of the validator set.
func (vals *ValidatorSet) Size() int {
	return len(vals.Validators)
}

func (vals *ValidatorSet) TotalVotingPower() int64 {
	return int64(len(vals.Validators))
}

// Makes a copy of the validator list.
func validatorListCopy(valsList []*Validator) []*Validator {
	if valsList == nil {
		return nil
	}
	valsCopy := make([]*Validator, len(valsList))
	for i, val := range valsList {
		valsCopy[i] = val.Copy()
	}
	return valsCopy
}

// IsNilOrEmpty returns true if validator set is nil or empty.
func (vals *ValidatorSet) IsNilOrEmpty() bool {
	return vals == nil || len(vals.Validators) == 0
}

func (vals *ValidatorSet) incrementProposerPriority() *Validator {
	// simple RR with 8 repetitions
	for i, val := range vals.Validators {
		if val.ProposerPriority != 0 {
			val.ProposerPriority += 1
			if val.ProposerPriority > proposerReptition {
				val.ProposerPriority = 0
				i = i + 1
				if i >= len(vals.Validators) {
					i = 0
				}
				vals.Validators[i].ProposerPriority = 1
				return vals.Validators[i]
			}
		}
	}

	// all zeros, start with the first one with priority 2, i.e.,
	// the previous block is proposed by validator 1
	vals.Validators[0].ProposerPriority = 2
	return vals.Validators[0]
}

func (vals *ValidatorSet) findProposer() *Validator {
	for _, val := range vals.Validators {
		if val.ProposerPriority != 0 {
			return val
		}
	}
	return vals.Validators[0]
}

func (vals *ValidatorSet) GetProposer() *Validator {
	if len(vals.Validators) == 0 {
		return nil
	}
	if vals.Proposer == nil {
		vals.Proposer = vals.findProposer()
	}
	return vals.Proposer.Copy()
}

// IncrementProposerPriority increments ProposerPriority of each validator and
// updates the proposer. Panics if validator set is empty.
// `times` must be positive.
func (vals *ValidatorSet) IncrementProposerPriority(times int32) {
	if vals.IsNilOrEmpty() {
		panic("empty validator set")
	}
	if times <= 0 {
		panic("Cannot call IncrementProposerPriority with non-positive times")
	}

	var proposer *Validator
	// Call IncrementProposerPriority(1) times times.
	for i := int32(0); i < times; i++ {
		proposer = vals.incrementProposerPriority()
	}

	vals.Proposer = proposer
}

// Copy each validator into a new ValidatorSet.
func (vals *ValidatorSet) Copy() *ValidatorSet {
	return &ValidatorSet{
		Validators: validatorListCopy(vals.Validators),
		Proposer:   vals.Proposer,
	}
}
