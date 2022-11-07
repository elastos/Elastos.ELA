package transaction

import (
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
)

func (s *txValidatorTestSuite) TestCheckStakeTransaction() {
	publicKey := "03878cbe6abdafc702befd90e2329c4f37e7cb166410f0ecb70488c74c85b81d66"
	publicKeyBytes, _ := common.HexStringToBytes(publicKey)
	code, _ := getCode(publicKeyBytes)
	c, _ := contract.CreateStakeContractByCode(code)
	stakeAddressUint168 := c.ToProgramHash()
	rpPayload := &outputpayload.ExchangeVotesOutput{
		Version:      0,
		StakeAddress: *stakeAddressUint168,
	}
	txn := functions.CreateTransaction(
		0,
		common2.ExchangeVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     common.Uint256{},
				Value:       100000000,
				OutputLock:  0,
				ProgramHash: *stakeAddressUint168,
				Payload:     rpPayload,
			},
			{
				AssetID:     common.Uint256{},
				Value:       100000000,
				OutputLock:  0,
				ProgramHash: *stakeAddressUint168,
				Payload:     rpPayload,
			},
			{
				AssetID:     common.Uint256{},
				Value:       100000000,
				OutputLock:  0,
				ProgramHash: *stakeAddressUint168,
				Payload:     rpPayload,
			},
		},
		0,
		[]*program.Program{{
			Code:      nil,
			Parameter: nil,
		}},
	)
	err := txn.CheckTransactionOutput()
	s.EqualError(err, "output count should not be greater than 2")

	txn = functions.CreateTransaction(
		0,
		common2.ExchangeVotes,
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
	err = txn.CheckTransactionOutput()
	s.EqualError(err, "transaction has no outputs")

	txn = functions.CreateTransaction(
		0,
		common2.ExchangeVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     common.Uint256{},
				Value:       100000000,
				OutputLock:  0,
				ProgramHash: *stakeAddressUint168,
				Payload:     rpPayload,
			},
		},
		0,
		[]*program.Program{{
			Code:      nil,
			Parameter: nil,
		}},
	)
	err = txn.CheckTransactionOutput()
	s.EqualError(err, "asset ID in output is invalid")

	txn = functions.CreateTransaction(
		0,
		common2.ExchangeVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     core.ELAAssetID,
				Value:       -1,
				OutputLock:  0,
				ProgramHash: *stakeAddressUint168,
				Payload:     rpPayload,
			},
		},
		0,
		[]*program.Program{{
			Code:      nil,
			Parameter: nil,
		}},
	)
	err = txn.CheckTransactionOutput()
	s.EqualError(err, "invalid transaction UTXO output")

	txn = functions.CreateTransaction(
		0,
		common2.ExchangeVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     core.ELAAssetID,
				Value:       100000000,
				OutputLock:  0,
				ProgramHash: *stakeAddressUint168,
				Payload:     rpPayload,
			},
		},
		0,
		[]*program.Program{{
			Code:      code,
			Parameter: nil,
		}},
	)
	err = txn.CheckTransactionOutput()
	s.EqualError(err, "invalid output type")

	rpPayload.Version = 1
	txn = functions.CreateTransaction(
		0,
		common2.ExchangeVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     core.ELAAssetID,
				Value:       100000000,
				OutputLock:  0,
				ProgramHash: *stakeAddressUint168,
				Payload:     rpPayload,
				Type:        common2.OTStake,
			},
		},
		0,
		[]*program.Program{{
			Code:      code,
			Parameter: nil,
		}},
	)
	err = txn.CheckTransactionOutput()
	s.EqualError(err, "invalid exchange vote version")

	rpPayload.Version = 0
	txn = functions.CreateTransaction(
		0,
		common2.ExchangeVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     core.ELAAssetID,
				Value:       100000000,
				OutputLock:  0,
				ProgramHash: *stakeAddressUint168,
				Payload:     rpPayload,
				Type:        common2.OTStake,
			},
		},
		0,
		[]*program.Program{{
			Code:      code,
			Parameter: nil,
		}},
	)
	param := s.Chain.GetParams()
	param.StakePoolUint168 = config.StakePoolAddressUint168
	tx := txn.(*ExchangeVotesTransaction)
	tx.DefaultChecker.SetParameters(&TransactionParameters{
		BlockChain: s.Chain,
		Config:     s.Chain.GetParams(),
	})
	err = txn.CheckTransactionOutput()
	s.EqualError(err, "first output address need to be stake address")

	txn = functions.CreateTransaction(
		0,
		common2.ExchangeVotes,
		1,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     core.ELAAssetID,
				Value:       100000000,
				OutputLock:  0,
				ProgramHash: *stakeAddressUint168,
				Payload:     rpPayload,
				Type:        common2.OTStake,
			},
		},
		0,
		[]*program.Program{{
			Code:      code,
			Parameter: nil,
		}},
	)
	param = s.Chain.GetParams()
	param.StakePoolUint168 = stakeAddressUint168
	tx = txn.(*ExchangeVotesTransaction)
	tx.DefaultChecker.SetParameters(&TransactionParameters{
		BlockChain: s.Chain,
		Config:     s.Chain.GetParams(),
	})
	err = txn.CheckTransactionOutput()
	s.NoError(err)
}

func (s *txValidatorTestSuite) TestCheckStakeTransaction2() {
	s.CurrentHeight = 1
	_, pk, _ := crypto.GenerateKeyPair()
	//publicKey, _ := pk.EncodePoint(true)
	cont, _ := contract.CreateStandardContract(pk)
	code := cont.Code
	ct, _ := contract.CreateStakeContractByCode(code)
	stakeAddress := ct.ToProgramHash()
	ps := &payload.ExchangeVotes{}
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
			AssetID:     core.ELAAssetID,
			Value:       2000,
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
			AssetID:     core.ELAAssetID,
			ProgramHash: *stakeAddress,
			Type:        common2.OTStake,
			Value:       common.Fixed64(1000 * 1e8),
			Payload: &outputpayload.ExchangeVotesOutput{
				StakeAddress: *stakeAddress,
			},
		}, {
			AssetID:     core.ELAAssetID,
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
		0,
		common2.ExchangeVotes,
		1,
		ps,
		attribute,
		[]*common2.Input{input},
		outputs,
		0,
		programs,
	)

	bc := s.Chain
	config := bc.GetParams()
	config.StakePoolUint168 = stakeAddress
	tx := txn.(*ExchangeVotesTransaction)
	tx.DefaultChecker.SetParameters(&TransactionParameters{
		BlockChain: bc,
		Config:     config,
	})

	err := txn.CheckTransactionOutput()
	s.NoError(err)

}
