// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package unit

import (
	"testing"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/stretchr/testify/assert"
)

func init() {
	testing.Init()

	functions.GetTransactionByTxType = transaction.GetTransaction
	functions.GetTransactionByBytes = transaction.GetTransactionByBytes
	functions.CreateTransaction = transaction.CreateTransaction
	functions.GetTransactionParameters = transaction.GetTransactionparameters
	config.DefaultParams = config.GetDefaultParams()
}

func TestCheckAssetPrecision(t *testing.T) {
	tx := buildTx()
	// valid precision
	for _, output := range tx.Outputs() {
		output.AssetID = config.ELAAssetID
		output.ProgramHash = common.Uint168{}
		output.Value = 123456789876
	}
	err := blockchain.CheckAssetPrecision(tx)
	assert.NoError(t, err)

	for _, output := range tx.Outputs() {
		output.AssetID = config.ELAAssetID
		output.ProgramHash = common.Uint168{}
		output.Value = 0
	}
	err = blockchain.CheckAssetPrecision(tx)
	assert.NoError(t, err)
}
