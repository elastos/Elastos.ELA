package common

import (
	"bytes"
	"fmt"
	"strconv"

	. "Elastos.ELA/common"
	"Elastos.ELA/core/ledger"
	tx "Elastos.ELA/core/transaction"
	"Elastos.ELA/core/transaction/payload"
	. "Elastos.ELA/errors"
	. "Elastos.ELA/net/httpjsonrpc"
	. "Elastos.ELA/net/protocol"
)

var node Noder

const TlsPort int = 443

type ApiServer interface {
	Start() error
	Stop()
}

func SetNode(n Noder) {
	node = n
}

//Node
func GetConnectionCount(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)
	if node != nil {
		resp["Result"] = node.GetConnectionCnt()
	}

	return resp
}

//Block
func GetBlockHeight(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)
	resp["Result"] = ledger.DefaultLedger.Blockchain.BlockHeight
	return resp
}
func GetBlockHash(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)
	param := cmd["Height"].(string)
	if len(param) == 0 {
		resp["Error"] = InvalidParams
		return resp
	}
	height, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		resp["Error"] = InvalidParams
		return resp
	}
	hash, err := ledger.DefaultLedger.Store.GetBlockHash(uint32(height))
	if err != nil {
		resp["Error"] = InvalidParams
		return resp
	}
	resp["Result"] = BytesToHexString(hash.ToArrayReverse())
	return resp
}

func GetTransactionPool(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)

	txs := []*Transactions{}
	txpool := node.GetTxnPool(false)
	for _, t := range txpool {
		txs = append(txs, TransArryByteToHexString(t))
	}
	resp["Result"] = txs
	return resp
}

/*
func GetTotalIssued(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)
	assetid, ok := cmd["Assetid"].(string)
	if !ok {
		resp["Error"] = InvalidParams
		return resp
	}
	var assetHash Uint256

	bys, err := HexStringToBytesReverse(assetid)
	if err != nil {
		resp["Error"] = InvalidParams
		return resp
	}
	if err := assetHash.Deserialize(bytes.NewReader(bys)); err != nil {
		resp["Error"] = InvalidParams
		return resp
	}
	amount, err := ledger.DefaultLedger.Store.GetQuantityIssued(assetHash)
	if err != nil {
		resp["Error"] = InvalidParams
		return resp
	}
	resp["Result"] = amount.String()
	return resp
}
*/
func GetBlockInfo(block *ledger.Block) BlockInfo {
	hash := block.Hash()
	blockHead := &BlockHead{
		Version:          block.Blockdata.Version,
		PrevBlockHash:    BytesToHexString(block.Blockdata.PrevBlockHash.ToArrayReverse()),
		TransactionsRoot: BytesToHexString(block.Blockdata.TransactionsRoot.ToArrayReverse()),
		Bits:             block.Blockdata.Bits,
		Timestamp:        block.Blockdata.Timestamp,
		Height:           block.Blockdata.Height,
		Nonce:            block.Blockdata.Nonce,

		Hash: BytesToHexString(hash.ToArrayReverse()),
	}

	trans := make([]*Transactions, len(block.Transactions))
	for i := 0; i < len(block.Transactions); i++ {
		trans[i] = TransArryByteToHexString(block.Transactions[i])
		trans[i].Timestamp = block.Blockdata.Timestamp
		trans[i].Confirminations = ledger.DefaultLedger.Blockchain.GetBestHeight() - block.Blockdata.Height + 1
		w := bytes.NewBuffer(nil)
		block.Transactions[i].Serialize(w)
		trans[i].TxSize = uint32(len(w.Bytes()))

	}

	coinbasePd := block.Transactions[0].Payload.(*payload.CoinBase)
	b := BlockInfo{
		Hash:            BytesToHexString(hash.ToArrayReverse()),
		BlockData:       blockHead,
		Transactions:    trans,
		Confirminations: ledger.DefaultLedger.Blockchain.GetBestHeight() - block.Blockdata.Height + 1,
		MinerInfo:       string(coinbasePd.CoinbaseData),
	}
	return b
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
func getBlock(hash Uint256, getTxBytes bool) (interface{}, ErrCode) {
	block, err := ledger.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		return "", UnknownBlock
	}
	if getTxBytes {
		w := bytes.NewBuffer(nil)
		block.Serialize(w)
		return BytesToHexString(w.Bytes()), Success
	}
	return GetBlockInfo(block), Success
}
func GetBlockByHash(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)
	param := cmd["Hash"].(string)
	if len(param) == 0 {
		resp["Error"] = InvalidParams
		return resp
	}
	var getTxBytes bool = false
	if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
		getTxBytes = true
	}
	var hash Uint256
	hex, err := HexStringToBytesReverse(param)
	if err != nil {
		resp["Error"] = InvalidParams
		return resp
	}
	if err := hash.Deserialize(bytes.NewReader(hex)); err != nil {
		resp["Error"] = InvalidTransaction
		return resp
	}

	resp["Result"], resp["Error"] = getBlock(hash, getTxBytes)

	return resp
}
func GetBlockTxsByHeight(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)

	param := cmd["Height"].(string)
	if len(param) == 0 {
		resp["Error"] = InvalidParams
		return resp
	}
	height, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		resp["Error"] = InvalidParams
		return resp
	}
	index := uint32(height)
	hash, err := ledger.DefaultLedger.Store.GetBlockHash(index)
	if err != nil {
		resp["Error"] = UnknownBlock
		return resp
	}
	block, err := ledger.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		resp["Error"] = UnknownBlock
		return resp
	}
	resp["Result"] = GetBlockTransactions(block)
	return resp
}
func GetBlockByHeight(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)

	param := cmd["Height"].(string)
	if len(param) == 0 {
		resp["Error"] = InvalidParams
		return resp
	}
	var getTxBytes bool = false
	if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
		getTxBytes = true
	}
	height, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		resp["Error"] = InvalidParams
		return resp
	}
	index := uint32(height)
	hash, err := ledger.DefaultLedger.Store.GetBlockHash(index)
	if err != nil {
		resp["Error"] = UnknownBlock
		return resp
	}
	resp["Result"], resp["Error"] = getBlock(hash, getTxBytes)
	return resp
}

//Asset
func GetAssetByHash(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)

	str := cmd["Hash"].(string)
	hex, err := HexStringToBytesReverse(str)
	if err != nil {
		resp["Error"] = InvalidParams
		return resp
	}
	var hash Uint256
	err = hash.Deserialize(bytes.NewReader(hex))
	if err != nil {
		resp["Error"] = InvalidAsset
		return resp
	}
	asset, err := ledger.DefaultLedger.Store.GetAsset(hash)
	if err != nil {
		resp["Error"] = UnknownAsset
		return resp
	}
	if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
		w := bytes.NewBuffer(nil)
		asset.Serialize(w)
		resp["Result"] = BytesToHexString(w.Bytes())
		return resp
	}
	resp["Result"] = asset
	return resp
}
func GetBalanceByAddr(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)
	addr, ok := cmd["Addr"].(string)
	if !ok {
		resp["Error"] = InvalidParams
		return resp
	}
	var programHash Uint168
	programHash, err := ToScriptHash(addr)
	if err != nil {
		resp["Error"] = InvalidParams
		return resp
	}
	unspends, err := ledger.DefaultLedger.Store.GetUnspentsFromProgramHash(programHash)
	var balance Fixed64 = 0
	for _, u := range unspends {
		for _, v := range u {
			balance = balance + v.Value
		}
	}
	resp["Result"] = balance.String()
	return resp
}
func GetBalanceByAsset(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)
	addr, ok := cmd["Addr"].(string)
	assetid, k := cmd["Assetid"].(string)
	if !ok || !k {
		resp["Error"] = InvalidParams
		return resp
	}
	var programHash Uint168
	programHash, err := ToScriptHash(addr)
	if err != nil {
		resp["Error"] = InvalidParams
		return resp
	}
	unspends, err := ledger.DefaultLedger.Store.GetUnspentsFromProgramHash(programHash)
	var balance Fixed64 = 0
	for k, u := range unspends {
		assid := BytesToHexString(k.ToArrayReverse())
		for _, v := range u {
			if assetid == assid {
				balance = balance + v.Value
			}
		}
	}
	resp["Result"] = balance.String()
	return resp
}
func GetUnspends(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)
	addr, ok := cmd["Addr"].(string)
	if !ok {
		resp["Error"] = InvalidParams
		return resp
	}
	var programHash Uint168

	programHash, err := ToScriptHash(addr)
	if err != nil {
		resp["Error"] = InvalidParams
		return resp
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
	unspends, err := ledger.DefaultLedger.Store.GetUnspentsFromProgramHash(programHash)

	for k, u := range unspends {
		assetid := BytesToHexString(k.ToArrayReverse())
		asset, err := ledger.DefaultLedger.Store.GetAsset(k)
		if err != nil {
			resp["Error"] = InternalError
			return resp
		}
		var unspendsInfo []UTXOUnspentInfo
		for _, v := range u {
			unspendsInfo = append(unspendsInfo, UTXOUnspentInfo{BytesToHexString(v.Txid.ToArrayReverse()), v.Index, v.Value.String()})
		}
		results = append(results, Result{assetid, asset.Name, unspendsInfo})
	}
	resp["Result"] = results
	return resp
}
func GetUnspendOutput(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)
	addr, ok := cmd["Addr"].(string)
	assetid, k := cmd["Assetid"].(string)
	if !ok || !k {
		resp["Error"] = InvalidParams
		return resp
	}

	var programHash Uint168
	var assetHash Uint256
	programHash, err := ToScriptHash(addr)
	if err != nil {
		resp["Error"] = InvalidParams
		return resp
	}
	bys, err := HexStringToBytesReverse(assetid)
	if err != nil {
		resp["Error"] = InvalidParams
		return resp
	}
	if err := assetHash.Deserialize(bytes.NewReader(bys)); err != nil {
		resp["Error"] = InvalidParams
		return resp
	}
	type UTXOUnspentInfo struct {
		Txid  string
		Index uint32
		Value string
	}
	infos, err := ledger.DefaultLedger.Store.GetUnspentFromProgramHash(programHash, assetHash)
	if err != nil {
		resp["Error"] = InvalidParams
		resp["Result"] = err
		return resp
	}
	var UTXOoutputs []UTXOUnspentInfo
	for _, v := range infos {
		UTXOoutputs = append(UTXOoutputs, UTXOUnspentInfo{Txid: BytesToHexString(v.Txid.ToArrayReverse()), Index: v.Index, Value: v.Value.String()})
	}
	resp["Result"] = UTXOoutputs
	return resp
}

//Transaction
func GetTransactionByHash(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)

	str := cmd["Hash"].(string)
	bys, err := HexStringToBytesReverse(str)
	if err != nil {
		resp["Error"] = InvalidParams
		return resp
	}
	var hash Uint256
	err = hash.Deserialize(bytes.NewReader(bys))
	if err != nil {
		resp["Error"] = InvalidTransaction
		return resp
	}
	txn, height, err := ledger.DefaultLedger.Store.GetTransaction(hash)
	if err != nil {
		resp["Error"] = UnknownTransaction
		return resp
	}
	if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
		w := bytes.NewBuffer(nil)
		txn.Serialize(w)
		resp["Result"] = BytesToHexString(w.Bytes())
		return resp
	}
	bHash, err := ledger.DefaultLedger.Store.GetBlockHash(height)
	if err != nil {
		resp["Error"] = UnknownBlock
		return resp
	}
	header, err := ledger.DefaultLedger.Store.GetHeader(bHash)
	if err != nil {
		resp["Error"] = UnknownBlock
		return resp
	}
	t := TransArryByteToHexString(txn)
	t.Timestamp = header.Blockdata.Timestamp
	t.Confirminations = ledger.DefaultLedger.Blockchain.GetBestHeight() - height + 1
	w := bytes.NewBuffer(nil)
	txn.Serialize(w)
	t.TxSize = uint32(len(w.Bytes()))

	resp["Result"] = t
	return resp
}
func SendRawTransaction(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)

	str, ok := cmd["Data"].(string)
	if !ok {
		resp["Error"] = InvalidParams
		return resp
	}
	bys, err := HexStringToBytes(str)
	if err != nil {
		resp["Error"] = InvalidParams
		return resp
	}
	var txn tx.Transaction
	if err := txn.Deserialize(bytes.NewReader(bys)); err != nil {
		resp["Error"] = InvalidTransaction
		return resp
	}
	var hash Uint256
	hash = txn.Hash()
	if errCode := VerifyAndSendTx(&txn); errCode != Success {
		resp["Error"] = int64(errCode)
		return resp
	}
	resp["Result"] = BytesToHexString(hash.ToArrayReverse())
	//TODO 0xd1 -> tx.InvokeCode
	if txn.TxType == 0xd1 {
		if userid, ok := cmd["Userid"].(string); ok && len(userid) > 0 {
			resp["Userid"] = userid
		}
	}
	return resp
}

//stateupdate
func GetStateUpdate(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)
	namespace, ok := cmd["Namespace"].(string)
	if !ok {
		resp["Error"] = InvalidParams
		return resp
	}
	key, ok := cmd["Key"].(string)
	if !ok {
		resp["Error"] = InvalidParams
		return resp
	}
	fmt.Println(cmd, namespace, key)
	//TODO get state from store
	return resp
}

func ResponsePack(errCode ErrCode) map[string]interface{} {
	resp := map[string]interface{}{
		"Action":  "",
		"Result":  "",
		"Error":   errCode,
		"Desc":    "",
		"Version": "1.0.0",
	}
	return resp
}
func GetContract(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)
	str := cmd["Hash"].(string)
	bys, err := HexStringToBytesReverse(str)
	if err != nil {
		resp["Error"] = InvalidParams
		return resp
	}
	var hash Uint168
	err = hash.Deserialize(bytes.NewReader(bys))
	if err != nil {
		resp["Error"] = InvalidParams
		return resp
	}
	//TODO GetContract from store
	//contract, err := ledger.DefaultLedger.Store.GetContract(hash)
	//if err != nil {
	//	resp["Error"] = InvalidParams
	//	return resp
	//}
	//resp["Result"] = string(contract)
	return resp
}
