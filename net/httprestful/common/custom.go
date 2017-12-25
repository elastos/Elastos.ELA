package common

import (
	. "Elastos.ELA/common"
	tx "Elastos.ELA/core/transaction"
	. "Elastos.ELA/errors"
	. "Elastos.ELA/net/httpjsonrpc"
	"bytes"
	"encoding/json"
	"time"
)

const AttributeMaxLen = 252

//record
func getRecordData(cmd map[string]interface{}) ([]byte, ErrCode) {
	if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
		str, ok := cmd["RecordData"].(string)
		if !ok {
			return nil, InvalidParams
		}
		bys, err := HexStringToBytes(str)
		if err != nil {
			return nil, InvalidParams
		}
		return bys, Success
	}
	type Data struct {
		Algrithem string `json:Algrithem`
		Hash      string `json:Hash`
		Signature string `json:Signature`
		Text      string `json:Text`
	}
	type RecordData struct {
		CAkey     string  `json:CAkey`
		Data      Data    `json:Data`
		SeqNo     string  `json:SeqNo`
		Timestamp float64 `json:Timestamp`
	}

	tmp := &RecordData{}
	reqRecordData, ok := cmd["RecordData"].(map[string]interface{})
	if !ok {
		return nil, InvalidParams
	}
	reqBtys, err := json.Marshal(reqRecordData)
	if err != nil {
		return nil, InvalidParams
	}

	if err := json.Unmarshal(reqBtys, tmp); err != nil {
		return nil, InvalidParams
	}
	tmp.CAkey, ok = cmd["CAkey"].(string)
	if !ok {
		return nil, InvalidParams
	}
	repBtys, err := json.Marshal(tmp)
	if err != nil {
		return nil, InvalidParams
	}
	return repBtys, Success
}
func getInnerTimestamp() ([]byte, ErrCode) {
	type InnerTimestamp struct {
		InnerTimestamp float64 `json:InnerTimestamp`
	}
	tmp := &InnerTimestamp{InnerTimestamp: float64(time.Now().Unix())}
	repBtys, err := json.Marshal(tmp)
	if err != nil {
		return nil, InvalidParams
	}
	return repBtys, Success
}
func SendRecord(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)
	var recordData []byte
	var innerTime []byte
	innerTime, resp["Error"] = getInnerTimestamp()
	if innerTime == nil {
		return resp
	}
	recordData, resp["Error"] = getRecordData(cmd)
	if recordData == nil {
		return resp
	}

	var inputs []*tx.UTXOTxInput
	var outputs []*tx.TxOutput

	transferTx, _ := tx.NewTransferAssetTransaction(inputs, outputs)

	rcdInner := tx.NewTxAttribute(tx.Description, innerTime)
	transferTx.Attributes = append(transferTx.Attributes, &rcdInner)

	bytesBuf := bytes.NewBuffer(recordData)

	buf := make([]byte, AttributeMaxLen)
	for {
		n, err := bytesBuf.Read(buf)
		if err != nil {
			break
		}
		var data = make([]byte, n)
		copy(data, buf[0:n])
		record := tx.NewTxAttribute(tx.Description, data)
		transferTx.Attributes = append(transferTx.Attributes, &record)
	}
	if errCode := VerifyAndSendTx(transferTx); errCode != Success {
		resp["Error"] = int64(errCode)
		return resp
	}
	hash := transferTx.Hash()
	resp["Result"] = BytesToHexString(hash.ToArrayReverse())
	return resp
}

func SendRecordTransaction(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)
	var recordData []byte
	recordData, resp["Error"] = getRecordData(cmd)
	if recordData == nil {
		return resp
	}
	recordType := "record"
	recordTx, _ := tx.NewRecordTransaction(recordType, recordData)

	hash := recordTx.Hash()
	resp["Result"] = BytesToHexString(hash.ToArrayReverse())
	if errCode := VerifyAndSendTx(recordTx); errCode != Success {
		resp["Error"] = int64(errCode)
		return resp
	}
	return resp
}
