// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"fmt"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/utils"
)

type ProposalStatus uint8

const (
	// Registered is the status means the CRC proposal tx has been on the best
	//	chain.
	Registered ProposalStatus = 0x00

	// CRAgreed means CRC has agreed the proposal.
	CRAgreed ProposalStatus = 0x01

	// VoterAgreed means there are not enough negative vote about the proposal.
	VoterAgreed ProposalStatus = 0x02

	// Finished means the proposal has run out the lifetime.
	Finished ProposalStatus = 0x03

	// CRCanceled means the proposal canceled by CRC voting.
	CRCanceled ProposalStatus = 0x04

	// VoterCanceled means the proposal canceled by voters' reject voting.
	VoterCanceled ProposalStatus = 0x05

	// Terminated means proposal had been approved by both CR and voters,
	// whoever the proposal related project has been decided to terminate for
	// some reason.
	Terminated ProposalStatus = 0x06

	// Aborted means the proposal was cancelled because of a snap election.
	Aborted ProposalStatus = 0x07
)

func (status ProposalStatus) String() string {
	switch status {
	case Registered:
		return "Registered"
	case CRAgreed:
		return "CRAgreed"
	case VoterAgreed:
		return "VoterAgreed"
	case Finished:
		return "Finished"
	case CRCanceled:
		return "CRCanceled"
	case VoterCanceled:
		return "VoterCanceled"
	case Terminated:
		return "Terminated"
	case Aborted:
		return "Aborted"
	default:
		return fmt.Sprintf("Unknown ProposalStatus (%d)", status)
	}
}

// ProposalManager used to manage all proposals existing in block chain.
type ProposalManager struct {
	ProposalKeyFrame
	params  *config.Configuration
	history *utils.History
}

func (p *ProposalManager) tryCancelReservedCustomID(height uint32) {
	if len(p.ReservedCustomIDLists) == 0 {
		oriReservedCustomID := p.ReservedCustomID
		p.history.Append(height, func() {
			p.ReservedCustomID = false
		}, func() {
			p.ReservedCustomID = oriReservedCustomID
		})

	}
}

// only init use
func (p *ProposalManager) InitSecretaryGeneralPublicKey(publicKey string) {
	p.SecretaryGeneralPublicKey = publicKey
}

// existDraft judge if specified draft (that related to a proposal) exist.
func (p *ProposalManager) existDraft(hash common.Uint256) bool {
	for _, v := range p.Proposals {
		if v.Proposal.DraftHash.IsEqual(hash) {
			return true
		}
	}
	return false
}

// existProposal judge if specified proposal exist.
func (p *ProposalManager) existProposal(hash common.Uint256) bool {
	_, ok := p.Proposals[hash]
	return ok
}

func (p *ProposalManager) getAllProposals() (dst ProposalsMap) {
	dst = NewProposalMap()
	for k, v := range p.Proposals {
		p := *v
		dst[k] = &p
	}
	return
}

func (p *ProposalManager) getProposalByDraftHash(draftHash common.Uint256) *ProposalState {
	for _, v := range p.Proposals {
		if v.Proposal.DraftHash.IsEqual(draftHash) {
			return v
		}
	}
	return nil
}

func (p *ProposalManager) getProposals(status ProposalStatus) (dst ProposalsMap) {
	dst = NewProposalMap()
	for k, v := range p.Proposals {
		if v.Status == status {
			p := *v
			dst[k] = &p
		}
	}
	return
}

func (p *ProposalManager) getRegisteredSideChainByHeight(height uint32) map[common.Uint256]payload.SideChainInfo {
	return p.RegisteredSideChainPayloadInfo[height]
}

func (p *ProposalManager) getAllRegisteredSideChain() map[uint32]map[common.Uint256]payload.SideChainInfo {
	return p.RegisteredSideChainPayloadInfo
}

// getProposal will return a proposal with specified hash,
// and return nil if not found.
func (p *ProposalManager) getProposal(hash common.Uint256) *ProposalState {
	result, ok := p.Proposals[hash]
	if !ok {
		return nil
	}
	return result
}

func (p *ProposalManager) availableWithdrawalAmount(hash common.Uint256) common.Fixed64 {
	proposal := p.getProposal(hash)
	amount := common.Fixed64(0)
	if proposal == nil {
		return amount
	}
	for i, a := range proposal.WithdrawableBudgets {
		if _, ok := proposal.WithdrawnBudgets[i]; !ok {
			amount += a
		}
	}
	return amount
}

func getProposalTotalBudgetAmount(proposal payload.CRCProposalInfo) common.Fixed64 {
	var budget common.Fixed64
	for _, b := range proposal.Budgets {
		budget += b.Amount
	}
	return budget
}

func getProposalUnusedBudgetAmount(proposalState *ProposalState) common.Fixed64 {
	var budget common.Fixed64
	for _, b := range proposalState.Proposal.Budgets {
		if _, ok := proposalState.WithdrawableBudgets[b.Stage]; !ok {
			budget += b.Amount
		}
	}
	return budget
}

// updateProposals will update proposals' status.
func (p *ProposalManager) updateProposals(state *State, height uint32,
	circulation common.Fixed64, inElectionPeriod bool) (common.Fixed64, []payload.ProposalResult) {
	var unusedAmount common.Fixed64
	endProposals := make(map[string]struct{})
	results := make([]payload.ProposalResult, 0)
	for k, v := range p.Proposals {
		proposalType := v.Proposal.ProposalType
		switch v.Status {
		case Registered:
			if !inElectionPeriod {
				p.abortProposal(v, height)
				unusedAmount += getProposalTotalBudgetAmount(v.Proposal)
				recordPartProposalResult(&results, proposalType, k, false)
				if proposalType == payload.ReserveCustomID {
					p.tryCancelReservedCustomID(height)
				}
				break
			}

			if p.shouldEndCRCVote(v.RegisterHeight, height) {
				pass := true
				if p.transferRegisteredState(v, height) == CRCanceled {
					p.removeRegisterSideChainInfo(v, height)
					unusedAmount += getProposalTotalBudgetAmount(v.Proposal)
					pass = false
					recordPartProposalResult(&results, proposalType, k, pass)
					if proposalType == payload.ReserveCustomID {
						p.tryCancelReservedCustomID(height)
					}
				}
			}
		case CRAgreed:
			if !inElectionPeriod {
				p.abortProposal(v, height)
				unusedAmount += getProposalTotalBudgetAmount(v.Proposal)
				recordPartProposalResult(&results, proposalType, k, false)
				if proposalType == payload.ReserveCustomID {
					p.tryCancelReservedCustomID(height)
				}
				endProposals[v.Proposal.Hash.String()] = struct{}{}
				break
			}
			if p.shouldEndPublicVote(v.VoteStartHeight, height) {
				// record finished proposals
				endProposals[v.Proposal.Hash.String()] = struct{}{}

				if p.transferCRAgreedState(v, height, circulation) == VoterCanceled {
					p.removeRegisterSideChainInfo(v, height)
					unusedAmount += getProposalTotalBudgetAmount(v.Proposal)
					recordPartProposalResult(&results, proposalType, k, false)
					if proposalType == payload.ReserveCustomID {
						p.tryCancelReservedCustomID(height)
					}
					continue
				}
				p.dealProposal(v, &unusedAmount, height)
				recordPartProposalResult(&results, proposalType, k, true)
			}
		}
	}

	newUsedCRCProposalVotes := map[common.Uint168][]payload.VotesWithLockTime{}
	for k, v := range state.UsedCRCProposalVotes {
		stakeAddress := k
		for _, voteInfo := range v {
			if _, ok := endProposals[common.BytesToHexString(voteInfo.Candidate)]; ok {
				continue
			}
			if _, ok := newUsedCRCProposalVotes[stakeAddress]; !ok {
				newUsedCRCProposalVotes[stakeAddress] = make([]payload.VotesWithLockTime, 0)
			}
			newUsedCRCProposalVotes[stakeAddress] = append(newUsedCRCProposalVotes[stakeAddress], voteInfo)
		}
	}

	oriUsedCRCProposalVotes := copyProgramHashVotesInfoSet(state.UsedCRCProposalVotes)
	p.history.Append(height, func() {
		state.UsedCRCProposalVotes = newUsedCRCProposalVotes
	}, func() {
		state.UsedCRCProposalVotes = oriUsedCRCProposalVotes
	})

	return unusedAmount, results
}

func recordPartProposalResult(results *[]payload.ProposalResult,
	proposalType payload.CRCProposalType, proposalHash common.Uint256, result bool) {
	var needRecordResult bool
	switch proposalType {
	case payload.ReserveCustomID, payload.ReceiveCustomID, payload.ChangeCustomIDFee:
		needRecordResult = true

	default:
		if proposalType > payload.MinUpgradeProposalType && proposalType <= payload.MaxUpgradeProposalType {
			needRecordResult = true
		}
	}

	if needRecordResult {
		*results = append(*results, payload.ProposalResult{
			ProposalHash: proposalHash,
			ProposalType: proposalType,
			Result:       result,
		})
	}
}

// abortProposal will transfer the status to aborted.
func (p *ProposalManager) abortProposal(proposalState *ProposalState,
	height uint32) {
	oriStatus := proposalState.Status
	oriBudgetsStatus := make(map[uint8]BudgetStatus)
	for k, v := range proposalState.BudgetsStatus {
		oriBudgetsStatus[k] = v
	}
	p.removeRegisterSideChainInfo(proposalState, height)
	p.history.Append(height, func() {
		proposalState.Status = Aborted
		for k, _ := range proposalState.BudgetsStatus {
			proposalState.BudgetsStatus[k] = Closed
		}
	}, func() {
		proposalState.Status = oriStatus
		proposalState.BudgetsStatus = oriBudgetsStatus
	})
}

// abortProposal will transfer the status to aborted.
func (p *ProposalManager) terminatedProposal(proposalState *ProposalState,
	height uint32) {
	oriStatus := proposalState.Status
	oriBudgetsStatus := make(map[uint8]BudgetStatus)
	for k, v := range proposalState.BudgetsStatus {
		oriBudgetsStatus[k] = v
	}
	p.history.Append(height, func() {
		proposalState.TerminatedHeight = height
		proposalState.Status = Terminated
		for k, v := range proposalState.BudgetsStatus {
			if v == Unfinished || v == Rejected {
				proposalState.BudgetsStatus[k] = Closed
			}
		}
	}, func() {
		proposalState.TerminatedHeight = 0
		proposalState.Status = oriStatus
		proposalState.BudgetsStatus = oriBudgetsStatus
	})
}

// transferRegisteredState will transfer the Registered State by CR agreement
// count.
func (p *ProposalManager) transferRegisteredState(proposalState *ProposalState,
	height uint32) (status ProposalStatus) {
	agreedCount := uint32(0)
	for _, v := range proposalState.CRVotes {
		if v == payload.Approve {
			agreedCount++
		}
	}

	oriVoteStartHeight := proposalState.VoteStartHeight
	if agreedCount >= p.params.CRConfiguration.CRAgreementCount {
		status = CRAgreed
		p.history.Append(height, func() {
			proposalState.Status = CRAgreed
			proposalState.VoteStartHeight = height
		}, func() {
			proposalState.Status = Registered
			proposalState.VoteStartHeight = oriVoteStartHeight
		})
	} else {
		status = CRCanceled
		oriBudgetsStatus := make(map[uint8]BudgetStatus)
		for k, v := range proposalState.BudgetsStatus {
			oriBudgetsStatus[k] = v
		}

		p.history.Append(height, func() {
			proposalState.Status = CRCanceled
			for k, _ := range proposalState.BudgetsStatus {
				proposalState.BudgetsStatus[k] = Closed
			}
		}, func() {
			proposalState.Status = Registered
			proposalState.BudgetsStatus = oriBudgetsStatus
		})
		if proposalState.Proposal.ProposalType == payload.ReceiveCustomID {
			oriPendingReceivedCustomIDMap := p.PendingReceivedCustomIDMap
			p.history.Append(height, func() {
				for _, id := range proposalState.Proposal.ReceivedCustomIDList {
					delete(p.PendingReceivedCustomIDMap, id)
				}
			}, func() {
				p.PendingReceivedCustomIDMap = oriPendingReceivedCustomIDMap
			})
		}
	}
	return
}

func (p *ProposalManager) dealProposal(proposalState *ProposalState, unusedAmount *common.Fixed64, height uint32) {
	switch proposalState.Proposal.ProposalType {
	case payload.ChangeProposalOwner:
		proposal := p.getProposal(proposalState.Proposal.TargetProposalHash)
		originRecipient := proposal.Recipient
		oriProposalOwner := proposalState.ProposalOwner
		emptyUint168 := common.Uint168{}
		p.history.Append(height, func() {
			proposal.ProposalOwner = proposalState.Proposal.NewOwnerPublicKey
			if proposalState.Proposal.NewRecipient != emptyUint168 {
				proposal.Recipient = proposalState.Proposal.NewRecipient
			}
		}, func() {
			proposal.ProposalOwner = oriProposalOwner
			proposal.Recipient = originRecipient
		})
	case payload.CloseProposal:
		closeProposal := p.Proposals[proposalState.Proposal.TargetProposalHash]
		if closeProposal.Status == Terminated || closeProposal.Status == Finished {
			return
		}
		*unusedAmount += getProposalUnusedBudgetAmount(closeProposal)
		p.terminatedProposal(closeProposal, height)
	case payload.SecretaryGeneral:
		oriSecretaryGeneralPublicKey := p.SecretaryGeneralPublicKey
		p.history.Append(height, func() {
			p.SecretaryGeneralPublicKey = common.BytesToHexString(proposalState.Proposal.SecretaryGeneralPublicKey)
		}, func() {
			p.SecretaryGeneralPublicKey = oriSecretaryGeneralPublicKey
		})
	case payload.ReserveCustomID:
		oriReservedCustomIDLists := p.ReservedCustomIDLists
		p.history.Append(height, func() {
			p.ReservedCustomIDLists = append(p.ReservedCustomIDLists, proposalState.Proposal.ReservedCustomIDList...)
		}, func() {
			p.ReservedCustomIDLists = oriReservedCustomIDLists
		})
	case payload.ReceiveCustomID:
		oriReceivedCustomIDLists := p.ReceivedCustomIDLists
		p.history.Append(height, func() {
			p.ReceivedCustomIDLists = append(p.ReceivedCustomIDLists, proposalState.Proposal.ReceivedCustomIDList...)
		}, func() {
			p.ReceivedCustomIDLists = oriReceivedCustomIDLists
		})
	case payload.RegisterSideChain:
		originRegisteredSideChainPayloadInfo := p.RegisteredSideChainPayloadInfo
		p.history.Append(height, func() {
			if info, ok := p.RegisteredSideChainPayloadInfo[height]; ok {
				info[proposalState.TxHash] = proposalState.Proposal.SideChainInfo
			} else {
				rs := make(map[common.Uint256]payload.SideChainInfo)
				rs[proposalState.TxHash] = proposalState.Proposal.SideChainInfo
				p.RegisteredSideChainPayloadInfo[height] = rs
			}
		}, func() {
			p.RegisteredSideChainPayloadInfo = originRegisteredSideChainPayloadInfo
		})
	}
}

// transferCRAgreedState will transfer CRAgreed State by Votes' reject amount.
func (p *ProposalManager) transferCRAgreedState(proposalState *ProposalState,
	height uint32, circulation common.Fixed64) (status ProposalStatus) {
	if proposalState.VotersRejectAmount >= common.Fixed64(float64(circulation)*
		p.params.CRConfiguration.VoterRejectPercentage/100.0) {
		status = VoterCanceled
		oriBudgetsStatus := make(map[uint8]BudgetStatus)
		for k, v := range proposalState.BudgetsStatus {
			oriBudgetsStatus[k] = v
		}
		p.history.Append(height, func() {
			proposalState.Status = VoterCanceled
			for k, _ := range proposalState.BudgetsStatus {
				proposalState.BudgetsStatus[k] = Closed
			}
		}, func() {
			proposalState.Status = CRAgreed
			proposalState.BudgetsStatus = oriBudgetsStatus
		})
		if proposalState.Proposal.ProposalType == payload.ReceiveCustomID {
			oriPendingReceivedCustomIDMap := p.PendingReceivedCustomIDMap
			p.history.Append(height, func() {
				for _, id := range proposalState.Proposal.ReceivedCustomIDList {
					delete(p.PendingReceivedCustomIDMap, id)
				}
			}, func() {
				p.PendingReceivedCustomIDMap = oriPendingReceivedCustomIDMap
			})
		}
	} else {
		if isSpecialProposal(proposalState.Proposal.ProposalType) {
			status = Finished
		} else {
			status = VoterAgreed
		}
		p.history.Append(height, func() {
			proposalState.Status = status
			for _, b := range proposalState.Proposal.Budgets {
				if b.Type == payload.Imprest {
					proposalState.WithdrawableBudgets[b.Stage] = b.Amount
					break
				}
			}
		}, func() {
			proposalState.Status = CRAgreed
			for _, b := range proposalState.Proposal.Budgets {
				if b.Type == payload.Imprest {
					delete(proposalState.WithdrawableBudgets, b.Stage)
					break
				}
			}
		})
	}
	return
}

func isSpecialProposal(proposalType payload.CRCProposalType) bool {
	switch proposalType {
	case payload.SecretaryGeneral, payload.ChangeProposalOwner, payload.CloseProposal, payload.ReserveCustomID,
		payload.ReceiveCustomID, payload.ChangeCustomIDFee, payload.RegisterSideChain:
		return true
	default:
		return false
	}
}

// shouldEndCRCVote returns if current Height should end CRC vote about
//
//	the specified proposal.
func (p *ProposalManager) shouldEndCRCVote(RegisterHeight uint32,
	height uint32) bool {
	//proposal.RegisterHeight
	return RegisterHeight+p.params.CRConfiguration.ProposalCRVotingPeriod <= height
}

func (p *ProposalManager) addRegisterSideChainInfo(proposalState *ProposalState, height uint32) {
	originRegisteredSideChainNames := p.RegisteredSideChainNames
	originRegisteredMagicNumbers := p.RegisteredMagicNumbers
	originRegisteredGenesisHash := p.RegisteredGenesisHashes
	p.history.Append(height, func() {
		p.RegisteredSideChainNames = append(p.RegisteredSideChainNames, proposalState.Proposal.SideChainName)
		p.RegisteredMagicNumbers = append(p.RegisteredMagicNumbers, proposalState.Proposal.MagicNumber)
		p.RegisteredGenesisHashes = append(p.RegisteredGenesisHashes, proposalState.Proposal.GenesisHash)
	}, func() {
		p.RegisteredSideChainNames = originRegisteredSideChainNames
		p.RegisteredMagicNumbers = originRegisteredMagicNumbers
		p.RegisteredGenesisHashes = originRegisteredGenesisHash
	})
}

func (p *ProposalManager) removeRegisterSideChainInfo(proposalState *ProposalState, height uint32) {
	if proposalState.Proposal.ProposalType != payload.RegisterSideChain {
		return
	}

	location := 0
	var exist bool
	for i, name := range p.RegisteredSideChainNames {
		if name == proposalState.Proposal.SideChainName {
			location = i
			exist = true
			break
		}
	}
	if exist {
		originRegisteredSideChainNames := p.RegisteredSideChainNames
		originRegisteredMagicNumbers := p.RegisteredMagicNumbers
		originRegisteredGenesisHash := p.RegisteredGenesisHashes

		p.history.Append(height, func() {
			p.RegisteredSideChainNames = []string{}
			p.RegisteredMagicNumbers = []uint32{}
			p.RegisteredGenesisHashes = []common.Uint256{}
			p.RegisteredSideChainNames = append(originRegisteredSideChainNames[0:location], originRegisteredSideChainNames[location+1:]...)
			p.RegisteredMagicNumbers = append(originRegisteredMagicNumbers[0:location], originRegisteredMagicNumbers[location+1:]...)
			p.RegisteredGenesisHashes = append(originRegisteredGenesisHash[0:location], originRegisteredGenesisHash[location+1:]...)
		}, func() {
			p.RegisteredSideChainNames = originRegisteredSideChainNames
			p.RegisteredMagicNumbers = originRegisteredMagicNumbers
			p.RegisteredGenesisHashes = originRegisteredGenesisHash
		})
	}
}

// shouldEndPublicVote returns if current Height should end public vote
// about the specified proposal.
func (p *ProposalManager) shouldEndPublicVote(VoteStartHeight uint32,
	height uint32) bool {
	return VoteStartHeight+p.params.CRConfiguration.ProposalPublicVotingPeriod <=
		height
}

func (p *ProposalManager) isProposalFull(did common.Uint168) bool {
	return p.getProposalCount(did) >= int(p.params.CRConfiguration.MaxCommitteeProposalCount)
}

func (p *ProposalManager) getProposalCount(did common.Uint168) int {
	proposalHashsSet, ok := p.ProposalHashes[did]
	if !ok {
		return 0
	}
	return proposalHashsSet.Len()
}

func (p *ProposalManager) addProposal(did common.Uint168,
	proposalHash common.Uint256) {
	proposalHashesSet, ok := p.ProposalHashes[did]
	if !ok {
		proposalHashesSet = NewProposalHashSet()
		proposalHashesSet.Add(proposalHash)
		p.ProposalHashes[did] = proposalHashesSet
		return
	}
	proposalHashesSet.Add(proposalHash)
}

func (p *ProposalManager) delProposal(did common.Uint168,
	proposalHash common.Uint256) {
	proposalHashesSet, ok := p.ProposalHashes[did]
	if ok {
		if len(proposalHashesSet) == 1 {
			delete(p.ProposalHashes, did)
			return
		}
		proposalHashesSet.Remove(proposalHash)
	}
}

// registerProposal will register proposal State in proposal manager
func (p *ProposalManager) registerProposal(tx interfaces.Transaction,
	height uint32, currentsSession uint64, history *utils.History) {
	proposal := tx.Payload().(*payload.CRCProposal)
	//The number of the proposals of the committee can not more than 128
	if p.isProposalFull(proposal.CRCouncilMemberDID) {
		return
	}
	budgetsStatus := make(map[uint8]BudgetStatus)
	for _, budget := range proposal.Budgets {
		if budget.Type == payload.Imprest {
			budgetsStatus[budget.Stage] = Withdrawable
			continue
		}
		budgetsStatus[budget.Stage] = Unfinished
	}
	proposalState := &ProposalState{
		Status:              Registered,
		Proposal:            proposal.ToProposalInfo(tx.PayloadVersion()),
		TxHash:              tx.Hash(),
		TxPayloadVer:        tx.PayloadVersion(),
		CRVotes:             map[common.Uint168]payload.VoteResult{},
		VotersRejectAmount:  common.Fixed64(0),
		RegisterHeight:      height,
		VoteStartHeight:     0,
		WithdrawnBudgets:    make(map[uint8]common.Fixed64),
		WithdrawableBudgets: make(map[uint8]common.Fixed64),
		BudgetsStatus:       budgetsStatus,
		FinalPaymentStatus:  false,
		TrackingCount:       0,
		TerminatedHeight:    0,
		ProposalOwner:       proposal.OwnerKey,
		Recipient:           proposal.Recipient,
	}
	crCouncilMemberDID := proposal.CRCouncilMemberDID
	hash := proposal.Hash(tx.PayloadVersion())

	history.Append(height, func() {
		hash := proposal.Hash(tx.PayloadVersion())
		log.Debug("registerProposal hash", hash.String())
		p.Proposals[hash] = proposalState
		p.addProposal(crCouncilMemberDID, hash)
		if _, ok := p.ProposalSession[currentsSession]; !ok {
			p.ProposalSession[currentsSession] = make([]common.Uint256, 0)
		}
		p.ProposalSession[currentsSession] =
			append(p.ProposalSession[currentsSession], proposal.Hash(tx.PayloadVersion()))
	}, func() {
		delete(p.Proposals, proposal.Hash(tx.PayloadVersion()))
		p.delProposal(crCouncilMemberDID, hash)
		if len(p.ProposalSession[currentsSession]) == 1 {
			delete(p.ProposalSession, currentsSession)
		} else {
			count := len(p.ProposalSession[currentsSession])
			p.ProposalSession[currentsSession] = p.ProposalSession[currentsSession][:count-1]
		}
	})

	// record to PendingReceivedCustomIDMap
	switch proposal.ProposalType {
	case payload.ReserveCustomID:
		oriReservedCustomID := p.ReservedCustomID
		history.Append(height, func() {
			p.ReservedCustomID = true
		}, func() {
			p.ReservedCustomID = oriReservedCustomID
		})
	case payload.ReceiveCustomID:
		oriPendingReceivedCustomIDMap := p.PendingReceivedCustomIDMap
		history.Append(height, func() {
			for _, id := range proposal.ReceivedCustomIDList {
				p.PendingReceivedCustomIDMap[id] = struct{}{}
			}
		}, func() {
			p.PendingReceivedCustomIDMap = oriPendingReceivedCustomIDMap
		})
	case payload.RegisterSideChain:
		p.addRegisterSideChainInfo(proposalState, height)
	}
}

func GetCIDByCode(code []byte) (*common.Uint168, error) {
	ct1, err := contract.CreateCRIDContractByCode(code)
	if err != nil {
		return nil, err
	}
	return ct1.ToProgramHash(), err
}

func GetDIDByCode(code []byte) (*common.Uint168, error) {
	didCode := make([]byte, len(code))
	copy(didCode, code)
	didCode = append(didCode[:len(code)-1], common.DID)
	ct1, err := contract.CreateCRIDContractByCode(didCode)
	if err != nil {
		return nil, err
	}
	return ct1.ToProgramHash(), err
}

func getCIDByPublicKey(publicKey []byte) (*common.Uint168, error) {
	pubkey, err := crypto.DecodePoint(publicKey)
	if err != nil {
		return nil, err
	}
	code, err := contract.CreateStandardRedeemScript(pubkey)
	if err != nil {
		return nil, err
	}
	ct, err := contract.CreateCRIDContractByCode(code)
	if err != nil {
		return nil, err
	}
	return ct.ToProgramHash(), nil
}

func (p *ProposalManager) proposalReview(tx interfaces.Transaction,
	height uint32, history *utils.History) {
	proposalReview := tx.Payload().(*payload.CRCProposalReview)
	proposalState := p.getProposal(proposalReview.ProposalHash)
	if proposalState == nil {
		return
	}
	did := proposalReview.DID
	oldVoteResult, oldVoteExist := proposalState.CRVotes[did]
	history.Append(height, func() {
		proposalState.CRVotes[did] = proposalReview.VoteResult
	}, func() {
		if oldVoteExist {
			proposalState.CRVotes[did] = oldVoteResult
		} else {
			delete(proposalState.CRVotes, did)
		}
	})
}

func (p *ProposalManager) proposalWithdraw(tx interfaces.Transaction,
	height uint32, history *utils.History) {
	withdrawPayload := tx.Payload().(*payload.CRCProposalWithdraw)
	proposalState := p.getProposal(withdrawPayload.ProposalHash)
	if proposalState == nil {
		return
	}
	withdrawingBudgets := make(map[uint8]common.Fixed64)
	for i, a := range proposalState.WithdrawableBudgets {
		if _, ok := proposalState.WithdrawnBudgets[i]; !ok {
			withdrawingBudgets[i] = a
		}
	}
	oriBudgetsStatus := make(map[uint8]BudgetStatus)
	for k, v := range proposalState.BudgetsStatus {
		oriBudgetsStatus[k] = v
	}
	history.Append(height, func() {
		for k, v := range withdrawingBudgets {
			proposalState.WithdrawnBudgets[k] = v
		}
		for k, v := range proposalState.BudgetsStatus {
			if v == Withdrawable {
				proposalState.BudgetsStatus[k] = Withdrawn
			}
		}
		if tx.PayloadVersion() == payload.CRCProposalWithdrawVersion01 {
			p.WithdrawableTxInfo[tx.Hash()] = common2.OutputInfo{
				Recipient: withdrawPayload.Recipient,
				Amount:    withdrawPayload.Amount,
			}
		}
	}, func() {
		for k, _ := range withdrawingBudgets {
			delete(proposalState.WithdrawnBudgets, k)
		}
		proposalState.BudgetsStatus = oriBudgetsStatus
		if tx.PayloadVersion() == payload.CRCProposalWithdrawVersion01 {
			delete(p.WithdrawableTxInfo, tx.Hash())
		}
	})
}

func (p *ProposalManager) proposalTracking(tx interfaces.Transaction,
	height uint32, history *utils.History) (unusedBudget common.Fixed64) {
	proposalTracking := tx.Payload().(*payload.CRCProposalTracking)
	proposalState := p.getProposal(proposalTracking.ProposalHash)
	if proposalState == nil {
		return
	}

	trackingType := proposalTracking.ProposalTrackingType
	owner := proposalState.ProposalOwner
	terminatedHeight := proposalState.TerminatedHeight
	status := proposalState.Status
	oriBudgetsStatus := make(map[uint8]BudgetStatus)
	for k, v := range proposalState.BudgetsStatus {
		oriBudgetsStatus[k] = v
	}

	if trackingType == payload.Terminated {
		if proposalState.Status == Terminated || proposalState.Status == Finished {
			return
		}
		for _, budget := range proposalState.Proposal.Budgets {
			if _, ok := proposalState.WithdrawableBudgets[budget.Stage]; !ok {
				unusedBudget += budget.Amount
			}
		}
	}
	if trackingType == payload.Finalized {
		for _, budget := range proposalState.Proposal.Budgets {
			if budget.Type == payload.FinalPayment {
				continue
			}
			if _, ok := proposalState.WithdrawableBudgets[budget.Stage]; !ok {
				unusedBudget += budget.Amount
			}
		}
	}

	history.Append(height, func() {
		proposalState.TrackingCount++
		switch trackingType {
		case payload.Common:
		case payload.Progress:
			proposalState.BudgetsStatus[proposalTracking.Stage] = Withdrawable
			for _, budget := range proposalState.Proposal.Budgets {
				if budget.Stage == proposalTracking.Stage {
					proposalState.WithdrawableBudgets[proposalTracking.Stage] = budget.Amount
					break
				}
			}
			if len(proposalState.WithdrawnBudgets) == len(proposalState.Proposal.Budgets)-1 {
				proposalState.FinalPaymentStatus = true
			}
		case payload.Rejected:
			if proposalTracking.Stage == 0 {
				break
			}
			if _, ok := proposalState.BudgetsStatus[proposalTracking.Stage]; !ok {
				break
			}
			proposalState.BudgetsStatus[proposalTracking.Stage] = Rejected
		case payload.ChangeOwner:
			proposalState.ProposalOwner = proposalTracking.NewOwnerKey
		case payload.Terminated:
			proposalState.TerminatedHeight = height
			proposalState.Status = Terminated
			for k, v := range proposalState.BudgetsStatus {
				if v == Unfinished || v == Rejected {
					proposalState.BudgetsStatus[k] = Closed
				}
			}
		case payload.Finalized:
			proposalState.Status = Finished
			for _, budget := range proposalState.Proposal.Budgets {
				if budget.Type == payload.FinalPayment {
					proposalState.WithdrawableBudgets[budget.Stage] = budget.Amount
					break
				}
			}
			proposalState.BudgetsStatus[proposalTracking.Stage] = Withdrawable
			for k, v := range proposalState.BudgetsStatus {
				if v == Unfinished || v == Rejected {
					proposalState.BudgetsStatus[k] = Closed
				}
			}
		}
	}, func() {
		proposalState.TrackingCount--
		switch trackingType {
		case payload.Common:
		case payload.Progress:
			delete(proposalState.WithdrawableBudgets, proposalTracking.Stage)
			proposalState.BudgetsStatus = oriBudgetsStatus
			proposalState.FinalPaymentStatus = false
		case payload.Rejected:
			proposalState.BudgetsStatus = oriBudgetsStatus
		case payload.ChangeOwner:
			proposalState.ProposalOwner = owner
		case payload.Terminated:
			proposalState.BudgetsStatus = oriBudgetsStatus
			proposalState.TerminatedHeight = terminatedHeight
			proposalState.Status = status
		case payload.Finalized:
			proposalState.BudgetsStatus = oriBudgetsStatus
			proposalState.Status = status
			for _, budget := range proposalState.Proposal.Budgets {
				if budget.Type == payload.FinalPayment {
					delete(proposalState.WithdrawableBudgets, budget.Stage)
					break
				}
			}
		}
	})

	return
}

func NewProposalManager(params *config.Configuration) *ProposalManager {
	return &ProposalManager{
		ProposalKeyFrame: *NewProposalKeyFrame(),
		params:           params,
		history:          utils.NewHistory(maxHistoryCapacity),
	}
}
