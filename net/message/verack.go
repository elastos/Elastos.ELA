package message

import (
	"encoding/hex"
	"errors"
	"strconv"

	"github.com/elastos/Elastos.ELA/common/log"
	. "github.com/elastos/Elastos.ELA/net/protocol"
)

type verACK struct {
	messageHeader
	// No payload
}

func NewVerack() ([]byte, error) {
	var msg verACK
	// Fixme the check is the []byte{0} instead of 0
	var sum []byte
	sum = []byte{0x5d, 0xf6, 0xe0, 0xe2}
	msg.messageHeader.init("verack", sum, 0)

	buf, err := msg.Serialization()
	if err != nil {
		return nil, err
	}

	str := hex.EncodeToString(buf)
	log.Debug("The message tx verack length is ", len(buf), ", ", str)

	return buf, err
}

/*
 * The node state switch table after rx message, there is time limitation for each action
 * The Hanshake status will switch to Init after TIMEOUT if not received the VerACK
 * in this time window
 *  _______________________________________________________________________
 * |          |    Init         | HandShake |  Establish | Inactive      |
 * |-----------------------------------------------------------------------|
 * | version  | HandShake(timer)|           |            | HandShake(timer)|
 * |          | if helloTime > 3| Tx verack | Depend on  | if helloTime > 3|
 * |          | Tx version      |           | node update| Tx version      |
 * |          | then Tx verack  |           |            | then Tx verack  |
 * |-----------------------------------------------------------------------|
 * | verack   |                 | Establish |            |                 |
 * |          |   No Action     |           | No Action  | No Action       |
 * |------------------------------------------------------------------------
 *
 */
// TODO The process should be adjusted based on above table
func (msg verACK) Handle(node Noder) error {

	s := node.GetState()
	if s != HandShake && s != HandShaked {
		log.Warn("Unknow status to received verack")
		return errors.New("Unknow status to received verack")
	}

	node.SetState(Establish)

	if s == HandShake {
		buf, _ := NewVerack()
		node.Tx(buf)
	}

	//node.DumpInfo()
	// Fixme, there is a race condition here,
	// but it doesn't matter to access the invalid
	// node which will trigger a warning
	if node.LocalNode().NeedMoreAddresses() {
		node.ReqNeighborList()
	}
	addr := node.GetAddr()
	port := node.GetPort()
	nodeAddr := addr + ":" + strconv.Itoa(int(port))
	node.LocalNode().RemoveAddrInConnectingList(nodeAddr)
	return nil
}
