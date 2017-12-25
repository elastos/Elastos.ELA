package elaapi

import (
	"bytes"
	"encoding/hex"
	"strconv"
	"strings"
	"fmt"

	. "Elastos.ELA/cli/common"
	. "Elastos.ELA/common"
	"Elastos.ELA/core/auxpow"
	"Elastos.ELA/core/ledger"
	tx "Elastos.ELA/core/transaction"
	"Elastos.ELA/net/httpjsonrpc"

	. "github.com/bitly/go-simplejson"
	"github.com/yuin/gopher-lua"
)

func Loader(L *lua.LState) int {
	// register functions to the table
	mod := L.SetFuncs(L.NewTable(), exports)
	// register other stuff
	L.SetField(mod, "version", lua.LString("0.1"))

	// returns the module
	L.Push(mod)
	return 1
}

var exports = map[string]lua.LGFunction{
	"hexStrReverse":           HexReverse,
	"sendRawTx":               sendRawTx,
	"getRawTx":                getRawTx,
	"getAssetID":              getAssetID,
	"getUnspendOutput":        getUnspendOutput,
	"getCoinbaseHashByHeight": getCoinbaseHashByHeight,
	"getBlockByHeight":        getBlockByHeight,
	"getBlockByHash":          getBlockByHash,
	"getCurrentBlockHash":     getCurrentBlockHash,
	"getCurrentBlockHeight":   getCurrentBlockHeight,
	"getCurrentTimeStamp":     getCurrentTimeStamp,
	"submitBlock":             sendRawBlock,
	"togglemining":            toggleMining,
	"discreteMining":          DiscreteMining,
	"getLatestBits":           getLatestBits,
}

func RegisterDataType(L *lua.LState) int {
	RegisterAssetType(L)
	RegisterBalanceTxInputType(L)
	RegisterClientType(L)
	RegisterFunctionCodeType(L)
	RegisterTxAttributeType(L)
	RegisterUTXOTxInputType(L)
	RegisterTxOutputType(L)
	RegisterCoinBaseType(L)
	RegisterTransferAssetType(L)
	RegisterRegisterAssetType(L)
	RegisterRecordType(L)
	RegisterDeployCodeType(L)
	RegisterTransactionType(L)
	RegisterBlockdataType(L)
	RegisterBlockType(L)
	return 0
}

func HexReverse(L *lua.LState) int {
	str := L.ToString(1)
	ret, _ := HexStringToBytesReverse(str)
	retHex := hex.EncodeToString(ret)

	L.Push(lua.LString(retHex))
	return 1
}

//func assertRetCode(L *lua.LState) int {
//	expect := L.ToString(1)
//	real := L.ToString(2)
//
//	return 1
//}

func getRawTx(L *lua.LState) int {
	hash := L.ToString(1)
	resp, _ := httpjsonrpc.Call(Address(), "getrawtransaction", 0, []interface{}{hash})
	FormatOutput(resp)

	js, _ := NewJson(resp)
	res := js.Get("result")
	txn, timeStamp, confirm := parseTxMsg(res)

	ud := L.NewUserData()
	ud.Value = txn
	L.SetMetatable(ud, L.GetTypeMetatable(luaTransactionTypeName))
	L.Push(ud)

	L.Push(lua.LNumber(timeStamp))
	L.Push(lua.LNumber(confirm))

	return 3
}

func sendRawTx(L *lua.LState) int {
	fmt.Println("send raw tx")
	txn := checkTransaction(L, 1)

	var buffer bytes.Buffer
	txn.Serialize(&buffer)
	txHex := hex.EncodeToString(buffer.Bytes())

	resp, _ := httpjsonrpc.Call(Address(), "sendrawtransaction", 0, []interface{}{txHex})
	js, _ := NewJson(resp)
	res, err := js.Get("result").String()
	if err != nil {
		FormatOutput(resp)
		return 0
	}
	L.Push(lua.LString(res))

	return 1
}

func getCurrentTimeStamp(L *lua.LState) int {
	resp, _ := httpjsonrpc.Call(Address(), "getblockcount", 0, []interface{}{})
	js, _ := NewJson(resp)
	height, _ := js.Get("result").Int()
	currentHeight := height - 1
	resp, _ = httpjsonrpc.Call(Address(), "getblock", 0, []interface{}{currentHeight})

	//FormatOutput(resp)
	js, _ = NewJson(resp)
	res, _ := js.Get("result").Get("BlockData").Get("Timestamp").Int()

	L.Push(lua.LNumber(res))
	return 1
}

func getCoinbaseHashByHeight(L *lua.LState) int {
	height := L.ToInt(1)
	resp, _ := httpjsonrpc.Call(Address(), "getblock", 0, []interface{}{height})

	//FormatOutput(resp)
	js, _ := NewJson(resp)
	res, _ := js.Get("result").Get("Transactions").GetIndex(0).Get("Hash").String()

	L.Push(lua.LString(res))
	return 1
}

func getAssetID(L *lua.LState) int {
	height := 0
	resp, _ := httpjsonrpc.Call(Address(), "getblock", 0, []interface{}{height})

	js, _ := NewJson(resp)
	res, _ := js.Get("result").Get("Transactions").GetIndex(0).Get("Outputs").GetIndex(0).Get("AssetID").String()

	L.Push(lua.LString(res))
	return 1
}

func getUnspendOutput(L *lua.LState) int {
	addr := L.ToString(1)
	assetID := L.ToString(2)
	resp, _ := httpjsonrpc.Call(Address(), "getunspendoutput", 0, []interface{}{addr, assetID})
	FormatOutput(resp)

	//js, _ := NewJson(resp)
	//res, _ := js.Get("result").StringArray()
	//
	//fmt.Println(res)
	//L.Push(lua.LString(res))
	//return 1
	return 0
}

func getBlockByHeight(L *lua.LState) int {
	height := L.ToInt(1)
	resp, _ := httpjsonrpc.Call(Address(), "getblock", 0, []interface{}{height})
	b := parseBlockMsg(resp)
	ud := L.NewUserData()
	ud.Value = b
	L.SetMetatable(ud, L.GetTypeMetatable(luaBlockTypeName))
	L.Push(ud)

	js, _ := NewJson(resp)
	confirm, _ := js.Get("result").Get("Confirminations").Int()
	L.Push(lua.LNumber(confirm))

	return 2
}

func getBlockByHash(L *lua.LState) int {
	hash := L.ToString(1)
	resp, _ := httpjsonrpc.Call(Address(), "getblock", 0, []interface{}{hash})
	b := parseBlockMsg(resp)
	ud := L.NewUserData()
	ud.Value = b
	L.SetMetatable(ud, L.GetTypeMetatable(luaBlockTypeName))
	L.Push(ud)

	js, _ := NewJson(resp)
	confirm, _ := js.Get("result").Get("Confirminations").Int()
	L.Push(lua.LNumber(confirm))

	return 2
}

func getCurrentBlockHash(L *lua.LState) int {
	resp, _ := httpjsonrpc.Call(Address(), "getbestblockhash", 0, []interface{}{})

	js, _ := NewJson(resp)
	blockHash, _ := js.Get("result").String()
	L.Push(lua.LString(blockHash))
	return 1
}

func getCurrentBlockHeight(L *lua.LState) int {
	resp, _ := httpjsonrpc.Call(Address(), "getblockcount", 0, []interface{}{})

	js, _ := NewJson(resp)
	height, _ := js.Get("result").Int()
	L.Push(lua.LNumber(height - 1))
	return 1
}

func sendRawBlock(L *lua.LState) int {
	block := checkBlock(L, 1)

	var buffer bytes.Buffer
	block.Serialize(&buffer)
	bHex := hex.EncodeToString(buffer.Bytes())

	resp, _ := httpjsonrpc.Call(Address(), "submitblock", 0, []interface{}{bHex})
	//	FormatOutput(resp)
	js, _ := NewJson(resp)
	res, err := js.Get("result").Bool()
	if err != nil {
		FormatOutput(resp)
		return 0
	}
	L.Push(lua.LBool(res))
	return 1
}

func toggleMining(L *lua.LState) int {
	isMining := L.ToBool(1)
	resp, _ := httpjsonrpc.Call(Address(), "togglecpumining", 0, []interface{}{isMining})
	FormatOutput(resp)
	//TODO return val

	return 0

}

func DiscreteMining(L *lua.LState) int {
	numBlocks := L.ToInt(1)
	resp, _ := httpjsonrpc.Call(Address(), "discretemining", 0, []interface{}{numBlocks})
	FormatOutput(resp)

	return 0
}

func parseBlockMsg(resp []byte) *ledger.Block {
	js, _ := NewJson(resp)
	version, _ := js.Get("result").Get("BlockData").Get("Version").Int()
	prevHash, _ := js.Get("result").Get("BlockData").Get("PrevBlockHash").String()
	txRoot, _ := js.Get("result").Get("BlockData").Get("TransactionsRoot").String()
	timeStamp, _ := js.Get("result").Get("BlockData").Get("Timestamp").Int()
	height, _ := js.Get("result").Get("BlockData").Get("Height").Int()
	bits, _ := js.Get("result").Get("BlockData").Get("Bits").Int()
	nonce, _ := js.Get("result").Get("BlockData").Get("Nonce").Int()

	bd := &ledger.Blockdata{
		Version:          uint32(version),
		PrevBlockHash:    U256FromString(prevHash),
		TransactionsRoot: U256FromString(txRoot),
		Timestamp:        uint32(timeStamp),
		Bits:             uint32(bits),
		Height:           uint32(height),
		Nonce:            uint32(nonce),
		AuxPow:           auxpow.AuxPow{},
	}

	b := &ledger.Block{
		Blockdata:    bd,
		Transactions: []*tx.Transaction{},
	}

	for i := 0; i >= 0; i++ {
		if _, ok := js.Get("result").Get("Transactions").GetIndex(i).CheckGet("TxType"); !ok {
			break
		}

		txn, _, _ := parseTxMsg(js.Get("result").Get("Transactions").GetIndex(i))
		b.Transactions = append(b.Transactions, txn)
	}

	return b
}

func parseTxMsg(js *Json) (*tx.Transaction, uint32, uint32) {
	var txn tx.Transaction
	txType, _ := js.Get("TxType").Int()
	txn.TxType = tx.TransactionType(txType)

	payloadVer, _ := js.Get("PayloadVersion").Int()
	txn.PayloadVersion = byte(payloadVer)

	payload, _ := js.Get("Payload").String()
	if payload == "" {
		txn.Payload = nil
	} else {
		//TODO transfer  payload acording txType
	}

	for i := 0; i >= 0; i++ {
		if _, ok := js.Get("Attributes").GetIndex(i).CheckGet("Usage"); !ok {
			break
		}

		usage, _ := js.Get("Attributes").GetIndex(i).Get("Usage").Int()
		dataStr, _ := js.Get("Attributes").GetIndex(i).Get("Data").String()
		data, _ := hex.DecodeString(dataStr)
		attr := &tx.TxAttribute{
			Usage: tx.TransactionAttributeUsage(usage),
			Data:  data,
			Size:  0,
		}
		txn.Attributes = append(txn.Attributes, attr)
	}

	for i := 0; i >= 0; i++ {
		if _, ok := js.Get("UTXOInputs").GetIndex(i).CheckGet("ReferTxID"); !ok {
			break
		}

		refID, _ := js.Get("UTXOInputs").GetIndex(i).Get("ReferTxID").String()
		refIdx, _ := js.Get("UTXOInputs").GetIndex(i).Get("ReferTxOutputIndex").Int()
		sequence, _ := js.Get("UTXOInputs").GetIndex(i).Get("Sequence").Int()

		var refTxID Uint256
		refIDSlice, _ := hex.DecodeString(refID)
		refIDSlice = BytesReverse(refIDSlice)
		copy(refTxID[:], refIDSlice[0:32])

		input := &tx.UTXOTxInput{
			ReferTxID:          refTxID,
			ReferTxOutputIndex: uint16(refIdx),
			Sequence:           uint32(sequence),
		}

		txn.UTXOInputs = append(txn.UTXOInputs, input)
	}

	//TODO Balance

	for i := 0; i >= 0; i++ {
		if _, ok := js.Get("Outputs").GetIndex(i).CheckGet("AssetID"); !ok {
			break
		}

		assetIDStr, _ := js.Get("Outputs").GetIndex(i).Get("AssetID").String()
		valueStr, _ := js.Get("Outputs").GetIndex(i).Get("Value").String()
		address, _ := js.Get("Outputs").GetIndex(i).Get("Address").String()

		var assetID Uint256
		assetIDSlice, _ := hex.DecodeString(assetIDStr)
		assetIDSlice = BytesReverse(assetIDSlice)
		copy(assetID[:], assetIDSlice[0:32])

		values := strings.Split(valueStr, ".")
		num1, _ := strconv.ParseInt(values[0], 10, 64)
		num2, _ := strconv.ParseInt(values[1], 10, 64)
		value := num1*100000000 + num2

		programHash, _ := ToScriptHash(address)

		output := &tx.TxOutput{
			AssetID:     assetID,
			Value:       Fixed64(value),
			ProgramHash: programHash,
		}

		txn.Outputs = append(txn.Outputs, output)
	}

	lockTime, _ := js.Get("LockTime").Int()
	txn.LockTime = uint32(lockTime)

	timeStamp, _ := js.Get("Timestamp").Int()
	confirm, _ := js.Get("Confirminations").Int()

	return &txn, uint32(timeStamp), uint32(confirm)
}

func getLatestBits(L *lua.LState) int {
	resp, _ := httpjsonrpc.Call(Address(), "getblockcount", 0, []interface{}{})
	js, _ := NewJson(resp)
	height, _ := js.Get("result").Int()
	currentHeight := height - 1
	resp, _ = httpjsonrpc.Call(Address(), "getblock", 0, []interface{}{currentHeight})

	//FormatOutput(resp)
	js, _ = NewJson(resp)
	res, _ := js.Get("result").Get("BlockData").Get("Bits").Int()

	L.Push(lua.LNumber(res))
	return 1
}
