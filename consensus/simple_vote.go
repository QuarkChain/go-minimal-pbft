package consensus

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/tendermint/crypto"
)

type PubKey interface {
	Address() common.Address
	Bytes() []byte
	VerifySignature(msg []byte, sig []byte) bool
	Equals(PubKey) bool
	Type() string
}

// Vote represents a prevote, precommit, or commit vote from validators for
// consensus.
type Vote struct {
	Type             SignedMsgType  `json:"type"`
	Height           uint64         `json:"height"`
	Round            int32          `json:"round"`    // assume there will not be greater than 2_147_483_647 rounds
	BlockID          common.Hash    `json:"block_id"` // zero if vote is nil.
	TimestampMs      uint64         `json:"timestamp"`
	ValidatorAddress common.Address `json:"validator_address"`
	ValidatorIndex   int32          `json:"validator_index"`
	Signature        []byte         `json:"signature"`
}

// VoteMessage is sent when voting for a proposal (or lack thereof).
type VoteMessage struct {
	Vote *Vote
}

// ValidateBasic performs basic validation.
func (m *VoteMessage) ValidateBasic() error {
	return m.Vote.ValidateBasic()
}

// String returns a string representation.
func (m *VoteMessage) String() string {
	return fmt.Sprintf("[Vote %v]", m.Vote)
}

func VoteSignBytes(chainID string, vote *Vote) []byte {
	// TODO: serialize to bytes
	return nil
}

func (vote *Vote) Verify(chainID string, pubKey PubKey) error {
	if pubKey.Address() != vote.ValidatorAddress {
		return ErrVoteInvalidValidatorAddress
	}
	if !pubKey.VerifySignature(VoteSignBytes(chainID, vote), vote.Signature) {
		return ErrVoteInvalidSignature
	}
	return nil
}

// CommitSig converts the Vote to a CommitSig.
func (vote *Vote) CommitSig() CommitSig {
	if vote == nil {
		return NewCommitSigAbsent()
	}

	var blockIDFlag BlockIDFlag
	switch {
	case vote.BlockID == common.Hash{}:
		blockIDFlag = BlockIDFlagNil
	default:
		blockIDFlag = BlockIDFlagCommit
		panic(fmt.Sprintf("Invalid vote %v - expected BlockID to be either empty or complete", vote))
	}

	return CommitSig{
		BlockIDFlag: blockIDFlag,
		// TODO: use eth address
		// ValidatorAddress: vote.ValidatorAddress,
		TimestampMs: vote.TimestampMs,
		Signature:   vote.Signature,
	}
}

// NewCommitSigAbsent returns new CommitSig with BlockIDFlagAbsent. Other
// fields are all empty.
func NewCommitSigAbsent() CommitSig {
	return CommitSig{
		BlockIDFlag: BlockIDFlagAbsent,
	}
}

// ForBlock returns true if CommitSig is for the block.
func (cs CommitSig) ForBlock() bool {
	return cs.BlockIDFlag == BlockIDFlagCommit
}

// ValidateBasic performs basic validation.
func (vote *Vote) ValidateBasic() error {
	if !IsVoteTypeValid(vote.Type) {
		return errors.New("invalid Type")
	}

	if vote.Round < 0 {
		return errors.New("negative Round")
	}

	// NOTE: Timestamp validation is subtle and handled elsewhere.

	// if err := vote.BlockID.ValidateBasic(); err != nil {
	// 	return fmt.Errorf("wrong BlockID: %v", err)
	// }

	// BlockID.ValidateBasic would not err if we for instance have an empty hash but a
	// non-empty PartsSetHeader:
	// if !vote.BlockID.IsZero() && !vote.BlockID.IsComplete() {
	// 	return fmt.Errorf("blockID must be either empty or complete, got: %v", vote.BlockID)
	// }

	if len(vote.ValidatorAddress) != crypto.AddressSize {
		return fmt.Errorf("expected ValidatorAddress size to be %d bytes, got %d bytes",
			crypto.AddressSize,
			len(vote.ValidatorAddress),
		)
	}
	if vote.ValidatorIndex < 0 {
		return errors.New("negative ValidatorIndex")
	}
	if len(vote.Signature) == 0 {
		return errors.New("signature is missing")
	}

	// if len(vote.Signature) > MaxSignatureSize {
	// 	return fmt.Errorf("signature is too big (max: %d)", MaxSignatureSize)
	// }

	return nil
}
