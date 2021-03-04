// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package manager

import (
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/dpos/log"
	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
	"github.com/elastos/Elastos.ELA/p2p/msg"
)

type DPOSOnDutyHandler struct {
	*DPOSHandlerSwitch
}

func (h *DPOSOnDutyHandler) ProcessAcceptVote(id peer.PID, p *payload.DPOSProposalVote) (succeed bool, finished bool) {
	log.Info("[Onduty-ProcessAcceptVote] start")
	defer log.Info("[Onduty-ProcessAcceptVote] end")

	currentProposal := h.proposalDispatcher.GetProcessingProposal()
	if currentProposal != nil && currentProposal.Hash().IsEqual(p.ProposalHash) && h.consensus.IsRunning() {
		log.Info("[OnVoteReceived] Received needed sign, collect it")
		return h.proposalDispatcher.ProcessVote(p, true)
	}

	return false, false
}

func (h *DPOSOnDutyHandler) ProcessRejectVote(id peer.PID, p *payload.DPOSProposalVote) (succeed bool, finished bool) {
	log.Info("[Onduty-ProcessRejectVote] start")

	currentProposal := h.proposalDispatcher.GetProcessingProposal()
	if currentProposal != nil && currentProposal.Hash().IsEqual(p.ProposalHash) && h.consensus.IsRunning() {
		return h.proposalDispatcher.ProcessVote(p, false)
	}

	return false, false
}

func (h *DPOSOnDutyHandler) ProcessProposal(id peer.PID, p *payload.DPOSProposal) (handled bool) {
	return false
}

func (h *DPOSOnDutyHandler) ChangeView(firstBlockHash *common.Uint256) {

	if !h.tryCreateInactiveArbitratorsTx() {
		b, ok := h.cfg.Manager.GetBlockCache().TryGetValue(*firstBlockHash)
		if !ok {
			log.Info("[OnViewChanged] get block failed for proposal")
		} else {
			log.Info("[OnViewChanged] start proposal")
			h.proposalDispatcher.CleanProposals(true)
			h.proposalDispatcher.StartProposal(b)
		}
	}
}

func (h *DPOSOnDutyHandler) TryStartNewConsensus(b *types.Block) bool {
	result := false

	if h.consensus.IsReady() {
		log.Info("[OnDuty][OnBlockReceived] received first unsigned block, start consensus")
		h.consensus.StartConsensus(b)
		h.proposalDispatcher.StartProposal(b)
		result = true
	} else { //finished
		log.Info("[OnDuty][OnBlockReceived] received unsigned block, record block")
		h.consensus.ProcessBlock(b)
		result = false
	}

	return result
}

func (h *DPOSOnDutyHandler) getActiveArbitersCount() int {
	peers := h.cfg.Network.GetActivePeers()

	activeArbitersCount := 0
	for _, p := range peers {
		pid := p.PID()
		if h.cfg.Arbitrators.IsActiveProducer(pid[:]) && h.cfg.Arbitrators.IsArbitrator(pid[:]) {
			activeArbitersCount++
		}
	}
	return activeArbitersCount
}

func (h *DPOSOnDutyHandler) TryCreateRevertToDPOSTx(BlockHeight uint32) bool {
	// connect count is not enough
	// if i am not onduty return
	activeArbitersCount := float64(h.getActiveArbitersCount())
	needCount := float64(h.cfg.Arbitrators.GetArbitersCount())*float64(4)/float64(5) + 1
	log.Info("[TryCreateRevertToDPOSTx] current active arbiters count:", activeArbitersCount, "need:", needCount)
	if activeArbitersCount < needCount {
		return false
	}
	// if it is in not pow mod
	if !h.cfg.Arbitrators.IsInPOWMode() {
		log.Warn("[TryCreateRevertToDPOSTx] is not in POW mode")
		return false
	}
	// is it onduty
	curPublicKey := h.proposalDispatcher.cfg.Account.PublicKeyBytes()
	if h.consensus.IsArbitratorOnDuty(curPublicKey) {
		tx, err := h.proposalDispatcher.CreateRevertToDPOS(h.cfg.Arbitrators.GetRevertToPOWBlockHeight())
		if err != nil {
			log.Warn("[TryCreateRevertToDPOSTx] failed to create revert to DPoS transaction:", err)
			return false
		}
		h.cfg.Network.BroadcastMessage(&msg.Tx{Serializable: tx})
		return true
	}
	return false
}

func (h *DPOSOnDutyHandler) tryCreateInactiveArbitratorsTx() bool {
	if h.proposalDispatcher.IsViewChangedTimeOut() {
		if h.cfg.Manager.isCRCArbiter() {
			tx, err := h.proposalDispatcher.CreateInactiveArbitrators()
			if err != nil {
				log.Warn("[tryCreateInactiveArbitratorsTx] create tx error: ", err)
				return false
			}

			h.cfg.Network.BroadcastMessage(&msg.Tx{Serializable: tx})
		}
		return true
	}
	return false
}
