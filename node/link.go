package node

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"sync/atomic"
	"time"

	chain "github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/log"
	"github.com/elastos/Elastos.ELA/protocol"

	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/p2p"
	"github.com/elastos/Elastos.ELA.Utility/p2p/addrmgr"
	"github.com/elastos/Elastos.ELA.Utility/p2p/connmgr"
	"github.com/elastos/Elastos.ELA.Utility/p2p/msg"
)

const (
	// negotiateTimeout is the duration of negotiate before we time out a peer.
	negotiateTimeout = 30 * time.Second

	// idleTimeout is the duration of inactivity before we time out a peer.
	idleTimeout = 5 * time.Minute

	// defaultRetryDuration is the default duration of time for retrying
	// persistent connections.
	defaultRetryDuration = time.Second * 5

	// defaultTargetOutbound is the default number of outbound connections to
	// maintain.
	defaultTargetOutbound = uint32(8)
)

// Handler is the P2P message handler interface.
type Handler interface {
	MakeEmptyMessage(cmd string) (p2p.Message, error)
	HandleMessage(message p2p.Message)
}

type link struct {
	magic      uint32
	addr       string // The address of the node
	inbound    bool
	persistent bool
	sentAddrs  bool
	disconnect int32
	connReq    *connmgr.ConnReq
	conn       net.Conn // Connect socket with the peer node

	flagsMtx       sync.Mutex // protects the node flags below
	na             *p2p.NetAddress
	versionKnown   bool
	verAckReceived bool
	port           uint16 // The server port of the node

	handler        Handler
	negotiate      chan struct{} // Notify protocol negotiated.
	knownAddresses map[string]struct{}
	sendQueue      chan p2p.Message
	quit           chan struct{}
}

func (node *node) String() string {
	direction := "outbound"
	if node.inbound {
		direction = "inbound"
	}
	return fmt.Sprintf("%s (%s)", node.addr, direction)
}

func (node *node) start() {
	go node.inHandler()
	go node.outHandler()
	go node.waitProtocolNegotiate()
}

// waitProtocolNegotiate wait for protocol negotiate finish or timeout.
func (node *node) waitProtocolNegotiate() {
	select {
	case <-node.negotiate:
		// Set node state to ESTABLISHED
		node.SetState(protocol.ESTABLISHED)

		// Start ping handler
		go node.pingHandler()

	case <-time.After(negotiateTimeout):
		node.Disconnect()
	}
}

func loadCertificate() (tls.Certificate, *x509.CertPool, error) {
	// load cert
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return cert, nil, err
	}
	// load root ca
	pemCerts, err := ioutil.ReadFile(caPath)
	if err != nil {
		return cert, nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemCerts) {
		return cert, nil, errors.New("failed to parse root certificate")
	}

	return cert, pool, nil
}

func newNodePortListener() (listener net.Listener, err error) {
	if isTls {
		// load certificate
		cert, pool, err := loadCertificate()
		if err != nil {
			log.Error(err)
			return nil, err
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      pool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    pool,
		}

		listener, err = tls.Listen("tcp", fmt.Sprint(":", nodePort),
			tlsConfig)
		if err != nil {
			return nil, err
		}

	} else {

		listener, err = net.Listen("tcp", fmt.Sprint(":", nodePort))
		if err != nil {
			return nil, err
		}

	}

	return listener, nil
}

func dialTimeout(addr net.Addr) (net.Conn, error) {
	var err error
	var conn net.Conn
	if isTls {
		// load certificate
		cert, pool, err := loadCertificate()
		if err != nil {
			return nil, err
		}
		conf := &tls.Config{
			RootCAs:      pool,
			Certificates: []tls.Certificate{cert},
		}

		var dialer net.Dialer
		dialer.Timeout = defaultDialTimeout
		conn, err = tls.DialWithDialer(&dialer, addr.Network(), addr.String(), conf)
		if err != nil {
			return nil, err
		}

	} else {

		conn, err = net.DialTimeout(addr.Network(), addr.String(), defaultDialTimeout)
		if err != nil {
			return nil, err
		}

	}

	return conn, err
}

func (node *node) readMessage() (p2p.Message, error) {
	return p2p.ReadMessage(node.conn, node.magic, node.makeEmptyMessage)
}

// shouldHandleReadError returns whether or not the passed error, which is
// expected to have come from reading from the remote peer in the inHandler,
// should be logged and responded to with a reject message.
func (node *node) shouldHandleReadError(err error) bool {
	// No logging or reject message when the peer is being forcibly
	// disconnected.
	if atomic.LoadInt32(&node.disconnect) != 0 {
		return false
	}

	// No logging or reject message when the remote peer has been
	// disconnected.
	if err == io.EOF {
		return false
	}
	if opErr, ok := err.(*net.OpError); ok && !opErr.Temporary() {
		return false
	}

	return true
}

func (node *node) inHandler() {
	// The timer is stopped when a new message is received and reset after it
	// is processed.
	idleTimer := time.AfterFunc(idleTimeout, func() {
		log.Warnf("Peer %s no answer for %s -- disconnecting", node, idleTimeout)
		node.Disconnect()
	})
out:
	for atomic.LoadInt32(&node.disconnect) == 0 {
		// Read a message and stop the idle timer as soon as the read
		// is done.  The timer is reset below for the next iteration if
		// needed.
		rmsg, err := node.readMessage()
		idleTimer.Stop()
		if err != nil {
			// Do not handle invalid external messages.
			if err == ErrInvalidExternalMessage {
				continue
			}

			// Only log the error and send reject message if the
			// local peer is not forcibly disconnecting and the
			// remote peer has not disconnected.
			if node.shouldHandleReadError(err) {
				errMsg := fmt.Sprintf("Can't read message from %s: %v", node, err)
				if err != io.ErrUnexpectedEOF {
					log.Errorf(errMsg)
				}

				// Push a reject message for the malformed message and wait for
				// the message to be sent before disconnecting.
				//
				// NOTE: Ideally this would include the command in the header if
				// at least that much of the message was valid, but that is not
				// currently exposed by wire, so just used malformed for the
				// command.
				reject := msg.NewReject("malformed", msg.RejectMalformed, errMsg)
				node.SendMessage(reject)
			}
			break out
		}

		switch m := rmsg.(type) {
		case *msg.Version:
			node.onVersion(m)

		case *msg.VerAck:
			node.onVerAck(m)

		case *msg.GetAddr:
			node.onGetAddr(m)

		case *msg.Addr:
			node.onAddr(m)

		case *msg.Ping:
			node.onPing(m)

		case *msg.Pong:
			node.onPong(m)

		default:
			if node.handler != nil {
				node.handler.HandleMessage(rmsg)
			}
		}

		// A message was received so reset the idle timer.
		idleTimer.Reset(idleTimeout)
	}

	// Ensure the idle timer is stopped to avoid leaking the resource.
	idleTimer.Stop()

	// Ensure connection is closed.
	node.Disconnect()

	log.Debugf("Peer input handler done for %s", node)
}

func (node *node) outHandler() {
out:
	for {
		select {
		case smsg := <-node.sendQueue:
			err := p2p.WriteMessage(node.conn, node.magic, smsg)
			if err != nil {
				node.Disconnect()
				continue
			}

		case <-node.quit:
			break out
		}
	}

	// Drain any wait channels before going away so there is nothing left
	// waiting on this goroutine.
cleanup:
	for {
		select {
		case <-node.sendQueue:
		default:
			break cleanup
		}
	}
	log.Debugf("Peer output handler done for %s", node)
}

func (node *node) SendMessage(msg p2p.Message) {
	if atomic.LoadInt32(&node.disconnect) != 0 {
		return
	}
	node.sendQueue <- msg
}

func (node *node) Connected() bool {
	return atomic.LoadInt32(&node.disconnect) == 0
}

func (node *node) Disconnect() {
	if atomic.AddInt32(&node.disconnect, 1) != 1 {
		return
	}
	node.SetState(protocol.INACTIVITY)

	log.Debugf("Disconnecting %s", node)
	node.conn.Close()
	close(node.quit)
}

// After message header decoded, this method will be
// called to create the message instance with the CMD
// which is the message type of the received message
func (node *node) makeEmptyMessage(cmd string) (m p2p.Message, err error) {
	if err = node.FilterMessage(cmd); err != nil {
		return nil, err
	}

	switch cmd {
	case p2p.CmdVersion:
		m = &msg.Version{}

	case p2p.CmdVerAck:
		m = &msg.VerAck{}

	case p2p.CmdGetAddr:
		m = &msg.GetAddr{}

	case p2p.CmdAddr:
		m = &msg.Addr{}

	case p2p.CmdPing:
		m = &msg.Ping{}

	case p2p.CmdPong:
		m = &msg.Pong{}

	default:
		if node.handler != nil {
			return node.handler.MakeEmptyMessage(cmd)
		}
	}

	return m, err
}

// addKnownAddresses adds the given addresses to the set of known addresses to
// the peer to prevent sending duplicate addresses.
func (node *node) addKnownAddresses(addresses []*p2p.NetAddress) {
	for _, na := range addresses {
		node.knownAddresses[addrmgr.NetAddressKey(na)] = struct{}{}
	}
}

// addressKnown true if the given address is already known to the peer.
func (node *node) addressKnown(na *p2p.NetAddress) bool {
	_, exists := node.knownAddresses[addrmgr.NetAddressKey(na)]
	return exists
}

// pushAddrMsg sends an addr message to the connected peer using the provided
// addresses.
func (node *node) pushAddrMsg(addresses []*p2p.NetAddress) {
	// Filter addresses already known to the peer.
	addrs := make([]*p2p.NetAddress, 0, len(addresses))
	for _, addr := range addresses {
		if !node.addressKnown(addr) {
			addrs = append(addrs, addr)
		}
	}
	known, err := node.PushAddrMsg(addrs)
	if err != nil {
		log.Errorf("Can't push address message to %s: %v", node, err)
		node.Disconnect()
		return
	}
	node.addKnownAddresses(known)
}

func (node *node) onVersion(v *msg.Version) {
	// Exclude the node itself
	if v.Nonce == LocalNode.ID() {
		log.Warn("The node handshake with itself")
		node.Disconnect()
		return
	}

	switch node.State() {
	case protocol.INIT, protocol.HAND:
	default:
		log.Warnf("invalid state %s to receive version", node.State())
		node.Disconnect()
		return
	}

	node.flagsMtx.Lock()
	node.timestamp = time.Unix(int64(v.TimeStamp), 0)
	node.id = v.Nonce
	node.version = v.Version
	node.services = v.Services
	node.port = v.Port
	node.relay = v.Relay
	node.height = uint64(v.Height)
	node.versionKnown = true
	node.flagsMtx.Unlock()

	// Update message handler according to the protocol version
	if v.Version < p2p.EIP001Version {
		node.handler = NewHandlerV0(node)
	} else {
		node.handler = NewHandlerEIP001(node)
	}

	switch node.State() {
	case protocol.INIT:

		node.SetState(protocol.HANDSHAKE)
		version := NewVersion(LocalNode)
		// External node connect with open port
		if node.IsExternal() {
			version.Port = openPort
		} else {
			version.Port = nodePort
		}
		node.SendMessage(version)

	case protocol.HAND:

		node.SetState(protocol.HANDSHAKED)
		node.SendMessage(&msg.VerAck{})
	}

	// Outbound connections.
	if !node.Inbound() {
		// Get address that best matches.
		lna := addrManager.GetBestLocalAddress(node.NA())
		if addrmgr.IsRoutable(lna) {
			// Filter addresses the peer already knows about.
			addresses := []*p2p.NetAddress{lna}
			node.pushAddrMsg(addresses)
		}

		// Request known addresses if the server address manager needs more.
		if addrManager.NeedMoreAddresses() {
			node.SendMessage(new(msg.GetAddr))
		}

		// Mark the address as a known good address.
		addrManager.Good(node.NA())
	}

	// Add node to neighbor list
	AddNode(node)
}

func (node *node) onVerAck(verAck *msg.VerAck) {
	switch node.State() {
	case protocol.HANDSHAKE:
		node.SendMessage(verAck)

	case protocol.HANDSHAKED:

	default:
		log.Warn("invalid state %s to received verack", node.State())
		node.Disconnect()
		return
	}

	node.flagsMtx.Lock()
	node.verAckReceived = true
	node.flagsMtx.Unlock()

	close(node.negotiate)
}

func (node *node) onPing(ping *msg.Ping) {
	node.SetHeight(ping.Nonce)
	node.SendMessage(msg.NewPong(uint64(chain.DefaultLedger.Store.GetHeight())))
}

func (node *node) onPong(pong *msg.Pong) {
	node.SetHeight(pong.Nonce)
}

func (node *node) onGetAddr(getAddr *msg.GetAddr) {
	// Do not accept getaddr requests from outbound peers.  This reduces
	// fingerprinting attacks.
	if !node.Inbound() {
		log.Debugf("Ignoring getaddr request from outbound peer ",
			"%v", node)
		return
	}

	// Only allow one getaddr request per connection to discourage
	// address stamping of inv announcements.
	if node.sentAddrs {
		log.Debugf("Ignoring repeated getaddr request from peer ",
			"%v", node)
		return
	}
	node.sentAddrs = true

	// Get the current known addresses from the address manager.
	addrCache := addrManager.AddressCache()

	// Push the addresses.
	node.pushAddrMsg(addrCache)
}

func (node *node) onAddr(msg *msg.Addr) {
	// A message that has no addresses is invalid.
	if len(msg.AddrList) == 0 {
		log.Errorf("Command [%s] from %s does not contain any addresses", msg.CMD(), node)
		node.Disconnect()
		return
	}

	// A addr message from external node is will be ignored.
	if node.IsExternal() {
		log.Debugf("ignore [%s] from %s (external) node", msg.CMD(), node)
		return
	}

	for _, na := range msg.AddrList {
		// Don't add more address if we're disconnecting.
		if !node.Connected() {
			return
		}

		// Set the timestamp to 5 days ago if it's more than 24 hours
		// in the future so this address is one of the first to be
		// removed when space is needed.
		now := time.Now()
		if na.Timestamp.After(now.Add(time.Minute * 10)) {
			na.Timestamp = now.Add(-1 * time.Hour * 24 * 5)
		}

		// Add address to known addresses for this peer.
		node.addKnownAddresses([]*p2p.NetAddress{na})
	}

	// Add addresses to server address manager.  The address manager handles
	// the details of things such as preventing duplicate addresses, max
	// addresses, and last seen updates.
	// XXX gives a 2 hour time penalty here, do we want to do the
	// same?
	addrManager.AddAddresses(msg.AddrList, node.NA())
}

func NewVersion(node protocol.Noder) *msg.Version {
	return &msg.Version{
		Version:   node.Version(),
		Services:  node.Services(),
		TimeStamp: uint32(time.Now().Unix()),
		Port:      node.Port(),
		Nonce:     node.ID(),
		Height:    uint64(chain.DefaultLedger.GetLocalBlockChainHeight()),
		Relay:     node.IsRelay(),
	}
}

func SendGetBlocks(node protocol.Noder, locator []*common.Uint256, hashStop common.Uint256) {
	if LocalNode.GetStartHash() == *locator[0] && LocalNode.GetStopHash() == hashStop {
		return
	}

	LocalNode.SetStartHash(*locator[0])
	LocalNode.SetStopHash(hashStop)
	node.SendMessage(msg.NewGetBlocks(locator, hashStop))
}

func GetBlockHashes(startHash common.Uint256, stopHash common.Uint256, maxBlockHashes uint32) ([]*common.Uint256, error) {
	var count = uint32(0)
	var startHeight uint32
	var stopHeight uint32
	curHeight := chain.DefaultLedger.Store.GetHeight()
	if stopHash == common.EmptyHash {
		if startHash == common.EmptyHash {
			if curHeight > maxBlockHashes {
				count = maxBlockHashes
			} else {
				count = curHeight
			}
		} else {
			startHeader, err := chain.DefaultLedger.Store.GetHeader(startHash)
			if err != nil {
				return nil, err
			}
			startHeight = startHeader.Height
			count = curHeight - startHeight
			if count > maxBlockHashes {
				count = maxBlockHashes
			}
		}
	} else {
		stopHeader, err := chain.DefaultLedger.Store.GetHeader(stopHash)
		if err != nil {
			return nil, err
		}
		stopHeight = stopHeader.Height
		if startHash != common.EmptyHash {
			startHeader, err := chain.DefaultLedger.Store.GetHeader(startHash)
			if err != nil {
				return nil, err
			}
			startHeight = startHeader.Height

			// avoid unsigned integer underflow
			if stopHeight < startHeight {
				return nil, errors.New("do not have header to send")
			}
			count = stopHeight - startHeight

			if count >= maxBlockHashes {
				count = maxBlockHashes
			}
		} else {
			if stopHeight > maxBlockHashes {
				count = maxBlockHashes
			} else {
				count = stopHeight
			}
		}
	}

	hashes := make([]*common.Uint256, 0)
	for i := uint32(1); i <= count; i++ {
		hash, err := chain.DefaultLedger.Store.GetBlockHash(startHeight + i)
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, &hash)
	}

	return hashes, nil
}
