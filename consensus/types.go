package consensus

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/chamber"
)

type (
	Vote          = chamber.Vote
	VoteMessage   = chamber.VoteMessage
	VoteForSign   = chamber.VoteForSign
	Validator     = chamber.Validator
	ValidatorSet  = chamber.ValidatorSet
	VoteSet       = chamber.VoteSet
	HeightVoteSet = chamber.HeightVoteSet

	Header = types.Header
	Block  = types.Block

	ErrVoteConflictingVotes = chamber.ErrVoteConflictingVotes
)

var (
	NewVoteSet                       = chamber.NewVoteSet
	NewValidatorSet                  = chamber.NewValidatorSet
	VerifyCommit                     = chamber.VerifyCommit
	NewHeightVoteSet                 = chamber.NewHeightVoteSet
	ErrVoteNonDeterministicSignature = chamber.ErrVoteNonDeterministicSignature
)

var MaxSignatureSize = 65

type VerifiedBlock struct {
	Block
	SeenCommit *Commit // not necessarily the LastCommit of next block, but enough to check the validity of the block
}

type Commit = chamber.Commit

var NewCommit = chamber.NewCommit
var CommitToVoteSet = chamber.CommitToVoteSet

// BlockIDFlag indicates which BlockID the signature is for.
type BlockIDFlag = chamber.BlockIDFlag
type CommitSig = chamber.CommitSig

var NewCommitSigAbsent = chamber.NewCommitSigAbsent

const (
	// BlockIDFlagAbsent - no vote was received from a validator.
	BlockIDFlagAbsent BlockIDFlag = chamber.BlockIDFlagAbsent
	// BlockIDFlagCommit - voted for the Commit.BlockID.
	BlockIDFlagCommit = chamber.BlockIDFlagCommit
	// BlockIDFlagNil - voted for nil.
	BlockIDFlagNil = chamber.BlockIDFlagAbsent
)
