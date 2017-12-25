package sdk

import (
	"Elastos.ELA/account"
	. "Elastos.ELA/common"
	"Elastos.ELA/core/contract"
	"Elastos.ELA/core/ledger"
	"Elastos.ELA/core/signature"
	"Elastos.ELA/core/transaction"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"Elastos.ELA/common/log"
)

type BatchOut struct {
	Address string
	Value   string
}

type sortedCoinsItem struct {
	input *transaction.UTXOTxInput
	coin  *account.Coin
}

// sortedCoins used for spend minor coins first
type sortedCoins []*sortedCoinsItem

func (sc sortedCoins) Len() int      { return len(sc) }
func (sc sortedCoins) Swap(i, j int) { sc[i], sc[j] = sc[j], sc[i] }
func (sc sortedCoins) Less(i, j int) bool {
	if sc[i].coin.Output.Value > sc[j].coin.Output.Value {
		return false
	} else {
		return true
	}
}

func sortAvailableCoinsByValue(coins map[*transaction.UTXOTxInput]*account.Coin, addrtype account.AddressType) sortedCoins {
	var coinList sortedCoins
	for in, c := range coins {
		if c.Height <= ledger.DefaultLedger.Blockchain.GetBestHeight() {
			if c.AddressType == addrtype {
				tmp := &sortedCoinsItem{
					input: in,
					coin:  c,
				}
				coinList = append(coinList, tmp)
			}
		}
	}
	sort.Sort(coinList)
	return coinList
}

func MakeTransferTransaction(wallet account.Client, assetID Uint256, fee string, lock string, batchOut ...BatchOut) (*transaction.Transaction, error) {
	// get main account which is used to receive changes
	mainAccount, err := wallet.GetDefaultAccount()
	if err != nil {
		return nil, err
	}
	utxolock, err := strconv.ParseUint(lock, 10, 32)
	if err != nil {
		return nil, err
	}
	// construct transaction outputs
	var expected Fixed64
	input := []*transaction.UTXOTxInput{}
	output := []*transaction.TxOutput{}
	txnfee, err := StringToFixed64(fee)
	if err != nil || txnfee <= 0 {
		return nil, errors.New("invalid transation fee")
	}
	expected += txnfee
	for _, o := range batchOut {
		outputValue, err := StringToFixed64(o.Value)
		if err != nil {
			return nil, err
		}
		expected += outputValue
		address, err := ToScriptHash(o.Address)
		if err != nil {
			return nil, errors.New("invalid address")
		}
		tmp := &transaction.TxOutput{
			AssetID:     assetID,
			Value:       outputValue,
			OutputLock:  uint32(utxolock),
			ProgramHash: address,
		}
		output = append(output, tmp)
	}

	// construct transaction inputs and changes
	coins := wallet.GetCoins()
	sorted := sortAvailableCoinsByValue(coins, account.SingleSign)
	for _, coinItem := range sorted {
		if coinItem.coin.Output.AssetID == assetID {
			if coinItem.coin.Output.OutputLock > 0 {
				//can not unlock
				if ledger.DefaultLedger.Blockchain.GetBestHeight() < coinItem.coin.Output.OutputLock {
					continue
				}
				//spend locked utxo,change the  input Sequence
				coinItem.input.Sequence = math.MaxUint32 - 1
			}
			input = append(input, coinItem.input)
			if coinItem.coin.Output.Value > expected {
				changes := &transaction.TxOutput{
					AssetID:     assetID,
					Value:       coinItem.coin.Output.Value - expected,
					OutputLock:  0,
					ProgramHash: mainAccount.ProgramHash,
				}
				// if any, the changes output of transaction will be the last one
				output = append(output, changes)
				expected = 0
				break
			} else if coinItem.coin.Output.Value == expected {
				expected = 0
				break
			} else if coinItem.coin.Output.Value < expected {
				expected = expected - coinItem.coin.Output.Value
			}
		}
	}
	if expected > 0 {
		return nil, errors.New("available token is not enough")
	}

	// construct transaction
	txn, err := transaction.NewTransferAssetTransaction(input, output)
	if err != nil {
		return nil, err
	}
	txn.LockTime = ledger.DefaultLedger.Blockchain.GetBestHeight()

	txAttr := transaction.NewTxAttribute(transaction.Nonce, []byte(strconv.FormatInt(rand.Int63(), 10)))
	txn.Attributes = make([]*transaction.TxAttribute, 0)
	txn.Attributes = append(txn.Attributes, &txAttr)

	// sign transaction contract
	ctx := contract.NewContractContext(txn)
	wallet.Sign(ctx)
	txn.SetPrograms(ctx.GetPrograms())

	return txn, nil
}

func MakeMultisigTransferTransaction(wallet account.Client, assetID Uint256, from string, fee string, batchOut ...BatchOut) (*transaction.Transaction, error) {
	//TODO: check if being transferred asset is System Token(IPT)
	outputNum := len(batchOut)
	if outputNum == 0 {
		return nil, errors.New("nil outputs")
	}

	spendAddress, err := ToScriptHash(from)
	if err != nil {
		return nil, errors.New("invalid sender address")
	}

	var expected Fixed64
	input := []*transaction.UTXOTxInput{}
	output := []*transaction.TxOutput{}
	txnfee, err := StringToFixed64(fee)
	if err != nil || txnfee <= 0 {
		return nil, errors.New("invalid transation fee")
	}
	expected += txnfee
	// construct transaction outputs
	for _, o := range batchOut {
		outputValue, err := StringToFixed64(o.Value)
		if err != nil {
			return nil, err
		}

		expected += outputValue
		address, err := ToScriptHash(o.Address)
		if err != nil {
			return nil, errors.New("invalid receiver address")
		}
		tmp := &transaction.TxOutput{
			AssetID:     assetID,
			Value:       outputValue,
			ProgramHash: address,
		}
		output = append(output, tmp)
	}
	log.Debug("expected = %v\n", expected)
	// construct transaction inputs and changes
	coins := wallet.GetCoins()
	sorted := sortAvailableCoinsByValue(coins, account.MultiSign)
	for _, coinItem := range sorted {
		if coinItem.coin.Output.AssetID == assetID && coinItem.coin.Output.ProgramHash == spendAddress {
			input = append(input, coinItem.input)
			log.Debug("coinItem.coin.Output.Value = %v ProgramHash = %x\n", coinItem.coin.Output.Value, spendAddress.ToArrayReverse())
			if coinItem.coin.Output.Value > expected {
				changes := &transaction.TxOutput{
					AssetID:     assetID,
					Value:       coinItem.coin.Output.Value - expected,
					ProgramHash: spendAddress,
				}
				// if any, the changes output of transaction will be the last one
				output = append(output, changes)
				expected = 0
				break
			} else if coinItem.coin.Output.Value == expected {
				expected = 0
				break
			} else if coinItem.coin.Output.Value < expected {
				expected = expected - coinItem.coin.Output.Value
				fmt.Printf("expected - coinItem.coin.Output.Value = %v\n", expected)
			}
		}
	}
	if expected > 0 {
		return nil, errors.New("available token is not enough")
	}

	// construct transaction
	txn, err := transaction.NewTransferAssetTransaction(input, output)
	if err != nil {
		return nil, err
	}
	txAttr := transaction.NewTxAttribute(transaction.Nonce, []byte(strconv.FormatInt(rand.Int63(), 10)))
	txn.Attributes = make([]*transaction.TxAttribute, 0)
	txn.Attributes = append(txn.Attributes, &txAttr)

	ctx := contract.NewContractContext(txn)
	err = wallet.Sign(ctx)
	if err != nil {
		fmt.Println(err)
	}

	if ctx.IsCompleted() {
		txn.SetPrograms(ctx.GetPrograms())
	} else {
		txn.SetPrograms(ctx.GetUncompletedPrograms())
	}

	return txn, nil
}

func signTransaction(signer *account.Account, tx *transaction.Transaction) error {
	signature, err := signature.SignBySigner(tx, signer)
	if err != nil {
		fmt.Println("SignBySigner failed")
		return err
	}
	transactionContract, err := contract.CreateSignatureContract(signer.PubKey())
	if err != nil {
		fmt.Println("CreateSignatureContract failed")
		return err
	}
	transactionContractContext := newContractContextWithoutProgramHashes(tx, 1)
	if err := transactionContractContext.AddContract(transactionContract, signer.PubKey(), signature); err != nil {
		fmt.Println("SaveContract failed")
		return err
	}
	tx.SetPrograms(transactionContractContext.GetPrograms())
	return nil
}

func newContractContextWithoutProgramHashes(data signature.SignableData, length int) *contract.ContractContext {
	return &contract.ContractContext{
		Data:       data,
		Codes:      make([][]byte, length),
		Parameters: make([][][]byte, length),
	}
}
