package message

import (
	"DNA_POW/common"
	"DNA_POW/common/config"
	"DNA_POW/common/log"
	"DNA_POW/common/serialization"
	"DNA_POW/core/ledger"
	. "DNA_POW/net/protocol"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
)

type headersReq struct {
	hdr msgHdr
	p   struct {
		len       uint32
		hashStart []common.Uint256
		hashEnd   common.Uint256
	}
}

type blkHeader struct {
	hdr    msgHdr
	cnt    uint32
	blkHdr []ledger.Header
}

func NewHeadersReq(startHash common.Uint256, stopHash common.Uint256) ([]byte, error) {
	var msg headersReq
	msg.hdr.Magic = config.Parameters.Magic
	cmd := "getheaders"
	copy(msg.hdr.CMD[0:len(cmd)], cmd)
	tmpBuffer := bytes.NewBuffer([]byte{})
	blocator := ledger.DefaultLedger.Blockchain.BlockLocatorFromHash(&startHash)

	msg.p.len = uint32(len(blocator))

	msg.p.hashStart = blocator
	serialization.WriteUint32(tmpBuffer, uint32(msg.p.len))

	for _, hash := range blocator {
		_, err := hash.Serialize(tmpBuffer)
		if err != nil {
			return nil, err
		}
	}

	msg.p.hashEnd = stopHash

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

func (msg headersReq) Verify(buf []byte) error {
	// TODO Verify the message Content
	err := msg.hdr.Verify(buf)
	return err
}

func (msg blkHeader) Verify(buf []byte) error {
	// TODO Verify the message Content
	err := msg.hdr.Verify(buf)
	return err
}

func (msg headersReq) Serialization() ([]byte, error) {
	hdrBuf, err := msg.hdr.Serialization()
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

func (msg *headersReq) Deserialization(p []byte) error {
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
		var hash common.Uint256
		err := (&hash).Deserialize(buf)
		msg.p.hashStart = append(msg.p.hashStart, hash)
		if err != nil {
			log.Debug("blkHeader req Deserialization failed")
			goto blkHdrReqErr
		}
	}

	err = msg.p.hashEnd.Deserialize(buf)
blkHdrReqErr:
	return err
}

func (msg blkHeader) Serialization() ([]byte, error) {
	hdrBuf, err := msg.hdr.Serialization()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(hdrBuf)
	err = binary.Write(buf, binary.LittleEndian, msg.cnt)
	if err != nil {
		return nil, err
	}

	for _, header := range msg.blkHdr {
		header.Serialize(buf)
	}
	return buf.Bytes(), err
}

func (msg *blkHeader) Deserialization(p []byte) error {
	buf := bytes.NewBuffer(p)
	err := binary.Read(buf, binary.LittleEndian, &(msg.hdr))
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.LittleEndian, &(msg.cnt))
	if err != nil {
		return err
	}

	for i := 0; i < int(msg.cnt); i++ {
		var headers ledger.Header
		err := (&headers).Deserialize(buf)
		msg.blkHdr = append(msg.blkHdr, headers)
		if err != nil {
			log.Debug("blkHeader Deserialization failed")
			goto blkHdrErr
		}
	}

blkHdrErr:
	return err
}

func (msg headersReq) Handle(node Noder) error {
	log.Debug()
	// lock
	var locatorHash []common.Uint256
	var startHash [HASHLEN]byte
	var stopHash [HASHLEN]byte
	locatorHash = msg.p.hashStart
	stopHash = msg.p.hashEnd

	startHash = ledger.DefaultLedger.Blockchain.LatestLocatorHash(locatorHash)
	headers, cnt, err := GetHeadersFromHash(startHash, stopHash)
	if err != nil {
		return err
	}
	buf, err := NewHeaders(headers, cnt)
	if err != nil {
		return err
	}

	go node.Tx(buf)
	return nil
}

func SendMsgSyncHeaders(node Noder, startHash common.Uint256) {
	nextCheckpointHash, err := node.LocalNode().GetNextCheckpointHash()
	var emptyHash common.Uint256
	if (bytes.Equal(emptyHash[:], nextCheckpointHash[:])) || (bytes.Equal(startHash[:], nextCheckpointHash[:])) {
		log.Debug("no more checkpoint")
		node.LocalNode().SetHeaderFirstMode(false)
		blocator := ledger.DefaultLedger.Blockchain.BlockLocatorFromHash(&startHash)
		SendMsgSyncBlockHeaders(node, blocator, emptyHash)
		return
	}
	buf, err := NewHeadersReq(startHash, nextCheckpointHash)
	if err != nil {
		log.Error("failed build a new headersReq")
	} else {
		node.LocalNode().SetSyncHeaders(true)
		node.SetSyncHeaders(true)
		go node.Tx(buf)
	}
}

func (msg blkHeader) sendGetDataReq(node Noder) {
	for _, header := range msg.blkHdr {
		hash := header.Blockdata.Hash()
		if !ledger.DefaultLedger.BlockInLedger(hash) {
			err := ReqBlkData(node, hash)
			if err != nil {
				log.Error("failed build a new getdata")
			}
		}
	}
}

func ReqBlkHdrFromOthers(node Noder) {
	//node.SetSyncFailed()
	n := node.LocalNode().GetBestHeightNoder()
	hash := ledger.DefaultLedger.Store.GetCurrentBlockHash()
	SendMsgSyncHeaders(n, hash)
}

func (msg blkHeader) Handle(node Noder) error {
	log.Debug()
	//If received headers message from unknown peer, return
	if node.LocalNode().IsNeighborNoder(node) == false {
		return errors.New("received headers message from unknown peer")
	}
	if node.LocalNode().GetHeaderFisrtModeStatus() == false {
		node.SetState(INACTIVITY)
		conn := node.GetConn()
		conn.Close()
		return errors.New("Not in header first mode")
	}
	if len(msg.blkHdr) == 0 {
		return errors.New("No headers")
	}

	err := ledger.DefaultLedger.Store.AddHeaders(msg.blkHdr, ledger.DefaultLedger)
	if err != nil {
		log.Warn("Add block Header error")
		node.SetState(INACTIVITY)
		conn := node.GetConn()
		conn.Close()
		return errors.New("Add block Header error, send new header request to another node\n")
	}
	receivedCheckpoint := false
	nextCheckpointHeight, err := node.LocalNode().GetNextCheckpointHeight()
	nextCheckpointHash, err := node.LocalNode().GetNextCheckpointHash()
	for i := 0; i < len(msg.blkHdr); i++ {
		if err == nil {
			if uint64(msg.blkHdr[i].Blockdata.Height) == nextCheckpointHeight {
				msgBlkHash := msg.blkHdr[i].Blockdata.Hash()
				if bytes.Equal(msgBlkHash[:], nextCheckpointHash[:]) == true {
					receivedCheckpoint = true
				} else {
					node.SetState(INACTIVITY)
					conn := node.GetConn()
					conn.Close()
				}
				break
			}
		}
	}
	if receivedCheckpoint {
		fetchHeaderBlocks(node)
		return nil
	}

	//msg.sendGetDataReq(node)
	lastHeaderHash := msg.blkHdr[len(msg.blkHdr)-1].Blockdata.Hash()
	SendMsgSyncHeaders(node, lastHeaderHash)

	return nil
}

func GetHeadersFromHash(startHash common.Uint256, stopHash common.Uint256) ([]ledger.Header, uint32, error) {
	var count uint32 = 0
	var empty [HASHLEN]byte
	headers := []ledger.Header{}
	var startHeight uint32
	var stopHeight uint32
	curHeight := ledger.DefaultLedger.Store.GetHeight()
	if stopHash == empty {
		if startHash == empty {
			if curHeight > MAXBLKHDRCNT {
				count = MAXBLKHDRCNT
			} else {
				count = curHeight
			}
		} else {
			bkstart, err := ledger.DefaultLedger.Store.GetHeader(startHash)
			if err != nil {
				log.Error("GetHeader(startHash) err ", err)
				return nil, 0, err
			}
			startHeight = bkstart.Blockdata.Height
			count = curHeight - startHeight
			if count > MAXBLKHDRCNT {
				count = MAXBLKHDRCNT
			}
		}
	} else {
		bkstop, err := ledger.DefaultLedger.Store.GetHeader(stopHash)
		if err != nil {
			log.Error("GetHeader(stopHash) err ", err)
			return nil, 0, err
		}
		stopHeight = bkstop.Blockdata.Height
		if startHash != empty {
			bkstart, err := ledger.DefaultLedger.Store.GetHeader(startHash)
			if err != nil {
				return nil, 0, err
			}
			startHeight = bkstart.Blockdata.Height

			// avoid unsigned integer underflow
			if stopHeight < startHeight {
				return nil, 0, errors.New("do not have header to send")
			}
			count = stopHeight - startHeight

			if count >= MAXBLKHDRCNT {
				count = MAXBLKHDRCNT
			}
		} else {
			if stopHeight > MAXBLKHDRCNT {
				count = MAXBLKHDRCNT
			} else {
				count = stopHeight
			}
		}
	}

	var i uint32
	for i = 1; i <= count; i++ {
		hash, err := ledger.DefaultLedger.Store.GetBlockHash(startHeight + i)
		hd, err := ledger.DefaultLedger.Store.GetHeader(hash)
		if err != nil {
			log.Error("GetBlockWithHeight failed ", err.Error())
			return nil, 0, err
		}
		headers = append(headers, *hd)
	}

	return headers, count, nil
}

func NewHeaders(headers []ledger.Header, count uint32) ([]byte, error) {
	var msg blkHeader
	msg.cnt = count
	msg.blkHdr = headers
	msg.hdr.Magic = config.Parameters.Magic
	cmd := "headers"
	copy(msg.hdr.CMD[0:len(cmd)], cmd)

	tmpBuffer := bytes.NewBuffer([]byte{})
	serialization.WriteUint32(tmpBuffer, msg.cnt)
	for _, header := range headers {
		header.Serialize(tmpBuffer)
	}
	b := new(bytes.Buffer)
	err := binary.Write(b, binary.LittleEndian, tmpBuffer.Bytes())
	if err != nil {
		log.Error("Binary Write failed at new Msg")
		return nil, err
	}
	s := sha256.Sum256(b.Bytes())
	s2 := s[:]
	s = sha256.Sum256(s2)
	buf := bytes.NewBuffer(s[:4])
	binary.Read(buf, binary.LittleEndian, &(msg.hdr.Checksum))
	msg.hdr.Length = uint32(len(b.Bytes()))

	m, err := msg.Serialization()
	if err != nil {
		log.Error("Error Convert net message ", err.Error())
		return nil, err
	}
	return m, nil
}
