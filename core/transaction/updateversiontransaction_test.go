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
)

func (s *txValidatorTestSuite) TestUpdateVersionTransaction() {

	publicKeyStr1 := "031e12374bae471aa09ad479f66c2306f4bcc4ca5b754609a82a1839b94b4721b9"
	//publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	//publicKeyStr2 := "027c4f35081821da858f5c7197bac5e33e77e5af4a3551285f8a8da0a59bd37c45"
	//publicKey2, _ := common.HexStringToBytes(publicKeyStr2)
	//publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"

	updateVersion := &payload.UpdateVersion{

	}

	programs := []*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr1),
		Parameter: nil,
	}}

	txn := functions.CreateTransaction(
		0,
		common2.UpdateVersion,
		0,
		updateVersion,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		programs,
	)

	txn = CreateTransactionByType(txn, s.Chain)
	err, _ := txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:invalid update version height")

	txn.SetPayload(&payload.UpdateVersion{
		StartHeight: 100,
		EndHeight: 200,
	})

	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:invalid multi sign script code")

	publicKeyStrs := []string{
		"02089d7e878171240ce0e3633d3ddc8b1128bc221f6b5f0d1551caa717c7493062", "027d816821705e425415eb64a9704f25b4cd7eaca79616b0881fc92ac44ff8a46b", "02871b650700137defc5d34a11e56a4187f43e74bb078e147dd4048b8f3c81209f","02fc66cba365f9957bcb2030e89a57fb3019c57ea057978756c1d46d40dfdd4df0",
		"03e3fe6124a4ea269224f5f43552250d627b4133cfd49d1f9e0283d0cd2fd209bc", "02b95b000f087a97e988c24331bf6769b4a75e4b7d5d2a38105092a3aa841be33b", "0268214956b8421c0621d62cf2f0b20a02c2dc8c2cc89528aff9bd43b45ed34b9f", "03cce325c55057d2c8e3fb03fb5871794e73b85821e8d0f96a7e4510b4a922fad5",
		"02661637ae97c3af0580e1954ee80a7323973b256ca862cfcf01b4a18432670db4", "02d4a8f5016ae22b1acdf8a2d72f6eb712932213804efd2ce30ca8d0b9b4295ac5", "029a4d8e4c99a1199f67a25d79724e14f8e6992a0c8b8acf102682bd8f500ce0c1", "02a0aa9eac0e168f3474c2a0d04e50130833905740a5270e8a44d6c6e85cf6d98c",
	}
	var publicKeys []*crypto.PublicKey
	for _, publicKeyStr := range publicKeyStrs {
		publicKeyBytes, _ := hex.DecodeString(publicKeyStr)
		publicKey, _ := crypto.DecodePoint(publicKeyBytes)
		publicKeys = append(publicKeys, publicKey)
	}

	multiCode, _ := contract.CreateRevertToPOWRedeemScript(9, publicKeys)

	txn.SetPrograms([]*program.Program{{
		Code:      multiCode,
		Parameter: nil,
	}})
	txn = CreateTransactionByType(txn, s.Chain)
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

}