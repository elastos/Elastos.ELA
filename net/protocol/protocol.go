package protocol

import (
	"Elastos.ELA/common"
	"Elastos.ELA/core/ledger"
	"Elastos.ELA/core/transaction"
	"Elastos.ELA/crypto"
	. "Elastos.ELA/errors"
	"Elastos.ELA/events"
	"bytes"
	"encoding/binary"
	"net"
	"time"
)

type NodeAddr struct {
	Time     int64
	Services uint64
	IpAddr   [16]byte
	Port     uint16
	ID       uint64 // Unique ID
}

// The node capability type
const (
	VERIFYNODE  = 1
	SERVICENODE = 2
)

const (
	VERIFYNODENAME  = "verify"
	SERVICENODENAME = "service"
)

const (
	MSGCMDLEN         = 12
	CMDOFFSET         = 4
	CHECKSUMLEN       = 4
	HASHLEN           = 32 // hash length in byte
	MSGHDRLEN         = 24
	NETMAGIC          = 0x74746e41
	MAXBLKHDRCNT      = 400
	MAXINVHDRCNT      = 50
	DIVHASHLEN        = 5
	MINCONNCNT        = 3
	MAXREQBLKONCE     = 16
	TIMESOFUPDATETIME = 2
	MAXCACHEHASH      = 16
)

const (
	HELLOTIMEOUT         = 3 // Seconds
	MAXHELLORETYR        = 3
	MAXBUFLEN            = 1024 * 16 // Fixme The maximum buffer to receive message
	MAXCHANBUF           = 512
	PROTOCOLVERSION      = 0
	PERIODUPDATETIME     = 3 // Time to update and sync information with other nodes
	HEARTBEAT            = 2
	KEEPALIVETIMEOUT     = 3
	DIALTIMEOUT          = 6
	CONNMONITOR          = 6
	CONNMAXBACK          = 4000
	MAXRETRYCOUNT        = 3
	MAXSYNCHDRREQ        = 2 //Max Concurrent Sync Header Request
	NEEDADDRESSTHRESHOLD = 1000
	MAXOUTBOUNDCNT       = 8
	DEFAULTMAXPEERS      = 125
	GETADDRMAX           = 2500
	MAXIDCACHED          = 5000
	MAXINVCACHEHASH      = 50000
	MinInFlightBlocks    = 10
)

// The node state
const (
	INIT       = 0
	HAND       = 1
	HANDSHAKE  = 2
	HANDSHAKED = 3
	ESTABLISH  = 4
	INACTIVITY = 5
)

var ReceiveDuplicateBlockCnt uint64 //an index to detecting networking status

type Noder interface {
	Version() uint32
	GetID() uint64
	Services() uint64
	GetAddr() string
	GetAddr16() ([16]byte, error)
	GetPort() uint16
	GetHttpInfoPort() int
	SetHttpInfoPort(uint16)
	GetHttpInfoState() bool
	SetHttpInfoState(bool)
	GetState() uint32
	GetRelay() bool
	SetState(state uint32)
	GetPubKey() *crypto.PubKey
	CompareAndSetState(old, new uint32) bool
	UpdateRXTime(t time.Time)
	LocalNode() Noder
	DelNbrNode(id uint64) (Noder, bool)
	AddNbrNode(Noder)
	CloseConn()
	GetHeight() uint64
	GetConnectionCnt() uint
	GetConn() net.Conn
	GetTxnPool(bool) map[common.Uint256]*transaction.Transaction
	AppendTxnPool(*transaction.Transaction) ErrCode
	ExistedID(id common.Uint256) bool
	ReqNeighborList()
	DumpInfo()
	UpdateInfo(t time.Time, version uint32, services uint64,
		port uint16, nonce uint64, relay uint8, height uint64)
	ConnectSeeds()
	Connect(nodeAddr string) error
	Tx(buf []byte)
	GetTime() int64
	NodeEstablished(uid uint64) bool
	GetEvent(eventName string) *events.Event
	GetNeighborAddrs() ([]NodeAddr, uint64)
	GetTransaction(hash common.Uint256) *transaction.Transaction
	IncRxTxnCnt()
	GetTxnCnt() uint64
	GetRxTxnCnt() uint64

	Xmit(interface{}) error
	GetNeighborHeights() ([]uint64, uint64)
	SyncNodeHeight()
	CleanSubmittedTransactions(block *ledger.Block) error
	MaybeAcceptTransaction(txn *transaction.Transaction) error
	RemoveTransaction(txn *transaction.Transaction)

	GetNeighborNoder() []Noder
	GetNbrNodeCnt() uint32
	StoreFlightHeight(height uint32)
	GetFlightHeightCnt() int
	RemoveFlightHeightLessThan(height uint32)
	RemoveFlightHeight(height uint32)
	GetLastRXTime() time.Time
	SetHeight(height uint64)
	WaitForFourPeersStart()
	GetFlightHeights() []uint32
	IsAddrInNbrList(addr string) bool
	SetAddrInConnectingList(addr string) bool
	RemoveAddrInConnectingList(addr string)
	AddInRetryList(addr string)
	RemoveFromRetryList(addr string)
	GetAddressCnt() uint64
	AddAddressToKnownAddress(na NodeAddr)
	RandGetAddresses(nbrAddrs []NodeAddr) []NodeAddr
	GetDefaultMaxPeers() uint
	GetMaxOutboundCnt() uint
	GetGetAddrMax() uint
	NeedMoreAddresses() bool
	RandSelectAddresses() []NodeAddr
	UpdateLastDisconn(id uint64)
	Relay(Noder, interface{}) error
	ExistHash(hash common.Uint256) bool
	CacheHash(hash common.Uint256)
	ExistFlightHeight(height uint32) bool
	IsSyncHeaders() bool
	SetSyncHeaders(b bool)
	IsSyncFailed() bool
	SetSyncFailed()
	StartSync()
	CacheInvHash(hash common.Uint256)
	ExistInvHash(hash common.Uint256) bool
	DeleteInvHash(hash common.Uint256)
	GetHeaderFisrtModeStatus() bool
	RequestedBlockExisted(hash common.Uint256) bool
	AddRequestedBlock(hash common.Uint256)
	DeleteRequestedBlock(hash common.Uint256)
	GetRequestBlockList() map[common.Uint256]time.Time
	IsNeighborNoder(n Noder) bool
	GetNextCheckpoint() *Checkpoint
	FindNextHeaderCheckpoint(height uint64) *Checkpoint
	GetNextCheckpointHeight() (uint64, error)
	GetNextCheckpointHash() (common.Uint256, error)
	SetHeaderFirstMode(b bool)
	FindSyncNode() (Noder, error)
	GetStartSync() bool
	GetBestHeightNoder() Noder
	AcqSyncBlkReqSem()
	RelSyncBlkReqSem()
	AcqSyncHdrReqSem()
	RelSyncHdrReqSem()
	SetStartHash(hash common.Uint256)
	GetStartHash() common.Uint256
	SetStopHash(hash common.Uint256)
	GetStopHash() common.Uint256
	ResetRequestedBlock()
}

// Checkpoint identifies a known good point in the block chain.
type Checkpoint struct {
	Height uint64
	Hash   common.Uint256
}

func (msg *NodeAddr) Deserialization(p []byte) error {
	buf := bytes.NewBuffer(p)
	err := binary.Read(buf, binary.LittleEndian, msg)
	return err
}

func (msg NodeAddr) Serialization() ([]byte, error) {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.LittleEndian, msg)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), err
}
