package consensus

import "github.com/ethereum/go-ethereum/common"

// State is a short description of the latest committed block of the Tendermint consensus.
// It keeps all information necessary to validate new blocks,
// including the last validator set and the consensus params.
// All fields are exposed so the struct can be easily serialized,
// but none of them should be mutated directly.
// Instead, use state.Copy() or updateState(...).
// NOTE: not goroutine-safe.
type ChainState struct {
	// immutable
	ChainID       string
	InitialHeight uint64 // should be 1, not 0, when starting from height 1

	// LastBlockHeight=0 at genesis (ie. block(H=0) does not exist)
	LastBlockHeight uint64
	LastBlockID     common.Hash
	LastBlockTime   uint64

	// LastValidators is used to validate block.LastCommit.
	// Validators are persisted to the database separately every time they change,
	// so we can query for historical validator sets.
	// Note that if s.LastBlockHeight causes a valset change,
	// we set s.LastHeightValidatorsChanged = s.LastBlockHeight + 1 + 1
	// Extra +1 due to nextValSet delay.
	NextValidators              *ValidatorSet
	Validators                  *ValidatorSet
	LastValidators              *ValidatorSet
	LastHeightValidatorsChanged int64

	// Consensus parameters used for validating blocks.
	// Changes returned by EndBlock and updated after Commit.
	// ConsensusParams                  types.ConsensusParams
	// LastHeightConsensusParamsChanged int64

	// Merkle root of the results from executing prev block
	// LastResultsHash []byte

	// the latest AppHash we've received from calling abci.Commit()
	AppHash []byte
}

// IsEmpty returns true if the State is equal to the empty State.
func (state ChainState) IsEmpty() bool {
	return state.Validators == nil // XXX can't compare to Empty
}

func (state ChainState) Copy() ChainState {
	return ChainState{
		ChainID:       state.ChainID,
		InitialHeight: state.InitialHeight,

		LastBlockHeight: state.LastBlockHeight,
		LastBlockID:     state.LastBlockID,
		LastBlockTime:   state.LastBlockTime,

		NextValidators:              state.NextValidators.Copy(),
		Validators:                  state.Validators.Copy(),
		LastValidators:              state.LastValidators.Copy(),
		LastHeightValidatorsChanged: state.LastHeightValidatorsChanged,

		// ConsensusParams:                  state.ConsensusParams,
		// LastHeightConsensusParamsChanged: state.LastHeightConsensusParamsChanged,

		AppHash: state.AppHash,

		// LastResultsHash: state.LastResultsHash,
	}
}

func MakeGenesisChainState(chainID string, genesisTimeMs uint64, validatorAddrs []common.Address) *ChainState {
	validators := make([]*Validator, len(validatorAddrs))
	for i, addr := range validatorAddrs {
		pubkey := NewEcdsaPubKey(addr)
		validators[i] = &Validator{Address: addr, PubKey: pubkey}
	}
	vs := &ValidatorSet{Validators: validators}
	nextVs := vs.Copy()
	nextVs.IncrementProposerPriority(1)
	return &ChainState{
		ChainID:                     chainID,
		InitialHeight:               1,
		LastBlockHeight:             0,
		LastBlockID:                 common.Hash{},
		LastBlockTime:               genesisTimeMs,
		Validators:                  vs,
		NextValidators:              nextVs,
		LastValidators:              &ValidatorSet{}, // not exist
		LastHeightValidatorsChanged: 1,
	}
}
