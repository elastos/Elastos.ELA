package net

import (
	. "ELA/common"
	"ELA/core/ledger"
	"ELA/core/transaction"
	"ELA/crypto"
	. "ELA/errors"
	"ELA/events"
	"ELA/net/node"
	"ELA/net/protocol"
)

type Neter interface {
	GetTxnPool(byCount bool) map[Uint256]*transaction.Transaction
	Xmit(interface{}) error
	GetEvent(eventName string) *events.Event
	GetBookKeepersAddrs() ([]*crypto.PubKey, uint64)
	CleanSubmittedTransactions(block *ledger.Block) error
	GetNeighborNoder() []protocol.Noder
	Tx(buf []byte)
	AppendTxnPool(*transaction.Transaction) ErrCode
	MaybeAcceptTransaction(txn *transaction.Transaction) error
	RemoveTransaction(txn *transaction.Transaction)
}

func StartProtocol(pubKey *crypto.PubKey) protocol.Noder {
	net := node.InitNode(pubKey)
	net.ConnectSeeds()

	return net
}
