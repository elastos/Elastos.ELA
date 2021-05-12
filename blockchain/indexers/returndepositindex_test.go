// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package indexers

import (
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/database"
	"github.com/elastos/Elastos.ELA/utils/test"

	"github.com/stretchr/testify/assert"
)

var (
	returnDepositHash = common.Uint256{1, 2, 3}
	txOutput          = &types.Output{
		Value:      100000000,
		OutputLock: 0,
		Type:       types.OTReturnSideChainDepositCoin,
		Payload: &outputpayload.ReturnSideChainDeposit{
			Version:                0,
			DepositTransactionHash: returnDepositHash,
		},
	}
	tx5 = &types.Transaction{
		TxType:         types.ReturnSideChainDepositCoin,
		PayloadVersion: 0,
		Inputs:         []*types.Input{},
		Outputs:        []*types.Output{txOutput},
	}

	testReturnDepositIndexBlock = &types.Block{
		Header: types.Header{},
		Transactions: []*types.Transaction{
			tx5,
		},
	}

	testReturnDepositIndex *ReturnDepositIndex
	returnDepositIndexDB   database.DB
)

func TestReturnDepositIndexInit(t *testing.T) {
	log.NewDefault(test.NodeLogPath, 0, 0, 0)

	var err error
	returnDepositIndexDB, err = LoadBlockDB(test.DataPath)
	assert.NoError(t, err)

	testReturnDepositIndex = NewReturnDepositIndex(returnDepositIndexDB)
	assert.Equal(t, returnDepositIndexName, testReturnDepositIndex.Name())
	assert.Equal(t, returnDepositIndexKey, testReturnDepositIndex.Key())
	assert.NoError(t, testReturnDepositIndex.Init())

	_ = returnDepositIndexDB.Update(func(dbTx database.Tx) error {
		err := testReturnDepositIndex.Create(dbTx)
		assert.NoError(t, err)
		return err
	})
}

func TestReturnDepositIndex_ConnectBlock(t *testing.T) {
	_ = returnDepositIndexDB.Update(func(dbTx database.Tx) error {
		// returnDeposit should not in db
		assert.Equal(t, false, dbFetchReturnDepositIndexEntry(dbTx, &returnDepositHash))

		// connect the block
		err := testReturnDepositIndex.ConnectBlock(dbTx, testReturnDepositIndexBlock)
		assert.NoError(t, err)

		// returnDeposit should be stored in db
		assert.Equal(t, true, dbFetchReturnDepositIndexEntry(dbTx, &returnDepositHash))

		return err
	})
}

func TestReturnDepositIndex_Disconnect(t *testing.T) {
	_ = returnDepositIndexDB.Update(func(dbTx database.Tx) error {
		// returnDeposit should be stored in db
		assert.Equal(t, true, dbFetchReturnDepositIndexEntry(dbTx, &returnDepositHash))

		// disconnect the block
		err := testReturnDepositIndex.DisconnectBlock(dbTx, testReturnDepositIndexBlock)
		assert.NoError(t, err)

		// returnDeposit should be removed from db
		assert.Equal(t, false, dbFetchReturnDepositIndexEntry(dbTx, &returnDepositHash))

		return nil
	})
}

func TestReturnDepositIndexEnd(t *testing.T) {
	_ = returnDepositIndexDB.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()
		err := meta.DeleteBucket(returnDepositIndexKey)
		assert.NoError(t, err)
		return nil
	})
	returnDepositIndexDB.Close()
}
