package errors

import (
	"fmt"
	"errors"
)

type ErrCode int

const (
	ErrNoCode               ErrCode = -2
	ErrNoError              ErrCode = 0
	ErrUnknown              ErrCode = -1
	ErrDuplicatedTx         ErrCode = 1
	ErrInvalidInput         ErrCode = 45003
	ErrInvalidOutput        ErrCode = 45004
	ErrAssetPrecision       ErrCode = 45005
	ErrTransactionBalance   ErrCode = 45006
	ErrAttributeProgram     ErrCode = 45007
	ErrTransactionContracts ErrCode = 45008
	ErrTransactionPayload   ErrCode = 45009
	ErrDoubleSpend          ErrCode = 45010
	ErrTxHashDuplicate      ErrCode = 45011
	ErrStateUpdaterVaild    ErrCode = 45012
	ErrSummaryAsset         ErrCode = 45013
	ErrXmitFail             ErrCode = 45014
	ErrTransactionSize      ErrCode = 45015
	ErrUnknownReferedTxn    ErrCode = 45016
	ErrInvalidReferedTxn    ErrCode = 45017
	ErrIneffectiveCoinbase  ErrCode = 45018
	ErrUTXOLocked           ErrCode = 45019
)

func (err ErrCode) Error() string {
	switch err {
	case ErrNoCode:
		return "no error code"
	case ErrNoError:
		return "not an error"
	case ErrUnknown:
		return "unknown error"
	case ErrDuplicatedTx:
		return "duplicated transaction detected"
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
	case ErrStateUpdaterVaild:
		return "invalid state updater"
	case ErrSummaryAsset:
		return "invalid summary asset"
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

func NewDetailErr(err error, errcode ErrCode, errmsg string) error {
	return errors.New(errmsg)
}
