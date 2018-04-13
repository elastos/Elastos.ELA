package ledger

import (
	"errors"
	"fmt"
	"math"

	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/core/asset"
	uti_tx "github.com/elastos/Elastos.ELA.Utility/core/transaction"
	uti_payload "github.com/elastos/Elastos.ELA.Utility/core/transaction/payload"
	. "github.com/elastos/Elastos.ELA.Utility/errors"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/common/log"
	tx "github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/transaction/payload"
)

var Validator TransactionValidator

type TransactionValidator interface {
	CheckTransactionSanity(txn *tx.NodeTransaction) ErrCode
	CheckTransactionContext(txn *tx.NodeTransaction, ledger *Ledger) ErrCode
	CheckTransactionInput(txn *tx.NodeTransaction) error
	CheckTransactionOutput(txn *tx.NodeTransaction) error
	CheckTransactionUTXOLock(txn *tx.NodeTransaction) error
	CheckTransactionSize(txn *tx.NodeTransaction) error
	IsDoubleSpend(tx *tx.NodeTransaction, ledger *Ledger) bool
	CheckAssetPrecision(Tx *tx.NodeTransaction) error
	CheckTransactionBalance(Tx *tx.NodeTransaction) error
	CheckAttributeProgram(txn *tx.NodeTransaction) error
	CheckTransactionSignature(txn *tx.NodeTransaction) error
	CheckTransactionPayload(Tx *tx.NodeTransaction) error
}

type TransactionValidatorImpl struct {
}

// CheckTransactionSanity verifys received single transaction
func (txValiator *TransactionValidatorImpl) CheckTransactionSanity(txn *tx.NodeTransaction) ErrCode {

	if err := txValiator.CheckTransactionSize(txn); err != nil {
		log.Warn("[CheckTransactionSize],", err)
		return ErrTransactionSize
	}

	if err := txValiator.CheckTransactionInput(txn); err != nil {
		log.Warn("[CheckTransactionInput],", err)
		return ErrInvalidInput
	}

	if err := txValiator.CheckTransactionOutput(txn); err != nil {
		log.Warn("[CheckTransactionOutput],", err)
		return ErrInvalidOutput
	}

	if err := txValiator.CheckAssetPrecision(txn); err != nil {
		log.Warn("[CheckAssetPrecesion],", err)
		return ErrAssetPrecision
	}

	if err := txValiator.CheckAttributeProgram(txn); err != nil {
		log.Warn("[CheckTransactionAttribute],", err)
		return ErrAttributeProgram
	}

	if err := txValiator.CheckTransactionPayload(txn); err != nil {
		log.Warn("[CheckTransactionPayload],", err)
		return ErrTransactionPayload
	}

	// check iterms above for Coinbase transaction
	if txn.IsCoinBaseTx() {
		return Success
	}

	return Success
}

// CheckTransactionContext verifys a transaction with history transaction in ledger
func (txValiator *TransactionValidatorImpl) CheckTransactionContext(txn *tx.NodeTransaction, ledger *Ledger) ErrCode {
	// check if duplicated with transaction in ledger
	if exist := ledger.Store.IsTxHashDuplicate(txn.Hash()); exist {
		log.Info("[CheckTransactionContext] duplicate transaction check faild.")
		return ErrTxHashDuplicate
	}

	if txn.IsCoinBaseTx() {
		return Success
	}

	// check double spent transaction
	if txValiator.IsDoubleSpend(txn, ledger) {
		log.Info("[CheckTransactionContext] IsDoubleSpend check faild.")
		return ErrDoubleSpend
	}

	if err := txValiator.CheckTransactionUTXOLock(txn); err != nil {
		log.Warn("[CheckTransactionUTXOLock],", err)
		return ErrUTXOLocked
	}

	if err := txValiator.CheckTransactionBalance(txn); err != nil {
		log.Warn("[CheckTransactionBalance],", err)
		return ErrTransactionBalance
	}

	if err := txValiator.CheckTransactionSignature(txn); err != nil {
		log.Warn("[CheckTransactionSignature],", err)
		return ErrTransactionSignature
	}
	// check referenced Output value
	for _, input := range txn.UTXOInputs {
		referHash := input.ReferTxID
		referTxnOutIndex := input.ReferTxOutputIndex
		referTxn, _, err := ledger.Store.GetTransaction(referHash)
		if err != nil {
			log.Warn("Referenced transaction can not be found", common.BytesToHexString(referHash.ToArray()))
			return ErrUnknownReferedTxn
		}
		referTxnOut := referTxn.Outputs[referTxnOutIndex]
		if referTxnOut.Value <= 0 {
			log.Warn("Value of referenced transaction output is invalid")
			return ErrInvalidReferedTxn
		}
		// coinbase transaction only can be spent after got SpendCoinbaseSpan times confirmations
		if referTxn.IsCoinBaseTx() {
			lockHeight := referTxn.LockTime
			currentHeight := ledger.Store.GetHeight()
			if currentHeight-lockHeight < config.Parameters.ChainParam.SpendCoinbaseSpan {
				return ErrIneffectiveCoinbase
			}
		}
	}

	return Success
}

//validate the transaction of duplicate UTXO input
func (txValiator *TransactionValidatorImpl) CheckTransactionInput(txn *tx.NodeTransaction) error {
	var zeroHash common.Uint256
	if txn.IsCoinBaseTx() {
		if len(txn.UTXOInputs) != 1 {
			return errors.New("coinbase must has only one input")
		}
		coinbaseInputHash := txn.UTXOInputs[0].ReferTxID
		coinbaseInputIndex := txn.UTXOInputs[0].ReferTxOutputIndex
		//TODO :check sequence
		if coinbaseInputHash.CompareTo(zeroHash) != 0 || coinbaseInputIndex != math.MaxUint16 {
			return errors.New("invalid coinbase input")
		}

		return nil
	}

	if len(txn.UTXOInputs) <= 0 {
		return errors.New("transaction has no inputs")
	}
	for i, utxoin := range txn.UTXOInputs {
		referTxnHash := utxoin.ReferTxID
		referTxnOutIndex := utxoin.ReferTxOutputIndex
		if (referTxnHash.CompareTo(zeroHash) == 0) && (referTxnOutIndex == math.MaxUint16) {
			return errors.New("invalid transaction input")
		}
		for j := 0; j < i; j++ {
			if referTxnHash == txn.UTXOInputs[j].ReferTxID && referTxnOutIndex == txn.UTXOInputs[j].ReferTxOutputIndex {
				return errors.New("duplicated transaction inputs")
			}
		}
	}

	return nil
}

func (txValiator *TransactionValidatorImpl) CheckTransactionOutput(txn *tx.NodeTransaction) error {
	if txn.IsCoinBaseTx() {
		if len(txn.Outputs) < 2 {
			return errors.New("coinbase output is not enough, at least 2")
		}
		found := false
		for _, output := range txn.Outputs {
			if output.AssetID != DefaultLedger.Blockchain.AssetID {
				return errors.New("asset ID in coinbase is invalid")
			}
			address, err := output.ProgramHash.ToAddress()
			if err != nil {
				return err
			}
			if address == FoundationAddress {
				found = true
			}
		}
		if !found {
			return errors.New("no foundation address in coinbase output")
		}

		return nil
	}

	if len(txn.Outputs) < 1 {
		return errors.New("transaction has no outputs")
	}

	// check if output address is valid
	for _, output := range txn.Outputs {
		if output.AssetID != DefaultLedger.Blockchain.AssetID {
			return errors.New("asset ID in coinbase is invalid")
		}

		if !output.ProgramHash.Valid() {
			return errors.New("output address is invalid")
		}
	}

	return nil
}

func (txValiator *TransactionValidatorImpl) CheckTransactionUTXOLock(txn *tx.NodeTransaction) error {
	if txn.IsCoinBaseTx() {
		return nil
	}
	if len(txn.UTXOInputs) <= 0 {
		return errors.New("NodeTransaction has no inputs")
	}
	referenceWithUTXO_Output, err := txn.GetReference()
	if err != nil {
		return errors.New(fmt.Sprintf("GetReference failed: %x", txn.Hash()))
	}
	for input, output := range referenceWithUTXO_Output {

		if output.OutputLock == 0 {
			//check next utxo
			continue
		}
		if input.Sequence != math.MaxUint32-1 {
			return errors.New("Invalid input sequence")
		}
		if txn.LockTime < output.OutputLock {
			return errors.New("UTXO output locked")
		}
	}
	return nil
}

func (txValiator *TransactionValidatorImpl) CheckTransactionSize(txn *tx.NodeTransaction) error {
	size := txn.GetSize()
	if size <= 0 || size > MaxBlockSize {
		return errors.New(fmt.Sprintf("Invalid transaction size: %d bytes", size))
	}

	return nil
}

func (txValiator *TransactionValidatorImpl) IsDoubleSpend(tx *tx.NodeTransaction, ledger *Ledger) bool {
	return ledger.IsDoubleSpend(tx)
}

func (txValiator *TransactionValidatorImpl) CheckAssetPrecision(Tx *tx.NodeTransaction) error {
	if len(Tx.Outputs) == 0 {
		return nil
	}
	assetOutputs := make(map[common.Uint256][]*uti_tx.TxOutput, len(Tx.Outputs))

	for _, v := range Tx.Outputs {
		assetOutputs[v.AssetID] = append(assetOutputs[v.AssetID], v)
	}
	for k, outputs := range assetOutputs {
		asset, err := DefaultLedger.GetAsset(k)
		if err != nil {
			return errors.New("The asset not exist in local blockchain.")
		}
		precision := asset.Precision
		for _, output := range outputs {
			if checkAmountPrecise(output.Value, precision) {
				return errors.New("The precision of asset is incorrect.")
			}
		}
	}
	return nil
}

func (txValiator *TransactionValidatorImpl) CheckTransactionBalance(Tx *tx.NodeTransaction) error {
	// TODO: check coinbase balance 30%-70%
	for _, v := range Tx.Outputs {
		if v.Value <= common.Fixed64(0) {
			return errors.New("Invalide transaction UTXO output.")
		}
	}
	results, err := Tx.GetTransactionResults()
	if err != nil {
		return err
	}
	for k, v := range results {

		if v < common.Fixed64(config.Parameters.PowConfiguration.MinTxFee) {
			log.Debug(fmt.Sprintf("AssetID %x in Transfer transactions %x , input < output .\n", k, Tx.Hash()))
			return errors.New(fmt.Sprintf("AssetID %x in Transfer transactions %x , input < output .\n", k, Tx.Hash()))
		}
	}
	return nil
}

func (txValiator *TransactionValidatorImpl) CheckAttributeProgram(txn *tx.NodeTransaction) error {
	//TODO: implement CheckAttributeProgram
	return nil
}

func (txValiator *TransactionValidatorImpl) CheckTransactionSignature(txn *tx.NodeTransaction) error {
	flag, err := VerifySignature(txn)
	if flag && err == nil {
		return nil
	} else {
		return err
	}
}

func checkAmountPrecise(amount common.Fixed64, precision byte) bool {
	return amount.GetData()%int64(math.Pow(10, 8-float64(precision))) != 0
}

func (txValiator *TransactionValidatorImpl) CheckTransactionPayload(Tx *tx.NodeTransaction) error {

	switch pld := Tx.Payload.(type) {
	case *uti_payload.RegisterAsset:
		if pld.Asset.Precision < asset.MinPrecision || pld.Asset.Precision > asset.MaxPrecision {
			return errors.New("Invalide asset Precision.")
		}
		if checkAmountPrecise(pld.Amount, pld.Asset.Precision) {
			return errors.New("Invalide asset value,out of precise.")
		}
	case *uti_payload.TransferAsset:
	case *uti_payload.Record:
	case *uti_payload.DeployCode:
	case *uti_payload.CoinBase:
	case *payload.WithdrawToken:
	case *payload.TransferCrossChainAsset:
	default:
		return errors.New("[txValidator],invalidate transaction payload type.")
	}
	return nil
}

func init() {
	Validator = &TransactionValidatorImpl{}
}
