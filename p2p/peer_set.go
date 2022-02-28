package p2p

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

type PeerConnectionState int

const (
	// PeerDisconnected means there is no connection to the peer.
	PeerDisconnected PeerConnectionState = iota
	// PeerDisconnecting means there is an on-going attempt to disconnect from the peer.
	PeerDisconnecting
	// PeerConnected means the peer has an active connection.
	PeerConnected
	// PeerConnecting means there is an on-going attempt to connect to the peer.
	PeerConnecting
)

const (
	// ColocationLimit restricts how many peer identities we can see from a single ip or ipv6 subnet.
	ColocationLimit = 5
)

var (
	// ErrPeerUnknown is returned when there is an attempt to obtain data from a peer that is not known.
	ErrPeerUnknown = errors.New("peer unknown")
)

type PeerSet struct {
	lock sync.RWMutex

	peerMap   map[peer.ID]*PeerData
	ipTracker map[string]uint64
}

func NewPeerSet() *PeerSet {
	return &PeerSet{
		peerMap:   make(map[peer.ID]*PeerData),
		ipTracker: make(map[string]uint64),
	}
}

// PeerDataGetOrCreate returns data associated with a given peer.
// If no data has been associated yet, newly created and associated data object is returned.
// Important: it is assumed that store mutex is locked when calling this method.
func (s *PeerSet) peerDataGetOrCreate(pid peer.ID) *PeerData {
	if peerData, ok := s.peerMap[pid]; ok {
		return peerData
	}
	s.peerMap[pid] = &PeerData{}
	return s.peerMap[pid]
}

// PeerData returns data associated with a given peer, if any.
// Important: it is assumed that store mutex is locked when calling this method.
func (s *PeerSet) peerData(pid peer.ID) (*PeerData, bool) {
	peerData, ok := s.peerMap[pid]
	return peerData, ok
}

// PeerData returns data associated with a given peer, if any.
// Important: it is assumed that store mutex is locked when calling this method.
func (s *PeerSet) setPeerData(pid peer.ID, peerData *PeerData) {
	s.peerMap[pid] = peerData
}

// SetConnectionState sets the connection state of the given remote peer.
func (p *PeerSet) SetConnectionState(pid peer.ID, state PeerConnectionState) {
	p.lock.Lock()
	defer p.lock.Unlock()

	peerData := p.peerDataGetOrCreate(pid)
	peerData.ConnState = state
}

// ConnectionState gets the connection state of the given remote peer.
// This will error if the peer does not exist.
func (p *PeerSet) ConnectionState(pid peer.ID) (PeerConnectionState, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if peerData, ok := p.peerData(pid); ok {
		return peerData.ConnState, nil
	}
	return PeerDisconnected, ErrPeerUnknown
}

// Add adds a peer.
// If a peer already exists with this ID its address and direction are updated with the supplied data.
func (p *PeerSet) Add(record *enr.Record, pid peer.ID, address ma.Multiaddr, direction network.Direction) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if peerData, ok := p.peerData(pid); ok {
		// Peer already exists, just update its address info.
		prevAddress := peerData.Address
		peerData.Address = address
		peerData.Direction = direction
		if record != nil {
			peerData.Enr = record
		}
		if !sameIP(prevAddress, address) {
			p.addIpToTracker(pid)
		}
		return
	}
	peerData := &PeerData{
		Address:   address,
		Direction: direction,
		// Peers start disconnected; state will be updated when the handshake process begins.
		ConnState: PeerDisconnected,
	}
	if record != nil {
		peerData.Enr = record
	}
	p.setPeerData(pid, peerData)
	p.addIpToTracker(pid)
}

func (p *PeerSet) GetActive(peerId peer.ID) *PeerData {
	p.lock.Lock()
	defer p.lock.Unlock()

	if peerData, ok := p.peerData(peerId); ok {
		if peerData.ConnState == PeerConnected || peerData.ConnState == PeerConnecting {
			return peerData
		}
		return nil
	}
	return nil
}

// Active returns the peers that are connecting or connected.
func (p *PeerSet) Active() []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, peerData := range p.peerMap {
		if peerData.ConnState == PeerConnecting || peerData.ConnState == PeerConnected {
			peers = append(peers, pid)
		}
	}
	return peers
}

// IsBad states if the peer is to be considered bad (by *any* of the registered scorers).
// If the peer is unknown this will return `false`, which makes using this function easier than returning an error.
func (p *PeerSet) IsBad(pid peer.ID) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.isBad(pid)
}

// isBad is the lock-free version of IsBad.
func (p *PeerSet) isBad(pid peer.ID) bool {
	return p.isfromBadIP(pid)
	//|| p.scorers.IsBadPeerNoLock(pid)
}

// this method assumes the store lock is acquired before
// executing the method.
func (p *PeerSet) isfromBadIP(pid peer.ID) bool {
	peerData, ok := p.peerData(pid)
	if !ok {
		return false
	}
	if peerData.Address == nil {
		return false
	}
	ip, err := manet.ToIP(peerData.Address)
	if err != nil {
		return true
	}
	if val, ok := p.ipTracker[ip.String()]; ok {
		if val > ColocationLimit {
			return true
		}
	}
	return false
}

func (p *PeerSet) addIpToTracker(pid peer.ID) {
	data, ok := p.peerData(pid)
	if !ok {
		return
	}
	if data.Address == nil {
		return
	}
	ip, err := manet.ToIP(data.Address)
	if err != nil {
		// Should never happen, it is
		// assumed every IP coming in
		// is a valid ip.
		return
	}
	// Ignore loopback addresses.
	if ip.IsLoopback() {
		return
	}
	stringIP := ip.String()
	p.ipTracker[stringIP] += 1
}

func sameIP(firstAddr, secondAddr ma.Multiaddr) bool {
	// Exit early if we do get nil multiaddresses
	if firstAddr == nil || secondAddr == nil {
		return false
	}
	firstIP, err := manet.ToIP(firstAddr)
	if err != nil {
		return false
	}
	secondIP, err := manet.ToIP(secondAddr)
	if err != nil {
		return false
	}
	return firstIP.Equal(secondIP)
}
