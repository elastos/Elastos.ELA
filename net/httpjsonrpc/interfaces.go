package httpjsonrpc

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"DNA_POW/account"
	. "DNA_POW/common"
	"DNA_POW/common/config"
	"DNA_POW/common/log"
	"DNA_POW/core/asset"
	"DNA_POW/core/ledger"
	tx "DNA_POW/core/transaction"
	"DNA_POW/core/transaction/payload"
	. "DNA_POW/errors"
	"DNA_POW/sdk"

	"github.com/mitchellh/go-homedir"
)

const (
	RANDBYTELEN = 4
)

func TransArryByteToHexString(ptx *tx.Transaction) *Transactions {

	trans := new(Transactions)
	trans.TxType = ptx.TxType
	trans.PayloadVersion = ptx.PayloadVersion
	trans.Payload = TransPayloadToHex(ptx.Payload)

	n := 0
	trans.Attributes = make([]TxAttributeInfo, len(ptx.Attributes))
	for _, v := range ptx.Attributes {
		trans.Attributes[n].Usage = v.Usage
		trans.Attributes[n].Data = ToHexString(v.Data)
		n++
	}

	n = 0
	trans.UTXOInputs = make([]UTXOTxInputInfo, len(ptx.UTXOInputs))
	for _, v := range ptx.UTXOInputs {
		trans.UTXOInputs[n].ReferTxID = ToHexString(v.ReferTxID.ToArray())
		trans.UTXOInputs[n].ReferTxOutputIndex = v.ReferTxOutputIndex
		n++
	}

	n = 0
	trans.BalanceInputs = make([]BalanceTxInputInfo, len(ptx.BalanceInputs))
	for _, v := range ptx.BalanceInputs {
		trans.BalanceInputs[n].AssetID = ToHexString(v.AssetID.ToArray())
		trans.BalanceInputs[n].Value = v.Value
		trans.BalanceInputs[n].ProgramHash = ToHexString(v.ProgramHash.ToArray())
		n++
	}

	n = 0
	trans.Outputs = make([]TxoutputInfo, len(ptx.Outputs))
	for _, v := range ptx.Outputs {
		trans.Outputs[n].AssetID = ToHexString(v.AssetID.ToArray())
		trans.Outputs[n].Value = asset.Fixed64toAssetValue(v.Value)
		address, _ := v.ProgramHash.ToAddress()
		trans.Outputs[n].Address = address
		n++
	}

	n = 0
	trans.Programs = make([]ProgramInfo, len(ptx.Programs))
	for _, v := range ptx.Programs {
		trans.Programs[n].Code = ToHexString(v.Code)
		trans.Programs[n].Parameter = ToHexString(v.Parameter)
		n++
	}

	n = 0
	trans.AssetOutputs = make([]TxoutputMap, len(ptx.AssetOutputs))
	for k, v := range ptx.AssetOutputs {
		trans.AssetOutputs[n].Key = k
		trans.AssetOutputs[n].Txout = make([]TxoutputInfo, len(v))
		for m := 0; m < len(v); m++ {
			trans.AssetOutputs[n].Txout[m].AssetID = ToHexString(v[m].AssetID.ToArray())
			trans.AssetOutputs[n].Txout[m].Value = asset.Fixed64toAssetValue(v[m].Value)
			address, _ := v[m].ProgramHash.ToAddress()
			trans.AssetOutputs[n].Txout[m].Address = address
		}
		n += 1
	}

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

	mhash := ptx.Hash()
	trans.Hash = ToHexString(mhash.ToArray())

	return trans
}
func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return dir
}
func getBestBlockHash(params []interface{}) map[string]interface{} {
	hash := ledger.DefaultLedger.Blockchain.CurrentBlockHash()
	return DnaRpc(ToHexString(hash.ToArray()))
}

// Input JSON string examples for getblock method as following:
//   {"jsonrpc": "2.0", "method": "getblock", "params": [1], "id": 0}
//   {"jsonrpc": "2.0", "method": "getblock", "params": ["aabbcc.."], "id": 0}
func getBlock(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return DnaRpcNil
	}
	var err error
	var hash Uint256
	switch (params[0]).(type) {
	// block height
	case float64:
		index := uint32(params[0].(float64))
		hash, err = ledger.DefaultLedger.Store.GetBlockHash(index)
		if err != nil {
			return DnaRpcUnknownBlock
		}
	// block hash
	case string:
		str := params[0].(string)
		hex, err := hex.DecodeString(str)
		if err != nil {
			return DnaRpcInvalidParameter
		}
		if err := hash.Deserialize(bytes.NewReader(hex)); err != nil {
			return DnaRpcInvalidTransaction
		}
	default:
		return DnaRpcInvalidParameter
	}

	block, err := ledger.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		return DnaRpcUnknownBlock
	}

	blockHead := &BlockHead{
		Version:          block.Blockdata.Version,
		PrevBlockHash:    ToHexString(block.Blockdata.PrevBlockHash.ToArray()),
		TransactionsRoot: ToHexString(block.Blockdata.TransactionsRoot.ToArray()),
		Timestamp:        block.Blockdata.Timestamp,
		Height:           block.Blockdata.Height,
		ConsensusData:    block.Blockdata.ConsensusData,
		NextBookKeeper:   ToHexString(block.Blockdata.NextBookKeeper.ToArray()),
		Program: ProgramInfo{
			Code:      ToHexString(block.Blockdata.Program.Code),
			Parameter: ToHexString(block.Blockdata.Program.Parameter),
		},
		Hash: ToHexString(hash.ToArray()),
	}

	trans := make([]*Transactions, len(block.Transactions))
	for i := 0; i < len(block.Transactions); i++ {
		trans[i] = TransArryByteToHexString(block.Transactions[i])
	}

	b := BlockInfo{
		Hash:         ToHexString(hash.ToArray()),
		BlockData:    blockHead,
		Transactions: trans,
	}
	return DnaRpc(b)
}

func getBlockCount(params []interface{}) map[string]interface{} {
	return DnaRpc(ledger.DefaultLedger.Blockchain.BlockHeight + 1)
}

// A JSON example for getblockhash method as following:
//   {"jsonrpc": "2.0", "method": "getblockhash", "params": [1], "id": 0}
func getBlockHash(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return DnaRpcNil
	}
	switch params[0].(type) {
	case float64:
		height := uint32(params[0].(float64))
		hash, err := ledger.DefaultLedger.Store.GetBlockHash(height)
		if err != nil {
			return DnaRpcUnknownBlock
		}
		return DnaRpc(fmt.Sprintf("%016x", hash))
	default:
		return DnaRpcInvalidParameter
	}
}

func getConnectionCount(params []interface{}) map[string]interface{} {
	return DnaRpc(node.GetConnectionCnt())
}

func getRawMemPool(params []interface{}) map[string]interface{} {
	txs := []*Transactions{}
	txpool := node.GetTxnPool(false)
	for _, t := range txpool {
		txs = append(txs, TransArryByteToHexString(t))
	}
	if len(txs) == 0 {
		return DnaRpcNil
	}
	return DnaRpc(txs)
}

// A JSON example for getrawtransaction method as following:
//   {"jsonrpc": "2.0", "method": "getrawtransaction", "params": ["transactioin hash in hex"], "id": 0}
func getRawTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return DnaRpcNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := hex.DecodeString(str)
		if err != nil {
			return DnaRpcInvalidParameter
		}
		var hash Uint256
		err = hash.Deserialize(bytes.NewReader(hex))
		if err != nil {
			return DnaRpcInvalidTransaction
		}
		tx, err := ledger.DefaultLedger.Store.GetTransaction(hash)
		if err != nil {
			return DnaRpcUnknownTransaction
		}
		tran := TransArryByteToHexString(tx)
		return DnaRpc(tran)
	default:
		return DnaRpcInvalidParameter
	}
}

// A JSON example for sendrawtransaction method as following:
//   {"jsonrpc": "2.0", "method": "sendrawtransaction", "params": ["raw transactioin in hex"], "id": 0}
func sendRawTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return DnaRpcNil
	}
	var hash Uint256
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := hex.DecodeString(str)
		if err != nil {
			return DnaRpcInvalidParameter
		}
		var txn tx.Transaction
		if err := txn.Deserialize(bytes.NewReader(hex)); err != nil {
			return DnaRpcInvalidTransaction
		}
		hash = txn.Hash()
		if errCode := VerifyAndSendTx(&txn); errCode != ErrNoError {
			return DnaRpcInternalError
		}
	default:
		return DnaRpcInvalidParameter
	}
	return DnaRpc(ToHexString(hash.ToArray()))
}

func getTxout(params []interface{}) map[string]interface{} {
	//TODO
	return DnaRpcUnsupported
}

// A JSON example for submitblock method as following:
//   {"jsonrpc": "2.0", "method": "submitblock", "params": ["raw block in hex"], "id": 0}
func submitBlock(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return DnaRpcNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, _ := hex.DecodeString(str)
		var block ledger.Block
		if err := block.Deserialize(bytes.NewReader(hex)); err != nil {
			return DnaRpcInvalidBlock
		}
		if err := ledger.DefaultLedger.Blockchain.AddBlock(&block); err != nil {
			return DnaRpcInvalidBlock
		}
		if err := node.Xmit(&block); err != nil {
			return DnaRpcInternalError
		}
	default:
		return DnaRpcInvalidParameter
	}
	return DnaRpcSuccess
}

func getNeighbor(params []interface{}) map[string]interface{} {
	addr, _ := node.GetNeighborAddrs()
	return DnaRpc(addr)
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
	return DnaRpc(n)
}

func startConsensus(params []interface{}) map[string]interface{} {
	if err := dBFT.Start(); err != nil {
		return DnaRpcFailed
	}
	return DnaRpcSuccess
}

func stopConsensus(params []interface{}) map[string]interface{} {
	if err := dBFT.Halt(); err != nil {
		return DnaRpcFailed
	}
	return DnaRpcSuccess
}

func sendSampleTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return DnaRpcNil
	}
	var txType string
	switch params[0].(type) {
	case string:
		txType = params[0].(string)
	default:
		return DnaRpcInvalidParameter
	}

	issuer, err := account.NewAccount()
	if err != nil {
		return DnaRpc("Failed to create account")
	}
	admin := issuer

	rbuf := make([]byte, RANDBYTELEN)
	rand.Read(rbuf)
	switch string(txType) {
	case "perf":
		num := 1
		if len(params) == 2 {
			switch params[1].(type) {
			case float64:
				num = int(params[1].(float64))
			}
		}
		for i := 0; i < num; i++ {
			regTx := NewRegTx(ToHexString(rbuf), i, admin, issuer)
			SignTx(admin, regTx)
			VerifyAndSendTx(regTx)
		}
		return DnaRpc(fmt.Sprintf("%d transaction(s) was sent", num))
	default:
		return DnaRpc("Invalid transacion type")
	}
}

func setDebugInfo(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return DnaRpcInvalidParameter
	}
	switch params[0].(type) {
	case float64:
		level := params[0].(float64)
		if err := log.Log.SetDebugLevel(int(level)); err != nil {
			return DnaRpcInvalidParameter
		}
	default:
		return DnaRpcInvalidParameter
	}
	return DnaRpcSuccess
}

func submitAuxBlock(params []interface{}) map[string]interface{} {
	auxPow, blockHash := "", ""
	switch params[0].(type) {
	case string:
		blockHash = params[0].(string)
	default:
		return DnaRpcInvalidParameter
	}

	switch params[1].(type) {
	case string:
		auxPow = params[1].(string)
	default:
		return DnaRpcInvalidParameter
	}
	return DnaRpcSuccess
}

func createAuxBlock(params []interface{}) map[string]interface{} {

	type AuxBlock struct {
		ChainId           int    `json:"chainid"`
		Height            int    `json:"height"`
		CoinBaseValue     int    `json:"coinbasevalue"`
		Bits              string `json:"bits"`
		Hash              string `json:"hash"`
		PreviousBlockHash string `json:"previousblockhash"`
	}

	switch params[0].(type) {
	case string:
		//coinbaseAddr := params[0].(string)
		SendToAux := AuxBlock{
			ChainId:           1,
			Height:            1,
			CoinBaseValue:     11,
			Bits:              "bits",
			Hash:              "temp-hash for test",
			PreviousBlockHash: "previousblockhash for test"}
		return DnaRpc(&SendToAux)

	default:
		return DnaRpc("Hello createAuxBlock")

	}
	return DnaRpc("Hello createAuxBlock")
}

func getInfo(params []interface{}) map[string]interface{} {
	RetVal := struct {
		Version         int    `josn:"version"`
		Protocolversion int    `josn:"protocolversion"`
		Walletversion   int    `josn:"walletversion"`
		Balance         int    `josn:"balance"`
		Blocks          int    `json:"blocks"`
		Timeoffset      int    `json:"timeoffset"`
		Connections     int    `json:"connections"`
		Proxy           string `json:"proxy"`
		Difficulty      int    `json:"difficulty"`
		Testnet         bool   `json:"testnet"`
		Keypoololdest   int    `json:"keypoololdest"`
		Keypoolsize     int    `json:"keypoolsize"`
		Unlocked_until  int    `json:"unlocked_until"`
		Paytxfee        int    `json:"paytxfee"`
		Relayfee        int    `json:"relayfee"`
		Errors          string `json:"errors"`
	}{
		Version:         1,
		Protocolversion: 1,
		Walletversion:   1,
		Balance:         1,
		Blocks:          1,
		Timeoffset:      1,
		Connections:     1,
		Proxy:           "5526",
		Difficulty:      1234567,
		Testnet:         true,
		Keypoololdest:   1,
		Keypoolsize:     1,
		Unlocked_until:  1,
		Paytxfee:        1,
		Relayfee:        1,
		Errors:          "no error"}
	return DnaRpc(&RetVal)
}

func auxHelp(params []interface{}) map[string]interface{} {

	return DnaRpc("createauxblock==submitauxblock")
}

func getVersion(params []interface{}) map[string]interface{} {
	return DnaRpc(config.Version)
}

func uploadDataFile(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return DnaRpcNil
	}

	rbuf := make([]byte, 4)
	rand.Read(rbuf)
	tmpname := hex.EncodeToString(rbuf)

	str := params[0].(string)

	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return DnaRpcInvalidParameter
	}
	f, err := os.OpenFile(tmpname, os.O_WRONLY|os.O_CREATE, 0664)
	if err != nil {
		return DnaRpcIOError
	}
	defer f.Close()
	f.Write(data)

	refpath, err := AddFileIPFS(tmpname, true)
	if err != nil {
		return DnaRpcAPIError
	}

	return DnaRpc(refpath)

}

func regDataFile(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return DnaRpcNil
	}
	var hash Uint256
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := hex.DecodeString(str)
		if err != nil {
			return DnaRpcInvalidParameter
		}
		var txn tx.Transaction
		if err := txn.Deserialize(bytes.NewReader(hex)); err != nil {
			return DnaRpcInvalidTransaction
		}

		hash = txn.Hash()
		if errCode := VerifyAndSendTx(&txn); errCode != ErrNoError {
			return DnaRpcInternalError
		}
	default:
		return DnaRpcInvalidParameter
	}
	return DnaRpc(ToHexString(hash.ToArray()))
}

func catDataRecord(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return DnaRpcNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		b, err := hex.DecodeString(str)
		if err != nil {
			return DnaRpcInvalidParameter
		}
		var hash Uint256
		err = hash.Deserialize(bytes.NewReader(b))
		if err != nil {
			return DnaRpcInvalidTransaction
		}
		tx, err := ledger.DefaultLedger.Store.GetTransaction(hash)
		if err != nil {
			return DnaRpcUnknownTransaction
		}
		tran := TransArryByteToHexString(tx)
		info := tran.Payload.(*DataFileInfo)
		//ref := string(record.RecordData[:])
		return DnaRpc(info)
	default:
		return DnaRpcInvalidParameter
	}
}

func getDataFile(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return DnaRpcNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := hex.DecodeString(str)
		if err != nil {
			return DnaRpcInvalidParameter
		}
		var hash Uint256
		err = hash.Deserialize(bytes.NewReader(hex))
		if err != nil {
			return DnaRpcInvalidTransaction
		}
		tx, err := ledger.DefaultLedger.Store.GetTransaction(hash)
		if err != nil {
			return DnaRpcUnknownTransaction
		}

		tran := TransArryByteToHexString(tx)
		info := tran.Payload.(*DataFileInfo)

		err = GetFileIPFS(info.IPFSPath, info.Filename)
		if err != nil {
			return DnaRpcAPIError
		}
		//TODO: shoud return download address
		return DnaRpcSuccess
	default:
		return DnaRpcInvalidParameter
	}
}

func searchTransactions(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return DnaRpcNil
	}
	var programHash string
	switch params[0].(type) {
	case string:
		programHash = params[0].(string)
	default:
		return DnaRpcInvalidParameter
	}

	resp := make(map[string]string)
	height := ledger.DefaultLedger.GetLocalBlockChainHeight()
	var i uint32
	for i = 1; i <= height; i++ {
		block, err := ledger.DefaultLedger.GetBlockWithHeight(i)
		if err != nil {
			return DnaRpcInternalError
		}
		// skip the bookkeeping transaction
		for _, t := range block.Transactions[1:] {
			switch t.TxType {
			case tx.RegisterAsset:
				regPayload := t.Payload.(*payload.RegisterAsset)
				controller := ToHexString(regPayload.Controller.ToArray())
				if controller == programHash {
					txHash := t.Hash()
					txid := ToHexString(txHash.ToArray())
					resp[txid] = "registration"
				}
			case tx.IssueAsset:
				for _, v := range t.Outputs {
					regTxn, err := ledger.DefaultLedger.Store.GetTransaction(v.AssetID)
					if err != nil {
						log.Warn("Can not find asset")
						continue

					}
					regPayload := regTxn.Payload.(*payload.RegisterAsset)
					controller := ToHexString(regPayload.Controller.ToArray())
					if controller == programHash {
						txHash := t.Hash()
						txid := ToHexString(txHash.ToArray())
						resp[txid] = "issuance"
					}
				}
			case tx.TransferAsset:
				transferTxnProgram := t.GetPrograms()[0]
				transferTxnProgramHash, _ := ToCodeHash(transferTxnProgram.Code)
				transferTxnProgramHashStr := ToHexString(transferTxnProgramHash.ToArray())
				fmt.Println(transferTxnProgramHashStr)
				if programHash == transferTxnProgramHashStr {
					txHash := t.Hash()
					txid := ToHexString(txHash.ToArray())
					resp[txid] = "transfer"
				}
			default:
				continue
			}
		}
	}

	return DnaRpc(resp)
}

var walletInstance *account.ClientImpl

func getWalletDir() string {
	home, _ := homedir.Dir()
	return home + "/.wallet/"
}

func createWallet(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return DnaRpcNil
	}
	var password []byte
	switch params[0].(type) {
	case string:
		password = []byte(params[0].(string))
	default:
		return DnaRpcInvalidParameter
	}
	walletDir := getWalletDir()
	if !FileExisted(walletDir) {
		err := os.MkdirAll(walletDir, 0755)
		if err != nil {
			return DnaRpcInternalError
		}
	}
	walletPath := walletDir + "wallet.dat"
	if FileExisted(walletPath) {
		return DnaRpcWalletAlreadyExists
	}
	_, err := account.Create(walletPath, password)
	if err != nil {
		return DnaRpcFailed
	}
	return DnaRpcSuccess
}

func openWallet(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return DnaRpcNil
	}
	var password []byte
	switch params[0].(type) {
	case string:
		password = []byte(params[0].(string))
	default:
		return DnaRpcInvalidParameter
	}
	resp := make(map[string]string)
	walletPath := getWalletDir() + "wallet.dat"
	if !FileExisted(walletPath) {
		resp["success"] = "false"
		resp["message"] = "wallet doesn't exist"
		return DnaRpc(resp)
	}
	wallet, err := account.Open(walletPath, password)
	if err != nil {
		resp["success"] = "false"
		resp["message"] = "password wrong"
		return DnaRpc(resp)
	}
	walletInstance = wallet
	programHash, err := wallet.LoadStoredData("ProgramHash")
	if err != nil {
		resp["success"] = "false"
		resp["message"] = "wallet file broken"
		return DnaRpc(resp)
	}
	resp["success"] = "true"
	resp["message"] = ToHexString(programHash)
	return DnaRpc(resp)
}

func closeWallet(params []interface{}) map[string]interface{} {
	walletInstance = nil
	return DnaRpcSuccess
}

func recoverWallet(params []interface{}) map[string]interface{} {
	if len(params) < 2 {
		return DnaRpcNil
	}
	var privateKey string
	var walletPassword string
	switch params[0].(type) {
	case string:
		privateKey = params[0].(string)
	default:
		return DnaRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		walletPassword = params[1].(string)
	default:
		return DnaRpcInvalidParameter
	}
	walletDir := getWalletDir()
	if !FileExisted(walletDir) {
		err := os.MkdirAll(walletDir, 0755)
		if err != nil {
			return DnaRpcInternalError
		}
	}
	walletName := fmt.Sprintf("wallet-%s-recovered.dat", time.Now().Format("2006-01-02-15-04-05"))
	walletPath := walletDir + walletName
	if FileExisted(walletPath) {
		return DnaRpcWalletAlreadyExists
	}
	_, err := account.Recover(walletPath, []byte(walletPassword), privateKey)
	if err != nil {
		return DnaRpc("wallet recovery failed")
	}

	return DnaRpcSuccess
}

func getWalletKey(params []interface{}) map[string]interface{} {
	if walletInstance == nil {
		return DnaRpc("open wallet first")
	}
	account, _ := walletInstance.GetDefaultAccount()
	encodedPublickKey, _ := account.PublicKey.EncodePoint(true)
	resp := make(map[string]string)
	resp["PublicKey"] = ToHexString(encodedPublickKey)
	resp["PrivateKey"] = ToHexString(account.PrivateKey)
	resp["ProgramHash"] = ToHexString(account.ProgramHash.ToArray())

	return DnaRpc(resp)
}

func addAccount(params []interface{}) map[string]interface{} {
	if walletInstance == nil {
		return DnaRpc("open wallet first")
	}
	account, err := walletInstance.CreateAccount()
	if err != nil {
		return DnaRpc("create account error:" + err.Error())
	}

	if err := walletInstance.CreateContract(account); err != nil {
		return DnaRpc("create contract error:" + err.Error())
	}

	return DnaRpc(ToHexString(account.ProgramHash.ToArray()))
}

func deleteAccount(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return DnaRpcNil
	}
	var programHash string
	switch params[0].(type) {
	case string:
		programHash = params[0].(string)
	default:
		return DnaRpcInvalidParameter
	}
	if walletInstance == nil {
		return DnaRpc("open wallet first")
	}
	phBytes, _ := HexToBytes(programHash)
	pbUint160, _ := Uint160ParseFromBytes(phBytes)
	if err := walletInstance.DeleteAccount(pbUint160); err != nil {
		return DnaRpc("Delete account error:" + err.Error())
	}
	if err := walletInstance.DeleteContract(pbUint160); err != nil {
		return DnaRpc("Delete contract error:" + err.Error())
	}
	if err := walletInstance.DeleteCoinsData(pbUint160); err != nil {
		return DnaRpc("Delete coins error:" + err.Error())
	}

	return DnaRpc(true)
}

func makeRegTxn(params []interface{}) map[string]interface{} {
	if len(params) < 2 {
		return DnaRpcNil
	}
	var assetName string
	var assetValue float64
	switch params[0].(type) {
	case string:
		assetName = params[0].(string)
	default:
		return DnaRpcInvalidParameter
	}
	switch params[1].(type) {
	case float64:
		assetValue = params[1].(float64)
	default:
		return DnaRpcInvalidParameter
	}
	if walletInstance == nil {
		return DnaRpc("open wallet first")
	}

	regTxn, err := sdk.MakeRegTransaction(walletInstance, assetName, assetValue)
	if err != nil {
		return DnaRpcInternalError
	}

	if errCode := VerifyAndSendTx(regTxn); errCode != ErrNoError {
		return DnaRpcInvalidTransaction
	}
	return DnaRpc(true)
}

func makeIssueTxn(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return DnaRpcNil
	}
	var asset string
	var value float64
	var address string
	switch params[0].(type) {
	case string:
		asset = params[0].(string)
	default:
		return DnaRpcInvalidParameter
	}
	switch params[1].(type) {
	case float64:
		value = params[1].(float64)
	default:
		return DnaRpcInvalidParameter
	}
	switch params[2].(type) {
	case string:
		address = params[2].(string)
	default:
		return DnaRpcInvalidParameter
	}
	if walletInstance == nil {
		return DnaRpc("open wallet first")
	}
	assetID, _ := StringToUint256(asset)
	issueTxn, err := sdk.MakeIssueTransaction(walletInstance, assetID, address, value)
	if err != nil {
		return DnaRpcInternalError
	}

	if errCode := VerifyAndSendTx(issueTxn); errCode != ErrNoError {
		return DnaRpcInvalidTransaction
	}

	return DnaRpc(true)
}

func makeTransferTxn(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return DnaRpcNil
	}
	var asset string
	var value float64
	var address string
	switch params[0].(type) {
	case string:
		asset = params[0].(string)
	default:
		return DnaRpcInvalidParameter
	}
	switch params[1].(type) {
	case float64:
		value = params[1].(float64)
	default:
		return DnaRpcInvalidParameter
	}
	switch params[2].(type) {
	case string:
		address = params[2].(string)
	default:
		return DnaRpcInvalidParameter
	}

	if walletInstance == nil {
		return DnaRpc("open wallet first")
	}

	batchOut := sdk.BatchOut{
		Address: address,
		Value:   value,
	}
	assetID, _ := StringToUint256(asset)
	txn, err := sdk.MakeTransferTransaction(walletInstance, assetID, batchOut)
	if err != nil {
		return DnaRpcInternalError
	}

	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return DnaRpcInvalidTransaction
	}

	return DnaRpc(true)
}

func getBalance(params []interface{}) map[string]interface{} {
	if walletInstance == nil {
		return DnaRpc("open wallet first")
	}
	type AssetInfo struct {
		AssetID string
		Value   float64
	}
	balances := make(map[string][]*AssetInfo)
	accounts := walletInstance.GetAccounts()
	coins := walletInstance.GetCoins()
	for _, account := range accounts {
		assetList := []*AssetInfo{}
		programHash := account.ProgramHash
		for _, coin := range coins {
			if programHash == coin.Output.ProgramHash {
				var existed bool
				assetString := ToHexString(coin.Output.AssetID.ToArray())
				for _, info := range assetList {
					if info.AssetID == assetString {
						info.Value += asset.Fixed64toAssetValue(coin.Output.Value)
						existed = true
						break
					}
				}
				if !existed {
					assetList = append(assetList, &AssetInfo{AssetID: assetString, Value: asset.Fixed64toAssetValue(coin.Output.Value)})
				}
			}
		}
		address, _ := programHash.ToAddress()
		balances[address] = assetList
	}

	return DnaRpc(balances)
}
