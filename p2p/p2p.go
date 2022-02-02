package p2p

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-minimal-pbft/consensus"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/libp2p/go-libp2p-core/network"
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

const (
	MsgProposal      = 0x01
	MsgVote          = 0x02
	MsgVerifiedBlock = 0x03
	MsgHelloRequest  = 0x04
	MsgHelloResponse = 0x05
)

func init() {
	prometheus.MustRegister(p2pHeartbeatsSent)
	prometheus.MustRegister(p2pMessagesSent)
	prometheus.MustRegister(p2pMessagesReceived)
	decoder[1] = decodeProposal
	decoder[2] = decodeVote
	decoder[3] = decodeVerifiedBlock
	decoder[4] = decodeHelloRequest
	decoder[5] = decodeHelloResponse
	decoder[6] = decodeGetVerifiedBlockRequest
}

type HelloRequest struct {
	LastHeight uint64
}

type HelloResponse struct {
	LastHeight uint64
}

type GetVerifiedBlockRequest struct {
	Height uint64
}

func decodeHelloRequest(data []byte) (interface{}, error) {
	var h HelloRequest
	err := rlp.DecodeBytes(data, &h)
	return h, err
}

func decodeHelloResponse(data []byte) (interface{}, error) {
	var h HelloResponse
	err := rlp.DecodeBytes(data, &h)
	return h, err
}

func (req *HelloRequest) ValidateBasic() error {
	return nil
}

func (req *HelloResponse) ValidateBasic() error {
	return nil
}

func decodeGetVerifiedBlockRequest(data []byte) (interface{}, error) {
	var req GetVerifiedBlockRequest
	err := rlp.DecodeBytes(data, &req)
	return req, err
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

func decodeVerifiedBlock(data []byte) (interface{}, error) {
	var b consensus.VerifiedBlock
	err := rlp.DecodeBytes(data, &b)
	return b, err
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

func encodeRaw(msg interface{}) ([]byte, error) {
	var data []byte
	var err error
	switch m := msg.(type) {
	case *consensus.Proposal:
		data, err = encodeProposal(m)
	case *consensus.Vote:
		data, err = encodeVote(m)
	case *consensus.VerifiedBlock:
		data, err = rlp.EncodeToBytes(m)
	case *HelloRequest:
		data, err = rlp.EncodeToBytes(m)
	case *HelloResponse:
		data, err = rlp.EncodeToBytes(m)
	}
	return data, err
}

func encode(msg interface{}) ([]byte, error) {
	var data []byte
	var err error
	var typeData []byte
	switch msg.(type) {
	case *consensus.Proposal:
		typeData = []byte{1}
	case *consensus.Vote:
		typeData = []byte{2}
	case *consensus.VerifiedBlock:
		typeData = []byte{3}
	case *HelloRequest:
		typeData = []byte{4}
	case *HelloResponse:
		typeData = []byte{5}
	case *GetVerifiedBlockRequest:
		typeData = []byte{6}
	}

	data, err = encodeRaw(msg)

	if err != nil {
		return nil, err
	}
	return append(typeData, data...), nil
}

func writeMsg(stream network.Stream, msg []byte) error {
	size := make([]byte, 4)
	binary.BigEndian.PutUint32(size, uint32(len(msg)))
	n, err := stream.Write(size)
	if err != nil {
		return err
	}
	if n != len(size) {
		return fmt.Errorf("not fully write")
	}

	n, err = stream.Write(msg)
	if err != nil {
		return err
	}
	if n != len(msg) {
		return fmt.Errorf("not fully write")
	}
	return nil
}

func Run(
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

		h.SetStreamHandler("/go-minimal-pbft/steam/1.0.0", func(stream network.Stream) {
			defer stream.Close()
			r := bufio.NewReader(stream)

			for {
				pktSize := make([]byte, 4)
				_, err := io.ReadFull(r, pktSize)
				if err != nil {
					return
				}

				size := binary.BigEndian.Uint32(pktSize)
				data := make([]byte, size)
				_, err = io.ReadFull(r, data)
				if err != nil {
					return
				}

				msg, err := decode(data)

				if err != nil {
					return
				}

				log.Debug("received message",
					"payload", msg,
					"raw", data)

				switch m := msg.(type) {
				case *HelloRequest:
					resp, err := encode(&HelloResponse{state.GetLastHeight()})
					if err != nil {
						return
					}
					err = writeMsg(stream, resp)
					if err != nil {
						return
					}
				case *consensus.VerifiedBlock:
					fmt.Println(m.Height)
				}
			}
		})

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

			if envelope.GetFrom() == h.ID() {
				log.Debug("received message from ourselves, ignoring",
					"payload", envelope.Data)
				p2pMessagesReceived.WithLabelValues("loopback").Inc()
				continue
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
			case *HelloRequest:
			case *HelloResponse:
			case *GetVerifiedBlockRequest:
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
