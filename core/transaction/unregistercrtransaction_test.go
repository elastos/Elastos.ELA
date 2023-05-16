// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

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
	"path/filepath"
)

func (s *txValidatorTestSuite) TestCheckUnregisterCRTransaction() {

	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"

	votingHeight := config.DefaultParams.CRConfiguration.CRVotingStartHeight
	nickName1 := "nickname 1"

	//register a cr to unregister
	registerCRTxn := s.getRegisterCRTx(publicKeyStr1, privateKeyStr1,
		nickName1, payload.CRInfoVersion, &common.Uint168{})
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
	block := &types.Block{
		Transactions: []interfaces.Transaction{
			registerCRTxn,
		},
		Header: common2.Header{Height: votingHeight},
	}
	s.Chain.GetCRCommittee().ProcessBlock(block, nil)
	//ok
	txn := s.getUnregisterCRTx(publicKeyStr1, privateKeyStr1)
	txn = CreateTransactionByType(txn, s.Chain)
	err := txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	//invalid payload need unregisterCR pass registerCr
	registerTx := s.getRegisterCRTx(publicKeyStr1, privateKeyStr1,
		nickName1, payload.CRInfoVersion, &common.Uint168{})
	registerTx = CreateTransactionByType(registerTx, s.Chain)
	err = registerTx.SetParameters(&TransactionParameters{
		Transaction: registerTx,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = registerTx.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:nick name nickname 1 already inuse")

	//not in vote Period lower
	err = txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: config.DefaultParams.CRConfiguration.CRVotingStartHeight - 1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:should create tx during voting period")

	//not in vote Period lower upper c.params.CRCommitteeStartHeight
	s.Chain.GetCRCommittee().InElectionPeriod = true
	config.DefaultParams.DPoSV2StartHeight = 2000000
	err = txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: config.DefaultParams.CRConfiguration.CRCommitteeStartHeight + 1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:should create tx during voting period")

	//unregister unknown CR
	txn2 := s.getUnregisterCRTx(publicKeyStr2, privateKeyStr2)
	txn2 = CreateTransactionByType(txn2, s.Chain)
	err = txn2.SetParameters(&TransactionParameters{
		Transaction: txn2,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn2.SpecialContextCheck()

	s.EqualError(err, "transaction validate error: payload content invalid:unregister unknown CR")

	//wrong signature
	txn.Payload().(*payload.UnregisterCR).Signature = randomSignature()
	err = txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:[Validation], Verify failed.")
}

func (s *txValidatorTestSuite) getUnregisterCRTx(publicKeyStr, privateKeyStr string) interfaces.Transaction {

	publicKeyStr1 := publicKeyStr
	privateKeyStr1 := privateKeyStr
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)

	code1 := getCodeByPubKeyStr(publicKeyStr1)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.UnregisterCR,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	unregisterCRPayload := &payload.UnregisterCR{
		CID: *getCID(code1),
	}
	signBuf := new(bytes.Buffer)
	err := unregisterCRPayload.SerializeUnsigned(signBuf, payload.UnregisterCRVersion)
	s.NoError(err)
	rcSig1, err := crypto.Sign(privateKey1, signBuf.Bytes())
	s.NoError(err)
	unregisterCRPayload.Signature = rcSig1
	txn.SetPayload(unregisterCRPayload)

	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr1),
		Parameter: nil,
	}})
	return txn
}
