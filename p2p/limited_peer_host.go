package p2p

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

type p2pHost host.Host

type LimitedPeerHost struct {
	p2pHost
	limit int
}

func WrapHost(h p2pHost, limit int) *LimitedPeerHost {
	return &LimitedPeerHost{
		p2pHost: h,
		limit:   limit,
	}
}

func (h *LimitedPeerHost) Connect(ctx context.Context, pi peer.AddrInfo) error {
	log.Info("LimitedPeerHost Connect func", "PeerId", pi.ID, "Peer Count", h.Peerstore().Peers().Len())
	// first, check if we're already connected.
	if h.Network().Connectedness(pi.ID) == network.Connected {
		return nil
	}

	if h.Peerstore().Peers().Len() > h.limit {
		return fmt.Errorf("PeerCount %d exceeds the limit %d.\r\n", h.Peerstore().Peers().Len(), h.limit)
	}

	return h.p2pHost.Connect(ctx, pi)
}
