package message

import (
	. "DNA_POW/common"
	"DNA_POW/common/log"
	"DNA_POW/common/serialization"
	"DNA_POW/core/ledger"
	. "DNA_POW/net/protocol"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

type blocksReq struct {
	hdr msgHdr
	p   struct {
		len       uint32
		hashStart []Uint256
		hashEnd   Uint256
	}
}

type InvPayload struct {
	InvType InventoryType
	Cnt     uint32
	Blk     []byte
}

type Inv struct {
	Hdr msgHdr
	P   InvPayload
}

func SendMsgSyncBlockHeaders(node Noder) {
	var emptyHash Uint256
	currentHash := ledger.DefaultLedger.Store.GetCurrentHeaderHash()
	blocator := ledger.DefaultLedger.Blockchain.BlockLocatorFromHash(&currentHash)
	buf, err := NewBlocksReq(blocator, emptyHash)
	if err != nil {
		log.Error("failed build a new getblocksReq")
	} else {
		node.LocalNode().SetSyncHeaders(true)
		node.SetSyncHeaders(true)
		log.Trace("Sync from ", node.GetAddr())
		go node.Tx(buf)
		node.StartRetryTimer()
	}
}

func ReqBlksHdrFromOthers(node Noder) {
	//node.SetSyncFailed()
	noders := node.LocalNode().GetNeighborNoder()
	for _, noder := range noders {
		if noder.IsSyncFailed() != true {
			SendMsgSyncBlockHeaders(noder)
			break
		}
	}
}

func NewBlocksReq(blocator []Uint256, hash Uint256) ([]byte, error) {
	var msg blocksReq
	msg.hdr.Magic = NETMAGIC
	cmd := "getblocks"
	copy(msg.hdr.CMD[0:len(cmd)], cmd)
	tmpBuffer := bytes.NewBuffer([]byte{})
	msg.p.len = uint32(len(blocator))
	msg.p.hashStart = blocator
	serialization.WriteUint32(tmpBuffer, uint32(msg.p.len))

	for _, hash := range blocator {
		_, err := hash.Serialize(tmpBuffer)
		if err != nil {
			return nil, err
		}
	}

	msg.p.hashEnd = hash

	_, err := msg.p.hashEnd.Serialize(tmpBuffer)
	if err != nil {
		return nil, err
	}
	p := new(bytes.Buffer)
	err = binary.Write(p, binary.LittleEndian, tmpBuffer.Bytes())
	if err != nil {
		log.Error("Binary Write failed at new Msg")
		return nil, err
	}
	s := sha256.Sum256(p.Bytes())
	s2 := s[:]
	s = sha256.Sum256(s2)
	buf := bytes.NewBuffer(s[:4])
	binary.Read(buf, binary.LittleEndian, &(msg.hdr.Checksum))
	msg.hdr.Length = uint32(len(p.Bytes()))
	log.Debug("The message payload length is ", msg.hdr.Length)

	m, err := msg.Serialization()
	if err != nil {
		log.Error("Error Convert net message ", err.Error())
		return nil, err
	}

	return m, nil
}

func (msg blocksReq) Verify(buf []byte) error {

	// TODO verify the message Content
	err := msg.hdr.Verify(buf)
	return err
}

func (msg blocksReq) Handle(node Noder) error {
	log.Debug()
	log.Trace("handle blocks request")
	// lock
	node.LocalNode().AcqSyncReqSem()
	defer node.LocalNode().RelSyncReqSem()
	var locatorHash []Uint256
	var startHash [HASHLEN]byte
	var stopHash [HASHLEN]byte
	locatorHash = msg.p.hashStart
	stopHash = msg.p.hashEnd

	startHash = ledger.DefaultLedger.Blockchain.LatestLocatorHash(locatorHash)
	inv, err := GetInvFromBlockHash(startHash, stopHash)
	if err != nil {
		return err
	}
	buf, err := NewInv(inv)
	if err != nil {
		return err
	}
	go node.Tx(buf)
	return nil
}

func (msg blocksReq) Serialization() ([]byte, error) {
	hdrBuf, err := msg.hdr.Serialization()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(hdrBuf)
	err = binary.Write(buf, binary.LittleEndian, msg.p.len)
	if err != nil {
		return nil, err
	}
	//err = binary.Write(buf, binary.LittleEndian, msg.p.hashStart)
	for _, hash := range msg.p.hashStart {
		hash.Serialize(buf)
	}

	msg.p.hashEnd.Serialize(buf)

	return buf.Bytes(), err
}

func (msg *blocksReq) Deserialization(p []byte) error {
	buf := bytes.NewBuffer(p)
	err := binary.Read(buf, binary.LittleEndian, &(msg.hdr))
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.LittleEndian, &(msg.p.len))
	if err != nil {
		return err
	}

	for i := 0; i < int(msg.p.len); i++ {
		var hash Uint256
		err := (&hash).Deserialize(buf)
		msg.p.hashStart = append(msg.p.hashStart, hash)
		if err != nil {
			log.Debug("blkHeader req Deserialization failed")
			goto blksReqErr
		}
	}

	err = msg.p.hashEnd.Deserialize(buf)
blksReqErr:
	return err
}

func (msg Inv) Verify(buf []byte) error {
	// TODO verify the message Content
	err := msg.Hdr.Verify(buf)
	return err
}

func (msg Inv) Handle(node Noder) error {
	log.Debug()
	var id Uint256
	str := hex.EncodeToString(msg.P.Blk)

	log.Info(fmt.Sprintf("The inv type: 0x%x block len: %d, %s\n",
		msg.P.InvType, len(msg.P.Blk), str))

	invType := InventoryType(msg.P.InvType)
	switch invType {
	case TRANSACTION:
		log.Debug("RX TRX message")
		// TODO check the ID queue
		id.Deserialize(bytes.NewReader(msg.P.Blk[:32]))
		if !node.ExistedID(id) {
			reqTxnData(node, id)
		}
	case BLOCK:
		log.Debug("RX block message")
		node.StopRetryTimer()
		if node.LocalNode().IsSyncHeaders() == true && node.IsSyncHeaders() == false {
			return nil
		}
		var i uint32
		count := msg.P.Cnt
		hashes := []Uint256{}
		for i = 0; i < count; i++ {
			id.Deserialize(bytes.NewReader(msg.P.Blk[HASHLEN*i:]))
			hashes = append(hashes, id)
			if ledger.DefaultLedger.Blockchain.IsKnownOrphan(&id) {
				orphanRoot := ledger.DefaultLedger.Blockchain.GetOrphanRoot(&id)
				locator, err := ledger.DefaultLedger.Blockchain.LatestBlockLocator()
				if err != nil {
					log.Errorf(" Failed to get block "+
						"locator for the latest block: "+
						"%v", err)
					continue
				}
				buf, err := NewBlocksReq(locator, *orphanRoot)
				if err != nil {
					log.Error("failed build a new getblocksReq")
					continue
				} else {
					go node.Tx(buf)
				}
				continue
			}
			if i == (count - 1) {
				var emptyHash Uint256
				blocator := ledger.DefaultLedger.Blockchain.BlockLocatorFromHash(&id)
				buf, err := NewBlocksReq(blocator, emptyHash)
				if err != nil {
					log.Error("failed build a new getblocksReq")
				} else {
					go node.Tx(buf)
				}
			}
		}
		for _, h := range hashes {
			// TODO check the ID queue
			if !ledger.DefaultLedger.BlockInLedger(h) {
				node.CacheHash(id) //cached hash would not relayed
				if !node.LocalNode().ExistedID(h) && !node.LocalNode().RequestedBlockExisted(h) {
					// send the block request
					node.LocalNode().AddRequestedBlock(h)
					ReqBlkData(node, h)
				}
			}
		}
	case CONSENSUS:
		log.Debug("RX consensus message")
		id.Deserialize(bytes.NewReader(msg.P.Blk[:32]))
		reqConsensusData(node, id)
	default:
		log.Warn("RX unknown inventory message")
	}
	return nil
}

func (msg Inv) Serialization() ([]byte, error) {
	hdrBuf, err := msg.Hdr.Serialization()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(hdrBuf)
	msg.P.Serialization(buf)

	return buf.Bytes(), err
}

func (msg *Inv) Deserialization(p []byte) error {
	err := msg.Hdr.Deserialization(p)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(p[MSGHDRLEN:])
	invType, err := serialization.ReadUint8(buf)
	if err != nil {
		return err
	}
	msg.P.InvType = InventoryType(invType)
	msg.P.Cnt, err = serialization.ReadUint32(buf)
	if err != nil {
		return err
	}

	msg.P.Blk = make([]byte, msg.P.Cnt*HASHLEN)
	err = binary.Read(buf, binary.LittleEndian, &(msg.P.Blk))

	return err
}

func (msg Inv) invType() InventoryType {
	return msg.P.InvType
}

func GetInvFromBlockHash(startHash Uint256, stopHash Uint256) (*InvPayload, error) {
	var count uint32 = 0
	var empty Uint256
	var startHeight uint32
	var stopHeight uint32
	curHeight := ledger.DefaultLedger.Store.GetHeight()
	if stopHash == empty {
		if startHash == empty {
			if curHeight > MAXINVHDRCNT {
				count = MAXINVHDRCNT
			} else {
				count = curHeight
			}
		} else {
			bkstart, err := ledger.DefaultLedger.Store.GetHeader(startHash)
			if err != nil {
				return nil, err
			}
			startHeight = bkstart.Blockdata.Height
			count = curHeight - startHeight
			if count > MAXINVHDRCNT {
				count = MAXINVHDRCNT
			}
		}
	} else {
		bkstop, err := ledger.DefaultLedger.Store.GetHeader(stopHash)
		if err != nil {
			return nil, err
		}
		stopHeight = bkstop.Blockdata.Height
		if startHash != empty {
			bkstart, err := ledger.DefaultLedger.Store.GetHeader(startHash)
			if err != nil {
				return nil, err
			}
			startHeight = bkstart.Blockdata.Height

			// avoid unsigned integer underflow
			if stopHeight < startHeight {
				return nil, errors.New("do not have header to send")
			}
			count = stopHeight - startHeight

			if count >= MAXINVHDRCNT {
				count = MAXINVHDRCNT
			}
		} else {
			if stopHeight > MAXINVHDRCNT {
				count = MAXINVHDRCNT
			} else {
				count = stopHeight
			}
		}
	}

	tmpBuffer := bytes.NewBuffer([]byte{})
	var i uint32
	for i = 1; i <= count; i++ {
		//FIXME need add error handle for GetBlockWithHash
		hash, _ := ledger.DefaultLedger.Store.GetBlockHash(startHeight + i)
		log.Debug("GetInvFromBlockHash i is ", i, " , hash is ", hash)
		hash.Serialize(tmpBuffer)
	}
	log.Debug("GetInvFromBlockHash hash is ", tmpBuffer.Bytes())
	return NewInvPayload(BLOCK, count, tmpBuffer.Bytes()), nil
}

func NewInvPayload(invType InventoryType, count uint32, msg []byte) *InvPayload {
	return &InvPayload{
		InvType: invType,
		Cnt:     count,
		Blk:     msg,
	}
}

func NewInv(inv *InvPayload) ([]byte, error) {
	var msg Inv
	msg.P.Blk = inv.Blk
	msg.P.InvType = inv.InvType
	msg.P.Cnt = inv.Cnt
	msg.Hdr.Magic = NETMAGIC
	cmd := "inv"
	copy(msg.Hdr.CMD[0:len(cmd)], cmd)
	tmpBuffer := bytes.NewBuffer([]byte{})
	inv.Serialization(tmpBuffer)

	b := new(bytes.Buffer)
	err := binary.Write(b, binary.LittleEndian, tmpBuffer.Bytes())
	if err != nil {
		log.Error("Binary Write failed at new Msg", err.Error())
		return nil, err
	}
	s := sha256.Sum256(b.Bytes())
	s2 := s[:]
	s = sha256.Sum256(s2)
	buf := bytes.NewBuffer(s[:4])
	binary.Read(buf, binary.LittleEndian, &(msg.Hdr.Checksum))
	msg.Hdr.Length = uint32(len(b.Bytes()))

	m, err := msg.Serialization()
	if err != nil {
		log.Error("Error Convert net message ", err.Error())
		return nil, err
	}

	return m, nil
}

func (msg *InvPayload) Serialization(w io.Writer) {
	serialization.WriteUint8(w, uint8(msg.InvType))
	serialization.WriteUint32(w, msg.Cnt)

	binary.Write(w, binary.LittleEndian, msg.Blk)
}
