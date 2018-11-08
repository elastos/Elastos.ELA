package node

import (
	"fmt"

	"github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/log"

	"github.com/elastos/Elastos.ELA.Utility/p2p"
	"github.com/elastos/Elastos.ELA.Utility/p2p/msg"
	"github.com/elastos/Elastos.ELA.Utility/p2p/msg/v0"
)

var _ Handler = (*HandlerV0)(nil)

type HandlerV0 struct {
	node            *node
	duplicateBlocks int
}

func NewHandlerV0(node *node) *HandlerV0 {
	return &HandlerV0{node: node}
}

// After message header decoded, this method will be
// called to create the message instance with the CMD
// which is the message type of the received message
func (h *HandlerV0) MakeEmptyMessage(cmd string) (message p2p.Message, err error) {
	switch cmd {
	case p2p.CmdGetBlocks:
		message = &msg.GetBlocks{}

	case p2p.CmdInv:
		message = &v0.Inv{}

	case p2p.CmdGetData:
		message = &v0.GetData{}

	case p2p.CmdBlock:
		message = msg.NewBlock(&core.Block{})

	case p2p.CmdTx:
		message = msg.NewTx(&core.Transaction{})

	case p2p.CmdNotFound:
		message = &v0.NotFound{}

	default:
		err = fmt.Errorf("unhanlded message %s", cmd)
	}

	return message, err
}

// After message has been successful decoded, this method
// will be called to pass the decoded message instance
func (h *HandlerV0) HandleMessage(message p2p.Message) {
	switch message := message.(type) {
	case *msg.GetBlocks:
		h.onGetBlocks(message)

	case *v0.Inv:
		h.onInv(message)

	case *v0.GetData:
		h.onGetData(message)

	case *msg.Block:
		h.onBlock(message)

	case *msg.Tx:
		h.onTx(message)

	case *v0.NotFound:
		h.onNotFound(message)
	}
}

func (h *HandlerV0) onGetBlocks(req *msg.GetBlocks) error {
	node := h.node
	LocalNode.AcqSyncBlkReqSem()
	defer LocalNode.RelSyncBlkReqSem()

	start := chain.LatestLocatorHash(req.Locator)
	hashes, err := GetBlockHashes(*start, req.HashStop, p2p.MaxHeaderHashes)
	if err != nil {
		return err
	}

	if len(hashes) > 0 {
		node.SendMessage(v0.NewInv(hashes))
	}
	return nil
}

func (h *HandlerV0) onInv(inv *v0.Inv) error {
	log.Debugf("[OnInv] count %d hashes", len(inv.Hashes))

	node := h.node
	if node.IsExternal() {
		return fmt.Errorf("receive inv message from external node")
	}

	if syncNode != nil && node != syncNode {
		return nil
	}

	for i, hash := range inv.Hashes {
		// Request block
		if !chain.BlockExists(hash) &&
			(!chain.IsKnownOrphan(hash) || !LocalNode.IsRequestedBlock(*hash)) {

			LocalNode.AddRequestedBlock(*hash)
			node.SendMessage(v0.NewGetData(*hash))
		}

		// Request fork chain
		if chain.IsKnownOrphan(hash) {
			orphanRoot := chain.GetOrphanRoot(hash)
			locator, err := chain.LatestBlockLocator()
			if err != nil {
				log.Errorf("Failed to get block locator for the latest block: %v", err)
				continue
			}
			node.PushGetBlocksMsg(locator, orphanRoot)
			continue
		}

		// Request next hashes
		if i == len(inv.Hashes)-1 {
			locator := chain.BlockLocatorFromHash(hash)
			node.PushGetBlocksMsg(locator, &zeroHash)
		}
	}
	return nil
}

func (h *HandlerV0) onGetData(req *v0.GetData) error {
	node := h.node
	hash := req.Hash

	block, err := store.GetBlock(hash)
	if err != nil {
		log.Debugf("Can't get block from hash %s, send not found message", hash)
		node.SendMessage(v0.NewNotFound(hash))
		return err
	}

	node.SendMessage(msg.NewBlock(block))

	return nil
}

func (h *HandlerV0) onBlock(msgBlock *msg.Block) error {
	node := h.node
	block := msgBlock.Serializable.(*core.Block)

	hash := block.Hash()
	if !IsNeighborNode(node.ID()) {
		log.Debug("received block message from unknown peer")
		return fmt.Errorf("received block message from unknown peer")
	}

	if chain.BlockExists(&hash) {
		h.duplicateBlocks++
		log.Debug("Receive ", h.duplicateBlocks, " duplicated block.")
		return fmt.Errorf("received duplicated block")
	}

	// Update sync timer
	node.stallTimer.update()
	store.RemoveHeaderListElement(hash)
	LocalNode.DeleteRequestedBlock(hash)
	_, isOrphan, err := chain.AddBlock(block)
	if err != nil {
		return fmt.Errorf("Block add failed: %s ,block hash %s ", err.Error(), hash.String())
	}

	if syncNode == nil {
		// relay
		if !LocalNode.ExistedID(hash) {
			LocalNode.Relay(node, block)
			log.Debug("Relay block")
		}

		if isOrphan && !LocalNode.IsRequestedBlock(hash) {
			orphanRoot := chain.GetOrphanRoot(&hash)
			locator, _ := chain.LatestBlockLocator()
			node.PushGetBlocksMsg(locator, orphanRoot)
		}
	}

	return nil
}

func (h *HandlerV0) onTx(msgTx *msg.Tx) error {
	node := h.node
	tx := msgTx.Serializable.(*core.Transaction)

	if !LocalNode.ExistedID(tx.Hash()) && syncNode == nil {
		if errCode := LocalNode.AppendToTxnPool(tx); errCode != errors.Success {
			return fmt.Errorf("[HandlerBase] VerifyTransaction failed when AppendToTxnPool")
		}
		LocalNode.Relay(node, tx)
		log.Debugf("Relay Transaction hash %s type %s", tx.Hash().String(), tx.TxType.Name())
		LocalNode.IncRxTxnCnt()
	}

	return nil
}

func (h *HandlerV0) onNotFound(msg *v0.NotFound) error {
	log.Debug("Received not found message, hash: ", msg.Hash.String())
	return nil
}
