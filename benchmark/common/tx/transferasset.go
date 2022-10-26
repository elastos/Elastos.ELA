// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package tx

import (
	"github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA/benchmark/common/utils"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

const (
	nonceByteLength = 20
)

type transferAssetGenerator struct {
	account []*account.Account
}

func (g *transferAssetGenerator) Generate() interfaces.Transaction {
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{
			{
				Usage: common2.Nonce,
				Data:  utils.RandomBytes(nonceByteLength),
			},
		},
		nil,
		[]*common2.Output{},
		0,
		nil,
	)
	ELAAssetID, _ := common.Uint256FromHexString(core.ELAAssetID)
	for _, v := range g.account {

		txn.SetOutputs(append(txn.Outputs(), &common2.Output{
			AssetID:     *ELAAssetID,
			Value:       0, // assign later
			OutputLock:  0,
			ProgramHash: v.ProgramHash,
			Type:        common2.OTNone,
			Payload:     &outputpayload.DefaultOutput{},
		}))
	}

	return txn
}
