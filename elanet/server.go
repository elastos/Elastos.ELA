// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package elanet

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/dpos/state"
	"github.com/elastos/Elastos.ELA/elanet/bloom"
	"github.com/elastos/Elastos.ELA/elanet/filter"
	"github.com/elastos/Elastos.ELA/elanet/filter/customidfilter"
	"github.com/elastos/Elastos.ELA/elanet/filter/nextturndposfilter"
	"github.com/elastos/Elastos.ELA/elanet/filter/returnsidechaindepositcoinfilter"
	"github.com/elastos/Elastos.ELA/elanet/filter/sidefilter"
	"github.com/elastos/Elastos.ELA/elanet/filter/upgradefilter"
	"github.com/elastos/Elastos.ELA/elanet/netsync"
	"github.com/elastos/Elastos.ELA/elanet/pact"
	"github.com/elastos/Elastos.ELA/elanet/peer"
	"github.com/elastos/Elastos.ELA/elanet/routes"
	"github.com/elastos/Elastos.ELA/mempool"
	"github.com/elastos/Elastos.ELA/p2p"
	"github.com/elastos/Elastos.ELA/p2p/msg"
	peer2 "github.com/elastos/Elastos.ELA/p2p/peer"
	svr "github.com/elastos/Elastos.ELA/p2p/server"
)

const (
	// defaultServices describes the default services that are supported by
	// the NetServer.
	defaultServices = pact.SFNodeNetwork | pact.SFTxFiltering | pact.SFNodeBloom

	// maxNonNodePeers defines the maximum count of accepting non-node peers.
	maxNonNodePeers = 100
)

// naFilter defines a network address filter for the main chain NetServer, for now
// it is used to filter SPV wallet addresses from relaying to other peers.
type naFilter struct{}

func (f *naFilter) Filter(na *p2p.NetAddress) bool {
	service := pact.ServiceFlag(na.Services)
	return service&pact.SFNodeNetwork == pact.SFNodeNetwork
}

// NewPeerMsg represent a new connected peer.
type NewPeerMsg struct {
	svr.IPeer
	Reply chan bool
}

// DonePeerMsg represent a disconnected peer.
type DonePeerMsg struct {
	svr.IPeer
	Reply chan struct{}
}

// relayMsg packages an inventory vector along with the newly discovered
// inventory so the relay has access to that information.
type relayMsg struct {
	invVect *msg.InvVect
	data    interface{}
}

// NetServer provides a NetServer for handling communications to and from peers.
type NetServer struct {
	svr.IServer
	SyncManager  *netsync.SyncManager
	chain        *blockchain.BlockChain
	ChainParams  *config.Configuration
	txMemPool    *mempool.TxPool
	blockMemPool *mempool.BlockPool
	Routes       *routes.Routes

	nonNodePeers int32 // This variable must be use atomically.
	peerQueue    chan interface{}
	relayInv     chan relayMsg
	quit         chan struct{}
	services     pact.ServiceFlag
}

// ServerPeer extends the peer to maintain state shared by the NetServer and
// the blockmanager.
type ServerPeer struct {
	*peer.Peer

	server        *NetServer
	continueHash  *common.Uint256
	isWhitelisted bool
	filter        *filter.Filter
	quit          chan struct{}
	// The following chans are used to sync blockmanager and NetServer.
	txProcessed    chan struct{}
	blockProcessed chan struct{}
}

// newServerPeer returns a new ServerPeer instance. The peer needs to be set by
// the caller.
func newServerPeer(s *NetServer) *ServerPeer {
	filter := filter.New(func(typ uint8) filter.TxFilter {
		switch typ {
		case filter.FTBloom:
			return bloom.NewTxFilter()
		case filter.FTDPOS:
			return sidefilter.New(s.chain.GetState())
		case filter.FTNexTTurnDPOSInfo:
			return nextturndposfilter.New()
		case filter.FTCustomID:
			return customidfilter.New()
		case filter.FTUpgrade:
			return upgradefilter.New()
		case filter.FTReturnSidechainDepositCoinFilter:
			return returnsidechaindepositcoinfilter.New()
		}
		return nil
	})

	return &ServerPeer{
		server:         s,
		filter:         filter,
		quit:           make(chan struct{}),
		txProcessed:    make(chan struct{}, 1),
		blockProcessed: make(chan struct{}, 1),
	}
}

// handleDisconnect handles peer disconnects and remove the peer from
// SyncManager and Routes.
func (sp *ServerPeer) handleDisconnect() {
	sp.WaitForDisconnect()
	sp.server.Routes.DonePeer(sp.Peer)
	sp.server.SyncManager.DonePeer(sp.Peer)

	// Decrease non node count.
	if !nodeFlag(uint64(sp.Services())) {
		atomic.AddInt32(&sp.server.nonNodePeers, -1)
	}
}

// OnVersion is invoked when a peer receives a version message and is
// used to negotiate the protocol version details as well as kick start
// the communications.
func (sp *ServerPeer) OnVersion(_ *peer.Peer, m *msg.Version) {

	// Disconnect full node peers that do not support DPOS protocol.
	if nodeFlag(m.Services) && sp.ProtocolVersion() < pact.DPOSStartVersion {
		sp.Disconnect()
		return
	}

	if !nodeFlag(m.Services) {
		// Disconnect non node peers if limitation arrived.
		if atomic.LoadInt32(&sp.server.nonNodePeers) >= maxNonNodePeers {
			sp.Disconnect()
			return
		}

		// Increase non node peers count.
		atomic.AddInt32(&sp.server.nonNodePeers, 1)
	}

	// Add the remote peer time as a sample for creating an offset against
	// the local clock to keep the network time in sync.
	sp.server.chain.TimeSource.AddTimeSample(sp.Addr(), m.Timestamp)

	// Signal the Routes this peer is a new sync candidate.
	sp.server.Routes.NewPeer(sp.Peer)

	// Signal the sync manager this peer is a new sync candidate.
	sp.server.SyncManager.NewPeer(sp.Peer)

	// Choose whether or not to relay transactions before a filter command
	// is received.
	sp.SetDisableRelayTx(!m.Relay)

	// Handle peer disconnect.
	go sp.handleDisconnect()
}

// OnMemPool is invoked when a peer receives a mempool message.
// It creates and sends an inventory message with the contents of the memory
// pool up to the maximum inventory allowed per message.  When the peer has a
// bloom filter loaded, the contents are filtered accordingly.
func (sp *ServerPeer) OnMemPool(_ *peer.Peer, _ *msg.MemPool) {
	// Only allow mempool requests if the NetServer has bloom filtering
	// enabled.
	if sp.server.services&pact.SFTxFiltering != pact.SFTxFiltering {
		log.Debugf("peer %v sent mempool request with bloom "+
			"filtering disabled -- disconnecting", sp)
		sp.Disconnect()
		return
	}

	// A decaying ban score increase is applied to prevent flooding.
	// The ban score accumulates and passes the ban threshold if a burst of
	// mempool messages comes from a peer. The score decays each minute to
	// half of its value.
	sp.AddBanScore(0, 33, "mempool")

	// Generate inventory message with the available transactions in the
	// transaction memory pool.  Limit it to the max allowed inventory
	// per message.  The NewMsgInvSizeHint function automatically limits
	// the passed hint to the maximum allowed, so it's safe to pass it
	// without double checking it here.
	txs := sp.server.txMemPool.GetTxsInPool()
	invMsg := msg.NewInvSize(uint(len(txs)))

	for _, tx := range txs {
		// Either add all transactions when there is no bloom filter,
		// or only the transactions that match the filter when there is
		// one.
		txId := tx.Hash()
		if !sp.filter.IsLoaded() || sp.filter.MatchUnconfirmed(tx) {
			iv := msg.NewInvVect(msg.InvTypeTx, &txId)
			invMsg.AddInvVect(iv)
			if len(invMsg.InvList)+1 > msg.MaxInvPerMsg {
				break
			}
		}
	}

	// Send the inventory message if there is anything to send.
	if len(invMsg.InvList) > 0 {
		sp.QueueMessage(invMsg, nil)
	}
}

// OnTx is invoked when a peer receives a tx message.  It blocks
// until the transaction has been fully processed.  Unlock the block
// handler this does not serialize all transactions through a single thread
// transactions don't rely on the previous one in a linear fashion like blocks.
func (sp *ServerPeer) OnTx(_ *peer.Peer, msgTx *msg.Tx) {
	// Add the transaction to the known inventory for the peer.
	// Convert the raw MsgTx to a btcutil.Tx which provides some convenience
	// methods and things such as hash caching.
	tx := msgTx.Serializable.(interfaces.Transaction)
	txId := tx.Hash()
	iv := msg.NewInvVect(msg.InvTypeTx, &txId)
	sp.AddKnownInventory(iv)

	// Queue the transaction up to be handled by the sync manager and
	// intentionally block further receives until the transaction is fully
	// processed and known good or bad.  This helps prevent a malicious peer
	// from queuing up a bunch of bad transactions before disconnecting (or
	// being disconnected) and wasting memory.
	sp.server.SyncManager.QueueTx(tx, sp.Peer, sp.txProcessed)
	<-sp.txProcessed
}

// OnBlock is invoked when a peer receives a block message.  It
// blocks until the block has been fully processed.
func (sp *ServerPeer) OnBlock(_ *peer.Peer, msgBlock *msg.Block) {
	block := msgBlock.Serializable.(*types.DposBlock)
	blockHash := block.Block.Hash()
	iv := msg.NewInvVect(msg.InvTypeBlock, &blockHash)
	if block.HaveConfirm {
		iv.Type = msg.InvTypeConfirmedBlock
	}

	// Add the block to the known inventory for the peer.
	sp.AddKnownInventory(iv)

	// Queue the block up to be handled by the block
	// manager and intentionally block further receives
	// until the block is fully processed and known
	// good or bad.  This helps prevent a malicious peer
	// from queuing up a bunch of bad blocks before
	// disconnecting (or being disconnected) and wasting
	// memory.  Additionally, this behavior is depended on
	// by at least the block acceptance test tool as the
	// reference implementation processes blocks in the same
	// thread and therefore blocks further messages until
	// the block has been fully processed.
	sp.server.SyncManager.QueueBlock(block, sp.Peer, sp.blockProcessed)
	<-sp.blockProcessed
}

// OnInv is invoked when a peer receives an inv message and is
// used to examine the inventory being advertised by the remote peer and react
// accordingly.  We pass the message down to blockmanager which will call
// QueueMessage with any appropriate responses.
func (sp *ServerPeer) OnInv(_ *peer.Peer, inv *msg.Inv) {
	if len(inv.InvList) > 0 {
		sp.server.SyncManager.QueueInv(inv, sp.Peer)
		sp.server.Routes.QueueInv(sp.Peer, inv)
	}
}

// OnNotFound is invoked when a peer receives an notfounc message.
// A peer should not response a notfound message so we just disconnect it.
func (sp *ServerPeer) OnNotFound(_ *peer.Peer, notFound *msg.NotFound) {
	for _, i := range notFound.InvList {
		if i.Type == msg.InvTypeTx {
			continue
		}

		log.Debugf("%s sent us notfound message --  disconnecting", sp)
		sp.AddBanScore(100, 0, notFound.CMD())
		sp.Disconnect()
		return
	}
}

// handleGetData is invoked when a peer receives a getdata message and
// is used to deliver block and transaction information.
func (sp *ServerPeer) OnGetData(_ *peer.Peer, getData *msg.GetData) {
	// Notify the GetData message to DPOS Routes.
	sp.server.Routes.OnGetData(sp.Peer, getData)

	numAdded := 0
	notFound := msg.NewNotFound()

	length := len(getData.InvList)
	// A decaying ban score increase is applied to prevent exhausting resources
	// with unusually large inventory queries.
	// Requesting more than the maximum inventory vector length within a short
	// period of time yields a score above the default ban threshold. Sustained
	// bursts of small requests are not penalized as that would potentially ban
	// peers performing IBD.
	// This incremental score decays each minute to half of its value.
	sp.AddBanScore(0, uint32(length)*99/msg.MaxInvPerMsg, "getdata")

	// We wait on this wait channel periodically to prevent queuing
	// far more data than we can send in a reasonable time, wasting memory.
	// The waiting occurs after the database fetch for the next one to
	// provide a little pipelining.
	var waitChan chan struct{}
	doneChan := make(chan struct{}, 1)

	for i, iv := range getData.InvList {
		var c chan struct{}
		// If this will be the last message we send.
		if i == length-1 && len(notFound.InvList) == 0 {
			c = doneChan
		} else if (i+1)%5 == 0 {
			// Buffered so as to not make the send goroutine block.
			c = make(chan struct{}, 1)
		}
		var err error
		switch iv.Type {
		case msg.InvTypeTx:
			err = sp.server.pushTxMsg(sp, &iv.Hash, c, waitChan)
		case msg.InvTypeBlock:
			err = sp.server.pushBlockMsg(sp, &iv.Hash, c, waitChan)
		case msg.InvTypeConfirmedBlock:
			err = sp.server.pushConfirmedBlockMsg(sp, &iv.Hash, c, waitChan)
		case msg.InvTypeFilteredBlock:
			err = sp.server.pushMerkleBlockMsg(sp, &iv.Hash, c, waitChan)
		case msg.InvTypeAddress:
			continue
		default:
			log.Warnf("Unknown type in inventory request %d", iv.Type)
			continue
		}
		if err != nil {
			notFound.AddInvVect(iv)

			// When there is a failure fetching the final entry
			// and the done channel was sent in due to there
			// being no outstanding not found inventory, consume
			// it here because there is now not found inventory
			// that will use the channel momentarily.
			if i == length-1 && c != nil {
				<-c
			}
		}
		numAdded++
		waitChan = c
	}
	if len(notFound.InvList) != 0 {
		sp.QueueMessage(notFound, doneChan)
	}

	// Wait for messages to be sent. We can send quite a lot of data at this
	// point and this will keep the peer busy for a decent amount of time.
	// We don't process anything else by them in this time so that we
	// have an idea of when we should hear back from them - else the idle
	// timeout could fire when we were only half done sending the blocks.
	if numAdded > 0 {
		<-doneChan
	}
}

// OnGetBlocks is invoked when a peer receives a getblocks
// message.
func (sp *ServerPeer) OnGetBlocks(_ *peer.Peer, m *msg.GetBlocks) {
	// Find the most recent known block in the best chain based on the block
	// locator and fetch all of the block hashes after it until either
	// wire.MaxBlocksPerMsg have been fetched or the provided stop hash is
	// encountered.
	//
	// Use the block after the genesis block if no other blocks in the
	// provided locator are known.  This does mean the client will start
	// over with the genesis block if unknown block locators are provided.
	//
	// This mirrors the behavior in the reference implementation.
	chain := sp.server.chain
	hashList := chain.LocateBlocks(m.Locator, &m.HashStop, pact.MaxBlocksPerMsg)

	// Generate inventory message.
	invMsg := msg.NewInv()
	for i := range hashList {
		invType := msg.InvTypeConfirmedBlock
		if sp.filter.IsLoaded() { // Compatible for SPV client.
			invType = msg.InvTypeBlock
		}
		iv := msg.NewInvVect(invType, hashList[i])
		invMsg.AddInvVect(iv)
	}

	// Send the inventory message if there is anything to send.
	if len(invMsg.InvList) > 0 {
		invListLen := len(invMsg.InvList)
		if invListLen == pact.MaxBlocksPerMsg {
			// Intentionally use a copy of the final hash so there
			// is not a reference into the inventory slice which
			// would prevent the entire slice from being eligible
			// for GC as soon as it's sent.
			continueHash := invMsg.InvList[invListLen-1].Hash
			sp.continueHash = &continueHash
		}
		sp.QueueMessage(invMsg, nil)
	}
}

// enforceTxFilterFlag disconnects the peer if the NetServer is not configured to
// allow tx filters.  Additionally, if the peer has negotiated to a protocol
// version  that is high enough to observe the bloom filter service support bit,
// it will be banned since it is intentionally violating the protocol.
func (sp *ServerPeer) enforceTxFilterFlag(cmd string) bool {
	if sp.server.services&pact.SFTxFiltering != pact.SFTxFiltering {
		// Disconnect the peer regardless of protocol version or banning
		// state.
		log.Debugf("%s sent an unsupported %s request -- "+
			"disconnecting", sp, cmd)
		sp.AddBanScore(100, 0, cmd)
		sp.Disconnect()
		return false
	}

	return true
}

// OnFilterAdd is invoked when a peer receives a filteradd
// message and is used by remote peers to add data to an already loaded bloom
// filter.  The peer will be disconnected if a filter is not loaded when this
// message is received or the NetServer is not configured to allow bloom filters.
func (sp *ServerPeer) OnFilterAdd(_ *peer.Peer, filterAdd *msg.FilterAdd) {
	// Disconnect and/or ban depending on the node bloom services flag and
	// negotiated protocol version.
	if !sp.enforceTxFilterFlag(filterAdd.CMD()) {
		return
	}

	if !sp.filter.IsLoaded() {
		log.Debugf("%s sent a filteradd request with no filter "+
			"loaded -- disconnecting", sp)
		sp.Disconnect()
		return
	}

	err := sp.filter.Add(filterAdd.Data)
	if err != nil {
		log.Debugf("%s sent invalid filteradd request with error %s"+
			" -- disconnecting", sp, err)
		sp.Disconnect()
	}
}

// OnFilterClear is invoked when a peer receives a filterclear
// message and is used by remote peers to clear an already loaded bloom filter.
// The peer will be disconnected if a filter is not loaded when this message is
// received  or the NetServer is not configured to allow bloom filters.
func (sp *ServerPeer) OnFilterClear(_ *peer.Peer, filterClear *msg.FilterClear) {
	// Disconnect and/or ban depending on the node bloom services flag and
	// negotiated protocol version.
	if !sp.enforceTxFilterFlag(filterClear.CMD()) {
		return
	}

	if !sp.filter.IsLoaded() {
		log.Debugf("%s sent a filterclear request with no "+
			"filter loaded -- disconnecting", sp)
		sp.Disconnect()
		return
	}

	sp.filter.Clear()

	sp.SetDisableRelayTx(true)
}

// OnFilterLoad is invoked when a peer receives a filterload
// message and it used to load a bloom filter that should be used for
// delivering merkle blocks and associated transactions that match the filter.
// The peer will be disconnected if the NetServer is not configured to allow bloom
// filters.
func (sp *ServerPeer) OnFilterLoad(_ *peer.Peer, filterLoad *msg.FilterLoad) {
	// Disconnect and/or ban depending on the node bloom services flag and
	// negotiated protocol version.
	if !sp.enforceTxFilterFlag(filterLoad.CMD()) {
		return
	}

	sp.SetDisableRelayTx(false)

	buf := new(bytes.Buffer)
	filterLoad.Serialize(buf)
	err := sp.filter.Load(&msg.TxFilterLoad{
		Type: filter.FTBloom,
		Data: buf.Bytes(),
	})
	if err != nil {
		log.Debugf("%s sent invalid filterload request with error %s"+
			" -- disconnecting", sp, err)
		sp.Disconnect()
	}
}

// OnTxFilterLoad is invoked when a peer receives a txfilter message and it used to
// load a transaction filter that should be used for delivering merkle blocks and
// associated transactions that match the filter. The peer will be disconnected
// if the NetServer is not configured to allow transaction filtering.
func (sp *ServerPeer) OnTxFilterLoad(_ *peer.Peer, tf *msg.TxFilterLoad) {
	// Disconnect and/or ban depending on the tx filter services flag and
	// negotiated protocol version.
	if !sp.enforceTxFilterFlag(tf.CMD()) {
		return
	}

	sp.SetDisableRelayTx(false)

	err := sp.filter.Load(tf)
	if err != nil {
		log.Debugf("%s sent invalid txfilter request with error %s"+
			" -- disconnecting", sp, err)
		sp.Disconnect()
		return
	}
}

// OnReject is invoked when a peer receives a reject message.
func (sp *ServerPeer) OnReject(_ *peer.Peer, msg *msg.Reject) {
	log.Infof("%s sent a reject message Code: %s, Hash %s, Reason: %s",
		sp, msg.RejectCode.String(), msg.Hash.String(), msg.Reason)
}

// pushTxMsg sends a tx message for the provided transaction hash to the
// connected peer.  An error is returned if the transaction hash is not known.
func (s *NetServer) pushTxMsg(sp *ServerPeer, hash *common.Uint256, doneChan chan<- struct{},
	waitChan <-chan struct{}) error {

	// Attempt to fetch the requested transaction from the pool.  A
	// call could be made to check for existence first, but simply trying
	// to fetch a missing transaction results in the same behavior.
	tx := s.txMemPool.GetTransaction(*hash)
	if tx == nil {
		if doneChan != nil {
			doneChan <- struct{}{}
		}
		return fmt.Errorf("unable to fetch tx %v from transaction pool", hash)
	}

	// Once we have fetched data wait for any previous operation to finish.
	if waitChan != nil {
		<-waitChan
	}

	sp.QueueMessage(msg.NewTx(tx), doneChan)

	return nil
}

// pushBlockMsg sends a block message for the provided block hash to the
// connected peer.  An error is returned if the block hash is not known.
func (s *NetServer) pushBlockMsg(sp *ServerPeer, hash *common.Uint256, doneChan chan<- struct{},
	waitChan <-chan struct{}) error {

	// Fetch the block from the database.
	block, _ := s.blockMemPool.GetDposBlockByHash(*hash)
	if block == nil {
		block, _ = s.chain.GetDposBlockByHash(*hash)
		if block == nil {
			if doneChan != nil {
				doneChan <- struct{}{}
			}
			return errors.New("block not found")
		}
	}
	block.HaveConfirm = false
	block.Confirm = nil

	// Once we have fetched data wait for any previous operation to finish.
	if waitChan != nil {
		<-waitChan
	}

	// We only send the channel for this message if we aren't sending
	// an inv straight after.
	var dc chan<- struct{}
	continueHash := sp.continueHash
	sendInv := continueHash != nil && continueHash.IsEqual(*hash)
	if !sendInv {
		dc = doneChan
	}
	sp.QueueMessage(msg.NewBlock(block), dc)

	// When the peer requests the final block that was advertised in
	// response to a getblocks message which requested more blocks than
	// would fit into a single message, send it a new inventory message
	// to trigger it to issue another getblocks message for the next
	// batch of inventory.
	if sendInv {
		best := sp.server.chain.GetBestChain()
		invMsg := msg.NewInvSize(1)
		iv := msg.NewInvVect(msg.InvTypeBlock, best.Hash)
		invMsg.AddInvVect(iv)
		sp.QueueMessage(invMsg, doneChan)
		sp.continueHash = nil
	}
	return nil
}

// pushBlockMsg sends a block message for the provided block hash to the
// connected peer.  An error is returned if the block hash is not known.
func (s *NetServer) pushConfirmedBlockMsg(sp *ServerPeer, hash *common.Uint256, doneChan chan<- struct{},
	waitChan <-chan struct{}) error {

	// Fetch the block from the database.
	block, _ := s.chain.GetDposBlockByHash(*hash)
	if block == nil {
		// Fetch the block from the block pool.
		block, _ = s.blockMemPool.GetDposBlockByHash(*hash)
		if block == nil || !block.HaveConfirm {
			if doneChan != nil {
				doneChan <- struct{}{}
			}
			return errors.New("confirmed block not found")
		}
	}

	// Once we have fetched data wait for any previous operation to finish.
	if waitChan != nil {
		<-waitChan
	}

	// We only send the channel for this message if we aren't sending
	// an inv straight after.
	var dc chan<- struct{}
	continueHash := sp.continueHash
	sendInv := continueHash != nil && continueHash.IsEqual(*hash)
	if !sendInv {
		dc = doneChan
	}
	sp.QueueMessage(msg.NewBlock(block), dc)

	// When the peer requests the final block that was advertised in
	// response to a getblocks message which requested more blocks than
	// would fit into a single message, send it a new inventory message
	// to trigger it to issue another getblocks message for the next
	// batch of inventory.
	if sendInv {
		best := sp.server.chain.GetBestChain()
		invMsg := msg.NewInvSize(1)
		iv := msg.NewInvVect(msg.InvTypeConfirmedBlock, best.Hash)
		invMsg.AddInvVect(iv)
		sp.QueueMessage(invMsg, doneChan)
		sp.continueHash = nil
	}
	return nil
}

// pushMerkleBlockMsg sends a merkleblock message for the provided block hash to
// the connected peer.  Since a merkle block requires the peer to have a filter
// loaded, this call will simply be ignored if there is no filter loaded.  An
// error is returned if the block hash is not known.
func (s *NetServer) pushMerkleBlockMsg(sp *ServerPeer, hash *common.Uint256,
	doneChan chan<- struct{}, waitChan <-chan struct{}) error {

	// Do not send a response if the peer doesn't have a filter loaded.
	if !sp.filter.IsLoaded() {
		if doneChan != nil {
			doneChan <- struct{}{}
		}
		return nil
	}

	// Fetch the block from the database.
	blk, err := s.chain.GetDposBlockByHash(*hash)
	if err != nil {
		if doneChan != nil {
			doneChan <- struct{}{}
		}
		return err
	}

	// Generate a merkle block by filtering the requested block according
	// to the filter for the peer.
	merkle, matchedTxIndices := filter.NewMerkleBlock(blk.Transactions,
		sp.filter)

	// Create block header.
	switch sp.filter.Filter().(type) {
	// Compatible with old version SPV clients.
	case *bloom.TxFilter:
		merkle.Header = &blk.Header

	// Side chain needs DPOS header format to receive confirm.
	case *sidefilter.Filter:
		var confirm payload.Confirm
		if blk.HaveConfirm {
			confirm = *blk.Confirm
		}
		merkle.Header = &types.DPOSHeader{
			Header:      blk.Header,
			HaveConfirm: blk.HaveConfirm,
			Confirm:     confirm,
		}
	case *nextturndposfilter.NextTurnDPOSInfoFilter:
		merkle.Header = &blk.Header
	case *returnsidechaindepositcoinfilter.ReturnSidechainDepositCoinFilter:
		merkle.Header = &blk.Header
	case *customidfilter.CustomIdFilter:
		merkle.Header = &blk.Header
	}
	// Once we have fetched data wait for any previous operation to finish.
	if waitChan != nil {
		<-waitChan
	}

	// Send the merkleblock.  Only send the done channel with this message
	// if no transactions will be sent afterwards.
	var dc chan<- struct{}
	if len(matchedTxIndices) == 0 {
		dc = doneChan
	}

	sp.QueueMessage(merkle, dc)

	// Finally, send any matched transactions.
	blkTransactions := blk.Transactions
	for i, txIndex := range matchedTxIndices {
		// Only send the done channel on the final transaction.
		var dc chan<- struct{}
		if i == len(matchedTxIndices)-1 {
			dc = doneChan
		}
		if txIndex < uint32(len(blkTransactions)) {
			sp.QueueMessage(msg.NewTx(blkTransactions[txIndex]), dc)
		}
	}

	return nil
}

// handleRelayInvMsg deals with relaying inventory to peers that are not already
// known to have it.  It is invoked from the peerHandler goroutine.
func (s *NetServer) handleRelayInvMsg(peers map[svr.IPeer]*ServerPeer, rmsg relayMsg) {
	// TODO remove after main net growth higher than H1 for efficiency.
	current := s.chain.GetHeight()

	for _, sp := range peers {
		if !sp.Connected() {
			continue
		}

		switch rmsg.invVect.Type {
		case msg.InvTypeTx:
			// Don't relay the transaction to the peer when it has
			// transaction relaying disabled.
			if sp.RelayTxDisabled() {
				continue
			}

			tx, ok := rmsg.data.(interfaces.Transaction)
			if !ok {
				log.Warnf("Underlying data for tx inv "+
					"relay is not a *core.BaseTransaction: %T",
					rmsg.data)
				return
			}

			// Don't relay the transaction if there is a bloom
			// filter loaded and the transaction doesn't match it.
			if sp.filter.IsLoaded() &&
				!sp.filter.MatchUnconfirmed(tx) {
				continue
			}

		case msg.InvTypeBlock:
			fallthrough
		case msg.InvTypeConfirmedBlock:
			// Compatible for old version SPV client.
			if sp.filter.IsLoaded() {
				// Do not send unconfirmed block to SPV client after H1.
				if current >= s.ChainParams.CRCOnlyDPOSHeight-1 &&
					s.chain.GetState().ConsensusAlgorithm != state.POW &&
					rmsg.invVect.Type == msg.InvTypeBlock {
					continue
				}

				// Change inv type to InvTypeBlock for compatible.
				invVect := *rmsg.invVect
				invVect.Type = msg.InvTypeBlock

				sp.QueueInventory(&invVect)
				continue
			}
		}

		// Queue the inventory to be relayed with the next batch.
		// It will be ignored if the peer is already known to
		// have the inventory.
		sp.QueueInventory(rmsg.invVect)
	}
}

// peerHandler is used to handle peer operations such as adding and removing
// peers to and from the NetServer, banning peers, and broadcasting messages to
// peers.  It must be run in a goroutine.
func (s *NetServer) peerHandler() {

	// Reset the TimeSource of BlockChain.
	s.resetTimeSource()

	// Start the address manager and sync manager, both of which are needed
	// by peers.  This is done here since their lifecycle is closely tied
	// to this handler and rather than adding more channels to sychronize
	// things, it's easier and slightly faster to simply start and stop them
	// in this handler.
	s.SyncManager.Start()

	peers := make(map[svr.IPeer]*ServerPeer)

out:
	for {
		select {
		// Deal with peer messages.
		case peer := <-s.peerQueue:
			s.HandlePeerMsg(peers, peer)

			// New inventory to potentially be relayed to other peers.
		case invMsg := <-s.relayInv:
			s.handleRelayInvMsg(peers, invMsg)

		case <-s.quit:
			break out
		}
	}

	s.SyncManager.Stop()

	// Drain channels before exiting so nothing is left waiting around
	// to send.
cleanup:
	for {
		select {
		case <-s.peerQueue:
		case <-s.relayInv:
		default:
			break cleanup
		}
	}
}

func (s *NetServer) isOverMaxNodePerHost(peers map[svr.IPeer]*ServerPeer,
	orgPeer svr.IPeer) bool {
	sp := orgPeer.ToPeer()
	hostNodeCount := uint32(1)
	for _, peer := range peers {
		var peerNa, spNa *p2p.NetAddress
		if peerNa = peer.NA(); peerNa == nil {
			continue
		}
		if spNa = sp.NA(); spNa == nil {
			return true
		}
		if peer.NA().IP.String() == sp.NA().IP.String() {
			hostNodeCount++
		}
	}
	if hostNodeCount > s.ChainParams.MaxNodePerHost {
		log.Infof("New peer %s ignored, "+
			"hostNodeCount %d is more than  MaxNodePerHost %d ",
			sp, hostNodeCount, s.ChainParams.MaxNodePerHost)
		return true
	}
	return false
}

// HandlePeerMsg deals with adding and removing peers.
func (s *NetServer) HandlePeerMsg(peers map[svr.IPeer]*ServerPeer, p interface{}) {
	switch p := p.(type) {
	case NewPeerMsg:
		sp := newServerPeer(s)
		if s.isOverMaxNodePerHost(peers, p) {
			p.Reply <- false
			return
		}
		sp.Peer = peer.New(p, &peer.Listeners{
			OnVersion:      sp.OnVersion,
			OnMemPool:      sp.OnMemPool,
			OnTx:           sp.OnTx,
			OnBlock:        sp.OnBlock,
			OnInv:          sp.OnInv,
			OnNotFound:     sp.OnNotFound,
			OnGetData:      sp.OnGetData,
			OnGetBlocks:    sp.OnGetBlocks,
			OnFilterAdd:    sp.OnFilterAdd,
			OnFilterClear:  sp.OnFilterClear,
			OnFilterLoad:   sp.OnFilterLoad,
			OnTxFilterLoad: sp.OnTxFilterLoad,
			OnReject:       sp.OnReject,
			OnDAddr:        s.Routes.QueueDAddr,
		})
		peers[p.IPeer] = sp
		p.Reply <- true
	case DonePeerMsg:
		delete(peers, p.IPeer)
		p.Reply <- struct{}{}
	}
}

// Reset TimeSource after one second to avoid accepting the wrong time
// in version message.
func (s *NetServer) resetTimeSource() {
	go func() {
		time.Sleep(time.Second)
		s.chain.TimeSource.Reset()
	}()
}

// Services returns the service flags the NetServer supports.
func (s *NetServer) Services() pact.ServiceFlag {
	return s.services
}

// NewPeer adds a new peer that has already been connected to the NetServer.
func (s *NetServer) NewPeer(p svr.IPeer) bool {
	reply := make(chan bool)
	s.peerQueue <- NewPeerMsg{p, reply}
	return <-reply
}

// DonePeer removes a peer that has already been connected to the NetServer by ip.
func (s *NetServer) DonePeer(p svr.IPeer) {
	reply := make(chan struct{})
	s.peerQueue <- DonePeerMsg{p, reply}
	<-reply
}

// RelayInventory relays the passed inventory vector to all connected peers
// that are not already known to have it.
func (s *NetServer) RelayInventory(invVect *msg.InvVect, data interface{}) {
	s.relayInv <- relayMsg{invVect: invVect, data: data}
}

// IsCurrent returns whether or not the sync manager believes it is synced with
// the connected peers.
func (s *NetServer) IsCurrent() bool {
	return s.SyncManager.IsCurrent()
}

// Start begins accepting connections from peers.
func (s *NetServer) Start() {
	s.Routes.Start()
	s.IServer.Start()

	go s.peerHandler()
}

// Stop gracefully shuts down the NetServer by stopping and disconnecting all
// peers and the main listener.
func (s *NetServer) Stop() error {
	s.Routes.Stop()
	err := s.IServer.Stop()

	// Signal the remaining goroutines to quit.
	close(s.quit)
	return err
}

// NewServer returns a new elanet NetServer configured to listen on addr for the
// network type specified by ChainParams.  Use start to begin accepting
// connections from peers.
func NewServer(dataDir string, cfg *Config, nodeVersion string) (*NetServer, error) {
	services := defaultServices
	params := cfg.ChainParams
	if params.DisableTxFilters {
		services &^= pact.SFNodeBloom
		services &^= pact.SFTxFiltering
	}

	// If no listeners added, create default listener.
	if len(params.ListenAddrs) == 0 {
		params.ListenAddrs = []string{fmt.Sprint(":", params.NodePort)}
	}

	var pver = pact.DPOSStartVersion
	if cfg.Chain.GetHeight() >= uint32(params.CRConfiguration.NewP2PProtocolVersionHeight) {
		pver = pact.CRProposalVersion
	}

	svrCfg := svr.NewDefaultConfig(
		params.Magic, pver, uint64(services),
		params.NodePort, params.DNSSeeds, params.ListenAddrs,
		nil, nil, createMessage,
		func() uint64 { return uint64(cfg.Chain.GetHeight()) },
		params.CRConfiguration.NewP2PProtocolVersionHeight, nodeVersion,
	)
	svrCfg.DataDir = dataDir
	svrCfg.NAFilter = &naFilter{}
	svrCfg.PermanentPeers = cfg.PermanentPeers

	s := NetServer{
		chain:        cfg.Chain,
		ChainParams:  cfg.ChainParams,
		txMemPool:    cfg.TxMemPool,
		blockMemPool: cfg.BlockMemPool,
		Routes:       cfg.Routes,
		peerQueue:    make(chan interface{}, svrCfg.MaxPeers),
		relayInv:     make(chan relayMsg, svrCfg.MaxPeers),
		quit:         make(chan struct{}),
		services:     services,
	}
	svrCfg.OnNewPeer = s.NewPeer
	svrCfg.OnDonePeer = s.DonePeer

	p2pServer, err := svr.NewServer(svrCfg)
	if err != nil {
		return nil, err
	}
	s.IServer = p2pServer

	s.SyncManager = netsync.New(&netsync.Config{
		PeerNotifier: &s,
		Chain:        cfg.Chain,
		ChainParams:  cfg.ChainParams,
		TxMemPool:    cfg.TxMemPool,
		BlockMemPool: cfg.BlockMemPool,
		MaxPeers:     svrCfg.MaxPeers,
	})

	return &s, nil
}

func createMessage(hdr p2p.Header, r net.Conn) (p2p.Message, error) {
	var message p2p.Message
	switch hdr.GetCMD() {
	case p2p.CmdMemPool:
		message = &msg.MemPool{}

	case p2p.CmdTx:
		return peer2.CheckAndCreateTxMessage(hdr, r)

	case p2p.CmdBlock:
		message = msg.NewBlock(&types.DposBlock{})

	case p2p.CmdInv:
		message = &msg.Inv{}

	case p2p.CmdNotFound:
		message = &msg.NotFound{}

	case p2p.CmdGetData:
		message = &msg.GetData{}

	case p2p.CmdGetBlocks:
		message = &msg.GetBlocks{}

	case p2p.CmdFilterAdd:
		message = &msg.FilterAdd{}

	case p2p.CmdFilterClear:
		message = &msg.FilterClear{}

	case p2p.CmdFilterLoad:
		message = &msg.FilterLoad{}

	case p2p.CmdTxFilter:
		message = &msg.TxFilterLoad{}

	case p2p.CmdReject:
		message = &msg.Reject{}

	case p2p.CmdDAddr:
		message = &msg.DAddr{}

	default:
		return nil, fmt.Errorf("unhandled command [%s]", hdr.GetCMD())
	}

	return peer2.CheckAndCreateMessage(hdr, message, r)
}

// nodeFlag returns if a peer contains the full node flag.
func nodeFlag(flag uint64) bool {
	return pact.ServiceFlag(flag)&pact.SFNodeNetwork == pact.SFNodeNetwork
}
