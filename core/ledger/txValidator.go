package ledger

import (
	"Elastos.ELA/common/config"
	"errors"
	"fmt"
	"math"

	"Elastos.ELA/common"
	"Elastos.ELA/common/log"
	"Elastos.ELA/core/asset"
	tx "Elastos.ELA/core/transaction"
	"Elastos.ELA/core/transaction/payload"
	. "Elastos.ELA/errors"
)

// CheckTransactionSanity verifys received single transaction
func CheckTransactionSanity(txn *tx.Transaction) ErrCode {

	if err := CheckTransactionSize(txn); err != nil {
		log.Warn("[CheckTransactionSize],", err)
		return ErrTransactionSize
	}

	if err := CheckTransactionInput(txn); err != nil {
		log.Warn("[CheckTransactionInput],", err)
		return ErrInvalidInput
	}

	if err := CheckTransactionOutput(txn); err != nil {
		log.Warn("[CheckTransactionOutput],", err)
		return ErrInvalidOutput
	}

	if err := CheckAssetPrecision(txn); err != nil {
		log.Warn("[CheckAssetPrecesion],", err)
		return ErrAssetPrecision
	}

	if err := CheckAttributeProgram(txn); err != nil {
		log.Warn("[CheckTransactionAttribute],", err)
		return ErrAttributeProgram
	}

	if err := CheckTransactionPayload(txn); err != nil {
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
func CheckTransactionContext(txn *tx.Transaction, ledger *Ledger) ErrCode {
	// check if duplicated with transaction in ledger
	if exist := ledger.Store.IsTxHashDuplicate(txn.Hash()); exist {
		log.Info("[CheckTransactionContext] duplicate transaction check faild.")
		return ErrTxHashDuplicate
	}

	if txn.IsCoinBaseTx() {
		return Success
	}

	// check double spent transaction
	if IsDoubleSpend(txn, ledger) {
		log.Info("[CheckTransactionContext] IsDoubleSpend check faild.")
		return ErrDoubleSpend
	}

	if err := CheckTransactionUTXOLock(txn); err != nil {
		log.Warn("[CheckTransactionUTXOLock],", err)
		return ErrUTXOLocked
	}

	if err := CheckTransactionBalance(txn); err != nil {
		log.Warn("[CheckTransactionBalance],", err)
		return ErrTransactionBalance
	}

	if err := CheckTransactionSignature(txn); err != nil {
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
		if referTxnOut.Value < 0 {
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
func CheckTransactionInput(txn *tx.Transaction) error {
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

func CheckTransactionOutput(txn *tx.Transaction) error {
	if txn.IsCoinBaseTx() {
		if len(txn.Outputs) < 2 {
			return errors.New("coinbase output is not enough, at least 2")
		}
		found := false
		for _, output := range txn.Outputs {
			if output.AssetID != DefaultLedger.Blockchain.AssetID {
				return errors.New("asset ID in coinbase is invalid")
			}
			addrInCoinbaseOutput, _ := output.ProgramHash.ToAddress()
			if addrInCoinbaseOutput == FoundationAddress {
				found = true
			}
		}
		if !found {
			return errors.New("no foundation address in coinbase output")
		}

		return nil
	}

	if len(txn.Outputs) <= 0 {
		return errors.New("transaction has no outputs")
	}

	return nil
}

func CheckTransactionUTXOLock(txn *tx.Transaction) error {
	if txn.IsCoinBaseTx() {
		return nil
	}
	if len(txn.UTXOInputs) <= 0 {
		return errors.New("Transaction has no inputs")
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

func CheckTransactionSize(txn *tx.Transaction) error {
	size := txn.GetSize()
	if size <= 0 || size > MaxBlockSize {
		return errors.New(fmt.Sprintf("Invalid transaction size: %d bytes", size))
	}

	return nil
}

func IsDoubleSpend(tx *tx.Transaction, ledger *Ledger) bool {
	return ledger.IsDoubleSpend(tx)
}

func CheckAssetPrecision(Tx *tx.Transaction) error {
	if len(Tx.Outputs) == 0 {
		return nil
	}
	assetOutputs := make(map[common.Uint256][]*tx.TxOutput, len(Tx.Outputs))

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

func CheckTransactionBalance(Tx *tx.Transaction) error {
	// TODO: check coinbase balance 30%-70%
	for _, v := range Tx.Outputs {
		if v.Value < common.Fixed64(0) {
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

func CheckAttributeProgram(txn *tx.Transaction) error {
	//TODO: implement CheckAttributeProgram
	for _, v := range txn.Outputs {
		address, _ := v.ProgramHash.ToAddress()
		prefix := address[0:1]
		if (prefix != "E" && prefix != "8") && !InAddressList(address) {
			return errors.New("invalid prefix " + prefix + " in address " + address)
		}
	}
	return nil
}

func InAddressList(address string) bool {
	addresses := [7]string{
		"AY88Sf2PqvDwrPefskbpsLQjfRHG4C6XY7",
		"AHC2skXznXvZFcv8jTdVy3qovLbZdDgEcZ",
		"APP7fGVJkiCHhURuPotxKFXf5HjyvpH52r",
		"AUtUKTsQVEAGjj4cE83jMQbsvyjqm3spNf",
		"AexFnGMhF1EnACJkd7iKhh8eKG2qPJcDcf",
		"AS3N7PWLPNARFBgkc9SW9qwiMqND17kSS7",
		"AKLMhPk1CW9HsV6UW5bMLrqz2y1ZKHcrSn",
	}
	for _, v := range addresses {
		if v == address {
			return true
		}
	}
	return false
}

func CheckTransactionSignature(txn *tx.Transaction) error {
	return tx.VerifySignature(txn)
}

func checkAmountPrecise(amount common.Fixed64, precision byte) bool {
	return amount.GetData()%int64(math.Pow(10, 8-float64(precision))) != 0
}

func CheckTransactionPayload(Tx *tx.Transaction) error {

	switch pld := Tx.Payload.(type) {
	case *payload.RegisterAsset:
		if pld.Asset.Precision < asset.MinPrecision || pld.Asset.Precision > asset.MaxPrecision {
			return errors.New("Invalide asset Precision.")
		}
		if checkAmountPrecise(pld.Amount, pld.Asset.Precision) {
			return errors.New("Invalide asset value,out of precise.")
		}
	case *payload.TransferAsset:
	case *payload.Record:
	case *payload.DeployCode:
	case *payload.CoinBase:
	default:
		return errors.New("[txValidator],invalidate transaction payload type.")
	}
	return nil
}
