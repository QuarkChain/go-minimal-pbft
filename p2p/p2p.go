package p2p

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-minimal-pbft/consensus"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	libp2ptls "github.com/libp2p/go-libp2p-tls"
	"go.uber.org/zap"
)

var (
	p2pHeartbeatsSent = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "p2p_heartbeats_sent_total",
			Help: "Total number of p2p heartbeats sent",
		})
	p2pMessagesSent = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "p2p_broadcast_messages_sent_total",
			Help: "Total number of p2p pubsub broadcast messages sent",
		})
	p2pMessagesReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "p2p_broadcast_messages_received_total",
			Help: "Total number of p2p pubsub broadcast messages received",
		}, []string{"type"})
	decoder = make(map[byte]func([]byte) (interface{}, error))
)

type ProposalRaw struct {
	Height    uint64
	Round     uint32
	POLRound  uint32
	BlockID   common.Hash
	Timestamp uint64
	Signature []byte
	Block     *consensus.Block
}

func init() {
	prometheus.MustRegister(p2pHeartbeatsSent)
	prometheus.MustRegister(p2pMessagesSent)
	prometheus.MustRegister(p2pMessagesReceived)
	decoder[1] = decodeProposal
	decoder[2] = decodeVote
}

// Vote represents a prevote, precommit, or commit vote from validators for
// consensus.
type VoteRaw struct {
	Type             uint32
	Height           uint64
	Round            uint32
	BlockID          common.Hash
	Timestamp        uint64
	ValidatorAddress common.Address
	ValidatorIndex   uint32
	Signature        []byte
}

func (v *VoteRaw) toVote() (*consensus.Vote, error) {
	cv := &consensus.Vote{
		Type:             consensus.SignedMsgType(v.Type),
		Height:           v.Height,
		Round:            int32(v.Round),
		BlockID:          v.BlockID,
		TimestampMs:      v.Timestamp,
		ValidatorAddress: v.ValidatorAddress,
		ValidatorIndex:   int32(v.ValidatorIndex),
		Signature:        v.Signature,
	}
	return cv, cv.ValidateBasic()
}

func encodeVote(v *consensus.Vote) ([]byte, error) {
	vr := &VoteRaw{
		Type:             uint32(v.Type),
		Height:           uint64(v.Height),
		Round:            uint32(v.Round),
		BlockID:          v.BlockID,
		Timestamp:        uint64(v.TimestampMs),
		ValidatorAddress: v.ValidatorAddress,
		ValidatorIndex:   uint32(v.ValidatorIndex),
		Signature:        v.Signature,
	}
	return rlp.EncodeToBytes(vr)
}

func (p *ProposalRaw) toProposal() (*consensus.Proposal, error) {
	cp := &consensus.Proposal{
		Height:      p.Height,
		Round:       int32(p.Round),
		POLRound:    int32(p.POLRound),
		BlockID:     p.BlockID,
		TimestampMs: int64(p.Timestamp),
		Signature:   p.Signature,
		Block:       p.Block,
	}
	return cp, cp.ValidateBasic()
}

func decodeProposal(data []byte) (interface{}, error) {
	var p ProposalRaw
	err := rlp.DecodeBytes(data, &p)
	if err != nil {
		return nil, err
	}
	return p.toProposal()
}

func encodeProposal(p *consensus.Proposal) ([]byte, error) {
	pr := &ProposalRaw{
		Height:    uint64(p.Height),
		Round:     uint32(p.Round),
		POLRound:  uint32(p.POLRound),
		BlockID:   p.BlockID,
		Timestamp: uint64(p.TimestampMs),
		Signature: p.Signature,
		Block:     p.Block,
	}
	return rlp.EncodeToBytes(pr)
}

func decodeVote(data []byte) (interface{}, error) {
	var v VoteRaw
	err := rlp.DecodeBytes(data, &v)
	if err != nil {
		return nil, err
	}
	return v.toVote()
}

func decode(data []byte) (interface{}, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("incorrect msg")
	}

	if decoder[data[0]] == nil {
		return nil, fmt.Errorf("deocder not found")
	}

	return decoder[data[0]](data[1:])
}

func encode(msg interface{}) ([]byte, error) {
	var data []byte
	var err error
	var typeData []byte
	switch m := msg.(type) {
	case *consensus.Proposal:
		typeData = []byte{1}
		data, err = encodeProposal(m)
	case *consensus.Vote:
		typeData = []byte{2}
		data, err = encodeVote(m)
	}

	if err != nil {
		return nil, err
	}
	return append(typeData, data...), nil
}

func Run(obsvC chan consensus.MsgInfo,
	sendC chan consensus.Message,
	priv crypto.PrivKey,
	port uint,
	networkID string,
	bootstrapPeers string,
	nodeName string,
	rootCtxCancel context.CancelFunc) func(ctx context.Context) error {

	return func(ctx context.Context) (re error) {
		fmt.Println("running p2p")
		h, err := libp2p.New(ctx,
			// Use the keypair we generated
			libp2p.Identity(priv),

			// Multiple listen addresses
			libp2p.ListenAddrStrings(
				// Listen on QUIC only.
				// https://github.com/libp2p/go-libp2p/issues/688
				fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", port),
				fmt.Sprintf("/ip6/::/udp/%d/quic", port),
			),

			// Enable TLS security as the only security protocol.
			libp2p.Security(libp2ptls.ID, libp2ptls.New),

			// Enable QUIC transport as the only transport.
			libp2p.Transport(libp2pquic.NewTransport),

			// Let's prevent our peer from having too many
			// connections by attaching a connection manager.
			libp2p.ConnectionManager(connmgr.NewConnManager(
				100,         // Lowwater
				400,         // HighWater,
				time.Minute, // GracePeriod
			)),

			// Let this host use the DHT to find other hosts
			libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
				// TODO(leo): Persistent data store (i.e. address book)
				idht, err := dht.New(ctx, h, dht.Mode(dht.ModeServer),
					// TODO(leo): This intentionally makes us incompatible with the global IPFS DHT
					dht.ProtocolPrefix(protocol.ID("/"+networkID)),
				)
				return idht, err
			}),
		)

		if err != nil {
			panic(err)
		}

		defer func() {
			// TODO: libp2p cannot be cleanly restarted (https://github.com/libp2p/go-libp2p/issues/992)
			log.Error("p2p routine has exited, cancelling root context...", "err", re)
			rootCtxCancel()
		}()

		log.Info("Connecting to bootstrap peers", "bootstrap_peers", bootstrapPeers)

		topic := fmt.Sprintf("%s/%s", networkID, "broadcast")

		log.Info("Subscribing pubsub topic", "topic", topic)
		ps, err := pubsub.NewGossipSub(ctx, h)
		if err != nil {
			panic(err)
		}

		th, err := ps.Join(topic)
		if err != nil {
			return fmt.Errorf("failed to join topic: %w", err)
		}

		sub, err := th.Subscribe()
		if err != nil {
			return fmt.Errorf("failed to subscribe topic: %w", err)
		}

		// Add our own bootstrap nodes

		// Count number of successful connection attempts. If we fail to connect to any bootstrap peer, kill
		// the service and have supervisor retry it.
		successes := 0
		// Are we a bootstrap node? If so, it's okay to not have any peers.
		bootstrapNode := false

		for _, addr := range strings.Split(bootstrapPeers, ",") {
			if addr == "" {
				continue
			}
			ma, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				log.Error("Invalid bootstrap address", "peer", addr, "err", err)
				continue
			}
			pi, err := peer.AddrInfoFromP2pAddr(ma)
			if err != nil {
				log.Error("Invalid bootstrap address", "peer", addr, "err", err)
				continue
			}

			if pi.ID == h.ID() {
				log.Info("We're a bootstrap node")
				bootstrapNode = true
				continue
			}

			if err = h.Connect(ctx, *pi); err != nil {
				log.Error("Failed to connect to bootstrap peer", "peer", addr, "err", err)
			} else {
				successes += 1
			}
		}

		// TODO: continually reconnect to bootstrap nodes?
		if successes == 0 && !bootstrapNode {
			return fmt.Errorf("failed to connect to any bootstrap peer")
		} else {
			log.Info("Connected to bootstrap peers", "num", successes)
		}

		log.Info("Node has been started", "peer_id", h.ID().String(),
			"addrs", fmt.Sprintf("%v", h.Addrs()))

		// TODO: create a thread to send heartbeat?

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-sendC:
					var err error
					var data []byte
					switch m := (msg).(type) {
					case *consensus.ProposalMessage:
						data, err = encode(m.Proposal)
						if err == nil {
							err = th.Publish(ctx, data)
							p2pMessagesSent.Inc()
						}
					case *consensus.VoteMessage:
						data, err = encode(m.Vote)
						if err == nil {
							err = th.Publish(ctx, data)
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

		for {
			envelope, err := sub.Next(ctx)
			if err != nil {
				return fmt.Errorf("failed to receive pubsub message: %w", err)
			}

			msg, err := decode(envelope.Data)

			if err != nil {
				log.Info("received invalid message",
					"err", err,
					"data", envelope.Data,
					"from", envelope.GetFrom().String())
				p2pMessagesReceived.WithLabelValues("invalid").Inc()
				continue
			}

			if envelope.GetFrom() == h.ID() {
				log.Debug("received message from ourselves, ignoring",
					"payload", msg)
				p2pMessagesReceived.WithLabelValues("loopback").Inc()
				continue
			}

			log.Debug("received message",
				"payload", msg,
				"raw", envelope.Data,
				"from", envelope.GetFrom().String())

			switch m := msg.(type) {
			case *consensus.Proposal:
				obsvC <- consensus.MsgInfo{Msg: &consensus.ProposalMessage{Proposal: m}, PeerID: string(envelope.GetFrom())}
				p2pMessagesReceived.WithLabelValues("observation").Inc()
			case *consensus.Vote:
				obsvC <- consensus.MsgInfo{Msg: &consensus.VoteMessage{Vote: m}, PeerID: string(envelope.GetFrom())}
				p2pMessagesReceived.WithLabelValues("observation").Inc()
			default:
				p2pMessagesReceived.WithLabelValues("unknown").Inc()
				log.Warn("received unknown message type (running outdated software?)",
					"payload", msg,
					"raw", envelope.Data,
					"from", envelope.GetFrom().String())
			}
		}
	}
}
