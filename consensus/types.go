package consensus

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types/chamber"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

type Vote = chamber.Vote
type VoteMessage = chamber.VoteMessage
type VoteForSign = chamber.VoteForSign
type Validator = chamber.Validator
type ValidatorSet = chamber.ValidatorSet

var NewValidatorSet = chamber.NewValidatorSet

var VerifyCommit = chamber.VerifyCommit

var MaxSignatureSize = 65

type Header struct {
	LastBlockID     common.Hash
	Height          uint64
	TimeMs          uint64
	ProposerAddress common.Address
	NextValidators  []common.Address
	LastCommitHash  common.Hash
}

type Block struct {
	Header
	Data       []byte
	LastCommit *Commit
}

func (b *Block) fillHeader() {
	b.LastCommitHash = b.LastCommit.Hash()
}

func (b *Block) Hash() common.Hash {
	b.fillHeader()

	data, err := rlp.EncodeToBytes(b.Header)
	if err != nil {
		panic("fail to rlp Commit")
	}
	return crypto.Keccak256Hash(data)
}

func (b *Block) HashTo(hash common.Hash) bool {
	if b == nil {
		return false
	}
	return b.Hash() == hash
}

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
