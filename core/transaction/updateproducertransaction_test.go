package transaction

import (
	"bytes"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
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
