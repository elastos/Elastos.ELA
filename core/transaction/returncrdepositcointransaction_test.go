// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
)

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
