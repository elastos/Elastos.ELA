package protocol

import (
	"fmt"
	"net"
	"time"

	"github.com/elastos/Elastos.ELA/bloom"
	"github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/errors"

	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/p2p"
	"github.com/elastos/Elastos.ELA.Utility/p2p/msg"
)

const (
	ProtocolVersion    = 0
	MinConnectionCount = 3
	MaxSyncHdrReq      = 2 //Max Concurrent Sync Header Request
	MaxOutboundCount   = 8
	DefaultMaxPeers    = 125
	MaxIDCached        = 5000
)

const (
	OpenService uint64 = 1 << 2
)

type State int32

const (
	INIT State = iota
	HAND
	HANDSHAKE
	HANDSHAKED
	ESTABLISHED
	INACTIVITY
)

var States = map[State]string{
	INIT:        "INIT",
	HAND:        "HAND",
	HANDSHAKE:   "HANDSHAKE",
	HANDSHAKED:  "HANDSHAKED",
	ESTABLISHED: "ESTABLISHED",
	INACTIVITY:  "INACTIVITY",
}

func (s State) String() string {
	state, ok := States[s]
	if ok {
		return state
	}
	return fmt.Sprintf("STATE%d", s)
}

type Noder interface {
	Version() uint32
	ID() uint64
	Services() uint64
	Addr() string
	NA() *p2p.NetAddress
	Port() uint16
	IsExternal() bool
	SetState(state State)
	State() State
	IsRelay() bool
	Height() uint64
	GetConn() net.Conn
	Connected() bool
	Disconnect()
	GetTransactionPool(bool) map[common.Uint256]*core.Transaction
	AppendToTxnPool(*core.Transaction) errors.ErrCode
	IsDuplicateSidechainTx(sidechainTxHash common.Uint256) bool
	ExistedID(id common.Uint256) bool
	LoadFilter(filter *msg.FilterLoad)
	BloomFilter() *bloom.Filter
	SendMessage(msg p2p.Message)
	GetTransaction(hash common.Uint256) *core.Transaction
	IncRxTxnCnt()
	GetTxnCnt() uint64
	GetRxTxnCnt() uint64

	CleanSubmittedTransactions(block *core.Block) error
	MaybeAcceptTransaction(txn *core.Transaction) error
	RemoveTransaction(txn *core.Transaction)

	SetHeight(height uint64)
	Relay(Noder, interface{}) error
	IsSyncHeaders() bool
	SetSyncHeaders(b bool)
	IsRequestedBlock(hash common.Uint256) bool
	AddRequestedBlock(hash common.Uint256)
	DeleteRequestedBlock(hash common.Uint256)
	GetRequestBlockList() map[common.Uint256]time.Time
	AcqSyncBlkReqSem()
	RelSyncBlkReqSem()
	SetStartHash(hash common.Uint256)
	GetStartHash() common.Uint256
	SetStopHash(hash common.Uint256)
	GetStopHash() common.Uint256
	ResetRequestedBlock()
}
