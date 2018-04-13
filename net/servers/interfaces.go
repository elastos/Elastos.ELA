package servers

import (
	"bytes"
	"fmt"
	"time"

	"strconv"

	. "github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/core/transaction/payload"
	. "github.com/elastos/Elastos.ELA.Utility/errors"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core/auxpow"
	"github.com/elastos/Elastos.ELA/core/ledger"
	tx "github.com/elastos/Elastos.ELA/core/transaction"
)

const (
	AUXBLOCK_GENERATED_INTERVAL_SECONDS = 60
)

var PreChainHeight uint64
var PreTime int64
var PreTransactionCount int

func TransArrayByteToHexString(ptx *tx.NodeTransaction) *Transactions {

	trans := new(Transactions)
	trans.TxType = ptx.TxType
	trans.PayloadVersion = ptx.PayloadVersion
	trans.Payload = TransPayloadToHex(ptx.Payload)

	n := 0
	trans.Attributes = make([]TxAttributeInfo, len(ptx.Attributes))
	for _, v := range ptx.Attributes {
		trans.Attributes[n].Usage = v.Usage
		trans.Attributes[n].Data = BytesToHexString(v.Data)
		n++
	}

	n = 0
	isCoinbase := ptx.IsCoinBaseTx()
	reference, _ := ptx.GetReference()
	trans.UTXOInputs = make([]UTXOTxInputInfo, len(ptx.UTXOInputs))
	for _, v := range ptx.UTXOInputs {
		trans.UTXOInputs[n].ReferTxID = BytesToHexString(v.ReferTxID.ToArrayReverse())
		trans.UTXOInputs[n].ReferTxOutputIndex = v.ReferTxOutputIndex
		trans.UTXOInputs[n].Sequence = v.Sequence
		if isCoinbase {
			trans.UTXOInputs[n].Address = ""
			trans.UTXOInputs[n].Value = ""
		} else {
			prevOutput := reference[v]
			trans.UTXOInputs[n].Address, _ = prevOutput.ProgramHash.ToAddress()
			trans.UTXOInputs[n].Value = prevOutput.Value.String()
		}
		n++
	}

	n = 0
	trans.BalanceInputs = make([]BalanceTxInputInfo, len(ptx.BalanceInputs))
	for _, v := range ptx.BalanceInputs {
		trans.BalanceInputs[n].AssetID = BytesToHexString(v.AssetID.ToArrayReverse())
		trans.BalanceInputs[n].Value = v.Value
		trans.BalanceInputs[n].ProgramHash = BytesToHexString(v.ProgramHash.ToArrayReverse())
		n++
	}

	n = 0
	trans.Outputs = make([]TxoutputInfo, len(ptx.Outputs))
	for _, v := range ptx.Outputs {
		trans.Outputs[n].AssetID = BytesToHexString(v.AssetID.ToArrayReverse())
		trans.Outputs[n].Value = v.Value.String()
		address, _ := v.ProgramHash.ToAddress()
		trans.Outputs[n].Address = address
		trans.Outputs[n].OutputLock = v.OutputLock
		n++
	}

	n = 0
	trans.Programs = make([]ProgramInfo, len(ptx.Programs))
	for _, v := range ptx.Programs {
		trans.Programs[n].Code = BytesToHexString(v.Code)
		trans.Programs[n].Parameter = BytesToHexString(v.Parameter)
		n++
	}

	n = 0
	trans.AssetOutputs = make([]TxoutputMap, len(ptx.AssetOutputs))
	for k, v := range ptx.AssetOutputs {
		trans.AssetOutputs[n].Key = k
		trans.AssetOutputs[n].Txout = make([]TxoutputInfo, len(v))
		for m := 0; m < len(v); m++ {
			trans.AssetOutputs[n].Txout[m].AssetID = BytesToHexString(v[m].AssetID.ToArrayReverse())
			trans.AssetOutputs[n].Txout[m].Value = v[m].Value.String()
			address, _ := v[m].ProgramHash.ToAddress()
			trans.AssetOutputs[n].Txout[m].Address = address
			trans.AssetOutputs[n].Txout[m].OutputLock = v[m].OutputLock
		}
		n += 1
	}

	trans.LockTime = ptx.LockTime

	n = 0
	trans.AssetInputAmount = make([]AmountMap, len(ptx.AssetInputAmount))
	for k, v := range ptx.AssetInputAmount {
		trans.AssetInputAmount[n].Key = k
		trans.AssetInputAmount[n].Value = v
		n += 1
	}

	n = 0
	trans.AssetOutputAmount = make([]AmountMap, len(ptx.AssetOutputAmount))
	for k, v := range ptx.AssetOutputAmount {
		trans.AssetInputAmount[n].Key = k
		trans.AssetInputAmount[n].Value = v
		n += 1
	}

	mHash := ptx.Hash()
	trans.Hash = BytesToHexString(mHash.ToArrayReverse())

	return trans
}

func checkParam(param map[string]interface{}, keys ...string) bool {
	if param == nil {
		return false
	}
	if len(param) < len(keys) {
		return false
	}
	for _, key := range keys {
		value, ok := param[key]
		if !ok {
			return false
		}
		switch value.(type) {
		case string:
		default:
			return false
		}
	}
	return true
}

// Input JSON string examples for getblock method as following:
func GetRawTransaction(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "hash") {
		return ResponsePack(InvalidParams, "")
	}

	str := param["hash"].(string)
	hex, err := HexStringToBytesReverse(str)
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}
	var hash Uint256
	err = hash.Deserialize(bytes.NewReader(hex))
	if err != nil {
		return ResponsePack(InvalidTransaction, "")
	}
	tx, height, err := ledger.DefaultLedger.Store.GetTransaction(hash)
	if err != nil {
		return ResponsePack(UnknownTransaction, "")
	}
	bHash, err := ledger.DefaultLedger.Store.GetBlockHash(height)
	if err != nil {
		return ResponsePack(UnknownTransaction, "")
	}
	header, err := ledger.DefaultLedger.Store.GetHeader(bHash)
	if err != nil {
		return ResponsePack(UnknownTransaction, "")
	}
	tran := TransArrayByteToHexString(tx)
	tran.Timestamp = header.Blockdata.Timestamp
	tran.Confirmations = ledger.DefaultLedger.Blockchain.GetBestHeight() - height + 1
	w := bytes.NewBuffer(nil)
	tx.Serialize(w)
	tran.TxSize = uint32(len(w.Bytes()))

	return ResponsePack(Success, tran)
}

func GetNeighbors(param map[string]interface{}) map[string]interface{} {
	addr, _ := NodeForServers.GetNeighborAddrs()
	return ResponsePack(Success, addr)
}

func GetNodeState(param map[string]interface{}) map[string]interface{} {
	n := NodeInfo{
		State:    uint(NodeForServers.GetState()),
		Time:     NodeForServers.GetTime(),
		Port:     NodeForServers.GetPort(),
		ID:       NodeForServers.GetID(),
		Version:  NodeForServers.Version(),
		Services: NodeForServers.Services(),
		Relay:    NodeForServers.GetRelay(),
		Height:   NodeForServers.GetHeight(),
		TxnCnt:   NodeForServers.GetTxnCnt(),
		RxTxnCnt: NodeForServers.GetRxTxnCnt(),
	}
	return ResponsePack(Success, n)
}

func SetLogLevel(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "level") {
		return ResponsePack(InvalidParams, "")
	}

	level, err := strconv.ParseInt(param["level"].(string), 10, 64)
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}

	if err := log.Log.SetDebugLevel(int(level)); err != nil {
		return ResponsePack(InvalidParams, err.Error())
	}
	return ResponsePack(Success, "")
}

func SubmitAuxBlock(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "blockhash", "auxpow") {
		return ResponsePack(InvalidParams, "")
	}

	blockHash := param["blockhash"].(string)
	if _, ok := Pow.MsgBlock.BlockData[blockHash]; !ok {
		log.Trace("[json-rpc:SubmitAuxBlock] receive invalid block hash value:", blockHash)
		return ResponsePack(InvalidParams, "")
	}

	auxPow := param["auxpow"].(string)
	temp, _ := HexStringToBytes(auxPow)
	r := bytes.NewBuffer(temp)
	Pow.MsgBlock.BlockData[blockHash].Blockdata.AuxPow.Deserialize(r)
	_, _, err := ledger.DefaultLedger.Blockchain.AddBlock(Pow.MsgBlock.BlockData[blockHash])
	if err != nil {
		log.Trace(err)
		return ResponsePack(InternalError, "")
	}

	Pow.MsgBlock.Mutex.Lock()
	for key := range Pow.MsgBlock.BlockData {
		delete(Pow.MsgBlock.BlockData, key)
	}
	Pow.MsgBlock.Mutex.Unlock()
	log.Trace("AddBlock called finished and Pow.MsgBlock.BlockData has been deleted completely")

	log.Info(auxPow, blockHash)
	return ResponsePack(Success, "")
}

func GenerateAuxBlock(addr string) (*ledger.Block, string, bool) {
	msgBlock := &ledger.Block{}

	if NodeForServers.GetHeight() == 0 || PreChainHeight != NodeForServers.GetHeight() || (time.Now().Unix()-PreTime > AUXBLOCK_GENERATED_INTERVAL_SECONDS) {
		if PreChainHeight != NodeForServers.GetHeight() {
			PreChainHeight = NodeForServers.GetHeight()
			PreTime = time.Now().Unix()
			PreTransactionCount = Pow.GetTransactionCount()
		}

		currentTxsCount := Pow.CollectTransactions(msgBlock)
		if 0 == currentTxsCount {
			// return nil, "currentTxs is nil", false
		}

		msgBlock, err := Pow.GenerateBlock(addr)
		if nil != err {
			return nil, "msgBlock generate err", false
		}

		curHash := msgBlock.Hash()
		curHashStr := BytesToHexString(curHash.ToArray())

		Pow.MsgBlock.Mutex.Lock()
		Pow.MsgBlock.BlockData[curHashStr] = msgBlock
		Pow.MsgBlock.Mutex.Unlock()

		PreChainHeight = NodeForServers.GetHeight()
		PreTime = time.Now().Unix()
		PreTransactionCount = currentTxsCount // Don't Call GetTransactionCount()

		return msgBlock, curHashStr, true
	}
	return nil, "", false
}

func CreateAuxBlock(param map[string]interface{}) map[string]interface{} {
	msgBlock, curHashStr, _ := GenerateAuxBlock(config.Parameters.PowConfiguration.PayToAddr)
	if nil == msgBlock {
		return ResponsePack(UnknownBlock, "")
	}

	if !checkParam(param, "paytoaddress") {
		return ResponsePack(InvalidParams, "")
	}

	type AuxBlock struct {
		ChainId           int    `json:"chainid"`
		Height            uint64 `json:"height"`
		CoinBaseValue     int    `json:"coinbasevalue"`
		Bits              string `json:"bits"`
		Hash              string `json:"hash"`
		PreviousBlockHash string `json:"previousblockhash"`
	}

	Pow.PayToAddr = param["paytoaddress"].(string)

	preHash := ledger.DefaultLedger.Blockchain.CurrentBlockHash()
	preHashStr := BytesToHexString(preHash.ToArray())

	SendToAux := AuxBlock{
		ChainId:           auxpow.AuxPowChainID,
		Height:            NodeForServers.GetHeight(),
		CoinBaseValue:     1,                                          //transaction content
		Bits:              fmt.Sprintf("%x", msgBlock.Blockdata.Bits), //difficulty
		Hash:              curHashStr,
		PreviousBlockHash: preHashStr,
	}
	return ResponsePack(Success, &SendToAux)
}

func GetInfo(param map[string]interface{}) map[string]interface{} {
	RetVal := struct {
		Version        int    `json:"version"`
		Balance        int    `json:"balance"`
		Blocks         uint64 `json:"blocks"`
		Timeoffset     int    `json:"timeoffset"`
		Connections    uint   `json:"connections"`
		Testnet        bool   `json:"testnet"`
		Keypoololdest  int    `json:"keypoololdest"`
		Keypoolsize    int    `json:"keypoolsize"`
		Unlocked_until int    `json:"unlocked_until"`
		Paytxfee       int    `json:"paytxfee"`
		Relayfee       int    `json:"relayfee"`
		Errors         string `json:"errors"`
	}{
		Version:        config.Parameters.Version,
		Balance:        0,
		Blocks:         NodeForServers.GetHeight(),
		Timeoffset:     0,
		Connections:    NodeForServers.GetConnectionCnt(),
		Testnet:        config.Parameters.PowConfiguration.TestNet,
		Keypoololdest:  0,
		Keypoolsize:    0,
		Unlocked_until: 0,
		Paytxfee:       0,
		Relayfee:       0,
		Errors:         "Tobe written"}
	return ResponsePack(Success, &RetVal)
}

func AuxHelp(param map[string]interface{}) map[string]interface{} {

	//TODO  and description for this rpc-interface
	return ResponsePack(Success, "createauxblock==submitauxblock")
}

func ToggleMining(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "mining") {
		return ResponsePack(InvalidParams, "")
	}

	if param["mining"] == "start" {
		go Pow.Start()
	} else {
		go Pow.Halt()
	}

	return ResponsePack(Success, "")
}

func ManualMining(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "count") {
		return ResponsePack(InvalidParams, "")
	}

	count, err := strconv.ParseInt(param["count"].(string), 10, 64)
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}

	if count == 0 {
		return ResponsePack(InvalidParams, "")
	}

	ret := make([]string, count)

	blockHashes, err := Pow.ManualMining(uint32(count))
	if err != nil {
		return ResponsePack(Error, err)
	}

	for i, hash := range blockHashes {
		//ret[i] = hash.ToString()
		w := bytes.NewBuffer(nil)
		hash.Serialize(w)
		ret[i] = BytesToHexString(w.Bytes())
	}

	return ResponsePack(Success, ret)
}

// A JSON example for submitblock method as following:
//   {"jsonrpc": "2.0", "method": "submitblock", "params": ["raw block in hex"], "id": 0}
func SubmitBlock(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "block") {
		return ResponsePack(InvalidParams, "")
	}

	str := param["block"].(string)
	hex, _ := HexStringToBytes(str)
	var block ledger.Block
	if err := block.Deserialize(bytes.NewReader(hex)); err != nil {
		return ResponsePack(UnknownBlock, "")
	}
	if _, _, err := ledger.DefaultLedger.Blockchain.AddBlock(&block); err != nil {
		return ResponsePack(UnknownBlock, "")
	}
	if err := NodeForServers.Xmit(&block); err != nil {
		return ResponsePack(InternalError, "")
	}

	return ResponsePack(Success, "")
}

func GetConnectionCount(param map[string]interface{}) map[string]interface{} {
	return ResponsePack(Success, NodeForServers.GetConnectionCnt())
}

//Block
func GetCurrentHeight(param map[string]interface{}) map[string]interface{} {
	return ResponsePack(Success, ledger.DefaultLedger.Blockchain.BlockHeight)
}

func GetTransactionPool(param map[string]interface{}) map[string]interface{} {
	txs := []*Transactions{}
	txpool := NodeForServers.GetTxnPool(false)
	for _, t := range txpool {
		txs = append(txs, TransArrayByteToHexString(t))
	}
	return ResponsePack(Success, txs)
}

func GetBlockInfo(block *ledger.Block) BlockInfo {
	hash := block.Hash()
	auxInfo := &AuxInfo{
		Version:    block.Blockdata.AuxPow.GetBlockHeader().Version,
		PrevBlock:  BytesToHexString(new(Uint256).ToArrayReverse()),
		MerkleRoot: BytesToHexString(block.Blockdata.AuxPow.GetBlockHeader().MerkleRoot.ToArrayReverse()),
		Timestamp:  block.Blockdata.AuxPow.GetBlockHeader().Timestamp,
		Bits:       0,
		Nonce:      block.Blockdata.AuxPow.GetBlockHeader().Nonce,
	}
	blockHead := &BlockHead{
		Version:          block.Blockdata.Version,
		PrevBlockHash:    BytesToHexString(block.Blockdata.PrevBlockHash.ToArrayReverse()),
		TransactionsRoot: BytesToHexString(block.Blockdata.TransactionsRoot.ToArrayReverse()),
		Bits:             block.Blockdata.Bits,
		Timestamp:        block.Blockdata.Timestamp,
		Height:           block.Blockdata.Height,
		Nonce:            block.Blockdata.Nonce,
		AuxPow:           auxInfo,
		Difficulty:       ledger.CalcCurrentDifficulty(block.Blockdata.Bits),
		BlockSize:        block.GetSize(),

		Hash: BytesToHexString(hash.ToArrayReverse()),
	}

	trans := make([]*Transactions, len(block.Transactions))
	for i := 0; i < len(block.Transactions); i++ {
		trans[i] = TransArrayByteToHexString(block.Transactions[i])
		trans[i].Timestamp = block.Blockdata.Timestamp
		trans[i].Confirmations = ledger.DefaultLedger.Blockchain.GetBestHeight() - block.Blockdata.Height + 1
		w := bytes.NewBuffer(nil)
		block.Transactions[i].Serialize(w)
		trans[i].TxSize = uint32(len(w.Bytes()))

	}

	coinbasePd := block.Transactions[0].Payload.(*payload.CoinBase)
	b := BlockInfo{
		Hash:          BytesToHexString(hash.ToArrayReverse()),
		BlockData:     blockHead,
		Transactions:  trans,
		Confirmations: ledger.DefaultLedger.Blockchain.GetBestHeight() - block.Blockdata.Height + 1,
		MinerInfo:     string(coinbasePd.CoinbaseData),
	}
	return b
}

func getBlock(hash Uint256) (interface{}, ErrCode) {
	block, err := ledger.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		return "", UnknownBlock
	}
	if false {
		w := bytes.NewBuffer(nil)
		block.Serialize(w)
		return BytesToHexString(w.Bytes()), Success
	}
	return GetBlockInfo(block), Success
}

func GetBlockByHash(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "hash") {
		return ResponsePack(InvalidParams, "")
	}

	var hash Uint256
	hex, err := HexStringToBytesReverse(param["hash"].(string))
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}
	if err := hash.Deserialize(bytes.NewReader(hex)); err != nil {
		ResponsePack(InvalidTransaction, "")
	}

	result, error := getBlock(hash)

	return ResponsePack(error, result)
}

func SendRawTransaction(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "Data") {
		return ResponsePack(InvalidParams, "")
	}

	bys, err := HexStringToBytes(param["Data"].(string))
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}
	var txn tx.NodeTransaction
	if err := txn.Deserialize(bytes.NewReader(bys)); err != nil {
		return ResponsePack(InvalidTransaction, "")
	}
	var hash Uint256
	hash = txn.Hash()
	if errCode := VerifyAndSendTx(&txn); errCode != Success {
		return ResponsePack(errCode, "")
	}

	return ResponsePack(Success, BytesToHexString(hash.ToArrayReverse()))
}

func GetBlockHeight(param map[string]interface{}) map[string]interface{} {
	return ResponsePack(Success, ledger.DefaultLedger.Blockchain.BlockHeight)
}

func GetBlockHash(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "height") {
		return ResponsePack(InvalidParams, "")
	}

	height, err := strconv.ParseInt(param["height"].(string), 10, 64)
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}

	hash, err := ledger.DefaultLedger.Store.GetBlockHash(uint32(height))
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}
	return ResponsePack(Success, BytesToHexString(hash.ToArrayReverse()))
}

func GetBlockTransactions(block *ledger.Block) interface{} {
	trans := make([]string, len(block.Transactions))
	for i := 0; i < len(block.Transactions); i++ {
		h := block.Transactions[i].Hash()
		trans[i] = BytesToHexString(h.ToArrayReverse())
	}
	hash := block.Hash()
	type BlockTransactions struct {
		Hash         string
		Height       uint32
		Transactions []string
	}
	b := BlockTransactions{
		Hash:         BytesToHexString(hash.ToArrayReverse()),
		Height:       block.Blockdata.Height,
		Transactions: trans,
	}
	return b
}

func GetTransactionsByHeight(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "height") {
		return ResponsePack(InvalidParams, "")
	}

	height, err := strconv.ParseInt(param["height"].(string), 10, 64)
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}

	hash, err := ledger.DefaultLedger.Store.GetBlockHash(uint32(height))
	if err != nil {
		return ResponsePack(UnknownBlock, "")

	}
	block, err := ledger.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		return ResponsePack(UnknownBlock, "")
	}
	return ResponsePack(Success, GetBlockTransactions(block))
}

func GetBlockByHeight(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "height") {
		return ResponsePack(InvalidParams, "")
	}

	height, err := strconv.ParseInt(param["height"].(string), 10, 64)
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}

	hash, err := ledger.DefaultLedger.Store.GetBlockHash(uint32(height))
	if err != nil {
		return ResponsePack(UnknownBlock, "")
	}

	result, errCode := getBlock(hash)

	return ResponsePack(errCode, result)
}

func GetArbitratorGroupByHeight(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "height") {
		return ResponsePack(InvalidParams, "")
	}

	height, err := strconv.ParseInt(param["height"].(string), 10, 64)
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}

	hash, err := ledger.DefaultLedger.Store.GetBlockHash(uint32(height))
	if err != nil {
		return ResponsePack(UnknownBlock, "")
	}

	block, err := ledger.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		return ResponsePack(InternalError, "")
	}

	arbitratorsBytes, err := block.GetArbitrators()
	if err != nil {
		return ResponsePack(InternalError, "")
	}

	index, err := block.GetCurrentArbitratorIndex()
	if err != nil {
		return ResponsePack(InternalError, "")
	}

	var arbitrators []string
	for _, data := range arbitratorsBytes {
		arbitrators = append(arbitrators, BytesToHexString(data))
	}

	result := ArbitratorGroupInfo{
		OnDutyArbitratorIndex: index,
		Arbitrators:           arbitrators,
	}

	return ResponsePack(Success, result)
}

//Asset
func GetAssetByHash(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "hash") {
		return ResponsePack(InvalidParams, "")
	}
	hex, err := HexStringToBytesReverse(param["hash"].(string))
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}
	var hash Uint256
	err = hash.Deserialize(bytes.NewReader(hex))
	if err != nil {
		return ResponsePack(InvalidAsset, "")
	}
	asset, err := ledger.DefaultLedger.Store.GetAsset(hash)
	if err != nil {
		return ResponsePack(UnknownAsset, "")
	}
	if false {
		w := bytes.NewBuffer(nil)
		asset.Serialize(w)
		return ResponsePack(Success, BytesToHexString(w.Bytes()))
	}
	return ResponsePack(Success, asset)
}

func GetBalanceByAddr(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "addr") {
		return ResponsePack(InvalidParams, "")
	}

	programHash, err := Uint168FromAddress(param["addr"].(string))
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}
	unspends, err := ledger.DefaultLedger.Store.GetUnspentsFromProgramHash(*programHash)
	var balance Fixed64 = 0
	for _, u := range unspends {
		for _, v := range u {
			balance = balance + v.Value
		}
	}
	return ResponsePack(Success, balance.String())
}

func GetBalanceByAsset(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "addr", "assetid") {
		return ResponsePack(InvalidParams, "")
	}

	programHash, err := Uint168FromAddress(param["addr"].(string))
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}

	unspends, err := ledger.DefaultLedger.Store.GetUnspentsFromProgramHash(*programHash)
	var balance Fixed64 = 0
	for k, u := range unspends {
		assid := BytesToHexString(k.ToArrayReverse())
		for _, v := range u {
			if param["assetid"].(string) == assid {
				balance = balance + v.Value
			}
		}
	}
	return ResponsePack(Success, balance.String())
}

func GetUnspends(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "addr") {
		return ResponsePack(InvalidParams, "")
	}

	programHash, err := Uint168FromAddress(param["addr"].(string))
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}
	type UTXOUnspentInfo struct {
		Txid  string
		Index uint32
		Value string
	}
	type Result struct {
		AssetId   string
		AssetName string
		Utxo      []UTXOUnspentInfo
	}
	var results []Result
	unspends, err := ledger.DefaultLedger.Store.GetUnspentsFromProgramHash(*programHash)

	for k, u := range unspends {
		assetid := BytesToHexString(k.ToArrayReverse())
		asset, err := ledger.DefaultLedger.Store.GetAsset(k)
		if err != nil {
			return ResponsePack(InternalError, "")
		}
		var unspendsInfo []UTXOUnspentInfo
		for _, v := range u {
			unspendsInfo = append(unspendsInfo, UTXOUnspentInfo{BytesToHexString(v.Txid.ToArrayReverse()), v.Index, v.Value.String()})
		}
		results = append(results, Result{assetid, asset.Name, unspendsInfo})
	}
	return ResponsePack(Success, results)
}

func GetUnspendOutput(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "addr", "assetid") {
		return ResponsePack(InvalidParams, "")

	}

	programHash, err := Uint168FromAddress(param["addr"].(string))
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}
	bys, err := HexStringToBytesReverse(param["assetid"].(string))
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}

	var assetHash Uint256
	if err := assetHash.Deserialize(bytes.NewReader(bys)); err != nil {
		return ResponsePack(InvalidParams, "")
	}
	type UTXOUnspentInfo struct {
		Txid  string
		Index uint32
		Value string
	}
	infos, err := ledger.DefaultLedger.Store.GetUnspentFromProgramHash(*programHash, assetHash)
	if err != nil {
		return ResponsePack(InvalidParams, "")

	}
	var UTXOoutputs []UTXOUnspentInfo
	for _, v := range infos {
		UTXOoutputs = append(UTXOoutputs, UTXOUnspentInfo{Txid: BytesToHexString(v.Txid.ToArrayReverse()), Index: v.Index, Value: v.Value.String()})
	}
	return ResponsePack(Success, UTXOoutputs)
}

//NodeTransaction
func GetTransactionByHash(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "hash") {
		return ResponsePack(InvalidParams, "")
	}

	bys, err := HexStringToBytesReverse(param["hash"].(string))
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}

	var hash Uint256
	err = hash.Deserialize(bytes.NewReader(bys))
	if err != nil {
		return ResponsePack(InvalidTransaction, "")
	}
	txn, height, err := ledger.DefaultLedger.Store.GetTransaction(hash)
	if err != nil {
		return ResponsePack(UnknownTransaction, "")
	}
	if false {
		w := bytes.NewBuffer(nil)
		txn.Serialize(w)
		return ResponsePack(Success, BytesToHexString(w.Bytes()))
	}
	bHash, err := ledger.DefaultLedger.Store.GetBlockHash(height)
	if err != nil {
		return ResponsePack(UnknownBlock, "")
	}
	header, err := ledger.DefaultLedger.Store.GetHeader(bHash)
	if err != nil {
		return ResponsePack(UnknownBlock, "")
	}
	t := TransArrayByteToHexString(txn)
	t.Timestamp = header.Blockdata.Timestamp
	t.Confirmations = ledger.DefaultLedger.Blockchain.GetBestHeight() - height + 1
	w := bytes.NewBuffer(nil)
	txn.Serialize(w)
	t.TxSize = uint32(len(w.Bytes()))

	return ResponsePack(Success, t)
}
