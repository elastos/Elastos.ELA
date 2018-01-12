package message

import (
	"Elastos.ELA/common"
	"Elastos.ELA/common/log"
	"Elastos.ELA/core/ledger"
	. "Elastos.ELA/net/protocol"
	"bytes"
	"encoding/binary"
	"errors"
)

type dataRequest struct {
	messageHeader
	hash common.Uint256
}

func (message dataRequest) Handle(node Noder) error {
	log.Debug()
	hash := message.hash
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

//what the hell begin
func (message dataRequest) Serialization() ([]byte, error) {
	hdrBuf, err := message.messageHeader.Serialization()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(hdrBuf)
	if err != nil {
		return nil, err
	}
	message.hash.Serialize(buf)

	return buf.Bytes(), err
}

func (message *dataRequest) Deserialization(p []byte) error {
	buf := bytes.NewBuffer(p)
	err := binary.Read(buf, binary.LittleEndian, &(message.messageHeader))
	if err != nil {
		log.Warn("Parse datareq message hdr error")
		return errors.New("Parse datareq message hdr error")
	}

	if err != nil {
		log.Warn("Parse datareq message dataType error")
		return errors.New("Parse datareq message dataType error")
	}

	err = message.hash.Deserialize(buf)
	if err != nil {
		log.Warn("Parse datareq message hash error")
		return errors.New("Parse datareq message hash error")
	}
	return nil
}

//what the hell end
