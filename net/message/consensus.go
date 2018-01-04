package message

import (
	"Elastos.ELA/common"
	"Elastos.ELA/core/contract/program"
	"Elastos.ELA/crypto"
	. "Elastos.ELA/net/protocol"
)

type ConsensusPayload struct {
	Version         uint32
	PrevHash        common.Uint256
	Height          uint32
	BookKeeperIndex uint16
	Timestamp       uint32
	Data            []byte
	Owner           *crypto.PubKey
	Program         *program.Program

	hash common.Uint256
}

func reqConsensusData(node Noder, hash common.Uint256) error {
	var msg dataReq
	msg.dataType = common.CONSENSUS
	// TODO handle the hash array case
	msg.hash = hash

	buf, _ := msg.Serialization()
	go node.Tx(buf)

	return nil
}
func (cp *ConsensusPayload) Type() common.InventoryType {

	//TODO:Temporary add for Interface signature.SignableData use.
	return common.CONSENSUS
}
