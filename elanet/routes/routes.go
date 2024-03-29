// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

/*
This package provides the DPOS routes(network addresses) protocol, it can
collect all DPOS peer addresses from the normal P2P network.
*/
package routes

import (
	"container/list"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto"
	dp "github.com/elastos/Elastos.ELA/dpos/p2p/peer"
	"github.com/elastos/Elastos.ELA/elanet/peer"
	"github.com/elastos/Elastos.ELA/events"
	"github.com/elastos/Elastos.ELA/p2p/msg"
)

const (
	// minPeersToAnnounce defines the minimum connected peers to announce
	// DPOS address into the P2P network.
	minPeersToAnnounce = 5

	// retryAnnounceDuration defines the time duration to retry an announce.
	retryAnnounceDuration = 3 * time.Second

	// maxTimeOffset indicates the maximum time offset with the to accept an
	// DAddr message.
	maxTimeOffset = 30 * time.Second

	// minAnnounceDuration indicates the minimum allowed time duration to
	// announce a new DAddr.
	minAnnounceDuration = 30 * time.Second

	// maxKnownAddrs indicates the maximum known DAddrs cached in memory.
	// The maximum of DAddrs can be calculated as [36(current)+72(candidate)]².
	maxKnownAddrs = 108 * 110
)

// cache stores the requested DAddrs from a peer.
type cache struct {
	requested map[common.Uint256]struct{}
}

// Config defines the parameters to create a Route instance.
type Config struct {
	// The PID of this peer if it is an arbiter.
	PID []byte

	// The network address of this arbiter.
	Addr string

	// TimeSource is the median time source of the P2P network.
	TimeSource blockchain.MedianTimeSource

	// Sign the addr message of this arbiter.
	Sign func(data []byte) (signature []byte)

	// IsCurrent returns whether BlockChain synced to best height.
	IsCurrent func() bool

	// RelayAddr relays the addresses inventory to the P2P network.
	RelayAddr func(iv *msg.InvVect, data interface{})

	// OnCipherAddr will be invoked when an address cipher received.
	OnCipherAddr func(pid dp.PID, addr []byte)
}

// state stores the DPOS addresses and other additional information tracking
// addresses syncing status.
type state struct {
	dposPeers map[dp.PID]struct{}
	crPeers   map[dp.PID]struct{}
	requested map[common.Uint256]struct{}
	peerCache map[*peer.Peer]*cache
}

type newPeerMsg *peer.Peer

type donePeerMsg *peer.Peer

type peersMsg struct {
	dposPeers []dp.PID
	crPeers   []dp.PID
}

type invMsg struct {
	peer *peer.Peer
	msg  *msg.Inv
}

type dAddrMsg struct {
	peer *peer.Peer
	msg  *msg.DAddr
}

// Routes is the DPOS routes implementation.
type Routes struct {
	pid  dp.PID
	cfg  *Config
	addr string
	sign func([]byte) []byte

	// The following variables must only be used atomically.
	started   int32
	stopped   int32
	waiting   int32
	crWaiting int32

	addrMtx   sync.RWMutex
	addrIndex map[dp.PID]map[dp.PID]common.Uint256
	knownAddr map[common.Uint256]*msg.DAddr
	knownList *list.List

	queue      chan interface{}
	announce   chan struct{}
	crAnnounce chan struct{}
	quit       chan struct{}
}

// addrHandler is the main handler to syncing the addresses state.
func (r *Routes) addrHandler() {
	state := &state{
		dposPeers: make(map[dp.PID]struct{}),
		crPeers:   make(map[dp.PID]struct{}),
		requested: make(map[common.Uint256]struct{}),
		peerCache: make(map[*peer.Peer]*cache),
	}

	// lastAnnounce indicates the time when last announce sent.
	var lastAnnounce time.Time

	// lastCRAnnounce indicates the time when last cr announce sent.
	var lastCRAnnounce time.Time

	// scheduleAnnounce schedules an announce according to the delay time.
	var scheduleAnnounce = func(delay time.Duration) {
		time.AfterFunc(delay, func() {
			r.announce <- struct{}{}
		})
	}

	// scheduleCRAnnounce schedules an announce according to the delay time.
	var scheduleCRAnnounce = func(delay time.Duration) {
		time.AfterFunc(delay, func() {
			r.crAnnounce <- struct{}{}
		})
	}

	ciphers := make(map[string]map[dp.PID][]byte)
	sentDPoSPeers := make(map[dp.PID]uint32)
	sentCrPeers := make(map[dp.PID]uint32)
	var roundDPoS uint32 = 1
	var roundCr uint32 = 1

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

			case invMsg:
				r.handleInv(state, m.peer, m.msg)

			case dAddrMsg:
				r.handleDAddr(state, m.peer, m.msg)

			case peersMsg:
				r.handlePeersMsg(state, m.dposPeers, m.crPeers)

			}

		// Handle the announce request.
		case <-r.announce:
			// This may be a retry or delayed announce, and the DPoS producers
			// have been changed.
			_, ok := state.dposPeers[r.pid]
			if !ok {
				// Waiting status must reset here or the announce will never
				// work again.
				atomic.StoreInt32(&r.waiting, 0)
				continue
			}

			// Do not announce address if connected peers not enough.
			if len(state.peerCache) < minPeersToAnnounce {
				// Retry announce after the retry duration.
				scheduleAnnounce(retryAnnounceDuration)
				continue
			}

			// Do not announce address too frequent.
			now := time.Now()
			if lastAnnounce.Add(minAnnounceDuration).After(now) {
				// Calculate next announce time and schedule an announce.
				nextAnnounce := minAnnounceDuration - now.Sub(lastAnnounce)
				scheduleAnnounce(nextAnnounce)
				continue
			}

			// Update last announce time.
			lastAnnounce = now

			// Reset waiting state to 0(false).
			atomic.StoreInt32(&r.waiting, 0)
			// add round atomicly
			atomic.AddUint32(&roundDPoS, 1)

			for pid := range state.dposPeers {
				// Do not create address for self.
				if r.pid.Equal(pid) {
					continue
				}

				// Do not create address when in DPoS peers
				if sentCrPeers[pid] == roundCr {
					continue
				}

				pubKey, err := crypto.DecodePoint(pid[:])
				if err != nil {
					continue
				}

				var cipher []byte
				if pidc, ok := ciphers[r.addr]; ok {
					if c, ok := pidc[pid]; ok {
						cipher = c
					}
				}

				if len(cipher) == 0 {
					// Generate DAddr for the given PID.
					cipher, err = crypto.Encrypt(pubKey, []byte(r.addr))
					if err != nil {
						log.Warnf("encrypt addr %s failed %s", r.addr, err)
						continue
					}
					ciphers[r.addr] = make(map[dp.PID][]byte)
					ciphers[r.addr][pid] = cipher
				}

				addr := msg.DAddr{
					PID:       r.pid,
					Timestamp: r.cfg.TimeSource.AdjustedTime(),
					Encode:    pid,
					Cipher:    cipher,
				}
				addr.Signature = r.sign(addr.Data())

				sentDPoSPeers[pid] = roundDPoS
				// Append and relay the local address.
				r.appendAddr(&addr)
			}

			// Handle the announce request.
		case <-r.crAnnounce:
			// This may be a retry or delayed announce, and the DPoS producers
			// have been changed.
			_, ok := state.crPeers[r.pid]
			if !ok {
				// Waiting status must reset here or the announce will never
				// work again.
				atomic.StoreInt32(&r.crWaiting, 0)
				continue
			}

			// Do not announce address if connected peers not enough.
			if len(state.peerCache) < minPeersToAnnounce {
				// Retry announce after the retry duration.
				scheduleCRAnnounce(retryAnnounceDuration)
				continue
			}

			// Do not announce address too frequent.
			now := time.Now()
			if lastCRAnnounce.Add(minAnnounceDuration).After(now) {
				// Calculate next announce time and schedule an announce.
				nextAnnounce := minAnnounceDuration - now.Sub(lastCRAnnounce)
				scheduleCRAnnounce(nextAnnounce)
				continue
			}

			// Update last announce time.
			lastCRAnnounce = now

			// Reset waiting state to 0(false).
			atomic.StoreInt32(&r.crWaiting, 0)
			// add round atomicly
			atomic.AddUint32(&roundCr, 1)

			for pid := range state.crPeers {
				// Do not create address for self.
				if r.pid.Equal(pid) {
					continue
				}

				// Do not create address when in DPoS peers
				if sentDPoSPeers[pid] == roundDPoS {
					continue
				}

				pubKey, err := crypto.DecodePoint(pid[:])
				if err != nil {
					continue
				}

				// Generate DAddr for the given PID.
				cipher, err := crypto.Encrypt(pubKey, []byte(r.addr))
				if err != nil {
					log.Warnf("encrypt addr %s failed %s", r.addr, err)
					continue
				}
				addr := msg.DAddr{
					PID:       r.pid,
					Timestamp: r.cfg.TimeSource.AdjustedTime(),
					Encode:    pid,
					Cipher:    cipher,
				}
				addr.Signature = r.sign(addr.Data())

				sentCrPeers[pid] = roundCr
				// Append and relay the local address.
				r.appendAddr(&addr)
			}

		case <-r.quit:
			break out
		}
	}

cleanup:
	for {
		select {
		case <-r.queue:
		case <-r.announce:
		case <-r.crAnnounce:
		default:
			break cleanup
		}
	}
}

func (r *Routes) appendAddr(m *msg.DAddr) {
	hash := m.Hash()

	// Append received addr into known addr index.
	r.addrMtx.Lock()
	if _, ok := r.addrIndex[m.PID]; !ok {
		r.addrMtx.Unlock()
		return
	}
	r.addrIndex[m.PID][m.Encode] = hash
	r.knownAddr[hash] = m
	if len(r.knownAddr) > maxKnownAddrs {
		node := r.knownList.Back()
		lru := node.Value.(common.Uint256)

		delete(r.knownAddr, lru)

		node.Value = hash
		r.knownList.MoveToFront(node)
	} else {
		r.knownList.PushFront(hash)
	}
	r.addrMtx.Unlock()

	// Relay addr to the P2P network.
	iv := msg.NewInvVect(msg.InvTypeAddress, &hash)
	r.cfg.RelayAddr(iv, m)
}

func (r *Routes) announceAddr() {
	// Ignore if BlockChain not sync to current.
	if !r.cfg.IsCurrent() {
		return
	}

	// Refuse new announce if a previous announce is waiting,
	// this is to reduce unnecessary announce.
	if !atomic.CompareAndSwapInt32(&r.waiting, 0, 1) {
		return
	}
	r.announce <- struct{}{}
}

func (r *Routes) announceCRAddr() {
	// Ignore if BlockChain not sync to current.
	if !r.cfg.IsCurrent() {
		return
	}

	// Refuse new announce if a previous announce is waiting,
	// this is to reduce unnecessary announce.
	if !atomic.CompareAndSwapInt32(&r.crWaiting, 0, 1) {
		return
	}
	r.crAnnounce <- struct{}{}
}

func (r *Routes) handleNewPeer(s *state, p *peer.Peer) {
	// Create state for the new peer.
	s.peerCache[p] = &cache{requested: make(map[common.Uint256]struct{})}
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
}

func (r *Routes) handlePeersMsg(state *state, dposPeers []dp.PID, crPeers []dp.PID) {

	// Compare current peers and new peers to find the difference.
	peers := append(dposPeers, crPeers...)
	var newDPoSPeers = make(map[dp.PID]struct{})
	var newCRPeers = make(map[dp.PID]struct{})
	for _, pid := range dposPeers {
		newDPoSPeers[pid] = struct{}{}
	}
	for _, pid := range crPeers {
		newCRPeers[pid] = struct{}{}
	}
	var newPeers = make(map[dp.PID]struct{})
	for _, pid := range peers {
		newPeers[pid] = struct{}{}

		// Initiate address index.
		r.addrMtx.RLock()
		_, ok := r.addrIndex[pid]
		r.addrMtx.RUnlock()
		if !ok {
			r.addrMtx.Lock()
			r.addrIndex[pid] = make(map[dp.PID]common.Uint256)
			r.addrMtx.Unlock()
		}
	}

	// Remove peers that not in new peers list.
	var delPeers []dp.PID
	for pid := range state.dposPeers {
		if _, ok := newPeers[pid]; ok {
			continue
		}
		delPeers = append(delPeers, pid)
	}
	for pid := range state.crPeers {
		if _, ok := newPeers[pid]; ok {
			continue
		}
		delPeers = append(delPeers, pid)
	}

	for _, pid := range delPeers {
		// Remove from index and known addr.
		r.addrMtx.RLock()
		pids, ok := r.addrIndex[pid]
		r.addrMtx.RUnlock()
		if !ok {
			continue
		}

		r.addrMtx.Lock()
		for _, pid := range pids {
			delete(r.knownAddr, pid)
		}
		delete(r.addrIndex, pid)
		r.addrMtx.Unlock()
	}

	var isDPoSArbiter, isCRArbiter bool
	for _, p := range dposPeers {
		if r.pid == p {
			isDPoSArbiter = true
			break
		}
	}
	for _, p := range crPeers {
		if r.pid == p {
			isCRArbiter = true
			break
		}
	}
	_, wasArbiter := state.dposPeers[r.pid]
	_, wasCRArbiter := state.crPeers[r.pid]

	// Update peers list.
	state.dposPeers = newDPoSPeers
	state.crPeers = newCRPeers

	// Announce address into P2P network if we become arbiter.
	if isDPoSArbiter && !wasArbiter {
		r.announceAddr()
	}
	if isCRArbiter && !wasCRArbiter {
		r.announceCRAddr()
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

		r.addrMtx.RLock()
		_, ok := r.knownAddr[iv.Hash]
		r.addrMtx.RUnlock()
		if ok {
			continue
		}

		if _, ok := s.requested[iv.Hash]; ok {
			continue
		}

		c.requested[iv.Hash] = struct{}{}
		s.requested[iv.Hash] = struct{}{}
		getData.AddInvVect(msg.NewInvVect(msg.InvTypeAddress, &iv.Hash))
	}

	if len(getData.InvList) > 0 {
		p.QueueMessage(getData, nil)
	}
}

// verifyDAddr verifies if this is a valid DPOS address message.
func (r *Routes) verifyDAddr(s *state, m *msg.DAddr) error {
	// Verify signature of the message.
	pubKey, err := crypto.DecodePoint(m.PID[:])
	if err != nil {
		return fmt.Errorf("invalid public key")
	}
	err = crypto.Verify(*pubKey, m.Data(), m.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature")
	}

	// Verify timestamp of the message. A DAddr to same arbiter can not be sent
	// frequently to prevent attack, and a DAddr timestamp must not to far from
	// the P2P network median time.
	r.addrMtx.RLock()
	defer r.addrMtx.RUnlock()
	if index, ok := r.addrIndex[m.PID]; ok {
		if hash, ok := index[m.Encode]; ok {
			ka, ok := r.knownAddr[hash]
			if !ok {
				// This may happen if the known DAddr has been deleted because
				// maxKnownAddrs arrived.  In this case we do not return any
				// error.
				log.Debugf("unknown addr %s", hash)
				return nil
			}

			// Abandon address older than the known address to the same arbiter.
			if ka.Timestamp.After(m.Timestamp) {
				return fmt.Errorf("timestamp is older than known")
			}

			// Check if timestamp out of median time offset.
			medianTime := r.cfg.TimeSource.AdjustedTime()
			minTime := medianTime.Add(-maxTimeOffset)
			maxTime := medianTime.Add(maxTimeOffset)
			if m.Timestamp.Before(minTime) || m.Timestamp.After(maxTime) {
				return fmt.Errorf("timestamp out of offset range")
			}

			// Check if the address announces too frequent.
			if ka.Timestamp.Add(minAnnounceDuration).After(m.Timestamp) {
				return fmt.Errorf("address announce too frequent")
			}
		}
	}

	return nil
}

func (r *Routes) handleDAddr(s *state, p *peer.Peer, m *msg.DAddr) {
	c, exists := s.peerCache[p]
	if !exists {
		log.Warnf("Received getdaddr message for unknown peer %s", p)
		return
	}

	hash := m.Hash()

	if _, ok := c.requested[hash]; !ok {
		log.Warnf("Got unrequested addr %s from %s -- disconnecting",
			hash, p)
		p.Disconnect()
		return
	}

	delete(c.requested, hash)
	delete(s.requested, hash)

	if err := r.verifyDAddr(s, m); err != nil {
		log.Warnf("Got invalid addr %s %s from %s -- disconnecting",
			hash, err, p)
		p.Disconnect()
		return
	}

	_, isCRPeers := s.crPeers[m.PID]
	_, isDPoSPeers := s.dposPeers[m.PID]
	if !isCRPeers && !isDPoSPeers {
		log.Debugf("PID not in arbiter list")

		// Peers may have disagree with the current producers, so some times we
		// receive addresses that not in the producers list.  We do not
		// disconnect the peer even the address not in producers list.
		return
	}

	// Append received addr into state.
	r.appendAddr(m)

	// Notify the received DPOS address if the Encode matches.
	if r.pid.Equal(m.Encode) && r.cfg.OnCipherAddr != nil {
		r.cfg.OnCipherAddr(m.PID, m.Cipher)
	}
}

// Start starts the Routes instance to sync DPOS addresses.
func (r *Routes) Start() {
	if !atomic.CompareAndSwapInt32(&r.started, 0, 1) {
		return
	}
	go r.addrHandler()
}

// Stop quits the syncing address handler.
func (r *Routes) Stop() {
	if !atomic.CompareAndSwapInt32(&r.stopped, 0, 1) {
		return
	}
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

// QueueInv adds the passed Inv message and peer to the addr handling queue.
func (r *Routes) QueueInv(p *peer.Peer, m *msg.Inv) {
	// Filter non-address inventory messages.
	for _, iv := range m.InvList {
		if iv.Type == msg.InvTypeAddress {
			r.queue <- invMsg{peer: p, msg: m}
			return
		}
	}
}

// OnGetData handles the passed GetData message of the peer.
func (r *Routes) OnGetData(p *peer.Peer, m *msg.GetData) {
	done := make(chan struct{}, 1)
	for _, iv := range m.InvList {
		switch iv.Type {
		case msg.InvTypeAddress:
			// Attempt to fetch the requested addr.
			r.addrMtx.RLock()
			addr, ok := r.knownAddr[iv.Hash]
			r.addrMtx.RUnlock()
			if !ok {
				log.Warnf("%s for DAddr not found", iv.Hash)
				continue
			}

			p.QueueMessage(addr, done)
			<-done

		default:
			continue
		}
	}
}

// QueueInv adds the passed DAddr message and peer to the addr handling queue.
func (r *Routes) QueueDAddr(p *peer.Peer, m *msg.DAddr) {
	r.queue <- dAddrMsg{peer: p, msg: m}
}

// AnnounceAddr schedules an local address announce to the P2P network, it used
// to re-announce the local address when DPoS network go bad.
func (r *Routes) AnnounceAddr() {
	if atomic.LoadInt32(&r.started) == 0 {
		return
	}
	r.announceAddr()
}

// New creates and return a Routes instance.
func New(cfg *Config) *Routes {
	var pid dp.PID
	copy(pid[:], cfg.PID)

	r := Routes{
		pid:        pid,
		cfg:        cfg,
		addr:       cfg.Addr,
		sign:       cfg.Sign,
		addrIndex:  make(map[dp.PID]map[dp.PID]common.Uint256),
		knownAddr:  make(map[common.Uint256]*msg.DAddr),
		knownList:  list.New(),
		queue:      make(chan interface{}, 125),
		announce:   make(chan struct{}, 1),
		crAnnounce: make(chan struct{}, 1),
		quit:       make(chan struct{}),
	}

	queuePeers := func(dposPeers []dp.PID, crPeers []dp.PID) {
		r.queue <- peersMsg{dposPeers: dposPeers, crPeers: crPeers}
	}

	events.Subscribe(func(e *events.Event) {
		switch e.Type {
		case events.ETDirectPeersChanged:
			peersInfo := e.Data.(*dp.PeersInfo)
			peers := peersInfo.CurrentPeers
			peers = append(peers, peersInfo.NextPeers...)
			go queuePeers(peers, peersInfo.CRPeers)
		}
	})
	return &r
}
