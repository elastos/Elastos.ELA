// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package tx

import (
	"github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA/benchmark/common/utils"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/transactions"
)

const (
	defaultAmount = 10000000 // 0.1 ELA
)

type fixAmountAssigner struct {
	account *account.Account
	utxo    *common2.UTXO
}

func (a *fixAmountAssigner) SignAndChange(tx *transactions.BaseTransaction) error {
	tx.Inputs = []*common2.Input{
		{
			Previous: common2.OutPoint{
				TxID:  a.utxo.TxID,
				Index: a.utxo.Index,
			},
			Sequence: 0,
		},
	}

	for _, o := range tx.Outputs {
		o.Value = defaultAmount
	}
	tx.Outputs = append(tx.Outputs, &common2.Output{
		AssetID: config.ELAAssetID,
		Value: a.utxo.Value -
			defaultAmount*common.Fixed64(len(tx.Outputs)) - defaultFee,
		OutputLock:  0,
		ProgramHash: a.account.ProgramHash,
		Type:        common2.OTNone,
		Payload:     &outputpayload.DefaultOutput{},
	})

	return utils.SignStandardTx(tx, a.account)
}
