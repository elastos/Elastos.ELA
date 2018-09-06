package node

import (
	"fmt"

	chain "github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/bloom"
	"github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/log"
	"github.com/elastos/Elastos.ELA/protocol"

	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/p2p"
	"github.com/elastos/Elastos.ELA.Utility/p2p/msg"
)

type HandlerEBIP001 struct {
	HandlerBase
	continueHash *common.Uint256
}

func NewHandlerEBIP001(node protocol.Noder, listener protocol.DposListener) *HandlerEBIP001 {
	return &HandlerEBIP001{HandlerBase: HandlerBase{node: node, listener: listener}}
}

// After message header decoded, this method will be
// called to create the message instance with the CMD
// which is the message type of the received message
func (h *HandlerEBIP001) OnMakeMessage(cmd string) (message p2p.Message, err error) {
	// Nothing to do if node already disconnected
	if h.node.State() == p2p.INACTIVITY {
		return message, fmt.Errorf("revice message from INACTIVE node [0x%x]", h.node.ID())
	}
	// Filter messages through open port message filter
	if err = h.FilterMessage(cmd); err != nil {
		return message, err
	}
	// Update node last active time
	h.node.UpdateLastActive()

	switch cmd {
	case p2p.CmdPing:
		message = new(msg.Ping)
	case p2p.CmdPong:
		message = new(msg.Pong)
	case p2p.CmdFilterLoad:
		message = new(msg.FilterLoad)
	case p2p.CmdGetBlocks:
		message = new(msg.GetBlocks)
	case p2p.CmdInv:
		message = new(msg.Inventory)
	case p2p.CmdGetData:
		message = new(msg.GetData)
	case p2p.CmdBlock:
		message = msg.NewBlock(new(core.Block))
	case p2p.CmdConfirmedBlock:
		message = msg.NewConfirmedBlock(new(msg.DposProposalVoteSlot))
	case p2p.CmdTx:
		message = msg.NewTx(new(core.Transaction))
	case p2p.CmdNotFound:
		message = new(msg.NotFound)
	case p2p.CmdMemPool:
		message = new(msg.MemPool)
	case p2p.CmdReject:
		message = new(msg.Reject)
	default:
		message, err = h.HandlerBase.OnMakeMessage(cmd)
	}

	return message, err
}

// After message has been successful decoded, this method
// will be called to pass the decoded message instance
func (h *HandlerEBIP001) OnMessageDecoded(message p2p.Message) {
	log.Debugf("-----> [%s] from peer [0x%x] STARTED", message.CMD(), h.node.ID())
	if err := h.HandleMessage(message); err != nil {
		log.Error("Handle message error: " + err.Error())
	}
	log.Debugf("-----> [%s] from peer [0x%x] FINISHED", message.CMD(), h.node.ID())
}

func (h *HandlerEBIP001) HandleMessage(message p2p.Message) error {
	var err error
	switch message := message.(type) {
	case *msg.Ping:
		err = h.onPing(message)
	case *msg.Pong:
		err = h.onPong(message)
	case *msg.FilterLoad:
		err = h.onFilterLoad(message)
	case *msg.GetBlocks:
		err = h.onGetBlocks(message)
	case *msg.Inventory:
		err = h.onInventory(message)
	case *msg.GetData:
		err = h.onGetData(message)
	case *msg.Block:
		err = h.onBlock(message)
	case *msg.ConfirmedBlock:
		err = h.onConfirmBlock(message)
	case *msg.Tx:
		err = h.onTx(message)
	case *msg.NotFound:
		err = h.onNotFound(message)
	case *msg.MemPool:
		err = h.onMemPool(message)
	case *msg.Reject:
		err = h.onReject(message)
	default:
		h.HandlerBase.OnMessageDecoded(message)
	}
	return err
}

func (h *HandlerEBIP001) onFilterLoad(msg *msg.FilterLoad) error {
	h.node.LoadFilter(msg)
	return nil
}

func (h *HandlerEBIP001) onPing(ping *msg.Ping) error {
	h.node.SetHeight(ping.Nonce)
	h.node.Send(msg.NewPong(chain.DefaultLedger.Blockchain.BestChain.Height))
	return nil
}

func (h *HandlerEBIP001) onPong(pong *msg.Pong) error {
	h.node.SetHeight(pong.Nonce)
	return nil
}

func (h *HandlerEBIP001) onGetBlocks(req *msg.GetBlocks) error {
	node := h.node
	LocalNode.AcqSyncBlkReqSem()
	defer LocalNode.RelSyncBlkReqSem()

	start := chain.DefaultLedger.Blockchain.LatestLocatorHash(req.Locator)
	hashes, err := GetBlockHashes(*start, req.HashStop, p2p.MaxBlocksPerMsg)
	if err != nil {
		return err
	}

	inv := msg.NewInventory()
	for i := range hashes {
		inv.AddInvVect(msg.NewInvVect(msg.InvTypeBlock, hashes[i]))
	}

	invListLen := len(inv.InvList)
	if invListLen > 0 {
		if invListLen == p2p.MaxBlocksPerMsg {
			continueHash := inv.InvList[invListLen-1].Hash
			h.continueHash = &continueHash
		}
		node.Send(inv)
	}

	return nil
}

func (h *HandlerEBIP001) onInventory(inv *msg.Inventory) error {
	node := h.node
	if LocalNode.IsSyncHeaders() && !node.IsSyncHeaders() {
		return nil
	}

	// Attempt to find the final block in the inventory list.
	// There may not be one.
	lastBlock := -1
	for i := len(inv.InvList) - 1; i >= 0; i-- {
		if inv.InvList[i].Type == msg.InvTypeBlock {
			lastBlock = i
			break
		}
	}

	getData := msg.NewGetData()

	for i, iv := range inv.InvList {
		hash := iv.Hash
		switch iv.Type {
		case msg.InvTypeBlock:
			if node.IsExternal() {
				return fmt.Errorf("receive InvTypeBlock from external node")
			}
			haveInv := chain.DefaultLedger.BlockInLedger(hash) ||
				chain.DefaultLedger.Blockchain.IsKnownOrphan(&hash) || LocalNode.IsRequestedBlock(hash)

			// Block need to be request
			if !haveInv {
				LocalNode.AddRequestedBlock(hash)
				getData.AddInvVect(iv)
				continue
			}

			// Request fork chain
			if chain.DefaultLedger.Blockchain.IsKnownOrphan(&hash) {
				orphanRoot := chain.DefaultLedger.Blockchain.GetOrphanRoot(&hash)
				locator, err := chain.DefaultLedger.Blockchain.LatestBlockLocator()
				if err != nil {
					log.Errorf(" Failed to get block locator for the latest block: %v", err)
					continue
				}
				SendGetBlocks(node, locator, *orphanRoot)
				continue
			}

			// Request next hashes
			if i == lastBlock {
				locator := chain.DefaultLedger.Blockchain.BlockLocatorFromHash(&hash)
				SendGetBlocks(node, locator, common.EmptyHash)
			}
		case msg.InvTypeTx:
			if _, ok := LocalNode.GetTransactionPool(false)[hash]; !ok {
				getData.AddInvVect(iv)
			}
		default:
			continue
		}
	}

	node.Send(getData)
	return nil
}

func (h *HandlerEBIP001) onGetData(getData *msg.GetData) error {
	node := h.node
	notFound := msg.NewNotFound()

	for _, iv := range getData.InvList {
		switch iv.Type {
		case msg.InvTypeBlock:
			block, err := chain.DefaultLedger.Store.GetBlock(iv.Hash)
			if err != nil {
				log.Debug("Can't get block from hash: ", iv.Hash, " ,send not found message")
				notFound.AddInvVect(iv)
				continue
			}
			log.Debug("block height is ", block.Header.Height, " ,hash is ", iv.Hash.String())

			node.Send(msg.NewBlock(block))

			if h.continueHash != nil && h.continueHash.IsEqual(iv.Hash) {
				best := chain.DefaultLedger.Blockchain.BestChain
				inv := msg.NewInventory()
				inv.AddInvVect(msg.NewInvVect(msg.InvTypeBlock, best.Hash))
				node.Send(inv)
				h.continueHash = nil
			}

		case msg.InvTypeTx:
			tx, ok := LocalNode.GetTransactionPool(false)[iv.Hash]
			if !ok {
				notFound.AddInvVect(iv)
				continue
			}

			node.Send(msg.NewTx(tx))

		case msg.InvTypeFilteredBlock:
			if !h.node.BloomFilter().IsLoaded() {
				return nil
			}

			block, err := chain.DefaultLedger.Store.GetBlock(iv.Hash)
			if err != nil {
				log.Debug("Can't get block from hash: ", iv.Hash, " ,send not found message")
				notFound.AddInvVect(iv)
				continue
			}

			merkle, matchedIndexes := bloom.NewMerkleBlock(block, h.node.BloomFilter())

			// Send merkleblock
			node.Send(merkle)

			// Send any matched transactions
			for _, index := range matchedIndexes {
				node.Send(msg.NewTx(block.Transactions[index]))
			}

		default:
			log.Warnf("Unknown type in inventory request %d", iv.Type)
			continue
		}
	}

	if len(notFound.InvList) > 0 {
		node.Send(notFound)
	}

	return nil
}

func (h *HandlerEBIP001) onBlock(msgBlock *msg.Block) error {
	node := h.node
	block := msgBlock.Block.(*core.Block)

	hash := block.Hash()
	if !LocalNode.IsNeighborNode(node.ID()) {
		return fmt.Errorf("receive block message from unknown peer")
	}

	if chain.DefaultLedger.BlockInLedger(hash) {
		return fmt.Errorf("receive duplicated block %s", hash.String())
	}

	// Update sync timer
	LocalNode.syncTimer.update()
	chain.DefaultLedger.Store.RemoveHeaderListElement(hash)
	LocalNode.DeleteRequestedBlock(hash)

	//todo [dpos] only record block (add block only when received confirm block message)
	chain.DefaultLedger.Blockchain.AddUnconfirmedBlock(block)
	if LocalNode.IsSyncHeaders() {
		_, isOrphan, err := chain.DefaultLedger.Blockchain.AddBlock(block)
		if err != nil {
			reject := msg.NewReject(msgBlock.CMD(), msg.RejectInvalid, err.Error())
			reject.Hash = block.Hash()

			node.Send(reject)
			return fmt.Errorf("Block add failed: %s ,block hash %s ", err.Error(), hash.String())
		}

		if isOrphan {
			orphanRoot := chain.DefaultLedger.Blockchain.GetOrphanRoot(&hash)
			locator, _ := chain.DefaultLedger.Blockchain.LatestBlockLocator()
			SendGetBlocks(node, locator, *orphanRoot)
		}
	} else if !LocalNode.IsSyncHeaders() && !LocalNode.ExistedID(hash) {
		LocalNode.Relay(node, block)
		log.Debug("Relay block")

		if h.listener != nil {
			h.listener.OnUnconfirmedBlock(block)
		}

		// Handle orphan blocks.
		prevHash := block.Header.Previous
		if !prevHash.IsEqual(common.EmptyHash) {
			if !chain.DefaultLedger.Blockchain.BlockExists(&prevHash) {
				log.Tracef("Adding orphan block %x with parent %x", block.Hash().Bytes(), prevHash.Bytes())
				chain.DefaultLedger.Blockchain.AddOrphanBlock(block)
				orphanRoot := chain.DefaultLedger.Blockchain.GetOrphanRoot(&hash)
				locator, _ := chain.DefaultLedger.Blockchain.LatestBlockLocator()
				SendGetBlocks(node, locator, *orphanRoot)
				return fmt.Errorf("Block is orphan")
			}
		}
	}
	return nil
}

func (h *HandlerEBIP001) onTx(msgTx *msg.Tx) error {
	node := h.node
	tx := msgTx.Transaction.(*core.Transaction)

	if !LocalNode.IsNeighborNode(node.ID()) {
		return fmt.Errorf("received transaction message from unknown peer")
	}

	if LocalNode.IsSyncHeaders() {
		return nil
	}

	if LocalNode.ExistedID(tx.Hash()) {
		reject := msg.NewReject(msgTx.CMD(), msg.RejectDuplicate, "duplicate transaction")
		reject.Hash = tx.Hash()
		node.Send(reject)
		return fmt.Errorf("[HandlerEBIP001] Transaction already exsisted")
	}

	if errCode := LocalNode.AppendToTxnPool(tx); errCode != errors.Success {
		reject := msg.NewReject(msgTx.CMD(), msg.RejectInvalid, errCode.Message())
		reject.Hash = tx.Hash()
		node.Send(reject)
		return fmt.Errorf("[HandlerEBIP001] VerifyTransaction failed when AppendToTxnPool")
	}

	LocalNode.Relay(node, tx)
	log.Infof("Relay Transaction type %s hash %s", tx.TxType.Name(), tx.Hash().String())
	LocalNode.IncRxTxnCnt()

	return nil
}

func (h *HandlerEBIP001) onNotFound(inv *msg.NotFound) error {
	for _, iv := range inv.InvList {
		log.Warnf("data not found type: %s hash: %s", iv.Type.String(), iv.Hash.String())
	}
	return nil
}

func (h *HandlerEBIP001) onMemPool(*msg.MemPool) error {
	// Only allow mempool requests if server enabled SPV service
	if LocalNode.Services()&protocol.OpenService != protocol.OpenService {
		h.node.CloseConn()
		return fmt.Errorf("peer %d sent mempool request with SPV service disabled", h.node.ID())
	}

	txMemPool := LocalNode.GetTransactionPool(false)
	inv := msg.NewInventory()

	for _, tx := range txMemPool {
		if !h.node.BloomFilter().IsLoaded() || h.node.BloomFilter().MatchTxAndUpdate(tx) {
			txId := tx.Hash()
			inv.AddInvVect(msg.NewInvVect(msg.InvTypeTx, &txId))
		}
	}

	if len(inv.InvList) > 0 {
		h.node.Send(inv)
	}

	return nil
}

func (h *HandlerEBIP001) onReject(msg *msg.Reject) error {
	return fmt.Errorf("Received reject message from peer %d: Code: %s, Hash %s, Reason: %s",
		h.node.ID(), msg.Code.String(), msg.Hash.String(), msg.Reason)
}

func (h *HandlerEBIP001) onConfirmBlock(msg *msg.ConfirmedBlock) error {
	log.Debug("Received confirm block message, hash: ", msg.Proposal.Hash.String())

	//todo [dpos] check confirmed block and add to leger
	block, ok := chain.DefaultLedger.Blockchain.GetUnconfirmedBlock(&msg.Proposal.Hash)
	if !ok {
		return fmt.Errorf("Get unconfirmedBlock failed: block hash %s ", msg.Proposal.Hash)
	}
	chain.DefaultLedger.Blockchain.ClearUnConfirmedBlock()
	isInMainChain, isOrphan, err := chain.DefaultLedger.Blockchain.AddBlock(block)
	if err != nil {
		return fmt.Errorf("Block add failed: %s ,block hash %s ", err.Error(), msg.Proposal.Hash)
	}

	if isOrphan && !LocalNode.IsRequestedBlock(msg.Proposal.Hash) {
		orphanRoot := chain.DefaultLedger.Blockchain.GetOrphanRoot(&msg.Proposal.Hash)
		locator, _ := chain.DefaultLedger.Blockchain.LatestBlockLocator()
		SendGetBlocks(h.node, locator, *orphanRoot)
	}

	if !isInMainChain {
		return fmt.Errorf("Block add failed: not in main chain, block hash %s ", msg.Proposal.Hash)
	}

	log.Info("[OnBlock] Add confirmed block succeed, hash:", block.Hash().String(), "height:", block.Height)
	if h.listener != nil {
		h.listener.OnConfirmedBlock()
	}
	return nil
}
