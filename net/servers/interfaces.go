package servers

import (
	"bytes"
	"fmt"
	"time"

	. "Elastos.ELA/common"
	"Elastos.ELA/common/config"
	"Elastos.ELA/common/log"
	"Elastos.ELA/core/ledger"
	tx "Elastos.ELA/core/transaction"
	"Elastos.ELA/core/transaction/payload"
	. "Elastos.ELA/errors"
	"strconv"
)

const (
	AUXBLOCK_GENERATED_INTERVAL_SECONDS = 60
)

var PreChainHeight uint64
var PreTime int64
var PreTransactionCount int

func TransArrayByteToHexString(ptx *tx.Transaction) *Transactions {

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

// Input JSON string examples for getblock method as following:
//   {"jsonrpc": "2.0", "method": "getblock", "params": [1], "id": 0}
//   {"jsonrpc": "2.0", "method": "getblock", "params": ["aabbcc.."], "id": 0}

func GetRawTransaction(cmd map[string]interface{}) map[string]interface{} {
	if len(cmd) < 1 {
		return ElaRpcNil
	}
	switch cmd["txid"].(type) {
	case string:
		str := cmd["txid"].(string)
		hex, err := HexStringToBytesReverse(str)
		if err != nil {
			return ElaRpcInvalidParameter
		}
		var hash Uint256
		err = hash.Deserialize(bytes.NewReader(hex))
		if err != nil {
			return ElaRpcInvalidTransaction
		}
		tx, height, err := ledger.DefaultLedger.Store.GetTransaction(hash)
		if err != nil {
			return ElaRpcUnknownTransaction
		}
		bHash, err := ledger.DefaultLedger.Store.GetBlockHash(height)
		if err != nil {
			return ElaRpcUnknownTransaction
		}
		header, err := ledger.DefaultLedger.Store.GetHeader(bHash)
		if err != nil {
			return ElaRpcUnknownTransaction
		}
		tran := TransArrayByteToHexString(tx)
		tran.Timestamp = header.Blockdata.Timestamp
		tran.Confirminations = ledger.DefaultLedger.Blockchain.GetBestHeight() - height + 1
		w := bytes.NewBuffer(nil)
		tx.Serialize(w)
		tran.TxSize = uint32(len(w.Bytes()))

		return ElaRpc(tran)
	default:
		return ElaRpcInvalidParameter
	}
}

func GetNeighbor(cmd map[string]interface{}) map[string]interface{} {
	addr, _ := NodeForServers.GetNeighborAddrs()
	return ElaRpc(addr)
}

func GetNodeState(cmd map[string]interface{}) map[string]interface{} {
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
	return ElaRpc(n)
}

func SetLogLevel(cmd map[string]interface{}) map[string]interface{} {
	if len(cmd) < 1 {
		return ElaRpcInvalidParameter
	}
	switch cmd["level"].(type) {
	case float64:
		level := cmd["level"].(float64)
		if err := log.Log.SetDebugLevel(int(level)); err != nil {
			return ElaRpcInvalidParameter
		}
	default:
		return ElaRpcInvalidParameter
	}
	return ElaRpcSuccess
}

func SubmitAuxBlock(cmd map[string]interface{}) map[string]interface{} {
	auxPow, blockHash := "", ""
	switch cmd["blockhash"].(type) {
	case string:
		blockHash = cmd["blockhash"].(string)
		if _, ok := Pow.MsgBlock.BlockData[blockHash]; !ok {
			log.Trace("[json-rpc:SubmitAuxBlock] receive invalid block hash value:", blockHash)
			return ElaRpcInvalidHash
		}

	default:
		return ElaRpcInvalidParameter
	}

	switch cmd["auxpow"].(type) {
	case string:
		auxPow = cmd["auxpow"].(string)
		temp, _ := HexStringToBytes(auxPow)
		r := bytes.NewBuffer(temp)
		Pow.MsgBlock.BlockData[blockHash].Blockdata.AuxPow.Deserialize(r)
		_, _, err := ledger.DefaultLedger.Blockchain.AddBlock(Pow.MsgBlock.BlockData[blockHash])
		if err != nil {
			log.Trace(err)
			return ElaRpcInternalError
		}

		Pow.MsgBlock.Mutex.Lock()
		for key := range Pow.MsgBlock.BlockData {
			delete(Pow.MsgBlock.BlockData, key)
		}
		Pow.MsgBlock.Mutex.Unlock()
		log.Trace("AddBlock called finished and Pow.MsgBlock.BlockData has been deleted completely")

	default:
		return ElaRpcInvalidParameter
	}
	log.Info(auxPow, blockHash)
	return ElaRpcSuccess
}

func GenerateAuxBlock(addr string) (*ledger.Block, string, bool) {
	msgBlock := &ledger.Block{}

	if NodeForServers.GetHeight() == 0 || PreChainHeight != NodeForServers.GetHeight() || (time.Now().Unix()-PreTime > AUXBLOCK_GENERATED_INTERVAL_SECONDS && Pow.GetTransactionCount() != PreTransactionCount) {
		if PreChainHeight != NodeForServers.GetHeight() {
			PreChainHeight = NodeForServers.GetHeight()
			PreTime = time.Now().Unix()
			PreTransactionCount = Pow.GetTransactionCount()
		}

		currentTxsCount := Pow.CollectTransactions(msgBlock)
		if 0 == currentTxsCount {
			return nil, "currentTxs is nil", false
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

func CreateAuxBlock(cmd map[string]interface{}) map[string]interface{} {
	msgBlock, curHashStr, _ := GenerateAuxBlock(config.Parameters.PowConfiguration.PayToAddr)
	if nil == msgBlock {
		return ElaRpcNil
	}

	type AuxBlock struct {
		ChainId           int    `json:"chainid"`
		Height            uint64 `json:"height"`
		CoinBaseValue     int    `json:"coinbasevalue"`
		Bits              string `json:"bits"`
		Hash              string `json:"hash"`
		PreviousBlockHash string `json:"previousblockhash"`
	}

	switch cmd["paytoaddress"].(type) {
	case string:
		Pow.PayToAddr = cmd["paytoaddress"].(string)

		preHash := ledger.DefaultLedger.Blockchain.CurrentBlockHash()
		preHashStr := BytesToHexString(preHash.ToArray())

		SendToAux := AuxBlock{
			ChainId:           1,
			Height:            NodeForServers.GetHeight(),
			CoinBaseValue:     1,                                          //transaction content
			Bits:              fmt.Sprintf("%x", msgBlock.Blockdata.Bits), //difficulty
			Hash:              curHashStr,
			PreviousBlockHash: preHashStr}
		return ElaRpc(&SendToAux)

	default:
		return ElaRpcInvalidParameter

	}
}

func GetInfo(cmd map[string]interface{}) map[string]interface{} {
	RetVal := struct {
		Version     int    `json:"version"`
		Balance     int    `json:"balance"`
		Blocks      uint64 `json:"blocks"`
		Timeoffset  int    `json:"timeoffset"`
		Connections uint   `json:"connections"`
		//Difficulty      int    `json:"difficulty"`
		Testnet        bool   `json:"testnet"`
		Keypoololdest  int    `json:"keypoololdest"`
		Keypoolsize    int    `json:"keypoolsize"`
		Unlocked_until int    `json:"unlocked_until"`
		Paytxfee       int    `json:"paytxfee"`
		Relayfee       int    `json:"relayfee"`
		Errors         string `json:"errors"`
	}{
		Version:     config.Parameters.Version,
		Balance:     0,
		Blocks:      NodeForServers.GetHeight(),
		Timeoffset:  0,
		Connections: NodeForServers.GetConnectionCnt(),
		//Difficulty:      ledger.PowLimitBits,
		Testnet:        config.Parameters.PowConfiguration.TestNet,
		Keypoololdest:  0,
		Keypoolsize:    0,
		Unlocked_until: 0,
		Paytxfee:       0,
		Relayfee:       0,
		Errors:         "Tobe written"}
	return ElaRpc(&RetVal)
}

func AuxHelp(cmd map[string]interface{}) map[string]interface{} {

	//TODO  and description for this rpc-interface
	return ElaRpc("createauxblock==submitauxblock")
}

func ToggleCpuMining(cmd map[string]interface{}) map[string]interface{} {
	var isMining bool
	switch cmd["mining"].(type) {
	case bool:
		isMining = cmd["mining"].(bool)

	default:
		return ElaRpcInvalidParameter
	}

	if isMining {
		go Pow.Start()
	} else {
		go Pow.Halt()
	}

	return ElaRpcSuccess
}

func ManualCpuMining(cmd map[string]interface{}) map[string]interface{} {
	var blockcount uint32
	switch cmd["count"].(type) {
	case float64:
		blockcount = uint32(cmd["count"].(float64))
	default:
		return ElaRpcInvalidParameter
	}

	if blockcount == 0 {
		return ElaRpcInvalidParameter
	}

	ret := make([]string, blockcount)

	blockHashes, err := Pow.ManualMining(blockcount)
	if err != nil {
		return ElaRpcFailed
	}

	for i, hash := range blockHashes {
		//ret[i] = hash.ToString()
		w := bytes.NewBuffer(nil)
		hash.Serialize(w)
		ret[i] = BytesToHexString(w.Bytes())
	}

	return ElaRpc(ret)
}

// A JSON example for submitblock method as following:
//   {"jsonrpc": "2.0", "method": "submitblock", "params": ["raw block in hex"], "id": 0}
func SubmitBlock(cmd map[string]interface{}) map[string]interface{} {
	if len(cmd) < 1 {
		return ElaRpcNil
	}
	switch cmd["block"].(type) {
	case string:
		str := cmd["block"].(string)
		hex, _ := HexStringToBytes(str)
		var block ledger.Block
		if err := block.Deserialize(bytes.NewReader(hex)); err != nil {
			return ElaRpcInvalidBlock
		}
		if _, _, err := ledger.DefaultLedger.Blockchain.AddBlock(&block); err != nil {
			return ElaRpcInvalidBlock
		}
		if err := NodeForServers.Xmit(&block); err != nil {
			return ElaRpcInternalError
		}
	default:
		return ElaRpcInvalidParameter
	}
	return ElaRpcSuccess
}

func GetConnectionCount(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)
	resp["Result"] = NodeForServers.GetConnectionCnt()

	return resp
}

//Block
func GetCurrentHeight(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)
	resp["Result"] = ledger.DefaultLedger.Blockchain.BlockHeight
	return resp
}

func GetBlockHashByHeight(cmd map[string]interface{}) map[string]interface{} {
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
	txpool := NodeForServers.GetTxnPool(false)
	for _, t := range txpool {
		txs = append(txs, TransArrayByteToHexString(t))
	}
	resp["Result"] = txs
	return resp
}

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
		trans[i] = TransArrayByteToHexString(block.Transactions[i])
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
	t := TransArrayByteToHexString(txn)
	t.Timestamp = header.Blockdata.Timestamp
	t.Confirminations = ledger.DefaultLedger.Blockchain.GetBestHeight() - height + 1
	w := bytes.NewBuffer(nil)
	txn.Serialize(w)
	t.TxSize = uint32(len(w.Bytes()))

	resp["Result"] = t
	return resp
}
