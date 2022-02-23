package consensus

import (
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/chamber"
	"github.com/ethereum/go-ethereum/rlp"
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

	ErrVoteConflictingVotes = chamber.ErrVoteConflictingVotes
)

var (
	NewVoteSet                       = chamber.NewVoteSet
	NewValidatorSet                  = chamber.NewValidatorSet
	VerifyCommit                     = chamber.VerifyCommit
	NewHeightVoteSet                 = chamber.NewHeightVoteSet
	ErrVoteNonDeterministicSignature = chamber.ErrVoteNonDeterministicSignature
)

type FullBlock struct {
	types.Block
	LastCommit *Commit
}

func (b *FullBlock) HashTo(hash common.Hash) bool {
	if b == nil {
		return false
	}
	return b.Hash() == hash
}

func (b *FullBlock) EncodeRLP(w io.Writer) error {
	err := b.Block.EncodeRLP(w)
	if err != nil {
		return err
	}
	return rlp.Encode(w, b.LastCommit)
}

func (b *FullBlock) DecodeRLP(s *rlp.Stream) error {
	err := b.Block.DecodeRLP(s)
	if err != nil {
		return err
	}
	b.LastCommit = &chamber.Commit{}
	return s.Decode(&b.LastCommit)
}

// func NewBlock(header *types.Header, body *types.Body, lastCommit *Commit) *Block {
// 	header.LastCommitHash = lastCommit.Hash()
// 	return &Block{Header: header, Body: body, LastCommit: lastCommit}
// }

// func (b *Block) NumberU64() uint64 {
// 	return b.Header.Number.Uint64()
// }

var MaxSignatureSize = 65

type VerifiedBlock struct {
	FullBlock
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
