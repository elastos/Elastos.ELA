package message

import (
	"Elastos.ELA/common"
	"Elastos.ELA/common/config"
	"Elastos.ELA/common/log"
	"Elastos.ELA/core/ledger"
	. "Elastos.ELA/net/protocol"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
)

type block struct {
	messageHeader
	blk ledger.Block
	// TBD
	//event *events.Event
}

func (msg block) Handle(node Noder) error {
	hash := msg.blk.Hash()
	//node.LocalNode().AcqSyncBlkReqSem()
	//defer node.LocalNode().RelSyncBlkReqSem()
	//log.Tracef("hash is %x", hash.ToArrayReverse())
	if node.LocalNode().IsNeighborNoder(node) == false {
		log.Trace("received headers message from unknown peer")
		return errors.New("received headers message from unknown peer")
	}

	if ledger.DefaultLedger.BlockInLedger(hash) {
		ReceiveDuplicateBlockCnt++
		log.Trace("Receive ", ReceiveDuplicateBlockCnt, " duplicated block.")
		return nil
	}

	isFastAdd := false

	if node.LocalNode().GetHeaderFisrtModeStatus() {
		preHash, err := ledger.DefaultLedger.Store.GetHeaderHashFront()
		if err == nil {
			if bytes.Equal(hash[:], preHash[:]) {
				isFastAdd = true
			}
		}
	}

	ledger.DefaultLedger.Store.RemoveHeaderListElement(hash)
	node.LocalNode().DeleteRequestedBlock(hash)
	isOrphan := false
	var err error
	if isFastAdd {
		_, isOrphan, err = ledger.DefaultLedger.Blockchain.AddBlockFast(&msg.blk)
	} else {
		_, isOrphan, err = ledger.DefaultLedger.Blockchain.AddBlock(&msg.blk)
	}

	if err != nil {
		log.Warn("Block add failed: ", err, " ,block hash is ", hash.ToArrayReverse())
		return err
	}
	//relay
	if node.LocalNode().IsSyncHeaders() == false {
		if !node.LocalNode().ExistedID(hash) {
			node.LocalNode().Relay(node, &msg.blk)
			log.Debug("Relay block")
		}
	}

	if isOrphan == true && node.LocalNode().IsSyncHeaders() == false {
		if !node.LocalNode().RequestedBlockExisted(hash) {
			orphanRoot := ledger.DefaultLedger.Blockchain.GetOrphanRoot(&hash)
			locator, _ := ledger.DefaultLedger.Blockchain.LatestBlockLocator()
			SendMsgSyncBlockHeaders(node, locator, *orphanRoot)
		}
	}

	// Nothing more to do if we aren't in headers-first mode.
	if !node.LocalNode().GetHeaderFisrtModeStatus() {
		return nil
	}

	requestedBlocks := node.LocalNode().GetRequestBlockList()
	if err == nil && len(requestedBlocks) < MinInFlightBlocks {
		fetchHeaderBlocks(node)
	}
	return nil
}

func fetchHeaderBlocks(node Noder) {
	// Nothing to do if there is no start header.
	preHash, err := ledger.DefaultLedger.Store.GetHeaderHashFront()
	if err != nil {
		log.Warn("fetchHeaderBlocks called with no start header")
		return
	}

	for {
		err := ReqBlkData(node, preHash)
		if err != nil {
			log.Error("failed build a new getdata")
		}
		nextHash, erro := ledger.DefaultLedger.Store.GetHeaderHashNext(preHash)
		if erro != nil {
			break
		} else {
			preHash = nextHash
		}
	}
}

func (msg dataReq) Handle(node Noder) error {
	log.Debug()
	reqtype := InventoryType(msg.dataType)
	hash := msg.hash
	switch reqtype {
	case Block:
		block, err := NewBlockFromHash(hash)
		if err != nil {
			log.Debug("Can't get block from hash: ", hash, " ,send not found message")
			//call notfound message
			b, err := NewNotFound(hash)
			node.Tx(b)
			return err
		}
		log.Debug("block height is ", block.Blockdata.Height, " ,hash is ", hash)
		buf, err := NewBlock(block)
		if err != nil {
			return err
		}
		node.Tx(buf)

	case Transaction:
		txn, err := NewTxnFromHash(hash)
		if err != nil {
			return err
		}
		buf, err := NewTxn(txn)
		if err != nil {
			return err
		}
		go node.Tx(buf)
	}
	return nil
}

func NewBlockFromHash(hash common.Uint256) (*ledger.Block, error) {
	bk, err := ledger.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		log.Errorf("Get Block error: %s, block hash: %x", err.Error(), hash)
		return nil, err
	}
	return bk, nil
}

func NewBlock(bk *ledger.Block) ([]byte, error) {
	log.Debug()
	var msg block
	msg.blk = *bk
	msg.messageHeader.Magic = config.Parameters.Magic
	cmd := "block"
	copy(msg.messageHeader.CMD[0:len(cmd)], cmd)
	tmpBuffer := bytes.NewBuffer([]byte{})
	bk.Serialize(tmpBuffer)
	p := new(bytes.Buffer)
	err := binary.Write(p, binary.LittleEndian, tmpBuffer.Bytes())
	if err != nil {
		log.Error("Binary Write failed at new Msg")
		return nil, err
	}
	s := sha256.Sum256(p.Bytes())
	s2 := s[:]
	s = sha256.Sum256(s2)
	buf := bytes.NewBuffer(s[:4])
	binary.Read(buf, binary.LittleEndian, &(msg.messageHeader.Checksum))
	msg.messageHeader.Length = uint32(len(p.Bytes()))
	log.Debug("The message payload length is ", msg.messageHeader.Length)

	m, err := msg.Serialization()
	if err != nil {
		log.Error("Error Convert net message ", err.Error())
		return nil, err
	}

	return m, nil
}

func ReqBlkData(node Noder, hash common.Uint256) error {
	node.LocalNode().AddRequestedBlock(hash)
	var msg dataReq
	msg.dataType = Block
	msg.hash = hash

	msg.messageHeader.Magic = config.Parameters.Magic
	copy(msg.messageHeader.CMD[0:7], "getdata")
	p := bytes.NewBuffer([]byte{})
	err := binary.Write(p, binary.LittleEndian, &(msg.dataType))
	msg.hash.Serialize(p)
	if err != nil {
		log.Error("Binary Write failed at new getdata Msg")
		return err
	}
	s := sha256.Sum256(p.Bytes())
	s2 := s[:]
	s = sha256.Sum256(s2)
	buf := bytes.NewBuffer(s[:4])
	binary.Read(buf, binary.LittleEndian, &(msg.messageHeader.Checksum))
	msg.messageHeader.Length = uint32(len(p.Bytes()))
	log.Debug("The message payload length is ", msg.messageHeader.Length)

	sendBuf, err := msg.Serialization()
	if err != nil {
		log.Error("Error Convert net message ", err.Error())
		return err
	}

	node.Tx(sendBuf)

	return nil
}

func (msg block) Verify(buf []byte) error {
	err := msg.messageHeader.Verify(buf)
	// TODO verify the message Content
	return err
}

func (msg block) Serialization() ([]byte, error) {
	hdrBuf, err := msg.messageHeader.Serialization()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(hdrBuf)
	msg.blk.Serialize(buf)

	return buf.Bytes(), err
}

func (msg *block) Deserialization(p []byte) error {
	buf := bytes.NewBuffer(p)

	err := binary.Read(buf, binary.LittleEndian, &(msg.messageHeader))
	if err != nil {
		log.Warn("Parse block message hdr error")
		return errors.New("Parse block message hdr error")
	}

	err = msg.blk.Deserialize(buf)
	if err != nil {
		log.Warn("Parse block message error")
		return errors.New("Parse block message error")
	}

	return err
}
