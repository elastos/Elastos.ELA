package message

import (
	"bytes"
	"encoding/binary"
	"errors"
	. "github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core/ledger"
	. "github.com/elastos/Elastos.ELA/net/protocol"
)

type blocksReq struct {
	messageHeader
	p struct {
		len       uint32
		hashStart []Uint256
		hashEnd   Uint256
	}
}

func (msg blocksReq) Handle(node Noder) error {
	log.Debug()
	// lock
	node.LocalNode().AcqSyncHdrReqSem()
	defer node.LocalNode().RelSyncHdrReqSem()
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
	hdrBuf, err := msg.messageHeader.Serialization()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(hdrBuf)
	err = binary.Write(buf, binary.LittleEndian, msg.p.len)
	if err != nil {
		return nil, err
	}
	for _, hash := range msg.p.hashStart {
		hash.Serialize(buf)
	}

	msg.p.hashEnd.Serialize(buf)

	return buf.Bytes(), err
}

func (msg *blocksReq) Deserialization(p []byte) error {
	buf := bytes.NewBuffer(p)
	err := binary.Read(buf, binary.LittleEndian, &(msg.messageHeader))
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
		hash.Serialize(tmpBuffer)
	}

	return &InvPayload{
		Cnt: count,
		Blk: tmpBuffer.Bytes(),
	}, nil
}
