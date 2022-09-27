package transaction

import (
	"bytes"
	"encoding/hex"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
)

func (s *txValidatorTestSuite) TestCheckUnstakeTransaction() {
	s.CurrentHeight = 1
	_, pk, _ := crypto.GenerateKeyPair()
	//publicKey, _ := pk.EncodePoint(true)
	cont, _ := contract.CreateStandardContract(pk)
	code := cont.Code
	ct, _ := contract.CreateStakeContractByCode(code)
	stakeAddress := ct.ToProgramHash()
	pl := &payload.ReturnVotes{
		Value: 100,
	}
	attribute := []*common2.Attribute{}

	tx1 := functions.CreateTransaction(
		0,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	tx1.SetOutputs([]*common2.Output{
		&common2.Output{
			AssetID:     config.ELAAssetID,
			Value:       1000,
			ProgramHash: blockchain.FoundationAddress,
		},
	})
	input := &common2.Input{
		Previous: common2.OutPoint{
			TxID:  tx1.Hash(),
			Index: 0,
		},
		Sequence: 0,
	}
	outputs := []*common2.Output{
		{
			AssetID:     config.ELAAssetID,
			ProgramHash: *cont.ToProgramHash(),
			Type:        common2.OTNone,
			Value:       common.Fixed64(1000 * 1e8),
		},
	}
	programs := []*program.Program{{
		Code:      code,
		Parameter: nil,
	}}
	txn := functions.CreateTransaction(
		9,
		common2.ReturnVotes,
		0,
		pl,
		attribute,
		[]*common2.Input{input},
		outputs,
		0,
		programs,
	)

	bc := s.Chain
	config := bc.GetParams()
	config.StakePool = *stakeAddress
	tx := txn.(*ReturnVotesTransaction)
	tx.DefaultChecker.SetParameters(&TransactionParameters{
		BlockChain: bc,
		Config:     config,
	})

	err := txn.CheckTransactionPayload()
	s.NoError(err)

	// todo complete me
	//err2, _ := txn.SpecialContextCheck()
	//s.EqualError(err2, "transaction validate error: output invalid")

	err3 := txn.CheckTransactionPayload()
	s.NoError(err3)

}

func (s *txValidatorTestSuite) TestCheckUnstakeTransaction2() {
	private := "97751342c819562a8d65059d759494fc9b2b565232bef047d1eae93f7c97baed"
	publicKey := "0228329FD319A5444F2265D08482B8C09360AE59945C50FA5211548C0C11D31F08"
	publicKeyBytes, _ := common.HexStringToBytes(publicKey)
	code, _ := getCode(publicKeyBytes)
	c, _ := contract.CreateStakeContractByCode(code)
	stakeAddress_uint168 := c.ToProgramHash()
	//toAddr , _ := stakeAddress_uint168.ToAddress()
	txn := functions.CreateTransaction(
		0,
		common2.ReturnVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{{
			Code:      code,
			Parameter: nil,
		}},
	)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err := txn.CheckTransactionOutput()

	s.EqualError(err, "transaction has no outputs")

	txn = functions.CreateTransaction(
		0,
		common2.ReturnVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     common.Uint256{},
				Value:       100000000,
				OutputLock:  0,
				ProgramHash: *stakeAddress_uint168,
				Payload:     nil,
			},
		},
		0,
		[]*program.Program{{
			Code:      nil,
			Parameter: nil,
		}},
	)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = txn.CheckTransactionOutput()
	s.EqualError(err, "asset ID in output is invalid")

	txn = functions.CreateTransaction(
		0,
		common2.ReturnVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     config.ELAAssetID,
				Value:       -1,
				OutputLock:  0,
				ProgramHash: *stakeAddress_uint168,
				Payload:     nil,
			},
		},
		0,
		[]*program.Program{{
			Code:      nil,
			Parameter: nil,
		}},
	)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = txn.CheckTransactionOutput()
	s.EqualError(err, "invalid transaction UTXO output")

	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid payload")

	txn = functions.CreateTransaction(
		0,
		common2.ReturnVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     config.ELAAssetID,
				Value:       1,
				OutputLock:  0,
				ProgramHash: *stakeAddress_uint168,
				Payload:     nil,
			},
		},
		0,
		[]*program.Program{{
			Code:      nil,
			Parameter: nil,
		}},
	)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err = txn.CheckTransactionOutput()
	s.NoError(err)

	txn = functions.CreateTransaction(
		0,
		common2.ReturnVotes,
		1,
		&payload.ReturnVotes{
			Value: -1,
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     config.ELAAssetID,
				Value:       1,
				OutputLock:  0,
				ProgramHash: *stakeAddress_uint168,
			},
		},
		0,
		[]*program.Program{{
			Code:      nil,
			Parameter: nil,
		}},
	)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})

	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid return votes value")

	txn.SetPayload(&payload.ReturnVotes{
		Value: 10001,
	})
	txn.SetPayloadVersion(0x02)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid payload version")

	txn.SetPayloadVersion(0x00)
	txn.SetPayload(&payload.ReturnVotes{
		Value: 10001,
		Code:  code,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:vote rights not enough")

	s.Chain.GetState().DposV2VoteRights = map[common.Uint168]common.Fixed64{
		*stakeAddress_uint168: 10001,
	}
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})

	buf := new(bytes.Buffer)
	tmpPayload := payload.ReturnVotes{
		ToAddr: *stakeAddress_uint168,
		Value:  10001,
		Code:   code,
	}

	tmpPayload.SerializeUnsigned(buf, payload.ReturnVotesVersionV0)
	privBuf, _ := hex.DecodeString(private)
	signature, _ := crypto.Sign(privBuf, buf.Bytes())
	txn.SetPayload(&payload.ReturnVotes{
		ToAddr:    *stakeAddress_uint168,
		Value:     10001,
		Code:      code,
		Signature: signature,
	})
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)
}
