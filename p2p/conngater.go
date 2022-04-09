package p2p

import (
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p-core/control"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"sync"
)

type connGater struct {
	sync.RWMutex
	h            host.Host
	MaxPeerCount int
}

func (cg *connGater) isPeerAtLimit() bool {
	cg.RLock()
	defer cg.RUnlock()
	if cg == nil {
		return false
	}
	if len(cg.h.Network().Peers()) >= cg.MaxPeerCount {
		log.Info("isPeerAtLimit", "MaxPeerCount", cg.MaxPeerCount, "Peer Count", len(cg.h.Network().Peers()))
		for index, p := range cg.h.Network().Peers() {
			log.Info("Peer List", "index", index, "peer id", p.String(), "connected", cg.h.Network().Connectedness(p))
		}
		log.Error(fmt.Sprintf("PeerCount %d exceeds the MaxPeerCount %d.\r\n", len(cg.h.Network().Peers()), cg.MaxPeerCount))
		return true
	}
	return false
}

func (cg *connGater) InterceptPeerDial(p peer.ID) (allow bool) {
	return cg.h.Network().Connectedness(p) == network.Connected || !cg.isPeerAtLimit()
}

func (cg *connGater) InterceptAddrDial(p peer.ID, a ma.Multiaddr) (allow bool) {
	return cg.h.Network().Connectedness(p) == network.Connected || !cg.isPeerAtLimit()
}

func (cg *connGater) InterceptAccept(cma network.ConnMultiaddrs) (allow bool) {
	return true
}

func (cg *connGater) InterceptSecured(dir network.Direction, p peer.ID, cma network.ConnMultiaddrs) (allow bool) {
	return cg.h.Network().Connectedness(p) == network.Connected || !cg.isPeerAtLimit()
}

func (cg *connGater) InterceptUpgraded(network.Conn) (allow bool, reason control.DisconnectReason) {
	return true, 0
}
