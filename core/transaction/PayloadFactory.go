package transaction

import (
	"errors"

	. "github.com/elastos/Elastos.ELA.Utility/core/transaction"
	"github.com/elastos/Elastos.ELA/core/transaction/payload"
)

const (
	SideMining              TransactionType = 0x05
	WithdrawToken           TransactionType = 0x07
	TransferCrossChainAsset TransactionType = 0x08
)

type PayloadFactoryMainNodeImpl struct {
	innerFactory *PayloadFactoryImpl
}

func (factor *PayloadFactoryMainNodeImpl) Name(txType TransactionType) string {
	if name := factor.innerFactory.Name(txType); name != "Unknown" {
		return name
	}

	switch txType {
	case SideMining:
		return "SideMining"
	case WithdrawToken:
		return "WithdrawToken"
	case TransferCrossChainAsset:
		return "TransferCrossChainAsset"
	default:
		return "Unknown"
	}
}

func (factor *PayloadFactoryMainNodeImpl) Create(txType TransactionType) (Payload, error) {
	if p, _ := factor.innerFactory.Create(txType); p != nil {
		return p, nil
	}

	switch txType {
	case SideMining:
		return new(payload.SideMining), nil
	case WithdrawToken:
		return new(payload.WithdrawToken), nil
	case TransferCrossChainAsset:
		return new(payload.TransferCrossChainAsset), nil
	default:
		return nil, errors.New("[NodeTransaction], invalid transaction type.")
	}
}

func init() {
	PayloadFactorySingleton = &PayloadFactoryMainNodeImpl{innerFactory: &PayloadFactoryImpl{}}
}
