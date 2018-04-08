package ledger

import (
	"errors"

	core_ledger "github.com/elastos/Elastos.ELA.Core/core/ledger"
	tx "github.com/elastos/Elastos.ELA.Core/core/transaction"
	"github.com/elastos/Elastos.ELA/core/transaction/payload"
)

type MainTransactionValidatorImpl struct {
	core_ledger.TransactionValidatorImpl
}

func (txValiator *MainTransactionValidatorImpl) CheckTransactionSignature(txn *tx.NodeTransaction) error {
	flag, err := VerifySignature(txn)
	if flag && err == nil {
		return nil
	} else {
		return err
	}
}

func (txValiator *MainTransactionValidatorImpl) CheckTransactionPayload(Tx *tx.NodeTransaction) error {
	if err := txValiator.TransactionValidatorImpl.CheckTransactionPayload(Tx); err == nil {
		return nil
	}

	switch Tx.Payload.(type) {
	case *payload.WithdrawToken:
	case *payload.TransferCrossChainAsset:
	default:
		return errors.New("[txValidator],invalidate transaction payload type.")
	}
	return nil
}

func init() {
	core_ledger.Validator = &MainTransactionValidatorImpl{
		core_ledger.TransactionValidatorImpl{}}
}
