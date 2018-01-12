package message

import (
	"Elastos.ELA/common/log"
	. "Elastos.ELA/net/protocol"
	"bytes"
	"encoding/binary"
	"encoding/hex"
)

type addrReq struct {
	messageHeader
}

func newGetAddr() ([]byte, error) {
	var msg addrReq
	// Fixme the check is the []byte{0} instead of 0
	var sum []byte
	sum = []byte{0x5d, 0xf6, 0xe0, 0xe2}
	msg.init("getaddr", sum, 0)

	buf, err := msg.Serialization()
	if err != nil {
		return nil, err
	}

	str := hex.EncodeToString(buf)
	log.Debug("The message get addr length is: ", len(buf), " ", str)

	return buf, err
}

func (msg addrReq) Handle(node Noder) error {
	log.Debug()
	// lock
	var addrstr []NodeAddr
	var count uint64

	addrstr = node.LocalNode().RandSelectAddresses()
	count = uint64(len(addrstr))
	buf, err := NewAddrs(addrstr, count)
	if err != nil {
		return err
	}
	go node.Tx(buf)
	return nil
}

func (msg addrReq) Serialization() ([]byte, error) {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.LittleEndian, msg)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), err
}

func (msg *addrReq) Deserialization(p []byte) error {
	buf := bytes.NewBuffer(p)
	err := binary.Read(buf, binary.LittleEndian, msg)
	return err
}
