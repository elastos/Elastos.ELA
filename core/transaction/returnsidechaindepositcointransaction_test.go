// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"encoding/hex"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/btcsuite/btcd/wire"
	"github.com/elastos/Elastos.ELA/blockchain/indexers"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/database"
	"github.com/elastos/Elastos.ELA/utils/test"
	"github.com/stretchr/testify/assert"
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

// TestReturnSideChainDepositCoinCrossChainAuthorization verifies Type-81/V0
// remains unchanged before H and requires the current arbiter witness at H.
func (s *txValidatorSpecialTxTestSuite) TestReturnSideChainDepositCoinCrossChainAuthorization() {
	chainParams := *s.Chain.GetParams()
	chainParams.CrossChainUTXOFreezeHeight = 0
	chainParams.CrossChainUTXORestrictionHeight = 1
	chainParams.DPoSConfiguration.DPOSNodeCrossChainHeight = math.MaxUint32
	chainParams.CRConfiguration.CRAgreementCount = uint32(s.arbitrators.MajorityCount)

	originalCRCArbiters := s.arbitrators.CRCArbitrators
	s.arbitrators.CRCArbitrators = s.arbitrators.CurrentArbitrators
	defer func() {
		s.arbitrators.CRCArbitrators = originalCRCArbiters
	}()

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.ReturnSideChainDepositCoin,
		payload.ReturnSideChainDepositCoinVersion,
		&payload.ReturnSideChainDepositCoin{},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		nil,
	)
	txn = CreateTransactionByType(txn, s.Chain)
	parameters := &TransactionParameters{
		Transaction: txn,
		BlockHeight: 0,
		Config:      &chainParams,
		BlockChain:  s.Chain,
	}
	txn.SetParameters(parameters)
	txn.SetReferences(crossChainUTXOReferences(contract.PrefixCrossChain))

	err, _ := txn.SpecialContextCheck()
	s.NoError(err)

	parameters.BlockHeight = 1
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:CrossChain transaction has no programs")

	txn.SetPrograms([]*program.Program{{Code: s.crossChainArbiterScript(
		s.arbitrators.MajorityCount, len(s.arbitrators.GetCrossChainArbiters()))}})
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)
	s.signCrossChainProgram(txn, txn.Programs()[0])
	s.NoError(checkTransactionSignature(txn,
		crossChainUTXOReferences(contract.PrefixCrossChain)))

	txn.SetPrograms([]*program.Program{{Code: s.crossChainArbiterScript(1, 2)}})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:invalid arbiters total count in code")
}

// TestReturnSideChainDepositCoinV0ArbiterFixture verifies an Arbiter-shaped
// refund completes sanity and contextual validation at H.
func (s *txValidatorSpecialTxTestSuite) TestReturnSideChainDepositCoinV0ArbiterFixture() {
	fixture := s.createArbiterCrossChainUTXOFixture()
	blockHeight := fixture.transactionHeight + 1
	chainParams := *s.Chain.GetParams()
	chainParams.CrossChainUTXOFreezeHeight = 0
	chainParams.CrossChainUTXORestrictionHeight = blockHeight
	chainParams.ReturnCrossChainCoinStartHeight = 0
	chainParams.DPoSConfiguration.DPOSNodeCrossChainHeight = math.MaxUint32
	chainParams.CRConfiguration.CRAgreementCount = uint32(s.arbitrators.MajorityCount)

	originalCRCArbiters := s.arbitrators.CRCArbitrators
	s.arbitrators.CRCArbitrators = s.arbitrators.CurrentArbitrators
	defer func() {
		s.arbitrators.CRCArbitrators = originalCRCArbiters
	}()

	witness := &program.Program{Code: s.crossChainArbiterScript(
		s.arbitrators.MajorityCount, len(s.arbitrators.GetCrossChainArbiters()))}
	nonce := common2.NewAttribute(common2.Nonce, []byte("crosschain-utxo-return"))
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.ReturnSideChainDepositCoin,
		payload.ReturnSideChainDepositCoinVersion,
		&payload.ReturnSideChainDepositCoin{},
		[]*common2.Attribute{&nonce},
		[]*common2.Input{fixture.depositInput},
		[]*common2.Output{{
			AssetID:     core.ELAAssetID,
			Value:       fixture.reserveAmount - chainParams.ReturnDepositCoinFee,
			ProgramHash: fixture.payerProgramHash,
			Type:        common2.OTReturnSideChainDepositCoin,
			Payload: &outputpayload.ReturnSideChainDeposit{
				Version:                outputpayload.ReturnSideChainDepositVersion,
				GenesisBlockAddress:    fixture.bankAddress,
				DepositTransactionHash: fixture.depositTxHash,
			},
		}},
		0,
		[]*program.Program{witness},
	)
	txn = CreateTransactionByType(txn, s.Chain)
	parameters := &TransactionParameters{
		Transaction: txn,
		BlockHeight: blockHeight,
		Config:      &chainParams,
		BlockChain:  s.Chain,
	}
	txn.SetParameters(parameters)
	s.signCrossChainProgram(txn, txn.Programs()[0])

	cleanup := s.prepareArbiterCrossChainContext(&chainParams)
	defer cleanup()

	s.NoError(txn.SanityCheck(parameters))
	_, err := txn.ContextCheck(parameters)
	s.NoError(err)
}

func (s *txValidatorSpecialTxTestSuite) crossChainArbiterScript(signers, publicKeyCount int) []byte {
	publicKeys := make([]*crypto.PublicKey, 0, publicKeyCount)
	for _, arbiter := range s.arbitrators.GetCrossChainArbiters()[:publicKeyCount] {
		publicKey, err := crypto.DecodePoint(arbiter.NodePublicKey)
		s.Require().NoError(err)
		publicKeys = append(publicKeys, publicKey)
	}

	code, err := contract.CreateMultiSigRedeemScript(signers, publicKeys)
	s.Require().NoError(err)
	code[len(code)-1] = common.CROSSCHAIN

	return code
}

func (s *txValidatorSpecialTxTestSuite) signCrossChainProgram(txn interfaces.Transaction,
	witness *program.Program) {
	unsignedTransaction := new(bytes.Buffer)
	txn.SerializeUnsigned(unsignedTransaction)
	for index := 0; index < s.arbitrators.MajorityCount; index++ {
		signature, err := crypto.Sign(s.arbitratorsPriKeys[index], unsignedTransaction.Bytes())
		s.Require().NoError(err)
		witness.Parameter, err = crypto.AppendSignature(index, signature,
			unsignedTransaction.Bytes(), witness.Code, witness.Parameter)
		s.Require().NoError(err)
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
