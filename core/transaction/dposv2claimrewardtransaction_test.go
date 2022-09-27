// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/errors"
)

func (s *txValidatorTestSuite) TestCreateClaimDposV2Transaction() {
	publicKeyStr1 := "02ca89a5fe6213da1b51046733529a84f0265abac59005f6c16f62330d20f02aeb"
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	pk, _ := crypto.DecodePoint(publicKey1)

	privateKeyStr1 := "7a50d2b036d64fcb3d344cee429f61c4a3285a934c45582b26e8c9227bc1f33a"
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)

	redeemScript, _ := contract.CreateStandardRedeemScript(pk)

	buf := new(bytes.Buffer)
	apPayload := &payload.DPoSV2ClaimReward{
		Value: common.Fixed64(100000000),
		Code:  redeemScript,
	}

	apPayload.SerializeUnsigned(buf, payload.ActivateProducerVersion)
	signature, _ := crypto.Sign(privateKey1, buf.Bytes())
	apPayload.Signature = signature

	// create program
	var txProgram = &program.Program{
		Code:      redeemScript,
		Parameter: nil,
	}
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.DposV2ClaimReward,
		0,
		apPayload,
		nil,
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{txProgram})
	tx := txn.(*DPoSV2ClaimRewardTransaction)
	tx.DefaultChecker.SetParameters(&TransactionParameters{
		BlockChain: s.Chain,
		Config:     s.Chain.GetParams(),
	})

	err, _ := tx.SpecialContextCheck()
	s.EqualError(err.(errors.ELAError).InnerError(), "can not claim reward before dposv2startheight")

	param := s.Chain.GetParams()
	param.DPoSV2StartHeight = 10
	tx.DefaultChecker.SetParameters(&TransactionParameters{
		BlockChain:  s.Chain,
		Config:      param,
		BlockHeight: 100,
	})
	err, _ = tx.SpecialContextCheck()
	s.EqualError(err.(errors.ELAError).InnerError(), "no reward to claim for such address")

	bc := s.Chain
	bc.GetState().DposV2RewardInfo["ERyUmNH51roR9qfru37Kqkaok2NghR7L5U"] = 100
	tx.DefaultChecker.SetParameters(&TransactionParameters{
		BlockChain:  bc,
		Config:      param,
		BlockHeight: 100,
	})

	err, _ = tx.SpecialContextCheck()
	s.EqualError(err.(errors.ELAError).InnerError(), "claim reward exceeded , max claim reward 0.00000100")

	bc = s.Chain
	bc.GetState().DposV2RewardInfo["ERyUmNH51roR9qfru37Kqkaok2NghR7L5U"] = 10000000000
	tx.DefaultChecker.SetParameters(&TransactionParameters{
		BlockChain:  bc,
		Config:      param,
		BlockHeight: 100,
	})
	err, _ = tx.SpecialContextCheck()
	s.NoError(err)
}
