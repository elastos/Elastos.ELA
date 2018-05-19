package node

import (
	"errors"
	"strconv"
	"time"
	"fmt"

	"github.com/elastos/Elastos.ELA/bloom"
	chain "github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core"
	. "github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/events"
	"github.com/elastos/Elastos.ELA/log"
	"github.com/elastos/Elastos.ELA/protocol"

	. "github.com/elastos/Elastos.ELA.Utility/common"
	. "github.com/elastos/Elastos.ELA.Utility/p2p"
	"github.com/elastos/Elastos.ELA.Utility/p2p/msg"
)

var DuplicateBlocks int

type MsgHandler struct {
	node protocol.Noder
}

// When something wrong on read or decode message
// this method will callback the error
func (h *MsgHandler) OnDecodeError(err error) {
	switch err {
	case ErrUnmatchedMagic:
		log.Error("Decode message magic not match")
		h.node.CloseConn()
	case ErrDisconnected:
		log.Error("Decode message peer disconnected")
		LocalNode.GetEvent("disconnect").Notify(events.EventNodeDisconnect, h.node)
	default:
		log.Error("Decode message error: ", err)
	}
}

// After message header decoded, this method will be
// called to create the message instance with the CMD
// which is the message type of the received message
func (h *MsgHandler) OnMakeMessage(cmd string) (Message, error) {
	err := FilterMessage(h.node, cmd)
	if err != nil {
		return nil, err
	}

	var message Message
	switch cmd {
	case "version":
		message = new(msg.Version)
	case "verack":
		message = new(msg.VerAck)
	case "getaddr":
		message = new(msg.AddrsReq)
	case "addr":
		message = new(msg.Addrs)
	case "filterload":
		message = new(bloom.FilterLoad)
	case "inv":
		message = new(msg.Inventory)
	case "getdata":
		message = new(msg.DataReq)
	case "block":
		message = new(core.Block)
	case "tx":
		message = new(core.Transaction)
	case "getblocks":
		message = new(msg.BlocksReq)
	case "notfound":
		message = new(msg.NotFound)
	case "ping":
		message = new(msg.Ping)
	case "pong":
		message = new(msg.Pong)
	default:
		return nil, errors.New("unknown message type")
	}

	// Update node last active time
	h.node.UpdateLastActive()
	return message, nil
}

// After message has been successful decoded, this method
// will be called to pass the decoded message instance
func (h *MsgHandler) OnMessageDecoded(message Message) {
	var err error
	switch message := message.(type) {
	case *msg.Version:
		err = h.onVersion(message)
	case *msg.VerAck:
		err = h.onVerAck(message)
	case *msg.AddrsReq:
		err = h.onAddrsReq(message)
	case *msg.Addrs:
		err = h.onAddrs(message)
	case *bloom.FilterLoad:
		err = h.onFilterLoad(message)
	case *msg.Inventory:
		err = h.onInventory(message)
	case *msg.DataReq:
		err = h.onDataReq(message)
	case *core.Block:
		err = h.onBlock(message)
	case *core.Transaction:
		err = h.onTx(message)
	case *msg.BlocksReq:
		err = h.onBlocksReq(message)
	case *msg.NotFound:
		err = h.onNotFound(message)
	case *msg.Ping:
		err = h.onPing(message)
	case *msg.Pong:
		err = h.onPong(message)
	default:
		err = errors.New("unknown message type")
	}
	if err != nil {
		log.Error("Decode message error: " + err.Error())
	}
}

func (h *MsgHandler) onVersion(version *msg.Version) error {
	log.Debug()
	node := h.node

	// Exclude the node itself
	if version.Nonce == LocalNode.ID() {
		log.Warn("The node handshake with itself")
		node.CloseConn()
		return errors.New("The node handshake with itself")
	}

	s := node.State()
	if s != INIT && s != HAND {
		log.Warn("Unknow status to receive version")
		return errors.New("Unknow status to receive version")
	}

	// Obsolete node
	n, ret := LocalNode.DelNbrNode(version.Nonce)
	if ret == true {
		log.Info(fmt.Sprintf("Node reconnect 0x%x", version.Nonce))
		// Close the connection and release the node soure
		n.SetState(INACTIVITY)
		n.CloseConn()
	}

	node.UpdateInfo(time.Now(), version.Version, version.Services,
		version.Port, version.Nonce, version.Relay, version.Height)
	LocalNode.AddNbrNode(node)

	// Do not add SPV client as a known address,
	// so node will not start a connection to SPV client
	if node.LocalPort() != protocol.SPVPort {
		ip, _ := node.Addr16()
		addr := msg.Addr{
			Time:     node.GetTime(),
			Services: version.Services,
			IP:       ip,
			Port:     version.Port,
			ID:       version.Nonce,
		}
		LocalNode.AddAddressToKnownAddress(addr)
	}

	var message Message
	if s == INIT {
		node.SetState(HANDSHAKE)
		message = NewVersion(LocalNode)
	} else if s == HAND {
		node.SetState(HANDSHAKED)
		message = new(msg.VerAck)
	}
	go node.Send(message)

	return nil
}

func (h *MsgHandler) onVerAck(verAck *msg.VerAck) error {
	log.Debug()
	node := h.node
	s := node.State()
	if s != HANDSHAKE && s != HANDSHAKED {
		log.Warn("unknown status to received verack")
		return errors.New("unknown status to received verack")
	}

	node.SetState(ESTABLISH)

	if s == HANDSHAKE {
		node.Send(verAck)
	}

	if LocalNode.NeedMoreAddresses() {
		node.ReqNeighborList()
	}
	addr := node.Addr()
	port := node.Port()
	nodeAddr := addr + ":" + strconv.Itoa(int(port))
	LocalNode.RemoveAddrInConnectingList(nodeAddr)
	return nil
}

func (h *MsgHandler) onAddrsReq(req *msg.AddrsReq) error {
	log.Debug()
	addrs := LocalNode.RandSelectAddresses()
	go h.node.Send(msg.NewAddrs(addrs))
	return nil
}

func (h *MsgHandler) onAddrs(addrs *msg.Addrs) error {
	log.Debug()
	for _, addr := range addrs.Addrs {
		log.Info(fmt.Sprintf("The ip address is %s id is 0x%x", addr.String(), addr.ID))

		if addr.ID == LocalNode.ID() {
			continue
		}
		if LocalNode.NodeEstablished(addr.ID) {
			continue
		}

		if addr.Port == 0 {
			continue
		}

		//save the node address in address list
		LocalNode.AddAddressToKnownAddress(addr)
	}
	return nil
}

func (h *MsgHandler) onFilterLoad(msg *bloom.FilterLoad) error {
	log.Debug()
	h.node.LoadFilter(msg)
	return nil
}

func (h *MsgHandler) onInventory(inv *msg.Inventory) error {
	log.Debug()
	node := h.node
	log.Debug(fmt.Sprintf("The inv type: no one. block len: %d, %s\n", len(inv.Hashes), inv.Hashes))

	log.Debug("RX block message")
	if LocalNode.IsSyncHeaders() == true && node.IsSyncHeaders() == false {
		return nil
	}

	for i, hash := range inv.Hashes {
		// Request block
		if !chain.DefaultLedger.BlockInLedger(*hash) {
			if !LocalNode.RequestedBlockExisted(*hash) && !chain.DefaultLedger.Blockchain.IsKnownOrphan(hash) {
				LocalNode.AddRequestedBlock(*hash)
				go node.Send(msg.NewDataReq(BlockData, *hash))
			}
		}

		// Request fork chain
		if chain.DefaultLedger.Blockchain.IsKnownOrphan(hash) {
			orphanRoot := chain.DefaultLedger.Blockchain.GetOrphanRoot(hash)
			locator, err := chain.DefaultLedger.Blockchain.LatestBlockLocator()
			if err != nil {
				log.Errorf(" Failed to get block locator for the latest block: %v", err)
				continue
			}
			SendBlocksReq(node, locator, *orphanRoot)
			continue
		}

		// Request next hashes
		if i == len(inv.Hashes)-1 {
			locator := chain.DefaultLedger.Blockchain.BlockLocatorFromHash(hash)
			SendBlocksReq(node, locator, EmptyHash)
		}
	}
	return nil
}

func (h *MsgHandler) onDataReq(req *msg.DataReq) error {
	log.Debug()
	node := h.node
	hash := req.Hash
	switch req.Type {
	case BlockData:
		block, err := chain.DefaultLedger.Store.GetBlock(hash)
		if err != nil {
			log.Debug("Can't get block from hash: ", hash, " ,send not found message")
			go node.Send(msg.NewNotFound(hash))
			return err
		}
		log.Debug("block height is ", block.Header.Height, " ,hash is ", hash)

		var message Message
		if node.BloomFilter().IsLoaded() {
			message = bloom.NewMerkleBlock(block, node.BloomFilter())
		} else {
			message = block
		}
		go node.Send(message)

	case TxData:
		txn, err := chain.DefaultLedger.GetTransactionWithHash(hash)
		if err != nil {
			log.Error("Get transaction with hash error: ", err.Error())
			return err
		}
		go node.Send(txn)
	}
	return nil
}

func (h *MsgHandler) onBlock(block *core.Block) error {
	log.Debug()
	node := h.node

	hash := block.Hash()
	if LocalNode.IsNeighborNoder(node) == false {
		log.Trace("received headers message from unknown peer")
		return errors.New("received headers message from unknown peer")
	}

	if chain.DefaultLedger.BlockInLedger(hash) {
		DuplicateBlocks++
		log.Trace("Receive ", DuplicateBlocks, " duplicated block.")
		return nil
	}

	chain.DefaultLedger.Store.RemoveHeaderListElement(hash)
	LocalNode.DeleteRequestedBlock(hash)
	_, isOrphan, err := chain.DefaultLedger.Blockchain.AddBlock(block)

	if err != nil {
		log.Warn("Block add failed: ", err, " ,block hash is ", hash.Bytes())
		return err
	}
	//relay
	if LocalNode.IsSyncHeaders() == false {
		if !LocalNode.ExistedID(hash) {
			LocalNode.Relay(node, block)
			log.Debug("Relay block")
		}
	}

	if isOrphan == true && LocalNode.IsSyncHeaders() == false {
		if !LocalNode.RequestedBlockExisted(hash) {
			orphanRoot := chain.DefaultLedger.Blockchain.GetOrphanRoot(&hash)
			locator, _ := chain.DefaultLedger.Blockchain.LatestBlockLocator()
			SendBlocksReq(node, locator, *orphanRoot)
		}
	}

	return nil
}

func (h *MsgHandler) onTx(tx *core.Transaction) error {
	log.Debug()
	node := h.node
	log.Debug("RX Transaction message")
	if !LocalNode.ExistedID(tx.Hash()) {
		if errCode := LocalNode.AppendToTxnPool(tx); errCode != Success {
			return errors.New("[message] VerifyTransaction failed when AppendToTxnPool.")
		}
		LocalNode.Relay(node, tx)
		log.Info("Relay Transaction")
		LocalNode.IncRxTxnCnt()
		log.Debug("RX Transaction message hash", tx.Hash().String())
		log.Debug("RX Transaction message type", tx.TxType.Name())
	}

	return nil
}

func (h *MsgHandler) onBlocksReq(req *msg.BlocksReq) error {
	log.Debug()
	node := h.node
	LocalNode.AcqSyncHdrReqSem()
	defer LocalNode.RelSyncHdrReqSem()
	start := chain.DefaultLedger.Blockchain.LatestLocatorHash(req.Locator)
	hashes, err := GetBlockHashes(*start, req.HashStop)
	if err != nil {
		return err
	}
	go node.Send(msg.NewInventory(BlockData, hashes))
	return nil
}

func (h *MsgHandler) onNotFound(msg *msg.NotFound) error {
	log.Debug("Received not found message, hash: ", msg.Hash.String())
	return nil
}

func (h *MsgHandler) onPing(ping *msg.Ping) error {
	log.Debug()
	h.node.SetHeight(ping.Height)
	go h.node.Send(msg.NewPong(chain.DefaultLedger.Store.GetHeight()))
	return nil
}

func (h *MsgHandler) onPong(pong *msg.Pong) error {
	log.Debug()
	h.node.SetHeight(pong.Height)
	return nil
}

func NewVersion(node protocol.Noder) *msg.Version {
	log.Debug()
	msg := new(msg.Version)
	msg.Version = node.Version()
	msg.Services = node.Services()

	msg.TimeStamp = uint32(time.Now().UTC().UnixNano())
	msg.Port = node.Port()
	msg.Nonce = node.ID()
	msg.Height = uint64(chain.DefaultLedger.GetLocalBlockChainHeight())
	if node.IsRelay() {
		msg.Relay = 1
	} else {
		msg.Relay = 0
	}

	return msg
}

func SendBlocksReq(node protocol.Noder, locator []*Uint256, hashStop Uint256) {
	if LocalNode.GetStartHash() == *locator[0] &&
		LocalNode.GetStopHash() == hashStop {
		return
	}
	LocalNode.SetSyncHeaders(true)
	node.SetSyncHeaders(true)
	LocalNode.SetStartHash(*locator[0])
	LocalNode.SetStopHash(hashStop)
	go node.Send(msg.NewBlocksReq(locator, hashStop))
}

func GetBlockHashes(startHash Uint256, stopHash Uint256) ([]*Uint256, error) {
	var count = uint32(0)
	var startHeight uint32
	var stopHeight uint32
	curHeight := chain.DefaultLedger.Store.GetHeight()
	if stopHash == EmptyHash {
		if startHash == EmptyHash {
			if curHeight > MaxHeaderHashes {
				count = MaxHeaderHashes
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
			if count > MaxHeaderHashes {
				count = MaxHeaderHashes
			}
		}
	} else {
		stopHeader, err := chain.DefaultLedger.Store.GetHeader(stopHash)
		if err != nil {
			return nil, err
		}
		stopHeight = stopHeader.Height
		if startHash != EmptyHash {
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

			if count >= MaxHeaderHashes {
				count = MaxHeaderHashes
			}
		} else {
			if stopHeight > MaxHeaderHashes {
				count = MaxHeaderHashes
			} else {
				count = stopHeight
			}
		}
	}

	hashes := make([]*Uint256, 0)
	for i := uint32(1); i <= count; i++ {
		hash, err := chain.DefaultLedger.Store.GetBlockHash(startHeight + i)
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, &hash)
	}

	return hashes, nil
}
