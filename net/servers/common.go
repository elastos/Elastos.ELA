package servers

import (
	"github.com/elastos/Elastos.ELA.Core/common/log"
	tx "github.com/elastos/Elastos.ELA.Core/core/transaction"
	. "github.com/elastos/Elastos.ELA.Core/net/protocol"
	. "github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/core/asset"
	. "github.com/elastos/Elastos.ELA.Utility/core/transaction"
	uti_payload "github.com/elastos/Elastos.ELA.Utility/core/transaction/payload"
	. "github.com/elastos/Elastos.ELA.Utility/errors"
	"github.com/elastos/Elastos.ELA/consensus/pow"
	"github.com/elastos/Elastos.ELA/core/transaction/payload"
)

const TlsPort = 443

var NodeForServers Noder
var Pow *pow.PowService

type TxAttributeInfo struct {
	Usage TransactionAttributeUsage
	Data  string
}

type UTXOTxInputInfo struct {
	ReferTxID          string
	ReferTxOutputIndex uint16
	Sequence           uint32
	Address            string
	Value              string
}

type BalanceTxInputInfo struct {
	AssetID     string
	Value       Fixed64
	ProgramHash string
}

type TxoutputInfo struct {
	AssetID    string
	Value      string
	Address    string
	OutputLock uint32
}

type TxoutputMap struct {
	Key   Uint256
	Txout []TxoutputInfo
}

type AmountMap struct {
	Key   Uint256
	Value Fixed64
}

type ProgramInfo struct {
	Code      string
	Parameter string
}

type Transactions struct {
	TxType         TransactionType
	PayloadVersion byte
	Payload        PayloadInfo
	Attributes     []TxAttributeInfo
	UTXOInputs     []UTXOTxInputInfo
	BalanceInputs  []BalanceTxInputInfo
	Outputs        []TxoutputInfo
	LockTime       uint32
	Programs       []ProgramInfo

	AssetOutputs      []TxoutputMap
	AssetInputAmount  []AmountMap
	AssetOutputAmount []AmountMap
	Timestamp         uint32 `json:",omitempty"`
	Confirmations     uint32 `json:",omitempty"`
	TxSize            uint32 `json:",omitempty"`
	Hash              string
}

type AuxInfo struct {
	Version    int32
	PrevBlock  string
	MerkleRoot string
	Timestamp  uint32
	Bits       uint32
	Nonce      uint32
}

type BlockHead struct {
	Version          uint32
	PrevBlockHash    string
	TransactionsRoot string
	Timestamp        uint32
	Bits             uint32
	Height           uint32
	Nonce            uint32
	AuxPow           *AuxInfo
	Difficulty       string
	BlockSize        int
	Hash             string
}

type BlockInfo struct {
	Hash          string
	BlockData     *BlockHead
	Transactions  []*Transactions
	Confirmations uint32
	MinerInfo     string
}

type NodeInfo struct {
	State    uint   // NodeForServers status
	Port     uint16 // The nodes's port
	ID       uint64 // The nodes's id
	Time     int64
	Version  uint32 // The network protocol the NodeForServers used
	Services uint64 // The services the NodeForServers supplied
	Relay    bool   // The relay capability of the NodeForServers (merge into capbility flag)
	Height   uint64 // The NodeForServers latest block height
	TxnCnt   uint64 // The transactions be transmit by this NodeForServers
	RxTxnCnt uint64 // The transaction received by this NodeForServers
}

type ArbitratorGroupInfo struct {
	OnDutyArbitratorIndex int
	Arbitrators           []string
}

type PayloadInfo interface{}

type CoinbaseInfo struct {
	CoinbaseData string
}

type RegisterAssetInfo struct {
	Asset      *asset.Asset
	Amount     string
	Controller string
}

type SideMiningInfo struct {
	SideBlockHash string
}

type TransferCrossChainAssetInfo struct {
	AddressesMap map[string]uint64
}

type WithdrawTokenInfo struct {
	BlockHeight uint32
}

func TransPayloadToHex(p Payload) PayloadInfo {
	switch object := p.(type) {
	case *uti_payload.CoinBase:
		obj := new(CoinbaseInfo)
		obj.CoinbaseData = string(object.CoinbaseData)
		return obj
	case *uti_payload.RegisterAsset:
		obj := new(RegisterAssetInfo)
		obj.Asset = object.Asset
		obj.Amount = object.Amount.String()
		obj.Controller = BytesToHexString(object.Controller.ToArrayReverse())
		return obj
	case *payload.SideMining:
		obj := new(SideMiningInfo)
		obj.SideBlockHash = object.SideBlockHash.String()
		return obj
	case *payload.WithdrawToken:
		obj := new(WithdrawTokenInfo)
		obj.BlockHeight = object.BlockHeight
		return obj
	case *payload.TransferCrossChainAsset:
		obj := new(TransferCrossChainAssetInfo)
		obj.AddressesMap = object.AddressesMap
		return obj
	case *uti_payload.TransferAsset:
	case *uti_payload.Record:
	case *uti_payload.DeployCode:
	}
	return nil
}

func VerifyAndSendTx(txn *tx.NodeTransaction) ErrCode {
	// if transaction is verified unsucessfully then will not put it into transaction pool
	if errCode := NodeForServers.AppendToTxnPool(txn); errCode != Success {
		log.Warn("Can NOT add the transaction to TxnPool")
		log.Info("[httpjsonrpc] VerifyTransaction failed when AppendToTxnPool.")
		return errCode
	}
	if err := NodeForServers.Xmit(txn); err != nil {
		log.Error("Xmit Tx Error:Xmit transaction failed.", err)
		return ErrXmitFail
	}
	return Success
}

func ResponsePack(errCode ErrCode, result interface{}) map[string]interface{} {
	if errCode != 0 && (result == "" || result == nil) {
		result = ErrMap[errCode]
	}
	return map[string]interface{}{"Result": result, "Error": errCode}
}
