package transaction

import (
	"bytes"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/checkpoint"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
	"path/filepath"
)

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
	txn.SetParameters(&TransactionParameters{
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
		ckpManager := checkpoint.NewManager(&config.DefaultParams)
		ckpManager.SetDataPath(filepath.Join(config.DefaultParams.DataDir, "checkpoints"))
		s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams(), ckpManager))
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
		txn.SetParameters(&TransactionParameters{
			Transaction: txn,
			BlockHeight: 0,
			TimeStamp:   s.Chain.BestChain.Timestamp,
			Config:      s.Chain.GetParams(),
			BlockChain:  s.Chain,
		})
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err, "transaction validate error: payload content invalid:insufficient deposit amount")

		s.Chain.GetState().GetProducer(publicKey1).SetTotalAmount(500100000000)
		s.Chain.GetParams().CRConfiguration.CRVotingStartHeight = 1
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
		ckpManager := checkpoint.NewManager(&config.DefaultParams)
		ckpManager.SetDataPath(filepath.Join(config.DefaultParams.DataDir, "checkpoints"))
		s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams(), ckpManager))
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
		s.Chain.GetParams().CRConfiguration.CRVotingStartHeight = 1
		s.Chain.BestChain.Height = 10
		txn = CreateTransactionByType(txn, s.Chain)
		err, _ = txn.SpecialContextCheck()
		s.NoError(err)
	}

}
