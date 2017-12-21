package common

import (
    "bytes"
    "fmt"
    "strconv"

    . "ELA/common"
    "ELA/core/ledger"
    tx "ELA/core/transaction"
    "ELA/core/transaction/payload"
    . "ELA/errors"
    . "ELA/net/httpjsonrpc"
    Err "ELA/net/httprestful/error"
    . "ELA/net/protocol"
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
    resp := ResponsePack(Err.SUCCESS)
    if node != nil {
        resp["Result"] = node.GetConnectionCnt()
    }

    return resp
}

//Block
func GetBlockHeight(cmd map[string]interface{}) map[string]interface{} {
    resp := ResponsePack(Err.SUCCESS)
    resp["Result"] = ledger.DefaultLedger.Blockchain.BlockHeight
    return resp
}
func GetBlockHash(cmd map[string]interface{}) map[string]interface{} {
    resp := ResponsePack(Err.SUCCESS)
    param := cmd["Height"].(string)
    if len(param) == 0 {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    height, err := strconv.ParseInt(param, 10, 64)
    if err != nil {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    hash, err := ledger.DefaultLedger.Store.GetBlockHash(uint32(height))
    if err != nil {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    resp["Result"] = BytesToHexString(hash.ToArrayReverse())
    return resp
}

func GetTransactionPool(cmd map[string]interface{}) map[string]interface{} {
    resp := ResponsePack(Err.SUCCESS)

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
	resp := ResponsePack(Err.SUCCESS)
	assetid, ok := cmd["Assetid"].(string)
	if !ok {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	var assetHash Uint256

	bys, err := HexStringToBytesReverse(assetid)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	if err := assetHash.Deserialize(bytes.NewReader(bys)); err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	amount, err := ledger.DefaultLedger.Store.GetQuantityIssued(assetHash)
	if err != nil {
		resp["Error"] = Err.INVALID_PARAMS
		return resp
	}
	resp["Result"] = amount.String()
	return resp
}
*/
func GetBlockInfo(block *ledger.Block) BlockInfo {
    hash := block.Hash()
    auxInfo := &AuxInfo{
        Version:    block.Blockdata.AuxPow.ParBlockHeader.Version,
        PrevBlock:  BytesToHexString(new(Uint256).ToArrayReverse()),
        MerkleRoot: BytesToHexString(block.Blockdata.AuxPow.ParBlockHeader.MerkleRoot.ToArrayReverse()),
        Timestamp:  block.Blockdata.AuxPow.ParBlockHeader.Timestamp,
        Bits:       0,
        Nonce:      block.Blockdata.AuxPow.ParBlockHeader.Nonce,
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
func getBlock(hash Uint256, getTxBytes bool) (interface{}, int64) {
    block, err := ledger.DefaultLedger.Store.GetBlock(hash)
    if err != nil {
        return "", Err.UNKNOWN_BLOCK
    }
    if getTxBytes {
        w := bytes.NewBuffer(nil)
        block.Serialize(w)
        return BytesToHexString(w.Bytes()), Err.SUCCESS
    }
    return GetBlockInfo(block), Err.SUCCESS
}
func GetBlockByHash(cmd map[string]interface{}) map[string]interface{} {
    resp := ResponsePack(Err.SUCCESS)
    param := cmd["Hash"].(string)
    if len(param) == 0 {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    var getTxBytes bool = false
    if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
        getTxBytes = true
    }
    var hash Uint256
    hex, err := HexStringToBytesReverse(param)
    if err != nil {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    if err := hash.Deserialize(bytes.NewReader(hex)); err != nil {
        resp["Error"] = Err.INVALID_TRANSACTION
        return resp
    }

    resp["Result"], resp["Error"] = getBlock(hash, getTxBytes)

    return resp
}
func GetBlockTxsByHeight(cmd map[string]interface{}) map[string]interface{} {
    resp := ResponsePack(Err.SUCCESS)

    param := cmd["Height"].(string)
    if len(param) == 0 {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    height, err := strconv.ParseInt(param, 10, 64)
    if err != nil {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    index := uint32(height)
    hash, err := ledger.DefaultLedger.Store.GetBlockHash(index)
    if err != nil {
        resp["Error"] = Err.UNKNOWN_BLOCK
        return resp
    }
    block, err := ledger.DefaultLedger.Store.GetBlock(hash)
    if err != nil {
        resp["Error"] = Err.UNKNOWN_BLOCK
        return resp
    }
    resp["Result"] = GetBlockTransactions(block)
    return resp
}
func GetBlockByHeight(cmd map[string]interface{}) map[string]interface{} {
    resp := ResponsePack(Err.SUCCESS)

    param := cmd["Height"].(string)
    if len(param) == 0 {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    var getTxBytes bool = false
    if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
        getTxBytes = true
    }
    height, err := strconv.ParseInt(param, 10, 64)
    if err != nil {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    index := uint32(height)
    hash, err := ledger.DefaultLedger.Store.GetBlockHash(index)
    if err != nil {
        resp["Error"] = Err.UNKNOWN_BLOCK
        return resp
    }
    resp["Result"], resp["Error"] = getBlock(hash, getTxBytes)
    return resp
}

//Asset
func GetAssetByHash(cmd map[string]interface{}) map[string]interface{} {
    resp := ResponsePack(Err.SUCCESS)

    str := cmd["Hash"].(string)
    hex, err := HexStringToBytesReverse(str)
    if err != nil {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    var hash Uint256
    err = hash.Deserialize(bytes.NewReader(hex))
    if err != nil {
        resp["Error"] = Err.INVALID_ASSET
        return resp
    }
    asset, err := ledger.DefaultLedger.Store.GetAsset(hash)
    if err != nil {
        resp["Error"] = Err.UNKNOWN_ASSET
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
    resp := ResponsePack(Err.SUCCESS)
    addr, ok := cmd["Addr"].(string)
    if !ok {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    var programHash Uint160
    programHash, err := ToScriptHash(addr)
    if err != nil {
        resp["Error"] = Err.INVALID_PARAMS
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
    resp := ResponsePack(Err.SUCCESS)
    addr, ok := cmd["Addr"].(string)
    assetid, k := cmd["Assetid"].(string)
    if !ok || !k {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    var programHash Uint160
    programHash, err := ToScriptHash(addr)
    if err != nil {
        resp["Error"] = Err.INVALID_PARAMS
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
    resp := ResponsePack(Err.SUCCESS)
    addr, ok := cmd["Addr"].(string)
    if !ok {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    var programHash Uint160

    programHash, err := ToScriptHash(addr)
    if err != nil {
        resp["Error"] = Err.INVALID_PARAMS
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
            resp["Error"] = Err.INTERNAL_ERROR
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
    resp := ResponsePack(Err.SUCCESS)
    addr, ok := cmd["Addr"].(string)
    assetid, k := cmd["Assetid"].(string)
    if !ok || !k {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }

    var programHash Uint160
    var assetHash Uint256
    programHash, err := ToScriptHash(addr)
    if err != nil {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    bys, err := HexStringToBytesReverse(assetid)
    if err != nil {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    if err := assetHash.Deserialize(bytes.NewReader(bys)); err != nil {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    type UTXOUnspentInfo struct {
        Txid  string
        Index uint32
        Value string
    }
    infos, err := ledger.DefaultLedger.Store.GetUnspentFromProgramHash(programHash, assetHash)
    if err != nil {
        resp["Error"] = Err.INVALID_PARAMS
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
    resp := ResponsePack(Err.SUCCESS)

    str := cmd["Hash"].(string)
    bys, err := HexStringToBytesReverse(str)
    if err != nil {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    var hash Uint256
    err = hash.Deserialize(bytes.NewReader(bys))
    if err != nil {
        resp["Error"] = Err.INVALID_TRANSACTION
        return resp
    }
    txn, height, err := ledger.DefaultLedger.Store.GetTransaction(hash)
    if err != nil {
        resp["Error"] = Err.UNKNOWN_TRANSACTION
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
        resp["Error"] = Err.UNKNOWN_BLOCK
        return resp
    }
    header, err := ledger.DefaultLedger.Store.GetHeader(bHash)
    if err != nil {
        resp["Error"] = Err.UNKNOWN_BLOCK
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
    resp := ResponsePack(Err.SUCCESS)

    str, ok := cmd["Data"].(string)
    if !ok {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    bys, err := HexStringToBytes(str)
    if err != nil {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    var txn tx.Transaction
    if err := txn.Deserialize(bytes.NewReader(bys)); err != nil {
        resp["Error"] = Err.INVALID_TRANSACTION
        return resp
    }
    var hash Uint256
    hash = txn.Hash()
    if errCode := VerifyAndSendTx(&txn); errCode != ErrNoError {
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
    resp := ResponsePack(Err.SUCCESS)
    namespace, ok := cmd["Namespace"].(string)
    if !ok {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    key, ok := cmd["Key"].(string)
    if !ok {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    fmt.Println(cmd, namespace, key)
    //TODO get state from store
    return resp
}

func ResponsePack(errCode int64) map[string]interface{} {
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
    resp := ResponsePack(Err.SUCCESS)
    str := cmd["Hash"].(string)
    bys, err := HexStringToBytesReverse(str)
    if err != nil {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    var hash Uint160
    err = hash.Deserialize(bytes.NewReader(bys))
    if err != nil {
        resp["Error"] = Err.INVALID_PARAMS
        return resp
    }
    //TODO GetContract from store
    //contract, err := ledger.DefaultLedger.Store.GetContract(hash)
    //if err != nil {
    //	resp["Error"] = Err.INVALID_PARAMS
    //	return resp
    //}
    //resp["Result"] = string(contract)
    return resp
}
