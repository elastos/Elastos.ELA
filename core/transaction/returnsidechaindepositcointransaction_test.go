// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"encoding/hex"
	"github.com/btcsuite/btcd/wire"
	"github.com/elastos/Elastos.ELA/blockchain/indexers"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/database"
	"github.com/elastos/Elastos.ELA/utils/test"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

const (
	// blockDbNamePrefix is the prefix for the block database name.  The
	// database type is appended to this value to form the full block
	// database name.
	blockDbNamePrefix = "blocks"
)

var (
	returnDepositHash = common.Uint256{1, 2, 3}
	txOutput          = &common2.Output{
		Value:      100000000,
		OutputLock: 0,
		Type:       common2.OTReturnSideChainDepositCoin,
		Payload: &outputpayload.ReturnSideChainDeposit{
			Version:                0,
			DepositTransactionHash: returnDepositHash,
		},
	}

	tx5                         interfaces.Transaction
	testReturnDepositIndexBlock = &types.Block{}

	testReturnDepositIndex *indexers.ReturnDepositIndex
	returnDepositIndexDB   database.DB
)

func initReturnDepositIndexBlock() {
	tx5 = functions.CreateTransaction(
		0,
		common2.ReturnSideChainDepositCoin,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{txOutput},
		0,
		[]*program.Program{},
	)

	testReturnDepositIndexBlock = &types.Block{
		Header: common2.Header{},
		Transactions: []interfaces.Transaction{
			tx5,
		},
	}
}

func TestReturnDepositIndex(t *testing.T) {
	initReturnDepositIndexBlock()
	var err error
	returnDepositIndexDB, err = LoadBlockDB(filepath.Join(test.DataPath, "returndeposit"))
	assert.NoError(t, err)

	{
		testReturnDepositIndex = indexers.NewReturnDepositIndex(returnDepositIndexDB)
		assert.Equal(t, indexers.ReturnDepositIndexName, testReturnDepositIndex.Name())
		assert.Equal(t, indexers.ReturnDepositIndexKey, testReturnDepositIndex.Key())
		assert.NoError(t, testReturnDepositIndex.Init())

		_ = returnDepositIndexDB.Update(func(dbTx database.Tx) error {
			err := testReturnDepositIndex.Create(dbTx)
			assert.NoError(t, err)
			return err
		})
	}
	{
		testReturnDepositIndex = indexers.NewReturnDepositIndex(returnDepositIndexDB)
		assert.Equal(t, indexers.ReturnDepositIndexName, testReturnDepositIndex.Name())
		assert.Equal(t, indexers.ReturnDepositIndexKey, testReturnDepositIndex.Key())
		assert.NoError(t, testReturnDepositIndex.Init())

		_ = returnDepositIndexDB.Update(func(dbTx database.Tx) error {
			err := testReturnDepositIndex.Create(dbTx)
			// returnDeposit should not in DB
			assert.Equal(t, false, indexers.DBFetchReturnDepositIndexEntry(dbTx, &returnDepositHash))

			// connect the block
			err = testReturnDepositIndex.ConnectBlock(dbTx, testReturnDepositIndexBlock)
			assert.NoError(t, err)

			// returnDeposit should be stored in DB
			assert.Equal(t, true, indexers.DBFetchReturnDepositIndexEntry(dbTx, &returnDepositHash))

			return err
		})
	}

	{
		testReturnDepositIndex = indexers.NewReturnDepositIndex(returnDepositIndexDB)
		assert.Equal(t, indexers.ReturnDepositIndexName, testReturnDepositIndex.Name())
		assert.Equal(t, indexers.ReturnDepositIndexKey, testReturnDepositIndex.Key())
		assert.NoError(t, testReturnDepositIndex.Init())
		_ = returnDepositIndexDB.Update(func(dbTx database.Tx) error {
			err := testReturnDepositIndex.Create(dbTx)
			// returnDeposit should be stored in DB
			assert.Equal(t, true, indexers.DBFetchReturnDepositIndexEntry(dbTx, &returnDepositHash))

			// disconnect the block
			err = testReturnDepositIndex.DisconnectBlock(dbTx, testReturnDepositIndexBlock)
			assert.NoError(t, err)

			// returnDeposit should be removed from DB
			assert.Equal(t, false, indexers.DBFetchReturnDepositIndexEntry(dbTx, &returnDepositHash))

			return nil
		})
	}

	{
		testReturnDepositIndex = indexers.NewReturnDepositIndex(returnDepositIndexDB)
		assert.Equal(t, indexers.ReturnDepositIndexName, testReturnDepositIndex.Name())
		assert.Equal(t, indexers.ReturnDepositIndexKey, testReturnDepositIndex.Key())
		assert.NoError(t, testReturnDepositIndex.Init())
		_ = returnDepositIndexDB.Update(func(dbTx database.Tx) error {
			err := testReturnDepositIndex.Create(dbTx)
			meta := dbTx.Metadata()
			err = meta.DeleteBucket(indexers.ReturnDepositIndexKey)
			assert.NoError(t, err)
			return nil
		})
		returnDepositIndexDB.Close()
	}
}

func (s *txValidatorTestSuite) TestReturnSideChainDepositCoinTransaction() {
	payload := &payload.ReturnSideChainDepositCoin{
		Signers: randomBytes(32),
	}
	{

		txn := functions.CreateTransaction(
			common2.TxVersion09,
			common2.ReturnSideChainDepositCoin,
			0,
			payload,
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{
				&common2.Output{
					Type:  common2.OTReturnSideChainDepositCoin,
					Value: 10*10 ^ 8,
				},
			},
			0,
			[]*program.Program{},
		)
		txn = CreateTransactionByType(txn, s.Chain)
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:invalid ReturnSideChainDeposit output payload")
		depositTx := *randomUint256()
		txn.SetOutputs([]*common2.Output{
			{
				Type:  common2.OTReturnSideChainDepositCoin,
				Value: 10*10 ^ 8,
				Payload: &outputpayload.ReturnSideChainDeposit{
					DepositTransactionHash: depositTx,
				},
			},
		})
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:invalid deposit tx:"+hex.EncodeToString(depositTx.Bytes()))
	}
}

// loadBlockDB loads (or creates when needed) the block database taking into
// account the selected database backend and returns a handle to it.  It also
// contains additional logic such warning the user if there are multiple
// databases which consume space on the file system and ensuring the regression
// test database is clean when in regression test mode.
func LoadBlockDB(dataPath string) (database.DB, error) {
	// The memdb backend does not have a file path associated with it, so
	// handle it uniquely.  We also don't want to worry about the multiple
	// database type warnings when running with the memory database.

	// The database name is based on the database type.
	dbType := "ffldb"
	dbPath := blockDbPath(dataPath, dbType)

	log.Infof("Loading block database from '%s'", dbPath)
	db, err := database.Open(dbType, dbPath, wire.MainNet)
	if err != nil {
		// Return the error if it's not because the database doesn't
		// exist.
		if dbErr, ok := err.(database.Error); !ok || dbErr.ErrorCode !=
			database.ErrDbDoesNotExist {

			return nil, err
		}

		// Create the db if it does not exist.
		err = os.MkdirAll(dataPath, 0700)
		if err != nil {
			return nil, err
		}
		db, err = database.Create(dbType, dbPath, wire.MainNet)
		if err != nil {
			return nil, err
		}
	}

	log.Info("Block database loaded")
	return db, nil
}

// dbPath returns the path to the block database given a database type.
func blockDbPath(dataPath, dbType string) string {
	// The database name is based on the database type.
	dbName := blockDbNamePrefix + "_" + dbType
	if dbType == "sqlite" {
		dbName = dbName + ".db"
	}
	dbPath := filepath.Join(dataPath, dbName)
	return dbPath
}
