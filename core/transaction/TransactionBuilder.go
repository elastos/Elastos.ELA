package transaction

import (
	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/core/contract/program"
	tx "github.com/elastos/Elastos.ELA.Utility/core/transaction"
	"github.com/elastos/Elastos.ELA.Utility/core/transaction/payload"
)

func NewCoinBaseTransaction(coinBasePayload *payload.CoinBase, currentHeight uint32) (*NodeTransaction, error) {
	return &NodeTransaction{Transaction: tx.Transaction{
		TxType:         tx.CoinBase,
		PayloadVersion: payload.CoinBasePayloadVersion,
		Payload:        coinBasePayload,
		UTXOInputs: []*tx.UTXOTxInput{
			{
				ReferTxID:          common.Uint256{},
				ReferTxOutputIndex: 0x0000,
				Sequence:           0x00000000,
			},
		},
		BalanceInputs: []*tx.BalanceTxInput{},
		Attributes:    []*tx.TxAttribute{},
		LockTime:      currentHeight,
		Programs:      []*program.Program{},
	},
	}, nil
}
