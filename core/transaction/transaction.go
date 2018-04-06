package transaction

import (
	"errors"
	"sort"

	. "github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/core/signature"
	uti_tx "github.com/elastos/Elastos.ELA.Utility/core/transaction"
)

var TxStore ILedgerStore

type NodeTransaction struct {
	uti_tx.Transaction

	//Inputs/Outputs map base on Asset (needn't serialize)
	AssetOutputs      map[Uint256][]*uti_tx.TxOutput
	AssetInputAmount  map[Uint256]Fixed64
	AssetOutputAmount map[Uint256]Fixed64
	Fee               Fixed64
	FeePerKB          Fixed64
}

func (txn *NodeTransaction) GetReference() (map[*uti_tx.UTXOTxInput]*uti_tx.TxOutput, error) {
	if txn.TxType == uti_tx.RegisterAsset {
		return nil, nil
	}
	//UTXO input /  Outputs
	reference := make(map[*uti_tx.UTXOTxInput]*uti_tx.TxOutput)
	// Key index，v UTXOInput
	for _, utxo := range txn.UTXOInputs {
		transaction, _, err := TxStore.GetTransaction(utxo.ReferTxID)
		if err != nil {
			return nil, errors.New("[NodeTransaction], GetReference failed.")
		}
		index := utxo.ReferTxOutputIndex
		if int(index) >= len(transaction.Outputs) {
			return nil, errors.New("[NodeTransaction], GetReference failed, refIdx out of range.")
		}
		reference[utxo] = transaction.Outputs[index]
	}
	return reference, nil
}
func (txn *NodeTransaction) GetTransactionResults() (TransactionResult, error) {
	result := make(map[Uint256]Fixed64)
	outputResult := txn.GetMergedAssetIDValueFromOutputs()
	InputResult, err := txn.GetMergedAssetIDValueFromReference()
	if err != nil {
		return nil, err
	}
	//calc the balance of input vs output
	for outputAssetid, outputValue := range outputResult {
		if inputValue, ok := InputResult[outputAssetid]; ok {
			result[outputAssetid] = inputValue - outputValue
		} else {
			result[outputAssetid] -= outputValue
		}
	}
	for inputAssetid, inputValue := range InputResult {
		if _, exist := result[inputAssetid]; !exist {
			result[inputAssetid] += inputValue
		}
	}
	return result, nil
}

func (txn *NodeTransaction) GetMergedAssetIDValueFromOutputs() TransactionResult {
	var result = make(map[Uint256]Fixed64)
	for _, v := range txn.Outputs {
		amout, ok := result[v.AssetID]
		if ok {
			result[v.AssetID] = amout + v.Value
		} else {
			result[v.AssetID] = v.Value
		}
	}
	return result
}

func (txn *NodeTransaction) GetMergedAssetIDValueFromReference() (TransactionResult, error) {
	reference, err := txn.GetReference()
	if err != nil {
		return nil, err
	}
	var result = make(map[Uint256]Fixed64)
	for _, v := range reference {
		amout, ok := result[v.AssetID]
		if ok {
			result[v.AssetID] = amout + v.Value
		} else {
			result[v.AssetID] = v.Value
		}
	}
	return result, nil
}

func (txn *NodeTransaction) GetFee(assetID Uint256) int64 {
	res, err := txn.GetTransactionResults()
	if err != nil {
		return 0
	}

	return int64(res[assetID])
}

func (txn *NodeTransaction) GetProgramHashes() ([]Uint168, error) {
	if txn == nil {
		return []Uint168{}, errors.New("[NodeTransaction],GetProgramHashes transaction is nil.")
	}
	hashs := []Uint168{}
	uniqHashes := []Uint168{}
	// add inputUTXO's transaction
	referenceWithUTXO_Output, err := txn.GetReference()
	if err != nil {
		return nil, errors.New("[NodeTransaction], GetProgramHashes failed.")
	}
	for _, output := range referenceWithUTXO_Output {
		programHash := output.ProgramHash
		hashs = append(hashs, programHash)
	}
	for _, attribute := range txn.Attributes {
		if attribute.Usage == uti_tx.Script {
			dataHash, err := Uint168FromBytes(attribute.Data)
			if err != nil {
				return nil, errors.New("[NodeTransaction], GetProgramHashes err.")
			}
			hashs = append(hashs, dataHash)
		}
	}
	switch txn.TxType {
	case uti_tx.RegisterAsset:
	case uti_tx.TransferAsset:
	case uti_tx.Record:
	case uti_tx.Deploy:
	case SideMining:
	case WithdrawToken:
	case TransferCrossChainAsset:
	default:
	}

	//remove dupilicated hashes
	uniq := make(map[Uint168]bool)
	for _, v := range hashs {
		uniq[v] = true
	}
	for k := range uniq {
		uniqHashes = append(uniqHashes, k)
	}
	sort.Sort(byProgramHashes(uniqHashes))
	return uniqHashes, nil
}

func (txn *NodeTransaction) GetOutputHashes() ([]Uint168, error) {
	//TODO: implement NodeTransaction.GetOutputHashes()

	return []Uint168{}, nil
}

func (txn *NodeTransaction) GenerateAssetMaps() {
	//TODO: implement NodeTransaction.GenerateAssetMaps()
}

func (txn *NodeTransaction) GetDataContent() []byte {
	return signature.GetDataContent(txn)
}

type byProgramHashes []Uint168

func (a byProgramHashes) Len() int      { return len(a) }
func (a byProgramHashes) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byProgramHashes) Less(i, j int) bool {
	if a[i].CompareTo(a[j]) > 0 {
		return false
	} else {
		return true
	}
}
