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
	"errors"
)

type blocksRequest struct {
	messageHeader
	p struct {
		len       uint32
		hashStart []Uint256
		hashEnd   Uint256
	}
}

func NewBlocksRequest(blocator []Uint256, hash Uint256) ([]byte, error) {
	var msg blocksRequest
	msg.Magic = config.Parameters.Magic
	cmd := "getblocks"
	copy(msg.CMD[0:len(cmd)], cmd)
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
	binary.Read(buf, binary.LittleEndian, &(msg.Checksum))
	msg.Length = uint32(len(p.Bytes()))
	log.Debug("The message payload length is ", msg.Length)

	m, err := msg.Serialization()
	if err != nil {
		log.Error("Error Convert net message ", err.Error())
		return nil, err
	}

	return m, nil
}

func (message blocksRequest) Handle(node Noder) error {
	log.Debug()
	// lock
	node.LocalNode().AcqSyncHdrReqSem()
	defer node.LocalNode().RelSyncHdrReqSem()
	var locatorHash []Uint256
	var startHash [HASHLEN]byte
	var stopHash [HASHLEN]byte
	locatorHash = message.p.hashStart
	stopHash = message.p.hashEnd
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

func (message blocksRequest) Serialization() ([]byte, error) {
	hdrBuf, err := message.messageHeader.Serialization()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(hdrBuf)
	err = binary.Write(buf, binary.LittleEndian, message.p.len)
	if err != nil {
		return nil, err
	}
	for _, hash := range message.p.hashStart {
		hash.Serialize(buf)
	}

	message.p.hashEnd.Serialize(buf)

	return buf.Bytes(), err
}

func (message *blocksRequest) Deserialization(p []byte) error {
	buf := bytes.NewBuffer(p)
	err := binary.Read(buf, binary.LittleEndian, &(message.messageHeader))
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.LittleEndian, &(message.p.len))
	if err != nil {
		return err
	}

	for i := 0; i < int(message.p.len); i++ {
		var hash Uint256
		err := (&hash).Deserialize(buf)
		message.p.hashStart = append(message.p.hashStart, hash)
		if err != nil {
			log.Debug("blkHeader req Deserialization failed")
			goto blksReqErr
		}
	}

	err = message.p.hashEnd.Deserialize(buf)
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
