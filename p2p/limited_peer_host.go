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
	MaxPeers int
}

func WrapHost(h p2pHost, maxPeers int) *LimitedPeerHost {
	return &LimitedPeerHost{
		p2pHost:  h,
		MaxPeers: maxPeers,
	}
}

func (h *LimitedPeerHost) Connect(ctx context.Context, pi peer.AddrInfo) error {
	log.Info("LimitedPeerHost Connect func", "PeerId", pi.ID, "Peer Count", len(h.Network().Peers()))
	// first, check if we're already connected.
	if h.Network().Connectedness(pi.ID) == network.Connected {
		log.Info("LimitedPeerHost Connect func, peer exist.", "PeerId", pi.ID)
		return nil
	}

	if len(h.Network().Peers()) > h.MaxPeers {
		log.Info("LimitedPeerHost Connect func", "limit", h.MaxPeers, "Peer Count", len(h.Network().Peers()))
		for index, p := range h.Network().Peers() {
			log.Info("Peer List", "index", index, "peer id", p.String(), "connected", h.Network().Connectedness(p))
		}
		return fmt.Errorf("PeerCount %d exceeds the limit %d.\r\n", len(h.Network().Peers()), h.MaxPeers)
	}

	return h.p2pHost.Connect(ctx, pi)
}
