package net

import (
	. "Elastos.ELA/common"
	"Elastos.ELA/core/ledger"
	"Elastos.ELA/core/transaction"
	"Elastos.ELA/crypto"
	. "Elastos.ELA/errors"
	"Elastos.ELA/events"
	"Elastos.ELA/net/node"
	"Elastos.ELA/net/protocol"
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
