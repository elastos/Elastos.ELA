package httpjsonrpc

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
)

const (
	AUXBLOCK_GENERATED_INTERVAL_SECONDS = 60
)

var PreChainHeight uint64
var PreTime int64
var PreTransactionCount int

func TransArryByteToHexString(ptx *tx.Transaction) *Transactions {

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

func getBestBlockHash(params []interface{}) map[string]interface{} {
	hash := ledger.DefaultLedger.Blockchain.CurrentBlockHash()
	return ElaRpc(BytesToHexString(hash.ToArrayReverse()))
}

// Input JSON string examples for getblock method as following:
//   {"jsonrpc": "2.0", "method": "getblock", "params": [1], "id": 0}
//   {"jsonrpc": "2.0", "method": "getblock", "params": ["aabbcc.."], "id": 0}
func getBlock(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return ElaRpcNil
	}
	var err error
	var hash Uint256
	switch (params[0]).(type) {
	// block height
	case float64:
		index := uint32(params[0].(float64))
		hash, err = ledger.DefaultLedger.Store.GetBlockHash(index)
		if err != nil {
			return ElaRpcUnknownBlock
		}
		// block hash
	case string:
		str := params[0].(string)
		hex, err := HexStringToBytesReverse(str)
		if err != nil {
			return ElaRpcInvalidParameter
		}
		if err := hash.Deserialize(bytes.NewReader(hex)); err != nil {
			return ElaRpcInvalidTransaction
		}
	default:
		return ElaRpcInvalidParameter
	}

	block, err := ledger.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		return ElaRpcUnknownBlock
	}

	blockHead := &BlockHead{
		Version:          block.Blockdata.Version,
		PrevBlockHash:    BytesToHexString(block.Blockdata.PrevBlockHash.ToArrayReverse()),
		TransactionsRoot: BytesToHexString(block.Blockdata.TransactionsRoot.ToArrayReverse()),
		Timestamp:        block.Blockdata.Timestamp,
		Bits:             block.Blockdata.Bits,
		Height:           block.Blockdata.Height,
		Nonce:            block.Blockdata.Nonce,
		Hash:             BytesToHexString(hash.ToArrayReverse()),
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
	return ElaRpc(b)
}

func getBlockCount(params []interface{}) map[string]interface{} {
	return ElaRpc(ledger.DefaultLedger.Blockchain.BlockHeight + 1)
}

// A JSON example for getblockhash method as following:
//   {"jsonrpc": "2.0", "method": "getblockhash", "params": [1], "id": 0}
func getBlockHash(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return ElaRpcNil
	}
	switch params[0].(type) {
	case float64:
		height := uint32(params[0].(float64))
		hash, err := ledger.DefaultLedger.Store.GetBlockHash(height)
		if err != nil {
			return ElaRpcUnknownBlock
		}
		return ElaRpc(BytesToHexString(hash.ToArrayReverse()))
	default:
		return ElaRpcInvalidParameter
	}
}

func getConnectionCount(params []interface{}) map[string]interface{} {
	return ElaRpc(node.GetConnectionCnt())
}

func getRawMemPool(params []interface{}) map[string]interface{} {
	txs := []*Transactions{}
	txpool := node.GetTxnPool(false)
	for _, t := range txpool {
		txs = append(txs, TransArryByteToHexString(t))
	}
	if len(txs) == 0 {
		return ElaRpcNil
	}
	return ElaRpc(txs)
}

// A JSON example for getrawtransaction method as following:
//   {"jsonrpc": "2.0", "method": "getrawtransaction", "params": ["transactioin hash in hex"], "id": 0}
func getRawTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return ElaRpcNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
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
		tran := TransArryByteToHexString(tx)
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

func getNeighbor(params []interface{}) map[string]interface{} {
	addr, _ := node.GetNeighborAddrs()
	return ElaRpc(addr)
}

func getNodeState(params []interface{}) map[string]interface{} {
	n := NodeInfo{
		State:    uint(node.GetState()),
		Time:     node.GetTime(),
		Port:     node.GetPort(),
		ID:       node.GetID(),
		Version:  node.Version(),
		Services: node.Services(),
		Relay:    node.GetRelay(),
		Height:   node.GetHeight(),
		TxnCnt:   node.GetTxnCnt(),
		RxTxnCnt: node.GetRxTxnCnt(),
	}
	return ElaRpc(n)
}

func setDebugInfo(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return ElaRpcInvalidParameter
	}
	switch params[0].(type) {
	case float64:
		level := params[0].(float64)
		if err := log.Log.SetDebugLevel(int(level)); err != nil {
			return ElaRpcInvalidParameter
		}
	default:
		return ElaRpcInvalidParameter
	}
	return ElaRpcSuccess
}

func submitAuxBlock(params []interface{}) map[string]interface{} {
	auxPow, blockHash := "", ""
	switch params[0].(type) {
	case string:
		blockHash = params[0].(string)
		if _, ok := Pow.MsgBlock.BlockData[blockHash]; !ok {
			log.Trace("[json-rpc:submitAuxBlock] receive invalid block hash value:", blockHash)
			return ElaRpcInvalidHash
		}

	default:
		return ElaRpcInvalidParameter
	}

	switch params[1].(type) {
	case string:
		auxPow = params[1].(string)
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

func generateAuxBlock(addr string) (*ledger.Block, string, bool) {
	msgBlock := &ledger.Block{}

	if node.GetHeight() == 0 || PreChainHeight != node.GetHeight() || (time.Now().Unix()-PreTime > AUXBLOCK_GENERATED_INTERVAL_SECONDS && Pow.GetTransactionCount() != PreTransactionCount) {
		if PreChainHeight != node.GetHeight() {
			PreChainHeight = node.GetHeight()
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

		PreChainHeight = node.GetHeight()
		PreTime = time.Now().Unix()
		PreTransactionCount = currentTxsCount // Don't Call GetTransactionCount()

		return msgBlock, curHashStr, true
	}
	return nil, "", false
}

func createAuxBlock(params []interface{}) map[string]interface{} {
	msgBlock, curHashStr, _ := generateAuxBlock(config.Parameters.PowConfiguration.PayToAddr)
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

	switch params[0].(type) {
	case string:
		Pow.PayToAddr = params[0].(string)

		preHash := ledger.DefaultLedger.Blockchain.CurrentBlockHash()
		preHashStr := BytesToHexString(preHash.ToArray())

		SendToAux := AuxBlock{
			ChainId:           1,
			Height:            node.GetHeight(),
			CoinBaseValue:     1,                                          //transaction content
			Bits:              fmt.Sprintf("%x", msgBlock.Blockdata.Bits), //difficulty
			Hash:              curHashStr,
			PreviousBlockHash: preHashStr}
		return ElaRpc(&SendToAux)

	default:
		return ElaRpcInvalidParameter

	}
}

func getInfo(params []interface{}) map[string]interface{} {
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
		Blocks:      node.GetHeight(),
		Timeoffset:  0,
		Connections: node.GetConnectionCnt(),
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

func auxHelp(params []interface{}) map[string]interface{} {

	//TODO  and description for this rpc-interface
	return ElaRpc("createauxblock==submitauxblock")
}

func getVersion(params []interface{}) map[string]interface{} {
	return ElaRpc(config.Version)
}

func toggleCpuMining(params []interface{}) map[string]interface{} {
	var isMining bool
	switch params[0].(type) {
	case bool:
		isMining = params[0].(bool)

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

func manualCpuMining(params []interface{}) map[string]interface{} {
	var numBlocks uint32
	switch params[0].(type) {
	case float64:
		numBlocks = uint32(params[0].(float64))
	default:
		return ElaRpcInvalidParameter
	}

	if numBlocks == 0 {
		return ElaRpcInvalidParameter
	}

	ret := make([]string, numBlocks)

	blockHashes, err := Pow.ManualMining(numBlocks)
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

// A JSON example for sendrawtransaction method as following:
//   {"jsonrpc": "2.0", "method": "sendrawtransaction", "params": ["raw transactioin in hex"], "id": 0}
func sendRawTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return ElaRpcNil
	}
	var hash Uint256
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := HexStringToBytes(str)
		if err != nil {
			return ElaRpcInvalidParameter
		}
		var txn tx.Transaction
		if err := txn.Deserialize(bytes.NewReader(hex)); err != nil {
			return ElaRpcInvalidTransaction
		}
		hash = txn.Hash()
		if errCode := VerifyAndSendTx(&txn); errCode != Success {
			return ElaRpc(errCode.Error())
		}
	default:
		return ElaRpcInvalidParameter
	}
	return ElaRpc(BytesToHexString(hash.ToArrayReverse()))
}

// A JSON example for submitblock method as following:
//   {"jsonrpc": "2.0", "method": "submitblock", "params": ["raw block in hex"], "id": 0}
func submitBlock(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return ElaRpcNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, _ := HexStringToBytes(str)
		var block ledger.Block
		if err := block.Deserialize(bytes.NewReader(hex)); err != nil {
			return ElaRpcInvalidBlock
		}
		if _, _, err := ledger.DefaultLedger.Blockchain.AddBlock(&block); err != nil {
			return ElaRpcInvalidBlock
		}
		if err := node.Xmit(&block); err != nil {
			return ElaRpcInternalError
		}
	default:
		return ElaRpcInvalidParameter
	}
	return ElaRpcSuccess
}
