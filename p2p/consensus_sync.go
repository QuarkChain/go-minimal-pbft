package p2p

import (
	"bytes"
	"context"
	"time"

	"github.com/QuarkChain/go-minimal-pbft/consensus"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p-core/peer"
)

func (s *Server) processNewPeer(ctx context.Context, peerID peer.ID) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Do not allow starting new broadcasting goroutines after reactor shutdown
	// has been initiated. This can happen after we've manually closed all
	// peer goroutines, but the router still sends in-flight peer updates.
	if !s.IsRunning {
		return
	}

	ps, ok := s.peerStateMap[peerID]
	if !ok {
		ps = NewPeerState(peerID)
		s.peerStateMap[peerID] = ps
	}

	if ps != nil {

		// if !ps.IsRunning() {
		// // Set the peer state's closer to signal to all spawned goroutines to exit
		// // when the peer is removed. We also set the running state to ensure we
		// // do not spawn multiple instances of the same goroutines and finally we
		// // set the waitgroup counter so we know when all goroutines have exited.
		// ps.broadcastWG.Add(3)
		// ps.SetRunning(true)

		// start goroutines for this peer
		// go s.gossipDataRoutine(ctx, ps)
		go s.gossipVotesRoutine(ctx, ps)
		// go s.queryMaj23Routine(ctx, ps)

		// // Send our state to the peer. If we're block-syncing, broadcast a
		// // RoundStepMessage later upon SwitchToConsensus().
		// if !r.waitSync {
		// 	go func() { _ = r.sendNewRoundStepMessage(ctx, ps.peerID) }()
		// }
		// }
	}
}

func (s *Server) processRemovePeer(ctx context.Context, peerID peer.ID) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Do not allow starting new broadcasting goroutines after reactor shutdown
	// has been initiated. This can happen after we've manually closed all
	// peer goroutines, but the router still sends in-flight peer updates.
	if !s.IsRunning {
		return
	}

	ps, ok := s.peerStateMap[peerID]
	if ok {
		// signal to all spawned goroutines for the peer to gracefully exit
		close(ps.DoneCh)

		go func() {
			// Wait for all spawned broadcast goroutines to exit before marking the
			// peer state as no longer running and removal from the peers map.
			// ps.broadcastWG.Wait()

			s.lock.Lock()
			delete(s.peerStateMap, peerID)
			s.lock.Unlock()

			// ps.SetRunning(false)
		}()
	}
}

func (s *Server) gossipVotesRoutine(ctx context.Context, ps *PeerState) {
	// defer ps.broadcastWG.Done()

	// XXX: simple hack to throttle logs upon sleep
	logThrottle := 0

	timer := time.NewTimer(0)
	defer timer.Stop()

OUTER_LOOP:
	for {
		if !s.IsRunning {
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-ps.DoneCh:
			// The peer is marked for removal via a PeerUpdate as the doneCh was
			// explicitly closed to signal we should exit.
			return

		default:
		}

		rs := s.consensusState.GetRoundState()
		prs := ps.GetRoundState()

		switch logThrottle {
		case 1: // first sleep
			logThrottle = 2
		case 2: // no more sleep
			logThrottle = 0
		}

		// if height matches, then send LastCommit, Prevotes, and Precommits
		if rs.Height == prs.Height {
			if ok, err := s.gossipVotesForHeight(ctx, rs, prs, ps); err != nil {
				return
			} else if ok {
				continue OUTER_LOOP
			}
		}

		// special catchup logic -- if peer is lagging by height 1, send LastCommit
		if prs.Height != 0 && rs.Height == prs.Height+1 {
			if ok, err := s.pickSendVote(ctx, ps, rs.LastCommit); err != nil {
				return
			} else if ok {
				log.Debug("picked rs.LastCommit to send", "peer", ps.peerID, "height", prs.Height)
				continue OUTER_LOOP
			}
		}

		// catchup logic -- if peer is lagging by more than 1, send Commit
		blockStoreBase := s.blockStore.Base()
		if blockStoreBase > 0 && prs.Height != 0 && rs.Height >= prs.Height+2 && prs.Height >= blockStoreBase {
			// Load the block commit for prs.Height, which contains precommit
			// signatures for prs.Height.
			if commit := s.blockStore.LoadBlockCommit(prs.Height); commit != nil {
				if ok, err := s.pickSendVote(ctx, ps, commit); err != nil {
					return
				} else if ok {
					log.Debug("picked Catchup commit to send", "peer", ps.peerID, "height", prs.Height)
					continue OUTER_LOOP
				}
			}
		}

		if logThrottle == 0 {
			// we sent nothing -- sleep
			logThrottle = 1
			log.Debug(
				"no votes to send; sleeping",
				"peer", ps.peerID,
				"rs.Height", rs.Height,
				"prs.Height", prs.Height,
				// "localPV", rs.Votes.Prevotes(rs.Round).BitArray(), "peerPV", prs.Prevotes,
				// "localPC", rs.Votes.Precommits(rs.Round).BitArray(), "peerPC", prs.Precommits,
			)
		} else if logThrottle == 2 {
			logThrottle = 1
		}

		timer.Reset(s.consensusState.Config.PeerGossipSleepDuration)
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		continue OUTER_LOOP
	}
}

func (s *Server) gossipVotesForHeight(
	ctx context.Context,
	rs *consensus.RoundState,
	prs *PeerRoundState,
	ps *PeerState,
) (bool, error) {
	// logger := r.logger.With("height", prs.Height).With("peer", ps.peerID)

	// if there are lastCommits to send...
	if prs.Step == consensus.RoundStepNewHeight {
		if ok, err := s.pickSendVote(ctx, ps, rs.LastCommit); err != nil {
			return false, err
		} else if ok {
			log.Debug("picked rs.LastCommit to send", "peer", ps.peerID, "height", prs.Height)
			return true, nil

		}
	}

	// if there are POL prevotes to send...
	if prs.Step <= consensus.RoundStepPropose && prs.Round != -1 && prs.Round <= rs.Round && prs.ProposalPOLRound != -1 {
		if polPrevotes := rs.Votes.Prevotes(prs.ProposalPOLRound); polPrevotes != nil {
			if ok, err := s.pickSendVote(ctx, ps, polPrevotes); err != nil {
				return false, err
			} else if ok {
				log.Debug("picked rs.Prevotes(prs.ProposalPOLRound) to send", "peer", ps.peerID, "height", prs.Height, "round", prs.ProposalPOLRound)
				return true, nil
			}
		}
	}

	// if there are prevotes to send...
	if prs.Step <= consensus.RoundStepPrevoteWait && prs.Round != -1 && prs.Round <= rs.Round {
		if ok, err := s.pickSendVote(ctx, ps, rs.Votes.Prevotes(prs.Round)); err != nil {
			return false, err
		} else if ok {
			log.Debug("picked rs.Prevotes(prs.Round) to send", "peer", ps.peerID, "height", prs.Height, "round", prs.Round)
			return true, nil
		}
	}

	// if there are precommits to send...
	if prs.Step <= consensus.RoundStepPrecommitWait && prs.Round != -1 && prs.Round <= rs.Round {
		if ok, err := s.pickSendVote(ctx, ps, rs.Votes.Precommits(prs.Round)); err != nil {
			return false, err
		} else if ok {
			log.Debug("picked rs.Precommits(prs.Round) to send", "peer", ps.peerID, "height", prs.Height, "round", prs.Round)
			return true, nil
		}
	}

	// if there are prevotes to send...(which are needed because of validBlock mechanism)
	if prs.Round != -1 && prs.Round <= rs.Round {
		if ok, err := s.pickSendVote(ctx, ps, rs.Votes.Prevotes(prs.Round)); err != nil {
			return false, err
		} else if ok {
			log.Debug("picked rs.Prevotes(prs.Round) to send", "peer", ps.peerID, "height", prs.Height, "round", prs.Round)
			return true, nil
		}
	}

	// if there are POLPrevotes to send...
	if prs.ProposalPOLRound != -1 {
		if polPrevotes := rs.Votes.Prevotes(prs.ProposalPOLRound); polPrevotes != nil {
			if ok, err := s.pickSendVote(ctx, ps, polPrevotes); err != nil {
				return false, err
			} else if ok {
				log.Debug("picked rs.Prevotes(prs.ProposalPOLRound) to send", "peer", ps.peerID, "height", prs.Height, "round", prs.ProposalPOLRound)
				return true, nil
			}
		}
	}

	return false, nil
}

// pickSendVote picks a vote and sends it to the peer. It will return true if
// there is a vote to send and false otherwise.
func (s *Server) pickSendVote(ctx context.Context, ps *PeerState, votes VoteSetReader) (bool, error) {
	vote, ok := ps.PickVoteToSend(votes)
	if !ok {
		return false, nil
	}

	log.Debug("sending vote message", "ps", ps, "vote", vote)

	var buf bytes.Buffer
	err := vote.EncodeRLP(&buf)
	if err != nil {
		return false, err
	}

	Send(s.ctx, s.Host, ps.peerID, TopicVote, buf.Bytes())

	ps.SetHasVote(vote)
	return true, nil
}
