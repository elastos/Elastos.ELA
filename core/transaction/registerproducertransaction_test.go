package transaction

import (
	"bytes"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/errors"
	math "math/rand"
)

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
		OwnerKey:      publicKey1,
		NodePublicKey: publicKey1,
		NickName:      "nickname 1",
		Url:           "http://www.elastos_test.com",
		Location:      1,
		NetAddress:    "127.0.0.1:20338",
	}
	rpSignBuf := new(bytes.Buffer)
	err := rpPayload.SerializeUnsigned(rpSignBuf, payload.ProducerInfoVersion)
	s.NoError(err)
	rpSig, err := crypto.Sign(privateKey1, rpSignBuf.Bytes())
	s.NoError(err)
	rpPayload.Signature = rpSig
	s.Chain.BestChain.Height = 0
	s.Chain.GetState().DPoSV2ActiveHeight = math.Uint32()
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
	txn.Payload().(*payload.ProducerInfo).OwnerKey = errPublicKey
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
	txn.Payload().(*payload.ProducerInfo).OwnerKey = publicKey2
	pk, _ := common.HexStringToBytes(config.DefaultParams.DPoSConfiguration.CRCArbiters[0])
	txn.Payload().(*payload.ProducerInfo).NodePublicKey = pk
	config.DefaultParams.PublicDPOSHeight = 0
	err, _ = txn.SpecialContextCheck()
	config.DefaultParams.PublicDPOSHeight = originHeight
	s.EqualError(err, "transaction validate error: payload content invalid:node public key can't equal with CRC")

	// check owner public key same with CRC
	txn.Payload().(*payload.ProducerInfo).NodePublicKey = publicKey2
	pk, _ = common.HexStringToBytes(config.DefaultParams.DPoSConfiguration.CRCArbiters[0])
	txn.Payload().(*payload.ProducerInfo).OwnerKey = pk
	config.DefaultParams.PublicDPOSHeight = 0
	err, _ = txn.SpecialContextCheck()
	config.DefaultParams.PublicDPOSHeight = originHeight
	s.EqualError(err, "transaction validate error: payload content invalid:owner public key can't equal with CRC")

	// Invalidates the signature in payload
	txn.Payload().(*payload.ProducerInfo).OwnerKey = publicKey2
	txn.Payload().(*payload.ProducerInfo).NodePublicKey = publicKey2
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid signature in payload")

	// Give a mismatching deposit address
	rpPayload.OwnerKey = publicKey1
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
		OwnerKey:      publicKey1,
		NodePublicKey: publicKey1,
		NickName:      "nickname 1",
		Url:           "http://www.elastos_test.com",
		Location:      1,
		NetAddress:    "127.0.0.1:20338",
		StakeUntil:    100000,
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
	tx := txn.(*RegisterProducerTransaction)
	param := s.Chain.GetParams()
	param.DPoSV2StartHeight = 10
	param.PublicDPOSHeight = 5
	s.Chain.Nodes = []*blockchain.BlockNode{
		{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {},
	}
	tx.DefaultChecker.SetParameters(&TransactionParameters{
		BlockChain: s.Chain,
		Config:     s.Chain.GetParams(),
	})

	err, _ = tx.SpecialContextCheck()
	s.NoError(err)

	// Give an invalid owner public key in payload
	txn.Payload().(*payload.ProducerInfo).OwnerKey = errPublicKey
	err, _ = tx.SpecialContextCheck()
	s.EqualError(err.(errors.ELAError).InnerError(), "invalid owner public key in payload")

	// check version when height is not higher than dposv2 height
	s.Chain.Nodes = []*blockchain.BlockNode{
		{}, {}, {}, {},
	}
	param.PublicDPOSHeight = 1
	txn.Payload().(*payload.ProducerInfo).OwnerKey = publicKey1
	err, _ = tx.SpecialContextCheck()
	s.EqualError(err.(errors.ELAError).InnerError(), "can not register dposv2 before dposv2 start height")

	// Invalidates public key in payload
	txn.Payload().(*payload.ProducerInfo).OwnerKey = publicKey2
	txn.Payload().(*payload.ProducerInfo).NodePublicKey = publicKey2
	param.PublicDPOSHeight = 5
	s.Chain.Nodes = []*blockchain.BlockNode{
		{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {},
	}
	err, _ = tx.SpecialContextCheck()
	s.EqualError(err.(errors.ELAError).InnerError(), "invalid signature in payload")

	// Give a insufficient deposit coin
	txn.Payload().(*payload.ProducerInfo).OwnerKey = publicKey1
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
