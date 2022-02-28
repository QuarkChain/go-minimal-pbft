package p2p

import (
	"fmt"
	"sync"
	"time"

	"github.com/QuarkChain/go-minimal-pbft/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// Common interface between *consensus.VoteSet and types.Commit
type VoteSetReader interface {
	GetHeight() uint64
	GetRound() int32
	Type() byte
	Size() int
	BitArray() *consensus.BitArray
	GetByIndex(int32) *consensus.Vote
	IsCommit() bool
}

// PeerRoundState contains the known state of a peer.
// NOTE: Read-only when returned by PeerState.GetRoundState().
type PeerRoundState struct {
	Height uint64                  `json:"height"` // Height peer is at
	Round  int32                   `json:"round"`  // Round peer is at, -1 if unknown.
	Step   consensus.RoundStepType `json:"step"`   // Step peer is at

	// Estimated start of round 0 at this height
	StartTime time.Time `json:"start_time"`

	// True if peer has proposal for this round
	Proposal bool `json:"proposal"`
	// Proposal's POL round. -1 if none.
	ProposalPOLRound int32 `json:"proposal_pol_round"`

	// nil until ProposalPOLMessage received.
	ProposalPOL     *consensus.BitArray `json:"proposal_pol"`
	Prevotes        *consensus.BitArray `json:"prevotes"`          // All votes peer has for this round
	Precommits      *consensus.BitArray `json:"precommits"`        // All precommits peer has for this round
	LastCommitRound int32               `json:"last_commit_round"` // Round of commit for last height. -1 if none.
	LastCommit      *consensus.BitArray `json:"last_commit"`       // All commit precommits of commit for last height.

	// Round that we have commit for. Not necessarily unique. -1 if none.
	CatchupCommitRound int32 `json:"catchup_commit_round"`

	// All commit precommits peer has for this height & CatchupCommitRound
	CatchupCommit *consensus.BitArray `json:"catchup_commit"`
}

type PeerData struct {
	ConnState PeerConnectionState
	Address   ma.Multiaddr
	Direction network.Direction
	Enr       *enr.Record
}

type PeerState struct {
	lock   sync.Mutex
	DoneCh chan struct{}
	PRS    PeerRoundState
	peerID peer.ID
}

// NewPeerState returns a new PeerState for the given node ID.
func NewPeerState(peerID peer.ID) *PeerState {
	return &PeerState{
		peerID: peerID,
		DoneCh: make(chan struct{}),
		PRS: PeerRoundState{
			Round:              -1,
			ProposalPOLRound:   -1,
			LastCommitRound:    -1,
			CatchupCommitRound: -1,
		},
	}
}

// GetRoundState returns a shallow copy of the PeerRoundState. There's no point
// in mutating it since it won't change PeerState.
func (ps *PeerState) GetRoundState() *PeerRoundState {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	prs := ps.PRS.Copy()
	return &prs
}

// Copy provides a deep copy operation. Because many of the fields in
// the PeerRound struct are pointers, we need an explicit deep copy
// operation to avoid a non-obvious shared data situation.
func (prs PeerRoundState) Copy() PeerRoundState {
	// this works because it's not a pointer receiver so it's
	// already, effectively a copy.

	prs.ProposalPOL = prs.ProposalPOL.Copy()
	prs.Prevotes = prs.Prevotes.Copy()
	prs.Precommits = prs.Precommits.Copy()
	prs.LastCommit = prs.LastCommit.Copy()
	prs.CatchupCommit = prs.CatchupCommit.Copy()

	return prs
}

// PickVoteToSend picks a vote to send to the peer. It will return true if a
// vote was picked.
//
// NOTE: `votes` must be the correct Size() for the Height().
func (ps *PeerState) PickVoteToSend(votes VoteSetReader) (*consensus.Vote, bool) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if votes.Size() == 0 {
		return nil, false
	}

	var (
		height    = votes.GetHeight()
		round     = votes.GetRound()
		votesType = consensus.SignedMsgType(votes.Type())
		size      = votes.Size()
	)

	// lazily set data using 'votes'
	if votes.IsCommit() {
		ps.ensureCatchupCommitRound(height, round, size)
	}

	ps.ensureVoteBitArrays(height, size)

	psVotes := ps.getVoteBitArray(height, round, votesType)
	if psVotes == nil {
		return nil, false // not something worth sending
	}

	if index, ok := votes.BitArray().Sub(psVotes).PickRandom(); ok {
		vote := votes.GetByIndex(int32(index))
		if vote != nil {
			return vote, true
		}
	}

	return nil, false
}

// 'round': A round for which we have a +2/3 commit.
func (ps *PeerState) ensureCatchupCommitRound(height uint64, round int32, numValidators int) {
	if ps.PRS.Height != height {
		return
	}

	/*
		NOTE: This is wrong, 'round' could change.
		e.g. if orig round is not the same as block LastCommit round.
		if ps.CatchupCommitRound != -1 && ps.CatchupCommitRound != round {
			panic(fmt.Sprintf(
				"Conflicting CatchupCommitRound. Height: %v,
				Orig: %v,
				New: %v",
				height,
				ps.CatchupCommitRound,
				round))
		}
	*/

	if ps.PRS.CatchupCommitRound == round {
		return // Nothing to do!
	}

	ps.PRS.CatchupCommitRound = round
	if round == ps.PRS.Round {
		ps.PRS.CatchupCommit = ps.PRS.Precommits
	} else {
		ps.PRS.CatchupCommit = consensus.NewBitArray(numValidators)
	}
}

func (ps *PeerState) ensureVoteBitArrays(height uint64, numValidators int) {
	if ps.PRS.Height == height {
		if ps.PRS.Prevotes == nil {
			ps.PRS.Prevotes = consensus.NewBitArray(numValidators)
		}
		if ps.PRS.Precommits == nil {
			ps.PRS.Precommits = consensus.NewBitArray(numValidators)
		}
		if ps.PRS.CatchupCommit == nil {
			ps.PRS.CatchupCommit = consensus.NewBitArray(numValidators)
		}
		if ps.PRS.ProposalPOL == nil {
			ps.PRS.ProposalPOL = consensus.NewBitArray(numValidators)
		}
	} else if ps.PRS.Height == height+1 {
		if ps.PRS.LastCommit == nil {
			ps.PRS.LastCommit = consensus.NewBitArray(numValidators)
		}
	}
}

func (ps *PeerState) getVoteBitArray(height uint64, round int32, votesType consensus.SignedMsgType) *consensus.BitArray {
	if !types.IsVoteTypeValid(votesType) {
		return nil
	}

	if ps.PRS.Height == height {
		if ps.PRS.Round == round {
			switch votesType {
			case consensus.PrevoteType:
				return ps.PRS.Prevotes

			case consensus.PrecommitType:
				return ps.PRS.Precommits
			}
		}

		if ps.PRS.CatchupCommitRound == round {
			switch votesType {
			case consensus.PrevoteType:
				return nil

			case consensus.PrecommitType:
				return ps.PRS.CatchupCommit
			}
		}

		if ps.PRS.ProposalPOLRound == round {
			switch votesType {
			case consensus.PrevoteType:
				return ps.PRS.ProposalPOL

			case consensus.PrecommitType:
				return nil
			}
		}

		return nil
	}
	if ps.PRS.Height == height+1 {
		if ps.PRS.LastCommitRound == round {
			switch votesType {
			case consensus.PrevoteType:
				return nil

			case consensus.PrecommitType:
				return ps.PRS.LastCommit
			}
		}

		return nil
	}

	return nil
}

// SetHasVote sets the given vote as known by the peer
func (ps *PeerState) SetHasVote(vote *types.Vote) {
	if vote == nil {
		return
	}
	ps.lock.Lock()
	defer ps.lock.Unlock()

	ps.setHasVote(vote.Height, vote.Round, vote.Type, vote.ValidatorIndex)
}

func (ps *PeerState) setHasVote(height uint64, round int32, voteType consensus.SignedMsgType, index int32) {
	log.Debug("setHasVote", "type", voteType, "index", index, "peerH/R", fmt.Sprintf("%d/%d", ps.PRS.Height, ps.PRS.Round),
		"H/R", fmt.Sprintf("%d/%d", height, round))

	// NOTE: some may be nil BitArrays -> no side effects
	psVotes := ps.getVoteBitArray(height, round, voteType)
	if psVotes != nil {
		psVotes.SetIndex(int(index), true)
	}
}
