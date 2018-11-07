package node

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	chain "github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/bloom"
	"github.com/elastos/Elastos.ELA/config"
	. "github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/log"
	"github.com/elastos/Elastos.ELA/protocol"

	. "github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/p2p"
	"github.com/elastos/Elastos.ELA.Utility/p2p/addrmgr"
	"github.com/elastos/Elastos.ELA.Utility/p2p/connmgr"
	"github.com/elastos/Elastos.ELA.Utility/p2p/msg"
)

const (
	// defaultDialTimeout is the time limit to finish dialing to an address.
	defaultDialTimeout = 10 * time.Second

	// stateMonitorInterval is the interval to monitor connection and syncing state.
	stateMonitorInterval = 10 * time.Second

	// pingInterval is the interval of time to wait in between sending ping
	// messages.
	pingInterval = 30 * time.Second

	// syncBlockTimeout is the time limit to trigger restart sync block.
	syncBlockTimeout = 30 * time.Second
)

var (
	LocalNode *node

	cfg = config.Parameters.Configuration

	services = protocol.OpenService
	nodePort = cfg.NodePort
	openPort = cfg.NodeOpenPort
	isTls    = cfg.IsTLS
	certPath = cfg.CertPath
	keyPath  = cfg.KeyPath
	caPath   = cfg.CAPath

	addrManager = addrmgr.New("data")
	connManager *connmgr.ConnManager
)

type Semaphore chan struct{}

func MakeSemaphore(n int) Semaphore {
	return make(chan struct{}, n)
}

func (s Semaphore) acquire() { s <- struct{}{} }
func (s Semaphore) release() { <-s }

// newNetAddress attempts to extract the IP address and port from the passed
// net.Addr interface and create a NetAddress structure using that information.
func newNetAddress(addr net.Addr, services uint64) (*p2p.NetAddress, error) {
	// addr will be a net.TCPAddr when not using a proxy.
	if tcpAddr, ok := addr.(*net.TCPAddr); ok {
		ip := tcpAddr.IP
		port := uint16(tcpAddr.Port)
		na := p2p.NewNetAddressIPPort(ip, port, services)
		return na, nil
	}

	// For the most part, addr should be one of the two above cases, but
	// to be safe, fall back to trying to parse the information from the
	// address string as a last resort.
	host, portStr, err := net.SplitHostPort(addr.String())
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(host)
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, err
	}
	na := p2p.NewNetAddressIPPort(ip, uint16(port), services)
	return na, nil
}

type node struct {
	//sync.RWMutex	//The Lock not be used as expected to use function channel instead of lock
	state        int32         // node state
	timestamp    time.Time     // The timestamp of node
	id           uint64        // The nodes's id
	version      uint32        // The network protocol the node used
	services     uint64        // The services the node supplied
	relay        bool          // The relay capability of the node (merge into capbility flag)
	height       uint64        // The node latest block height
	external     bool          // Indicate if this is an external node
	txnCnt       uint64        // The transactions be transmit by this node
	rxTxnCnt     uint64        // The transaction received by this node
	link                       // The link status and infomation
	chain.TxPool               // Unconfirmed transaction pool
	idCache                    // The buffer to store the id of the items which already be processed
	filter       *bloom.Filter // The bloom filter of a spv node
	/*
	 * |--|--|--|--|--|--|isSyncFailed|isSyncHeaders|
	 */
	syncFlag           uint8
	flagLock           sync.RWMutex
	cachelock          sync.RWMutex
	requestedBlockLock sync.RWMutex
	DefaultMaxPeers    uint
	headerFirstMode    bool
	RequestedBlockList map[Uint256]time.Time
	stallTimer         *stallTimer
	SyncBlkReqSem      Semaphore
	StartHash          Uint256
	StopHash           Uint256
}

// newNodeBase returns a new base peer based on the inbound flag.  This
// is used by the NewInboundPeer and NewOutboundPeer functions to perform base
// setup needed by both types of peers.
func newNodeBase(conn net.Conn, inbound, persistent bool) *node {
	n := node{
		link: link{
			magic:          cfg.Magic,
			conn:           conn,
			inbound:        inbound,
			persistent:     persistent,
			negotiate:      make(chan struct{}),
			knownAddresses: make(map[string]struct{}),
			sendQueue:      make(chan p2p.Message, 1),
			quit:           make(chan struct{}),
		},
		filter: bloom.LoadFilter(nil),
	}
	n.start()

	return &n
}

// NewInboundNode returns a new inbound peer. Use Start to begin
// processing incoming and outgoing messages.
func NewInboundNode(conn net.Conn) *node {
	n := newNodeBase(conn, true, false)
	n.addr = conn.RemoteAddr().String()
	// Set up a NetAddress for the peer to be used with AddrManager.  We
	// only do this inbound because outbound set this up at connection time
	// and no point recomputing.
	na, err := newNetAddress(conn.RemoteAddr(), services)
	if err != nil {
		log.Errorf("Cannot create remote net address: %v", err)
		n.Disconnect()
		return nil
	}
	// Mark node from open port as external.
	la, err := newNetAddress(conn.LocalAddr(), services)
	if err != nil {
		log.Errorf("Cannot parse local net address: %v", err)
		n.Disconnect()
		return nil
	}
	if la.Port == openPort {
		n.external = true
	}
	n.na = na

	return n
}

// NewOutboundPeer returns a new outbound peer.
func NewOutboundPeer(conn net.Conn, addr string, persistent bool) (*node, error) {
	p := newNodeBase(conn, false, persistent)
	p.addr = addr

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, err
	}

	na, err := addrManager.HostToNetAddress(host, uint16(port), services)
	if err != nil {
		return nil, err
	}
	p.na = na

	return p, nil
}

func Start() (protocol.Noder, error) {
	LocalNode = &node{
		id:                 rand.New(rand.NewSource(time.Now().Unix())).Uint64(),
		version:            protocol.ProtocolVersion,
		services:           services,
		relay:              true,
		SyncBlkReqSem:      MakeSemaphore(protocol.MaxSyncHdrReq),
		RequestedBlockList: make(map[Uint256]time.Time),
		stallTimer:         newSyncTimer(stopSyncing),
		height:             uint64(chain.DefaultLedger.Blockchain.GetBestHeight()),
		link: link{
			magic: cfg.Magic,
			port:  cfg.NodePort,
		},
	}

	if !cfg.OpenService {
		LocalNode.services &^= protocol.OpenService
	}

	LocalNode.TxPool.Init()
	LocalNode.idCache.init()

	listeners := make([]net.Listener, 0, 2)
	listener, err := newNodePortListener()
	if err != nil {
		return nil, err
	}
	listeners = append(listeners, listener)

	// Listen open port if OpenService enabled
	if cfg.OpenService {
		listener, err := newOpenPortListener()
		if err != nil {
			return nil, err
		}
		listeners = append(listeners, listener)
	}

	// Setup a function to return new addresses to connect to.
	var newAddressFunc = func() (net.Addr, error) {
		for tries := 0; tries < 100; tries++ {
			addr := addrManager.GetAddress()
			if addr == nil {
				break
			}

			// Address will not be invalid, local or unroutable
			// because addrmanager rejects those on addition.
			// Just check that we don't already have an address
			// in the same group so that we are not connecting
			// to the same network segment at the expense of
			// others.
			key := addrmgr.GroupKey(addr.NetAddress())
			if OutboundGroupCount(key) != 0 {
				continue
			}

			// only allow recent nodes (10mins) after we failed 30 times
			if tries < 30 && time.Since(addr.LastAttempt()) < 10*time.Minute {
				continue
			}

			// allow nondefault ports after 50 failed tries.
			if tries < 50 && addr.NetAddress().Port != nodePort {
				continue
			}

			addrString := addrmgr.NetAddressKey(addr.NetAddress())
			return addrStringToNetAddr(addrString)
		}

		return nil, errors.New("no valid connect address")
	}

	cmgr, err := connmgr.New(&connmgr.Config{
		Listeners:      listeners,
		OnAccept:       inboundNodeConnected,
		RetryDuration:  defaultRetryDuration,
		TargetOutbound: defaultTargetOutbound,
		Dial:           dialTimeout,
		OnConnection:   outboundNodeConnected,
		GetNewAddress:  newAddressFunc,
	})
	if err != nil {
		return nil, err
	}
	connManager = cmgr

	go nodeHandler()

	// Startup persistent peers.
	for _, addr := range cfg.SeedList {
		netAddr, err := addrStringToNetAddr(addr)
		if err != nil {
			return nil, err
		}

		go connManager.Connect(&connmgr.ConnReq{
			Addr:      netAddr,
			Permanent: true,
		})
	}

	go func() {
		ticker := time.NewTicker(stateMonitorInterval)
		for {
			go LocalNode.SyncBlocks()
			<-ticker.C
		}
	}()

	go monitorNodeState()

	return LocalNode, nil
}

func Stop() {
	close(quit)
}

// addrStringToNetAddr takes an address in the form of 'host:port' and returns
// a net.Addr which maps to the original address with any host names resolved
// to IP addresses.  It also handles tor addresses properly by returning a
// net.Addr that encapsulates the address.
func addrStringToNetAddr(addr string) (net.Addr, error) {
	host, strPort, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	port, err := strconv.Atoi(strPort)
	if err != nil {
		return nil, err
	}

	// Skip if host is already an IP address.
	if ip := net.ParseIP(host); ip != nil {
		return &net.TCPAddr{
			IP:   ip,
			Port: port,
		}, nil
	}

	// Attempt to look up an IP address associated with the parsed host.
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses found for %s", host)
	}

	return &net.TCPAddr{
		IP:   ips[0],
		Port: port,
	}, nil
}

// ID returns the peer id.
//
// This function is safe for concurrent access.
func (node *node) ID() uint64 {
	node.flagsMtx.Lock()
	id := node.id
	node.flagsMtx.Unlock()

	return id
}

// NA returns the peer network address.
//
// This function is safe for concurrent access.
func (node *node) NA() *p2p.NetAddress {
	node.flagsMtx.Lock()
	na := node.na
	node.flagsMtx.Unlock()

	return na
}

// Addr returns the peer address.
//
// This function is safe for concurrent access.
func (node *node) Addr() string {
	// The address doesn't change after initialization, therefore it is not
	// protected by a mutex.
	return node.addr
}

// Inbound returns whether the peer is inbound.
//
// This function is safe for concurrent access.
func (node *node) Inbound() bool {
	return node.inbound
}

// VersionKnown returns the whether or not the version of a peer is known
// locally.
//
// This function is safe for concurrent access.
func (node *node) VersionKnown() bool {
	node.flagsMtx.Lock()
	versionKnown := node.versionKnown
	node.flagsMtx.Unlock()

	return versionKnown
}

// VerAckReceived returns whether or not a verack message was received by the
// peer.
//
// This function is safe for concurrent access.
func (node *node) VerAckReceived() bool {
	node.flagsMtx.Lock()
	verAckReceived := node.verAckReceived
	node.flagsMtx.Unlock()

	return verAckReceived
}

func (node *node) State() protocol.State {
	return protocol.State(atomic.LoadInt32(&node.state))
}

func (node *node) SetState(state protocol.State) {
	atomic.StoreInt32(&node.state, int32(state))
}

func (node *node) TimeStamp() time.Time {
	return node.timestamp
}

func (node *node) GetConn() net.Conn {
	return node.conn
}

func (node *node) Port() uint16 {
	return node.port
}

func (node *node) IsExternal() bool {
	return node.external
}

func (node *node) IsRelay() bool {
	return node.relay
}

func (node *node) Version() uint32 {
	return node.version
}

func (node *node) Services() uint64 {
	return node.services
}

func (node *node) IncRxTxnCnt() {
	node.rxTxnCnt++
}

func (node *node) GetTxnCnt() uint64 {
	return node.txnCnt
}

func (node *node) GetRxTxnCnt() uint64 {
	return node.rxTxnCnt
}

func (node *node) Height() uint64 {
	return node.height
}

func (node *node) SetHeight(height uint64) {
	node.height = height
}

func WaitForSyncFinish() {
	for len(cfg.SeedList) > 0 && !IsCurrent() {
		time.Sleep(5 * time.Second)
	}
}

func (node *node) LoadFilter(filter *msg.FilterLoad) {
	node.filter.Reload(filter)
}

func (node *node) BloomFilter() *bloom.Filter {
	return node.filter
}

func (node *node) Relay(from protocol.Noder, message interface{}) error {
	log.Debug()
	if from != nil && LocalNode.IsSyncHeaders() {
		return nil
	}

	for _, nbr := range GetNeighborNodes() {
		if from == nil || nbr.ID() != from.ID() {

			switch message := message.(type) {
			case *Transaction:
				if nbr.BloomFilter().IsLoaded() && nbr.BloomFilter().MatchTxAndUpdate(message) {
					inv := msg.NewInventory()
					txID := message.Hash()
					inv.AddInvVect(msg.NewInvVect(msg.InvTypeTx, &txID))
					nbr.SendMessage(inv)
					continue
				}

				if nbr.IsRelay() {
					nbr.SendMessage(msg.NewTx(message))
					node.txnCnt++
				}
			case *Block:
				if nbr.BloomFilter().IsLoaded() {
					inv := msg.NewInventory()
					blockHash := message.Hash()
					inv.AddInvVect(msg.NewInvVect(msg.InvTypeBlock, &blockHash))
					nbr.SendMessage(inv)
					continue
				}

				if nbr.IsRelay() {
					nbr.SendMessage(msg.NewBlock(message))
				}
			default:
				log.Warn("unknown relay message type")
				return errors.New("unknown relay message type")
			}
		}
	}

	return nil
}

func (node *node) IsSyncHeaders() bool {
	node.flagLock.RLock()
	defer node.flagLock.RUnlock()
	if (node.syncFlag & 0x01) == 0x01 {
		return true
	} else {
		return false
	}
}

func (node *node) SetSyncHeaders(b bool) {
	node.flagLock.Lock()
	defer node.flagLock.Unlock()
	if b == true {
		node.syncFlag = node.syncFlag | 0x01
	} else {
		node.syncFlag = node.syncFlag & 0xFE
	}
}

// PushAddrMsg sends an addr message to the connected peer using the provided
// addresses.  This function is useful over manually sending the message via
// QueueMessage since it automatically limits the addresses to the maximum
// number allowed by the message and randomizes the chosen addresses when there
// are too many.  It returns the addresses that were actually sent and no
// message will be sent if there are no entries in the provided addresses slice.
//
// This function is safe for concurrent access.
func (node *node) PushAddrMsg(addresses []*p2p.NetAddress) ([]*p2p.NetAddress, error) {
	addressCount := len(addresses)

	// Nothing to send.
	if addressCount == 0 {
		return nil, nil
	}

	addr := msg.NewAddr(addresses)

	// Randomize the addresses sent if there are more than the maximum allowed.
	if addressCount > msg.MaxAddrPerMsg {
		// Shuffle the address list.
		for i := 0; i < msg.MaxAddrPerMsg; i++ {
			j := i + rand.Intn(addressCount-i)
			addr.AddrList[i], addr.AddrList[j] = addr.AddrList[j], addr.AddrList[i]
		}

		// Truncate it to the maximum size.
		addr.AddrList = addr.AddrList[:msg.MaxAddrPerMsg]
	}

	node.SendMessage(addr)
	return addr.AddrList, nil
}

func IsCurrent() bool {
	current := true
	blockHeight := chain.DefaultLedger.Blockchain.BlockHeight
	nodes := GetNeighborNodes()
	internal := make([]protocol.Noder, 0, len(nodes))
	external := make([]protocol.Noder, 0, len(nodes))
	for _, node := range nodes {
		if node.IsExternal() {
			external = append(external, node)
		} else {
			internal = append(internal, node)
			if node.Height() > uint64(blockHeight) {
				current = false
			}
		}
	}

	printNodes(fmt.Sprintf("internal nbr(%d) --> %d", len(internal),
		blockHeight), internal)
	printNodes(fmt.Sprintf("external nbr(%d) -->", len(external)), external)
	return current
}

func printNodes(prefix string, nodes []protocol.Noder) {
	if len(nodes) == 0 {
		return
	}
	buf := bytes.NewBufferString(prefix)
	// Left start
	buf.WriteString(" [")
	// Append node height and address.
	for _, node := range nodes {
		buf.WriteString(log.Color(log.Green, strconv.FormatUint(node.Height(), 10)))
		buf.WriteString(" ")
		buf.WriteString(node.String())
		buf.WriteString(", ")
	}
	// Remove last ","
	buf.Truncate(buf.Len() - 2)
	// Right end
	buf.WriteString("]")
	log.Info(buf.String())
}

func (node *node) GetRequestBlockList() map[Uint256]time.Time {
	return node.RequestedBlockList
}

func (node *node) IsRequestedBlock(hash Uint256) bool {
	node.requestedBlockLock.Lock()
	defer node.requestedBlockLock.Unlock()
	_, ok := node.RequestedBlockList[hash]
	return ok
}

func (node *node) AddRequestedBlock(hash Uint256) {
	node.requestedBlockLock.Lock()
	defer node.requestedBlockLock.Unlock()
	node.RequestedBlockList[hash] = time.Now()
}

func (node *node) ResetRequestedBlock() {
	node.requestedBlockLock.Lock()
	defer node.requestedBlockLock.Unlock()

	node.RequestedBlockList = make(map[Uint256]time.Time)
}

func (node *node) DeleteRequestedBlock(hash Uint256) {
	node.requestedBlockLock.Lock()
	defer node.requestedBlockLock.Unlock()
	_, ok := node.RequestedBlockList[hash]
	if ok == false {
		return
	}
	delete(node.RequestedBlockList, hash)
}

func (node *node) AcqSyncBlkReqSem() {
	node.SyncBlkReqSem.acquire()
}

func (node *node) RelSyncBlkReqSem() {
	node.SyncBlkReqSem.release()
}

func (node *node) SetStartHash(hash Uint256) {
	node.StartHash = hash
}

func (node *node) GetStartHash() Uint256 {
	return node.StartHash
}

func (node *node) SetStopHash(hash Uint256) {
	node.StopHash = hash
}

func (node *node) GetStopHash() Uint256 {
	return node.StopHash
}
