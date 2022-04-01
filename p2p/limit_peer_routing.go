package p2p

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

type LimitPeerRouting struct {
	h     host.Host
	dht   *dht.IpfsDHT
	limit int
}

func Wrap(h host.Host, dht *dht.IpfsDHT, limit int) routing.PeerRouting {
	return &LimitPeerRouting{
		h:     h,
		dht:   dht,
		limit: limit,
	}
}

func (lpr *LimitPeerRouting) FindPeer(ctx context.Context, id peer.ID) (peer.AddrInfo, error) {
	log.Info("FindPeer", "PeerId", id)
	// Check if were already connected to them
	if pi := lpr.dht.FindLocal(id); pi.ID != "" {
		return pi, nil
	}

	if lpr.h.Peerstore().Peers().Len() > lpr.limit {
		return peer.AddrInfo{},
			fmt.Errorf("PeerCount %d exceeds the limit %d.\r\n", lpr.h.Peerstore().Peers().Len(), lpr.limit)
	}

	return lpr.dht.FindPeer(ctx, id)
}
