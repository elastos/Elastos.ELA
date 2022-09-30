// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"encoding/hex"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
)

func (s *txValidatorTestSuite) TestRevertToDposTransaction() {
	publicKeyStr1 := "031e12374bae471aa09ad479f66c2306f4bcc4ca5b754609a82a1839b94b4721b9"
	//publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	publicKeyStr2 := "027c4f35081821da858f5c7197bac5e33e77e5af4a3551285f8a8da0a59bd37c45"
	//publicKey2, _ := common.HexStringToBytes(publicKeyStr2)
	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"

	revertToDpos := &payload.RevertToDPOS{
		WorkHeightInterval:     100,
		RevertToPOWBlockHeight: 100,
	}

	programs := []*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr1),
		Parameter: nil,
	}}

	txn := functions.CreateTransaction(
		0,
		common2.RevertToDPOS,
		0,
		revertToDpos,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		programs,
	)

	txn = CreateTransactionByType(txn, s.Chain)
	err, _ := txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:invalid WorkHeightInterval")

	txn.SetPayload(&payload.RevertToDPOS{
		WorkHeightInterval:     10,
		RevertToPOWBlockHeight: 100,
	})

	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid GetConsensusAlgorithm() != state.POW")

	s.Chain.GetState().ConsensusAlgorithm = state.POW
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid multi sign script code")

	s.Chain.GetState().DPOSWorkHeight = 100
	s.Chain.BestChain.Height = 0
	txn = CreateTransactionByType(txn, s.Chain)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:already receieved  revertodpos")

	s.Chain.BestChain.Height = 1000
	txn = CreateTransactionByType(txn, s.Chain)

	{
		publicKeyStrs := []string{
			publicKeyStr1, publicKeyStr2, publicKeyStr3,
		}
		var publicKeys []*crypto.PublicKey
		for _, publicKeyStr := range publicKeyStrs {
			publicKeyBytes, _ := hex.DecodeString(publicKeyStr)
			publicKey, _ := crypto.DecodePoint(publicKeyBytes)
			publicKeys = append(publicKeys, publicKey)
		}

		multiCode, _ := contract.CreateRevertToPOWRedeemScript(2, publicKeys)

		txn.SetPrograms([]*program.Program{{
			Code:      multiCode,
			Parameter: nil,
		}})
		txn = CreateTransactionByType(txn, s.Chain)
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err, "transaction validate error: payload content invalid:invalid multi sign script code")
	}

	publicKeyStrs := []string{
		"0248df6705a909432be041e0baa25b8f648741018f70d1911f2ed28778db4b8fe4", "02771faf0f4d4235744b30972d5f2c470993920846c761e4d08889ecfdc061cddf", "0342196610e57d75ba3afa26e030092020aec56822104e465cba1d8f69f8d83c8e",
		"02fa3e0d14e0e93ca41c3c0f008679e417cf2adb6375dd4bbbee9ed8e8db606a56", "03ab3ecd1148b018d480224520917c6c3663a3631f198e3b25cf4c9c76786b7850",
	}
	var publicKeys []*crypto.PublicKey
	for _, publicKeyStr := range publicKeyStrs {
		publicKeyBytes, _ := hex.DecodeString(publicKeyStr)
		publicKey, _ := crypto.DecodePoint(publicKeyBytes)
		publicKeys = append(publicKeys, publicKey)
	}

	multiCode, _ := contract.CreateRevertToPOWRedeemScript(4, publicKeys)

	txn.SetPrograms([]*program.Program{{
		Code:      multiCode,
		Parameter: nil,
	}})
	txn = CreateTransactionByType(txn, s.Chain)
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

}
