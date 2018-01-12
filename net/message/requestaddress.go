package message

import (
	"Elastos.ELA/common/log"
	. "Elastos.ELA/net/protocol"
	"encoding/hex"
)

type addressesRequest struct {
	messageHeader
	// No payload
}

func NewGetAddr() ([]byte, error) {
	var message addressesRequest
	// Fixme the check is the []byte{0} instead of 0
	var sum []byte
	sum = []byte{0x5d, 0xf6, 0xe0, 0xe2}
	message.init("getaddr", sum, 0)

	buf, err := message.Serialization()
	if err != nil {
		return nil, err
	}

	str := hex.EncodeToString(buf)
	log.Debug("The message get addr length is: ", len(buf), " ", str)

	return buf, err
}

func (message addressesRequest) Handle(node Noder) error {
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
