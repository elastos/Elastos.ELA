package transaction

import (
	"crypto/rand"
	"fmt"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"math"
	mrand "math/rand"
)

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
		{AssetID: core.ELAAssetID, ProgramHash: s.foundationAddress, Value: outputValue1},
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
		{AssetID: core.ELAAssetID, ProgramHash: s.foundationAddress, Value: outputValue1},
		{AssetID: core.ELAAssetID, ProgramHash: common.Uint168{}, Value: outputValue2},
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
		{AssetID: core.ELAAssetID, ProgramHash: s.foundationAddress},
		{AssetID: core.ELAAssetID, ProgramHash: s.foundationAddress},
	})
	err := s.Chain.CheckTransactionOutput(tx, s.HeightVersion1)
	s.NoError(err)

	// outputs < 2
	tx.SetOutputs([]*common2.Output{
		{AssetID: core.ELAAssetID, ProgramHash: s.foundationAddress},
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
	totalReward := config.DefaultParams.PowConfiguration.RewardPerBlock
	fmt.Printf("Block reward amount %s", totalReward.String())
	foundationReward := common.Fixed64(float64(totalReward) * 0.3)
	fmt.Printf("Foundation reward amount %s", foundationReward.String())
	tx.SetOutputs([]*common2.Output{
		{AssetID: core.ELAAssetID, ProgramHash: s.foundationAddress, Value: foundationReward},
		{AssetID: core.ELAAssetID, ProgramHash: common.Uint168{}, Value: totalReward - foundationReward},
	})
	err = s.Chain.CheckTransactionOutput(tx, s.HeightVersion1)
	s.NoError(err)

	// reward to foundation in coinbase < 30% (CheckTxOut version)
	foundationReward = common.Fixed64(float64(totalReward) * 0.299999)
	fmt.Printf("Foundation reward amount %s", foundationReward.String())
	tx.SetOutputs([]*common2.Output{
		{AssetID: core.ELAAssetID, ProgramHash: s.foundationAddress, Value: foundationReward},
		{AssetID: core.ELAAssetID, ProgramHash: common.Uint168{}, Value: totalReward - foundationReward},
	})
	err = s.Chain.CheckTransactionOutput(tx, s.HeightVersion1)
	s.EqualError(err, "reward to foundation in coinbase < 30%")

	// normal transaction
	tx = buildTx()
	for _, output := range tx.Outputs() {
		output.AssetID = core.ELAAssetID
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
			AssetID:     core.ELAAssetID,
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
		output.AssetID = core.ELAAssetID
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

func (s *txValidatorTestSuite) TestCheckCoinbaseTransaction() {
	// coinbase
	{
		tx := newCoinBaseTransaction(new(payload.CoinBase), 0)
		randomAddr := randomUint168()
		tx.SetOutputs([]*common2.Output{
			{AssetID: core.ELAAssetID, ProgramHash: *randomAddr},
			{AssetID: core.ELAAssetID, ProgramHash: s.foundationAddress},
		})

		tx = CreateTransactionByType(tx, s.Chain)
		err, _ := tx.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: output invalid:first output address should be foundation address")

		tx.SetOutputs([]*common2.Output{
			{AssetID: core.ELAAssetID, ProgramHash: s.foundationAddress},
			{AssetID: core.ELAAssetID, ProgramHash: s.foundationAddress},
		})

		err, _ = tx.SpecialContextCheck()
		s.NoError(err)
	}

	{
		tx := newCoinBaseTransaction(new(payload.CoinBase), 0)
		randomAddr := randomUint168()
		tx.SetOutputs([]*common2.Output{
			{AssetID: core.ELAAssetID, ProgramHash: *randomAddr},
			{AssetID: core.ELAAssetID, ProgramHash: s.foundationAddress},
		})

		s.Chain.GetBestChain().Height = 1000000
		s.Chain.GetState().ConsensusAlgorithm = 0x00
		tx = CreateTransactionByType(tx, s.Chain)
		err, _ := tx.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: output invalid:first output address should be CR assets address")

		addr := s.Chain.GetParams().CRConfiguration.CRAssetsProgramHash
		tx.SetOutputs([]*common2.Output{
			{AssetID: core.ELAAssetID, ProgramHash: *addr},
			{AssetID: core.ELAAssetID, ProgramHash: s.foundationAddress},
		})

		err, _ = tx.SpecialContextCheck()
		s.NoError(err)

		s.Chain.GetBestChain().Height = 1000000
		s.Chain.GetState().ConsensusAlgorithm = 0x01
		tx = CreateTransactionByType(tx, s.Chain)
		err, _ = tx.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: output invalid:first output address should be DestroyAddress in POW consensus algorithm")

		addr = s.Chain.GetParams().DestroyELAProgramHash
		tx.SetOutputs([]*common2.Output{
			{AssetID: core.ELAAssetID, ProgramHash: *addr},
			{AssetID: core.ELAAssetID, ProgramHash: s.foundationAddress},
		})

		err, _ = tx.SpecialContextCheck()
		s.NoError(err)
	}
}
