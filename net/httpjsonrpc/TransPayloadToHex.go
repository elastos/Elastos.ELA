package httpjsonrpc

import (
	. "ELA/common"
	"ELA/core/asset"
	. "ELA/core/transaction"
	"ELA/core/transaction/payload"
)

type PayloadInfo interface{}

type CoinbaseInfo struct {
	CoinbaseData string
}

type RegisterAssetInfo struct {
	Asset      *asset.Asset
	Amount     string
	Controller string
}

type RecordInfo struct {
	RecordType string
	RecordData string
}

type FunctionCodeInfo struct {
	Code           string
	ParameterTypes string
	ReturnTypes    string
}

type DeployCodeInfo struct {
	Code        *FunctionCodeInfo
	Name        string
	CodeVersion string
	Author      string
	Email       string
	Description string
}

func TransPayloadToHex(p Payload) PayloadInfo {
	switch object := p.(type) {
	case *payload.CoinBase:
		obj := new(CoinbaseInfo)
		obj.CoinbaseData = string(object.CoinbaseData)
		return obj
	case *payload.RegisterAsset:
		obj := new(RegisterAssetInfo)
		obj.Asset = object.Asset
		obj.Amount = object.Amount.String()
		obj.Controller = BytesToHexString(object.Controller.ToArrayReverse())
		return obj
	case *payload.TransferAsset:
	case *payload.Record:
	case *payload.DeployCode:
	}
	return nil
}
