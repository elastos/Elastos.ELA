package errors

import (
	"fmt"
	"errors"
	"runtime/debug"
)

type ErrCode int

const (
	ErrNoCode               ErrCode = -1
	Success                 ErrCode = 0
	ErrInvalidInput         ErrCode = 45003
	ErrInvalidOutput        ErrCode = 45004
	ErrAssetPrecision       ErrCode = 45005
	ErrTransactionBalance   ErrCode = 45006
	ErrAttributeProgram     ErrCode = 45007
	ErrTransactionContracts ErrCode = 45008
	ErrTransactionPayload   ErrCode = 45009
	ErrDoubleSpend          ErrCode = 45010
	ErrTxHashDuplicate      ErrCode = 45011
	ErrXmitFail             ErrCode = 45014
	ErrTransactionSize      ErrCode = 45015
	ErrUnknownReferedTxn    ErrCode = 45016
	ErrInvalidReferedTxn    ErrCode = 45017
	ErrIneffectiveCoinbase  ErrCode = 45018
	ErrUTXOLocked           ErrCode = 45019

	//what the hell
	SessionExpired     		ErrCode = 41001
	IllegalDataFormat  		ErrCode = 41003
	OauthTimeout       		ErrCode = 41004
	InvalidMethod      		ErrCode = 42001
	InvalidParams      		ErrCode = 42002
	InvalidToken       		ErrCode = 42003
	InvalidTransaction 		ErrCode = 43001
	InvalidAsset       		ErrCode = 43002
	UnknownTransaction 		ErrCode = 44001
	UnknownAsset       		ErrCode = 44002
	UnknownBlock       		ErrCode = 44003
	InternalError      		ErrCode = 45002
)

var ErrMap = map[ErrCode]string{
	Success:                 "Success",
	SessionExpired:          "Session expired",
	IllegalDataFormat:       "Illegal Dataformat",
	OauthTimeout:            "Connect to oauth timeout",
	InvalidMethod:           "Invalid method",
	InvalidParams:           "Invalid Params",
	InvalidToken:            "Verify token error",
	InvalidTransaction:      "Invalid transaction",
	InvalidAsset:            "Invalid asset",
	UnknownTransaction:      "Unknown Transaction",
	UnknownAsset:            "Unknown asset",
	UnknownBlock:            "Unknown Block",
	InternalError:           "Internal error",
	ErrInvalidInput:         "INTERNAL ERROR, ErrInvalidInput",
	ErrInvalidOutput:        "INTERNAL ERROR, ErrInvalidOutput",
	ErrAssetPrecision:       "INTERNAL ERROR, ErrAssetPrecision",
	ErrTransactionBalance:   "INTERNAL ERROR, ErrTransactionBalance",
	ErrAttributeProgram:     "INTERNAL ERROR, ErrAttributeProgram",
	ErrTransactionContracts: "INTERNAL ERROR, ErrTransactionContracts",
	ErrTransactionPayload:   "INTERNAL ERROR, ErrTransactionPayload",
	ErrDoubleSpend:          "INTERNAL ERROR, ErrDoubleSpend",
	ErrTxHashDuplicate:      "INTERNAL ERROR, ErrTxHashDuplicate",
	ErrXmitFail:             "INTERNAL ERROR, ErrXmitFail",
	ErrTransactionSize:      "INTERNAL ERROR, ErrTransactionSize",
	ErrUnknownReferedTxn:    "INTERNAL ERROR, ErrUnknownReferedTxn",
	ErrInvalidReferedTxn:    "INTERNAL ERROR, ErrInvalidReferedTxn",
	ErrIneffectiveCoinbase:  "INTERNAL ERROR, ErrIneffectiveCoinbase",
}

func (err ErrCode) Error() string {
	switch err {
	case ErrNoCode:
		return "no error code"
	case Success:
		return "not an error"
	case ErrInvalidInput:
		return "invalid transaction input detected"
	case ErrInvalidOutput:
		return "invalid transaction output detected"
	case ErrAssetPrecision:
		return "invalid asset precision"
	case ErrTransactionBalance:
		return "transaction balance unmatched"
	case ErrAttributeProgram:
		return "attribute program error"
	case ErrTransactionContracts:
		return "invalid transaction contract"
	case ErrTransactionPayload:
		return "invalid transaction payload"
	case ErrDoubleSpend:
		return "double spent transaction detected"
	case ErrTxHashDuplicate:
		return "duplicated transaction hash detected"
	case ErrXmitFail:
		return "transmit error"
	case ErrTransactionSize:
		return "invalid transaction size"
	case ErrUnknownReferedTxn:
		return "unknown referenced transaction"
	case ErrInvalidReferedTxn:
		return "invalid referenced transaction"
	case ErrIneffectiveCoinbase:
		return "ineffective coinbase"
	case ErrUTXOLocked:
		return "unspend utxo locked"
	}

	return fmt.Sprintf("Unknown error? Error code = %d", err)
}

func NewDetailErr(err error, errCode ErrCode, errmsg string) error {
	if errCode != 0 {
		//TODO: 这里的错误机制需要重写，但我还没想好怎么改
		//现在我只是把多余的代码删掉了，但它的设计也很糟糕，所以要重新设计
		//目前初步设想是要有一个错误码
		//但是错误码在对外的API肯定有用，内部报错不确定要不要用，这个要看具体业务，所以我得先把代码看完。
		//堆栈是肯定要打的，为了排错，所以我先把堆栈加上。
	}
	fmt.Println(string(debug.Stack()))

	if err.Error() == "" {
		return errors.New(errmsg)
	} else {
		return err
	}
}
