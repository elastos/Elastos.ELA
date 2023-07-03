// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"github.com/elastos/Elastos.ELA/common/config"
	"path/filepath"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/checkpoint"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
)

func (s *txValidatorTestSuite) TestCheckTransactionDepositUTXO() {
	references := make(map[*common2.Input]common2.Output)
	input := &common2.Input{}
	// Use the deposit UTXO in a TransferAsset transaction
	depositHash, _ := common.Uint168FromAddress("DVgnDnVfPVuPa2y2E4JitaWjWgRGJDuyrD")
	depositOutput := common2.Output{
		ProgramHash: *depositHash,
	}
	references[input] = depositOutput

	txn, _ := GetTransaction(common2.TransferAsset)
	err := blockchain.CheckTransactionDepositUTXO(txn, references)
	s.EqualError(err, "only the ReturnDepositCoin and "+
		"ReturnCRDepositCoin transaction can use the deposit UTXO")

	// Use the deposit UTXO in a ReturnDepositCoin transaction
	txn, _ = GetTransaction(common2.ReturnDepositCoin)
	err = blockchain.CheckTransactionDepositUTXO(txn, references)
	s.NoError(err)

	// Use the standard UTXO in a ReturnDepositCoin transaction
	normalHash, _ := common.Uint168FromAddress("EJMzC16Eorq9CuFCGtyMrq4Jmgw9jYCHQR")
	normalOutput := common2.Output{
		ProgramHash: *normalHash,
	}
	references[input] = normalOutput
	txn, _ = GetTransaction(common2.ReturnDepositCoin)
	err = blockchain.CheckTransactionDepositUTXO(txn, references)
	s.EqualError(err, "the ReturnDepositCoin and ReturnCRDepositCoin "+
		"transaction can only use the deposit UTXO")

	// Use the deposit UTXO in a ReturnDepositCoin transaction
	references[input] = depositOutput
	txn, _ = GetTransaction(common2.ReturnCRDepositCoin)
	err = blockchain.CheckTransactionDepositUTXO(txn, references)
	s.NoError(err)

	references[input] = normalOutput
	txn, _ = GetTransaction(common2.ReturnCRDepositCoin)
	err = blockchain.CheckTransactionDepositUTXO(txn, references)
	s.EqualError(err, "the ReturnDepositCoin and ReturnCRDepositCoin "+
		"transaction can only use the deposit UTXO")
}

func (s *txValidatorTestSuite) TestCheckReturnDepositCoinTransaction() {
	s.CurrentHeight = 1
	ckpManager := checkpoint.NewManager(&config.DefaultParams)
	ckpManager.SetDataPath(filepath.Join(config.DefaultParams.DataDir, "checkpoints"))
	s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams(), ckpManager))
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
			OwnerKey:      publicKey,
			NodePublicKey: publicKey,
			NickName:      randomString(),
			Url:           randomString(),
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
			OwnerKey: publicKey,
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

	s.CurrentHeight += s.Chain.GetParams().CRConfiguration.DepositLockupBlocks
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

	// check a correct return deposit coin
	rdTx.Outputs()[0].Value = 4999 * 100000000
	err, _ = rdTx.SpecialContextCheck()
	s.NoError(err)
}
