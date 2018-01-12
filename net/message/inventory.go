package message

import (
	. "Elastos.ELA/common"
	"Elastos.ELA/common/config"
	"Elastos.ELA/common/log"
	"Elastos.ELA/common/serialization"
	"Elastos.ELA/core/ledger"
	. "Elastos.ELA/net/protocol"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"time"
)

type InvPayload struct {
	Cnt uint32
	Blk []byte
}

type Inv struct {
	messageHeader
	P   InvPayload
}

func SendMessageSyncBlockHeaders(node Noder, blocator []Uint256, hash Uint256) {
	if node.LocalNode().GetStartHash() == blocator[0] &&
		node.LocalNode().GetStopHash() == hash {
		return
	}

	buf, err := NewBlocksRequest(blocator, hash)
	if err != nil {
		log.Error("failed build a new getblocksReq")
	} else {
		node.LocalNode().SetSyncHeaders(true)
		node.SetSyncHeaders(true)
		go node.Tx(buf)
		node.LocalNode().SetStartHash(blocator[0])
		node.LocalNode().SetStopHash(hash)
	}
}

func (msg Inv) Serialization() ([]byte, error) {
	hdrBuf, err := msg.Serialization()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(hdrBuf)
	msg.P.Serialization(buf)

	return buf.Bytes(), err
}

func NewInv(inv *InvPayload) ([]byte, error) {
	var msg Inv
	msg.P.Blk = inv.Blk
	msg.P.Cnt = inv.Cnt
	msg.Magic = config.Parameters.Magic
	cmd := "inv"
	copy(msg.CMD[0:len(cmd)], cmd)
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
	binary.Read(buf, binary.LittleEndian, &(msg.Checksum))
	msg.Length = uint32(len(b.Bytes()))

	m, err := msg.Serialization()
	if err != nil {
		log.Error("Error Convert net message ", err.Error())
		return nil, err
	}

	return m, nil
}

func (msg *InvPayload) Serialization(w io.Writer) {
	serialization.WriteUint32(w, msg.Cnt)
	binary.Write(w, binary.LittleEndian, msg.Blk)
}

func (msg Inv) Handle(node Noder) error {
	var id Uint256
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
			SendMessageSyncBlockHeaders(node, locator, *orphanRoot)
			continue
		}

		if i == (count - 1) {
			var emptyHash Uint256
			blocator := ledger.DefaultLedger.Blockchain.BlockLocatorFromHash(&id)
			SendMessageSyncBlockHeaders(node, blocator, emptyHash)
		}
	}
	for _, h := range hashes {
		// TODO check the ID queue
		if !ledger.DefaultLedger.BlockInLedger(h) {
			if !(node.LocalNode().RequestedBlockExisted(h) || ledger.DefaultLedger.Blockchain.IsKnownOrphan(&h)) {
				<-time.After(time.Millisecond * 50)
				RequestBlockData(node, h)
			}
		}
	}
	return nil
}