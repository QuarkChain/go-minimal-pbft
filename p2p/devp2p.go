package p2p

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/QuarkChain/go-minimal-pbft/consensus"
	"github.com/ethereum/go-ethereum/log"
	ethp2p "github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/libp2p/go-libp2p-core/crypto"
	"go.uber.org/zap"
)

type PeerHandler struct {
	p  *ethp2p.Peer
	rw ethp2p.MsgReadWriter
}

func broadcast(peers map[string]*PeerHandler, code uint64, data []byte) error {
	var err1 error
	for _, v := range peers {
		err := v.rw.WriteMsg(ethp2p.Msg{Code: code, Size: uint32(len(data)), Payload: bytes.NewReader(data)})
		if err != nil {
			err1 = err
		}
	}
	return err1
}

// SplitAndTrim splits input separated by a comma
// and trims excessive white space from the substrings.
func SplitAndTrim(input string) (ret []string) {
	l := strings.Split(input, ",")
	for _, r := range l {
		if r = strings.TrimSpace(r); r != "" {
			ret = append(ret, r)
		}
	}
	return ret
}

func RunDevp2p(
	state *consensus.ConsensusState,
	obsvC chan consensus.MsgInfo,
	sendC chan consensus.Message,
	priv crypto.PrivKey,
	port uint,
	networkID string,
	bootstrapPeers string,
	nodeName string,
	rootCtxCancel context.CancelFunc) func(ctx context.Context) error {

	return func(ctx context.Context) (re error) {

		cfg := ethp2p.Config{
			ListenAddr:  fmt.Sprintf(":%d", port),
			MaxPeers:    50,
			NAT:         nat.Any(),
			DiscoveryV5: true,
		}

		urls := SplitAndTrim(bootstrapPeers)
		cfg.BootstrapNodes = make([]*enode.Node, 0, len(urls))

		for _, url := range urls {
			if url != "" {
				node, err := enode.Parse(enode.ValidSchemes, url)
				if err != nil {
					log.Crit("Bootstrap URL invalid", "enode", url, "err", err)
					continue
				}
				cfg.BootstrapNodes = append(cfg.BootstrapNodes, node)
			}
		}

		server := &ethp2p.Server{Config: cfg}

		var pmu sync.Mutex
		peers := make(map[string]*PeerHandler)

		protocols := make([]ethp2p.Protocol, 1)
		protocols[0] = ethp2p.Protocol{
			Name:    "go-minimal-pbft-p2p",
			Version: 1,
			Length:  4,

			Run: func(p *ethp2p.Peer, rw ethp2p.MsgReadWriter) error {
				pmu.Lock()
				// TODO: Check max peers reached?
				if _, ok := peers[p.ID().String()]; ok {
					pmu.Unlock()
					return fmt.Errorf("already connected")
				}
				pmu.Unlock()

				var exitErr error
			LOOP:
				for {
					pkt, err := rw.ReadMsg()
					if err != nil {
						exitErr = err
						break
					}

					data := make([]byte, pkt.Size)
					n, err := pkt.Payload.Read(data)
					pkt.Discard()
					if err != nil {
						exitErr = err
						break
					}

					if uint32(n) != pkt.Size {
						exitErr = fmt.Errorf("incorrect size")
						break
					}

					msg, err := decode(data)

					if err != nil {
						log.Info("received invalid message",
							"err", err,
							"data", data,
							"from", p.Fullname())
						continue
					}

					log.Debug("received message",
						"payload", msg,
						"raw", data,
						"from", p.Fullname())

					switch m := msg.(type) {
					case *consensus.Proposal:
						obsvC <- consensus.MsgInfo{Msg: &consensus.ProposalMessage{Proposal: m}, PeerID: p.Fullname()}
						p2pMessagesReceived.WithLabelValues("observation").Inc()
					case *consensus.Vote:
						obsvC <- consensus.MsgInfo{Msg: &consensus.VoteMessage{Vote: m}, PeerID: p.Fullname()}
						p2pMessagesReceived.WithLabelValues("observation").Inc()
					case *HelloRequest:
						resp := &HelloResponse{state.GetLastHeight()}
						err := ethp2p.Send(rw, MsgHelloResponse, resp)

						if err != nil {
							exitErr = err
							break LOOP
						}
					default:
						p2pMessagesReceived.WithLabelValues("unknown").Inc()
						log.Warn("received unknown message type (running outdated software?)",
							"payload", msg,
							"raw", data,
							"from", p.Fullname())
					}

				}

				pmu.Lock()
				delete(peers, p.ID().String())
				pmu.Unlock()
				return exitErr
			},
			NodeInfo:       nil,
			PeerInfo:       nil,
			Attributes:     nil,
			DialCandidates: nil,
		}

		server.Protocols = append(server.Protocols, protocols...)
		if err := server.Start(); err != nil {
			return err
		}

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-sendC:
					var err error
					var data []byte

					// make a copy to avoid race
					pmu.Lock()
					peersMap := make(map[string]*PeerHandler)
					for k, v := range peers {
						peersMap[k] = v
					}
					pmu.Unlock()

					switch m := (msg).(type) {
					case *consensus.ProposalMessage:
						data, err = encodeRaw(m.Proposal)
						if err == nil {
							err = broadcast(peersMap, MsgProposal, data)
							p2pMessagesSent.Inc()
						}
					case *consensus.VoteMessage:
						data, err = encode(m.Vote)
						if err == nil {
							err = broadcast(peersMap, MsgVote, data)
							p2pMessagesSent.Inc()
						}
					default:
						log.Error("unrecognized data to sent")
					}

					if err != nil {
						log.Error("failed to publish message from queue", zap.Error(err))
					}

				}
			}
		}()

		return nil
	}
}
