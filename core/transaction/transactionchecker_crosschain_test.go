// Copyright (c) 2026 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

func TestCheckTransactionCrossChainUTXO(t *testing.T) {
	const activationHeight uint32 = 100

	testCases := []struct {
		name           string
		txType         common2.TxType
		payloadVersion byte
		references     map[*common2.Input]common2.Output
		blockHeight    uint32
		wantError      string
	}{
		{
			name:        "allows CrossChain UTXO before activation",
			txType:      common2.TransferAsset,
			references:  crossChainUTXOReferences(contract.PrefixCrossChain),
			blockHeight: activationHeight - 1,
		},
		{
			name:        "rejects TransferAsset after activation",
			txType:      common2.TransferAsset,
			references:  crossChainUTXOReferences(contract.PrefixCrossChain),
			blockHeight: activationHeight,
			wantError:   "only WithdrawFromSideChain and ReturnSideChainDepositCoin can spend CrossChain UTXOs",
		},
		{
			name:        "allows legacy WithdrawFromSideChain after activation",
			txType:      common2.WithdrawFromSideChain,
			references:  crossChainUTXOReferences(contract.PrefixCrossChain),
			blockHeight: activationHeight,
		},
		{
			name:           "allows V1 WithdrawFromSideChain after activation",
			txType:         common2.WithdrawFromSideChain,
			payloadVersion: payload.WithdrawFromSideChainVersionV1,
			references:     crossChainUTXOReferences(contract.PrefixCrossChain),
			blockHeight:    activationHeight,
		},
		{
			name:           "allows V2 WithdrawFromSideChain after activation",
			txType:         common2.WithdrawFromSideChain,
			payloadVersion: payload.WithdrawFromSideChainVersionV2,
			references:     crossChainUTXOReferences(contract.PrefixCrossChain),
			blockHeight:    activationHeight,
		},
		{
			name:           "allows unknown WithdrawFromSideChain version before activation",
			txType:         common2.WithdrawFromSideChain,
			payloadVersion: 0xff,
			references:     crossChainUTXOReferences(contract.PrefixCrossChain),
			blockHeight:    activationHeight - 1,
		},
		{
			name:           "rejects unknown WithdrawFromSideChain version after activation",
			txType:         common2.WithdrawFromSideChain,
			payloadVersion: 0xff,
			references:     crossChainUTXOReferences(contract.PrefixCrossChain),
			blockHeight:    activationHeight,
			wantError:      "unsupported WithdrawFromSideChain payload version cannot spend CrossChain UTXOs",
		},
		{
			name:        "allows legacy ReturnSideChainDepositCoin after activation",
			txType:      common2.ReturnSideChainDepositCoin,
			references:  crossChainUTXOReferences(contract.PrefixCrossChain),
			blockHeight: activationHeight,
		},
		{
			name:           "rejects nonlegacy ReturnSideChainDepositCoin after activation",
			txType:         common2.ReturnSideChainDepositCoin,
			payloadVersion: 1,
			references:     crossChainUTXOReferences(contract.PrefixCrossChain),
			blockHeight:    activationHeight,
			wantError:      "only legacy ReturnSideChainDepositCoin can spend CrossChain UTXOs",
		},
		{
			name:        "rejects mixed inputs in ReturnSideChainDepositCoin",
			txType:      common2.ReturnSideChainDepositCoin,
			references:  crossChainUTXOReferences(contract.PrefixCrossChain, contract.PrefixStandard),
			blockHeight: activationHeight,
			wantError:   "ReturnSideChainDepositCoin can only spend CrossChain UTXOs",
		},
		{
			name:        "allows nonCrossChain UTXO after activation",
			txType:      common2.TransferAsset,
			references:  crossChainUTXOReferences(contract.PrefixStandard),
			blockHeight: activationHeight,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			transaction := newCrossChainUTXOTestTransaction(t, testCase.txType,
				testCase.payloadVersion)
			err := checkTransactionCrossChainUTXO(transaction, testCase.references,
				testCase.blockHeight, activationHeight)
			if testCase.wantError == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err == nil || err.Error() != testCase.wantError {
				t.Fatalf("error = %v, want %q", err, testCase.wantError)
			}
		})
	}
}

func (s *txValidatorTestSuite) TestContextCheckRejectsCrossChainTransfer() {
	chainParams := *s.Chain.GetParams()
	chainParams.CrossChainUTXORestrictionHeight = 0

	input := s.insertCrossChainUTXOReference()

	transaction := CreateTransaction(
		common2.TxVersionDefault,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		nil,
		[]*common2.Input{input},
		[]*common2.Output{{
			AssetID:     core.ELAAssetID,
			Value:       1,
			ProgramHash: *randomUint168(),
		}},
		0,
		nil,
	)

	parameters := &TransactionParameters{
		Transaction: transaction,
		BlockHeight: 0,
		Config:      &chainParams,
		BlockChain:  s.Chain,
	}
	transaction.SetParameters(parameters)

	_, err := new(DefaultChecker).ContextCheck(parameters)
	s.Require().NotNil(err)
	s.Equal(elaerr.ErrTxInvalidInput, err.Code())
	s.Equal("only WithdrawFromSideChain and ReturnSideChainDepositCoin can spend CrossChain UTXOs",
		err.InnerError().Error())
}

func (s *txValidatorTestSuite) TestContextCheckRejectsUnknownCrossChainWithdrawVersion() {
	chainParams := *s.Chain.GetParams()
	chainParams.CrossChainUTXORestrictionHeight = 0

	input := s.insertCrossChainUTXOReference()

	transaction := CreateTransaction(
		common2.TxVersionDefault,
		common2.WithdrawFromSideChain,
		0xff,
		&payload.WithdrawFromSideChain{},
		nil,
		[]*common2.Input{input},
		[]*common2.Output{{
			AssetID:     core.ELAAssetID,
			Value:       1,
			ProgramHash: *randomUint168(),
		}},
		0,
		nil,
	)

	parameters := &TransactionParameters{
		Transaction: transaction,
		BlockHeight: 0,
		Config:      &chainParams,
		BlockChain:  s.Chain,
	}
	transaction.SetParameters(parameters)

	_, err := new(DefaultChecker).ContextCheck(parameters)
	s.Require().NotNil(err)
	s.Equal(elaerr.ErrTxInvalidInput, err.Code())
	s.Equal("unsupported WithdrawFromSideChain payload version cannot spend CrossChain UTXOs",
		err.InnerError().Error())
}

func (s *txValidatorTestSuite) insertCrossChainUTXOReference() *common2.Input {
	input := &common2.Input{
		Previous: common2.OutPoint{TxID: *randomUint256()},
	}
	crossChainProgramHash := common.Uint168{}
	crossChainProgramHash[0] = byte(contract.PrefixCrossChain)
	s.Chain.UTXOCache.InsertReference(input, &common2.Output{
		AssetID:     core.ELAAssetID,
		Value:       1,
		ProgramHash: crossChainProgramHash,
	})

	return input
}

func newCrossChainUTXOTestTransaction(t *testing.T, txType common2.TxType,
	payloadVersion byte) interfaces.Transaction {
	t.Helper()

	transaction, err := GetTransaction(txType)
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	transaction.SetTxType(txType)
	transaction.SetPayloadVersion(payloadVersion)

	return transaction
}

func crossChainUTXOReferences(prefixes ...contract.PrefixType) map[*common2.Input]common2.Output {
	references := make(map[*common2.Input]common2.Output, len(prefixes))
	for _, prefix := range prefixes {
		programHash := common.Uint168{}
		programHash[0] = byte(prefix)
		references[&common2.Input{}] = common2.Output{ProgramHash: programHash}
	}

	return references
}
