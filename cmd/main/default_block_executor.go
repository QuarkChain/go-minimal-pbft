package main

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/go-minimal-pbft/consensus"
	"github.com/syndtr/goleveldb/leveldb"
)

type DefaultBlockExecutor struct {
	db *leveldb.DB
}

func NewDefaultBlockExecutor(db *leveldb.DB) consensus.BlockExecutor {
	return &DefaultBlockExecutor{}
}

func (be *DefaultBlockExecutor) ValidateBlock(state consensus.ChainState, b *consensus.Block) error {
	return validateBlock(state, b)
}

func validateBlock(state consensus.ChainState, block *consensus.Block) error {

	// Validate basic info.

	if state.LastBlockHeight == 0 && block.Height != state.InitialHeight {
		return fmt.Errorf("wrong Block.Header.Height. Expected %v for initial block, got %v",
			block.Height, state.InitialHeight)
	}
	if state.LastBlockHeight > 0 && block.Height != state.LastBlockHeight+1 {
		return fmt.Errorf("wrong Block.Header.Height. Expected %v, got %v",
			state.LastBlockHeight+1,
			block.Height,
		)
	}
	// Validate prev block info.
	if block.LastBlockID != state.LastBlockID {
		return fmt.Errorf("wrong Block.Header.LastBlockID.  Expected %v, got %v",
			state.LastBlockID,
			block.LastBlockID,
		)
	}

	// Validate block LastCommit.
	if block.Height == state.InitialHeight {
		if len(block.LastCommit.Signatures) != 0 {
			return errors.New("initial block can't have LastCommit signatures")
		}
	} else {
		// LastCommit.Signatures length is checked in VerifyCommit.
		if err := state.LastValidators.VerifyCommit(
			state.ChainID, state.LastBlockID, block.Height-1, block.LastCommit); err != nil {
			return err
		}
	}

	// NOTE: We can't actually verify it's the right proposer because we don't
	// know what round the block was first proposed. So just check that it's
	// a legit address and a known validator.
	// The length is checked in ValidateBasic above.
	if !state.Validators.HasAddress(block.ProposerAddress) {
		return fmt.Errorf("block.Header.ProposerAddress %X is not a validator",
			block.ProposerAddress,
		)
	}

	// Validate block Time
	switch {
	case block.Height > state.InitialHeight:
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

	case block.Height == state.InitialHeight:
		genesisTime := state.LastBlockTime
		if block.TimeMs != genesisTime {
			return fmt.Errorf("block time %v is not equal to genesis time %v",
				block.TimeMs,
				genesisTime,
			)
		}

	default:
		return fmt.Errorf("block height %v lower than initial height %v",
			block.Height, state.InitialHeight)
	}

	return nil
}

// weightedTime for computing a median.
type weightedTime struct {
	TimeMs uint64
	Weight int64
}

func MedianTime(commit *consensus.Commit, validators *consensus.ValidatorSet) uint64 {
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

func (be *DefaultBlockExecutor) ApplyBlock(ctx context.Context, state consensus.ChainState, b *consensus.Block) (consensus.ChainState, error) {
	return state, nil
}
