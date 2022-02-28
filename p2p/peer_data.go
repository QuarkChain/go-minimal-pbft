package p2p

import (
	"sync"
	"time"

	"github.com/QuarkChain/go-minimal-pbft/consensus"
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
	// BitArray() *BitArray
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
	ProposalPOL     *BitArray `json:"proposal_pol"`
	Prevotes        *BitArray `json:"prevotes"`          // All votes peer has for this round
	Precommits      *BitArray `json:"precommits"`        // All precommits peer has for this round
	LastCommitRound int32     `json:"last_commit_round"` // Round of commit for last height. -1 if none.
	LastCommit      *BitArray `json:"last_commit"`       // All commit precommits of commit for last height.

	// Round that we have commit for. Not necessarily unique. -1 if none.
	CatchupCommitRound int32 `json:"catchup_commit_round"`

	// All commit precommits peer has for this height & CatchupCommitRound
	CatchupCommit *BitArray `json:"catchup_commit"`
}

type PeerData struct {
	ConnState PeerConnectionState
	Address   ma.Multiaddr
	Direction network.Direction
	Enr       *enr.Record
	DoneCh    chan struct{}
	lock      sync.Mutex
	PRS       PeerRoundState
	peerId    peer.ID
}

// GetRoundState returns a shallow copy of the PeerRoundState. There's no point
// in mutating it since it won't change PeerState.
func (ps *PeerData) GetRoundState() *PeerRoundState {
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

// // PickVoteToSend picks a vote to send to the peer. It will return true if a
// // vote was picked.
// //
// // NOTE: `votes` must be the correct Size() for the Height().
// func (ps *PeerData) PickVoteToSend(votes types.VoteSetReader) (*consensus.Vote, bool) {
// 	ps.mtx.Lock()
// 	defer ps.mtx.Unlock()

// 	if votes.Size() == 0 {
// 		return nil, false
// 	}

// 	var (
// 		height    = votes.GetHeight()
// 		round     = votes.GetRound()
// 		votesType = tmproto.SignedMsgType(votes.Type())
// 		size      = votes.Size()
// 	)

// 	// lazily set data using 'votes'
// 	if votes.IsCommit() {
// 		ps.ensureCatchupCommitRound(height, round, size)
// 	}

// 	ps.ensureVoteBitArrays(height, size)

// 	psVotes := ps.getVoteBitArray(height, round, votesType)
// 	if psVotes == nil {
// 		return nil, false // not something worth sending
// 	}

// 	if index, ok := votes.BitArray().Sub(psVotes).PickRandom(); ok {
// 		vote := votes.GetByIndex(int32(index))
// 		if vote != nil {
// 			return vote, true
// 		}
// 	}

// 	return nil, false
// }
