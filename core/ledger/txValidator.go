package ledger

import (
	"DNA_POW/common/config"
	"errors"
	"fmt"
	"math"

	"DNA_POW/common"
	"DNA_POW/common/log"
	"DNA_POW/core/asset"
	tx "DNA_POW/core/transaction"
	"DNA_POW/core/transaction/payload"
	"DNA_POW/core/validation"
	"DNA_POW/crypto"
	. "DNA_POW/errors"
)

const (
	SpendCoinbaseSpan = 100
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
		return ErrNoError
	}

	if err := CheckTransactionBalance(txn); err != nil {
		log.Warn("[CheckTransactionBalance],", err)
		return ErrTransactionBalance
	}

	if err := CheckTransactionContracts(txn); err != nil {
		log.Warn("[CheckTransactionSignature],", err)
		return ErrTransactionContracts
	}

	return ErrNoError
}

// CheckTransactionContext verifys a transaction with history transaction in ledger
func CheckTransactionContext(txn *tx.Transaction, ledger *Ledger) ErrCode {
	// check if duplicated with transaction in ledger
	if exist := ledger.Store.IsTxHashDuplicate(txn.Hash()); exist {
		log.Info("[CheckTransactionContext] duplicate transaction check faild.")
		return ErrTxHashDuplicate
	}

	if txn.IsCoinBaseTx() {
		return ErrNoError
	}

	// check double spent transaction
	if IsDoubleSpend(txn, ledger) {
		log.Info("[CheckTransactionContext] IsDoubleSpend check faild.")
		return ErrDoubleSpend
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
			if currentHeight-lockHeight < config.Parameters.PowConfiguration.SpendCoinbaseSpan {
				return ErrIneffectiveCoinbase
			}
		}
	}

	return ErrNoError
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
		if v.Value <= common.Fixed64(0) {
			return errors.New("Invalide transaction UTXO output.")
		}
	}
	results, err := Tx.GetTransactionResults()
	if err != nil {
		return err
	}
	for k, v := range results {

		if v <= 0 {
			log.Debug(fmt.Sprintf("AssetID %x in Transfer transactions %x , input <= output .\n", k, Tx.Hash()))
			return errors.New(fmt.Sprintf("AssetID %x in Transfer transactions %x , input <= output .\n", k, Tx.Hash()))
		}
	}
	return nil
}

func CheckAttributeProgram(Tx *tx.Transaction) error {
	//TODO: implement CheckAttributeProgram
	return nil
}

func CheckTransactionContracts(Tx *tx.Transaction) error {
	flag, err := validation.VerifySignableData(Tx)
	if flag && err == nil {
		return nil
	} else {
		return err
	}
}

func checkAmountPrecise(amount common.Fixed64, precision byte) bool {
	return amount.GetData()%int64(math.Pow(10, 8-float64(precision))) != 0
}

func checkIssuerInBookkeeperList(issuer *crypto.PubKey, bookKeepers []*crypto.PubKey) bool {
	for _, bk := range bookKeepers {
		r := crypto.Equal(issuer, bk)
		if r == true {
			log.Debug("issuer is in bookkeeperlist")
			return true
		}
	}
	log.Debug("issuer is NOT in bookkeeperlist")
	return false
}

func CheckTransactionPayload(Tx *tx.Transaction) error {

	switch pld := Tx.Payload.(type) {
	case *payload.BookKeeper:
		//Todo: validate bookKeeper Cert
		_ = pld.Cert
		bookKeepers, _, _ := DefaultLedger.Store.GetBookKeeperList()
		r := checkIssuerInBookkeeperList(pld.Issuer, bookKeepers)
		if r == false {
			return errors.New("The issuer isn't bookekeeper, can't add other in bookkeepers list.")
		}
		return nil
	case *payload.RegisterAsset:
		if pld.Asset.Precision < asset.MinPrecision || pld.Asset.Precision > asset.MaxPrecision {
			return errors.New("Invalide asset Precision.")
		}
		if checkAmountPrecise(pld.Amount, pld.Asset.Precision) {
			return errors.New("Invalide asset value,out of precise.")
		}
	case *payload.IssueAsset:
	case *payload.TransferAsset:
	case *payload.BookKeeping:
	case *payload.PrivacyPayload:
	case *payload.Record:
	case *payload.DeployCode:
	case *payload.DataFile:
	case *payload.CoinBase:
	default:
		return errors.New("[txValidator],invalidate transaction payload type.")
	}
	return nil
}
