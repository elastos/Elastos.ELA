// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package tx

import (
	"github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA/benchmark/common/utils"
	"github.com/elastos/Elastos.ELA/common/config"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/core/types/transactions"
)

const (
	nonceByteLength = 20
)

type transferAssetGenerator struct {
	account []*account.Account
}

func (g *transferAssetGenerator) Generate() *transactions.BaseTransaction {
	txn := &transactions.BaseTransaction{
		Version:        common2.TxVersion09,
		TxType:         common2.TransferAsset,
		PayloadVersion: 0,
		Payload:        &payload.TransferAsset{},
		Attributes: []*common2.Attribute{
			{
				Usage: common2.Nonce,
				Data:  utils.RandomBytes(nonceByteLength),
			},
		},
		Inputs:   nil,
		Outputs:  []*common2.Output{},
		LockTime: 0,
		Programs: nil,
	}
	for _, v := range g.account {
		txn.Outputs = append(txn.Outputs, &common2.Output{
			AssetID:     config.ELAAssetID,
			Value:       0, // assign later
			OutputLock:  0,
			ProgramHash: v.ProgramHash,
			Type:        common2.OTNone,
			Payload:     &outputpayload.DefaultOutput{},
		})
	}

	return txn
}
