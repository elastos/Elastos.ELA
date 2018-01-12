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
	block ledger.Block
}

func (message block) Handle(node Noder) error {
	hash := message.block.Hash()
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
		_, isOrphan, err = ledger.DefaultLedger.Blockchain.AddBlockFast(&message.block)
	} else {
		_, isOrphan, err = ledger.DefaultLedger.Blockchain.AddBlock(&message.block)
	}

	if err != nil {
		log.Warn("Block add failed: ", err, " ,block hash is ", hash.ToArrayReverse())
		return err
	}
	//relay
	if node.LocalNode().IsSyncHeaders() == false {
		if !node.LocalNode().ExistedID(hash) {
			node.LocalNode().Relay(node, &message.block)
			log.Debug("Relay block")
		}
	}

	if isOrphan == true && node.LocalNode().IsSyncHeaders() == false {
		if !node.LocalNode().RequestedBlockExisted(hash) {
			orphanRoot := ledger.DefaultLedger.Blockchain.GetOrphanRoot(&hash)
			locator, _ := ledger.DefaultLedger.Blockchain.LatestBlockLocator()
			SendMessageSyncBlockHeaders(node, locator, *orphanRoot)
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
		err := RequestBlockData(node, preHash)
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

func NewBlock(bk *ledger.Block) ([]byte, error) {
	log.Debug()
	var msg block
	msg.block = *bk
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

func RequestBlockData(node Noder, hash common.Uint256) error {
	node.LocalNode().AddRequestedBlock(hash)
	var msg dataRequest
	msg.hash = hash

	msg.messageHeader.Magic = config.Parameters.Magic
	copy(msg.messageHeader.CMD[0:7], "getdata")
	p := bytes.NewBuffer([]byte{})
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

func (message block) Serialization() ([]byte, error) {
	hdrBuf, err := message.messageHeader.Serialization()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(hdrBuf)
	message.block.Serialize(buf)

	return buf.Bytes(), err
}

func (message *block) Deserialization(p []byte) error {
	buf := bytes.NewBuffer(p)

	err := binary.Read(buf, binary.LittleEndian, &(message.messageHeader))
	if err != nil {
		log.Warn("Parse block message hdr error")
		return errors.New("Parse block message hdr error")
	}

	err = message.block.Deserialize(buf)
	if err != nil {
		log.Warn("Parse block message error")
		return errors.New("Parse block message error")
	}

	return err
}
