package consensus

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

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

// Proposal defines a block proposal for the consensus.
// It refers to the block by BlockID field.
// It must be signed by the correct proposer for the given Height/Round
// to be considered valid. It may depend on votes from a previous round,
// a so-called Proof-of-Lock (POL) round, as noted in the POLRound.
// If POLRound >= 0, then BlockID corresponds to the block that is locked in POLRound.
type Proposal struct {
	Height    uint64      `json:"height"`
	Round     int32       `json:"round"`     // there can not be greater than 2_147_483_647 rounds
	POLRound  int32       `json:"pol_round"` // -1 if null.
	BlockID   common.Hash `json:"block_id"`
	Timestamp int64       `json:"timestamp"` // unix ms
	Signature []byte      `json:"signature"`
}

// NewProposal returns a new Proposal.
// If there is no POLRound, polRound should be -1.
func NewProposal(height uint64, round int32, polRound int32, blockID common.Hash) *Proposal {
	return &Proposal{
		Height:    height,
		Round:     round,
		POLRound:  polRound,
		BlockID:   blockID,
		Timestamp: CanonicalNowMs(),
	}
}

func (p *Proposal) ToUnsignedBytes() []byte {
	return nil
}

// ProposalMessage is sent when a new block is proposed.
type ProposalMessage struct {
	Proposal *Proposal
}

// ValidateBasic performs basic validation.
func (m *ProposalMessage) ValidateBasic() error {
	return m.Proposal.ValidateBasic()
}

// String returns a string representation.
func (m *ProposalMessage) String() string {
	return fmt.Sprintf("[Proposal %v]", m.Proposal)
}

// ValidateBasic performs basic validation.
func (p *Proposal) ValidateBasic() error {
	// if p.Type != tmproto.ProposalType {
	// 	return errors.New("invalid Type")
	// }
	if p.Height < 0 {
		return errors.New("negative Height")
	}
	if p.Round < 0 {
		return errors.New("negative Round")
	}
	if p.POLRound < -1 {
		return errors.New("negative POLRound (exception: -1)")
	}
	// if err := p.BlockID.ValidateBasic(); err != nil {
	// 	return fmt.Errorf("wrong BlockID: %v", err)
	// }
	// ValidateBasic above would pass even if the BlockID was empty:
	// if !p.BlockID.IsComplete() {
	// 	return fmt.Errorf("expected a complete, non-empty BlockID, got: %v", p.BlockID)
	// }

	// NOTE: Timestamp validation is subtle and handled elsewhere.

	if len(p.Signature) == 0 {
		return errors.New("signature is missing")
	}

	// if len(p.Signature) > MaxSignatureSize {
	// 	return fmt.Errorf("signature is too big (max: %d)", MaxSignatureSize)
	// }
	return nil
}
