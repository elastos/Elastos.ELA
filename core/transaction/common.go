// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"io"
)

func GetTransactionByBytes(r io.Reader) (interfaces.Transaction, error) {
	flagByte, err := common.ReadBytes(r, 1)
	if err != nil {
		return nil, err
	}

	var version common2.TransactionVersion
	var txType common2.TxType
	if common2.TransactionVersion(flagByte[0]) >= common2.TxVersion09 {
		version = common2.TransactionVersion(flagByte[0])
		txTypeBytes, err := common.ReadBytes(r, 1)
		if err != nil {
			return nil, err
		}
		txType = common2.TxType(txTypeBytes[0])
	} else {
		version = common2.TxVersionDefault
		txType = common2.TxType(flagByte[0])
	}

	tx, err := GetTransaction(txType)
	if err != nil {
		return nil, err
	}
	tx.SetVersion(version)
	tx.SetTxType(txType)

	return tx, nil
}

func CreateTransaction(
	version common2.TransactionVersion,
	txType common2.TxType,
	payloadVersion byte,
	payload interfaces.Payload,
	attributes []*common2.Attribute,
	inputs []*common2.Input,
	outputs []*common2.Output,
	lockTime uint32,
	programs []*pg.Program,
) interfaces.Transaction {
	txn, err := functions.GetTransactionByTxType(txType)
	if err != nil {
		fmt.Println(err)
	}
	txn.SetVersion(version)
	txn.SetTxType(txType)
	txn.SetPayloadVersion(payloadVersion)
	txn.SetPayload(payload)
	txn.SetAttributes(attributes)
	txn.SetInputs(inputs)
	txn.SetOutputs(outputs)
	txn.SetLockTime(lockTime)
	txn.SetPrograms(programs)
	return txn
}

func GetTransactionparameters(
	transaction interfaces.Transaction,
	blockHeight uint32,
	timeStamp uint32,
	cfg interface{},
	bc interface{},
	proposalsUsedAmount common.Fixed64) interfaces.Parameters {
	return &TransactionParameters{
		Transaction:         transaction,
		BlockHeight:         blockHeight,
		TimeStamp:           timeStamp,
		Config:              cfg.(*config.Params),
		BlockChain:          bc.(*blockchain.BlockChain),
		ProposalsUsedAmount: proposalsUsedAmount,
	}
}

func GetTransaction(txType common2.TxType) (txn interfaces.Transaction, err error) {
	switch txType {
	case common2.CoinBase:
		txn = new(CoinBaseTransaction)

	case common2.RegisterAsset:
		txn = new(RegisterAssetTransaction)

	case common2.TransferAsset:
		txn = new(TransferAssetTransaction)

	case common2.IllegalProposalEvidence:
		txn = new(IllegalProposalTransaction)

	case common2.IllegalVoteEvidence:
		txn = new(IllegalVoteTransaction)

	case common2.IllegalBlockEvidence:
		txn = new(IllegalBlockTransaction)

	case common2.IllegalSidechainEvidence:
		txn = new(IllegalSideChainTransaction)

	case common2.InactiveArbitrators:
		txn = new(InactiveArbitratorsTransaction)

	case common2.RevertToDPOS:
		txn = new(RevertToDPOSTransaction)

	case common2.UpdateVersion:
		txn = new(UpdateVersionTransaction)

	case common2.SideChainPow:
		txn = new(SideChainPOWTransaction)

	case common2.RegisterProducer:
		txn = new(RegisterProducerTransaction)

	case common2.UpdateProducer:
		txn = new(UpdateProducerTransaction)

	case common2.CancelProducer:
		txn = new(CancelProducerTransaction)

	case common2.ActivateProducer:
		txn = new(ActivateProducerTransaction)

	case common2.RegisterCR:
		txn = new(RegisterCRTransaction)

	case common2.UpdateCR:
		txn = new(UpdateCRTransaction)

	case common2.UnregisterCR:
		txn = new(UnregisterCRTransaction)

	case common2.NextTurnDPOSInfo:
		txn = new(NextTurnDPOSInfoTransaction)

	case common2.ProposalResult:
		txn = new(CRCProposalResultTransaction)

	case common2.CRCProposal:
		txn = new(CRCProposalTransaction)

	case common2.CRCProposalReview:
		txn = new(CRCProposalReviewTransaction)

	case common2.CRCProposalTracking:
		txn = new(CRCProposalTrackingTransaction)

	case common2.CRCProposalWithdraw:
		txn = new(CRCProposalWithdrawTransaction)

	case common2.WithdrawFromSideChain:
		txn = new(WithdrawFromSideChainTransaction)

	case common2.TransferCrossChainAsset:
		txn = new(TransferCrossChainAssetTransaction)

	case common2.ReturnDepositCoin:
		txn = new(ReturnDepositCoinTransaction)

	case common2.ReturnCRDepositCoin:
		txn = new(ReturnCRDepositCoinTransaction)

	case common2.CRCAppropriation:
		txn = new(CRCAppropriationTransaction)

	case common2.CRCProposalRealWithdraw:
		txn = new(CRCProposalRealWithdrawTransaction)

	case common2.CRAssetsRectify:
		txn = new(CRAssetsRectifyTransaction)

	case common2.CRCouncilMemberClaimNode:
		txn = new(CRCouncilMemberClaimNodeTransaction)

	case common2.RevertToPOW:
		txn = new(RevertToPOWTransaction)

	case common2.ReturnSideChainDepositCoin:
		txn = new(ReturnSideChainDepositCoinTransaction)

	case common2.Record:
		txn = new(RecordTransaction)

	default:
		return nil, errors.New("invalid transaction type")
	}

	return txn, nil
}
