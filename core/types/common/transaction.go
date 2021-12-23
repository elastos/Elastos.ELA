// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package common

type TransactionVersion byte

const (
	TxVersionDefault TransactionVersion = 0x00
	TxVersion09      TransactionVersion = 0x09
)

// TxType represents different transaction types with different payload format.
// The TxType range is 0x00 - 0x08. When it is greater than 0x08 it will be
// interpreted as a TransactionVersion.
type TxType byte

const (
	CoinBase                TxType = 0x00
	RegisterAsset           TxType = 0x01
	TransferAsset           TxType = 0x02
	Record                  TxType = 0x03
	Deploy                  TxType = 0x04
	SideChainPow            TxType = 0x05
	RechargeToSideChain     TxType = 0x06
	WithdrawFromSideChain   TxType = 0x07
	TransferCrossChainAsset TxType = 0x08

	RegisterProducer  TxType = 0x09
	CancelProducer    TxType = 0x0a
	UpdateProducer    TxType = 0x0b
	ReturnDepositCoin TxType = 0x0c
	ActivateProducer  TxType = 0x0d

	IllegalProposalEvidence  TxType = 0x0e
	IllegalVoteEvidence      TxType = 0x0f
	IllegalBlockEvidence     TxType = 0x10
	IllegalSidechainEvidence TxType = 0x11
	InactiveArbitrators      TxType = 0x12
	UpdateVersion            TxType = 0x13
	NextTurnDPOSInfo         TxType = 0x14
	ProposalResult           TxType = 0x15

	RegisterCR          TxType = 0x21
	UnregisterCR        TxType = 0x22
	UpdateCR            TxType = 0x23
	ReturnCRDepositCoin TxType = 0x24

	CRCProposal              TxType = 0x25
	CRCProposalReview        TxType = 0x26
	CRCProposalTracking      TxType = 0x27
	CRCAppropriation         TxType = 0x28
	CRCProposalWithdraw      TxType = 0x29
	CRCProposalRealWithdraw  TxType = 0x2a
	CRAssetsRectify          TxType = 0x2b
	CRCouncilMemberClaimNode TxType = 0x31

	RevertToPOW  TxType = 0x41
	RevertToDPOS TxType = 0x42

	ReturnSideChainDepositCoin TxType = 0x51

	// DPoS2.0
	DposV2ClaimReward             TxType = 0x60
	DposV2ClaimRewardRealWithdraw TxType = 0x61
	ExchangeVotes                 TxType = 0x62
	Voting                        TxType = 0x63
	CancelVotes                   TxType = 0x64
)

func (self TxType) Name() string {
	switch self {
	case CoinBase:
		return "CoinBase"
	case RegisterAsset:
		return "RegisterAsset"
	case TransferAsset:
		return "TransferAsset"
	case Record:
		return "Record"
	case Deploy:
		return "Deploy"
	case SideChainPow:
		return "SideChainPow"
	case RechargeToSideChain:
		return "RechargeToSideChain"
	case WithdrawFromSideChain:
		return "WithdrawFromSideChain"
	case TransferCrossChainAsset:
		return "TransferCrossChainAsset"
	case RegisterProducer:
		return "RegisterProducer"
	case CancelProducer:
		return "CancelProducer"
	case UpdateProducer:
		return "UpdateProducer"
	case ReturnDepositCoin:
		return "ReturnDepositCoin"
	case ActivateProducer:
		return "ActivateProducer"
	case IllegalProposalEvidence:
		return "IllegalProposalEvidence"
	case IllegalVoteEvidence:
		return "IllegalVoteEvidence"
	case IllegalBlockEvidence:
		return "IllegalBlockEvidence"
	case IllegalSidechainEvidence:
		return "IllegalSidechainEvidence"
	case InactiveArbitrators:
		return "InactiveArbitrators"
	case UpdateVersion:
		return "UpdateVersion"
	case RegisterCR:
		return "RegisterCR"
	case UnregisterCR:
		return "UnregisterCR"
	case UpdateCR:
		return "UpdateCR"
	case ReturnCRDepositCoin:
		return "ReturnCRDepositCoin"
	case CRCProposal:
		return "CRCProposal"
	case CRCProposalReview:
		return "CRCProposalReview"
	case CRCProposalWithdraw:
		return "CRCProposalWithdraw"
	case CRCProposalTracking:
		return "CRCProposalTracking"
	case CRCAppropriation:
		return "CRCAppropriation"
	case CRCProposalRealWithdraw:
		return "CRCProposalRealWithdraw"
	case CRAssetsRectify:
		return "CRAssetsRectify"
	case CRCouncilMemberClaimNode:
		return "CRCouncilMemberClaimNode"
	case NextTurnDPOSInfo:
		return "NextTurnDPOSInfo"
	case ProposalResult:
		return "ProposalResult"
	case RevertToPOW:
		return "RevertToPOW"
	case RevertToDPOS:
		return "RevertToDPOS"
	case ReturnSideChainDepositCoin:
		return "ReturnSideChainDepositCoin"
	case ExchangeVotes:
		return "ExchangeVotes"
	case Voting:
		return "Voting"
	case CancelVotes:
		return "CancelVotes"
	default:
		return "Unknown"
	}
}
