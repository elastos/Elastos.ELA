// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package unit

import (
	"testing"

	"github.com/elastos/Elastos.ELA/blockchain/indexers"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/database"
	"github.com/elastos/Elastos.ELA/utils/test"

	"github.com/stretchr/testify/assert"
)

func init() {
	functions.GetTransactionByTxType = transaction.GetTransaction
	functions.GetTransactionByBytes = transaction.GetTransactionByBytes
	functions.CreateTransaction = transaction.CreateTransaction
	functions.GetTransactionParameters = transaction.GetTransactionparameters
	config.DefaultParams = config.GetDefaultParams()
}

var (
	tx3Hash = common.Uint256{1, 2, 3}

	tx4               interfaces.Transaction
	testTx3IndexBlock *types.Block

	testTx3Index *indexers.Tx3Index
	tx3IndexDB   database.DB
)

func initIndexBlock() {
	tx4 = functions.CreateTransaction(
		0,
		common2.WithdrawFromSideChain,
		0,
		&payload.WithdrawFromSideChain{
			SideChainTransactionHashes: []common.Uint256{
				tx3Hash,
			},
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	testTx3IndexBlock = &types.Block{
		Header: common2.Header{},
		Transactions: []interfaces.Transaction{
			tx4,
		},
	}
}

func TestTx3IndexInit(t *testing.T) {
	log.NewDefault(test.NodeLogPath, 0, 0, 0)

	var err error
	tx3IndexDB, err = LoadBlockDB(test.DataPath)
	assert.NoError(t, err)

	testTx3Index = indexers.NewTx3Index(tx3IndexDB)
	assert.Equal(t, indexers.Tx3IndexName, testTx3Index.Name())
	assert.Equal(t, indexers.Tx3IndexKey, testTx3Index.Key())
	assert.NoError(t, testTx3Index.Init())

	_ = tx3IndexDB.Update(func(dbTx database.Tx) error {
		err := testTx3Index.Create(dbTx)
		assert.NoError(t, err)
		return err
	})
}

func TestTx3Index_ConnectBlock(t *testing.T) {
	initIndexBlock()
	_ = tx3IndexDB.Update(func(dbTx database.Tx) error {
		// tx3 should not in DB
		assert.Equal(t, false, indexers.DBFetchTx3IndexEntry(dbTx, &tx3Hash))

		// connect the block
		err := testTx3Index.ConnectBlock(dbTx, testTx3IndexBlock)
		assert.NoError(t, err)

		// tx3 should be stored in DB
		assert.Equal(t, true, indexers.DBFetchTx3IndexEntry(dbTx, &tx3Hash))

		return err
	})
}

func TestTx3Index_Disconnect(t *testing.T) {
	initIndexBlock()
	_ = tx3IndexDB.Update(func(dbTx database.Tx) error {
		// tx3 should be stored in DB
		assert.Equal(t, true, indexers.DBFetchTx3IndexEntry(dbTx, &tx3Hash))

		// disconnect the block
		err := testTx3Index.DisconnectBlock(dbTx, testTx3IndexBlock)
		assert.NoError(t, err)

		// tx3 should be removed from DB
		assert.Equal(t, false, indexers.DBFetchTx3IndexEntry(dbTx, &tx3Hash))

		return nil
	})
}

func TestTx3IndexEnd(t *testing.T) {
	_ = tx3IndexDB.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()
		err := meta.DeleteBucket(indexers.Tx3IndexKey)
		assert.NoError(t, err)
		return nil
	})
	tx3IndexDB.Close()
}
