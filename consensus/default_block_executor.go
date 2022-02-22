package consensus

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/syndtr/goleveldb/leveldb"
)

type DefaultBlockExecutor struct {
	db *leveldb.DB
}

func NewDefaultBlockExecutor(db *leveldb.DB) BlockExecutor {
	return &DefaultBlockExecutor{}
}

func (be *DefaultBlockExecutor) ValidateBlock(state ChainState, b *Block) error {
	return validateBlock(state, b)
}

func validateBlock(state ChainState, block *Block) error {

	// Validate basic info.

	if state.LastBlockHeight == 0 && block.Number != state.InitialHeight {
		return fmt.Errorf("wrong Block.Header.Height. Expected %v for initial block, got %v",
			block.Number, state.InitialHeight)
	}
	if state.LastBlockHeight > 0 && block.Number != state.LastBlockHeight+1 {
		return fmt.Errorf("wrong Block.Header.Height. Expected %v, got %v",
			state.LastBlockHeight+1,
			block.Number,
		)
	}
	// Validate prev block info.
	if block.ParentHash != state.LastBlockID {
		return fmt.Errorf("wrong Block.Header.LastBlockID.  Expected %v, got %v",
			state.LastBlockID,
			block.ParentHash,
		)
	}

	// Validate block LastCommit.
	if block.Number == state.InitialHeight {
		if len(block.LastCommit.Signatures) != 0 {
			return errors.New("initial block can't have LastCommit signatures")
		}
	} else {
		// LastCommit.Signatures length is checked in VerifyCommit.
		if err := state.LastValidators.VerifyCommit(
			state.ChainID, state.LastBlockID, block.Number-1, block.LastCommit); err != nil {
			return err
		}
	}

	// Don't allow validator change within the epoch
	if block.Number%state.Epoch != 0 && len(block.NextValidators) != 0 {
		return errors.New("cannot change validators within epoch")
	}

	// NOTE: We can't actually verify it's the right proposer because we don't
	// know what round the block was first proposed. So just check that it's
	// a legit address and a known validator.
	// The length is checked in ValidateBasic above.
	if !state.Validators.HasAddress(block.Coinbase) {
		return fmt.Errorf("block.Header.ProposerAddress %X is not a validator",
			block.Coinbase,
		)
	}

	// Validate block Time
	switch {
	case block.Number > state.InitialHeight:
		if block.TimeMs <= state.LastBlockTime {
			return fmt.Errorf("block time %v not greater than last block time %v",
				block.TimeMs,
				state.LastBlockTime,
			)
		}
		medianTime := MedianTime(block.LastCommit, state.LastValidators)
		if block.TimeMs != medianTime {
			return fmt.Errorf("invalid block time. Expected %v, got %v",
				medianTime,
				block.TimeMs,
			)
		}

	case block.Number == state.InitialHeight:
		genesisTime := state.LastBlockTime
		if block.TimeMs != genesisTime {
			return fmt.Errorf("block time %v is not equal to genesis time %v",
				block.TimeMs,
				genesisTime,
			)
		}

	default:
		return fmt.Errorf("block height %v lower than initial height %v",
			block.Number, state.InitialHeight)
	}

	return nil
}

// weightedTime for computing a median.
type weightedTime struct {
	TimeMs uint64
	Weight int64
}

// MedianTime computes a median time for a given Commit (based on Timestamp field of votes messages) and the
// corresponding validator set. The computed time is always between timestamps of
// the votes sent by honest processes, i.e., a faulty processes can not arbitrarily increase or decrease the
// computed value.
func MedianTime(commit *Commit, validators *ValidatorSet) uint64 {
	weightedTimes := make([]*weightedTime, len(commit.Signatures))

	for i, commitSig := range commit.Signatures {
		if commitSig.Absent() {
			continue
		}
		_, validator := validators.GetByAddress(commitSig.ValidatorAddress)
		// If there's no condition, TestValidateBlockCommit panics; not needed normally.
		if validator != nil {
			// totalVotingPower += validator.VotingPower
			weightedTimes[i] = &weightedTime{TimeMs: commitSig.TimestampMs, Weight: 1}
		}
	}

	return weightedMedian(weightedTimes, int64(len(commit.Signatures)))
}

// weightedMedian computes weighted median time for a given array of WeightedTime and the total voting power.
func weightedMedian(weightedTimes []*weightedTime, totalVotingPower int64) (res uint64) {
	median := totalVotingPower / 2

	sort.Slice(weightedTimes, func(i, j int) bool {
		if weightedTimes[i] == nil {
			return false
		}
		if weightedTimes[j] == nil {
			return true
		}
		return weightedTimes[i].TimeMs < weightedTimes[j].TimeMs
	})

	for _, weightedTime := range weightedTimes {
		if weightedTime != nil {
			if median <= weightedTime.Weight {
				res = weightedTime.TimeMs
				break
			}
			median -= weightedTime.Weight
		}
	}
	return
}

func (be *DefaultBlockExecutor) ApplyBlock(ctx context.Context, state ChainState, block *Block) (ChainState, error) {
	// TOOD: execute the block & new validator change
	// Update the state with the block and responses.
	state, err := updateState(state, block.Hash(), &block.Header, []common.Address{})
	if err != nil {
		return state, fmt.Errorf("commit failed for application: %v", err)
	}

	return state, nil
}

func updateState(
	state ChainState,
	blockID common.Hash,
	header *Header,
	nextValidators []common.Address,
) (ChainState, error) {

	// Copy the valset so we can apply changes from EndBlock
	// and update s.LastValidators and s.Validators.
	nValSet := state.NextValidators.Copy()

	if len(nextValidators) != 0 {
		nValSet = NewValidatorSet(nextValidators)
	}

	// Update validator proposer priority and set state variables.
	nValSet.IncrementProposerPriority(1)

	// // Update the validator set with the latest abciResponses.
	// lastHeightValsChanged := state.LastHeightValidatorsChanged
	// if len(validatorUpdates) > 0 {
	// 	err := nValSet.UpdateWithChangeSet(validatorUpdates)
	// 	if err != nil {
	// 		return state, fmt.Errorf("error changing validator set: %v", err)
	// 	}
	// 	// Change results from this height but only applies to the next next height.
	// 	lastHeightValsChanged = header.Height + 1 + 1
	// }

	// Update the params with the latest abciResponses.
	// nextParams := state.ConsensusParams
	// lastHeightParamsChanged := state.LastHeightConsensusParamsChanged
	// if abciResponses.EndBlock.ConsensusParamUpdates != nil {
	// 	// NOTE: must not mutate s.ConsensusParams
	// 	nextParams = state.ConsensusParams.UpdateConsensusParams(abciResponses.EndBlock.ConsensusParamUpdates)
	// 	err := nextParams.ValidateConsensusParams()
	// 	if err != nil {
	// 		return state, fmt.Errorf("error updating consensus params: %v", err)
	// 	}

	// 	state.Version.App = nextParams.Version.AppVersion

	// 	// Change results from this height but only applies to the next height.
	// 	lastHeightParamsChanged = header.Height + 1
	// }

	// NOTE: the AppHash has not been populated.
	// It will be filled on state.Save.
	return ChainState{
		ChainID:         state.ChainID,
		InitialHeight:   state.InitialHeight,
		LastBlockHeight: header.Number,
		LastBlockID:     blockID,
		LastBlockTime:   header.TimeMs,
		NextValidators:  nValSet,
		Validators:      state.NextValidators.Copy(),
		LastValidators:  state.Validators.Copy(),
		AppHash:         nil,
		Epoch:           state.Epoch,
	}, nil
}
