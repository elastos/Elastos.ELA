// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package interfaces

import (
	"errors"

	common "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

func GetPayload(txType common.TxType, payloadVersion byte) (Payload, error) {
	// todo use payloadVersion

	var p Payload
	switch txType {
	case common.CoinBase:
		p = new(payload.CoinBase)
	case common.RegisterAsset:
		p = new(payload.RegisterAsset)
	case common.TransferAsset:
		p = new(payload.TransferAsset)
	case common.Record:
		p = new(payload.Record)
	case common.SideChainPow:
		p = new(payload.SideChainPow)
	case common.WithdrawFromSideChain:
		p = new(payload.WithdrawFromSideChain)
	case common.NFTDestroyFromSideChain:
		p = new(payload.NFTDestroyFromSideChain)
	case common.TransferCrossChainAsset:
		p = new(payload.TransferCrossChainAsset)
	case common.RegisterProducer:
		p = new(payload.ProducerInfo)
	case common.CancelProducer:
		p = new(payload.ProcessProducer)
	case common.UpdateProducer:
		p = new(payload.ProducerInfo)
	case common.ReturnDepositCoin:
		p = new(payload.ReturnDepositCoin)
	case common.ActivateProducer:
		p = new(payload.ActivateProducer)
	case common.IllegalProposalEvidence:
		p = new(payload.DPOSIllegalProposals)
	case common.IllegalVoteEvidence:
		p = new(payload.DPOSIllegalVotes)
	case common.IllegalBlockEvidence:
		p = new(payload.DPOSIllegalBlocks)
	case common.IllegalSidechainEvidence:
		p = new(payload.SidechainIllegalData)
	case common.InactiveArbitrators:
		p = new(payload.InactiveArbitrators)
	case common.RevertToDPOS:
		p = new(payload.RevertToDPOS)
	case common.UpdateVersion:
		p = new(payload.UpdateVersion)
	case common.RegisterCR:
		p = new(payload.CRInfo)
	case common.UpdateCR:
		p = new(payload.CRInfo)
	case common.UnregisterCR:
		p = new(payload.UnregisterCR)
	case common.ReturnCRDepositCoin:
		p = new(payload.ReturnDepositCoin)
	case common.CRCProposal:
		p = new(payload.CRCProposal)
	case common.CRCProposalReview:
		p = new(payload.CRCProposalReview)
	case common.CRCProposalWithdraw:
		p = new(payload.CRCProposalWithdraw)
	case common.CRCProposalTracking:
		p = new(payload.CRCProposalTracking)
	case common.CRCAppropriation:
		p = new(payload.CRCAppropriation)
	case common.CRAssetsRectify:
		p = new(payload.CRAssetsRectify)
	case common.CRCProposalRealWithdraw:
		p = new(payload.CRCProposalRealWithdraw)
	case common.CRCouncilMemberClaimNode:
		p = new(payload.CRCouncilMemberClaimNode)
	case common.NextTurnDPOSInfo:
		p = new(payload.NextTurnDPOSInfo)
	case common.RevertToPOW:
		p = new(payload.RevertToPOW)
	case common.ProposalResult:
		p = new(payload.RecordProposalResult)
	case common.ReturnSideChainDepositCoin:
		p = new(payload.ReturnSideChainDepositCoin)
	case common.ExchangeVotes:
		p = new(payload.ExchangeVotes)
	case common.Voting:
		p = new(payload.Voting)
	case common.ReturnVotes:
		p = new(payload.ReturnVotes)
	case common.VotesRealWithdraw:
		p = new(payload.VotesRealWithdrawPayload)
	case common.DposV2ClaimReward:
		p = new(payload.DPoSV2ClaimReward)
	case common.DposV2ClaimRewardRealWithdraw:
		p = new(payload.DposV2ClaimRewardRealWithdraw)
	default:
		return nil, errors.New("[BaseTransaction], invalid transaction type.")
	}
	return p, nil
}
