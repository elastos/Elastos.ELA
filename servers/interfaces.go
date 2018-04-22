package servers

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	chain "github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/config"
	. "github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/log"

	. "github.com/elastos/Elastos.ELA.Utility/common"
	. "github.com/elastos/Elastos.ELA.Utility/core"
)

const (
	AUXBLOCK_GENERATED_INTERVAL_SECONDS = 60
)

var PreChainHeight uint64
var PreTime int64

func TransArrayByteToHexString(ptx *Transaction) *Transactions {
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
	reference, _ := chain.DefaultLedger.Store.GetTxReference(ptx)
	trans.UTXOInputs = make([]UTXOTxInputInfo, len(ptx.Inputs))
	for _, v := range ptx.Inputs {
		trans.UTXOInputs[n].ReferTxID = BytesToHexString(BytesReverse(v.Previous.TxID.Bytes()))
		trans.UTXOInputs[n].ReferTxOutputIndex = v.Previous.Index
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
	trans.Outputs = make([]TxoutputInfo, len(ptx.Outputs))
	for _, v := range ptx.Outputs {
		trans.Outputs[n].AssetID = BytesToHexString(BytesReverse(v.AssetID.Bytes()))
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

	mHash := ptx.Hash()
	trans.Hash = BytesToHexString(BytesReverse(mHash.Bytes()))

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
	tx, height, err := chain.DefaultLedger.Store.GetTransaction(hash)
	if err != nil {
		return ResponsePack(UnknownTransaction, "")
	}
	bHash, err := chain.DefaultLedger.Store.GetBlockHash(height)
	if err != nil {
		return ResponsePack(UnknownTransaction, "")
	}
	header, err := chain.DefaultLedger.Store.GetHeader(bHash)
	if err != nil {
		return ResponsePack(UnknownTransaction, "")
	}
	tran := TransArrayByteToHexString(tx)
	tran.Timestamp = header.Timestamp
	tran.Confirmations = chain.DefaultLedger.Blockchain.GetBestHeight() - height + 1
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
		State:    uint(NodeForServers.State()),
		Time:     NodeForServers.GetTime(),
		Port:     NodeForServers.Port(),
		ID:       NodeForServers.ID(),
		Version:  NodeForServers.Version(),
		Services: NodeForServers.Services(),
		Relay:    NodeForServers.IsRelay(),
		Height:   NodeForServers.Height(),
		TxnCnt:   NodeForServers.GetTxnCnt(),
		RxTxnCnt: NodeForServers.GetRxTxnCnt(),
	}
	return ResponsePack(Success, n)
}

func SetLogLevel(param map[string]interface{}) map[string]interface{} {
	if !checkParam(param, "level") {
		return ResponsePack(InvalidParams, "")
	}

	level, ok := param["level"].(float64)
	if !ok {
		return ResponsePack(InvalidParams, "need a level, with positive integer in 0-6")
	}
	levelInt := int(level)
	if !ok {
		return ResponsePack(InvalidParams, "level should be an integer")
	}

	if err := log.Log.SetDebugLevel(levelInt); err != nil {
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
	Pow.MsgBlock.BlockData[blockHash].Header.AuxPow.Deserialize(r)
	_, _, err := chain.DefaultLedger.Blockchain.AddBlock(Pow.MsgBlock.BlockData[blockHash])
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

func GenerateAuxBlock(addr string) (*Block, string, bool) {
	msgBlock := &Block{}
	if NodeForServers.Height() == 0 || PreChainHeight != NodeForServers.Height() ||
		time.Now().Unix()-PreTime > AUXBLOCK_GENERATED_INTERVAL_SECONDS {

		if PreChainHeight != NodeForServers.Height() {
			PreChainHeight = NodeForServers.Height()
			PreTime = time.Now().Unix()
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
		curHashStr := BytesToHexString(curHash.Bytes())

		Pow.MsgBlock.Mutex.Lock()
		Pow.MsgBlock.BlockData[curHashStr] = msgBlock
		Pow.MsgBlock.Mutex.Unlock()

		PreChainHeight = NodeForServers.Height()
		PreTime = time.Now().Unix()

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

	preHash := chain.DefaultLedger.Blockchain.CurrentBlockHash()
	preHashStr := BytesToHexString(preHash.Bytes())

	SendToAux := AuxBlock{
		ChainId:           AuxPowChainID,
		Height:            NodeForServers.Height(),
		CoinBaseValue:     1,                                       //transaction content
		Bits:              fmt.Sprintf("%x", msgBlock.Header.Bits), //difficulty
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
		Blocks:         NodeForServers.Height(),
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
	if !checkParam(param, "mine") {
		return ResponsePack(InvalidParams, "")
	}

	if param["mining"] == true {
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

	count, _ := strconv.ParseInt(param["count"].(string), 10, 32)
	if count <= 0 {
		return ResponsePack(InvalidParams, "need an positive integer")
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
	var block Block
	if err := block.Deserialize(bytes.NewReader(hex)); err != nil {
		return ResponsePack(UnknownBlock, "")
	}
	if _, _, err := chain.DefaultLedger.Blockchain.AddBlock(&block); err != nil {
		return ResponsePack(UnknownBlock, "")
	}
	if err := NodeForServers.Relay(nil, &block); err != nil {
		return ResponsePack(InternalError, "")
	}

	return ResponsePack(Success, "")
}

func GetConnectionCount(param map[string]interface{}) map[string]interface{} {
	return ResponsePack(Success, NodeForServers.GetConnectionCnt())
}

//Block
func GetCurrentHeight(param map[string]interface{}) map[string]interface{} {
	return ResponsePack(Success, chain.DefaultLedger.Blockchain.BlockHeight)
}

func GetTransactionPool(param map[string]interface{}) map[string]interface{} {
	txs := []*Transactions{}
	txpool := NodeForServers.GetTxnPool(false)
	for _, t := range txpool {
		txs = append(txs, TransArrayByteToHexString(t))
	}
	return ResponsePack(Success, txs)
}

func GetBlockInfo(block *Block) map[string]interface{} {
	hash := block.Hash()

	var txHashes []string
	for i := 0; i < len(block.Transactions); i++ {
		hash := block.Transactions[i].Hash()
		txHashes = append(txHashes, BytesToHexString(BytesReverse(hash.Bytes())))
	}

	return map[string]interface{}{
		"hash":              BytesToHexString(BytesReverse(hash.Bytes())),
		"confirmations":     chain.DefaultLedger.Blockchain.GetBestHeight() - block.Header.Height + 1,
		"size":              block.GetSize(),
		"height":            block.Header.Height,
		"version":           block.Header.Version,
		"merkleroot":        BytesToHexString(BytesReverse(block.Header.AuxPow.ParBlockHeader.MerkleRoot.Bytes())),
		"time":              block.Header.Timestamp,
		"nonce":             block.Header.Nonce,
		"difficulty":        chain.CalcCurrentDifficulty(block.Header.Bits),
		"bits":              block.Header.Bits,
		"previousblockhash": BytesToHexString(BytesReverse(block.Header.Previous.Bytes())),
		"tx":                txHashes,
	}
}

func getBlock(hash Uint256) (interface{}, ErrCode) {
	block, err := chain.DefaultLedger.Store.GetBlock(hash)
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
	var txn Transaction
	if err := txn.Deserialize(bytes.NewReader(bys)); err != nil {
		return ResponsePack(InvalidTransaction, "")
	}
	var hash Uint256
	if errCode := VerifyAndSendTx(&txn); errCode != Success {
		return ResponsePack(errCode, "")
	}

	return ResponsePack(Success, BytesToHexString(BytesReverse(hash.Bytes())))
}

func GetBlockHeight(param map[string]interface{}) map[string]interface{} {
	return ResponsePack(Success, chain.DefaultLedger.Blockchain.BlockHeight)
}

func GetBestBlockHash(param map[string]interface{}) map[string]interface{} {
	bestHeight := chain.DefaultLedger.Blockchain.BlockHeight
	return GetBlockHash(map[string]interface{}{"index": float64(bestHeight)})
}

func GetBlockCount(param map[string]interface{}) map[string]interface{} {
	return ResponsePack(Success, chain.DefaultLedger.Blockchain.BlockHeight+1)
}

func GetBlockHash(param map[string]interface{}) map[string]interface{} {

	height := uint32(param["index"].(float64))
	log.Info("my height:", height)

	hash, err := chain.DefaultLedger.Store.GetBlockHash(uint32(height))
	if err != nil {
		return ResponsePack(InvalidParams, "")
	}
	return ResponsePack(Success, BytesToHexString(BytesReverse(hash.Bytes())))
}

func GetBlockTransactions(block *Block) interface{} {
	trans := make([]string, len(block.Transactions))
	for i := 0; i < len(block.Transactions); i++ {
		h := block.Transactions[i].Hash()
		trans[i] = BytesToHexString(BytesReverse(h.Bytes()))
	}
	hash := block.Hash()
	type BlockTransactions struct {
		Hash         string
		Height       uint32
		Transactions []string
	}
	b := BlockTransactions{
		Hash:         BytesToHexString(BytesReverse(hash.Bytes())),
		Height:       block.Header.Height,
		Transactions: trans,
	}
	return b
}

func GetTransactionsByHeight(param map[string]interface{}) map[string]interface{} {

	height := uint32(param["height"].(float64))
	if height <= 0 {
		return ResponsePack(InvalidParams, "")
	}

	hash, err := chain.DefaultLedger.Store.GetBlockHash(uint32(height))
	if err != nil {
		return ResponsePack(UnknownBlock, "")

	}
	block, err := chain.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		return ResponsePack(UnknownBlock, "")
	}
	return ResponsePack(Success, GetBlockTransactions(block))
}

func GetBlockByHeight(param map[string]interface{}) map[string]interface{} {

	height := uint32(param["height"].(float64))
	if height <= 0 {
		return ResponsePack(InvalidParams, "")
	}

	hash, err := chain.DefaultLedger.Store.GetBlockHash(uint32(height))
	if err != nil {
		return ResponsePack(UnknownBlock, "")
	}

	result, errCode := getBlock(hash)

	return ResponsePack(errCode, result)
}

func GetArbitratorGroupByHeight(param map[string]interface{}) map[string]interface{} {

	height := uint32(param["height"].(float64))

	if height <= 0 {
		return ResponsePack(InvalidParams, "")
	}

	hash, err := chain.DefaultLedger.Store.GetBlockHash(uint32(height))
	if err != nil {
		return ResponsePack(UnknownBlock, "")
	}

	block, err := chain.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		return ResponsePack(InternalError, "")
	}

	arbitratorsBytes, err := block.GetArbitrators(config.Parameters.Arbiters)
	if err != nil {
		return ResponsePack(InternalError, "")
	}

	index := int(block.Header.Height) % len(arbitratorsBytes)

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
	asset, err := chain.DefaultLedger.Store.GetAsset(hash)
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
	unspends, err := chain.DefaultLedger.Store.GetUnspentsFromProgramHash(*programHash)
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

	unspends, err := chain.DefaultLedger.Store.GetUnspentsFromProgramHash(*programHash)
	var balance Fixed64 = 0
	for k, u := range unspends {
		assid := BytesToHexString(BytesReverse(k.Bytes()))
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
	unspends, err := chain.DefaultLedger.Store.GetUnspentsFromProgramHash(*programHash)

	for k, u := range unspends {
		assetid := BytesToHexString(BytesReverse(k.Bytes()))
		asset, err := chain.DefaultLedger.Store.GetAsset(k)
		if err != nil {
			return ResponsePack(InternalError, "")
		}
		var unspendsInfo []UTXOUnspentInfo
		for _, v := range u {
			unspendsInfo = append(unspendsInfo, UTXOUnspentInfo{BytesToHexString(BytesReverse(v.TxId.Bytes())), v.Index, v.Value.String()})
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
	infos, err := chain.DefaultLedger.Store.GetUnspentFromProgramHash(*programHash, assetHash)
	if err != nil {
		return ResponsePack(InvalidParams, "")

	}
	var UTXOoutputs []UTXOUnspentInfo
	for _, v := range infos {
		UTXOoutputs = append(UTXOoutputs, UTXOUnspentInfo{Txid: BytesToHexString(BytesReverse(v.TxId.Bytes())), Index: v.Index, Value: v.Value.String()})
	}
	return ResponsePack(Success, UTXOoutputs)
}

//Transaction
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
	txn, height, err := chain.DefaultLedger.Store.GetTransaction(hash)
	if err != nil {
		return ResponsePack(UnknownTransaction, "")
	}
	if false {
		w := bytes.NewBuffer(nil)
		txn.Serialize(w)
		return ResponsePack(Success, BytesToHexString(w.Bytes()))
	}
	bHash, err := chain.DefaultLedger.Store.GetBlockHash(height)
	if err != nil {
		return ResponsePack(UnknownBlock, "")
	}
	header, err := chain.DefaultLedger.Store.GetHeader(bHash)
	if err != nil {
		return ResponsePack(UnknownBlock, "")
	}
	t := TransArrayByteToHexString(txn)
	t.Timestamp = header.Timestamp
	t.Confirmations = chain.DefaultLedger.Blockchain.GetBestHeight() - height + 1
	w := bytes.NewBuffer(nil)
	txn.Serialize(w)
	t.TxSize = uint32(len(w.Bytes()))

	return ResponsePack(Success, t)
}
