// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package unit

import (
	"testing"
	"time"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core"
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
	config.DefaultParams = *config.GetDefaultParams()
}

func TestGenesisBlock(t *testing.T) {
	mainNetFoundation := "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta"

	block := core.GenesisBlock(mainNetFoundation)
	assert.Equal(t, len(block.Transactions), 2)
	genesisHash := block.Hash().String()
	assert.Equal(t, "8d7014f2f941caa1972c8033b2f0a860ec8d4938b12bae2c62512852a558f405", genesisHash)

	testNetFoundation := "8ZNizBf4KhhPjeJRGpox6rPcHE5Np6tFx3"
	genesisHash = core.GenesisBlock(testNetFoundation).Hash().String()
	assert.Equal(t, "b3314f465ea5556d570bcc473d59a0855b4405a25b1ea0c957c81b2920be1864", genesisHash)

	date := time.Date(2017, time.December, 22, 10,
		0, 0, 0, time.UTC).Unix()
	dateUnix := time.Unix(time.Date(2017, time.December, 22, 10,
		0, 0, 0, time.UTC).Unix(), 0).Unix()

	dateTime, err := time.Parse(time.RFC3339, "2017-12-22T10:00:00Z")
	assert.NoError(t, err)
	assert.Equal(t, date, dateUnix)
	assert.Equal(t, date, dateTime.Unix())
}

func TestCheckAssetPrecision(t *testing.T) {
	tx := buildTx()
	ELAAssetID, _ := common.Uint256FromHexString(core.ELAAssetID)
	// valid precision
	for _, output := range tx.Outputs() {
		output.AssetID = *ELAAssetID
		output.ProgramHash = common.Uint168{}
		output.Value = 123456789876
	}
	err := blockchain.CheckAssetPrecision(tx)
	assert.NoError(t, err)

	for _, output := range tx.Outputs() {
		output.AssetID = *ELAAssetID
		output.ProgramHash = common.Uint168{}
		output.Value = 0
	}
	err = blockchain.CheckAssetPrecision(tx)
	assert.NoError(t, err)
}
