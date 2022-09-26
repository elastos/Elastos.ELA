// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package unit

import (
	"bytes"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	mrand "math/rand"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	elaact "github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
	"github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/utils"
	"github.com/elastos/Elastos.ELA/utils/test"

	"github.com/stretchr/testify/suite"
)

type txValidatorTestSuite struct {
	suite.Suite

	ELA               int64
	foundationAddress common.Uint168
	HeightVersion1    uint32
	CurrentHeight     uint32
	Chain             *blockchain.BlockChain
	OriginalLedger    *blockchain.Ledger
}

func init() {
	testing.Init()

	functions.GetTransactionByTxType = transaction.GetTransaction
	functions.GetTransactionByBytes = transaction.GetTransactionByBytes
	functions.CreateTransaction = transaction.CreateTransaction
	functions.GetTransactionParameters = transaction.GetTransactionparameters
	config.DefaultParams = config.GetDefaultParams()
}

func (s *txValidatorTestSuite) SetupSuite() {
	log.NewDefault(test.NodeLogPath, 0, 0, 0)

	params := &config.DefaultParams
	params.DPoSV2StartHeight = 0
	blockchain.FoundationAddress = params.Foundation
	s.foundationAddress = params.Foundation

	chainStore, err := blockchain.NewChainStore(filepath.Join(test.DataPath, "txvalidator"), params)
	if err != nil {
		s.Error(err)
	}
	s.Chain, err = blockchain.New(chainStore, params,
		state.NewState(params, nil, nil, nil,
			func() bool { return false },
			nil, nil,
			nil, nil, nil, nil, nil),
		crstate.NewCommittee(params))
	if err != nil {
		s.Error(err)
	}
	s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
		GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
		GetUTXO:                          chainStore.GetFFLDB().GetUTXO,
		GetHeight:                        func() uint32 { return s.CurrentHeight },
		CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
	})

	if err := s.Chain.Init(nil); err != nil {
		s.Error(err)
	}
	s.OriginalLedger = blockchain.DefaultLedger

	arbiters, err := state.NewArbitrators(params,
		nil, nil, nil,
		nil, nil, nil, nil, nil)
	if err != nil {
		s.Fail("initialize arbitrator failed")
	}
	arbiters.RegisterFunction(chainStore.GetHeight,
		func() *common.Uint256 { return &common.Uint256{} },
		func(height uint32) (*types.Block, error) {
			return nil, nil
		}, nil)
	blockchain.DefaultLedger = &blockchain.Ledger{Arbitrators: arbiters}
}

func (s *txValidatorTestSuite) TearDownSuite() {
	s.Chain.GetDB().Close()
	blockchain.DefaultLedger = s.OriginalLedger
}

func (s *txValidatorTestSuite) TestCheckTxHeightVersion() {
	// set blockHeight1 less than CRVotingStartHeight and set blockHeight2
	// to CRVotingStartHeight.
	blockHeight1 := s.Chain.GetParams().CRVotingStartHeight - 1
	blockHeight2 := s.Chain.GetParams().CRVotingStartHeight
	blockHeight3 := s.Chain.GetParams().RegisterCRByDIDHeight
	blockHeight4 := s.Chain.GetParams().DPoSV2StartHeight

	stake, _ := functions.GetTransactionByTxType(common2.ExchangeVotes)
	stake = CreateTransactionByType(stake, s.Chain)
	stake.SetParameters(&transaction.TransactionParameters{
		Transaction: stake,
		BlockHeight: blockHeight1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err := stake.HeightVersionCheck()
	s.EqualError(err, "not support ExchangeVotes transaction before DPoSV2StartHeight")
	stake.SetParameters(&transaction.TransactionParameters{
		Transaction: stake,
		BlockHeight: blockHeight4,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = stake.HeightVersionCheck()
	s.NoError(err)

	returnVotes, _ := functions.GetTransactionByTxType(common2.ReturnVotes)
	returnVotes = CreateTransactionByType(returnVotes, s.Chain)
	returnVotes.SetParameters(&transaction.TransactionParameters{
		Transaction: returnVotes,
		BlockHeight: blockHeight1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = returnVotes.HeightVersionCheck()
	s.EqualError(err, "not support ReturnVotes transaction before DPoSV2StartHeight")
	returnVotes.SetParameters(&transaction.TransactionParameters{
		Transaction: returnVotes,
		BlockHeight: blockHeight4,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = returnVotes.HeightVersionCheck()
	s.NoError(err)

	voting, _ := functions.GetTransactionByTxType(common2.Voting)
	voting = CreateTransactionByType(voting, s.Chain)
	voting.SetParameters(&transaction.TransactionParameters{
		Transaction: voting,
		BlockHeight: blockHeight1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = voting.HeightVersionCheck()
	s.EqualError(err, "not support Voting transaction before DPoSV2StartHeight")
	voting.SetParameters(&transaction.TransactionParameters{
		Transaction: voting,
		BlockHeight: blockHeight4,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = voting.HeightVersionCheck()
	s.NoError(err)

	dposV2ClaimReward, _ := functions.GetTransactionByTxType(common2.DposV2ClaimReward)
	dposV2ClaimReward = CreateTransactionByType(dposV2ClaimReward, s.Chain)
	dposV2ClaimReward.SetParameters(&transaction.TransactionParameters{
		Transaction: dposV2ClaimReward,
		BlockHeight: blockHeight1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = dposV2ClaimReward.HeightVersionCheck()
	s.EqualError(err, "not support DposV2ClaimReward transaction before DPoSV2StartHeight")
	dposV2ClaimReward.SetParameters(&transaction.TransactionParameters{
		Transaction: dposV2ClaimReward,
		BlockHeight: blockHeight4,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = dposV2ClaimReward.HeightVersionCheck()
	s.NoError(err)

	// check height version of registerCR transaction.
	registerCR, _ := functions.GetTransactionByTxType(common2.RegisterCR)
	registerCR = CreateTransactionByType(registerCR, s.Chain)
	registerCR.SetParameters(&transaction.TransactionParameters{
		Transaction: registerCR,
		BlockHeight: blockHeight1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = registerCR.HeightVersionCheck()
	s.EqualError(err, "not support RegisterCR transaction before CRVotingStartHeight")
	registerCR.SetParameters(&transaction.TransactionParameters{
		Transaction: registerCR,
		BlockHeight: blockHeight2,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = registerCR.HeightVersionCheck()
	s.NoError(err)

	registerCR2 := functions.CreateTransaction(
		0,
		common2.RegisterCR,
		payload.CRInfoDIDVersion,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	registerCR2 = CreateTransactionByType(registerCR2, s.Chain)
	registerCR2.SetParameters(&transaction.TransactionParameters{
		Transaction: registerCR2,
		BlockHeight: blockHeight1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = registerCR2.HeightVersionCheck()
	s.EqualError(err, "not support RegisterCR transaction before CRVotingStartHeight")
	registerCR2.SetParameters(&transaction.TransactionParameters{
		Transaction: registerCR2,
		BlockHeight: blockHeight3,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = registerCR2.HeightVersionCheck()
	s.NoError(err)

	// check height version of unregister transaction.
	unregisterCR := functions.CreateTransaction(
		0,
		common2.UnregisterCR,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	unregisterCR = CreateTransactionByType(unregisterCR, s.Chain)
	unregisterCR.SetParameters(&transaction.TransactionParameters{
		Transaction: unregisterCR,
		BlockHeight: blockHeight1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = unregisterCR.HeightVersionCheck()
	s.EqualError(err, "not support UnregisterCR transaction before CRVotingStartHeight")
	unregisterCR.SetParameters(&transaction.TransactionParameters{
		Transaction: unregisterCR,
		BlockHeight: blockHeight2,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = unregisterCR.HeightVersionCheck()
	s.NoError(err)

	// check height version of unregister transaction.
	returnCoin := functions.CreateTransaction(
		0,
		common2.ReturnCRDepositCoin,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	returnCoin = CreateTransactionByType(returnCoin, s.Chain)
	returnCoin.SetParameters(&transaction.TransactionParameters{
		Transaction: returnCoin,
		BlockHeight: blockHeight1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = returnCoin.HeightVersionCheck()
	s.EqualError(err, "not support ReturnCRDepositCoin transaction before CRVotingStartHeight")
	returnCoin.SetParameters(&transaction.TransactionParameters{
		Transaction: returnCoin,
		BlockHeight: blockHeight2,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = returnCoin.HeightVersionCheck()
	s.NoError(err)

	// check height version of vote CR.
	voteCR := functions.CreateTransaction(
		0x09,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     common.Uint256{},
				Value:       0,
				OutputLock:  0,
				ProgramHash: common.Uint168{},
				Type:        common2.OTVote,
				Payload: &outputpayload.VoteOutput{
					Version: outputpayload.VoteProducerAndCRVersion,
				},
			},
		},
		0,
		[]*program.Program{},
	)
	voteCR = CreateTransactionByType(voteCR, s.Chain)
	voteCR.SetParameters(&transaction.TransactionParameters{
		Transaction: voteCR,
		BlockHeight: blockHeight1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = voteCR.HeightVersionCheck()
	s.EqualError(err, "not support VoteProducerAndCRVersion "+
		"before CRVotingStartHeight")
	voteCR.SetParameters(&transaction.TransactionParameters{
		Transaction: voteCR,
		BlockHeight: blockHeight2,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = voteCR.HeightVersionCheck()
	s.NoError(err)
}

func (s *txValidatorTestSuite) TestCheckTransactionSize() {
	tx := buildTx()
	buf := new(bytes.Buffer)
	err := tx.Serialize(buf)
	if !s.NoError(err) {
		return
	}

	// normal
	err = blockchain.CheckTransactionSize(tx)
	s.NoError(err, "[CheckTransactionSize] passed normal size")
}

func (s *txValidatorTestSuite) TestCheckTransactionInput() {
	// coinbase transaction
	tx := newCoinBaseTransaction(new(payload.CoinBase), 0)
	err := blockchain.CheckTransactionInput(tx)
	s.NoError(err)

	// invalid coinbase refer index
	tx.Inputs()[0].Previous.Index = 0
	err = blockchain.CheckTransactionInput(tx)
	s.EqualError(err, "invalid coinbase input")

	// invalid coinbase refer id
	tx.Inputs()[0].Previous.Index = math.MaxUint16
	rand.Read(tx.Inputs()[0].Previous.TxID[:])
	err = blockchain.CheckTransactionInput(tx)
	s.EqualError(err, "invalid coinbase input")

	// multiple coinbase inputs
	tx.SetInputs(append(tx.Inputs(), &common2.Input{}))
	err = blockchain.CheckTransactionInput(tx)
	s.EqualError(err, "coinbase must has only one input")

	// normal transaction
	tx = buildTx()
	err = blockchain.CheckTransactionInput(tx)
	s.NoError(err)

	// no inputs
	tx.SetInputs(nil)
	err = blockchain.CheckTransactionInput(tx)
	s.EqualError(err, "transaction has no inputs")

	// normal transaction with coinbase input
	tx.SetInputs(append(tx.Inputs(), &common2.Input{Previous: *common2.NewOutPoint(common.EmptyHash, math.MaxUint16)}))
	err = blockchain.CheckTransactionInput(tx)
	s.EqualError(err, "invalid transaction input")

	// duplicated inputs
	tx = buildTx()
	tx.SetInputs(append(tx.Inputs(), tx.Inputs()[0]))
	err = blockchain.CheckTransactionInput(tx)
	s.EqualError(err, "duplicated transaction inputs")
}

func (s *txValidatorTestSuite) TestCheckTransactionOutput() {
	// coinbase
	tx := newCoinBaseTransaction(new(payload.CoinBase), 0)
	tx.SetOutputs([]*common2.Output{
		{AssetID: config.ELAAssetID, ProgramHash: s.foundationAddress},
		{AssetID: config.ELAAssetID, ProgramHash: s.foundationAddress},
	})
	err := s.Chain.CheckTransactionOutput(tx, s.HeightVersion1)
	s.NoError(err)

	// outputs < 2
	tx.SetOutputs([]*common2.Output{
		{AssetID: config.ELAAssetID, ProgramHash: s.foundationAddress},
	})
	err = s.Chain.CheckTransactionOutput(tx, s.HeightVersion1)
	s.EqualError(err, "coinbase output is not enough, at least 2")

	// invalid asset id
	tx.SetOutputs([]*common2.Output{
		{AssetID: common.EmptyHash, ProgramHash: s.foundationAddress},
		{AssetID: common.EmptyHash, ProgramHash: s.foundationAddress},
	})
	err = s.Chain.CheckTransactionOutput(tx, s.HeightVersion1)
	s.EqualError(err, "asset ID in coinbase is invalid")

	// reward to foundation in coinbase = 30% (CheckTxOut version)
	totalReward := config.DefaultParams.RewardPerBlock
	fmt.Printf("Block reward amount %s", totalReward.String())
	foundationReward := common.Fixed64(float64(totalReward) * 0.3)
	fmt.Printf("Foundation reward amount %s", foundationReward.String())
	tx.SetOutputs([]*common2.Output{
		{AssetID: config.ELAAssetID, ProgramHash: s.foundationAddress, Value: foundationReward},
		{AssetID: config.ELAAssetID, ProgramHash: common.Uint168{}, Value: totalReward - foundationReward},
	})
	err = s.Chain.CheckTransactionOutput(tx, s.HeightVersion1)
	s.NoError(err)

	// reward to foundation in coinbase < 30% (CheckTxOut version)
	foundationReward = common.Fixed64(float64(totalReward) * 0.299999)
	fmt.Printf("Foundation reward amount %s", foundationReward.String())
	tx.SetOutputs([]*common2.Output{
		{AssetID: config.ELAAssetID, ProgramHash: s.foundationAddress, Value: foundationReward},
		{AssetID: config.ELAAssetID, ProgramHash: common.Uint168{}, Value: totalReward - foundationReward},
	})
	err = s.Chain.CheckTransactionOutput(tx, s.HeightVersion1)
	s.EqualError(err, "reward to foundation in coinbase < 30%")

	// normal transaction
	tx = buildTx()
	for _, output := range tx.Outputs() {
		output.AssetID = config.ELAAssetID
		output.ProgramHash = common.Uint168{}
	}
	err = s.Chain.CheckTransactionOutput(tx, s.HeightVersion1)
	s.NoError(err)

	// outputs < 1
	tx.SetOutputs(nil)
	err = s.Chain.CheckTransactionOutput(tx, s.HeightVersion1)
	s.EqualError(err, "transaction has no outputs")

	// invalid asset ID
	tx.SetOutputs(randomOutputs())
	for _, output := range tx.Outputs() {
		output.AssetID = common.EmptyHash
		output.ProgramHash = common.Uint168{}
	}
	err = s.Chain.CheckTransactionOutput(tx, s.HeightVersion1)
	s.EqualError(err, "asset ID in output is invalid")

	// should only have one special output
	tx.SetVersion(common2.TxVersion09)
	tx.SetOutputs([]*common2.Output{})
	address := common.Uint168{}
	address[0] = byte(contract.PrefixStandard)
	appendSpecial := func() []*common2.Output {
		return append(tx.Outputs(), &common2.Output{
			Type:        common2.OTVote,
			AssetID:     config.ELAAssetID,
			ProgramHash: address,
			Value:       common.Fixed64(mrand.Int63()),
			OutputLock:  mrand.Uint32(),
			Payload: &outputpayload.VoteOutput{
				Contents: []outputpayload.VoteContent{},
			},
		})
	}
	tx.SetOutputs(appendSpecial())
	s.NoError(s.Chain.CheckTransactionOutput(tx, s.HeightVersion1))
	tx.SetOutputs(appendSpecial()) // add another special output here
	originHeight := config.DefaultParams.PublicDPOSHeight
	config.DefaultParams.PublicDPOSHeight = 0
	err = s.Chain.CheckTransactionOutput(tx, s.HeightVersion1)
	config.DefaultParams.PublicDPOSHeight = originHeight
	s.EqualError(err, "special output count should less equal than 1")

	// invalid program hash
	tx.SetVersion(common2.TxVersionDefault)
	tx.SetOutputs(randomOutputs())
	for _, output := range tx.Outputs() {
		output.AssetID = config.ELAAssetID
		address := common.Uint168{}
		address[0] = 0x23
		output.ProgramHash = address
	}
	config.DefaultParams.PublicDPOSHeight = 0
	s.NoError(s.Chain.CheckTransactionOutput(tx, s.HeightVersion1))
	config.DefaultParams.PublicDPOSHeight = originHeight

	// new sideChainPow
	tx = functions.CreateTransaction(
		0,
		common2.SideChainPow,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				Value: 0,
				Type:  0,
			},
		},
		0,
		[]*program.Program{},
	)

	s.NoError(s.Chain.CheckTransactionOutput(tx, s.HeightVersion1))

	tx.SetOutputs([]*common2.Output{
		{
			Value: 0,
			Type:  0,
		},
		{
			Value: 0,
			Type:  0,
		},
	})
	err = s.Chain.CheckTransactionOutput(tx, s.HeightVersion1)
	s.EqualError(err, "new sideChainPow tx must have only one output")

	tx.SetOutputs([]*common2.Output{
		{
			Value: 100,
			Type:  0,
		},
	})
	err = s.Chain.CheckTransactionOutput(tx, s.HeightVersion1)
	s.EqualError(err, "the value of new sideChainPow tx output must be 0")

	tx.SetOutputs([]*common2.Output{
		{
			Value: 0,
			Type:  1,
		},
	})
	err = s.Chain.CheckTransactionOutput(tx, s.HeightVersion1)
	s.EqualError(err, "the type of new sideChainPow tx output must be OTNone")
}

func (s *txValidatorTestSuite) TestCheckAmountPrecision() {
	// precision check
	for i := 8; i >= 0; i-- {
		amount := common.Fixed64(math.Pow(10, float64(i)))
		fmt.Printf("Amount %s", amount.String())
		s.Equal(true, blockchain.CheckAmountPrecise(amount, byte(8-i)))
		s.Equal(false, blockchain.CheckAmountPrecise(amount, byte(8-i-1)))
	}
}

func (s *txValidatorTestSuite) TestCheckAttributeProgram() {
	// valid attributes
	tx := buildTx()
	usages := []common2.AttributeUsage{
		common2.Nonce,
		common2.Script,
		common2.Description,
		common2.DescriptionUrl,
		common2.Memo,
	}
	for _, usage := range usages {
		attr := common2.NewAttribute(usage, nil)
		tx.SetAttributes(append(tx.Attributes(), &attr))
	}
	tx = CreateTransactionByType(tx, s.Chain)
	tx.SetParameters(&transaction.TransactionParameters{
		Transaction: tx,
		BlockHeight: 0,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err := tx.CheckAttributeProgram()
	s.EqualError(err, "no programs found in transaction")

	// invalid attributes
	getInvalidUsage := func() common2.AttributeUsage {
		var usage = make([]byte, 1)
	NEXT:
		rand.Read(usage)
		if common2.IsValidAttributeType(common2.AttributeUsage(usage[0])) {
			goto NEXT
		}
		return common2.AttributeUsage(usage[0])
	}
	for i := 0; i < 10; i++ {
		attr := common2.NewAttribute(getInvalidUsage(), nil)
		tx.SetAttributes([]*common2.Attribute{&attr})
		err := tx.CheckAttributeProgram()
		s.EqualError(err, fmt.Sprintf("invalid attribute usage %v", attr.Usage))
	}
	tx.SetAttributes(nil)

	// empty programs
	tx.SetPrograms([]*program.Program{})
	err = tx.CheckAttributeProgram()
	s.EqualError(err, "no programs found in transaction")

	// nil program code
	p := &program.Program{}
	tx.SetPrograms(append(tx.Programs(), p))
	err = tx.CheckAttributeProgram()
	s.EqualError(err, "invalid program code nil")

	// nil program parameter
	var code = make([]byte, 21)
	rand.Read(code)
	p = &program.Program{Code: code}
	tx.SetPrograms([]*program.Program{p})
	err = tx.CheckAttributeProgram()
	s.EqualError(err, "invalid program parameter nil")
}

func (s *txValidatorTestSuite) TestCheckTransactionPayload() {
	// normal
	pd := &payload.RegisterAsset{
		Asset: payload.Asset{
			Name:      "ELA",
			Precision: 0x08,
			AssetType: payload.Token,
		},
		Amount: 3300 * 10000 * 10000000,
	}
	tx := functions.CreateTransaction(
		0,
		common2.RegisterAsset,
		0,
		pd,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	tx = CreateTransactionByType(tx, s.Chain)
	err := tx.CheckTransactionPayload()
	s.NoError(err)

	// invalid precision
	pd.Asset.Precision = 9
	tx = CreateTransactionByType(tx, s.Chain)
	err = tx.CheckTransactionPayload()
	s.EqualError(err, "invalid asset precision")

	// invalid amount
	pd.Asset.Precision = 0
	pd.Amount = 1234567
	tx = CreateTransactionByType(tx, s.Chain)
	err = tx.CheckTransactionPayload()
	s.EqualError(err, "invalid asset value, out of precise")
}

func (s *txValidatorTestSuite) TestCheckDuplicateSidechainTx() {
	hashStr1 := "8a6cb4b5ff1a4f8368c6513a536c663381e3fdeff738e9b437bd8fce3fb30b62"
	hashBytes1, _ := common.HexStringToBytes(hashStr1)
	hash1, _ := common.Uint256FromBytes(hashBytes1)
	hashStr2 := "cc62e14f5f9526b7f4ff9d34dcd0643dacb7886707c57f49ec97b95ec5c4edac"
	hashBytes2, _ := common.HexStringToBytes(hashStr2)
	hash2, _ := common.Uint256FromBytes(hashBytes2)

	// 1. Generate the ill withdraw transaction which have duplicate sidechain tx
	pd := &payload.WithdrawFromSideChain{
		BlockHeight:         100,
		GenesisBlockAddress: "eb7adb1fea0dd6185b09a43bdcd4924bb22bff7151f0b1b4e08699840ab1384b",
		SideChainTransactionHashes: []common.Uint256{
			*hash1,
			*hash2,
			*hash1, // duplicate tx hash
		},
	}

	txn := functions.CreateTransaction(
		0,
		common2.WithdrawFromSideChain,
		0,
		pd,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	// 2. Run CheckDuplicateSidechainTx
	err := blockchain.CheckDuplicateSidechainTx(txn)
	s.EqualError(err, "Duplicate sidechain tx detected in a transaction")
}

func (s *txValidatorTestSuite) TestCheckTransactionBalance() {
	// WithdrawFromSideChain will pass check in any condition
	tx := functions.CreateTransaction(
		0,
		common2.WithdrawFromSideChain,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	// single output

	outputValue1 := common.Fixed64(100 * s.ELA)
	deposit := newCoinBaseTransaction(new(payload.CoinBase), 0)
	deposit.SetOutputs([]*common2.Output{
		{AssetID: config.ELAAssetID, ProgramHash: s.foundationAddress, Value: outputValue1},
	})

	references := map[*common2.Input]common2.Output{
		&common2.Input{}: {
			Value: outputValue1,
		},
	}
	s.EqualError(s.Chain.CheckTransactionFee(tx, references), "transaction fee not enough")

	references = map[*common2.Input]common2.Output{
		&common2.Input{}: {
			Value: outputValue1 + s.Chain.GetParams().MinTransactionFee,
		},
	}
	s.NoError(s.Chain.CheckTransactionFee(tx, references))

	// multiple output

	outputValue1 = common.Fixed64(30 * s.ELA)
	outputValue2 := common.Fixed64(70 * s.ELA)
	tx.SetOutputs([]*common2.Output{
		{AssetID: config.ELAAssetID, ProgramHash: s.foundationAddress, Value: outputValue1},
		{AssetID: config.ELAAssetID, ProgramHash: common.Uint168{}, Value: outputValue2},
	})

	references = map[*common2.Input]common2.Output{
		&common2.Input{}: {
			Value: outputValue1 + outputValue2,
		},
	}
	s.EqualError(s.Chain.CheckTransactionFee(tx, references), "transaction fee not enough")

	references = map[*common2.Input]common2.Output{
		&common2.Input{}: {
			Value: outputValue1 + outputValue2 + s.Chain.GetParams().MinTransactionFee,
		},
	}
	s.NoError(s.Chain.CheckTransactionFee(tx, references))
}

func (s *txValidatorTestSuite) TestCheckSideChainPowConsensus() {
	// 1. Generate a side chain pow transaction
	pd := &payload.SideChainPow{
		SideBlockHash:   common.Uint256{1, 1, 1},
		SideGenesisHash: common.Uint256{2, 2, 2},
		BlockHeight:     uint32(10),
	}
	txn := functions.CreateTransaction(
		0,
		common2.SideChainPow,
		0,
		pd,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	//2. Get arbitrator
	password1 := "1234"
	privateKey1, _ := common.HexStringToBytes(password1)
	publicKey := new(crypto.PublicKey)
	publicKey.X, publicKey.Y = elliptic.P256().ScalarBaseMult(privateKey1)
	arbitrator1, _ := publicKey.EncodePoint(true)

	password2 := "5678"
	privateKey2, _ := common.HexStringToBytes(password2)
	publicKey2 := new(crypto.PublicKey)
	publicKey2.X, publicKey2.Y = elliptic.P256().ScalarBaseMult(privateKey2)
	arbitrator2, _ := publicKey2.EncodePoint(true)

	//3. Sign transaction by arbitrator1
	buf := new(bytes.Buffer)
	txn.Payload().Serialize(buf, payload.SideChainPowVersion)
	signature, _ := crypto.Sign(privateKey1, buf.Bytes()[0:68])
	txn.Payload().(*payload.SideChainPow).Signature = signature

	//4. Run CheckSideChainPowConsensus
	s.NoError(blockchain.CheckSideChainPowConsensus(txn, arbitrator1), "TestCheckSideChainPowConsensus failed.")

	s.Error(blockchain.CheckSideChainPowConsensus(txn, arbitrator2), "TestCheckSideChainPowConsensus failed.")
}

func (s *txValidatorTestSuite) TestCheckDestructionAddress() {
	destructionAddress := "ELANULLXXXXXXXXXXXXXXXXXXXXXYvs3rr"
	txID, _ := common.Uint256FromHexString("7e8863a503e90e6464529feb1c25d98c903e01bec00ccfea2475db4e37d7328b")
	programHash, _ := common.Uint168FromAddress(destructionAddress)
	reference := map[*common2.Input]common2.Output{
		&common2.Input{Previous: common2.OutPoint{*txID, 1234}, Sequence: 123456}: {
			ProgramHash: *programHash,
		},
	}

	err := blockchain.CheckDestructionAddress(reference)
	s.EqualError(err, fmt.Sprintf("cannot use utxo from the destruction address"))
}

func (s *txValidatorTestSuite) TestCheckRegisterProducerTransaction() {
	// Generate a register producer transaction
	publicKeyStr1 := "02ca89a5fe6213da1b51046733529a84f0265abac59005f6c16f62330d20f02aeb"
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	privateKeyStr1 := "7a50d2b036d64fcb3d344cee429f61c4a3285a934c45582b26e8c9227bc1f33a"
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)
	publicKeyStr2 := "027c4f35081821da858f5c7197bac5e33e77e5af4a3551285f8a8da0a59bd37c45"
	publicKey2, _ := common.HexStringToBytes(publicKeyStr2)
	errPublicKeyStr := "02b611f07341d5ddce51b5c4366aca7b889cfe0993bd63fd4"
	errPublicKey, _ := common.HexStringToBytes(errPublicKeyStr)

	rpPayload := &payload.ProducerInfo{
		OwnerPublicKey: publicKey1,
		NodePublicKey:  publicKey1,
		NickName:       "nickname 1",
		Url:            "http://www.elastos_test.com",
		Location:       1,
		NetAddress:     "127.0.0.1:20338",
	}
	rpSignBuf := new(bytes.Buffer)
	err := rpPayload.SerializeUnsigned(rpSignBuf, payload.ProducerInfoVersion)
	s.NoError(err)
	rpSig, err := crypto.Sign(privateKey1, rpSignBuf.Bytes())
	s.NoError(err)
	rpPayload.Signature = rpSig

	txn := functions.CreateTransaction(
		0,
		common2.RegisterProducer,
		0,
		rpPayload,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{{
			Code:      getCodeByPubKeyStr(publicKeyStr1),
			Parameter: nil,
		}},
	)

	publicKeyDeposit1, _ := contract.PublicKeyToDepositProgramHash(publicKey1)
	txn.SetOutputs([]*common2.Output{{
		AssetID:     common.Uint256{},
		Value:       5000 * 100000000,
		OutputLock:  0,
		ProgramHash: *publicKeyDeposit1,
	}})
	txn = CreateTransactionByType(txn, s.Chain)
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	// Give an invalid owner public key in payload
	txn.Payload().(*payload.ProducerInfo).OwnerPublicKey = errPublicKey
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid owner public key in payload")

	// check node public when block height is higher than h2
	originHeight := config.DefaultParams.PublicDPOSHeight
	txn.Payload().(*payload.ProducerInfo).NodePublicKey = errPublicKey
	config.DefaultParams.PublicDPOSHeight = 0
	err, _ = txn.SpecialContextCheck()
	config.DefaultParams.PublicDPOSHeight = originHeight
	s.EqualError(err, "transaction validate error: payload content invalid:invalid node public key in payload")

	// check node public key same with CRC
	txn.Payload().(*payload.ProducerInfo).OwnerPublicKey = publicKey2
	pk, _ := common.HexStringToBytes(config.DefaultParams.CRCArbiters[0])
	txn.Payload().(*payload.ProducerInfo).NodePublicKey = pk
	config.DefaultParams.PublicDPOSHeight = 0
	err, _ = txn.SpecialContextCheck()
	config.DefaultParams.PublicDPOSHeight = originHeight
	s.EqualError(err, "transaction validate error: payload content invalid:node public key can't equal with CRC")

	// check owner public key same with CRC
	txn.Payload().(*payload.ProducerInfo).NodePublicKey = publicKey2
	pk, _ = common.HexStringToBytes(config.DefaultParams.CRCArbiters[0])
	txn.Payload().(*payload.ProducerInfo).OwnerPublicKey = pk
	config.DefaultParams.PublicDPOSHeight = 0
	err, _ = txn.SpecialContextCheck()
	config.DefaultParams.PublicDPOSHeight = originHeight
	s.EqualError(err, "transaction validate error: payload content invalid:owner public key can't equal with CRC")

	// Invalidates the signature in payload
	txn.Payload().(*payload.ProducerInfo).OwnerPublicKey = publicKey2
	txn.Payload().(*payload.ProducerInfo).NodePublicKey = publicKey2
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid signature in payload")

	// Give a mismatching deposit address
	rpPayload.OwnerPublicKey = publicKey1
	rpPayload.Url = "www.test.com"
	rpSignBuf = new(bytes.Buffer)
	err = rpPayload.SerializeUnsigned(rpSignBuf, payload.ProducerInfoVersion)
	s.NoError(err)
	rpSig, err = crypto.Sign(privateKey1, rpSignBuf.Bytes())
	s.NoError(err)
	rpPayload.Signature = rpSig
	txn.SetPayload(rpPayload)

	publicKeyDeposit2, _ := contract.PublicKeyToDepositProgramHash(publicKey2)
	txn.SetOutputs([]*common2.Output{{
		AssetID:     common.Uint256{},
		Value:       5000 * 100000000,
		OutputLock:  0,
		ProgramHash: *publicKeyDeposit2,
	}})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:deposit address does not match the public key in payload")

	// Give a insufficient deposit coin
	txn.SetOutputs([]*common2.Output{{
		AssetID:     common.Uint256{},
		Value:       4000,
		OutputLock:  0,
		ProgramHash: *publicKeyDeposit1,
	}})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:producer deposit amount is insufficient")

	// Multi deposit addresses
	txn.SetOutputs([]*common2.Output{
		{
			AssetID:     common.Uint256{},
			Value:       5000 * 100000000,
			OutputLock:  0,
			ProgramHash: *publicKeyDeposit1,
		},
		{
			AssetID:     common.Uint256{},
			Value:       5000 * 100000000,
			OutputLock:  0,
			ProgramHash: *publicKeyDeposit1,
		}})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:there must be only one deposit address in outputs")
}

func (s *txValidatorTestSuite) TestCheckRegisterDposV2ProducerTransaction() {
	publicKeyStr1 := "02ca89a5fe6213da1b51046733529a84f0265abac59005f6c16f62330d20f02aeb"
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	privateKeyStr1 := "7a50d2b036d64fcb3d344cee429f61c4a3285a934c45582b26e8c9227bc1f33a"
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)
	publicKeyStr2 := "027c4f35081821da858f5c7197bac5e33e77e5af4a3551285f8a8da0a59bd37c45"
	publicKey2, _ := common.HexStringToBytes(publicKeyStr2)
	errPublicKeyStr := "02b611f07341d5ddce51b5c4366aca7b889cfe0993bd63fd4"
	errPublicKey, _ := common.HexStringToBytes(errPublicKeyStr)

	rpPayload := &payload.ProducerInfo{
		OwnerPublicKey: publicKey1,
		NodePublicKey:  publicKey1,
		NickName:       "nickname 1",
		Url:            "http://www.elastos_test.com",
		Location:       1,
		NetAddress:     "127.0.0.1:20338",
		StakeUntil:     100000,
	}
	rpSignBuf := new(bytes.Buffer)
	err := rpPayload.SerializeUnsigned(rpSignBuf, payload.ProducerInfoDposV2Version)
	s.NoError(err)
	rpSig, err := crypto.Sign(privateKey1, rpSignBuf.Bytes())
	s.NoError(err)
	rpPayload.Signature = rpSig

	txn := functions.CreateTransaction(
		0,
		common2.RegisterProducer,
		1,
		rpPayload,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{{
			Code:      getCodeByPubKeyStr(publicKeyStr1),
			Parameter: nil,
		}},
	)

	publicKeyDeposit1, _ := contract.PublicKeyToDepositProgramHash(publicKey1)
	txn.SetOutputs([]*common2.Output{{
		AssetID:     common.Uint256{},
		Value:       5000 * 100000000,
		OutputLock:  0,
		ProgramHash: *publicKeyDeposit1,
	}})
	tx := txn.(*transaction.RegisterProducerTransaction)
	param := s.Chain.GetParams()
	param.DPoSV2StartHeight = 10
	param.PublicDPOSHeight = 5
	s.Chain.Nodes = []*blockchain.BlockNode{
		{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {},
	}
	tx.DefaultChecker.SetParameters(&transaction.TransactionParameters{
		BlockChain: s.Chain,
		Config:     s.Chain.GetParams(),
	})

	err, _ = tx.SpecialContextCheck()
	s.NoError(err)

	// Give an invalid owner public key in payload
	txn.Payload().(*payload.ProducerInfo).OwnerPublicKey = errPublicKey
	err, _ = tx.SpecialContextCheck()
	s.EqualError(err.(errors.ELAError).InnerError(), "invalid owner public key in payload")

	// check version when height is not higher than dposv2 height
	s.Chain.Nodes = []*blockchain.BlockNode{
		{}, {}, {}, {},
	}
	param.PublicDPOSHeight = 1
	txn.Payload().(*payload.ProducerInfo).OwnerPublicKey = publicKey1
	err, _ = tx.SpecialContextCheck()
	s.EqualError(err.(errors.ELAError).InnerError(), "can not register dposv2 before dposv2 start height")

	// Invalidates public key in payload
	txn.Payload().(*payload.ProducerInfo).OwnerPublicKey = publicKey2
	txn.Payload().(*payload.ProducerInfo).NodePublicKey = publicKey2
	param.PublicDPOSHeight = 5
	s.Chain.Nodes = []*blockchain.BlockNode{
		{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {},
	}
	err, _ = tx.SpecialContextCheck()
	s.EqualError(err.(errors.ELAError).InnerError(), "invalid signature in payload")

	// Give a insufficient deposit coin
	txn.Payload().(*payload.ProducerInfo).OwnerPublicKey = publicKey1
	txn.Payload().(*payload.ProducerInfo).NodePublicKey = publicKey1
	txn.SetOutputs([]*common2.Output{{
		AssetID:     common.Uint256{},
		Value:       1000,
		OutputLock:  0,
		ProgramHash: *publicKeyDeposit1,
	}})
	err, _ = tx.SpecialContextCheck()
	s.EqualError(err.(errors.ELAError).InnerError(), "producer deposit amount is insufficient")

	// Multi deposit addresses
	txn.SetOutputs([]*common2.Output{
		{
			AssetID:     common.Uint256{},
			Value:       5000 * 100000000,
			OutputLock:  0,
			ProgramHash: *publicKeyDeposit1,
		},
		{
			AssetID:     common.Uint256{},
			Value:       5000 * 100000000,
			OutputLock:  0,
			ProgramHash: *publicKeyDeposit1,
		}})
	err, _ = tx.SpecialContextCheck()
	s.EqualError(err.(errors.ELAError).InnerError(), "there must be only one deposit address in outputs")
}

func (s *txValidatorTestSuite) TestCheckStakeTransaction() {
	publicKey := "03878cbe6abdafc702befd90e2329c4f37e7cb166410f0ecb70488c74c85b81d66"
	publicKeyBytes, _ := common.HexStringToBytes(publicKey)
	code := getCode(publicKeyBytes)
	c, _ := contract.CreateStakeContractByCode(code)
	stakeAddress_uint168 := c.ToProgramHash()
	rpPayload := &outputpayload.ExchangeVotesOutput{
		Version:      0,
		StakeAddress: *stakeAddress_uint168,
	}
	txn := functions.CreateTransaction(
		0,
		common2.ExchangeVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     common.Uint256{},
				Value:       100000000,
				OutputLock:  0,
				ProgramHash: *stakeAddress_uint168,
				Payload:     rpPayload,
			},
			{
				AssetID:     common.Uint256{},
				Value:       100000000,
				OutputLock:  0,
				ProgramHash: *stakeAddress_uint168,
				Payload:     rpPayload,
			},
			{
				AssetID:     common.Uint256{},
				Value:       100000000,
				OutputLock:  0,
				ProgramHash: *stakeAddress_uint168,
				Payload:     rpPayload,
			},
		},
		0,
		[]*program.Program{{
			Code:      nil,
			Parameter: nil,
		}},
	)
	err := txn.CheckTransactionOutput()
	s.EqualError(err, "output count should not be greater than 2")

	txn = functions.CreateTransaction(
		0,
		common2.ExchangeVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{{
			Code:      code,
			Parameter: nil,
		}},
	)
	err = txn.CheckTransactionOutput()
	s.EqualError(err, "transaction has no outputs")

	txn = functions.CreateTransaction(
		0,
		common2.ExchangeVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     common.Uint256{},
				Value:       100000000,
				OutputLock:  0,
				ProgramHash: *stakeAddress_uint168,
				Payload:     rpPayload,
			},
		},
		0,
		[]*program.Program{{
			Code:      nil,
			Parameter: nil,
		}},
	)
	err = txn.CheckTransactionOutput()
	s.EqualError(err, "asset ID in output is invalid")

	txn = functions.CreateTransaction(
		0,
		common2.ExchangeVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     config.ELAAssetID,
				Value:       -1,
				OutputLock:  0,
				ProgramHash: *stakeAddress_uint168,
				Payload:     rpPayload,
			},
		},
		0,
		[]*program.Program{{
			Code:      nil,
			Parameter: nil,
		}},
	)
	err = txn.CheckTransactionOutput()
	s.EqualError(err, "invalid transaction UTXO output")

	txn = functions.CreateTransaction(
		0,
		common2.ExchangeVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     config.ELAAssetID,
				Value:       100000000,
				OutputLock:  0,
				ProgramHash: *stakeAddress_uint168,
				Payload:     rpPayload,
			},
		},
		0,
		[]*program.Program{{
			Code:      code,
			Parameter: nil,
		}},
	)
	err = txn.CheckTransactionOutput()
	s.EqualError(err, "invalid output type")

	rpPayload.Version = 1
	txn = functions.CreateTransaction(
		0,
		common2.ExchangeVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     config.ELAAssetID,
				Value:       100000000,
				OutputLock:  0,
				ProgramHash: *stakeAddress_uint168,
				Payload:     rpPayload,
				Type:        common2.OTStake,
			},
		},
		0,
		[]*program.Program{{
			Code:      code,
			Parameter: nil,
		}},
	)
	err = txn.CheckTransactionOutput()
	s.EqualError(err, "invalid exchange vote version")

	rpPayload.Version = 0
	txn = functions.CreateTransaction(
		0,
		common2.ExchangeVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     config.ELAAssetID,
				Value:       100000000,
				OutputLock:  0,
				ProgramHash: *stakeAddress_uint168,
				Payload:     rpPayload,
				Type:        common2.OTStake,
			},
		},
		0,
		[]*program.Program{{
			Code:      code,
			Parameter: nil,
		}},
	)
	param := s.Chain.GetParams()
	param.StakePool = common.Uint168{0x1, 0x2, 0x3}
	tx := txn.(*transaction.ExchangeVotesTransaction)
	tx.DefaultChecker.SetParameters(&transaction.TransactionParameters{
		BlockChain: s.Chain,
		Config:     s.Chain.GetParams(),
	})
	err = txn.CheckTransactionOutput()
	s.EqualError(err, "first output address need to be stake address")

	txn = functions.CreateTransaction(
		0,
		common2.ExchangeVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     config.ELAAssetID,
				Value:       100000000,
				OutputLock:  0,
				ProgramHash: *stakeAddress_uint168,
				Payload:     rpPayload,
				Type:        common2.OTStake,
			},
		},
		0,
		[]*program.Program{{
			Code:      code,
			Parameter: nil,
		}},
	)
	param = s.Chain.GetParams()
	param.StakePool = *stakeAddress_uint168
	tx = txn.(*transaction.ExchangeVotesTransaction)
	tx.DefaultChecker.SetParameters(&transaction.TransactionParameters{
		BlockChain: s.Chain,
		Config:     s.Chain.GetParams(),
	})
	err = txn.CheckTransactionOutput()
	s.NoError(err)
}

func getCodeByPubKeyStr(publicKey string) []byte {
	pkBytes, _ := common.HexStringToBytes(publicKey)
	pk, _ := crypto.DecodePoint(pkBytes)
	redeemScript, _ := contract.CreateStandardRedeemScript(pk)
	return redeemScript
}
func getCodeHexStr(publicKey string) string {
	pkBytes, _ := common.HexStringToBytes(publicKey)
	pk, _ := crypto.DecodePoint(pkBytes)
	redeemScript, _ := contract.CreateStandardRedeemScript(pk)
	codeHexStr := common.BytesToHexString(redeemScript)
	return codeHexStr
}

func (s *txValidatorTestSuite) TestCheckDposV2VoteProducerOutput() {
	// 1. Generate a vote output v0
	publicKeyStr1 := "02b611f07341d5ddce51b5c4366aca7b889cfe0993bd63fd47e944507292ea08dd"
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	referKey := randomUint256()
	outputs1 := []*payload.Voting{
		{
			Contents: []payload.VotesContent{
				{
					VoteType: outputpayload.DposV2,
					VotesInfo: []payload.VotesWithLockTime{
						{
							Candidate: publicKey1,
							Votes:     10000,
							LockTime:  100000,
						},
					},
				},
			},
			RenewalContents: []payload.RenewalVotesContent{},
		},
		{
			Contents: []payload.VotesContent{
				{
					VoteType: outputpayload.DposV2,
					VotesInfo: []payload.VotesWithLockTime{
						{
							Candidate: publicKey1,
							Votes:     10000,
							LockTime:  100000,
						},
					},
				},
				{
					VoteType: outputpayload.DposV2,
					VotesInfo: []payload.VotesWithLockTime{
						{
							Candidate: publicKey1,
							Votes:     10000,
							LockTime:  100000,
						},
					},
				},
			},
			RenewalContents: []payload.RenewalVotesContent{},
		},
		{
			Contents: []payload.VotesContent{
				{
					VoteType: 0x05,
					VotesInfo: []payload.VotesWithLockTime{
						{
							Candidate: publicKey1,
							Votes:     10000,
							LockTime:  100000,
						},
					},
				},
			},
			RenewalContents: []payload.RenewalVotesContent{},
		},
		{
			Contents: []payload.VotesContent{
				{
					VoteType: outputpayload.DposV2,
					VotesInfo: []payload.VotesWithLockTime{
						{
							Candidate: publicKey1,
							Votes:     10000,
							LockTime:  100000,
						},
						{
							Candidate: publicKey1,
							Votes:     10000,
							LockTime:  100000,
						},
					},
				},
			},
			RenewalContents: []payload.RenewalVotesContent{},
		},
		{
			Contents: []payload.VotesContent{
				{
					VoteType: outputpayload.DposV2,
					VotesInfo: []payload.VotesWithLockTime{
						{
							Candidate: publicKey1,
							Votes:     -100,
							LockTime:  100000,
						},
					},
				},
			},
			RenewalContents: []payload.RenewalVotesContent{},
		},
		{
			Contents: []payload.VotesContent{
				{
					VoteType: outputpayload.DposV2,
					VotesInfo: []payload.VotesWithLockTime{
						{
							Candidate: publicKey1,
							Votes:     10000,
							LockTime:  100000,
						},
					},
				},
			},
			RenewalContents: []payload.RenewalVotesContent{
				{
					ReferKey: *referKey,
					VotesInfo: payload.VotesWithLockTime{
						Candidate: publicKey1,
						Votes:     10000,
						LockTime:  100000,
					},
				},
				{
					ReferKey: *referKey,
					VotesInfo: payload.VotesWithLockTime{
						Candidate: publicKey1,
						Votes:     10000,
						LockTime:  100000,
					},
				},
			},
		},
	}

	// 2. Check output payload v0
	err := outputs1[0].Validate()
	s.NoError(err)
	err = outputs1[1].Validate()
	s.EqualError(err, "duplicate vote type")
	err = outputs1[2].Validate()
	s.EqualError(err, "invalid vote type")
	err = outputs1[3].Validate()
	s.EqualError(err, "duplicate candidate")
	err = outputs1[4].Validate()
	s.EqualError(err, "invalid candidate votes")
	err = outputs1[5].Validate()
	s.EqualError(err, "duplicate refer key")

}

func (s *txValidatorTestSuite) TestCheckVoteProducerOutput() {
	// 1. Generate a vote output v0
	publicKeyStr1 := "02b611f07341d5ddce51b5c4366aca7b889cfe0993bd63fd47e944507292ea08dd"
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	outputs1 := []*common2.Output{
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 0,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 0,
				Contents: []outputpayload.VoteContent{
					{
						VoteType:       outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 0,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
							{publicKey1, 0},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 3,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 0,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
						},
					},
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 0,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: 2,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 0,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
						},
					},
				},
			},
		},
	}

	// 2. Check output payload v0
	err := outputs1[0].Payload.(*outputpayload.VoteOutput).Validate()
	s.NoError(err)

	err = outputs1[1].Payload.(*outputpayload.VoteOutput).Validate()
	s.EqualError(err, "invalid public key count")

	err = outputs1[2].Payload.(*outputpayload.VoteOutput).Validate()
	s.EqualError(err, "duplicate candidate")

	err = outputs1[3].Payload.(*outputpayload.VoteOutput).Validate()
	s.EqualError(err, "invalid vote version")

	err = outputs1[4].Payload.(*outputpayload.VoteOutput).Validate()
	s.EqualError(err, "duplicate vote type")

	err = outputs1[5].Payload.(*outputpayload.VoteOutput).Validate()
	s.NoError(err)

	err = outputs1[6].Payload.(*outputpayload.VoteOutput).Validate()
	s.NoError(err)

	// 3. Generate a vote output v1
	outputs := []*common2.Output{
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: outputpayload.VoteProducerAndCRVersion,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 1},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: outputpayload.VoteProducerAndCRVersion,
				Contents: []outputpayload.VoteContent{
					{
						VoteType:       outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: outputpayload.VoteProducerAndCRVersion,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 1},
							{publicKey1, 1},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 3,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 1},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: outputpayload.VoteProducerAndCRVersion,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 1},
						},
					},
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 1},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: outputpayload.VoteProducerAndCRVersion,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: 2,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 1},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: outputpayload.VoteProducerAndCRVersion,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
						},
					},
				},
			},
		},
	}

	// 2. Check output payload v1
	err = outputs[0].Payload.(*outputpayload.VoteOutput).Validate()
	s.NoError(err)

	err = outputs[1].Payload.(*outputpayload.VoteOutput).Validate()
	s.EqualError(err, "invalid public key count")

	err = outputs[2].Payload.(*outputpayload.VoteOutput).Validate()
	s.EqualError(err, "duplicate candidate")

	err = outputs[3].Payload.(*outputpayload.VoteOutput).Validate()
	s.EqualError(err, "invalid vote version")

	err = outputs[4].Payload.(*outputpayload.VoteOutput).Validate()
	s.EqualError(err, "duplicate vote type")

	err = outputs[5].Payload.(*outputpayload.VoteOutput).Validate()
	s.NoError(err)

	err = outputs[6].Payload.(*outputpayload.VoteOutput).Validate()
	s.EqualError(err, "invalid candidate votes")
}

func (s *txValidatorTestSuite) TestCheckRegisterProducerTransaction2() {
	publicKeyStr1 := "031e12374bae471aa09ad479f66c2306f4bcc4ca5b754609a82a1839b94b4721b9"
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	//privateKeyStr1 := "94396a69462208b8fd96d83842855b867d3b0e663203cb31d0dfaec0362ec034"
	//privateKey1, _ := common.HexStringToBytes(privateKeyStr1)

	publicKeyStr2 := "027c4f35081821da858f5c7197bac5e33e77e5af4a3551285f8a8da0a59bd37c45"
	publicKey2, _ := common.HexStringToBytes(publicKeyStr2)
	//errPublicKeyStr := "02b611f07341d5ddce51b5c4366aca7b889cfe0993bd63fd4"
	//errPublicKey, _ := common.HexStringToBytes(errPublicKeyStr)

	publicKeyStr3 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	publicKey3, _ := common.HexStringToBytes(publicKeyStr3)
	//privateKeyStr3 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	//privateKey3, _ := common.HexStringToBytes(privateKeyStr3)

	publicKeyStr4 := "03c77af162438d4b7140f8544ad6523b9734cca9c7a62476d54ed5d1bddc7a39c3"
	publicKey4, _ := common.HexStringToBytes(publicKeyStr4)

	publicKeyStr5 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	publicKey5, _ := common.HexStringToBytes(publicKeyStr5)
	privateKeyStr5 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	privateKey5, _ := common.HexStringToBytes(privateKeyStr5)

	errorPrefix := "transaction validate error: payload content invalid:"
	registerProducer := func() {
		registerPayload := &payload.ProducerInfo{
			OwnerPublicKey: publicKey1,
			NodePublicKey:  publicKey3,
			NickName:       "producer1",
			Url:            "url1",
			Location:       1,
			NetAddress:     "",
		}
		txn := getRegisterProducerTX(publicKeyStr1, registerPayload, s.Chain)
		s.CurrentHeight = 1
		params := s.Chain.GetParams()
		params.DPoSV2StartHeight = 0
		s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams()))
		s.Chain.SetState(state.NewState(s.Chain.GetParams(), nil, nil, nil,
			func() bool { return false }, func(programHash common.Uint168) (common.Fixed64,
				error) {
				amount := common.Fixed64(0)
				utxos, err := s.Chain.GetDB().GetFFLDB().GetUTXO(&programHash)
				if err != nil {
					return amount, err
				}
				for _, utxo := range utxos {
					amount += utxo.Value
				}
				return amount, nil
			}, nil, nil, nil, nil, nil, nil))
		s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
			GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
			GetUTXO:                          s.Chain.GetDB().GetFFLDB().GetUTXO,
			GetHeight:                        func() uint32 { return s.CurrentHeight },
			CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
		})
		block := &types.Block{
			Transactions: []interfaces.Transaction{
				txn,
			},
			Header: common2.Header{Height: s.CurrentHeight},
		}
		//fmt.Println("houpei t.parameters.Config.DPoSV2StartHeight", txn..Config.DPoSV2StartHeight)
		s.Chain.GetState().ProcessBlock(block, nil, 0)
	}
	//register and process
	registerProducer()
	//  OwnerPublicKey is already other's NodePublicKey
	ownerPublicKeyIsOtherNodePublicKey := func() {
		registerPayload := &payload.ProducerInfo{
			OwnerPublicKey: publicKey3,
			NodePublicKey:  publicKey2,
			NickName:       "producer2",
			Url:            "url1",
			Location:       1,
			NetAddress:     "",
		}
		txn := getRegisterProducerTX(publicKeyStr3, registerPayload, s.Chain)
		s.CurrentHeight = 2
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			errorPrefix+"OwnerPublicKey is  already other's NodePublicKey")
	}
	ownerPublicKeyIsOtherNodePublicKey()

	// NodePublicKey is  already other's OwnerPublicKey
	nodePublicKeyIsOtherOwnerPublicKey := func() {

		registerPayload := &payload.ProducerInfo{
			OwnerPublicKey: publicKey2,
			NodePublicKey:  publicKey1,
			NickName:       "producer2",
			Url:            "url1",
			Location:       1,
			NetAddress:     "",
		}
		txn := getRegisterProducerTX(publicKeyStr2, registerPayload, s.Chain)
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			errorPrefix+"NodePublicKey is  already other's OwnerPublicKey")
	}
	nodePublicKeyIsOtherOwnerPublicKey()

	// invalid payload
	invalidPayload := func() {
		wrongPayload := &payload.CRCAppropriation{}
		txn := getRegisterProducerTX(publicKeyStr2, wrongPayload, s.Chain)
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			errorPrefix+"invalid payload")
	}
	invalidPayload()
	//wrong nickname
	wrongNickName := func() {
		registerPayload := &payload.ProducerInfo{
			OwnerPublicKey: publicKey2,
			NodePublicKey:  publicKey1,
			NickName:       "",
			Url:            "url1",
			Location:       1,
			NetAddress:     "",
		}
		txn := getRegisterProducerTX(publicKeyStr3, registerPayload, s.Chain)
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			errorPrefix+"field NickName has invalid string length")
	}
	wrongNickName()
	//wrong url
	wrongURL := func() {
		registerPayload := &payload.ProducerInfo{
			OwnerPublicKey: publicKey2,
			NodePublicKey:  publicKey1,
			NickName:       "NickName",
			Url:            randomUrl(),
			Location:       1,
			NetAddress:     "",
		}
		txn := getRegisterProducerTX(publicKeyStr3, registerPayload, s.Chain)
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			errorPrefix+"field Url has invalid string length")
	}
	wrongURL()
	// check duplication of node public key.
	duplicateNodePublcKey := func() {
		registerPayload := &payload.ProducerInfo{
			OwnerPublicKey: publicKey2,
			NodePublicKey:  publicKey3,
			NickName:       "NickName",
			Url:            "",
			Location:       1,
			NetAddress:     "",
		}
		txn := getRegisterProducerTX(publicKeyStr3, registerPayload, s.Chain)
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			errorPrefix+"Same NodePublicKey producer/cr already registered")
	}
	duplicateNodePublcKey()
	// check duplication of owner public key.
	duplicateOwnerPublcKey := func() {
		registerPayload := &payload.ProducerInfo{
			OwnerPublicKey: publicKey1,
			NodePublicKey:  publicKey5,
			NickName:       "NickName",
			Url:            "",
			Location:       1,
			NetAddress:     "",
		}
		txn := getRegisterProducerTX(publicKeyStr1, registerPayload, s.Chain)
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			errorPrefix+"producer owner already registered")
	}
	duplicateOwnerPublcKey()

	// check duplication of nickname.
	duplicateNickName := func() {
		registerPayload := &payload.ProducerInfo{
			OwnerPublicKey: publicKey2,
			NodePublicKey:  publicKey2,
			NickName:       "producer1",
			Url:            "",
			Location:       1,
			NetAddress:     "",
		}
		txn := getRegisterProducerTX(publicKeyStr3, registerPayload, s.Chain)
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			errorPrefix+"nick name producer1 already inuse")
	}
	duplicateNickName()
	//owner public key is already exist in cr list
	ownerKeyInCrList := func() {
		registerPayload := &payload.ProducerInfo{
			OwnerPublicKey: publicKey2,
			NodePublicKey:  publicKey2,
			NickName:       "producer111",
			Url:            "",
			Location:       1,
			NetAddress:     "",
		}
		//add cr
		code := getCode(publicKey2)
		cid := getCID(code)
		txn := getRegisterProducerTX(publicKeyStr3, registerPayload, s.Chain)
		s.Chain.GetCRCommittee().GetState().CodeCIDMap[common.BytesToHexString(code)] = *cid

		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			errorPrefix+"owner public key 027c4f35081821da858f5c7197bac5e33e77e5af4a3551285f8a8da0a59bd37c45 already exist in cr list")
		delete(s.Chain.GetCRCommittee().GetState().CodeCIDMap, common.BytesToHexString(code))

	}
	ownerKeyInCrList()

	//node public key is already exist in cr list
	nodePublicKeyInCrList := func() {
		registerPayload := &payload.ProducerInfo{
			OwnerPublicKey: publicKey4,
			NodePublicKey:  publicKey5,
			NickName:       "producer111",
			Url:            "",
			Location:       1,
			NetAddress:     "",
		}
		//add cr
		code := getCode(publicKey5)
		cid := getCID(code)
		txn := getRegisterProducerTX(publicKeyStr5, registerPayload, s.Chain)
		s.Chain.GetCRCommittee().GetState().CodeCIDMap[common.BytesToHexString(code)] = *cid

		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			errorPrefix+"node public key 036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535 already exist in cr list")
		s.Chain.GetCRCommittee().GetState().CodeCIDMap[common.BytesToHexString(code)] = *cid
		delete(s.Chain.GetCRCommittee().GetState().CodeCIDMap, common.BytesToHexString(code))
	}
	nodePublicKeyInCrList()
	//can not register dposv1 after dposv2 active height
	noRegisterDPOSV1 := func() {
		registerPayload := &payload.ProducerInfo{
			OwnerPublicKey: publicKey5,
			NodePublicKey:  publicKey4,
			NickName:       "producer111",
			Url:            "",
			Location:       1,
			NetAddress:     "",
		}
		//
		signedBuf := new(bytes.Buffer)
		registerPayload.SerializeUnsigned(signedBuf, payload.ProducerInfoVersion)
		registerPayload.Signature, _ = crypto.Sign(privateKey5, signedBuf.Bytes())

		txn := getRegisterProducerTX(publicKeyStr3, registerPayload, s.Chain)
		s.Chain.GetState().DPoSV2ActiveHeight = 0
		header := &common2.Header{
			Version:    0,
			Previous:   common.Uint256{},
			MerkleRoot: common.EmptyHash,
			Timestamp:  uint32(time.Now().Unix()),
			Bits:       config.DefaultParams.PowLimitBits,
			Height:     1,
			Nonce:      1,
		}
		blockNode := blockchain.NewBlockNode(header, &common.Uint256{})
		s.Chain.SetTip(blockNode)
		s.CurrentHeight = 2
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			errorPrefix+"can not register dposv1 after dposv2 active height")
	}
	noRegisterDPOSV1()

}

func getRegisterProducerTX(publicKeyStr3 string, registerPayload interfaces.Payload,
	chain *blockchain.BlockChain) interfaces.Transaction {
	programs := []*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr3),
		Parameter: nil,
	}}

	txn := functions.CreateTransaction(
		0,
		common2.RegisterProducer,
		0,
		registerPayload,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		programs,
	)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: chain.GetHeight(),
		TimeStamp:   chain.BestChain.Timestamp,
		Config:      chain.GetParams(),
		BlockChain:  chain,
	})
	return txn
}

func randomUrl() string {
	a := make([]byte, 101)
	rand.Read(a)
	return common.BytesToHexString(a)
}

func (s *txValidatorTestSuite) TestCheckUpdateProducerTransaction() {
	publicKeyStr1 := "031e12374bae471aa09ad479f66c2306f4bcc4ca5b754609a82a1839b94b4721b9"
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	privateKeyStr1 := "94396a69462208b8fd96d83842855b867d3b0e663203cb31d0dfaec0362ec034"
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)
	publicKeyStr2 := "027c4f35081821da858f5c7197bac5e33e77e5af4a3551285f8a8da0a59bd37c45"
	publicKey2, _ := common.HexStringToBytes(publicKeyStr2)
	errPublicKeyStr := "02b611f07341d5ddce51b5c4366aca7b889cfe0993bd63fd4"
	errPublicKey, _ := common.HexStringToBytes(errPublicKeyStr)

	registerPayload := &payload.ProducerInfo{
		OwnerPublicKey: publicKey1,
		NodePublicKey:  publicKey1,
		NickName:       "",
		Url:            "",
		Location:       1,
		NetAddress:     "",
	}
	programs := []*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr1),
		Parameter: nil,
	}}

	txn := functions.CreateTransaction(
		0,
		common2.RegisterProducer,
		0,
		registerPayload,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		programs,
	)

	s.CurrentHeight = 1
	s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams()))
	s.Chain.SetState(state.NewState(s.Chain.GetParams(), nil, nil, nil,
		func() bool { return false }, func(programHash common.Uint168) (common.Fixed64,
			error) {
			amount := common.Fixed64(0)
			utxos, err := s.Chain.GetDB().GetFFLDB().GetUTXO(&programHash)
			if err != nil {
				return amount, err
			}
			for _, utxo := range utxos {
				amount += utxo.Value
			}
			return amount, nil
		}, nil, nil, nil, nil, nil, nil))
	s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
		GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
		GetUTXO:                          s.Chain.GetDB().GetFFLDB().GetUTXO,
		GetHeight:                        func() uint32 { return s.CurrentHeight },
		CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
	})
	block := &types.Block{
		Transactions: []interfaces.Transaction{
			txn,
		},
		Header: common2.Header{Height: s.CurrentHeight},
	}
	s.Chain.GetState().ProcessBlock(block, nil, 0)

	txn.SetTxType(common2.UpdateProducer)
	updatePayload := &payload.ProducerInfo{
		OwnerPublicKey: publicKey1,
		NodePublicKey:  publicKey1,
		NickName:       "",
		Url:            "",
		Location:       2,
		NetAddress:     "",
	}
	txn.SetPayload(updatePayload)
	s.CurrentHeight++
	block.Header = common2.Header{Height: s.CurrentHeight}
	s.Chain.GetState().ProcessBlock(block, nil, 0)
	txn = CreateTransactionByType(txn, s.Chain)
	err, _ := txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:field NickName has invalid string length")
	updatePayload.NickName = "nick name"

	updatePayload.Url = "www.elastos.org"
	updatePayload.OwnerPublicKey = errPublicKey
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid owner public key in payload")

	// check node public when block height is higher than h2
	originHeight := config.DefaultParams.PublicDPOSHeight
	updatePayload.NodePublicKey = errPublicKey
	config.DefaultParams.PublicDPOSHeight = 0
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid node public key in payload")
	config.DefaultParams.PublicDPOSHeight = originHeight

	// check node public key same with CRC
	txn.Payload().(*payload.ProducerInfo).OwnerPublicKey = publicKey2
	pk, _ := common.HexStringToBytes(config.DefaultParams.CRCArbiters[0])
	txn.Payload().(*payload.ProducerInfo).NodePublicKey = pk
	config.DefaultParams.PublicDPOSHeight = 0
	err, _ = txn.SpecialContextCheck()
	config.DefaultParams.PublicDPOSHeight = originHeight
	s.EqualError(err, "transaction validate error: payload content invalid:node public key can't equal with CR Arbiters")

	// check owner public key same with CRC
	txn.Payload().(*payload.ProducerInfo).NodePublicKey = publicKey2
	pk, _ = common.HexStringToBytes(config.DefaultParams.CRCArbiters[0])
	txn.Payload().(*payload.ProducerInfo).OwnerPublicKey = pk
	config.DefaultParams.PublicDPOSHeight = 0
	err, _ = txn.SpecialContextCheck()
	config.DefaultParams.PublicDPOSHeight = originHeight
	s.EqualError(err, "transaction validate error: payload content invalid:invalid signature in payload")

	updatePayload.OwnerPublicKey = publicKey2
	updatePayload.NodePublicKey = publicKey1
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid signature in payload")

	updatePayload.OwnerPublicKey = publicKey1
	updateSignBuf := new(bytes.Buffer)
	err1 := updatePayload.SerializeUnsigned(updateSignBuf, payload.ProducerInfoVersion)
	s.NoError(err1)
	updateSig, err1 := crypto.Sign(privateKey1, updateSignBuf.Bytes())
	s.NoError(err1)
	updatePayload.Signature = updateSig
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	//rest of check test will be continued in chain test
}

func (s *txValidatorTestSuite) TestCheckUpdateProducerV1V2Transaction() {
	publicKeyStr1 := "031e12374bae471aa09ad479f66c2306f4bcc4ca5b754609a82a1839b94b4721b9"
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	privateKeyStr1 := "94396a69462208b8fd96d83842855b867d3b0e663203cb31d0dfaec0362ec034"
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)
	publicKeyStr2 := "027c4f35081821da858f5c7197bac5e33e77e5af4a3551285f8a8da0a59bd37c45"
	publicKey2, _ := common.HexStringToBytes(publicKeyStr2)
	errPublicKeyStr := "02b611f07341d5ddce51b5c4366aca7b889cfe0993bd63fd4"
	errPublicKey, _ := common.HexStringToBytes(errPublicKeyStr)

	registerPayload := &payload.ProducerInfo{
		OwnerPublicKey: publicKey1,
		NodePublicKey:  publicKey1,
		NickName:       "",
		Url:            "",
		Location:       1,
		NetAddress:     "",
	}
	programs := []*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr1),
		Parameter: nil,
	}}

	txn := functions.CreateTransaction(
		0,
		common2.RegisterProducer,
		0,
		registerPayload,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		programs,
	)

	s.CurrentHeight = 1
	s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams()))
	s.Chain.SetState(state.NewState(s.Chain.GetParams(), nil, nil, nil,
		func() bool { return false }, func(programHash common.Uint168) (common.Fixed64,
			error) {
			amount := common.Fixed64(0)
			utxos, err := s.Chain.GetDB().GetFFLDB().GetUTXO(&programHash)
			if err != nil {
				return amount, err
			}
			for _, utxo := range utxos {
				amount += utxo.Value
			}
			return amount, nil
		}, nil, nil, nil, nil, nil, nil))
	s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
		GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
		GetUTXO:                          s.Chain.GetDB().GetFFLDB().GetUTXO,
		GetHeight:                        func() uint32 { return s.CurrentHeight },
		CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
	})
	block := &types.Block{
		Transactions: []interfaces.Transaction{
			txn,
		},
		Header: common2.Header{Height: s.CurrentHeight},
	}
	s.Chain.GetState().ProcessBlock(block, nil, 0)

	txn.SetTxType(common2.UpdateProducer)
	updatePayload := &payload.ProducerInfo{
		OwnerPublicKey: publicKey1,
		NodePublicKey:  publicKey1,
		NickName:       "",
		Url:            "",
		Location:       2,
		NetAddress:     "",
		StakeUntil:     10,
	}
	txn.SetPayload(updatePayload)
	s.CurrentHeight++
	block.Header = common2.Header{Height: s.CurrentHeight}
	s.Chain.GetState().ProcessBlock(block, nil, 0)
	txn = CreateTransactionByType(txn, s.Chain)
	err, _ := txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:field NickName has invalid string length")
	updatePayload.NickName = "nick name"

	updatePayload.Url = "www.elastos.org"
	updatePayload.OwnerPublicKey = errPublicKey
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid owner public key in payload")

	// check node public when block height is higher than h2
	originHeight := config.DefaultParams.PublicDPOSHeight
	updatePayload.NodePublicKey = errPublicKey
	config.DefaultParams.PublicDPOSHeight = 0
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid node public key in payload")
	config.DefaultParams.PublicDPOSHeight = originHeight

	// check node public key same with CRC
	txn.Payload().(*payload.ProducerInfo).OwnerPublicKey = publicKey2
	pk, _ := common.HexStringToBytes(config.DefaultParams.CRCArbiters[0])
	txn.Payload().(*payload.ProducerInfo).NodePublicKey = pk
	config.DefaultParams.PublicDPOSHeight = 0
	err, _ = txn.SpecialContextCheck()
	config.DefaultParams.PublicDPOSHeight = originHeight
	s.EqualError(err, "transaction validate error: payload content invalid:node public key can't equal with CR Arbiters")

	// check owner public key same with CRC
	txn.Payload().(*payload.ProducerInfo).NodePublicKey = publicKey2
	pk, _ = common.HexStringToBytes(config.DefaultParams.CRCArbiters[0])
	txn.Payload().(*payload.ProducerInfo).OwnerPublicKey = pk
	config.DefaultParams.PublicDPOSHeight = 0
	err, _ = txn.SpecialContextCheck()
	config.DefaultParams.PublicDPOSHeight = originHeight
	s.EqualError(err, "transaction validate error: payload content invalid:invalid signature in payload")

	updatePayload.OwnerPublicKey = publicKey2
	updatePayload.NodePublicKey = publicKey1
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid signature in payload")

	updatePayload.OwnerPublicKey = publicKey1
	updateSignBuf := new(bytes.Buffer)
	err1 := updatePayload.SerializeUnsigned(updateSignBuf, payload.ProducerInfoVersion)
	s.NoError(err1)
	updateSig, err1 := crypto.Sign(privateKey1, updateSignBuf.Bytes())
	s.NoError(err1)
	updatePayload.Signature = updateSig
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	//process block
	block = &types.Block{
		Transactions: []interfaces.Transaction{
			txn,
		},
		Header: common2.Header{Height: s.CurrentHeight},
	}
	s.Chain.GetState().ProcessBlock(block, nil, 0)
	// update stakeuntil
	updatePayload.StakeUntil = 20
	txn.SetPayload(updatePayload)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:Pending Canceled or Returned producer can  not update  StakeUntil ")

	s.Chain.BestChain.Height = 100
	s.Chain.GetState().DPoSV2ActiveHeight = 10
	txn = CreateTransactionByType(txn, s.Chain)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:producer already expired and dposv2 already started, can not update anything ")

	s.Chain.BestChain.Height = 5
	s.Chain.GetState().DPoSV2ActiveHeight = 2
	txn = CreateTransactionByType(txn, s.Chain)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:Pending Canceled or Returned producer can  not update  StakeUntil ")

	producer := s.Chain.GetState().GetProducer(publicKey1)
	producer.SetState(state.Active)
	txn = CreateTransactionByType(txn, s.Chain)
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)
}

func (s *txValidatorTestSuite) TestCheckUpdateProducerV2Transaction() {
	publicKeyStr1 := "031e12374bae471aa09ad479f66c2306f4bcc4ca5b754609a82a1839b94b4721b9"
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	privateKeyStr1 := "94396a69462208b8fd96d83842855b867d3b0e663203cb31d0dfaec0362ec034"
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)
	publicKeyStr2 := "027c4f35081821da858f5c7197bac5e33e77e5af4a3551285f8a8da0a59bd37c45"
	publicKey2, _ := common.HexStringToBytes(publicKeyStr2)
	errPublicKeyStr := "02b611f07341d5ddce51b5c4366aca7b889cfe0993bd63fd4"
	errPublicKey, _ := common.HexStringToBytes(errPublicKeyStr)

	registerPayload := &payload.ProducerInfo{
		OwnerPublicKey: publicKey1,
		NodePublicKey:  publicKey1,
		NickName:       "",
		Url:            "",
		Location:       1,
		NetAddress:     "",
		StakeUntil:     100,
	}
	programs := []*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr1),
		Parameter: nil,
	}}

	txn := functions.CreateTransaction(
		0,
		common2.RegisterProducer,
		0,
		registerPayload,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		programs,
	)

	s.CurrentHeight = 1
	s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams()))
	s.Chain.SetState(state.NewState(s.Chain.GetParams(), nil, nil, nil,
		func() bool { return false }, func(programHash common.Uint168) (common.Fixed64,
			error) {
			amount := common.Fixed64(0)
			utxos, err := s.Chain.GetDB().GetFFLDB().GetUTXO(&programHash)
			if err != nil {
				return amount, err
			}
			for _, utxo := range utxos {
				amount += utxo.Value
			}
			return amount, nil
		}, nil, nil, nil, nil, nil, nil))
	s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
		GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
		GetUTXO:                          s.Chain.GetDB().GetFFLDB().GetUTXO,
		GetHeight:                        func() uint32 { return s.CurrentHeight },
		CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
	})
	block := &types.Block{
		Transactions: []interfaces.Transaction{
			txn,
		},
		Header: common2.Header{Height: s.CurrentHeight},
	}
	s.Chain.GetState().ProcessBlock(block, nil, 0)

	txn.SetTxType(common2.UpdateProducer)
	updatePayload := &payload.ProducerInfo{
		OwnerPublicKey: publicKey1,
		NodePublicKey:  publicKey1,
		NickName:       "",
		Url:            "",
		Location:       2,
		NetAddress:     "",
		StakeUntil:     1000,
	}
	txn.SetPayload(updatePayload)
	s.CurrentHeight++
	block.Header = common2.Header{Height: s.CurrentHeight}
	s.Chain.GetState().ProcessBlock(block, nil, 0)
	txn = CreateTransactionByType(txn, s.Chain)
	err, _ := txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:field NickName has invalid string length")
	updatePayload.NickName = "nick name"

	updatePayload.Url = "www.elastos.org"
	updatePayload.OwnerPublicKey = errPublicKey
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid owner public key in payload")

	// check node public when block height is higher than h2
	originHeight := config.DefaultParams.PublicDPOSHeight
	updatePayload.NodePublicKey = errPublicKey
	config.DefaultParams.PublicDPOSHeight = 0
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid node public key in payload")
	config.DefaultParams.PublicDPOSHeight = originHeight

	// check node public key same with CRC
	txn.Payload().(*payload.ProducerInfo).OwnerPublicKey = publicKey2
	pk, _ := common.HexStringToBytes(config.DefaultParams.CRCArbiters[0])
	txn.Payload().(*payload.ProducerInfo).NodePublicKey = pk
	config.DefaultParams.PublicDPOSHeight = 0
	err, _ = txn.SpecialContextCheck()
	config.DefaultParams.PublicDPOSHeight = originHeight
	s.EqualError(err, "transaction validate error: payload content invalid:node public key can't equal with CR Arbiters")

	// check owner public key same with CRC
	txn.Payload().(*payload.ProducerInfo).NodePublicKey = publicKey2
	pk, _ = common.HexStringToBytes(config.DefaultParams.CRCArbiters[0])
	txn.Payload().(*payload.ProducerInfo).OwnerPublicKey = pk
	config.DefaultParams.PublicDPOSHeight = 0
	err, _ = txn.SpecialContextCheck()
	config.DefaultParams.PublicDPOSHeight = originHeight
	s.EqualError(err, "transaction validate error: payload content invalid:invalid signature in payload")

	updatePayload.OwnerPublicKey = publicKey2
	updatePayload.NodePublicKey = publicKey1
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid signature in payload")

	updatePayload.OwnerPublicKey = publicKey1
	updateSignBuf := new(bytes.Buffer)
	err1 := updatePayload.SerializeUnsigned(updateSignBuf, payload.ProducerInfoVersion)
	s.NoError(err1)
	updateSig, err1 := crypto.Sign(privateKey1, updateSignBuf.Bytes())
	s.NoError(err1)
	updatePayload.Signature = updateSig
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	s.Chain.BestChain.Height = 10000
	txn = CreateTransactionByType(txn, s.Chain)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:DPoS 2.0 node has expired")

	s.Chain.BestChain.Height = 100
	updatePayload.StakeUntil = 10
	txn.SetPayload(updatePayload)
	txn = CreateTransactionByType(txn, s.Chain)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:stake time is smaller than before")

	s.Chain.BestChain.Height = 100
	updatePayload.StakeUntil = 10000
	txn.SetPayload(updatePayload)
	txn = CreateTransactionByType(txn, s.Chain)
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)
}

func (s *txValidatorTestSuite) TestCheckCancelProducerTransaction() {
	publicKeyStr1 := "031e12374bae471aa09ad479f66c2306f4bcc4ca5b754609a82a1839b94b4721b9"
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	privateKeyStr1 := "94396a69462208b8fd96d83842855b867d3b0e663203cb31d0dfaec0362ec034"
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)
	publicKeyStr2 := "027c4f35081821da858f5c7197bac5e33e77e5af4a3551285f8a8da0a59bd37c45"
	publicKey2, _ := common.HexStringToBytes(publicKeyStr2)
	errPublicKeyStr := "02b611f07341d5ddce51b5c4366aca7b889cfe0993bd63fd4"
	errPublicKey, _ := common.HexStringToBytes(errPublicKeyStr)

	cancelPayload := &payload.ProcessProducer{
		OwnerPublicKey: publicKey1,
	}

	programs := []*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr1),
		Parameter: nil,
	}}

	txn := functions.CreateTransaction(
		0,
		common2.CancelProducer,
		0,
		cancelPayload,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		programs,
	)

	cancelPayload.OwnerPublicKey = errPublicKey
	txn = CreateTransactionByType(txn, s.Chain)
	err, _ := txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid public key in payload")

	cancelPayload.OwnerPublicKey = publicKey2
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid signature in payload")

	buf := new(bytes.Buffer)
	cancelPayload.OwnerPublicKey = publicKey1
	cancelPayload.SerializeUnsigned(buf, 0)

	sig, _ := crypto.Sign(privateKey1, buf.Bytes())
	cancelPayload.Signature = sig
	s.Chain.SetState(state.NewState(s.Chain.GetParams(), nil, nil, nil,
		func() bool { return false }, func(programHash common.Uint168) (common.Fixed64,
			error) {
			amount := common.Fixed64(0)
			utxos, err := s.Chain.GetDB().GetFFLDB().GetUTXO(&programHash)
			if err != nil {
				return amount, err
			}
			for _, utxo := range utxos {
				amount += utxo.Value
			}
			return amount, nil
		}, nil, nil, nil, nil, nil, nil))
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:getting unknown producer")

	{
		registerPayload := &payload.ProducerInfo{
			OwnerPublicKey: publicKey1,
			NodePublicKey:  publicKey1,
			NickName:       "",
			Url:            "",
			Location:       1,
			NetAddress:     "",
		}
		programs = []*program.Program{{
			Code:      getCodeByPubKeyStr(publicKeyStr1),
			Parameter: nil,
		}}

		txn1 := functions.CreateTransaction(
			0,
			common2.RegisterProducer,
			0,
			registerPayload,
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			programs,
		)

		s.CurrentHeight = 1
		s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams()))
		s.Chain.SetState(state.NewState(s.Chain.GetParams(), nil, nil, nil,
			func() bool { return false }, func(programHash common.Uint168) (common.Fixed64,
				error) {
				amount := common.Fixed64(0)
				utxos, err := s.Chain.GetDB().GetFFLDB().GetUTXO(&programHash)
				if err != nil {
					return amount, err
				}
				for _, utxo := range utxos {
					amount += utxo.Value
				}
				return amount, nil
			}, nil, nil, nil, nil, nil, nil))
		s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
			GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
			GetUTXO:                          s.Chain.GetDB().GetFFLDB().GetUTXO,
			GetHeight:                        func() uint32 { return s.CurrentHeight },
			CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
		})
		block := &types.Block{
			Transactions: []interfaces.Transaction{
				txn1,
			},
			Header: common2.Header{Height: s.CurrentHeight},
		}
		s.Chain.GetState().ProcessBlock(block, nil, 0)

		err, _ = txn.SpecialContextCheck()
		s.NoError(err)
	}

	{
		registerPayload := &payload.ProducerInfo{
			OwnerPublicKey: publicKey1,
			NodePublicKey:  publicKey1,
			NickName:       "",
			Url:            "",
			Location:       1,
			NetAddress:     "",
			StakeUntil:     100,
		}
		programs = []*program.Program{{
			Code:      getCodeByPubKeyStr(publicKeyStr1),
			Parameter: nil,
		}}

		txn1 := functions.CreateTransaction(
			0,
			common2.RegisterProducer,
			0,
			registerPayload,
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			programs,
		)

		s.CurrentHeight = 1
		s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams()))
		s.Chain.SetState(state.NewState(s.Chain.GetParams(), nil, nil, nil,
			func() bool { return false }, func(programHash common.Uint168) (common.Fixed64,
				error) {
				amount := common.Fixed64(0)
				utxos, err := s.Chain.GetDB().GetFFLDB().GetUTXO(&programHash)
				if err != nil {
					return amount, err
				}
				for _, utxo := range utxos {
					amount += utxo.Value
				}
				return amount, nil
			}, nil, nil, nil, nil, nil, nil))
		s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
			GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
			GetUTXO:                          s.Chain.GetDB().GetFFLDB().GetUTXO,
			GetHeight:                        func() uint32 { return s.CurrentHeight },
			CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
		})
		block := &types.Block{
			Transactions: []interfaces.Transaction{
				txn1,
			},
			Header: common2.Header{Height: s.CurrentHeight},
		}
		s.Chain.GetState().ProcessBlock(block, nil, 0)

		err, _ = txn.SpecialContextCheck()
		s.EqualError(err, "transaction validate error: payload content invalid:can not cancel DPoS V2 producer")
	}

	{
		registerPayload := &payload.ProducerInfo{
			OwnerPublicKey: publicKey1,
			NodePublicKey:  publicKey1,
			NickName:       "",
			Url:            "",
			Location:       1,
			NetAddress:     "",
		}
		programs = []*program.Program{{
			Code:      getCodeByPubKeyStr(publicKeyStr1),
			Parameter: nil,
		}}

		txn1 := functions.CreateTransaction(
			0,
			common2.RegisterProducer,
			0,
			registerPayload,
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			programs,
		)

		s.CurrentHeight = 1
		s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams()))
		s.Chain.SetState(state.NewState(s.Chain.GetParams(), nil, nil, nil,
			func() bool { return false }, func(programHash common.Uint168) (common.Fixed64,
				error) {
				amount := common.Fixed64(0)
				utxos, err := s.Chain.GetDB().GetFFLDB().GetUTXO(&programHash)
				if err != nil {
					return amount, err
				}
				for _, utxo := range utxos {
					amount += utxo.Value
				}
				return amount, nil
			}, nil, nil, nil, nil, nil, nil))
		s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
			GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
			GetUTXO:                          s.Chain.GetDB().GetFFLDB().GetUTXO,
			GetHeight:                        func() uint32 { return s.CurrentHeight },
			CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
		})
		block := &types.Block{
			Transactions: []interfaces.Transaction{
				txn1,
			},
			Header: common2.Header{Height: s.CurrentHeight},
		}
		s.Chain.GetState().ProcessBlock(block, nil, 0)

		txn2 := functions.CreateTransaction(
			0,
			common2.UpdateProducer,
			0,
			nil,
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			programs,
		)
		updatePayload := &payload.ProducerInfo{
			OwnerPublicKey: publicKey1,
			NodePublicKey:  publicKey1,
			NickName:       "nick name",
			Url:            "www.elastos.org",
			Location:       2,
			NetAddress:     "",
			StakeUntil:     10,
		}
		txn2.SetPayload(updatePayload)

		updateSignBuf := new(bytes.Buffer)
		err1 := updatePayload.SerializeUnsigned(updateSignBuf, payload.ProducerInfoVersion)
		s.NoError(err1)
		updateSig, err1 := crypto.Sign(privateKey1, updateSignBuf.Bytes())
		s.NoError(err1)
		updatePayload.Signature = updateSig
		s.Chain.GetState().GetProducer(publicKey1).SetState(state.Active)
		s.Chain.GetParams().DPoSV2DepositCoinMinLockTime = 1
		s.Chain.BestChain.Height = 1
		txn2 = CreateTransactionByType(txn2, s.Chain)
		err, _ = txn2.SpecialContextCheck()
		s.NoError(err)
		block = &types.Block{
			Transactions: []interfaces.Transaction{
				txn2,
			},
			Header: common2.Header{Height: s.CurrentHeight},
		}
		s.Chain.GetState().ProcessBlock(block, nil, 0)

		err, _ = txn.SpecialContextCheck()
		s.EqualError(err, "transaction validate error: payload content invalid:can not cancel DPoS V1&V2 producer")

		s.Chain.GetState().GetProducer(publicKey1).SetState(state.Illegal)
		s.Chain.BestChain.Height = 1000
		txn = CreateTransactionByType(txn, s.Chain)
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err, "transaction validate error: payload content invalid:can not cancel this producer")

		s.Chain.GetState().GetProducer(publicKey1).SetState(state.Active)
		txn = CreateTransactionByType(txn, s.Chain)
		err, _ = txn.SpecialContextCheck()
		s.NoError(err)
	}

}

func (s *txValidatorTestSuite) TestCheckActivateProducerTransaction() {
	publicKeyStr1 := "031e12374bae471aa09ad479f66c2306f4bcc4ca5b754609a82a1839b94b4721b9"
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	privateKeyStr1 := "94396a69462208b8fd96d83842855b867d3b0e663203cb31d0dfaec0362ec034"
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)
	publicKeyStr2 := "027c4f35081821da858f5c7197bac5e33e77e5af4a3551285f8a8da0a59bd37c45"
	publicKey2, _ := common.HexStringToBytes(publicKeyStr2)
	errPublicKeyStr := "02b611f07341d5ddce51b5c4366aca7b889cfe0993bd63fd4"
	errPublicKey, _ := common.HexStringToBytes(errPublicKeyStr)

	activatePayload := &payload.ActivateProducer{
		NodePublicKey: publicKey1,
	}

	programs := []*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr1),
		Parameter: nil,
	}}

	txn := functions.CreateTransaction(
		0,
		common2.ActivateProducer,
		0,
		activatePayload,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		programs,
	)

	activatePayload.NodePublicKey = errPublicKey

	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: 0,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ := txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:invalid public key in payload")

	activatePayload.NodePublicKey = publicKey2
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:invalid signature in payload")

	buf := new(bytes.Buffer)
	activatePayload.NodePublicKey = publicKey1
	activatePayload.SerializeUnsigned(buf, 0)
	sig, _ := crypto.Sign(privateKey1, buf.Bytes())
	activatePayload.Signature = sig
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:getting unknown producer")

	{
		registerPayload := &payload.ProducerInfo{
			OwnerPublicKey: publicKey1,
			NodePublicKey:  publicKey1,
			NickName:       "",
			Url:            "",
			Location:       1,
			NetAddress:     "",
		}
		programs = []*program.Program{{
			Code:      getCodeByPubKeyStr(publicKeyStr1),
			Parameter: nil,
		}}

		txn1 := functions.CreateTransaction(
			0,
			common2.RegisterProducer,
			0,
			registerPayload,
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			programs,
		)

		s.CurrentHeight = 1
		s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams()))
		s.Chain.SetState(state.NewState(s.Chain.GetParams(), nil, nil, nil,
			func() bool { return false }, func(programHash common.Uint168) (common.Fixed64,
				error) {
				amount := common.Fixed64(0)
				utxos, err := s.Chain.GetDB().GetFFLDB().GetUTXO(&programHash)
				if err != nil {
					return amount, err
				}
				for _, utxo := range utxos {
					amount += utxo.Value
				}
				return amount, nil
			}, nil, nil, nil, nil, nil, nil))
		s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
			GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
			GetUTXO:                          s.Chain.GetDB().GetFFLDB().GetUTXO,
			GetHeight:                        func() uint32 { return s.CurrentHeight },
			CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
		})
		block := &types.Block{
			Transactions: []interfaces.Transaction{
				txn1,
			},
			Header: common2.Header{Height: s.CurrentHeight},
		}
		s.Chain.GetState().ProcessBlock(block, nil, 0)

		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:can not activate this producer")

		s.Chain.GetState().GetProducer(publicKey1).SetState(state.Inactive)
		txn = CreateTransactionByType(txn, s.Chain)
		txn.SetParameters(&transaction.TransactionParameters{
			Transaction: txn,
			BlockHeight: 0,
			TimeStamp:   s.Chain.BestChain.Timestamp,
			Config:      s.Chain.GetParams(),
			BlockChain:  s.Chain,
		})
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err, "transaction validate error: payload content invalid:insufficient deposit amount")

		s.Chain.GetState().GetProducer(publicKey1).SetTotalAmount(500100000000)
		s.Chain.GetParams().CRVotingStartHeight = 1
		s.Chain.BestChain.Height = 10
		txn = CreateTransactionByType(txn, s.Chain)
		err, _ = txn.SpecialContextCheck()
		s.NoError(err)
	}

	{
		registerPayload := &payload.ProducerInfo{
			OwnerPublicKey: publicKey1,
			NodePublicKey:  publicKey1,
			NickName:       "",
			Url:            "",
			Location:       1,
			NetAddress:     "",
			StakeUntil:     100,
		}
		programs = []*program.Program{{
			Code:      getCodeByPubKeyStr(publicKeyStr1),
			Parameter: nil,
		}}

		txn1 := functions.CreateTransaction(
			0,
			common2.RegisterProducer,
			0,
			registerPayload,
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			programs,
		)

		s.CurrentHeight = 1
		s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams()))
		s.Chain.SetState(state.NewState(s.Chain.GetParams(), nil, nil, nil,
			func() bool { return false }, func(programHash common.Uint168) (common.Fixed64,
				error) {
				amount := common.Fixed64(0)
				utxos, err := s.Chain.GetDB().GetFFLDB().GetUTXO(&programHash)
				if err != nil {
					return amount, err
				}
				for _, utxo := range utxos {
					amount += utxo.Value
				}
				return amount, nil
			}, nil, nil, nil, nil, nil, nil))
		s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
			GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
			GetUTXO:                          s.Chain.GetDB().GetFFLDB().GetUTXO,
			GetHeight:                        func() uint32 { return s.CurrentHeight },
			CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
		})
		block := &types.Block{
			Transactions: []interfaces.Transaction{
				txn1,
			},
			Header: common2.Header{Height: s.CurrentHeight},
		}
		s.Chain.GetState().ProcessBlock(block, nil, 0)

		s.Chain.GetState().GetProducer(publicKey1).SetState(state.Inactive)
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err, "transaction validate error: payload content invalid:insufficient deposit amount")

		s.Chain.GetState().GetProducer(publicKey1).SetTotalAmount(200100000000)
		s.Chain.GetParams().CRVotingStartHeight = 1
		s.Chain.BestChain.Height = 10
		txn = CreateTransactionByType(txn, s.Chain)
		err, _ = txn.SpecialContextCheck()
		s.NoError(err)
	}

}

func (s *txValidatorTestSuite) TestCheckRegisterCRTransaction() {
	config.DefaultParams = config.GetDefaultParams()

	// Generate a register CR transaction
	publicKeyStr1 := "03c77af162438d4b7140f8544ad6523b9734cca9c7a62476d54ed5d1bddc7a39c3"
	privateKeyStr1 := "7638c2a799d93185279a4a6ae84a5b76bd89e41fa9f465d9ae9b2120533983a1"
	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"
	nickName1 := randomString()

	hash1, _ := getDepositAddress(publicKeyStr1)
	hash2, _ := getDepositAddress(publicKeyStr2)

	txn := s.getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1,
		payload.CRInfoVersion, &common.Uint168{})

	code1 := getCodeByPubKeyStr(publicKeyStr1)
	code2 := getCodeByPubKeyStr(publicKeyStr2)
	codeStr1 := common.BytesToHexString(code1)

	cid1 := getCID(code1)
	cid2 := getCID(code2)

	votingHeight := config.DefaultParams.CRVotingStartHeight
	registerCRByDIDHeight := config.DefaultParams.RegisterCRByDIDHeight

	// All ok
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ := txn.SpecialContextCheck()
	s.NoError(err)

	// Give an invalid NickName length 0 in payload
	nickName := txn.Payload().(*payload.CRInfo).NickName
	txn.Payload().(*payload.CRInfo).NickName = ""
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:field NickName has invalid string length")

	// Give an invalid NickName length more than 100 in payload
	txn.Payload().(*payload.CRInfo).NickName = "012345678901234567890123456789012345678901234567890" +
		"12345678901234567890123456789012345678901234567890123456789"
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:field NickName has invalid string length")

	// Give an invalid url length more than 100 in payload
	url := txn.Payload().(*payload.CRInfo).Url
	txn.Payload().(*payload.CRInfo).NickName = nickName
	txn.Payload().(*payload.CRInfo).Url = "012345678901234567890123456789012345678901234567890" +
		"12345678901234567890123456789012345678901234567890123456789"
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:field Url has invalid string length")

	// Not in vote Period lower
	txn.Payload().(*payload.CRInfo).Url = url
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: config.DefaultParams.CRVotingStartHeight - 1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:should create tx during voting period")

	// Not in vote Period upper c.params.CRCommitteeStartHeight
	s.Chain.GetCRCommittee().InElectionPeriod = true
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: config.DefaultParams.CRCommitteeStartHeight + 1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:should create tx during voting period")

	// Nickname already in use
	s.Chain.GetCRCommittee().GetState().Nicknames[nickName1] = struct{}{}
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:nick name "+nickName1+" already inuse")

	delete(s.Chain.GetCRCommittee().GetState().Nicknames, nickName1)
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: 0,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:should create tx during voting period")

	delete(s.Chain.GetCRCommittee().GetState().CodeCIDMap, codeStr1)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	// CID already exist
	s.Chain.GetCRCommittee().GetState().CodeCIDMap[codeStr1] = *cid1
	s.Chain.GetCRCommittee().GetState().Candidates[*cid1] = &crstate.Candidate{}
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:cid "+cid1.String()+" already exist")
	delete(s.Chain.GetCRCommittee().GetState().Candidates, *cid1)

	// Give an invalid code in payload
	txn.Payload().(*payload.CRInfo).Code = []byte{}
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:code is nil")

	// Give an invalid CID in payload
	txn.Payload().(*payload.CRInfo).Code = code1
	txn.Payload().(*payload.CRInfo).CID = common.Uint168{1, 2, 3}
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid cid address")

	// Give a mismatching code and CID in payload
	txn.Payload().(*payload.CRInfo).CID = *cid2
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid cid address")

	// Invalidates the signature in payload
	txn.Payload().(*payload.CRInfo).CID = *cid1
	signatature := txn.Payload().(*payload.CRInfo).Signature
	txn.Payload().(*payload.CRInfo).Signature = randomSignature()
	err, _ = txn.SpecialContextCheck()
	txn.Payload().(*payload.CRInfo).Signature = signatature
	s.EqualError(err, "transaction validate error: payload content invalid:[Validation], Verify failed.")

	// Give a mismatching deposit address
	outPuts := txn.Outputs()
	txn.SetOutputs([]*common2.Output{{
		AssetID:     common.Uint256{},
		Value:       5000 * 100000000,
		OutputLock:  0,
		ProgramHash: *hash2,
		Payload:     new(outputpayload.DefaultOutput),
	}})
	err, _ = txn.SpecialContextCheck()
	txn.SetOutputs(outPuts)
	s.EqualError(err, "transaction validate error: payload content invalid:deposit address does not match the code in payload")

	// Give a insufficient deposit coin
	txn.SetOutputs([]*common2.Output{{
		AssetID:     common.Uint256{},
		Value:       4000 * 100000000,
		OutputLock:  0,
		ProgramHash: *hash1,
		Payload:     new(outputpayload.DefaultOutput),
	}})
	err, _ = txn.SpecialContextCheck()
	txn.SetOutputs(outPuts)
	s.EqualError(err, "transaction validate error: payload content invalid:CR deposit amount is insufficient")

	// Multi deposit addresses
	txn.SetOutputs([]*common2.Output{
		{
			AssetID:     common.Uint256{},
			Value:       5000 * 100000000,
			OutputLock:  0,
			ProgramHash: *hash1,
			Payload:     new(outputpayload.DefaultOutput),
		},
		{
			AssetID:     common.Uint256{},
			Value:       5000 * 100000000,
			OutputLock:  0,
			ProgramHash: *hash1,
			Payload:     new(outputpayload.DefaultOutput),
		}})
	err, _ = txn.SpecialContextCheck()
	txn.SetOutputs(outPuts)
	s.EqualError(err, "transaction validate error: payload content invalid:there must be only one deposit address in outputs")

	// Check correct register CR transaction with multi sign code.
	txn = s.getMultiSigRegisterCRTx(
		[]string{publicKeyStr1, publicKeyStr2, publicKeyStr3},
		[]string{privateKeyStr1, privateKeyStr2, privateKeyStr3}, nickName1)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CR not support multi sign code")

	txn = s.getMultiSigRegisterCRTx(
		[]string{publicKeyStr1, publicKeyStr2, publicKeyStr3},
		[]string{privateKeyStr1, privateKeyStr2}, nickName1)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CR not support multi sign code")

	txn = s.getMultiSigRegisterCRTx(
		[]string{publicKeyStr1, publicKeyStr2, publicKeyStr3},
		[]string{privateKeyStr1}, nickName1)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CR not support multi sign code")

	//check register cr with CRInfoDIDVersion
	txn2 := s.getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1,
		payload.CRInfoDIDVersion, &common.Uint168{1, 2, 3})
	txn2 = CreateTransactionByType(txn2, s.Chain)
	txn2.SetParameters(&transaction.TransactionParameters{
		Transaction: txn2,
		BlockHeight: registerCRByDIDHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn2.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid did address")
	did2, _ := blockchain.GetDIDFromCode(code2)
	txn2 = s.getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1,
		payload.CRInfoDIDVersion, did2)
	txn2 = CreateTransactionByType(txn2, s.Chain)
	txn2.SetParameters(&transaction.TransactionParameters{
		Transaction: txn2,
		BlockHeight: registerCRByDIDHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn2.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid did address")

	did1, _ := blockchain.GetDIDFromCode(code1)
	txn2 = s.getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1,
		payload.CRInfoDIDVersion, did1)
	txn2 = CreateTransactionByType(txn2, s.Chain)
	txn2.SetParameters(&transaction.TransactionParameters{
		Transaction: txn2,
		BlockHeight: registerCRByDIDHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn2.SpecialContextCheck()
	s.NoError(err)
}

func getDepositAddress(publicKeyStr string) (*common.Uint168, error) {
	publicKey, _ := common.HexStringToBytes(publicKeyStr)
	hash, err := contract.PublicKeyToDepositProgramHash(publicKey)
	if err != nil {
		return nil, err
	}
	return hash, nil
}

func (s *txValidatorTestSuite) getRegisterCRTx(publicKeyStr, privateKeyStr,
	nickName string, payloadVersion byte, did *common.Uint168) interfaces.Transaction {

	publicKeyStr1 := publicKeyStr
	privateKeyStr1 := privateKeyStr
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)

	code1 := getCodeByPubKeyStr(publicKeyStr1)
	ct1, _ := contract.CreateCRIDContractByCode(code1)
	cid1 := ct1.ToProgramHash()

	hash1, _ := contract.PublicKeyToDepositProgramHash(publicKey1)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.RegisterCR,
		payloadVersion,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	crInfoPayload := &payload.CRInfo{
		Code:     code1,
		CID:      *cid1,
		DID:      *did,
		NickName: nickName,
		Url:      "http://www.elastos_test.com",
		Location: 1,
	}
	signBuf := new(bytes.Buffer)
	crInfoPayload.SerializeUnsigned(signBuf, payloadVersion)
	rcSig1, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crInfoPayload.Signature = rcSig1
	txn.SetPayload(crInfoPayload)

	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr1),
		Parameter: nil,
	}})

	txn.SetOutputs([]*common2.Output{{
		AssetID:     common.Uint256{},
		Value:       5000 * 100000000,
		OutputLock:  0,
		ProgramHash: *hash1,
		Type:        0,
		Payload:     new(outputpayload.DefaultOutput),
	}})
	return txn
}

func (s *txValidatorTestSuite) getMultiSigRegisterCRTx(
	publicKeyStrs, privateKeyStrs []string, nickName string) interfaces.Transaction {

	var publicKeys []*crypto.PublicKey
	for _, publicKeyStr := range publicKeyStrs {
		publicKeyBytes, _ := hex.DecodeString(publicKeyStr)
		publicKey, _ := crypto.DecodePoint(publicKeyBytes)
		publicKeys = append(publicKeys, publicKey)
	}

	multiCode, _ := contract.CreateMultiSigRedeemScript(len(publicKeys)*2/3, publicKeys)

	ctDID, _ := contract.CreateCRIDContractByCode(multiCode)
	cid := ctDID.ToProgramHash()

	ctDeposit, _ := contract.CreateDepositContractByCode(multiCode)
	deposit := ctDeposit.ToProgramHash()

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.RegisterCR,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	crInfoPayload := &payload.CRInfo{
		Code:     multiCode,
		CID:      *cid,
		NickName: nickName,
		Url:      "http://www.elastos_test.com",
		Location: 1,
	}

	signBuf := new(bytes.Buffer)
	crInfoPayload.SerializeUnsigned(signBuf, payload.CRInfoVersion)
	for _, privateKeyStr := range privateKeyStrs {
		privateKeyBytes, _ := hex.DecodeString(privateKeyStr)
		sig, _ := crypto.Sign(privateKeyBytes, signBuf.Bytes())
		crInfoPayload.Signature = append(crInfoPayload.Signature, byte(len(sig)))
		crInfoPayload.Signature = append(crInfoPayload.Signature, sig...)
	}

	txn.SetPayload(crInfoPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      multiCode,
		Parameter: nil,
	}})
	txn.SetOutputs([]*common2.Output{{
		AssetID:     common.Uint256{},
		Value:       5000 * 100000000,
		OutputLock:  0,
		ProgramHash: *deposit,
		Type:        0,
		Payload:     new(outputpayload.DefaultOutput),
	}})
	return txn
}

func (s *txValidatorTestSuite) getUpdateCRTx(publicKeyStr, privateKeyStr, nickName string) interfaces.Transaction {

	publicKeyStr1 := publicKeyStr
	privateKeyStr1 := privateKeyStr
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)
	code1 := getCodeByPubKeyStr(publicKeyStr1)
	ct1, _ := contract.CreateCRIDContractByCode(code1)
	cid1 := ct1.ToProgramHash()

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.UpdateCR,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	crInfoPayload := &payload.CRInfo{
		Code:     code1,
		CID:      *cid1,
		NickName: nickName,
		Url:      "http://www.elastos_test.com",
		Location: 1,
	}
	signBuf := new(bytes.Buffer)
	err := crInfoPayload.SerializeUnsigned(signBuf, payload.CRInfoVersion)
	s.NoError(err)
	rcSig1, err := crypto.Sign(privateKey1, signBuf.Bytes())
	s.NoError(err)
	crInfoPayload.Signature = rcSig1
	txn.SetPayload(crInfoPayload)

	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr1),
		Parameter: nil,
	}})
	return txn
}

func (s *txValidatorTestSuite) getUnregisterCRTx(publicKeyStr, privateKeyStr string) interfaces.Transaction {

	publicKeyStr1 := publicKeyStr
	privateKeyStr1 := privateKeyStr
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)

	code1 := getCodeByPubKeyStr(publicKeyStr1)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.UnregisterCR,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	unregisterCRPayload := &payload.UnregisterCR{
		CID: *getCID(code1),
	}
	signBuf := new(bytes.Buffer)
	err := unregisterCRPayload.SerializeUnsigned(signBuf, payload.UnregisterCRVersion)
	s.NoError(err)
	rcSig1, err := crypto.Sign(privateKey1, signBuf.Bytes())
	s.NoError(err)
	unregisterCRPayload.Signature = rcSig1
	txn.SetPayload(unregisterCRPayload)

	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr1),
		Parameter: nil,
	}})
	return txn
}

func (s *txValidatorTestSuite) getCRMember(publicKeyStr, privateKeyStr, nickName string) *crstate.CRMember {
	publicKeyStr1 := publicKeyStr
	privateKeyStr1 := privateKeyStr
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)
	code1 := getCodeByPubKeyStr(publicKeyStr1)
	did1, _ := blockchain.GetDIDFromCode(code1)

	crInfoPayload := payload.CRInfo{
		Code:     code1,
		DID:      *did1,
		NickName: nickName,
		Url:      "http://www.elastos_test.com",
		Location: 1,
	}
	signBuf := new(bytes.Buffer)
	crInfoPayload.SerializeUnsigned(signBuf, payload.CRInfoVersion)
	rcSig1, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crInfoPayload.Signature = rcSig1

	return &crstate.CRMember{
		Info: crInfoPayload,
	}
}

func (s *txValidatorTestSuite) getSecretaryGeneralCRCProposalTx(ownerPublicKeyStr, ownerPrivateKeyStr,
	crPublicKeyStr, crPrivateKeyStr, secretaryPublicKeyStr, secretaryPrivateKeyStr string) interfaces.Transaction {

	ownerPublicKey, _ := common.HexStringToBytes(ownerPublicKeyStr)
	ownerPrivateKey, _ := common.HexStringToBytes(ownerPrivateKeyStr)

	secretaryPublicKey, _ := common.HexStringToBytes(secretaryPublicKeyStr)
	secretaryGeneralDID, _ := blockchain.GetDiDFromPublicKey(secretaryPublicKey)
	secretaryGeneralPrivateKey, _ := common.HexStringToBytes(secretaryPrivateKeyStr)

	crPrivateKey, _ := common.HexStringToBytes(crPrivateKeyStr)
	crCode := getCodeByPubKeyStr(crPublicKeyStr)

	draftData := randomBytes(10)
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposal,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	recipient := *randomUint168()
	recipient[0] = uint8(contract.PrefixStandard)
	crDID, _ := blockchain.GetDIDFromCode(crCode)
	crcProposalPayload := &payload.CRCProposal{
		ProposalType:              payload.SecretaryGeneral,
		CategoryData:              "111",
		OwnerPublicKey:            ownerPublicKey,
		DraftHash:                 common.Hash(draftData),
		SecretaryGeneralPublicKey: secretaryPublicKey,
		SecretaryGeneralDID:       *secretaryGeneralDID,
		CRCouncilMemberDID:        *crDID,
	}

	signBuf := new(bytes.Buffer)
	crcProposalPayload.SerializeUnsigned(signBuf, payload.CRCProposalVersion)
	sig, _ := crypto.Sign(ownerPrivateKey, signBuf.Bytes())
	crcProposalPayload.Signature = sig

	secretaryGeneralSig, _ := crypto.Sign(secretaryGeneralPrivateKey, signBuf.Bytes())
	crcProposalPayload.SecretaryGeneraSignature = secretaryGeneralSig

	common.WriteVarBytes(signBuf, sig)
	common.WriteVarBytes(signBuf, secretaryGeneralSig)
	crcProposalPayload.CRCouncilMemberDID.Serialize(signBuf)
	crSig, _ := crypto.Sign(crPrivateKey, signBuf.Bytes())
	crcProposalPayload.CRCouncilMemberSignature = crSig

	txn.SetPayload(crcProposalPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(ownerPublicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func (s *txValidatorTestSuite) getCRCProposalTx(publicKeyStr, privateKeyStr,
	crPublicKeyStr, crPrivateKeyStr string) interfaces.Transaction {

	publicKey1, _ := common.HexStringToBytes(publicKeyStr)
	privateKey1, _ := common.HexStringToBytes(privateKeyStr)

	privateKey2, _ := common.HexStringToBytes(crPrivateKeyStr)
	code2 := getCodeByPubKeyStr(crPublicKeyStr)

	draftData := randomBytes(10)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposal,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	recipient := *randomUint168()
	recipient[0] = uint8(contract.PrefixStandard)
	did2, _ := blockchain.GetDIDFromCode(code2)
	crcProposalPayload := &payload.CRCProposal{
		ProposalType:       payload.Normal,
		OwnerPublicKey:     publicKey1,
		CRCouncilMemberDID: *did2,
		DraftHash:          common.Hash(draftData),
		Budgets:            createBudgets(3),
		Recipient:          recipient,
	}

	signBuf := new(bytes.Buffer)
	crcProposalPayload.SerializeUnsigned(signBuf, payload.CRCProposalVersion)
	sig, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crcProposalPayload.Signature = sig

	common.WriteVarBytes(signBuf, sig)
	crcProposalPayload.CRCouncilMemberDID.Serialize(signBuf)
	crSig, _ := crypto.Sign(privateKey2, signBuf.Bytes())
	crcProposalPayload.CRCouncilMemberSignature = crSig

	txn.SetPayload(crcProposalPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func (s *txValidatorTestSuite) createSpecificStatusProposal(publicKey1, publicKey2 []byte, height uint32,
	status crstate.ProposalStatus, proposalType payload.CRCProposalType) (*crstate.ProposalState, *payload.CRCProposal) {
	draftData := randomBytes(10)
	recipient := *randomUint168()
	recipient[0] = uint8(contract.PrefixStandard)
	code2 := getCodeByPubKeyStr(hex.EncodeToString(publicKey2))
	CRCouncilMemberDID, _ := blockchain.GetDIDFromCode(code2)
	proposal := &payload.CRCProposal{
		ProposalType:       proposalType,
		OwnerPublicKey:     publicKey1,
		CRCouncilMemberDID: *CRCouncilMemberDID,
		DraftHash:          common.Hash(draftData),
		Budgets:            createBudgets(3),
		Recipient:          recipient,
	}
	budgetsStatus := make(map[uint8]crstate.BudgetStatus)
	for _, budget := range proposal.Budgets {
		if budget.Type == payload.Imprest {
			budgetsStatus[budget.Stage] = crstate.Withdrawable
			continue
		}
		budgetsStatus[budget.Stage] = crstate.Unfinished
	}
	proposalState := &crstate.ProposalState{
		Status:              status,
		Proposal:            proposal.ToProposalInfo(0),
		TxHash:              common.Hash(randomBytes(10)),
		CRVotes:             map[common.Uint168]payload.VoteResult{},
		VotersRejectAmount:  common.Fixed64(0),
		RegisterHeight:      height,
		VoteStartHeight:     0,
		WithdrawnBudgets:    make(map[uint8]common.Fixed64),
		WithdrawableBudgets: make(map[uint8]common.Fixed64),
		BudgetsStatus:       budgetsStatus,
		FinalPaymentStatus:  false,
		TrackingCount:       0,
		TerminatedHeight:    0,
		ProposalOwner:       proposal.OwnerPublicKey,
	}
	return proposalState, proposal
}

func (s *txValidatorTestSuite) getCRCCloseProposalTxWithHash(publicKeyStr, privateKeyStr,
	crPublicKeyStr, crPrivateKeyStr string, closeProposalHash common.Uint256) interfaces.Transaction {
	draftData := randomBytes(10)

	privateKey1, _ := common.HexStringToBytes(privateKeyStr)
	publicKey1, _ := common.HexStringToBytes(publicKeyStr)

	privateKey2, _ := common.HexStringToBytes(crPrivateKeyStr)
	//publicKey2, _ := common.HexStringToBytes(crPublicKeyStr)
	code2 := getCodeByPubKeyStr(crPublicKeyStr)
	//did2, _ := getDIDFromCode(code2)

	//draftData := randomBytes(10)
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposal,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	CRCouncilMemberDID, _ := blockchain.GetDIDFromCode(code2)
	crcProposalPayload := &payload.CRCProposal{
		ProposalType:       payload.CloseProposal,
		OwnerPublicKey:     publicKey1,
		CRCouncilMemberDID: *CRCouncilMemberDID,
		DraftHash:          common.Hash(draftData),
		TargetProposalHash: closeProposalHash,
	}

	signBuf := new(bytes.Buffer)
	crcProposalPayload.SerializeUnsigned(signBuf, payload.CRCProposalVersion)
	sig, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crcProposalPayload.Signature = sig

	common.WriteVarBytes(signBuf, sig)
	crcProposalPayload.CRCouncilMemberDID.Serialize(signBuf)
	crSig, _ := crypto.Sign(privateKey2, signBuf.Bytes())
	crcProposalPayload.CRCouncilMemberSignature = crSig

	txn.SetPayload(crcProposalPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func (s *txValidatorTestSuite) getCRCRegisterSideChainProposalTx(publicKeyStr, privateKeyStr,
	crPublicKeyStr, crPrivateKeyStr string) interfaces.Transaction {

	normalPrivateKey, _ := common.HexStringToBytes(privateKeyStr)
	normalPublicKey, _ := common.HexStringToBytes(publicKeyStr)
	crPrivateKey, _ := common.HexStringToBytes(crPrivateKeyStr)

	draftData := randomBytes(10)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposal,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	CRCouncilMemberDID, _ := blockchain.GetDIDFromCode(getCodeByPubKeyStr(crPublicKeyStr))
	crcProposalPayload := &payload.CRCProposal{
		ProposalType:       payload.RegisterSideChain,
		OwnerPublicKey:     normalPublicKey,
		CRCouncilMemberDID: *CRCouncilMemberDID,
		DraftHash:          common.Hash(draftData),
		SideChainInfo: payload.SideChainInfo{
			SideChainName:   "NEO",
			MagicNumber:     100,
			GenesisHash:     *randomUint256(),
			ExchangeRate:    100000000,
			EffectiveHeight: 100000,
		},
	}

	signBuf := new(bytes.Buffer)
	crcProposalPayload.SerializeUnsigned(signBuf, payload.CRCProposalVersion)

	sig, _ := crypto.Sign(normalPrivateKey, signBuf.Bytes())
	crcProposalPayload.Signature = sig

	common.WriteVarBytes(signBuf, sig)
	crcProposalPayload.CRCouncilMemberDID.Serialize(signBuf)
	crSig, _ := crypto.Sign(crPrivateKey, signBuf.Bytes())
	crcProposalPayload.CRCouncilMemberSignature = crSig

	txn.SetPayload(crcProposalPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func (s *txValidatorTestSuite) getCRCCloseProposalTx(publicKeyStr, privateKeyStr,
	crPublicKeyStr, crPrivateKeyStr string) interfaces.Transaction {

	privateKey1, _ := common.HexStringToBytes(privateKeyStr)

	privateKey2, _ := common.HexStringToBytes(crPrivateKeyStr)
	publicKey2, _ := common.HexStringToBytes(crPublicKeyStr)
	code2 := getCodeByPubKeyStr(crPublicKeyStr)
	//did2, _ := getDIDFromCode(code2)

	draftData := randomBytes(10)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposal,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	CRCouncilMemberDID, _ := blockchain.GetDIDFromCode(code2)
	crcProposalPayload := &payload.CRCProposal{
		ProposalType:       payload.CloseProposal,
		OwnerPublicKey:     publicKey2,
		CRCouncilMemberDID: *CRCouncilMemberDID,
		DraftHash:          common.Hash(draftData),
		TargetProposalHash: common.Hash(randomBytes(10)),
	}

	signBuf := new(bytes.Buffer)
	crcProposalPayload.SerializeUnsigned(signBuf, payload.CRCProposalVersion)
	sig, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crcProposalPayload.Signature = sig

	common.WriteVarBytes(signBuf, sig)
	crcProposalPayload.CRCouncilMemberDID.Serialize(signBuf)
	crSig, _ := crypto.Sign(privateKey2, signBuf.Bytes())
	crcProposalPayload.CRCouncilMemberSignature = crSig

	txn.SetPayload(crcProposalPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func randomName(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[mrand.Intn(len(charset))]
	}
	return string(b)
}

func (s *txValidatorTestSuite) getCRCReceivedCustomIDProposalTx(publicKeyStr, privateKeyStr,
	crPublicKeyStr, crPrivateKeyStr string, receivedList []string) interfaces.Transaction {

	privateKey1, _ := common.HexStringToBytes(privateKeyStr)
	publicKey1, _ := common.HexStringToBytes(publicKeyStr)
	privateKey2, _ := common.HexStringToBytes(crPrivateKeyStr)
	//publicKey2, _ := common.HexStringToBytes(crPublicKeyStr)

	code2 := getCodeByPubKeyStr(crPublicKeyStr)
	//did2, _ := getDIDFromCode(code2)

	draftData := randomBytes(10)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposal,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	CRCouncilMemberDID, _ := blockchain.GetDIDFromCode(code2)
	crcProposalPayload := &payload.CRCProposal{
		ProposalType:         payload.ReceiveCustomID,
		OwnerPublicKey:       publicKey1,
		CRCouncilMemberDID:   *CRCouncilMemberDID,
		DraftHash:            common.Hash(draftData),
		ReceivedCustomIDList: receivedList,
		ReceiverDID:          *randomUint168(),
	}

	signBuf := new(bytes.Buffer)
	crcProposalPayload.SerializeUnsigned(signBuf, payload.CRCProposalVersion)
	sig, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crcProposalPayload.Signature = sig

	common.WriteVarBytes(signBuf, sig)
	crcProposalPayload.CRCouncilMemberDID.Serialize(signBuf)
	crSig, _ := crypto.Sign(privateKey2, signBuf.Bytes())
	crcProposalPayload.CRCouncilMemberSignature = crSig

	txn.SetPayload(crcProposalPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func (s *txValidatorTestSuite) getCRCReservedCustomIDProposalTx(publicKeyStr, privateKeyStr,
	crPublicKeyStr, crPrivateKeyStr string) interfaces.Transaction {

	privateKey1, _ := common.HexStringToBytes(privateKeyStr)

	privateKey2, _ := common.HexStringToBytes(crPrivateKeyStr)
	publicKey2, _ := common.HexStringToBytes(crPublicKeyStr)
	code2 := getCodeByPubKeyStr(crPublicKeyStr)
	//did2, _ := getDIDFromCode(code2)

	draftData := randomBytes(10)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposal,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	CRCouncilMemberDID, _ := blockchain.GetDIDFromCode(code2)
	crcProposalPayload := &payload.CRCProposal{
		ProposalType:         payload.ReserveCustomID,
		OwnerPublicKey:       publicKey2,
		CRCouncilMemberDID:   *CRCouncilMemberDID,
		DraftHash:            common.Hash(draftData),
		ReservedCustomIDList: []string{randomName(3), randomName(3), randomName(3)},
	}

	signBuf := new(bytes.Buffer)
	crcProposalPayload.SerializeUnsigned(signBuf, payload.CRCProposalVersion)
	sig, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crcProposalPayload.Signature = sig

	common.WriteVarBytes(signBuf, sig)
	crcProposalPayload.CRCouncilMemberDID.Serialize(signBuf)
	crSig, _ := crypto.Sign(privateKey2, signBuf.Bytes())
	crcProposalPayload.CRCouncilMemberSignature = crSig

	txn.SetPayload(crcProposalPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func (s *txValidatorTestSuite) TestCheckCRCProposalTrackingTransaction() {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"

	ownerPubKey, _ := common.HexStringToBytes(publicKeyStr1)

	proposalHash := randomUint256()
	recipient := randomUint168()
	votingHeight := config.DefaultParams.CRVotingStartHeight

	// Set secretary general.
	s.Chain.GetCRCommittee().GetProposalManager().SecretaryGeneralPublicKey = publicKeyStr3
	// Check Common tracking tx.
	txn := s.getCRCProposalTrackingTx(payload.Common, *proposalHash, 0,
		publicKeyStr1, privateKeyStr1, "", "",
		publicKeyStr3, privateKeyStr3)

	pld := payload.CRCProposal{
		ProposalType:       0,
		OwnerPublicKey:     ownerPubKey,
		CRCouncilMemberDID: *randomUint168(),
		DraftHash:          *randomUint256(),
		Budgets:            createBudgets(3),
		Recipient:          *recipient,
	}
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[*proposalHash] =
		&crstate.ProposalState{
			Proposal:      pld.ToProposalInfo(0),
			Status:        crstate.VoterAgreed,
			ProposalOwner: ownerPubKey,
		}

	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ := txn.SpecialContextCheck()
	s.NoError(err)

	txn = s.getCRCProposalTrackingTx(payload.Common, *proposalHash, 1,
		publicKeyStr1, privateKeyStr1, publicKeyStr2, privateKeyStr2,
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:stage should assignment zero value")

	txn = s.getCRCProposalTrackingTx(payload.Common, *proposalHash, 0,
		publicKeyStr1, privateKeyStr1, publicKeyStr2, privateKeyStr2,
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:the NewOwnerPublicKey need to be empty")

	// Check Progress tracking tx.
	txn = s.getCRCProposalTrackingTx(payload.Progress, *proposalHash, 1,
		publicKeyStr1, privateKeyStr1, "", "",
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	txn = s.getCRCProposalTrackingTx(payload.Progress, *proposalHash, 1,
		publicKeyStr1, privateKeyStr1, publicKeyStr2, privateKeyStr2,
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:the NewOwnerPublicKey need to be empty")

	// Check Terminated tracking tx.
	txn = s.getCRCProposalTrackingTx(payload.Terminated, *proposalHash, 0,
		publicKeyStr1, privateKeyStr1, "", "",
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	txn = s.getCRCProposalTrackingTx(payload.Terminated, *proposalHash, 1,
		publicKeyStr1, privateKeyStr1, publicKeyStr2, privateKeyStr2,
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:stage should assignment zero value")

	txn = s.getCRCProposalTrackingTx(payload.Terminated, *proposalHash, 0,
		publicKeyStr1, privateKeyStr1, publicKeyStr2, privateKeyStr2,
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:the NewOwnerPublicKey need to be empty")

	// Check ChangeOwner tracking tx.
	txn = s.getCRCProposalTrackingTx(payload.ChangeOwner, *proposalHash, 0,
		publicKeyStr1, privateKeyStr1, publicKeyStr2, privateKeyStr2,
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	txn = s.getCRCProposalTrackingTx(payload.ChangeOwner, *proposalHash, 1,
		publicKeyStr1, privateKeyStr1, publicKeyStr2, privateKeyStr2,
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:stage should assignment zero value")

	txn = s.getCRCProposalTrackingTx(payload.ChangeOwner, *proposalHash, 0,
		publicKeyStr1, privateKeyStr1, "", "",
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid new proposal owner public key")

	// Check invalid proposal hash.
	txn = s.getCRCProposalTrackingTx(payload.Common, *randomUint256(), 0,
		publicKeyStr1, privateKeyStr1, "", "",
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:proposal not exist")

	txn = s.getCRCProposalTrackingTx(payload.Common, *proposalHash, 0,
		publicKeyStr1, privateKeyStr1, "", "",
		publicKeyStr3, privateKeyStr3)

	// Check proposal status is not VoterAgreed.
	pld = payload.CRCProposal{
		ProposalType:       0,
		OwnerPublicKey:     ownerPubKey,
		CRCouncilMemberDID: *randomUint168(),
		DraftHash:          *randomUint256(),
		Budgets:            createBudgets(3),
		Recipient:          *recipient,
	}
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[*proposalHash] =
		&crstate.ProposalState{
			Proposal:         pld.ToProposalInfo(0),
			TerminatedHeight: 100,
			Status:           crstate.VoterCanceled,
			ProposalOwner:    ownerPubKey,
		}
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[*proposalHash].TerminatedHeight = 100
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:proposal status is not VoterAgreed")

	// Check reach max proposal tracking count.
	pld = payload.CRCProposal{
		ProposalType:       0,
		OwnerPublicKey:     ownerPubKey,
		CRCouncilMemberDID: *randomUint168(),
		DraftHash:          *randomUint256(),
		Budgets:            createBudgets(3),
		Recipient:          *recipient,
	}
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[*proposalHash] =
		&crstate.ProposalState{
			Proposal:      pld.ToProposalInfo(0),
			TrackingCount: 128,
			Status:        crstate.VoterAgreed,
			ProposalOwner: ownerPubKey,
		}
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:reached max tracking count")

}

func (s *txValidatorTestSuite) getCRCProposalTrackingTx(
	trackingType payload.CRCProposalTrackingType,
	proposalHash common.Uint256, stage uint8,
	ownerPublicKeyStr, ownerPrivateKeyStr,
	newownerpublickeyStr, newownerprivatekeyStr,
	sgPublicKeyStr, sgPrivateKeyStr string) interfaces.Transaction {

	ownerPublicKey, _ := common.HexStringToBytes(ownerPublicKeyStr)
	ownerPrivateKey, _ := common.HexStringToBytes(ownerPrivateKeyStr)

	newownerpublickey, _ := common.HexStringToBytes(newownerpublickeyStr)
	newownerprivatekey, _ := common.HexStringToBytes(newownerprivatekeyStr)

	sgPrivateKey, _ := common.HexStringToBytes(sgPrivateKeyStr)

	documentData := randomBytes(10)
	opinionHash := randomBytes(10)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposalTracking,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	cPayload := &payload.CRCProposalTracking{
		ProposalTrackingType:        trackingType,
		ProposalHash:                proposalHash,
		Stage:                       stage,
		MessageHash:                 common.Hash(documentData),
		OwnerPublicKey:              ownerPublicKey,
		NewOwnerPublicKey:           newownerpublickey,
		SecretaryGeneralOpinionHash: common.Hash(opinionHash),
	}

	signBuf := new(bytes.Buffer)
	cPayload.SerializeUnsigned(signBuf, payload.CRCProposalTrackingVersion)
	sig, _ := crypto.Sign(ownerPrivateKey, signBuf.Bytes())
	cPayload.OwnerSignature = sig
	common.WriteVarBytes(signBuf, sig)

	if newownerpublickeyStr != "" && newownerprivatekeyStr != "" {
		crSig, _ := crypto.Sign(newownerprivatekey, signBuf.Bytes())
		cPayload.NewOwnerSignature = crSig
		sig = crSig
	} else {
		sig = []byte{}
	}

	common.WriteVarBytes(signBuf, sig)
	signBuf.Write([]byte{byte(cPayload.ProposalTrackingType)})
	cPayload.SecretaryGeneralOpinionHash.Serialize(signBuf)
	crSig, _ := crypto.Sign(sgPrivateKey, signBuf.Bytes())
	cPayload.SecretaryGeneralSignature = crSig

	txn.SetPayload(cPayload)
	return txn
}

func (s *txValidatorTestSuite) TestCheckCRCAppropriationTransaction() {
	// Set CR assets address and CR expenses address.
	s.Chain.GetParams().CRAssetsAddress = *randomUint168()
	s.Chain.GetParams().CRExpensesAddress = *randomUint168()

	// Set CR assets and CRC committee amount.
	s.Chain.GetCRCommittee().CRCFoundationBalance = common.Fixed64(900 * 1e8)
	s.Chain.GetCRCommittee().AppropriationAmount = common.Fixed64(90 * 1e8)
	s.Chain.GetCRCommittee().CRCCommitteeUsedAmount = common.Fixed64(0 * 1e8)

	// Create reference.
	reference := make(map[*common2.Input]common2.Output)
	input := &common2.Input{
		Previous: common2.OutPoint{
			TxID:  *randomUint256(),
			Index: 0,
		},
	}
	refOutput := common2.Output{
		Value:       900 * 1e8,
		ProgramHash: s.Chain.GetParams().CRAssetsAddress,
	}
	refOutputErr := common2.Output{
		Value:       900 * 1e8,
		ProgramHash: *randomUint168(),
	}
	reference[input] = refOutput

	// Create CRC appropriation transaction.
	output1 := &common2.Output{
		Value:       90 * 1e8,
		ProgramHash: s.Chain.GetParams().CRExpensesAddress,
	}
	output2 := &common2.Output{
		Value:       810 * 1e8,
		ProgramHash: s.Chain.GetParams().CRAssetsAddress,
	}
	output1Err := &common2.Output{
		Value:       91 * 1e8,
		ProgramHash: s.Chain.GetParams().CRExpensesAddress,
	}
	output2Err := &common2.Output{
		Value:       809 * 1e8,
		ProgramHash: s.Chain.GetParams().CRAssetsAddress,
	}

	// Check correct transaction.
	s.Chain.GetCRCommittee().NeedAppropriation = true
	txn := s.getCRCAppropriationTx(input, output1, output2)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetReferences(reference)
	err, _ := txn.SpecialContextCheck()
	s.NoError(err)

	// Appropriation transaction already exist.
	s.Chain.GetCRCommittee().NeedAppropriation = false
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:should have no appropriation transaction")

	// Input does not from CR assets address
	s.Chain.GetCRCommittee().NeedAppropriation = true
	reference[input] = refOutputErr
	txn = s.getCRCAppropriationTx(input, output1, output2)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetReferences(reference)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:input does not from CR assets address")

	// Inputs total amount does not equal to outputs total amount.
	reference[input] = refOutput
	txn = s.getCRCAppropriationTx(input, output1, output2Err)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetReferences(reference)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:inputs does not equal to outputs "+
		"amount, inputs:900 outputs:899")

	// Invalid CRC appropriation amount.
	txn = s.getCRCAppropriationTx(input, output1Err, output2Err)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetReferences(reference)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid appropriation amount 91, need to be 90")
}

func (s *txValidatorTestSuite) getCRCAppropriationTx(input *common2.Input,
	output1 *common2.Output, output2 *common2.Output) interfaces.Transaction {
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCAppropriation,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	cPayload := &payload.CRCAppropriation{}
	txn.SetPayload(cPayload)
	txn.SetInputs([]*common2.Input{input})
	txn.SetOutputs([]*common2.Output{output1, output2})

	return txn
}

func (s *txValidatorTestSuite) getCRCProposalRealWithdrawTx(input *common2.Input,
	hashes []common.Uint256, outputs []*common2.Output) interfaces.Transaction {

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposalRealWithdraw,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	cPayload := &payload.CRCProposalRealWithdraw{WithdrawTransactionHashes: hashes}
	txn.SetPayload(cPayload)
	txn.SetInputs([]*common2.Input{input})
	txn.SetOutputs(outputs)
	return txn
}

func (s *txValidatorTestSuite) TestCrInfoSanityCheck() {
	publicKeyStr1 := "03c77af162438d4b7140f8544ad6523b9734cca9c7a62476d54ed5d1bddc7a39c3"
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)

	pk1, _ := crypto.DecodePoint(publicKey1)
	ct1, _ := contract.CreateStandardContract(pk1)
	hash1, _ := contract.PublicKeyToDepositProgramHash(publicKey1)

	rcPayload := &payload.CRInfo{
		Code:     ct1.Code,
		CID:      *hash1,
		NickName: "nickname 1",
		Url:      "http://www.elastos_test.com",
		Location: 1,
	}

	rcSignBuf := new(bytes.Buffer)
	err := rcPayload.SerializeUnsigned(rcSignBuf, payload.CRInfoVersion)
	s.NoError(err)

	privateKeyStr1 := "7638c2a799d93185279a4a6ae84a5b76bd89e41fa9f465d9ae9b2120533983a1"
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)
	rcSig1, err := crypto.Sign(privateKey1, rcSignBuf.Bytes())
	s.NoError(err)

	//test ok
	rcPayload.Signature = rcSig1
	err = blockchain.CrInfoSanityCheck(rcPayload, payload.CRInfoVersion)
	s.NoError(err)

	//invalid code
	rcPayload.Code = []byte{1, 2, 3, 4, 5}
	err = blockchain.CrInfoSanityCheck(rcPayload, payload.CRInfoVersion)
	s.EqualError(err, "invalid code")

	//todo CHECKMULTISIG
}

func (s *txValidatorTestSuite) TestCheckUpdateCRTransaction() {

	// Generate a UpdateCR CR transaction
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"

	nickName1 := "nickname 1"
	nickName2 := "nickname 2"
	nickName3 := "nickname 3"

	votingHeight := config.DefaultParams.CRVotingStartHeight
	//
	//registe an cr to update
	registerCRTxn1 := s.getRegisterCRTx(publicKeyStr1, privateKeyStr1,
		nickName1, payload.CRInfoVersion, &common.Uint168{})
	registerCRTxn2 := s.getRegisterCRTx(publicKeyStr2, privateKeyStr2,
		nickName2, payload.CRInfoDIDVersion, &common.Uint168{})

	s.CurrentHeight = s.Chain.GetParams().CRVotingStartHeight + 1
	s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams()))
	s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
		GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
		GetUTXO:                          s.Chain.GetDB().GetFFLDB().GetUTXO,
		GetHeight:                        func() uint32 { return s.CurrentHeight },
		CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
	})
	block := &types.Block{
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
		},
		Header: common2.Header{Height: s.CurrentHeight},
	}
	s.Chain.GetCRCommittee().ProcessBlock(block, nil)

	//ok nothing wrong
	hash2, err := getDepositAddress(publicKeyStr2)
	txn := s.getUpdateCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	// Give an invalid NickName length 0 in payload
	nickName := txn.Payload().(*payload.CRInfo).NickName
	txn.Payload().(*payload.CRInfo).NickName = ""
	err, _ = txn.SpecialContextCheck()
	txn.Payload().(*payload.CRInfo).NickName = nickName
	s.EqualError(err, "transaction validate error: payload content invalid:field NickName has invalid string length")

	// Give an invalid NickName length more than 100 in payload
	txn.Payload().(*payload.CRInfo).NickName = "012345678901234567890123456789012345678901234567890" +
		"12345678901234567890123456789012345678901234567890123456789"
	err, _ = txn.SpecialContextCheck()
	txn.Payload().(*payload.CRInfo).NickName = nickName
	s.EqualError(err, "transaction validate error: payload content invalid:field NickName has invalid string length")

	// Give an invalid url length more than 100 in payload
	url := txn.Payload().(*payload.CRInfo).Url
	txn.Payload().(*payload.CRInfo).Url = "012345678901234567890123456789012345678901234567890" +
		"12345678901234567890123456789012345678901234567890123456789"
	err, _ = txn.SpecialContextCheck()
	txn.Payload().(*payload.CRInfo).Url = url
	s.EqualError(err, "transaction validate error: payload content invalid:field Url has invalid string length")

	// Give an invalid code in payload
	code := txn.Payload().(*payload.CRInfo).Code
	txn.Payload().(*payload.CRInfo).Code = []byte{1, 2, 3, 4, 5}
	err, _ = txn.SpecialContextCheck()
	txn.Payload().(*payload.CRInfo).Code = code
	s.EqualError(err, "transaction validate error: payload content invalid:invalid cid address")

	// Give an invalid CID in payload
	cid := txn.Payload().(*payload.CRInfo).CID
	txn.Payload().(*payload.CRInfo).CID = common.Uint168{1, 2, 3}
	err, _ = txn.SpecialContextCheck()
	txn.Payload().(*payload.CRInfo).CID = cid
	s.EqualError(err, "transaction validate error: payload content invalid:invalid cid address")

	// Give a mismatching code and CID in payload
	txn.Payload().(*payload.CRInfo).CID = *hash2
	err, _ = txn.SpecialContextCheck()
	txn.Payload().(*payload.CRInfo).CID = cid
	s.EqualError(err, "transaction validate error: payload content invalid:invalid cid address")

	// Invalidates the signature in payload
	signatur := txn.Payload().(*payload.CRInfo).Signature
	txn.Payload().(*payload.CRInfo).Signature = randomSignature()
	err, _ = txn.SpecialContextCheck()
	txn.Payload().(*payload.CRInfo).Signature = signatur
	s.EqualError(err, "transaction validate error: payload content invalid:[Validation], Verify failed.")

	//not in vote Period lower
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: config.DefaultParams.CRVotingStartHeight - 1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:should create tx during voting period")

	// set RegisterCRByDIDHeight after CRCommitteeStartHeight
	s.Chain.GetParams().RegisterCRByDIDHeight = config.DefaultParams.CRCommitteeStartHeight + 10

	//not in vote Period lower upper c.params.CRCommitteeStartHeight
	s.Chain.GetCRCommittee().InElectionPeriod = true
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: config.DefaultParams.CRCommitteeStartHeight + 1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:should create tx during voting period")

	//updating unknown CR
	txn3 := s.getUpdateCRTx(publicKeyStr3, privateKeyStr3, nickName3)
	txn3 = CreateTransactionByType(txn3, s.Chain)
	txn3.SetParameters(&transaction.TransactionParameters{
		Transaction: txn3,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn3.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:updating unknown CR")

	//nick name already exist
	txn1Copy := s.getUpdateCRTx(publicKeyStr1, privateKeyStr1, nickName2)
	txn1Copy = CreateTransactionByType(txn1Copy, s.Chain)
	txn1Copy.SetParameters(&transaction.TransactionParameters{
		Transaction: txn1Copy,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn1Copy.SpecialContextCheck()
	str := fmt.Sprintf("transaction validate error: payload content invalid:nick name %s already exist", nickName2)
	s.EqualError(err, str)

}

func (s *txValidatorTestSuite) TestCheckCRCProposalRealWithdrawTransaction() {
	// Set CR expenses address.
	s.Chain.GetParams().CRExpensesAddress = *randomUint168()

	// Set WithdrawableTxInfo
	withdrawTransactionHash1 := *randomUint256()
	recipient1 := *randomUint168()
	withdrawTransactionHash2 := *randomUint256()
	recipient2 := *randomUint168()
	wtHashes := make(map[common.Uint256]common2.OutputInfo, 0)
	wtHashes[withdrawTransactionHash1] = common2.OutputInfo{
		Recipient: recipient1,
		Amount:    10 * 1e8,
	}
	wtHashes[withdrawTransactionHash2] = common2.OutputInfo{
		Recipient: recipient2,
		Amount:    9 * 1e8,
	}
	s.Chain.GetCRCommittee().GetProposalManager().WithdrawableTxInfo = wtHashes

	// Create reference.
	reference := make(map[*common2.Input]common2.Output)
	input := &common2.Input{
		Previous: common2.OutPoint{
			TxID:  *randomUint256(),
			Index: 0,
		},
	}
	refOutput := common2.Output{
		Value:       20 * 1e8,
		ProgramHash: s.Chain.GetParams().CRExpensesAddress,
	}
	reference[input] = refOutput

	// create outputs
	output1 := &common2.Output{
		Value:       10*1e8 - 10000,
		ProgramHash: recipient1,
	}
	output2 := &common2.Output{
		Value:       9*1e8 - 10000,
		ProgramHash: recipient2,
	}
	output3 := &common2.Output{
		Value:       1 * 1e8,
		ProgramHash: s.Chain.GetParams().CRExpensesAddress,
	}
	output1Err := &common2.Output{
		Value:       10 * 1e8,
		ProgramHash: recipient1,
	}
	output2Err := &common2.Output{
		Value:       9*1e8 - 10000,
		ProgramHash: recipient1,
	}
	output3Err := &common2.Output{
		Value:       1 * 1e8,
		ProgramHash: recipient1,
	}

	// check transaction
	txn := s.getCRCProposalRealWithdrawTx(input,
		[]common.Uint256{},
		[]*common2.Output{output1, output2})
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetReferences(reference)
	err, _ := txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid real withdraw transaction hashes count")

	txn = s.getCRCProposalRealWithdrawTx(input,
		[]common.Uint256{withdrawTransactionHash1, withdrawTransactionHash2},
		[]*common2.Output{output1Err, output2, output3})
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetReferences(reference)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid real withdraw output amount:10, need to be:9.99990000")

	txn = s.getCRCProposalRealWithdrawTx(input,
		[]common.Uint256{withdrawTransactionHash1, withdrawTransactionHash2},
		[]*common2.Output{output1, output2Err, output3})
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetReferences(reference)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid real withdraw output address")

	txn = s.getCRCProposalRealWithdrawTx(input,
		[]common.Uint256{withdrawTransactionHash1, withdrawTransactionHash2},
		[]*common2.Output{output1, output1, output3Err})
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetReferences(reference)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:last output is invalid")

	txn = s.getCRCProposalRealWithdrawTx(input,
		[]common.Uint256{withdrawTransactionHash1, withdrawTransactionHash1},
		[]*common2.Output{output1, output1, output3})
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetReferences(reference)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:duplicated real withdraw transactions hash")

	txn = s.getCRCProposalRealWithdrawTx(input,
		[]common.Uint256{withdrawTransactionHash1, withdrawTransactionHash2},
		[]*common2.Output{output1, output2, output3})
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetReferences(reference)
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)
}

func (s *txValidatorTestSuite) TestCheckUnregisterCRTransaction() {

	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"

	votingHeight := config.DefaultParams.CRVotingStartHeight
	nickName1 := "nickname 1"

	//register a cr to unregister
	registerCRTxn := s.getRegisterCRTx(publicKeyStr1, privateKeyStr1,
		nickName1, payload.CRInfoVersion, &common.Uint168{})
	s.CurrentHeight = 1
	s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams()))
	s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
		GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
		GetUTXO:                          s.Chain.GetDB().GetFFLDB().GetUTXO,
		GetHeight:                        func() uint32 { return s.CurrentHeight },
		CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
	})
	block := &types.Block{
		Transactions: []interfaces.Transaction{
			registerCRTxn,
		},
		Header: common2.Header{Height: votingHeight},
	}
	s.Chain.GetCRCommittee().ProcessBlock(block, nil)
	//ok
	txn := s.getUnregisterCRTx(publicKeyStr1, privateKeyStr1)
	txn = CreateTransactionByType(txn, s.Chain)
	err := txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	//invalid payload need unregisterCR pass registerCr
	registerTx := s.getRegisterCRTx(publicKeyStr1, privateKeyStr1,
		nickName1, payload.CRInfoVersion, &common.Uint168{})
	registerTx = CreateTransactionByType(registerTx, s.Chain)
	err = registerTx.SetParameters(&transaction.TransactionParameters{
		Transaction: registerTx,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = registerTx.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:nick name nickname 1 already inuse")

	//not in vote Period lower
	err = txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: config.DefaultParams.CRVotingStartHeight - 1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:should create tx during voting period")

	//not in vote Period lower upper c.params.CRCommitteeStartHeight
	s.Chain.GetCRCommittee().InElectionPeriod = true
	config.DefaultParams.DPoSV2StartHeight = 2000000
	err = txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: config.DefaultParams.CRCommitteeStartHeight + 1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:should create tx during voting period")

	//unregister unknown CR
	txn2 := s.getUnregisterCRTx(publicKeyStr2, privateKeyStr2)
	txn2 = CreateTransactionByType(txn2, s.Chain)
	err = txn2.SetParameters(&transaction.TransactionParameters{
		Transaction: txn2,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn2.SpecialContextCheck()

	s.EqualError(err, "transaction validate error: payload content invalid:unregister unknown CR")

	//wrong signature
	txn.Payload().(*payload.UnregisterCR).Signature = randomSignature()
	err = txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:[Validation], Verify failed.")
}

func (s *txValidatorTestSuite) getCRCProposalReviewTx(crPublicKeyStr,
	crPrivateKeyStr string) interfaces.Transaction {

	privateKey1, _ := common.HexStringToBytes(crPrivateKeyStr)
	code := getCodeByPubKeyStr(crPublicKeyStr)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposalReview,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	did, _ := blockchain.GetDIDFromCode(code)
	crcProposalReviewPayload := &payload.CRCProposalReview{
		ProposalHash: *randomUint256(),
		VoteResult:   payload.Approve,
		DID:          *did,
	}

	signBuf := new(bytes.Buffer)
	crcProposalReviewPayload.SerializeUnsigned(signBuf, payload.CRCProposalReviewVersion)
	sig, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crcProposalReviewPayload.Signature = sig

	txn.SetPayload(crcProposalReviewPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(crPublicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func (s *txValidatorTestSuite) TestCheckCRCProposalReviewTransaction() {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	tenureHeight := config.DefaultParams.CRCommitteeStartHeight
	nickName1 := "nickname 1"

	fmt.Println("getcode ", getCodeHexStr("02e23f70b9b967af35571c32b1442d787c180753bbed5cd6e7d5a5cfe75c7fc1ff"))

	member1 := s.getCRMember(publicKeyStr1, privateKeyStr1, nickName1)
	s.Chain.GetCRCommittee().Members[member1.Info.DID] = member1

	// ok
	txn := s.getCRCProposalReviewTx(publicKeyStr1, privateKeyStr1)
	crcProposalReview, _ := txn.Payload().(*payload.CRCProposalReview)
	manager := s.Chain.GetCRCommittee().GetProposalManager()
	manager.Proposals[crcProposalReview.ProposalHash] = &crstate.ProposalState{
		Status: crstate.Registered,
	}
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: tenureHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ := txn.SpecialContextCheck()
	s.NoError(err)

	// member status is not elected
	member1.MemberState = crstate.MemberImpeached
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:should be an elected CR members")

	// invalid payload
	txn.SetPayload(&payload.CRInfo{})
	member1.MemberState = crstate.MemberElected
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid payload")

	// invalid content type
	txn = s.getCRCProposalReviewTx(publicKeyStr1, privateKeyStr1)
	txn.Payload().(*payload.CRCProposalReview).VoteResult = 0x10
	crcProposalReview2, _ := txn.Payload().(*payload.CRCProposalReview)
	manager.Proposals[crcProposalReview2.ProposalHash] = &crstate.ProposalState{
		Status: crstate.Registered,
	}
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: tenureHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:VoteResult should be known")

	// proposal reviewer is not CR member
	txn = s.getCRCProposalReviewTx(publicKeyStr2, privateKeyStr2)
	crcProposalReview3, _ := txn.Payload().(*payload.CRCProposalReview)
	manager.Proposals[crcProposalReview3.ProposalHash] = &crstate.ProposalState{
		Status: crstate.Registered,
	}
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: tenureHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:did correspond crMember not exists")

	delete(manager.Proposals, crcProposalReview.ProposalHash)
	// invalid CR proposal reviewer signature
	txn = s.getCRCProposalReviewTx(publicKeyStr1, privateKeyStr1)
	txn.Payload().(*payload.CRCProposalReview).Signature = []byte{}
	crcProposalReview, _ = txn.Payload().(*payload.CRCProposalReview)
	manager.Proposals[crcProposalReview.ProposalHash] = &crstate.ProposalState{
		Status: crstate.Registered,
	}
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: tenureHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid signature length")
	delete(s.Chain.GetCRCommittee().GetProposalManager().Proposals, crcProposalReview.ProposalHash)
}

func (s *txValidatorTestSuite) getCRCProposalWithdrawTx(crPublicKeyStr,
	crPrivateKeyStr string, recipient,
	commitee *common.Uint168, recipAmout, commiteAmout common.Fixed64, payloadVersion byte) interfaces.Transaction {

	privateKey1, _ := common.HexStringToBytes(crPrivateKeyStr)
	pkBytes, _ := common.HexStringToBytes(crPublicKeyStr)

	txn := functions.CreateTransaction(
		common2.TxVersionDefault,
		common2.CRCProposalWithdraw,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	var crcProposalWithdraw *payload.CRCProposalWithdraw
	switch payloadVersion {
	case 0x00:
		crcProposalWithdraw = &payload.CRCProposalWithdraw{
			ProposalHash:   *randomUint256(),
			OwnerPublicKey: pkBytes,
		}
	case 0x01:
		crcProposalWithdraw = &payload.CRCProposalWithdraw{
			ProposalHash:   *randomUint256(),
			OwnerPublicKey: pkBytes,
			Recipient:      *recipient,
			Amount:         recipAmout,
		}
		txn.SetPayloadVersion(payload.CRCProposalWithdrawVersion01)
	}

	signBuf := new(bytes.Buffer)
	crcProposalWithdraw.SerializeUnsigned(signBuf, txn.PayloadVersion())
	sig, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crcProposalWithdraw.Signature = sig

	txn.SetInputs([]*common2.Input{
		{
			Previous: common2.OutPoint{
				TxID:  common.EmptyHash,
				Index: math.MaxUint16,
			},
			Sequence: math.MaxUint32,
		},
	})
	txn.SetOutputs([]*common2.Output{
		{
			AssetID:     config.ELAAssetID,
			ProgramHash: *recipient,
			Value:       recipAmout,
		},
		{
			AssetID:     config.ELAAssetID,
			ProgramHash: *commitee,
			Value:       commiteAmout,
		},
	})

	txn.SetPayload(crcProposalWithdraw)
	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(crPublicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func (s *txValidatorTestSuite) TestCheckCRCProposalWithdrawTransaction() {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	RecipientAddress := "ERyUmNH51roR9qfru37Kqkaok2NghR7L5U"
	CRExpensesAddress := "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta"
	NOCRExpensesAddress := "EWm2ZGeSyDBBAsVSsvSvspPKV4wQBKPjUk"
	Recipient, _ := common.Uint168FromAddress(RecipientAddress)
	tenureHeight := config.DefaultParams.CRCommitteeStartHeight
	pk1Bytes, _ := common.HexStringToBytes(publicKeyStr1)
	ela := common.Fixed64(100000000)
	CRExpensesAddressU168, _ := common.Uint168FromAddress(CRExpensesAddress)
	NOCRExpensesAddressU168, _ := common.Uint168FromAddress(NOCRExpensesAddress)

	inputs := []*common2.Input{
		{
			Previous: common2.OutPoint{
				TxID:  common.EmptyHash,
				Index: 1,
			},
			Sequence: math.MaxUint32,
		},
	}
	outputs := []*common2.Output{
		{
			AssetID:     config.ELAAssetID,
			ProgramHash: *CRExpensesAddressU168,
			Value:       common.Fixed64(60 * ela),
		},
		{
			AssetID:     config.ELAAssetID,
			ProgramHash: *NOCRExpensesAddressU168,
			Value:       common.Fixed64(600 * ela),
		},
	}

	references := make(map[*common2.Input]common2.Output)
	references[inputs[0]] = *outputs[0]

	s.Chain.GetParams().CRExpensesAddress = *CRExpensesAddressU168
	// stage = 1 ok
	txn := s.getCRCProposalWithdrawTx(publicKeyStr1, privateKeyStr1,
		Recipient, CRExpensesAddressU168, 9*ela, 50*ela, 0)
	crcProposalWithdraw, _ := txn.Payload().(*payload.CRCProposalWithdraw)
	pld := payload.CRCProposal{
		OwnerPublicKey: pk1Bytes,
		Recipient:      *Recipient,
		Budgets:        createBudgets(3),
	}
	propState := &crstate.ProposalState{
		Status:              crstate.VoterAgreed,
		Proposal:            pld.ToProposalInfo(0),
		FinalPaymentStatus:  false,
		WithdrawableBudgets: map[uint8]common.Fixed64{0: 10 * 1e8},
		ProposalOwner:       pk1Bytes,
		Recipient:           *Recipient,
	}
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[crcProposalWithdraw.
		ProposalHash] = propState
	err := s.Chain.CheckTransactionOutput(txn, tenureHeight)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: tenureHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	txn.SetReferences(references)
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	//CRCProposalWithdraw Stage wrong too small
	propState.WithdrawnBudgets = map[uint8]common.Fixed64{0: 10 * 1e8}
	err = s.Chain.CheckTransactionOutput(txn, tenureHeight)
	//err = s.Chain.CheckCRCProposalWithdrawTransaction(txn, references, tenureHeight)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:no need to withdraw")

	//stage =2 ok
	txn = s.getCRCProposalWithdrawTx(publicKeyStr1, privateKeyStr1,
		Recipient, CRExpensesAddressU168, 19*ela, 40*ela, 0)
	crcProposalWithdraw, _ = txn.Payload().(*payload.CRCProposalWithdraw)
	propState.WithdrawableBudgets = map[uint8]common.Fixed64{0: 10 * 1e8, 1: 20 * 1e8}
	propState.FinalPaymentStatus = false
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[crcProposalWithdraw.
		ProposalHash] = propState
	err = s.Chain.CheckTransactionOutput(txn, tenureHeight)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: tenureHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	txn.SetReferences(references)
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	//stage =3 ok
	txn = s.getCRCProposalWithdrawTx(publicKeyStr1, privateKeyStr1,
		Recipient, CRExpensesAddressU168, 29*ela, 30*ela, 0)
	crcProposalWithdraw, _ = txn.Payload().(*payload.CRCProposalWithdraw)
	propState.WithdrawableBudgets = map[uint8]common.Fixed64{0: 10 * 1e8, 1: 20 * 1e8, 2: 30 * 1e8}
	propState.WithdrawnBudgets = map[uint8]common.Fixed64{0: 10 * 1e8, 1: 20 * 1e8}
	propState.FinalPaymentStatus = true
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[crcProposalWithdraw.
		ProposalHash] = propState
	err = s.Chain.CheckTransactionOutput(txn, tenureHeight)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: tenureHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	txn.SetReferences(references)
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	//len(txn.Outputs) ==0 transaction has no outputs
	txn.SetOutputs([]*common2.Output{})
	err = s.Chain.CheckTransactionOutput(txn, tenureHeight)
	s.EqualError(err, "transaction has no outputs")

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	pk2Bytes, _ := common.HexStringToBytes(publicKeyStr2)

	propState.ProposalOwner = pk2Bytes
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:the OwnerPublicKey is not owner of proposal")

	references[inputs[0]] = *outputs[1]
	txn.SetReferences(references)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:proposal withdrawal transaction for non-crc committee address")

	txn = s.getCRCProposalWithdrawTx(publicKeyStr1, privateKeyStr1,
		Recipient, CRExpensesAddressU168, 19*ela, 40*ela, 1)
	crcProposalWithdraw, _ = txn.Payload().(*payload.CRCProposalWithdraw)
	propState.WithdrawableBudgets = map[uint8]common.Fixed64{0: 10 * 1e8, 1: 20 * 1e8}
	propState.WithdrawnBudgets = map[uint8]common.Fixed64{0: 10 * 1e8}
	propState.FinalPaymentStatus = false
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[crcProposalWithdraw.
		ProposalHash] = propState
	propState.ProposalOwner = pk1Bytes
	err = s.Chain.CheckTransactionOutput(txn, tenureHeight)
	inputs = []*common2.Input{
		{
			Previous: common2.OutPoint{
				TxID:  common.EmptyHash,
				Index: 1,
			},
			Sequence: math.MaxUint32,
		},
	}
	outputs = []*common2.Output{
		{
			AssetID:     config.ELAAssetID,
			ProgramHash: *CRExpensesAddressU168,
			Value:       common.Fixed64(61 * ela),
		},
	}
	references = make(map[*common2.Input]common2.Output)
	references[inputs[0]] = *outputs[0]
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: tenureHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	txn.SetReferences(references)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:withdrawPayload.Amount != withdrawAmount ")
	outputs = []*common2.Output{
		{
			AssetID:     config.ELAAssetID,
			ProgramHash: *CRExpensesAddressU168,
			Value:       common.Fixed64(61 * ela),
		},
	}

	txn = s.getCRCProposalWithdrawTx(publicKeyStr1, privateKeyStr1,
		Recipient, CRExpensesAddressU168, 20*ela, 40*ela, 1)
	crcProposalWithdraw, _ = txn.Payload().(*payload.CRCProposalWithdraw)
	propState.WithdrawableBudgets = map[uint8]common.Fixed64{0: 10 * 1e8, 1: 20 * 1e8}
	propState.WithdrawnBudgets = map[uint8]common.Fixed64{0: 10 * 1e8}
	propState.FinalPaymentStatus = false
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[crcProposalWithdraw.
		ProposalHash] = propState
	references = make(map[*common2.Input]common2.Output)
	references[inputs[0]] = *outputs[0]
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		BlockHeight: tenureHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	txn.SetReferences(references)
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

}

func (s *txValidatorTestSuite) getCRChangeProposalOwnerProposalTx(publicKeyStr, privateKeyStr,
	crPublicKeyStr, crPrivateKeyStr, newOwnerPublicKeyStr string, targetHash common.Uint256) interfaces.Transaction {

	privateKey, _ := common.HexStringToBytes(privateKeyStr)
	crPrivateKey, _ := common.HexStringToBytes(crPrivateKeyStr)
	crPublicKey, _ := common.HexStringToBytes(crPublicKeyStr)
	crDid, _ := blockchain.GetDIDFromCode(getCodeByPubKeyStr(crPublicKeyStr))
	newOwnerPublicKey, _ := common.HexStringToBytes(newOwnerPublicKeyStr)
	draftData := randomBytes(10)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposal,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	crcProposalPayload := &payload.CRCProposal{
		ProposalType:       payload.ChangeProposalOwner,
		OwnerPublicKey:     crPublicKey,
		NewOwnerPublicKey:  newOwnerPublicKey,
		TargetProposalHash: targetHash,
		DraftHash:          common.Hash(draftData),
		CRCouncilMemberDID: *crDid,
	}

	signBuf := new(bytes.Buffer)
	crcProposalPayload.SerializeUnsigned(signBuf, payload.CRCProposalVersion)
	sig, _ := crypto.Sign(privateKey, signBuf.Bytes())
	crcProposalPayload.Signature = sig

	common.WriteVarBytes(signBuf, sig)
	crcProposalPayload.CRCouncilMemberDID.Serialize(signBuf)
	crSig, _ := crypto.Sign(crPrivateKey, signBuf.Bytes())
	crcProposalPayload.CRCouncilMemberSignature = crSig

	txn.SetPayload(crcProposalPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func (s *txValidatorTestSuite) TestGenrateTxFromRawTxStr() {
	rawTxStr := "0925000004033131312102f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a51f06ed7688f2b6e445f6579ca802fd1b6425b1e02a1944a8df714301f629363521031e12374bae471aa09ad479f66c2306f4bcc4ca5b754609a82a1839b94b4721b967e7bbc540fab57abb2dc9ba1e6cdbf9ae3979e3cb40a83441dea2934a24287233dcbe994bc67f4b9001ccf907a04080ef7b884ff1653853c51d955c0ea7ec62f1d871e834c6da65e5863c47f67fc0fc0449c3ffefda4028d1440852764247bc90c71cb946b545501c08a4d72813b8481bb48b1f95cb6997192e463862469f1729acb403fc8f64dcee80e6dbd9d0f59a136003585a0da3670993d2b44ca80090642ab67053a90918f1dbedf24016b130872665b86dfa57500bced7bd87c1f5be313d46f49907c41e07ec032ad8c28810fc2a6564dc18d20cafb526250434f14c6359b4c060b6d5e0f7750d55a6000000000000000100232102f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5ac"
	data, err := common.HexStringToBytes(rawTxStr)
	if err != nil {
		fmt.Println("err", err)
		return

	}
	reader := bytes.NewReader(data)
	tx, _ := functions.GetTransactionByBytes(reader)
	err2 := tx.Deserialize(reader)
	if err2 != nil {
		fmt.Println("err2", err2)
		return
	}

	buf2 := new(bytes.Buffer)
	tx.Serialize(buf2)
	rawTxStr2 := common.BytesToHexString(buf2.Bytes())
	s.Equal(rawTxStr2, rawTxStr)
	s.Equal("dc327a5ef958385a23082e8c73b2fa7c330793ad601587db84ecec3977989b33", tx.Hash().String())
}

func (s *txValidatorTestSuite) TestGenerateRawTransactionStr() {
	//generate raw tx str
	ownerPublicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	ownerPrivateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	crPublicKeyStr := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	crPrivateKeyStr := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	secretaryPublicKeyStr := "031e12374bae471aa09ad479f66c2306f4bcc4ca5b754609a82a1839b94b4721b9"
	secretaryPrivateKeyStr := "94396a69462208b8fd96d83842855b867d3b0e663203cb31d0dfaec0362ec034"
	//recipent, draftData are all random data so hash is changing all the time
	txn := s.getSecretaryGeneralCRCProposalTx(ownerPublicKeyStr1, ownerPrivateKeyStr1, crPublicKeyStr, crPrivateKeyStr,
		secretaryPublicKeyStr, secretaryPrivateKeyStr)
	buf := new(bytes.Buffer)
	txn.Serialize(buf)

	rawTxStr := common.BytesToHexString(buf.Bytes())

	data, err2 := common.HexStringToBytes(rawTxStr)
	if err2 != nil {
		fmt.Println("HexStringToBytes err2", err2)
	}
	reader2 := bytes.NewReader(data)
	txn2, err3 := functions.GetTransactionByBytes(reader2)
	if err3 != nil {
		s.Assert()
	}
	err2 = txn2.Deserialize(reader2)
	if err2 != nil {
		fmt.Println("txn2.Deserialize err2", err2)
	}
	s.Equal(txn2.Hash().String(), txn.Hash().String())

}

func (s *txValidatorTestSuite) TestCheckSecretaryGeneralProposalTransaction() {

	ownerPublicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	ownerPrivateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"

	crPublicKeyStr := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	crPrivateKeyStr := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"

	secretaryPublicKeyStr := "031e12374bae471aa09ad479f66c2306f4bcc4ca5b754609a82a1839b94b4721b9"
	secretaryPrivateKeyStr := "94396a69462208b8fd96d83842855b867d3b0e663203cb31d0dfaec0362ec034"

	tenureHeight := config.DefaultParams.CRCommitteeStartHeight + 1
	ownerNickName := "nickname owner"
	crNickName := "nickname cr"

	memberOwner := s.getCRMember(ownerPublicKeyStr1, ownerPrivateKeyStr1, ownerNickName)
	memberCr := s.getCRMember(crPublicKeyStr, crPrivateKeyStr, crNickName)

	memebers := make(map[common.Uint168]*crstate.CRMember)

	s.Chain.GetCRCommittee().Members = memebers
	s.Chain.GetCRCommittee().CRCCommitteeBalance = common.Fixed64(100 * 1e8)
	s.Chain.GetCRCommittee().CRCCurrentStageAmount = common.Fixed64(100 * 1e8)
	s.Chain.GetCRCommittee().InElectionPeriod = true
	s.Chain.GetCRCommittee().NeedAppropriation = false

	//owner not elected cr
	txn := s.getSecretaryGeneralCRCProposalTx(ownerPublicKeyStr1, ownerPrivateKeyStr1, crPublicKeyStr, crPrivateKeyStr,
		secretaryPublicKeyStr, secretaryPrivateKeyStr)

	//CRCouncilMember not elected cr
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ := txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CR Council Member should be one of the CR members")
	memebers[memberCr.Info.DID] = memberCr
	memebers[memberOwner.Info.DID] = memberOwner

	//owner signature check failed
	rightSign := txn.Payload().(*payload.CRCProposal).Signature
	txn.Payload().(*payload.CRCProposal).Signature = []byte{}
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:owner signature check failed")
	txn.Payload().(*payload.CRCProposal).Signature = rightSign

	//SecretaryGeneral signature check failed
	secretaryGeneralSign := txn.Payload().(*payload.CRCProposal).SecretaryGeneraSignature
	txn.Payload().(*payload.CRCProposal).SecretaryGeneraSignature = []byte{}
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:SecretaryGeneral signature check failed")
	txn.Payload().(*payload.CRCProposal).SecretaryGeneraSignature = secretaryGeneralSign

	//CRCouncilMemberSignature signature check failed
	crcouncilMemberSignature := txn.Payload().(*payload.CRCProposal).CRCouncilMemberSignature
	txn.Payload().(*payload.CRCProposal).CRCouncilMemberSignature = []byte{}
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CR Council Member signature check failed")
	txn.Payload().(*payload.CRCProposal).CRCouncilMemberSignature = crcouncilMemberSignature

	//SecretaryGeneralPublicKey and SecretaryGeneralDID not match
	secretaryGeneralPublicKey := txn.Payload().(*payload.CRCProposal).SecretaryGeneralPublicKey
	txn.Payload().(*payload.CRCProposal).SecretaryGeneralPublicKey, _ = common.HexStringToBytes(ownerPublicKeyStr1)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:SecretaryGeneral NodePublicKey and DID is not matching")
	txn.Payload().(*payload.CRCProposal).SecretaryGeneralPublicKey = secretaryGeneralPublicKey

	// ok
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	//ChangeSecretaryGeneralProposal tx must InElectionPeriod and not during voting period
	config.DefaultParams.DPoSV2StartHeight = 2000000
	s.Chain.GetCRCommittee().LastCommitteeHeight = config.DefaultParams.CRCommitteeStartHeight
	tenureHeight = config.DefaultParams.CRCommitteeStartHeight + config.DefaultParams.CRDutyPeriod -
		config.DefaultParams.CRVotingPeriod + 1
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:cr proposal tx must not during voting period")
}

func (s *txValidatorTestSuite) TestCheckCRCProposalRegisterSideChainTransaction() {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"

	tenureHeight := config.DefaultParams.CRCommitteeStartHeight + 1
	nickName1 := "nickname 1"

	member1 := s.getCRMember(publicKeyStr1, privateKeyStr1, nickName1)
	memebers := make(map[common.Uint168]*crstate.CRMember)
	memebers[member1.Info.DID] = member1
	s.Chain.GetCRCommittee().Members = memebers
	s.Chain.GetCRCommittee().CRCCommitteeBalance = common.Fixed64(100 * 1e8)
	s.Chain.GetCRCommittee().CRCCurrentStageAmount = common.Fixed64(100 * 1e8)
	s.Chain.GetCRCommittee().InElectionPeriod = true
	s.Chain.GetCRCommittee().NeedAppropriation = false

	{
		// no error
		txn := s.getCRCRegisterSideChainProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
		txn = CreateTransactionByType(txn, s.Chain)
		txn.SetParameters(&transaction.TransactionParameters{
			Transaction:         txn,
			BlockHeight:         tenureHeight,
			TimeStamp:           s.Chain.BestChain.Timestamp,
			Config:              s.Chain.GetParams(),
			BlockChain:          s.Chain,
			ProposalsUsedAmount: 0,
		})
		err, _ := txn.SpecialContextCheck()
		s.NoError(err)

		// genesis hash can not be blank
		payload, _ := txn.Payload().(*payload.CRCProposal)
		payload.GenesisHash = common.Uint256{}
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err, "transaction validate error: payload content invalid:GenesisHash can not be empty")
	}

	{
		txn := s.getCRCRegisterSideChainProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
		payload, _ := txn.Payload().(*payload.CRCProposal)
		payload.SideChainName = ""
		txn = CreateTransactionByType(txn, s.Chain)
		txn.SetParameters(&transaction.TransactionParameters{
			Transaction:         txn,
			BlockHeight:         tenureHeight,
			TimeStamp:           s.Chain.BestChain.Timestamp,
			Config:              s.Chain.GetParams(),
			BlockChain:          s.Chain,
			ProposalsUsedAmount: 0,
		})
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err, "transaction validate error: payload content invalid:SideChainName can not be empty")
	}

	{
		s.Chain.GetCRCommittee().GetProposalManager().RegisteredSideChainNames = []string{"NEO"}
		txn := s.getCRCRegisterSideChainProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
		txn = CreateTransactionByType(txn, s.Chain)
		txn.SetParameters(&transaction.TransactionParameters{
			Transaction:         txn,
			BlockHeight:         tenureHeight,
			TimeStamp:           s.Chain.BestChain.Timestamp,
			Config:              s.Chain.GetParams(),
			BlockChain:          s.Chain,
			ProposalsUsedAmount: 0,
		})
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err, "transaction validate error: payload content invalid:SideChainName already registered")
	}

}

func (s *txValidatorTestSuite) TestCheckCRCProposalTransaction() {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"

	tenureHeight := config.DefaultParams.CRCommitteeStartHeight + 1
	nickName1 := "nickname 1"
	nickName2 := "nickname 2"

	member1 := s.getCRMember(publicKeyStr1, privateKeyStr1, nickName1)
	memebers := make(map[common.Uint168]*crstate.CRMember)
	memebers[member1.Info.DID] = member1
	s.Chain.GetCRCommittee().Members = memebers
	s.Chain.GetCRCommittee().CRCCommitteeBalance = common.Fixed64(100 * 1e8)
	s.Chain.GetCRCommittee().CRCCurrentStageAmount = common.Fixed64(100 * 1e8)
	s.Chain.GetCRCommittee().InElectionPeriod = true
	s.Chain.GetCRCommittee().NeedAppropriation = false

	// ok
	txn := s.getCRCProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)

	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ := txn.SpecialContextCheck()
	s.NoError(err)

	// member status is not elected
	member1.MemberState = crstate.MemberImpeached
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CR Council Member should be an elected CR members")

	// register cr proposal in voting period
	member1.MemberState = crstate.MemberElected
	tenureHeight = config.DefaultParams.CRCommitteeStartHeight +
		config.DefaultParams.CRDutyPeriod - config.DefaultParams.CRVotingPeriod
	s.Chain.GetCRCommittee().InElectionPeriod = false
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:cr proposal tx must not during voting period")

	// recipient is empty
	s.Chain.GetCRCommittee().InElectionPeriod = true
	tenureHeight = config.DefaultParams.CRCommitteeStartHeight + 1
	txn.Payload().(*payload.CRCProposal).Recipient = common.Uint168{}
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:recipient is empty")

	// invalid payload
	txn.SetPayload(&payload.CRInfo{})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid payload")

	// invalid proposal type
	txn = s.getCRCProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
	txn.Payload().(*payload.CRCProposal).ProposalType = 0x1000
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:type of proposal should be known")

	// invalid outputs of ELIP.
	txn.Payload().(*payload.CRCProposal).ProposalType = 0x0100
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:ELIP needs to have and only have two budget")

	// invalid budgets.
	txn.Payload().(*payload.CRCProposal).ProposalType = 0x0000
	s.Chain.GetCRCommittee().CRCCommitteeBalance = common.Fixed64(10 * 1e8)
	s.Chain.GetCRCommittee().CRCCurrentStageAmount = common.Fixed64(10 * 1e8)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:budgets exceeds 10% of CRC committee balance")

	s.Chain.GetCRCommittee().CRCCommitteeBalance = common.Fixed64(100 * 1e8)
	s.Chain.GetCRCommittee().CRCCurrentStageAmount = common.Fixed64(100 * 1e8)
	s.Chain.GetCRCommittee().CRCCommitteeUsedAmount = common.Fixed64(99 * 1e8)
	err, _ = txn.SpecialContextCheck()
	s.Error(err, "transaction validate error: payload content invalid:budgets exceeds the balance of CRC committee")

	s.Chain.GetCRCommittee().CRCCommitteeUsedAmount = common.Fixed64(0)

	// CRCouncilMemberSignature is not signed by CR member
	txn = s.getCRCProposalTx(publicKeyStr1, privateKeyStr1, publicKeyStr2, privateKeyStr2)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CR Council Member should be one of the CR members")

	// invalid owner
	txn = s.getCRCProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
	txn.Payload().(*payload.CRCProposal).OwnerPublicKey = []byte{}
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid owner")

	// invalid owner signature
	txn = s.getCRCProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	txn.Payload().(*payload.CRCProposal).OwnerPublicKey = publicKey1
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:owner signature check failed")

	// invalid CR owner signature
	txn = s.getCRCProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
	txn.Payload().(*payload.CRCProposal).CRCouncilMemberSignature = []byte{}
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:failed to check CR Council Member signature")

	// proposal status is not VoterAgreed
	newOwnerPublicKeyStr := publicKeyStr2
	publicKey2, _ := hex.DecodeString(publicKeyStr2)
	proposalState, proposal := s.createSpecificStatusProposal(publicKey1, publicKey2, tenureHeight,
		crstate.Registered, payload.Normal)

	s.Chain.GetCRCommittee().GetProposalManager().Proposals[proposal.Hash(payload.CRCProposalVersion01)] = proposalState

	txn = s.getCRChangeProposalOwnerProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1,
		newOwnerPublicKeyStr, proposal.Hash(payload.CRCProposalVersion01))
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:proposal status is not VoterAgreed")

	//proposal sponsors must be members
	targetHash := proposal.Hash(payload.CRCProposalVersion01)
	newOwnerPublicKey, _ := hex.DecodeString(newOwnerPublicKeyStr)
	proposalState2, proposal2 := s.createSpecificStatusProposal(publicKey1, publicKey2, tenureHeight+1,
		crstate.VoterAgreed, payload.ChangeProposalOwner)
	proposal2.TargetProposalHash = targetHash
	proposal2.OwnerPublicKey = newOwnerPublicKey
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[targetHash] = proposalState2
	txn = s.getCRChangeProposalOwnerProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1,
		newOwnerPublicKeyStr, targetHash)

	s.Chain.GetCRCommittee().InElectionPeriod = false
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:cr proposal tx must not during voting period")

	// invalid proposal owner
	s.Chain.GetCRCommittee().InElectionPeriod = true
	proposalState3, proposal3 := s.createSpecificStatusProposal(publicKey1, publicKey2, tenureHeight,
		crstate.Registered, payload.Normal)
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[proposal3.Hash(payload.CRCProposalVersion01)] = proposalState3

	txn = s.getCRCCloseProposalTxWithHash(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1,
		proposal.Hash(payload.CRCProposalVersion01))

	// invalid closeProposalHash
	txn = s.getCRCCloseProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CloseProposalHash does not exist")

	// invalid proposal status
	hash := proposal.Hash(payload.CRCProposalVersion01)
	member2 := s.getCRMember(publicKeyStr2, privateKeyStr2, nickName2)
	memebers[member2.Info.DID] = member2
	txn = s.getCRCCloseProposalTxWithHash(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1,
		proposal.Hash(payload.CRCProposalVersion01))

	proposalState.Status = crstate.Registered
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[hash] = proposalState
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CloseProposalHash has to be voterAgreed")

	// invalid receipt
	proposalState, proposal = s.createSpecificStatusProposal(publicKey1, publicKey2, tenureHeight,
		crstate.VoterAgreed, payload.Normal)
	hash = proposal.Hash(payload.CRCProposalVersion01)
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[hash] = proposalState
	txn = s.getCRCCloseProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
	txn.Payload().(*payload.CRCProposal).TargetProposalHash = hash
	txn.Payload().(*payload.CRCProposal).Recipient = *randomUint168()
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CloseProposal recipient must be empty")

	// invalid budget
	txn = s.getCRCCloseProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
	txn.Payload().(*payload.CRCProposal).TargetProposalHash = hash
	txn.Payload().(*payload.CRCProposal).Budgets = []payload.Budget{{
		payload.Imprest,
		0x01,
		common.Fixed64(10000000000),
	}}
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CloseProposal cannot have budget")

	// proposals can not more than MaxCommitteeProposalCount
	txn = s.getCRCProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
	crcProposal, _ := txn.Payload().(*payload.CRCProposal)
	proposalHashSet := crstate.NewProposalHashSet()
	for i := 0; i < int(s.Chain.GetParams().MaxCommitteeProposalCount); i++ {
		proposalHashSet.Add(*randomUint256())
	}
	s.Chain.GetCRCommittee().GetProposalManager().ProposalHashes[crcProposal.
		CRCouncilMemberDID] = proposalHashSet
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:proposal is full")

	s.Chain.GetParams().MaxCommitteeProposalCount = s.Chain.GetParams().MaxCommitteeProposalCount + 100
	// invalid reserved custom id
	txn = s.getCRCReservedCustomIDProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
	proposal, _ = txn.Payload().(*payload.CRCProposal)
	proposal.ReservedCustomIDList = append(proposal.ReservedCustomIDList, randomName(260))
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid reserved custom id length")
}

func (s *txValidatorTestSuite) TestCheckStringField() {
	s.NoError(blockchain.CheckStringField("Normal", "test", false))
	s.EqualError(blockchain.CheckStringField("", "test", false),
		"field test has invalid string length")
	s.EqualError(blockchain.CheckStringField("I am more than 100, 1234567890123456"+
		"789012345678901234567890123456789012345678901234567890123456789012345"+
		"678901234567890", "test", false), "field test"+
		" has invalid string length")
}

func (s *txValidatorTestSuite) TestCheckTransactionDepositUTXO() {
	references := make(map[*common2.Input]common2.Output)
	input := &common2.Input{}
	// Use the deposit UTXO in a TransferAsset transaction
	depositHash, _ := common.Uint168FromAddress("DVgnDnVfPVuPa2y2E4JitaWjWgRGJDuyrD")
	depositOutput := common2.Output{
		ProgramHash: *depositHash,
	}
	references[input] = depositOutput

	txn, _ := transaction.GetTransaction(common2.TransferAsset)
	err := blockchain.CheckTransactionDepositUTXO(txn, references)
	s.EqualError(err, "only the ReturnDepositCoin and "+
		"ReturnCRDepositCoin transaction can use the deposit UTXO")

	// Use the deposit UTXO in a ReturnDepositCoin transaction
	txn, _ = transaction.GetTransaction(common2.ReturnDepositCoin)
	err = blockchain.CheckTransactionDepositUTXO(txn, references)
	s.NoError(err)

	// Use the standard UTXO in a ReturnDepositCoin transaction
	normalHash, _ := common.Uint168FromAddress("EJMzC16Eorq9CuFCGtyMrq4Jmgw9jYCHQR")
	normalOutput := common2.Output{
		ProgramHash: *normalHash,
	}
	references[input] = normalOutput
	txn, _ = transaction.GetTransaction(common2.ReturnDepositCoin)
	err = blockchain.CheckTransactionDepositUTXO(txn, references)
	s.EqualError(err, "the ReturnDepositCoin and ReturnCRDepositCoin "+
		"transaction can only use the deposit UTXO")

	// Use the deposit UTXO in a ReturnDepositCoin transaction
	references[input] = depositOutput
	txn, _ = transaction.GetTransaction(common2.ReturnCRDepositCoin)
	err = blockchain.CheckTransactionDepositUTXO(txn, references)
	s.NoError(err)

	references[input] = normalOutput
	txn, _ = transaction.GetTransaction(common2.ReturnCRDepositCoin)
	err = blockchain.CheckTransactionDepositUTXO(txn, references)
	s.EqualError(err, "the ReturnDepositCoin and ReturnCRDepositCoin "+
		"transaction can only use the deposit UTXO")
}

func (s *txValidatorTestSuite) TestCheckReturnDepositCoinTransaction() {
	s.CurrentHeight = 1
	s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams()))
	s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
		GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
		GetUTXO:                          s.Chain.GetDB().GetFFLDB().GetUTXO,
		GetHeight:                        func() uint32 { return s.CurrentHeight },
		CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
	})
	_, pk, _ := crypto.GenerateKeyPair()
	depositCont, _ := contract.CreateDepositContractByPubKey(pk)
	publicKey, _ := pk.EncodePoint(true)
	// register CR

	txn := functions.CreateTransaction(
		0,
		common2.RegisterProducer,
		0,
		&payload.ProducerInfo{
			OwnerPublicKey: publicKey,
			NodePublicKey:  publicKey,
			NickName:       randomString(),
			Url:            randomString(),
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				ProgramHash: *depositCont.ToProgramHash(),
				Value:       common.Fixed64(5000 * 1e8),
			},
		},
		0,
		[]*program.Program{},
	)

	s.Chain.GetState().ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: s.CurrentHeight,
		},
		Transactions: []interfaces.Transaction{txn},
	}, nil, 0)
	s.CurrentHeight++
	producer := s.Chain.GetState().GetProducer(publicKey)
	s.True(producer.State() == state.Pending, "register producer failed")

	for i := 0; i < 6; i++ {
		s.Chain.GetState().ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: s.CurrentHeight,
			},
			Transactions: []interfaces.Transaction{},
		}, nil, 0)
		s.CurrentHeight++
	}
	s.True(producer.State() == state.Active, "active producer failed")

	// check a return deposit coin transaction with wrong state.
	references := make(map[*common2.Input]common2.Output)
	references[&common2.Input{}] = common2.Output{
		ProgramHash: *randomUint168(),
		Value:       common.Fixed64(5000 * 100000000),
	}

	code1, _ := contract.CreateStandardRedeemScript(pk)
	rdTx := functions.CreateTransaction(
		0,
		common2.ReturnDepositCoin,
		0,
		&payload.ReturnDepositCoin{},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{Value: 4999 * 100000000},
		},
		0,
		[]*program.Program{
			{Code: code1},
		},
	)

	rdTx = CreateTransactionByType(rdTx, s.Chain)
	rdTx.SetReferences(references)
	err, _ := rdTx.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:overspend deposit")

	// cancel CR
	ctx := functions.CreateTransaction(
		0,
		common2.CancelProducer,
		0,
		&payload.ProcessProducer{
			OwnerPublicKey: publicKey,
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{Value: 4999 * 100000000},
		},
		0,
		[]*program.Program{
			{Code: code1},
		},
	)

	s.Chain.GetState().ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: s.CurrentHeight,
		},
		Transactions: []interfaces.Transaction{ctx},
	}, nil, 0)
	s.True(producer.State() == state.Canceled, "cancel producer failed")

	// check a return deposit coin transaction with wrong code.
	publicKey2 := "030a26f8b4ab0ea219eb461d1e454ce5f0bd0d289a6a64ffc0743dab7bd5be0be9"
	pubKeyBytes2, _ := common.HexStringToBytes(publicKey2)
	pubkey2, _ := crypto.DecodePoint(pubKeyBytes2)
	code2, _ := contract.CreateStandardRedeemScript(pubkey2)
	rdTx.Programs()[0].Code = code2
	err, _ = rdTx.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:signer must be producer")

	// check a return deposit coin transaction when not reached the
	// count of DepositLockupBlocks.
	rdTx.Programs()[0].Code = code1
	err, _ = rdTx.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:overspend deposit")

	s.CurrentHeight += s.Chain.GetParams().CRDepositLockupBlocks
	s.Chain.GetState().ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: s.CurrentHeight,
		},
		Transactions: []interfaces.Transaction{},
	}, nil, 0)

	// check a return deposit coin transaction with wrong output amount.
	rdTx.Outputs()[0].Value = 5000 * 100000000
	err, _ = rdTx.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:overspend deposit")

	// check a correct return deposit coin transaction.
	rdTx.Outputs()[0].Value = 4999 * 100000000
	err, _ = rdTx.SpecialContextCheck()
	s.NoError(err)
}

func (s *txValidatorTestSuite) TestCheckStakeTransaction2() {
	s.CurrentHeight = 1
	_, pk, _ := crypto.GenerateKeyPair()
	//publicKey, _ := pk.EncodePoint(true)
	cont, _ := contract.CreateStandardContract(pk)
	code := cont.Code
	ct, _ := contract.CreateStakeContractByCode(code)
	stakeAddress := ct.ToProgramHash()
	ps := &payload.ExchangeVotes{}
	attribute := []*common2.Attribute{}

	tx1 := functions.CreateTransaction(
		0,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	tx1.SetOutputs([]*common2.Output{
		&common2.Output{
			AssetID:     config.ELAAssetID,
			Value:       2000,
			ProgramHash: blockchain.FoundationAddress,
		},
	})
	input := &common2.Input{
		Previous: common2.OutPoint{
			TxID:  tx1.Hash(),
			Index: 0,
		},
		Sequence: 0,
	}
	outputs := []*common2.Output{
		{
			AssetID:     config.ELAAssetID,
			ProgramHash: *stakeAddress,
			Type:        common2.OTStake,
			Value:       common.Fixed64(1000 * 1e8),
			Payload: &outputpayload.ExchangeVotesOutput{
				StakeAddress: *stakeAddress,
			},
		}, {
			AssetID:     config.ELAAssetID,
			ProgramHash: *cont.ToProgramHash(),
			Type:        common2.OTNone,
			Value:       common.Fixed64(1000 * 1e8),
		},
	}
	programs := []*program.Program{{
		Code:      code,
		Parameter: nil,
	}}
	txn := functions.CreateTransaction(
		0,
		common2.ExchangeVotes,
		1,
		ps,
		attribute,
		[]*common2.Input{input},
		outputs,
		0,
		programs,
	)

	bc := s.Chain
	config := bc.GetParams()
	config.StakePool = *stakeAddress
	tx := txn.(*transaction.ExchangeVotesTransaction)
	tx.DefaultChecker.SetParameters(&transaction.TransactionParameters{
		BlockChain: bc,
		Config:     config,
	})

	err := txn.CheckTransactionOutput()
	s.NoError(err)

}

func (s *txValidatorTestSuite) TestCheckReutrnVotesTransaction() {
	s.CurrentHeight = 1
	_, pk, _ := crypto.GenerateKeyPair()
	//publicKey, _ := pk.EncodePoint(true)
	cont, _ := contract.CreateStandardContract(pk)
	code := cont.Code
	ct, _ := contract.CreateStakeContractByCode(code)
	stakeAddress := ct.ToProgramHash()
	pl := &payload.ReturnVotes{
		Value: 100,
	}
	attribute := []*common2.Attribute{}

	tx1 := functions.CreateTransaction(
		0,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	tx1.SetOutputs([]*common2.Output{
		&common2.Output{
			AssetID:     config.ELAAssetID,
			Value:       1000,
			ProgramHash: blockchain.FoundationAddress,
		},
	})
	input := &common2.Input{
		Previous: common2.OutPoint{
			TxID:  tx1.Hash(),
			Index: 0,
		},
		Sequence: 0,
	}
	outputs := []*common2.Output{
		{
			AssetID:     config.ELAAssetID,
			ProgramHash: *cont.ToProgramHash(),
			Type:        common2.OTNone,
			Value:       common.Fixed64(1000 * 1e8),
		},
	}
	programs := []*program.Program{{
		Code:      code,
		Parameter: nil,
	}}
	txn := functions.CreateTransaction(
		9,
		common2.ReturnVotes,
		0,
		pl,
		attribute,
		[]*common2.Input{input},
		outputs,
		0,
		programs,
	)

	bc := s.Chain
	config := bc.GetParams()
	config.StakePool = *stakeAddress
	tx := txn.(*transaction.ReturnVotesTransaction)
	tx.DefaultChecker.SetParameters(&transaction.TransactionParameters{
		BlockChain: bc,
		Config:     config,
	})

	err := txn.CheckTransactionPayload()
	s.NoError(err)

	// todo complete me
	//err2, _ := txn.SpecialContextCheck()
	//s.EqualError(err2, "transaction validate error: output invalid")

	err3 := txn.CheckTransactionPayload()
	s.NoError(err3)

}

func (s *txValidatorTestSuite) TestCheckReturnVotesTransaction2() {
	private := "97751342c819562a8d65059d759494fc9b2b565232bef047d1eae93f7c97baed"
	publicKey := "0228329FD319A5444F2265D08482B8C09360AE59945C50FA5211548C0C11D31F08"
	publicKeyBytes, _ := common.HexStringToBytes(publicKey)
	code := getCode(publicKeyBytes)
	c, _ := contract.CreateStakeContractByCode(code)
	stakeAddress_uint168 := c.ToProgramHash()
	//toAddr , _ := stakeAddress_uint168.ToAddress()
	txn := functions.CreateTransaction(
		0,
		common2.ReturnVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{{
			Code:      code,
			Parameter: nil,
		}},
	)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err := txn.CheckTransactionOutput()

	s.EqualError(err, "transaction has no outputs")

	txn = functions.CreateTransaction(
		0,
		common2.ReturnVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     common.Uint256{},
				Value:       100000000,
				OutputLock:  0,
				ProgramHash: *stakeAddress_uint168,
				Payload:     nil,
			},
		},
		0,
		[]*program.Program{{
			Code:      nil,
			Parameter: nil,
		}},
	)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = txn.CheckTransactionOutput()
	s.EqualError(err, "asset ID in output is invalid")

	txn = functions.CreateTransaction(
		0,
		common2.ReturnVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     config.ELAAssetID,
				Value:       -1,
				OutputLock:  0,
				ProgramHash: *stakeAddress_uint168,
				Payload:     nil,
			},
		},
		0,
		[]*program.Program{{
			Code:      nil,
			Parameter: nil,
		}},
	)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = txn.CheckTransactionOutput()
	s.EqualError(err, "invalid transaction UTXO output")

	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid payload")

	txn = functions.CreateTransaction(
		0,
		common2.ReturnVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     config.ELAAssetID,
				Value:       1,
				OutputLock:  0,
				ProgramHash: *stakeAddress_uint168,
				Payload:     nil,
			},
		},
		0,
		[]*program.Program{{
			Code:      nil,
			Parameter: nil,
		}},
	)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = txn.CheckTransactionOutput()
	s.NoError(err)

	txn = functions.CreateTransaction(
		0,
		common2.ReturnVotes,
		1,
		&payload.ReturnVotes{
			Value: -1,
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     config.ELAAssetID,
				Value:       1,
				OutputLock:  0,
				ProgramHash: *stakeAddress_uint168,
			},
		},
		0,
		[]*program.Program{{
			Code:      nil,
			Parameter: nil,
		}},
	)
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})

	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid return votes value")

	txn.SetPayload(&payload.ReturnVotes{
		Value: 10001,
	})
	txn.SetPayloadVersion(0x02)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid payload version")

	txn.SetPayloadVersion(0x00)
	txn.SetPayload(&payload.ReturnVotes{
		Value: 10001,
		Code:  code,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:vote rights not enough")

	s.Chain.GetState().DposV2VoteRights = map[common.Uint168]common.Fixed64{
		*stakeAddress_uint168: 10001,
	}
	txn.SetParameters(&transaction.TransactionParameters{
		Transaction: txn,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})

	buf := new(bytes.Buffer)
	tmpPayload := payload.ReturnVotes{
		ToAddr: *stakeAddress_uint168,
		Value:  10001,
		Code:   code,
	}

	tmpPayload.SerializeUnsigned(buf, payload.ReturnVotesVersionV0)
	privBuf, _ := hex.DecodeString(private)
	signature, _ := crypto.Sign(privBuf, buf.Bytes())
	txn.SetPayload(&payload.ReturnVotes{
		ToAddr:    *stakeAddress_uint168,
		Value:     10001,
		Code:      code,
		Signature: signature,
	})
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)
}

func (s *txValidatorTestSuite) TestCheckReturnCRDepositCoinTransaction() {
	s.CurrentHeight = 1
	_, pk, _ := crypto.GenerateKeyPair()
	cont, _ := contract.CreateStandardContract(pk)
	code := cont.Code
	depositCont, _ := contract.CreateDepositContractByPubKey(pk)
	ct, _ := contract.CreateCRIDContractByCode(code)
	cid := ct.ToProgramHash()

	s.Chain.GetParams().CRVotingStartHeight = uint32(1)
	s.Chain.GetParams().CRCommitteeStartHeight = uint32(3000)
	s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams()))
	s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
		GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
		GetUTXO:                          s.Chain.GetDB().GetFFLDB().GetUTXO,
		GetHeight:                        func() uint32 { return s.CurrentHeight },
		CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
	})
	// register CR
	p := &payload.CRInfo{
		Code:     code,
		CID:      *cid,
		NickName: randomString(),
	}
	outputs := []*common2.Output{
		{
			ProgramHash: *depositCont.ToProgramHash(),
			Value:       common.Fixed64(5000 * 1e8),
		},
	}
	txn := functions.CreateTransaction(
		0,
		common2.RegisterCR,
		0,
		p,
		[]*common2.Attribute{},
		[]*common2.Input{},
		outputs,
		0,
		[]*program.Program{},
	)
	s.Chain.GetCRCommittee().ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: s.CurrentHeight,
		},
		Transactions: []interfaces.Transaction{txn},
	}, nil)
	s.CurrentHeight++
	candidate := s.Chain.GetCRCommittee().GetCandidate(*cid)
	s.True(candidate.State == crstate.Pending, "register CR failed")

	for i := 0; i < 6; i++ {
		s.Chain.GetCRCommittee().ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: s.CurrentHeight,
			},
			Transactions: []interfaces.Transaction{},
		}, nil)
		s.CurrentHeight++
	}
	s.True(candidate.State == crstate.Active, "active CR failed")

	references := make(map[*common2.Input]common2.Output)
	references[&common2.Input{}] = common2.Output{
		ProgramHash: *randomUint168(),
		Value:       common.Fixed64(5000 * 100000000),
	}

	rdTx := functions.CreateTransaction(
		0,
		common2.ReturnCRDepositCoin,
		0,
		p,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{Value: 4999 * 100000000},
		},
		0,
		[]*program.Program{
			{Code: code},
		},
	)

	canceledHeight := uint32(8)

	// unregister CR
	cancelPayload := &payload.UnregisterCR{
		CID: *getCID(code),
	}
	canceltx := functions.CreateTransaction(
		0,
		common2.UnregisterCR,
		0,
		cancelPayload,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{Value: 4999 * 100000000},
		},
		0,
		[]*program.Program{
			{Code: code},
		},
	)
	s.Chain.GetCRCommittee().ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: s.CurrentHeight,
		},
		Transactions: []interfaces.Transaction{canceltx},
	}, nil)
	s.CurrentHeight++
	s.True(candidate.State == crstate.Canceled, "canceled CR failed")

	publicKey2 := "030a26f8b4ab0ea219eb461d1e454ce5f0bd0d289a6a64ffc0743dab7bd5be0be9"
	pubKeyBytes2, _ := common.HexStringToBytes(publicKey2)
	pubkey2, _ := crypto.DecodePoint(pubKeyBytes2)
	code2, _ := contract.CreateStandardRedeemScript(pubkey2)

	s.CurrentHeight = 2160 + canceledHeight
	s.Chain.GetCRCommittee().ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: s.CurrentHeight,
		},
		Transactions: []interfaces.Transaction{},
	}, nil)

	// check a return cr deposit coin transaction with wrong code in voting period.
	rdTx.Programs()[0].Code = code2

	rdTx = CreateTransactionByType(rdTx, s.Chain)
	rdTx.SetReferences(references)
	err, _ := rdTx.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:signer must be candidate or member")

	// check a return cr deposit coin transaction with wrong output amount.
	rdTx.Outputs()[0].Value = 5000 * 100000000
	s.CurrentHeight = 2160 + canceledHeight
	err, _ = rdTx.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:signer must be candidate or member")

	// check a correct return cr deposit coin transaction.
	rdTx.Outputs()[0].Value = 4999 * 100000000
	rdTx.Programs()[0].Code = code
	s.CurrentHeight = s.Chain.GetParams().CRCommitteeStartHeight
	err, _ = rdTx.SpecialContextCheck()
	s.NoError(err)

	// return CR deposit coin.
	rdTx.Programs()[0].Code = code
	s.Chain.GetCRCommittee().ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: s.CurrentHeight,
		},
		Transactions: []interfaces.Transaction{
			rdTx,
		},
	}, nil)
	s.CurrentHeight++

	// check a return cr deposit coin transaction with the amount has returned.
	err, _ = rdTx.SpecialContextCheck()
	s.NoError(err)

}

func (s *txValidatorTestSuite) TestCheckOutputPayload() {
	publicKeyStr1 := "02b611f07341d5ddce51b5c4366aca7b889cfe0993bd63fd47e944507292ea08dd"
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	programHash, _ := common.Uint168FromAddress("EJMzC16Eorq9CuFCGtyMrq4Jmgw9jYCHQR")

	outputs := []*common2.Output{
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: *programHash,
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 0,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: *programHash,
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 0,
				Contents: []outputpayload.VoteContent{
					{
						VoteType:       outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: *programHash,
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 0,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
							{publicKey1, 0},
						},
					},
				},
			},
		},
		{
			AssetID:     common.Uint256{},
			Value:       1.0,
			OutputLock:  0,
			ProgramHash: common.Uint168{123},
			Type:        common2.OTVote,
			Payload: &outputpayload.VoteOutput{
				Version: 0,
				Contents: []outputpayload.VoteContent{
					{
						VoteType: outputpayload.Delegate,
						CandidateVotes: []outputpayload.CandidateVotes{
							{publicKey1, 0},
						},
					},
				},
			},
		},
	}

	err := blockchain.CheckOutputPayload(common2.TransferAsset, outputs[0])
	s.NoError(err)

	err = blockchain.CheckOutputPayload(common2.RechargeToSideChain, outputs[0])
	s.EqualError(err, "transaction type dose not match the output payload type")

	err = blockchain.CheckOutputPayload(common2.TransferAsset, outputs[1])
	s.EqualError(err, "invalid public key count")

	err = blockchain.CheckOutputPayload(common2.TransferAsset, outputs[2])
	s.EqualError(err, "duplicate candidate")

	err = blockchain.CheckOutputPayload(common2.TransferAsset, outputs[3])
	s.EqualError(err, "output address should be standard")
}

func (s *txValidatorTestSuite) TestCheckVoteOutputs() {

	references := make(map[*common2.Input]common2.Output)
	outputs := []*common2.Output{{Type: common2.OTNone}}
	s.NoError(s.Chain.CheckVoteOutputs(0, outputs, references, nil, nil, nil))

	publicKey1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	publicKey2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	publicKey3 := "031e12374bae471aa09ad479f66c2306f4bcc4ca5b754609a82a1839b94b4721b9"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	privateKeyStr3 := "94396a69462208b8fd96d83842855b867d3b0e663203cb31d0dfaec0362ec034"

	registerCRTxn1 := s.getRegisterCRTx(publicKey1, privateKeyStr1,
		"nickName1", payload.CRInfoVersion, &common.Uint168{})
	registerCRTxn2 := s.getRegisterCRTx(publicKey2, privateKeyStr2,
		"nickName2", payload.CRInfoVersion, &common.Uint168{})
	registerCRTxn3 := s.getRegisterCRTx(publicKey3, privateKeyStr3,
		"nickName3", payload.CRInfoVersion, &common.Uint168{})

	s.CurrentHeight = 1
	s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams()))
	s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
		GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
		GetUTXO:                          s.Chain.GetDB().GetFFLDB().GetUTXO,
		GetHeight:                        func() uint32 { return s.CurrentHeight },
		CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
	})
	block := &types.Block{
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
		Header: common2.Header{Height: s.CurrentHeight},
	}
	s.Chain.GetCRCommittee().ProcessBlock(block, nil)
	code1 := getCodeByPubKeyStr(publicKey1)
	code2 := getCodeByPubKeyStr(publicKey2)
	code3 := getCodeByPubKeyStr(publicKey3)

	candidate1, _ := common.HexStringToBytes(publicKey1)
	candidate2, _ := common.HexStringToBytes(publicKey2)
	candidateCID1 := getCID(code1)
	candidateCID2 := getCID(code2)
	candidateCID3 := getCID(code3)

	producersMap := make(map[string]struct{})
	producersMap[publicKey1] = struct{}{}
	producersMap2 := make(map[string]uint32)
	producersMap2[publicKey1] = 0
	crsMap := make(map[common.Uint168]struct{})

	crsMap[*candidateCID1] = struct{}{}
	crsMap[*candidateCID3] = struct{}{}

	hashStr := "21c5656c65028fe21f2222e8f0cd46a1ec734cbdb6"
	hashByte, _ := common.HexStringToBytes(hashStr)
	hash, _ := common.Uint168FromBytes(hashByte)

	// Check vote output of v0 with delegate type and wrong output program hash
	outputs1 := []*common2.Output{{Type: common2.OTNone}}
	outputs1 = append(outputs1, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 0,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.Delegate,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate1, 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs1, references, producersMap, nil, crsMap),
		"the output address of vote tx should exist in its input")

	// Check vote output of v0 with crc type and with wrong output program hash
	outputs2 := []*common2.Output{{Type: common2.OTNone}}
	outputs2 = append(outputs2, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.CRC,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID3.Bytes(), 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs2, references, producersMap, nil, crsMap),
		"the output address of vote tx should exist in its input")

	// Check vote output of v0 with crc type and with wrong output program hash
	outputs20 := []*common2.Output{{Type: common2.OTNone}}
	outputs20 = append(outputs20, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.CRCProposal,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID3.Bytes(), 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs20, references, producersMap, nil, crsMap),
		"the output address of vote tx should exist in its input")

	// Check vote output of v0 with crc type and with wrong output program hash
	outputs21 := []*common2.Output{{Type: common2.OTNone}}
	outputs21 = append(outputs21, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.CRCImpeachment,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID3.Bytes(), 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs21, references, producersMap, nil, crsMap),
		"the output address of vote tx should exist in its input")

	// Check vote output of v0 with crc type and with wrong output program hash
	outputs22 := []*common2.Output{{Type: common2.OTNone}}
	outputs22 = append(outputs22, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.DposV2,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID3.Bytes(), 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs22, references, producersMap, nil, crsMap),
		"the output address of vote tx should exist in its input")

	// Check vote output of v1 with wrong output program hash
	outputs3 := []*common2.Output{{Type: common2.OTNone}}
	outputs3 = append(outputs3, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.Delegate,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate1, 0},
					},
				},
				{
					VoteType: outputpayload.CRC,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID3.Bytes(), 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs3, references, producersMap, nil, crsMap),
		"the output address of vote tx should exist in its input")

	references[&common2.Input{}] = common2.Output{
		ProgramHash: *hash,
	}

	// Check vote output of v0 with delegate type and invalid candidate
	outputs4 := []*common2.Output{{Type: common2.OTNone}}
	outputs4 = append(outputs4, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 0,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.Delegate,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate2, 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs4, references, producersMap, nil, crsMap),
		"invalid vote output payload producer candidate: "+publicKey2)

	// Check vote output of v0 with delegate type and invalid candidate
	outputs23 := []*common2.Output{{Type: common2.OTNone}}
	outputs23 = append(outputs23, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 0,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.DposV2,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate2, 0},
					},
				},
			},
		},
		OutputLock: 0,
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs23, references, producersMap, producersMap2, crsMap),
		"invalid vote output payload producer candidate: "+publicKey2)

	outputs23 = []*common2.Output{{Type: common2.OTNone}}
	outputs23 = append(outputs23, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 0,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.DposV2,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate1, 0},
					},
				},
			},
		},
		OutputLock: 0,
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs23, references, producersMap, producersMap2, crsMap),
		fmt.Sprintf("payload VoteDposV2Version not support vote DposV2"))

	outputs23 = []*common2.Output{{Type: common2.OTNone}}
	outputs23 = append(outputs23, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: outputpayload.VoteDposV2Version,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.DposV2,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate1, 0},
					},
				},
			},
		},
		OutputLock: 0,
	})
	s.NoError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs23, references, producersMap, producersMap2, crsMap))

	// Check vote output v0 with correct output program hash
	s.NoError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs1, references, producersMap, nil, crsMap))
	s.NoError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs2, references, producersMap, nil, crsMap))
	s.NoError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs3, references, producersMap, nil, crsMap))

	// Check vote output of v0 with crc type and invalid candidate
	outputs5 := []*common2.Output{{Type: common2.OTNone}}
	outputs5 = append(outputs5, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 0,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.CRC,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID2.Bytes(), 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs5, references, producersMap, nil, crsMap),
		"payload VoteProducerVersion not support vote CR")

	// Check vote output of v1 with crc type and invalid candidate
	outputs6 := []*common2.Output{{Type: common2.OTNone}}
	outputs6 = append(outputs6, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.CRC,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID2.Bytes(), 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs6, references, producersMap, nil, crsMap),
		"invalid vote output payload CR candidate: "+candidateCID2.String())

	// Check vote output of v0 with invalid candidate
	outputs7 := []*common2.Output{{Type: common2.OTNone}}
	outputs7 = append(outputs7, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Payload: &outputpayload.VoteOutput{
			Version: 0,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.Delegate,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate1, 0},
					},
				},
				{
					VoteType: outputpayload.CRC,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID2.Bytes(), 0},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs7, references, producersMap, nil, crsMap),
		"payload VoteProducerVersion not support vote CR")

	// Check vote output of v1 with delegate type and wrong votes
	outputs8 := []*common2.Output{{Type: common2.OTNone}}
	outputs8 = append(outputs8, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Value:       common.Fixed64(10),
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.Delegate,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate1, 20},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs8, references, producersMap, nil, crsMap),
		"votes larger than output amount")

	// Check vote output of v1 with crc type and wrong votes
	outputs9 := []*common2.Output{{Type: common2.OTNone}}
	outputs9 = append(outputs9, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Value:       common.Fixed64(10),
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.CRC,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID1.Bytes(), 10},
						{candidateCID3.Bytes(), 10},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs9, references, producersMap, nil, crsMap),
		"total votes larger than output amount")

	// Check vote output of v1 with wrong votes
	outputs10 := []*common2.Output{{Type: common2.OTNone}}
	outputs10 = append(outputs10, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Value:       common.Fixed64(10),
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.Delegate,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate1, 20},
					},
				},
				{
					VoteType: outputpayload.CRC,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID3.Bytes(), 20},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs10, references, producersMap, nil, crsMap),
		"votes larger than output amount")

	// Check vote output v1 with correct votes
	outputs11 := []*common2.Output{{Type: common2.OTNone}}
	outputs11 = append(outputs11, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Value:       common.Fixed64(10),
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.Delegate,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate1, 10},
					},
				},
				{
					VoteType: outputpayload.CRC,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID3.Bytes(), 10},
					},
				},
			},
		},
	})
	s.NoError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs11, references, producersMap, nil, crsMap))

	// Check vote output of v1 with wrong votes
	outputs12 := []*common2.Output{{Type: common2.OTNone}}
	outputs12 = append(outputs12, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Value:       common.Fixed64(10),
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.Delegate,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidate1, 1},
					},
				},
				{
					VoteType: outputpayload.CRC,
					CandidateVotes: []outputpayload.CandidateVotes{
						{candidateCID3.Bytes(), 1},
					},
				},
			},
		},
	})
	s.NoError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs12, references, producersMap, nil, crsMap))

	// Check vote output v1 with correct votes
	proposalHashStr1 := "5df40cc0a4c6791acb5ebe89a96dd4f3fe21c94275589a65357406216a27ae36"
	proposalHash1, _ := common.Uint256FromHexString(proposalHashStr1)
	outputs13 := []*common2.Output{{Type: common2.OTNone}}
	outputs13 = append(outputs13, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Value:       common.Fixed64(10),
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.CRCProposal,
					CandidateVotes: []outputpayload.CandidateVotes{
						{proposalHash1.Bytes(), 10},
					},
				},
			},
		},
	})
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[*proposalHash1] =
		&crstate.ProposalState{Status: 1}
	s.NoError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs13, references, producersMap, nil, crsMap))

	// Check vote output of v1 with wrong votes
	proposalHashStr2 := "9c5ab8998718e0c1c405a719542879dc7553fca05b4e89132ec8d0e88551fcc0"
	proposalHash2, _ := common.Uint256FromHexString(proposalHashStr2)
	outputs14 := []*common2.Output{{Type: common2.OTNone}}
	outputs14 = append(outputs14, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: *hash,
		Value:       common.Fixed64(10),
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.CRCProposal,
					CandidateVotes: []outputpayload.CandidateVotes{
						{proposalHash2.Bytes(), 10},
					},
				},
			},
		},
	})
	s.EqualError(s.Chain.CheckVoteOutputs(config.DefaultParams.CRVotingStartHeight,
		outputs14, references, producersMap, nil, crsMap),
		"invalid CRCProposal: c0fc5185e8d0c82e13894e5ba0fc5375dc79285419a705c4c1e0188799b85a9c")
}

func (s *txValidatorTestSuite) TestCheckOutputProgramHash() {
	programHash := common.Uint168{}

	// empty program hash should pass
	s.NoError(blockchain.CheckOutputProgramHash(88813, programHash))

	// prefix standard program hash should pass
	programHash[0] = uint8(contract.PrefixStandard)
	s.NoError(blockchain.CheckOutputProgramHash(88813, programHash))

	// prefix multisig program hash should pass
	programHash[0] = uint8(contract.PrefixMultiSig)
	s.NoError(blockchain.CheckOutputProgramHash(88813, programHash))

	// prefix crosschain program hash should pass
	programHash[0] = uint8(contract.PrefixCrossChain)
	s.NoError(blockchain.CheckOutputProgramHash(88813, programHash))

	// other prefix program hash should not pass
	programHash[0] = 0x34
	s.Error(blockchain.CheckOutputProgramHash(88813, programHash))

	// other prefix program hash should pass in old version
	programHash[0] = 0x34
	s.NoError(blockchain.CheckOutputProgramHash(88811, programHash))
}

func (s *txValidatorTestSuite) TestCreateCRCAppropriationTransaction() {
	crAddress := "ERyUmNH51roR9qfru37Kqkaok2NghR7L5U"
	crcFoundation, _ := common.Uint168FromAddress(crAddress)

	s.Chain.GetParams().CRAssetsAddress = *crcFoundation
	crcCommiteeAddressStr := "ESq12oQrvGqHfTkEDYJyR9MxZj1NMnonjo"

	crcCommiteeAddressHash, _ := common.Uint168FromAddress(crcCommiteeAddressStr)
	s.Chain.GetParams().CRExpensesAddress = *crcCommiteeAddressHash

	s.CurrentHeight = 1
	s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams()))
	s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
		GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
		GetUTXO:                          s.Chain.GetDB().GetFFLDB().GetUTXO,
		GetHeight:                        func() uint32 { return s.CurrentHeight },
		CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
	})

	var txOutputs []*common2.Output
	txOutput := &common2.Output{
		AssetID:     *elaact.SystemAssetID,
		ProgramHash: *crcFoundation,
		Value:       common.Fixed64(0),
		OutputLock:  0,
		Type:        common2.OTNone,
		Payload:     &outputpayload.DefaultOutput{},
	}
	for i := 1; i < 5; i++ {
		txOutPutNew := *txOutput
		txOutPutNew.Value = common.Fixed64(i * 100)
		txOutputs = append(txOutputs, &txOutPutNew)
	}

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{},
		[]*common2.Input{},
		txOutputs,
		0,
		[]*program.Program{},
	)

	txOutputs = nil
	txOutputCoinBase := *txOutput
	txOutputCoinBase.Value = common.Fixed64(500)
	txOutputCoinBase.OutputLock = uint32(100)
	txOutputs = append(txOutputs, &txOutputCoinBase)

	txnCoinBase := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CoinBase,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{},
		[]*common2.Input{},
		txOutputs,
		0,
		[]*program.Program{},
	)

	block := &types.Block{
		Transactions: []interfaces.Transaction{
			txn,
			txnCoinBase,
		},
		Header: common2.Header{
			Height:   1,
			Previous: s.Chain.GetParams().GenesisBlock.Hash(),
		},
	}
	hash := block.Hash()
	node, _ := s.Chain.LoadBlockNode(&block.Header, &hash)
	s.Chain.GetDB().SaveBlock(block, node, nil, blockchain.CalcPastMedianTime(node))
	txCrcAppropriation, _, _ := s.Chain.CreateCRCAppropriationTransaction()
	s.NotNil(txCrcAppropriation)
}

func (s *txValidatorTestSuite) TestCreateCRClaimDposV2Transaction() {
	publicKeyStr1 := "02ca89a5fe6213da1b51046733529a84f0265abac59005f6c16f62330d20f02aeb"
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	pk, _ := crypto.DecodePoint(publicKey1)

	privateKeyStr1 := "7a50d2b036d64fcb3d344cee429f61c4a3285a934c45582b26e8c9227bc1f33a"
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)

	redeemScript, _ := contract.CreateStandardRedeemScript(pk)

	buf := new(bytes.Buffer)
	apPayload := &payload.DPoSV2ClaimReward{
		Value: common.Fixed64(100000000),
		Code:  redeemScript,
	}

	apPayload.SerializeUnsigned(buf, payload.ActivateProducerVersion)
	signature, _ := crypto.Sign(privateKey1, buf.Bytes())
	apPayload.Signature = signature

	// create program
	var txProgram = &program.Program{
		Code:      redeemScript,
		Parameter: nil,
	}
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.DposV2ClaimReward,
		0,
		apPayload,
		nil,
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{txProgram})
	tx := txn.(*transaction.DPoSV2ClaimRewardTransaction)
	tx.DefaultChecker.SetParameters(&transaction.TransactionParameters{
		BlockChain: s.Chain,
		Config:     s.Chain.GetParams(),
	})

	err, _ := tx.SpecialContextCheck()
	s.EqualError(err.(errors.ELAError).InnerError(), "can not claim reward before dposv2startheight")

	param := s.Chain.GetParams()
	param.DPoSV2StartHeight = 10
	tx.DefaultChecker.SetParameters(&transaction.TransactionParameters{
		BlockChain:  s.Chain,
		Config:      param,
		BlockHeight: 100,
	})
	err, _ = tx.SpecialContextCheck()
	s.EqualError(err.(errors.ELAError).InnerError(), "no reward to claim for such address")

	bc := s.Chain
	bc.GetState().DposV2RewardInfo["ERyUmNH51roR9qfru37Kqkaok2NghR7L5U"] = 100
	tx.DefaultChecker.SetParameters(&transaction.TransactionParameters{
		BlockChain:  bc,
		Config:      param,
		BlockHeight: 100,
	})

	err, _ = tx.SpecialContextCheck()
	s.EqualError(err.(errors.ELAError).InnerError(), "claim reward exceeded , max claim reward 0.00000100")

	bc = s.Chain
	bc.GetState().DposV2RewardInfo["ERyUmNH51roR9qfru37Kqkaok2NghR7L5U"] = 10000000000
	tx.DefaultChecker.SetParameters(&transaction.TransactionParameters{
		BlockChain:  bc,
		Config:      param,
		BlockHeight: 100,
	})
	err, _ = tx.SpecialContextCheck()
	s.NoError(err)
}

func TestTxValidatorSuite(t *testing.T) {
	suite.Run(t, new(txValidatorTestSuite))
}

func newCoinBaseTransaction(coinBasePayload *payload.CoinBase,
	currentHeight uint32) interfaces.Transaction {
	txn := functions.CreateTransaction(
		0,
		common2.CoinBase,
		payload.CoinBaseVersion,
		coinBasePayload,
		[]*common2.Attribute{},
		[]*common2.Input{
			{
				Previous: common2.OutPoint{
					TxID:  common.EmptyHash,
					Index: math.MaxUint16,
				},
				Sequence: math.MaxUint32,
			},
		},
		[]*common2.Output{},
		currentHeight,
		[]*program.Program{},
	)

	return txn
}

func (a *txValidatorTestSuite) createNextTurnDPOSInfoTransaction(crcArbiters, normalDPOSArbiters [][]byte) interfaces.Transaction {

	var nextTurnDPOSInfo payload.NextTurnDPOSInfo
	for _, v := range crcArbiters {
		nextTurnDPOSInfo.CRPublicKeys = append(nextTurnDPOSInfo.CRPublicKeys, v)
	}
	for _, v := range normalDPOSArbiters {
		nextTurnDPOSInfo.DPOSPublicKeys = append(nextTurnDPOSInfo.DPOSPublicKeys, v)
	}
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.NextTurnDPOSInfo,
		0,
		&nextTurnDPOSInfo,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	return txn
}
func (s *txValidatorTestSuite) TestHostPort() {
	seeds := "one.elastos.cn:20821,two.elastos.cn:20822"
	seedArr := strings.Split(seeds, ",")
	for _, seed := range seedArr {
		host, _, err := net.SplitHostPort(seed)
		if err != nil {
			host = seed
		}
		s.True(payload.SeedRegexp.MatchString(host), seed+" not correct")
	}
}

func (s *txValidatorTestSuite) TestArbitersAccumulateReward() {
	tx := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CoinBase,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	tx.SetOutputs([]*common2.Output{
		{ProgramHash: blockchain.FoundationAddress, Value: 0},
		{ProgramHash: common.Uint168{}, Value: 100},
	})
	ownerPubKeyStr := "0306e3deefee78e0e25f88e98f1f3290ccea98f08dd3a890616755f1a066c4b9b8"
	nodePubKeyStr := "0250c5019a00f8bb4fd59bb6d613c70a39bb3026b87cfa247fd26f59fd04987855"

	nodePubKey, err := hex.DecodeString(ownerPubKeyStr)
	if err != nil {
		fmt.Println("err", err)
	}

	type fields struct {
		State                      *state.State
		ChainParams                *config.Params
		CRCommittee                *crstate.Committee
		bestHeight                 func() uint32
		bestBlockHash              func() *common.Uint256
		getBlockByHeight           func(uint32) (*types.Block, error)
		mtx                        sync.Mutex
		started                    bool
		DutyIndex                  int
		CurrentReward              state.RewardData
		NextReward                 state.RewardData
		CurrentArbitrators         []state.ArbiterMember
		CurrentCandidates          []state.ArbiterMember
		nextArbitrators            []state.ArbiterMember
		nextCandidates             []state.ArbiterMember
		CurrentCRCArbitersMap      map[common.Uint168]state.ArbiterMember
		nextCRCArbitersMap         map[common.Uint168]state.ArbiterMember
		nextCRCArbiters            []state.ArbiterMember
		crcChangedHeight           uint32
		accumulativeReward         common.Fixed64
		finalRoundChange           common.Fixed64
		clearingHeight             uint32
		arbitersRoundReward        map[common.Uint168]common.Fixed64
		illegalBlocksPayloadHashes map[common.Uint256]interface{}
		Snapshots                  map[uint32][]*state.CheckPoint
		SnapshotKeysDesc           []uint32
		forceChanged               bool
		DposV2ActiveHeight         uint32
		dposV2EffectedProducers    map[string]*state.Producer
		History                    *utils.History
	}
	type args struct {
		block   *types.Block
		confirm *payload.Confirm
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
		{
			"Normal",
			fields{
				ChainParams: &config.Params{
					PublicDPOSHeight:      1,
					CRVotingStartHeight:   1,
					NewELAIssuanceHeight:  1,
					HalvingRewardHeight:   1,
					HalvingRewardInterval: 1,
					CRMemberCount:         6,
					GeneralArbiters:       12,
				},
				DposV2ActiveHeight:      1,
				dposV2EffectedProducers: make(map[string]*state.Producer),
				forceChanged:            false,
				History:                 utils.NewHistory(10),
			},
			args{
				block: &types.Block{
					Header: common2.Header{
						Height: 20,
					},
					Transactions: []interfaces.Transaction{
						tx,
					},
				},
				confirm: &payload.Confirm{
					Proposal: payload.DPOSProposal{
						Sponsor: nodePubKey,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			a := &state.Arbiters{
				//State:                      tt.fields.State,
				ChainParams: tt.fields.ChainParams,
				CRCommittee: tt.fields.CRCommittee,
				//bestHeight:                 tt.fields.bestHeight,
				//bestBlockHash:              tt.fields.bestBlockHash,
				//getBlockByHeight:           tt.fields.getBlockByHeight,
				//mtx:                        tt.fields.mtx,
				//started:                    tt.fields.started,
				DutyIndex:             tt.fields.DutyIndex,
				CurrentReward:         tt.fields.CurrentReward,
				NextReward:            tt.fields.NextReward,
				CurrentArbitrators:    tt.fields.CurrentArbitrators,
				CurrentCandidates:     tt.fields.CurrentCandidates,
				CurrentCRCArbitersMap: tt.fields.CurrentCRCArbitersMap,
				Snapshots:             tt.fields.Snapshots,
				SnapshotKeysDesc:      tt.fields.SnapshotKeysDesc,
				State: &state.State{
					StateKeyFrame: &state.StateKeyFrame{
						DposV2EffectedProducers: tt.fields.dposV2EffectedProducers,
						NodeOwnerKeys:           make(map[string]string),
						DposV2RewardInfo:        make(map[string]common.Fixed64),
						ActivityProducers:       make(map[string]*state.Producer),
						DPoSV2ActiveHeight:      tt.fields.DposV2ActiveHeight,
					},
					ChainParams: tt.fields.ChainParams,
				},
				History: tt.fields.History,
			}

			a.State.NodeOwnerKeys[nodePubKeyStr] = ownerPubKeyStr
			//should be more than a.ChainParams.GeneralArbiters*3/2
			for i := 0; i < 20; i++ {
				a.State.DposV2EffectedProducers[randomString()] = nil
			}
			a.State.ActivityProducers[ownerPubKeyStr] = &state.Producer{}
			//CurrentCRCArbitersMap
			a.AccumulateReward(tt.args.block, tt.args.confirm)
			a.History.Commit(tt.args.block.Height)
			if a.State.DposV2RewardInfo["ET54cpnGG4JHeRatvPij6hGV6zN18eVSSj"] != 102 {
				t.Errorf("DposV2RewardInfo() addr %v, want %v", "ET54cpnGG4JHeRatvPij6hGV6zN18eVSSj", 102)
			}
		})
	}
}

//func (s *txValidatorTestSuite) TestCheckNextTurnDPOSInfoTx() {
//	//var nextTurnDPOSInfo payload.NextTurnDPOSInfo
//	crc1PubKey, _ := common.HexStringToBytes("03e435ccd6073813917c2d841a0815d21301ec3286bc1412bb5b099178c68a10b6")
//	crc2PubKey, _ := common.HexStringToBytes("038a1829b4b2bee784a99bebabbfecfec53f33dadeeeff21b460f8b4fc7c2ca771")
//
//	normalArbitratorsStr := []string{
//		"023a133480176214f88848c6eaa684a54b316849df2b8570b57f3a917f19bbc77a",
//		"030a26f8b4ab0ea219eb461d1e454ce5f0bd0d289a6a64ffc0743dab7bd5be0be9",
//		"0288e79636e41edce04d4fa95d8f62fed73a76164f8631ccc42f5425f960e4a0c7",
//		"03e281f89d85b3a7de177c240c4961cb5b1f2106f09daa42d15874a38bbeae85dd",
//		"0393e823c2087ed30871cbea9fa5121fa932550821e9f3b17acef0e581971efab0",
//	}
//	normal1PubKey, _ := common.HexStringToBytes(normalArbitratorsStr[0])
//	normal2PubKey, _ := common.HexStringToBytes(normalArbitratorsStr[1])
//
//	crcArbiters := [][]byte{
//		crc1PubKey,
//		crc2PubKey,
//	}
//
//	normalDPOSArbiters := [][]byte{
//		normal1PubKey,
//		normal2PubKey,
//	}
//	nextTurnDPOSInfoTx := s.createNextTurnDPOSInfoTransaction(crcArbiters, normalDPOSArbiters)
//	// Check correct transaction.
//	//DefaultLedger.Arbitrators.SetNeedNextTurnDPOSInfo(true)
//	err := s.Chain.checkNextTurnDPOSInfoTransaction(nextTurnDPOSInfoTx)
//	s.NoError(err)
//
//	// Appropriation transaction already exist.
//	s.Chain.GetCRCommittee().NeedAppropriation = false
//	err = s.Chain.checkNextTurnDPOSInfoTransaction(nextTurnDPOSInfoTx)
//	s.EqualError(err, "should have no appropriation transaction")
//
//}

func CreateTransactionByType(ori interfaces.Transaction, chain *blockchain.BlockChain) interfaces.Transaction {
	tx := functions.CreateTransaction(
		ori.Version(),
		ori.TxType(),
		ori.PayloadVersion(),
		ori.Payload(),
		ori.Attributes(),
		ori.Inputs(),
		ori.Outputs(),
		ori.LockTime(),
		ori.Programs(),
	)

	tx.SetParameters(&transaction.TransactionParameters{
		Transaction: tx,
		BlockHeight: chain.BestChain.Height,
		TimeStamp:   chain.BestChain.Timestamp,
		Config:      chain.GetParams(),
		BlockChain:  chain,
	})

	return tx
}
