/*
This package provides the DPOS routes(network addresses) protocol, it can
collect all DPOS peer addresses from the normal P2P network.
*/
package routes

import (
	"fmt"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto"
	dp "github.com/elastos/Elastos.ELA/dpos/p2p/peer"
	"github.com/elastos/Elastos.ELA/elanet/peer"
	"github.com/elastos/Elastos.ELA/events"
	"github.com/elastos/Elastos.ELA/p2p/msg"
)

// Config defines the parameters to create a Route instance.
type Config struct {
	// The PID of this peer if it is an arbiter.
	PID []byte

	// The network address of this arbiter.
	Addr string

	// Sign the addr message of this arbiter.
	Sign func(data []byte) (signature []byte)

	// IsCurrent returns whether BlockChain synced to best height.
	IsCurrent func() bool

	// RelayAddr relays the addresses inventory to the P2P network.
	RelayAddr func(iv *msg.InvVect, data interface{})
}

// cache stores the additional information tracking with a peer.
type cache struct {
	requested map[dp.PID]map[dp.PID]struct{}
	knownInvs map[common.Uint256]struct{}
}

// state stores the DPOS addresses and other additional information tracking
// addresses syncing status.
type state struct {
	peers       map[dp.PID]struct{}
	addrIndex   map[dp.PID]map[dp.PID]common.Uint256
	knownAddr   map[common.Uint256]*msg.DAddr
	unknownAddr map[dp.PID]map[dp.PID]struct{}
	requested   map[common.Uint256]struct{}
	peerCache   map[*peer.Peer]*cache
}

// String creates a human readable string for the state information.
func (s *state) String() string {
	str := fmt.Sprintf("\n----- ROUTE STATE PEERS(%d) -----",
		len(s.peers))
	for pid := range s.peers {
		str += "\n" + pid.String()
	}

	str += fmt.Sprintf("\n----- ROUTE STATE INDEX(%d) -----",
		len(s.addrIndex))
	for pid, idx := range s.addrIndex {
		str += fmt.Sprintf("\n PID(%d):%s", len(idx), pid)
		str += "\n["
		for pid := range idx {
			str += pid.String() + ", "
		}
		if len(idx) > 0 {
			str = str[:len(str)-2]
		}
		str += "]"
	}

	str += fmt.Sprintf("\n----- ROUTE STATE KNOWN(%d) -----",
		len(s.knownAddr))
	for hash := range s.knownAddr {
		str += "\n" + hash.String()
	}

	str += fmt.Sprintf("\n----- ROUTE STATE UNKNOWN(%d) -----",
		len(s.unknownAddr))
	for pid, idx := range s.unknownAddr {
		str += fmt.Sprintf("\n PID(%d):%s", len(idx), pid)
		str += "\n["
		for pid := range idx {
			str += pid.String() + ", "
		}
		str = str[:len(str)-2]
		str += "]"
	}
	str += "\n"
	return str
}

type newPeerMsg *peer.Peer

type donePeerMsg *peer.Peer

type peersMsg struct {
	peers []dp.PID
}

type getDAddrMsg struct {
	peer *peer.Peer
	msg  *msg.GetDAddr
}

type invMsg struct {
	peer *peer.Peer
	msg  *msg.Inv
}

type getDataMsg struct {
	peer *peer.Peer
	msg  *msg.GetData
}

type dAddrMsg struct {
	peer *peer.Peer
	msg  *msg.DAddr
}

type getCipherAddr struct {
	pid   dp.PID
	reply chan []byte
}

// Routes is the DPOS routes implementation.
type Routes struct {
	pid   dp.PID
	cfg   *Config
	addr  string
	sign  func([]byte) []byte
	queue chan interface{}
	quit  chan struct{}
}

// addrHandler is the main handler to syncing the addresses state.
func (r *Routes) addrHandler() {
	state := &state{
		peers:       make(map[dp.PID]struct{}),
		addrIndex:   make(map[dp.PID]map[dp.PID]common.Uint256),
		knownAddr:   make(map[common.Uint256]*msg.DAddr),
		unknownAddr: make(map[dp.PID]map[dp.PID]struct{}),
		requested:   make(map[common.Uint256]struct{}),
		peerCache:   make(map[*peer.Peer]*cache),
	}

out:
	for {
		select {
		// Handle the messages from queue.
		case m := <-r.queue:
			switch m := m.(type) {
			case newPeerMsg:
				r.handleNewPeer(state, m)

			case donePeerMsg:
				r.handleDonePeer(state, m)

			case getDAddrMsg:
				r.handleGetDAddr(state, m.peer, m.msg)

			case invMsg:
				r.handleInv(state, m.peer, m.msg)

			case getDataMsg:
				r.handleGetData(state, m.peer, m.msg)

			case dAddrMsg:
				r.handleDAddr(state, m.peer, m.msg)

			case *peersMsg:
				r.handlePeersMsg(state, m.peers)

			case *getCipherAddr:
				// Ignore non-arbiter address request.
				if _, ok := state.peers[m.pid]; !ok {
					continue
				}

				cipher := r.handleGetCipherAddr(state, m.pid)
				m.reply <- cipher

				// If there are unknown addresses, request them.
				if cipher == nil {
					r.pushGetDAddr(state)
				}
			}

		case <-r.quit:
			break out
		}
	}
}

func (r *Routes) pushGetDAddr(s *state) {
	// If no unknown addresses, do nothing.
	if len(s.unknownAddr) == 0 {
		return
	}

	// Request unknown addresses from other peers.
	for p, c := range s.peerCache {

		// Create get address message.
		getAddr := make(map[[33]byte][][33]byte)

		for pid, ids := range s.unknownAddr {
			// Check if we already requested the unknown address
			// from the peer.
			requested, ok := c.requested[pid]
			if !ok {
				requested = make(map[dp.PID]struct{})
			}

			pids := make([][33]byte, 0, len(ids))
			for id := range ids {
				// Skip requested addresses.
				if _, ok := requested[id]; ok {
					continue
				}

				pids = append(pids, id)
				requested[id] = struct{}{}
			}

			if len(pids) > 0 {
				getAddr[pid] = pids
				c.requested[pid] = requested
			}
		}

		if len(getAddr) > 0 {
			p.QueueMessage(msg.NewGetDAddr(getAddr), nil)
		}
	}
}

func (r *Routes) handleNewPeer(s *state, p *peer.Peer) {
	// Create state for the new peer.
	s.peerCache[p] = &cache{
		requested: make(map[dp.PID]map[dp.PID]struct{}),
		knownInvs: make(map[common.Uint256]struct{}),
	}

	// Push getdaddr if there are unknown addresses.
	r.pushGetDAddr(s)
}

func (r *Routes) handleDonePeer(s *state, p *peer.Peer) {
	c, exists := s.peerCache[p]
	if !exists {
		log.Warnf("Received done peer message for unknown peer %s", p)
		return
	}

	// Remove done peer from peer state.
	delete(s.peerCache, p)

	// Clear cached information.
	for pid := range c.requested {
		delete(c.requested, pid)
	}
	for hash := range c.knownInvs {
		delete(c.knownInvs, hash)
	}
}

// announceAddr broadcast the arbiter's own addr into P2P network.
func (r *Routes) announceAddr(state *state, peers []dp.PID) {
	for _, pid := range peers {
		// Do not create address for self.
		if r.pid.Equal(pid) {
			continue
		}

		pubKey, err := crypto.DecodePoint(pid[:])
		if err != nil {
			continue
		}

		// Generate DAddr for the given PID.
		cipher, err := crypto.Encrypt(pubKey, []byte(r.addr))
		if err != nil {
			log.Warnf("encrypt addr failed %s", err)
			continue
		}
		addr := msg.DAddr{PID: r.pid, Encode: pid, Cipher: cipher}
		addr.Signature = r.sign(addr.Data())

		// Append and relay the address.
		r.appendAddr(state, &addr)
	}
}

func (r *Routes) handlePeersMsg(state *state, peers []dp.PID) {
	// Compare current peers and new peers to find the difference.
	var newPeers = make(map[dp.PID]struct{})
	for _, pid := range peers {
		newPeers[pid] = struct{}{}

		// Initiate address index.
		_, ok := state.addrIndex[pid]
		if !ok {
			state.addrIndex[pid] = make(map[dp.PID]common.Uint256)
		}
	}

	// Find unknown DAddr by query the state index.
	for _, pid := range peers {
		// Do not add self address to unknown.
		if r.pid.Equal(pid) {
			continue
		}

		// Find unknown DAddr of this PID index.
		index := state.addrIndex[pid]
		unknown := make(map[dp.PID]struct{})
		for _, encode := range peers {
			// Do not add self address to unknown.
			if pid.Equal(encode) {
				continue
			}

			if _, ok := index[encode]; ok {
				continue
			}
			unknown[encode] = struct{}{}
		}
		if len(unknown) > 0 {
			state.unknownAddr[pid] = unknown
		}
	}

	// Remove peers that not in new peers list.
	var delPeers []dp.PID
	for pid := range state.peers {
		if _, ok := newPeers[pid]; ok {
			continue
		}
		delPeers = append(delPeers, pid)
	}

	for _, pid := range delPeers {
		// Remove from lack addr.
		delete(state.unknownAddr, pid)

		// Remove from index and known addr.
		pids, ok := state.addrIndex[pid]
		if !ok {
			continue
		}
		for _, pid := range pids {
			delete(state.knownAddr, pid)
		}
		delete(state.addrIndex, pid)
	}

	// Update peers list.
	state.peers = newPeers

	// If this node is an arbiter, announce the address into P2P network.
	if _, ok := state.peers[r.pid]; ok {
		r.announceAddr(state, peers)
	}
}

func (r *Routes) handleGetDAddr(s *state, p *peer.Peer, m *msg.GetDAddr) {
	_, exists := s.peerCache[p]
	if !exists {
		log.Warnf("Received getdaddr message for unknown peer %s", p)
		return
	}

	// Find all known addresses according to the GetDAddr message.
	inv := msg.NewInv()
	for pid, pids := range m.PIDs {
		index, ok := s.addrIndex[pid]
		if !ok {
			continue
		}

		for _, id := range pids {
			hash, ok := index[id]
			if !ok {
				continue
			}
			inv.AddInvVect(msg.NewInvVect(msg.InvTypeAddress, &hash))
		}
	}

	if len(inv.InvList) > 0 {
		p.QueueMessage(inv, nil)
	}
}

func (r *Routes) handleInv(s *state, p *peer.Peer, m *msg.Inv) {
	c, exists := s.peerCache[p]
	if !exists {
		log.Warnf("Received inv message for unknown peer %s", p)
		return
	}

	// Push GetData message according to the Inv message.
	getData := msg.NewGetData()
	for _, iv := range m.InvList {
		switch iv.Type {
		case msg.InvTypeAddress:
		default:
			continue
		}

		// Add the inventory to the cache of known inventory
		// for the peer.
		p.AddKnownInventory(iv)

		_, ok := s.knownAddr[iv.Hash]
		if ok {
			continue
		}

		if _, ok := s.requested[iv.Hash]; ok {
			continue
		}

		c.knownInvs[iv.Hash] = struct{}{}
		s.requested[iv.Hash] = struct{}{}
		getData.AddInvVect(msg.NewInvVect(msg.InvTypeAddress, &iv.Hash))
	}

	if len(getData.InvList) > 0 {
		p.QueueMessage(getData, nil)
	}
}

func (r *Routes) handleGetData(s *state, p *peer.Peer, m *msg.GetData) {
	_, exists := s.peerCache[p]
	if !exists {
		log.Warnf("Received getdata message for unknown peer %s", p)
		return
	}

	done := make(chan struct{})
	for _, iv := range m.InvList {
		switch iv.Type {
		case msg.InvTypeAddress:
			// Attempt to fetch the requested addr.
			addr, ok := s.knownAddr[iv.Hash]
			if !ok {
				done <- struct{}{}
				continue
			}

			p.QueueMessage(addr, done)
			<-done

		default:
			continue
		}
	}
}

func (r *Routes) appendAddr(s *state, m *msg.DAddr) error {
	hash := m.Hash()

	// Append received addr into known addr index.
	index, ok := s.addrIndex[m.PID]
	if !ok {
		return fmt.Errorf("PID not in arbiter list")
	}
	index[m.Encode] = hash
	s.knownAddr[hash] = m
	s.addrIndex[m.PID] = index

	// Remove received addr from unknown addr.
	unknown, ok := s.unknownAddr[m.PID]
	if !ok {
		return fmt.Errorf("PID not in arbiter list")
	}

	_, ok = unknown[m.Encode]
	if !ok {
		return fmt.Errorf("encode not in arbiter list")
	}

	delete(unknown, m.Encode)
	if len(unknown) == 0 {
		delete(s.unknownAddr, m.PID)
	}

	// Relay addr to the P2P network.
	iv := msg.NewInvVect(msg.InvTypeAddress, &hash)
	r.cfg.RelayAddr(iv, m)

	return nil
}

func (r *Routes) handleDAddr(s *state, p *peer.Peer, m *msg.DAddr) {
	c, exists := s.peerCache[p]
	if !exists {
		log.Warnf("Received getdaddr message for unknown peer %s", p)
		return
	}

	hash := m.Hash()

	if _, ok := c.knownInvs[hash]; !ok {
		log.Warnf("Got unrequested addr %s from %s -- disconnecting",
			hash, p)
		p.Disconnect()
		return
	}

	delete(c.knownInvs, hash)
	delete(s.requested, hash)

	if ok := verifyDAddr(m); !ok {
		log.Warnf("Got invalid addr %s from %s -- disconnecting",
			hash, p)
		p.Disconnect()
		return
	}

	// Append received addr into state.
	if err := r.appendAddr(s, m); err != nil {
		log.Warnf("Got invalid addr %s from %s -- disconnecting",
			err, p)
		p.Disconnect()
	}
}

func (r *Routes) handleGetCipherAddr(state *state, pid dp.PID) []byte {
	ciphers, ok := state.addrIndex[pid]
	if !ok {
		return nil
	}

	cipher, ok := ciphers[r.pid]
	if !ok {
		return nil
	}

	addr, ok := state.knownAddr[cipher]
	if !ok {
		return nil
	}

	return addr.Cipher
}

// Start starts the Routes instance to sync DPOS addresses.
func (r *Routes) Start() {
	go r.addrHandler()
}

// Stop quits the syncing address handler.
func (r *Routes) Stop() {
	close(r.quit)
}

// NewPeer notifies the new connected peer.
func (r *Routes) NewPeer(peer *peer.Peer) {
	r.queue <- newPeerMsg(peer)
}

// DonePeer notifies the disconnected peer.
func (r *Routes) DonePeer(peer *peer.Peer) {
	r.queue <- donePeerMsg(peer)
}

// QueueInv adds the passed GetDAddr message and peer to the addr handling queue.
func (r *Routes) QueueGetDAddr(p *peer.Peer, m *msg.GetDAddr) {
	r.queue <- getDAddrMsg{peer: p, msg: m}
}

// QueueInv adds the passed Inv message and peer to the addr handling queue.
func (r *Routes) QueueInv(p *peer.Peer, m *msg.Inv) {
	r.queue <- invMsg{peer: p, msg: m}
}

// QueueInv adds the passed GetData message and peer to the addr handling queue.
func (r *Routes) QueueGetData(p *peer.Peer, m *msg.GetData) {
	r.queue <- getDataMsg{peer: p, msg: m}
}

// QueueInv adds the passed DAddr message and peer to the addr handling queue.
func (r *Routes) QueueDAddr(p *peer.Peer, m *msg.DAddr) {
	r.queue <- dAddrMsg{peer: p, msg: m}
}

// GetCipherAddr get the encrypted network address of the given PID.
func (r *Routes) GetCipherAddr(pid [33]byte) []byte {
	reply := make(chan []byte)
	r.queue <- &getCipherAddr{pid: pid, reply: reply}
	return <-reply
}

// New creates and return a Routes instance.
func New(cfg *Config) *Routes {
	var pid dp.PID
	copy(pid[:], cfg.PID)

	r := Routes{
		pid:   pid,
		cfg:   cfg,
		addr:  cfg.Addr,
		sign:  cfg.Sign,
		queue: make(chan interface{}, 256),
		quit:  make(chan struct{}),
	}

	queuePeers := func(peers []dp.PID) {
		// Ignore if BlockChain not sync to current.
		if !cfg.IsCurrent() {
			return
		}
		r.queue <- &peersMsg{peers: peers}
	}

	events.Subscribe(func(e *events.Event) {
		switch e.Type {
		case events.ETDirectPeersChanged:
			go queuePeers(e.Data.([]dp.PID))
		}
	})
	return &r
}

// verifyDAddr verifies the message was sent by the valid owner.
func verifyDAddr(m *msg.DAddr) bool {
	pubKey, err := crypto.DecodePoint(m.PID[:])
	if err != nil {
		return false
	}

	err = crypto.Verify(*pubKey, m.Data(), m.Signature)
	return err == nil
}
