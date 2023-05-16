// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

/*
Hub is a network hub to provide different services through one network address.
*/
package hub

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastos/Elastos.ELA/dpos/p2p/addrmgr"
	"github.com/elastos/Elastos.ELA/dpos/p2p/msg"
	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
	"github.com/elastos/Elastos.ELA/events"
	"github.com/elastos/Elastos.ELA/p2p"
	"github.com/elastos/Elastos.ELA/utils/signal"
)

const (
	// buffSize is the data buffer size for each pipe way (1KB), and there are
	// to ways for each pipe instance (2KB). If there are 100 pipe instances,
	// they will take 200KB memory cache, is not too large for a computer that
	// have a 8GB(1024MB*8) or larger memory.
	buffSize = 1024 // 1KB

	// pipeTimeout defines the time duration to timeout a pipe.
	pipeTimeout = 2 * time.Minute
)

// pipe represent a pipeline from the local connection to the mapping net
// address.
type pipe struct {
	closed int32
	inlet  net.Conn
	outlet net.Conn
}

// start creates the data pipeline between inlet and outlet.
func (p *pipe) start(manager *addrmgr.AddrManager, connChan chan bool) {
	// Create two way flow between inlet and outlet.
	go p.flow(manager, p.inlet, p.outlet, connChan)
	go p.flow(manager, p.outlet, p.inlet, connChan)
}

// isAllowedReadError returns whether or not the passed error is allowed without
// close the pipe.
func (p *pipe) isAllowedIOError(err error) bool {
	if atomic.LoadInt32(&p.closed) != 0 {
		return false
	}

	if err == io.EOF {
		return false
	}
	if opErr, ok := err.(*net.OpError); ok && !opErr.Temporary() {
		return false
	}
	return true
}

// flow creates a one way flow between from and to.
func (p *pipe) flow(manager *addrmgr.AddrManager, from net.Conn, to net.Conn, connChan chan bool) {
	defer func() {
		connChan <- true
	}()

	buf := make([]byte, buffSize)
	idleTimer := time.NewTimer(pipeTimeout)
	defer idleTimer.Stop()

	ioFunc := func() error {
		if fc, ok := from.(*Conn); ok {
			if fc.buf != nil && len(fc.buf.Bytes()) != 0 {
				n, err := from.Read(buf)
				if err != nil {
					return err
				}
				_, err = to.Write(buf[:n])
				return err
			}
		}

		// Read message header
		var headerBytes [p2p.HeaderSize]byte
		if _, err := io.ReadFull(from, headerBytes[:]); err != nil {
			return err
		}

		// Deserialize message header
		var hdr p2p.Header
		if err := hdr.Deserialize(headerBytes[:]); err != nil {
			return err
		}

		// Read payload
		payload := make([]byte, hdr.Length)
		_, err := io.ReadFull(from, payload[:])
		if err != nil {
			return err
		}

		// Verify checksum
		if err = hdr.Verify(payload); err != nil {
			return err
		}

		if hdr.GetCMD() == p2p.CmdDAddr {
			message := msg.Daddr{}
			if err = message.Deserialize(bytes.NewBuffer(payload)); err != nil {
				return err
			}
			manager.AddAddress(from.(*Conn).PID(), &message)
		} else {
			hdrData := headerBytes[:]
			data := append(hdrData, payload...)
			_, err = to.Write(data[:])
		}

		return err
	}
	done := make(chan error)
out:
	for {
		go func() {
			done <- ioFunc()
		}()

		select {
		case err := <-done:
			if !p.isAllowedIOError(err) {
				break out
			}

			idleTimer.Reset(pipeTimeout)

		case <-idleTimer.C:
			log.Warnf("pipe no response for %s -- timeout", pipeTimeout)
			break out
		}
	}
	atomic.AddInt32(&p.closed, 1)
	_ = from.Close()
	_ = to.Close()
}

// state stores the current connect peers and local service index.
type state struct {
	peers         map[[16]byte]peer.PID
	inboundPipes  map[peer.PID]struct{}
	outboundPipes map[peer.PID]struct{}
	index         map[uint32]net.Addr
	lock          sync.RWMutex
}

// peerList represents the connect peers list.
type peerList []peer.PID

// inbound represents an inbound connection.
type inbound *Conn

// outbound represents an outbound connection.
type outbound *Conn

type Hub struct {
	magic             uint32
	pid               peer.PID
	admgr             *addrmgr.AddrManager
	queue             chan interface{}
	quit              chan struct{}
	addr              string
	pingNonce         func(pid peer.PID) uint64
	dposV2StartHeight uint32
}

// createPipe creates a pipe between inlet connection and the network address.
func createPipe(manager *addrmgr.AddrManager, inlet net.Conn, addr net.Addr, connChan chan bool) net.Conn {
	// Attempt to connect to target address.
	outlet, err := net.Dial(addr.Network(), addr.String())
	if err != nil {
		// If the outlet address can not be connected, close the inlet
		// connection to signal the pipe can not be created.
		_ = inlet.Close()
		connChan <- true
		return nil
	}

	// Creates a new pipe between connection and service address.
	p := pipe{inlet: inlet, outlet: outlet}
	p.start(manager, connChan)

	return outlet
}

// connHandler is the main handler of the hub implementation.
func (h *Hub) connHandler() {
	state := &state{
		peers:         make(map[[16]byte]peer.PID),
		inboundPipes:  make(map[peer.PID]struct{}),
		outboundPipes: make(map[peer.PID]struct{}),
		index:         make(map[uint32]net.Addr),
	}

out:
	for {
		select {
		case msg := <-h.queue:
			switch msg := msg.(type) {
			case peerList:
				// Update connect peers.
				h.handlePeers(state, msg)

			case outbound:
				// Register to the index and create a connection to target
				// arbiter.
				h.handleOutbound(state, msg)

			case inbound:
				// Dispatch the connection to local service according to the
				// magic.
				h.handleInbound(state, msg)
			}

		case <-h.quit:
			break out
		}
	}
}

func (h *Hub) handlePeers(state *state, peers []peer.PID) {
	// Convert origin peer list to map.
	newPeers := make(map[[16]byte]peer.PID)
	for _, pid := range peers {
		newPeers[PIDTo16(pid)] = pid
	}

	// Update the state peers.
	state.peers = newPeers
}

func (h *Hub) handleOutbound(state *state, conn *Conn) {
	// Refuse connection not in connect peers list.
	if _, ok := state.peers[PIDTo16(conn.PID())]; !ok {
		log.Debugf("%s not in peers list", peer.PID(conn.PID()))
		_ = conn.Close()
		return
	}

	// Refuse target not in connect peers list.
	target, ok := state.peers[conn.target]
	if !ok {
		log.Debugf("target %s not in peers list", target)
		_ = conn.Close()
		return
	}

	// Register local service to index.
	state.index[conn.Magic()] = conn.NetAddr()

	// Find the target address from addrmanger.
	addr := h.admgr.GetAddress(target)
	if addr == nil {
		log.Debugf("target %s address not found", target)
		_ = conn.Close()
		return
	}

	// Create the pipe between local service and target address.
	go func() {
		state.lock.Lock()
		state.outboundPipes[target] = struct{}{}
		state.lock.Unlock()
		connChan := make(chan bool)
		remoteConn := createPipe(h.admgr, conn, addr, connChan)

		// if only in outbound pipes, not in inbound pipes, need to announce addr
		if uint32(h.pingNonce(peer.PID{})) > h.dposV2StartHeight {
			go func() {
				time.Sleep(time.Second * 10)
				state.lock.RLock()
				if _, ok := state.inboundPipes[target]; !ok {
					err := h.announceDaddr(remoteConn, conn)
					if err != nil {
						state.lock.RUnlock()
						log.Debugf("service announce daddr error:", err)
						_ = conn.Close()
						return
					}
				}
				state.lock.RUnlock()
			}()
		} else {
			log.Info("Don't send daddr message in hub when height is not reach dposV2StartHeight")
		}

		<-connChan
		state.lock.Lock()
		delete(state.outboundPipes, target)
		state.lock.Unlock()
	}()
}

func (h *Hub) handleInbound(state *state, conn *Conn) {
	// Refuse connection not in connect peers list.
	if _, ok := state.peers[PIDTo16(conn.PID())]; !ok {
		log.Debugf("%x not in peers list", conn.PID())
		_ = conn.Close()
		return
	}

	// Find our service address from index.
	addr, ok := state.index[conn.Magic()]
	if !ok {
		log.Debugf("service magic %d not found", conn.Magic())
		_ = conn.Close()
		return
	}

	// Create the pipe between inbound connection and local service.
	go func() {
		state.lock.Lock()
		state.inboundPipes[conn.PID()] = struct{}{}
		state.lock.Unlock()
		connChan := make(chan bool)
		createPipe(h.admgr, conn, addr, connChan)
		<-connChan
		state.lock.Lock()
		delete(state.inboundPipes, conn.PID())
		state.lock.Unlock()
	}()
}

func (h *Hub) announceDaddr(remote net.Conn, current *Conn) error {
	// send our addr to conn
	msg := msg.NewDaddr(h.addr)

	buf := new(bytes.Buffer)
	if err := msg.Serialize(buf); err != nil {
		return fmt.Errorf("serialize message failed %s", err.Error())
	}
	payload := buf.Bytes()

	// Create message header
	hdr, err := p2p.BuildHeader(current.magic, msg.CMD(), payload).Serialize()
	if err != nil {
		return fmt.Errorf("serialize message header failed %s", err.Error())
	}

	// Set write deadline
	err = remote.SetWriteDeadline(time.Now().Add(p2p.WriteMessageTimeOut))
	if err != nil {
		return fmt.Errorf("set write deadline failed %s", err.Error())
	}

	// Write header
	if _, err = remote.Write(hdr); err != nil {
		return err
	}

	// Write payload
	_, err = remote.Write(payload)
	return err
}

// Intercept intercepts the accepted connection and distribute the connection to
// the right service, returns nil if the connection has been intercepted.
func (h *Hub) Intercept(conn net.Conn) net.Conn {
	c, err := WrapConn(conn)
	if err != nil {
		_ = conn.Close()
		return nil
	}

	// The connection from main chain arbiter, do not intercept.
	if h.magic == c.Magic() {
		return c
	}

	// The connection come from our own service.
	if h.pid.Equal(c.PID()) {
		h.queue <- outbound(c)
		return nil
	}

	// The connection come from other peers.
	h.queue <- inbound(c)
	return nil
}

// New creates a new Hub instance with the main network magic, arbiter PID and
// DPOS network AddrManager.
func New(magic uint32, pid [33]byte, admgr *addrmgr.AddrManager, addr string,
	pingNonce func(pid peer.PID) uint64, dposV2startHeight uint32) *Hub {
	h := Hub{
		magic:             magic,
		pid:               pid,
		admgr:             admgr,
		queue:             make(chan interface{}, 125),
		quit:              make(chan struct{}),
		addr:              addr,
		pingNonce:         pingNonce,
		dposV2StartHeight: dposV2startHeight,
	}

	// Start the hub.
	go h.connHandler()

	// Wait for stop signal.
	go func() {
		<-signal.NewInterrupt().C
		close(h.quit)
	}()

	// Subscribe peers changed event.
	events.Subscribe(func(e *events.Event) {
		switch e.Type {
		case events.ETDirectPeersChanged:
			peersInfo := e.Data.(*peer.PeersInfo)
			peers := peersInfo.CurrentPeers
			peers = append(peers, peersInfo.NextPeers...)
			peers = append(peers, peersInfo.CRPeers...)

			h.queue <- peerList(peers)
		}
	})

	return &h
}

// PIDTo16 converts a PID to [16]byte with the last 16 bytes of PID.
func PIDTo16(pid [33]byte) [16]byte {
	var key [16]byte
	copy(key[:], pid[17:])
	return key
}
