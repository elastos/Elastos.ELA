package sdk

import (
	"errors"
	"math/rand"
	"sort"
	"strconv"

	"DNA_POW/account"
	. "DNA_POW/common"
	"DNA_POW/core/contract"
	"DNA_POW/core/transaction"
	"DNA_POW/core/ledger"
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

func sortAvailableCoinsByValue(coins map[*transaction.UTXOTxInput]*account.Coin) sortedCoins {
	var coinList sortedCoins
	for in, c := range coins {
		if c.Height <= ledger.DefaultLedger.Blockchain.GetBestHeight() {
			tmp := &sortedCoinsItem{
				input: in,
				coin:  c,
			}
			coinList = append(coinList, tmp)
		}
	}
	sort.Sort(coinList)
	return coinList
}

func MakeTransferTransaction(wallet account.Client, assetID Uint256, fee string, batchOut ...BatchOut) (*transaction.Transaction, error) {
	// get main account which is used to receive changes
	mainAccount, err := wallet.GetDefaultAccount()
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
			ProgramHash: address,
		}
		output = append(output, tmp)
	}

	// construct transaction inputs and changes
	coins := wallet.GetCoins()
	sorted := sortAvailableCoinsByValue(coins)
	for _, coinItem := range sorted {
		if coinItem.coin.Output.AssetID == assetID {
			input = append(input, coinItem.input)
			if coinItem.coin.Output.Value > expected {
				changes := &transaction.TxOutput{
					AssetID:     assetID,
					Value:       coinItem.coin.Output.Value - expected,
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
	txAttr := transaction.NewTxAttribute(transaction.Nonce, []byte(strconv.FormatInt(rand.Int63(), 10)))
	txn.Attributes = make([]*transaction.TxAttribute, 0)
	txn.Attributes = append(txn.Attributes, &txAttr)

	// sign transaction contract
	ctx := contract.NewContractContext(txn)
	wallet.Sign(ctx)
	txn.SetPrograms(ctx.GetPrograms())

	return txn, nil
}
