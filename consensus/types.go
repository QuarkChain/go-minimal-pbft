package consensus

import (
	"time"

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
	FullBlock     = chamber.FullBlock
	Commit        = chamber.Commit
	// BlockIDFlag indicates which BlockID the signature is for.
	BlockIDFlag     = chamber.BlockIDFlag
	CommitSig       = chamber.CommitSig
	Proposal        = chamber.Proposal
	ProposalForSign = chamber.ProposalForSign
	ProposalMessage = chamber.ProposalMessage

	Header = types.Header

	ErrVoteConflictingVotes = chamber.ErrVoteConflictingVotes
)

var (
	NewVoteSet                       = chamber.NewVoteSet
	NewValidatorSet                  = chamber.NewValidatorSet
	VerifyCommit                     = chamber.VerifyCommit
	NewHeightVoteSet                 = chamber.NewHeightVoteSet
	ErrVoteNonDeterministicSignature = chamber.ErrVoteNonDeterministicSignature
	// BlockIDFlagAbsent - no vote was received from a validator.
	BlockIDFlagAbsent BlockIDFlag = chamber.BlockIDFlagAbsent
	// BlockIDFlagCommit - voted for the Commit.BlockID.
	BlockIDFlagCommit = chamber.BlockIDFlagCommit
	// BlockIDFlagNil - voted for nil.
	BlockIDFlagNil = chamber.BlockIDFlagAbsent

	NewCommit          = chamber.NewCommit
	CommitToVoteSet    = chamber.CommitToVoteSet
	NewProposal        = chamber.NewProposal
	NewCommitSigAbsent = chamber.NewCommitSigAbsent
)

// func NewBlock(header *types.Header, body *types.Body, lastCommit *Commit) *Block {
// 	header.LastCommitHash = lastCommit.Hash()
// 	return &Block{Header: header, Body: body, LastCommit: lastCommit}
// }

var MaxSignatureSize = 65

type VerifiedBlock struct {
	FullBlock
	SeenCommit *Commit // not necessarily the LastCommit of next block, but enough to check the validity of the block
}

// Now returns the current time in UTC with no monotonic component.
func CanonicalNow() time.Time {
	return Canonical(time.Now())
}

func CanonicalNowMs() int64 {
	return time.Now().UnixMilli()
}

// Canonical returns UTC time with no monotonic component.
// Stripping the monotonic component is for time equality.
// See https://github.com/tendermint/tendermint/pull/2203#discussion_r215064334
func Canonical(t time.Time) time.Time {
	return t.Round(0).UTC()
}
