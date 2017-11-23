package transaction

import (
	"DNA_POW/common"
	"DNA_POW/core/contract/program"
	"DNA_POW/core/transaction/payload"
)

func NewCoinBaseTransaction(coinBasePayload *payload.CoinBase, currentHeight uint32) (*Transaction, error) {
	return &Transaction{
		TxType:         CoinBase,
		PayloadVersion: payload.CoinBasePayloadVersion,
		Payload:        coinBasePayload,
		UTXOInputs: []*UTXOTxInput{
			{
				ReferTxID:          common.Uint256{},
				ReferTxOutputIndex: 0x0000,
				Sequence:           0x00000000,
			},
		},
		BalanceInputs: []*BalanceTxInput{},
		Attributes:    []*TxAttribute{},
		LockTime:      currentHeight,
		Programs: []*program.Program{},
	}, nil
}

func NewTransferAssetTransaction(inputs []*UTXOTxInput, outputs []*TxOutput) (*Transaction, error) {
	assetRegPayload := &payload.TransferAsset{}

	return &Transaction{
		TxType:        TransferAsset,
		Payload:       assetRegPayload,
		Attributes:    []*TxAttribute{},
		UTXOInputs:    inputs,
		BalanceInputs: []*BalanceTxInput{},
		Outputs:       outputs,
		Programs:      []*program.Program{},
	}, nil
}

func NewRecordTransaction(recordType string, recordData []byte) (*Transaction, error) {
	recordPayload := &payload.Record{
		RecordType: recordType,
		RecordData: recordData,
	}

	return &Transaction{
		TxType:        Record,
		Payload:       recordPayload,
		Attributes:    []*TxAttribute{},
		UTXOInputs:    []*UTXOTxInput{},
		BalanceInputs: []*BalanceTxInput{},
		Programs:      []*program.Program{},
	}, nil
}
