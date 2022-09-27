package transaction

import (
	elaact "github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
)

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
