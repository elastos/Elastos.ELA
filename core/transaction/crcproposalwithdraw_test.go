package transaction

import (
	"bytes"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
	"math"
)

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
	txn.SetParameters(&TransactionParameters{
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
	txn.SetParameters(&TransactionParameters{
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
	txn.SetParameters(&TransactionParameters{
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
	txn.SetParameters(&TransactionParameters{
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
	txn.SetParameters(&TransactionParameters{
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
