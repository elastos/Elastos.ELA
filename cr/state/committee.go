// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	elaerr "github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/events"
	"github.com/elastos/Elastos.ELA/p2p"
	"github.com/elastos/Elastos.ELA/p2p/msg"
	"github.com/elastos/Elastos.ELA/utils"
)

const CRAssetsRectifyInterval = time.Minute

type Committee struct {
	KeyFrame
	mtx                  sync.RWMutex
	state                *State
	Params               *config.Params
	manager              *ProposalManager
	firstHistory         *utils.History
	lastHistory          *utils.History
	appropriationHistory *utils.History

	getCheckpoint                    func(height uint32) *Checkpoint
	GetHeight                        func() uint32
	isCurrent                        func() bool
	broadcast                        func(msg p2p.Message)
	appendToTxpool                   func(transaction interfaces.Transaction) elaerr.ELAError
	createCRCAppropriationTx         func() (interfaces.Transaction, common.Fixed64, error)
	createCRAssetsRectifyTransaction func() (interfaces.Transaction, error)
	createCRRealWithdrawTransaction  func(withdrawTransactionHashes []common.Uint256,
		outputs []*common2.OutputInfo) (interfaces.Transaction, error)
	getUTXO            func(programHash *common.Uint168) ([]*common2.UTXO, error)
	getCurrentArbiters func() [][]byte
}

type CommitteeKeyFrame struct {
	*KeyFrame
	*StateKeyFrame
	*ProposalKeyFrame
}

// Deprecated: just for testing
func (c *Committee) GetState() *State {
	return c.state
}

// Deprecated: just for testing
func (c *Committee) GetProposalManager() *ProposalManager {
	return c.manager
}

func (c *Committee) GetDetailedCRVotes(referKey common.Uint256) (
	pl payload.DetailedVoteInfo, err error) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	vote, ok := c.DetailedCRVotes[referKey]
	if !ok {
		err = errors.New("refer key not found in DetailedCRVotes")
	}
	pl = vote
	return
}

func (c *Committee) GetDetailedCRImpeachmentVotes(referKey common.Uint256) (
	pl payload.DetailedVoteInfo, err error) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	vote, ok := c.DetailedCRImpeachmentVotes[referKey]
	if !ok {
		err = errors.New("refer key not found in DetailedCRImpeachmentVotes")
	}
	pl = vote
	return
}

func (c *Committee) GetDetailedCRCProposalVotes(referKey common.Uint256) (
	pl payload.DetailedVoteInfo, err error) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	vote, ok := c.manager.DetailedCRCProposalVotes[referKey]
	if !ok {
		err = errors.New("refer key not found in DetailedCRCProposalVotes")
	}
	pl = vote
	return
}

func (c *Committee) GetAllCRCProposalVotes() (pl []payload.DetailedVoteInfo, referKeys []string, err error) {
	for referKey, info := range c.manager.DetailedCRCProposalVotes {
		pl = append(pl, info)
		referKeys = append(referKeys, referKey.String())
	}
	return
}

func (c *Committee) ExistCR(programCode []byte) bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	existCandidate := c.state.existCandidate(programCode)
	if existCandidate {
		return true
	}

	did, err := GetDIDByCode(programCode)
	if err != nil {
		return false
	}

	return c.isCRMemberByDID(*did)
}

func (c *Committee) IsCRMember(programCode []byte) bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	for _, v := range c.Members {
		if bytes.Equal(programCode, v.Info.Code) {
			return true
		}
	}
	return false
}

func (c *Committee) IsElectedCRMemberByDID(did common.Uint168) bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	for _, v := range c.Members {
		if v.Info.DID.IsEqual(did) && v.MemberState == MemberElected {
			return true
		}
	}
	return false
}

func (c *Committee) isCRMemberByDID(did common.Uint168) bool {

	for _, v := range c.Members {
		if v.Info.DID.IsEqual(did) {
			return true
		}
	}
	return false
}

func (c *Committee) IsCRMemberByDID(did common.Uint168) bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.isCRMemberByDID(did)
}

func (c *Committee) IsInVotingPeriod(height uint32) bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.isInVotingPeriod(height)
}

func (c *Committee) IsInElectionPeriod() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.InElectionPeriod
}

func (c *Committee) GetCROnDutyStartHeight() uint32 {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.LastCommitteeHeight
}

func (c *Committee) GetCROnDutyPeriod() uint32 {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.Params.CRDutyPeriod
}

func (c *Committee) GetCRVotingStartHeight() uint32 {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.LastVotingStartHeight
}

func (c *Committee) GetCRVotingPeriod() uint32 {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.Params.CRVotingPeriod
}

func (c *Committee) IsProposalAllowed(height uint32) bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	if !c.InElectionPeriod {
		return false
	}
	return !c.isInVotingPeriod(height)
}

func (c *Committee) IsAppropriationNeeded() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.NeedAppropriation
}

func (c *Committee) IsProposalResultNeeded() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.NeedRecordProposalResult
}

func (c *Committee) GetCustomIDResults() []payload.ProposalResult {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.PartProposalResults
}

func (c *Committee) GetMembersDIDs() []common.Uint168 {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	result := make([]common.Uint168, 0, len(c.Members))
	for _, v := range c.Members {
		result = append(result, v.Info.DID)
	}
	return result
}

// get all CRMembers ordered by CID
func (c *Committee) GetAllMembers() []*CRMember {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	result := getCRMembers(c.Members)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Info.DID.Compare(result[j].Info.DID) <= 0
	})
	return result
}

// copy all CRMembers ordered by CID
func (c *Committee) GetAllMembersCopy() []*CRMember {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	result := getCRMembersCopy(c.Members)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Info.DID.Compare(result[j].Info.DID) <= 0
	})
	return result
}

// get all elected CRMembers
func (c *Committee) GetElectedMembers() []*CRMember {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	if !c.InElectionPeriod {
		return []*CRMember{}
	}
	return getElectedCRMembers(c.Members)
}

// get all impeachable CRMembers
func (c *Committee) GetImpeachableMembers() []*CRMember {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	if !c.InElectionPeriod {
		return []*CRMember{}
	}
	return getImpeachableCRMembers(c.Members)
}

func (c *Committee) GetMembersCodes() [][]byte {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	result := make([][]byte, 0, len(c.Members))
	for _, v := range c.Members {
		result = append(result, v.Info.Code)
	}
	return result
}

func (c *Committee) GetMember(did common.Uint168) *CRMember {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	return c.getMember(did)
}

func (c *Committee) GetNextMember(did common.Uint168) *CRMember {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	return c.getNextMember(did)
}

func (c *Committee) GetReservedCustomIDLists() []string {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	return c.getReservedCustomIDLists()
}

func (c *Committee) getReservedCustomIDLists() []string {
	return c.manager.ReservedCustomIDLists
}

func (c *Committee) GetReceivedCustomIDLists() []string {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	return c.getReceivedCustomIDLists()
}

func (c *Committee) GetPendingReceivedCustomIDMap() map[string]struct{} {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	return c.manager.PendingReceivedCustomIDMap
}

func (c *Committee) getReceivedCustomIDLists() []string {
	return c.manager.ReceivedCustomIDLists
}

func (c *Committee) getMember(did common.Uint168) *CRMember {
	for _, m := range c.Members {
		if m.Info.DID.IsEqual(did) {
			return m
		}
	}
	return nil
}

func (c *Committee) getNextMember(did common.Uint168) *CRMember {
	for _, m := range c.NextMembers {
		if m.Info.DID.IsEqual(did) {
			return m
		}
	}
	return nil
}

func (c *Committee) GetMemberByNodePublicKey(nodePK []byte) *CRMember {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.getMemberByNodePublicKey(nodePK)
}

func (c *Committee) getMemberByNodePublicKey(nodePK []byte) *CRMember {
	for _, m := range c.Members {
		if bytes.Equal(m.DPOSPublicKey, nodePK) {
			return m
		}
	}
	return nil
}

// update Candidates State in voting period
func (c *Committee) updateVotingCandidatesState(height uint32) {
	if c.isInVotingPeriod(height) {
		// Check if any pending candidates has got 6 confirms, set them to activate.
		activateCandidateFromPending :=
			func(key common.Uint168, candidate *Candidate) {
				c.state.History.Append(height, func() {
					candidate.State = Active
					c.state.Candidates[key] = candidate
				}, func() {
					candidate.State = Pending
					c.state.Candidates[key] = candidate
				})
			}

		pendingCandidates := c.state.GetCandidates(Pending)

		if len(pendingCandidates) > 0 {
			for _, candidate := range pendingCandidates {
				if height-candidate.RegisterHeight+1 >= ActivateDuration {
					activateCandidateFromPending(candidate.Info.CID, candidate)
				}
			}
		}
	}
}

// update Candidates deposit coin
func (c *Committee) updateCandidatesDepositCoin(height uint32) {
	updateDepositCoin := func(key common.Uint168, candidate *Candidate) {
		oriDepositAmount := c.state.DepositInfo[key].DepositAmount
		c.state.History.Append(height, func() {
			c.state.DepositInfo[key].DepositAmount -= MinDepositAmount
		}, func() {
			c.state.DepositInfo[key].DepositAmount = oriDepositAmount
		})
	}

	canceledCandidates := c.state.GetCandidates(Canceled)
	for _, candidate := range canceledCandidates {
		if height-candidate.CancelHeight == c.Params.CRDepositLockupBlocks {
			updateDepositCoin(candidate.Info.CID, candidate)
		}
	}
}

func (c *Committee) ProcessBlock(block *types.Block, confirm *payload.Confirm) {
	c.mtx.Lock()
	if block.Height < c.Params.CRVotingStartHeight {
		c.mtx.Unlock()
		return
	}

	// Get UTXOs of CR assets address and committee address.
	c.recordCRCRelatedAddressOutputs(block)

	// If reached the voting start Height, record the last voting start Height.
	c.recordLastVotingStartHeight(block.Height)

	c.processTransactions(block.Transactions, block.Height)
	c.updateVotingCandidatesState(block.Height)
	c.updateCandidatesDepositCoin(block.Height)
	c.state.History.Commit(block.Height)

	inElectionPeriod := c.tryStartVotingPeriod(block.Height)
	c.updateProposals(block.Height, inElectionPeriod)
	c.updateCirculationAmount(c.lastHistory, block.Height)
	c.updateInactiveCountPenalty(c.lastHistory, block.Height)
	c.updateCRInactiveStatus(c.lastHistory, block.Height)

	if block.Height >= c.Params.DPoSV2StartHeight {
		if c.shouldEndVoting(block.Height) {
			c.endVoting(block.Height)
		}
	}
	needChg := false
	if c.shouldChangeCommittee(block.Height) && c.changeCommittee(block.Height) {
		needChg = true
	}
	c.lastHistory.Commit(block.Height)

	if block.Height >= c.Params.CRCProposalWithdrawPayloadV1Height &&
		len(c.manager.WithdrawableTxInfo) != 0 {
		c.createRealWithdrawTransaction(block.Height)
	}

	if c.NeedRecordProposalResult {
		c.createProposalResultTransaction(block.Height)
	}
	if needChg {
		lockedAmount := c.createAppropriationTransaction(block.Height)
		c.recordCurrentStageAmount(block.Height, lockedAmount)
		c.appropriationHistory.Commit(block.Height)
	} else {
		if c.CRAssetsAddressUTXOCount >=
			c.Params.MaxCRAssetsAddressUTXOCount+c.Params.CoinbaseMaturity &&
			block.Height >= c.Params.CRAssetsRectifyTransactionHeight {
			c.createRectifyCRAssetsTransaction(block.Height)
		}
	}
	c.mtx.Unlock()

	if needChg {
		events.Notify(events.ETCRCChangeCommittee, block)
	}
}

func (c *Committee) updateInactiveCountPenalty(history *utils.History, height uint32) {
	for _, v := range c.Members {
		cr := v
		if cr.MemberState == MemberInactive || cr.MemberState == MemberIllegal {
			history.Append(height, func() {
				cr.PenaltyBlockCount += 1
			}, func() {
				cr.PenaltyBlockCount -= 1
			})
		}
	}
}

func (c *Committee) checkAndSetMemberToInactive(history *utils.History, height uint32) {
	for _, v := range c.Members {
		m := v
		if len(m.DPOSPublicKey) == 0 && m.MemberState == MemberElected {
			history.Append(height, func() {
				m.MemberState = MemberInactive
				if height >= c.Params.ChangeCommitteeNewCRHeight {
					c.state.UpdateCRInactivePenalty(m.Info.CID, height)
				}
			}, func() {
				m.MemberState = MemberElected
				if height >= c.Params.ChangeCommitteeNewCRHeight {
					c.state.RevertUpdateCRInactivePenalty(m.Info.CID, height)
				}
			})
		}
	}
}

func (c *Committee) updateCRInactiveStatus(history *utils.History, height uint32) {
	if height > c.Params.DPoSV2StartHeight {
		if height < c.LastVotingStartHeight+c.Params.CRVotingPeriod+c.Params.CRClaimPeriod {
			return
		}

		c.checkAndSetMemberToInactive(history, height)
		return
	}

	if c.state.CurrentSession == 0 {
		return
	} else if c.state.CurrentSession == 1 {
		if height < c.Params.CRClaimDPOSNodeStartHeight+c.Params.CRClaimDPOSNodePeriod {
			return
		}
		c.checkAndSetMemberToInactive(history, height)
	} else {
		if height < c.LastCommitteeHeight+c.Params.CRClaimDPOSNodePeriod {
			return
		}
		c.checkAndSetMemberToInactive(history, height)
	}
}

func (c *Committee) updateProposals(height uint32, inElectionPeriod bool) {
	unusedAmount, results := c.manager.updateProposals(
		height, c.CirculationAmount, inElectionPeriod)
	oriLastCIDProposalResults := c.PartProposalResults
	oriNeedCIDProposalResult := c.NeedRecordProposalResult
	var needCIDProposalResult bool
	if len(results) != 0 {
		needCIDProposalResult = true
	}
	c.manager.history.Append(height, func() {
		c.CRCCommitteeUsedAmount -= unusedAmount
		c.PartProposalResults = results
		c.NeedRecordProposalResult = needCIDProposalResult
	}, func() {
		c.CRCCommitteeUsedAmount += unusedAmount
		c.PartProposalResults = oriLastCIDProposalResults
		c.NeedRecordProposalResult = oriNeedCIDProposalResult
	})
	c.manager.history.Commit(height)
}

func (c *Committee) updateCRMembers(
	height uint32, inElectionPeriod bool) (newImpeachedCount uint32) {
	if !inElectionPeriod {
		return
	}
	circulation := c.CirculationAmount
	for _, v := range c.Members {
		if v.MemberState != MemberElected && v.MemberState != MemberInactive &&
			v.MemberState != MemberIllegal {
			continue
		}

		if v.ImpeachmentVotes >= common.Fixed64(float64(circulation)*
			c.Params.VoterRejectPercentage/100.0) {
			c.transferCRMemberState(v, height)
			newImpeachedCount++
		}
	}
	return
}

func (c *Committee) transferCRMemberState(crMember *CRMember, height uint32) {
	oriPenalty := c.state.DepositInfo[crMember.Info.CID].Penalty
	oriDepositAmount := c.state.DepositInfo[crMember.Info.CID].DepositAmount
	oriMemberState := crMember.MemberState
	penalty := c.getMemberPenalty(height, crMember, true)
	c.lastHistory.Append(height, func() {
		crMember.MemberState = MemberImpeached
		c.state.DepositInfo[crMember.Info.CID].Penalty = penalty
		c.state.DepositInfo[crMember.Info.CID].DepositAmount -= MinDepositAmount
	}, func() {
		crMember.MemberState = oriMemberState
		c.state.DepositInfo[crMember.Info.CID].Penalty = oriPenalty
		c.state.DepositInfo[crMember.Info.CID].DepositAmount = oriDepositAmount
	})
	return
}

func (c *Committee) endVoting(height uint32) bool {
	if err := c.updateNextCommitteeMembers(height); err != nil {
		log.Warn("[ProcessBlock] end voting period error: ", err)
		return false
	}

	return true
}

func (c *Committee) changeCommittee(height uint32) bool {
	if c.shouldCleanHistory(height) {
		oriHistoryMembers := copyHistoryMembersMap(c.HistoryMembers)
		oriHistoryCandidates := copyHistoryCandidateMap(c.state.HistoryCandidates)
		c.lastHistory.Append(height, func() {
			c.HistoryMembers = make(map[uint64]map[common.Uint168]*CRMember)
			c.state.HistoryCandidates = make(map[uint64]map[common.Uint168]*Candidate)
		}, func() {
			c.HistoryMembers = oriHistoryMembers
			c.state.HistoryCandidates = oriHistoryCandidates
		})
	}
	if err := c.changeCommitteeMembers(height); err != nil {
		log.Warn("[ProcessBlock] change committee members error: ", err)
		return false
	}

	c.resetCRCCommitteeUsedAmount(height)
	c.resetProposalHashesSet(height)
	return true
}

func (c *Committee) createProposalResultTransaction(height uint32) {

	if height == c.GetHeight() {
		sort.Slice(c.PartProposalResults, func(i, j int) bool {
			return c.PartProposalResults[i].ProposalHash.Compare(c.PartProposalResults[j].ProposalHash) < 0
		})
		tx := functions.CreateTransaction(
			common2.TxVersion09,
			common2.ProposalResult,
			0,
			&payload.RecordProposalResult{
				ProposalResults: c.PartProposalResults,
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		log.Info("create record proposal result transaction:", tx.Hash())
		if c.isCurrent != nil && c.broadcast != nil && c.
			appendToTxpool != nil {
			go func() {
				if c.isCurrent() {
					if err := c.appendToTxpool(tx); err == nil {
						c.broadcast(msg.NewTx(tx))
					} else {
						log.Warn("create record proposal result transaction"+
							" append to tx pool err ", err)
					}
				}
			}()
		}
	}
	return
}

func (c *Committee) createAppropriationTransaction(height uint32) common.Fixed64 {
	lockedAmount := common.Fixed64(0)
	if c.createCRCAppropriationTx != nil && height == c.GetHeight() {
		tx, amount, err := c.createCRCAppropriationTx()
		if err != nil {
			log.Error("create appropriation tx failed:", err.Error())
			return 0
		} else if tx == nil {
			log.Info("no need to create appropriation")
			oriNeedAppropriation := c.NeedAppropriation
			c.appropriationHistory.Append(height, func() {
				c.NeedAppropriation = false
			}, func() {
				c.NeedAppropriation = oriNeedAppropriation
			})
			return 0
		}
		lockedAmount = amount

		log.Info("create CRCAppropriation transaction:", tx.Hash())
		if c.isCurrent != nil && c.broadcast != nil && c.
			appendToTxpool != nil {
			go func() {
				if c.isCurrent() {
					if err := c.appendToTxpool(tx); err == nil {
						c.broadcast(msg.NewTx(tx))
					} else {
						log.Warn("create CRCAppropriation append to tx pool err ", err)
					}
				}
			}()
		}
	}
	return lockedAmount
}

func (c *Committee) createRectifyCRAssetsTransaction(height uint32) {
	if c.createCRAssetsRectifyTransaction != nil && height == c.GetHeight() {
		if c.isCurrent != nil && c.broadcast != nil && c.
			appendToTxpool != nil {
			go func() {
				time.Sleep(CRAssetsRectifyInterval)
				tx, err := c.createCRAssetsRectifyTransaction()
				if err != nil {
					log.Error("create rectify UTXOs tx failed:", err.Error())
					return
				}
				log.Info("create rectify UTXOs transaction:", tx.Hash())
				if c.isCurrent() {
					if err := c.appendToTxpool(tx); err == nil {
						c.broadcast(msg.NewTx(tx))
					} else {
						log.Warn("create rectify UTXOs append to tx pool err ", err)
					}
				}
			}()
		}
	}
	return
}

func (c *Committee) createRealWithdrawTransaction(height uint32) {
	if c.createCRRealWithdrawTransaction != nil && height == c.GetHeight() {
		withdrawTransactionHahses := make([]common.Uint256, 0)
		ouputs := make([]*common2.OutputInfo, 0)
		for k, v := range c.manager.WithdrawableTxInfo {
			withdrawTransactionHahses = append(withdrawTransactionHahses, k)
			outputInfo := v
			ouputs = append(ouputs, &outputInfo)
		}
		tx, err := c.createCRRealWithdrawTransaction(withdrawTransactionHahses, ouputs)
		if err != nil {
			log.Error("create real withdraw tx failed:", err.Error())
			return
		}

		log.Info("create real withdraw transaction:", tx.Hash())
		if c.isCurrent != nil && c.broadcast != nil && c.
			appendToTxpool != nil {
			go func() {
				if c.isCurrent() {
					if err := c.appendToTxpool(tx); err == nil {
						c.broadcast(msg.NewTx(tx))
					} else {
						log.Warn("create real withdraw transaction "+
							"append to tx pool err ", err)
					}
				}
			}()
		}
	}
	return
}

func (c *Committee) resetCRCCommitteeUsedAmount(height uint32) {
	// todo add finished proposals into finished map
	var budget common.Fixed64
	for _, v := range c.manager.Proposals {
		if v.Status == CRCanceled || v.Status == VoterCanceled ||
			v.Status == Aborted {
			continue
		}
		if v.Status == Terminated || v.Status == Finished {
			for _, b := range v.Proposal.Budgets {
				if _, ok := v.WithdrawableBudgets[b.Stage]; !ok {
					continue
				}
				if _, ok := v.WithdrawnBudgets[b.Stage]; ok {
					continue
				}
				budget += b.Amount
			}
			continue
		}
		for _, b := range v.Proposal.Budgets {
			if _, ok := v.WithdrawnBudgets[b.Stage]; !ok {
				budget += b.Amount
			}
		}
	}

	oriUsedAmount := c.CRCCommitteeUsedAmount
	c.lastHistory.Append(height, func() {
		c.CRCCommitteeUsedAmount = budget
	}, func() {
		c.CRCCommitteeUsedAmount = oriUsedAmount
	})

	oriNeedAppropriation := c.NeedAppropriation
	if c.GetHeight != nil && height == c.GetHeight() {
		c.lastHistory.Append(height, func() {
			c.NeedAppropriation = true
		}, func() {
			c.NeedAppropriation = oriNeedAppropriation
		})
	}

}

func (c *Committee) resetProposalHashesSet(height uint32) {
	oriHashesSet := c.manager.ProposalHashes
	c.lastHistory.Append(height, func() {
		c.manager.ProposalHashes = make(map[common.Uint168]ProposalHashSet)
	}, func() {
		c.manager.ProposalHashes = oriHashesSet
	})
}

func (c *Committee) GetCommitteeCanUseAmount() common.Fixed64 {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.CRCCurrentStageAmount - c.CRCCommitteeUsedAmount
}

func (c *Committee) recordCurrentStageAmount(height uint32, lockedAmount common.Fixed64) {
	oriCurrentStageAmount := c.CRCCurrentStageAmount
	oriAppropriationAmount := c.AppropriationAmount
	oriCommitteeUsedAmount := c.CommitteeUsedAmount
	c.appropriationHistory.Append(height, func() {
		c.CommitteeUsedAmount = c.CRCCommitteeUsedAmount
		c.AppropriationAmount = common.Fixed64(float64(c.CRCFoundationBalance-
			lockedAmount) * c.Params.CRCAppropriatePercentage / 100.0)
		c.CRCCurrentStageAmount = c.CRCCommitteeBalance + c.AppropriationAmount
		log.Infof("current stage amount:%s,appropriation amount:%s",
			c.CRCCurrentStageAmount, c.AppropriationAmount)
		log.Infof("CR expenses address balance: %s,CR assets address "+
			"balance:%s, locked amount: %s,CR expenses address used amount: %s",
			c.CRCCommitteeBalance, c.CRCFoundationBalance,
			lockedAmount, c.CRCCommitteeUsedAmount)
	}, func() {
		c.CommitteeUsedAmount = oriCommitteeUsedAmount
		c.CRCCurrentStageAmount = oriCurrentStageAmount
		c.AppropriationAmount = oriAppropriationAmount
	})
}

func (c *Committee) recordCRCRelatedAddressOutputs(block *types.Block) {
	for _, tx := range block.Transactions {
		for i, output := range tx.Outputs() {
			if output.ProgramHash.IsEqual(c.Params.CRAssetsAddress) {
				key := common2.NewOutPoint(tx.Hash(), uint16(i)).ReferKey()
				value := output.Value
				c.firstHistory.Append(block.Height, func() {
					c.state.CRCFoundationOutputs[key] = value
				}, func() {
					delete(c.state.CRCFoundationOutputs, key)
				})
			} else if output.ProgramHash.IsEqual(c.Params.CRExpensesAddress) {
				key := common2.NewOutPoint(tx.Hash(), uint16(i)).ReferKey()
				value := output.Value
				c.firstHistory.Append(block.Height, func() {
					c.state.CRCCommitteeOutputs[key] = value
				}, func() {
					delete(c.state.CRCCommitteeOutputs, key)
				})
			}
		}
	}
	c.firstHistory.Commit(block.Height)
}

func (c *Committee) updateCirculationAmount(history *utils.History, height uint32) {
	circulationAmount := common.Fixed64(config.OriginIssuanceAmount) +
		common.Fixed64(height)*c.Params.GetBlockReward(height) -
		c.CRCFoundationBalance - c.CRCCommitteeBalance - c.DestroyedAmount
	oriCirculationAmount := c.CirculationAmount
	history.Append(height, func() {
		c.CirculationAmount = circulationAmount
	}, func() {
		c.CirculationAmount = oriCirculationAmount
	})
}

func (c *Committee) recordLastVotingStartHeight(height uint32) {
	if c.isInVotingPeriod(height) {
		return
	}
	// Update last voting start Height one block ahead.
	if height == c.getNextVotingStartHeight(height) {
		lastVotingStartHeight := c.LastVotingStartHeight
		c.state.History.Append(height, func() {
			c.LastVotingStartHeight = height + 1
		}, func() {
			c.LastVotingStartHeight = lastVotingStartHeight
		})
	}
}

func (c *Committee) getNextVotingStartHeight(height uint32) uint32 {
	if height >= c.Params.DPoSV2StartHeight {
		return c.LastCommitteeHeight + c.Params.CRDutyPeriod -
			c.Params.CRVotingPeriod - c.Params.CRClaimPeriod - 1
	}
	return c.LastCommitteeHeight + c.Params.CRDutyPeriod -
		c.Params.CRVotingPeriod - 1
}

func (c *Committee) tryStartVotingPeriod(height uint32) (inElection bool) {
	inElection = c.InElectionPeriod
	if !c.InElectionPeriod {
		return
	}

	newImpeachedCount := c.updateCRMembers(height, inElection)

	var impeachedCount uint32
	for _, m := range c.Members {
		if m.MemberState == MemberImpeached {
			impeachedCount++
		}
	}
	if impeachedCount+newImpeachedCount >
		c.Params.CRMemberCount-c.Params.CRAgreementCount {
		lastVotingStartHeight := c.LastVotingStartHeight
		inElectionPeriod := c.InElectionPeriod
		c.lastHistory.Append(height, func() {
			c.InElectionPeriod = false
			if c.LastVotingStartHeight == 0 {
				c.LastVotingStartHeight = height
			} else if !c.isInVotingPeriod(height) {
				c.LastVotingStartHeight = height
			}
		}, func() {
			c.InElectionPeriod = inElectionPeriod
			c.LastVotingStartHeight = lastVotingStartHeight
		})
		inElection = false

		for _, v := range c.Members {
			if (v.MemberState == MemberElected || v.MemberState == MemberInactive ||
				v.MemberState == MemberIllegal) &&
				v.ImpeachmentVotes < common.Fixed64(float64(c.CirculationAmount)*
					c.Params.VoterRejectPercentage/100.0) {
				c.terminateCRMember(v, height)
			}
		}
	}

	return
}

func (c *Committee) terminateCRMember(crMember *CRMember, height uint32) {
	oriPenalty := c.state.DepositInfo[crMember.Info.CID].Penalty
	oriDepositAmount := c.state.DepositInfo[crMember.Info.CID].DepositAmount
	oriMemberState := crMember.MemberState
	penalty := c.getMemberPenalty(height, crMember, false)
	c.lastHistory.Append(height, func() {
		crMember.MemberState = MemberTerminated
		c.state.DepositInfo[crMember.Info.CID].Penalty = penalty
		c.state.DepositInfo[crMember.Info.CID].DepositAmount -= MinDepositAmount
	}, func() {
		crMember.MemberState = oriMemberState
		c.state.DepositInfo[crMember.Info.CID].Penalty = oriPenalty
		c.state.DepositInfo[crMember.Info.CID].DepositAmount = oriDepositAmount
	})
}

func (c *Committee) processImpeachment(height uint32, member []byte,
	votes common.Fixed64, history *utils.History) {
	var crMember *CRMember
	for _, v := range c.Members {
		if bytes.Equal(v.Info.CID.Bytes(), member) &&
			(v.MemberState == MemberElected ||
				v.MemberState == MemberInactive || v.MemberState == MemberIllegal) {
			crMember = v
			break
		}
	}
	if crMember == nil {
		return
	}
	history.Append(height, func() {
		crMember.ImpeachmentVotes += votes
	}, func() {
		crMember.ImpeachmentVotes -= votes
	})
	return
}

func (c *Committee) processCancelImpeachmentV2(height uint32, member []byte,
	votes common.Fixed64, history *utils.History) {
	var crMember *CRMember
	for _, v := range c.Members {
		if bytes.Equal(v.Info.CID.Bytes(), member) &&
			(v.MemberState == MemberElected ||
				v.MemberState == MemberInactive || v.MemberState == MemberIllegal) {
			crMember = v
			break
		}
	}
	if crMember == nil {
		return
	}
	history.Append(height, func() {
		crMember.ImpeachmentVotes -= votes
	}, func() {
		crMember.ImpeachmentVotes += votes
	})
	return
}

func (c *Committee) processCRCAppropriation(height uint32, history *utils.History) {
	history.Append(height, func() {
		c.NeedAppropriation = false
	}, func() {
		c.NeedAppropriation = true
	})
}

func (c *Committee) processCRCRealWithdraw(tx interfaces.Transaction,
	height uint32, history *utils.History) {

	txs := make(map[common.Uint256]common2.OutputInfo)
	for k, v := range c.manager.WithdrawableTxInfo {
		txs[k] = v
	}
	withdrawPayload := tx.Payload().(*payload.CRCProposalRealWithdraw)
	history.Append(height, func() {
		for _, hash := range withdrawPayload.WithdrawTransactionHashes {
			delete(c.manager.WithdrawableTxInfo, hash)
		}
	}, func() {
		c.manager.WithdrawableTxInfo = txs
	})
}

func getCrossChainSignedPubKeys(program program.Program, data []byte) ([][]byte, error) {
	code := program.Code
	// Get N parameter
	n := int(code[len(code)-2]) - crypto.PUSH1 + 1
	// Get M parameter
	m := int(code[0]) - crypto.PUSH1 + 1
	publicKeys, err := crypto.ParseCrossChainScript(code)
	if err != nil {
		return nil, err
	}

	return getSignedPubKeys(m, n, publicKeys, program.Parameter, data)
}

func getSignedPubKeys(m, n int, publicKeys [][]byte, signatures, data []byte) ([][]byte, error) {
	if len(publicKeys) != n {
		return nil, errors.New("invalid multi sign public key script count")
	}
	if len(signatures)%crypto.SignatureScriptLength != 0 {
		return nil, errors.New("invalid multi sign signatures, length not match")
	}
	if len(signatures)/crypto.SignatureScriptLength < m {
		return nil, errors.New("invalid signatures, not enough signatures")
	}
	if len(signatures)/crypto.SignatureScriptLength > n {
		return nil, errors.New("invalid signatures, too many signatures")
	}

	var verified = make(map[common.Uint256]struct{})
	var retKeys [][]byte
	for i := 0; i < len(signatures); i += crypto.SignatureScriptLength {
		// Remove length byte
		sign := signatures[i : i+crypto.SignatureScriptLength][1:]
		// Match public key with signature
		for _, publicKey := range publicKeys {
			pubKey, err := crypto.DecodePoint(publicKey[1:])
			if err != nil {
				return nil, err
			}
			err = crypto.Verify(*pubKey, data, sign)
			if err == nil {
				hash := sha256.Sum256(publicKey)
				if _, ok := verified[hash]; ok {
					return nil, errors.New("duplicated signatures")
				}
				verified[hash] = struct{}{}
				retKeys = append(retKeys, publicKey[1:])
				break // back to public keys loop
			}
		}
	}
	// Check signatures count
	if len(verified) < m {
		return nil, errors.New("matched signatures not enough")
	}

	return retKeys, nil
}

func (c *Committee) processsWithdrawFromSideChain(tx interfaces.Transaction,
	height uint32, history *utils.History) {
	log.Infof("currentWithdrawFromSideChainIndex is %d, CrossChainMonitorInterval is %d", c.CurrentWithdrawFromSideChainIndex, c.Params.CrossChainMonitorInterval)
	reachTop := false
	if c.CurrentWithdrawFromSideChainIndex == c.Params.CrossChainMonitorInterval {
		reachTop = true
		c.CurrentWithdrawFromSideChainIndex = 0
	} else {
		c.CurrentWithdrawFromSideChainIndex += 1
	}
	log.Infof("CurrentSignedWithdrawFromSideChainKeys %v", c.CurrentSignedWithdrawFromSideChainKeys)
	electedMembers := getOriginElectedCRMembers(c.Members)
	electedMemAll := make(map[string]*CRMember)
	for _, elected := range electedMembers {
		electedMemAll[hex.EncodeToString(elected.DPOSPublicKey)] = elected
	}
	var publicKeys [][]byte
	if tx.PayloadVersion() == payload.WithdrawFromSideChainVersionV2 {
		allPulicKeys := c.getCurrentArbiters()
		pld := tx.Payload().(*payload.WithdrawFromSideChain)
		for _, index := range pld.Signers {
			publicKeys = append(publicKeys, allPulicKeys[index])
		}
	} else {
		buf := new(bytes.Buffer)
		tx.SerializeUnsigned(buf)
		data := buf.Bytes()
		var err error
		for _, p := range tx.Programs() {
			publicKeys, err = getCrossChainSignedPubKeys(*p, data)
			if err != nil {
				return
			}
			if len(publicKeys) != 0 {
				break
			}
		}
	}
	for _, pub := range publicKeys {
		pubStr := hex.EncodeToString(pub)
		if !isArbiterEixst(pubStr, c.CurrentSignedWithdrawFromSideChainKeys) {
			c.CurrentSignedWithdrawFromSideChainKeys = append(c.CurrentSignedWithdrawFromSideChainKeys, pubStr)
		}
	}

	if reachTop {
		log.Info("reach top")
		for k, m := range electedMemAll {
			tmp := k
			tmpMem := m
			if !isArbiterEixst(tmp, c.CurrentSignedWithdrawFromSideChainKeys) {
				if tmpMem != nil && tmpMem.MemberState == MemberElected {
					history.Append(height, func() {
						tmpMem.MemberState = MemberInactive
						log.Info("Set to inactive", tmpMem.Info.NickName)
						if height >= c.Params.ChangeCommitteeNewCRHeight {
							c.state.UpdateCRInactivePenalty(tmpMem.Info.CID, height)
						}
					}, func() {
						tmpMem.MemberState = MemberElected
						if height >= c.Params.ChangeCommitteeNewCRHeight {
							c.state.RevertUpdateCRInactivePenalty(tmpMem.Info.CID, height)
						}
					})
				}
			}
		}
		c.CurrentSignedWithdrawFromSideChainKeys = make([]string, 0)
	}

}

func isArbiterEixst(cmpK string, keys []string) bool {
	for _, v := range keys {
		if cmpK == v {
			return true
		}
	}
	return false
}

func (c *Committee) activateProducer(tx interfaces.Transaction,
	height uint32, history *utils.History) {
	apPayload := tx.Payload().(*payload.ActivateProducer)
	crMember := c.getMemberByNodePublicKey(apPayload.NodePublicKey)
	if crMember != nil && (crMember.MemberState == MemberInactive ||
		crMember.MemberState == MemberIllegal) {
		oriInactiveCount := crMember.InactiveCount
		history.Append(height, func() {
			crMember.ActivateRequestHeight = height
			crMember.InactiveCount = 0
		}, func() {
			crMember.ActivateRequestHeight = math.MaxUint32
			crMember.InactiveCount = oriInactiveCount
		})
	}
}

func (c *Committee) processCRCouncilMemberClaimNode(tx interfaces.Transaction,
	height uint32, history *utils.History) {
	claimNodePayload := tx.Payload().(*payload.CRCouncilMemberClaimNode)
	var cr *CRMember
	if height >= c.Params.DPoSV2StartHeight {
		switch tx.PayloadVersion() {
		case payload.CurrentCRClaimDPoSNodeVersion:
			cr = c.getMember(claimNodePayload.CRCouncilCommitteeDID)
			if cr == nil {
				return
			}
			oriClaimDPoSKeys := copyClaimedDPoSKeysMap(c.ClaimedDPoSKeys)
			history.Append(height, func() {
				c.ClaimedDPoSKeys[hex.EncodeToString(claimNodePayload.NodePublicKey)] = struct{}{}
				if len(cr.DPOSPublicKey) != 0 {
					delete(c.ClaimedDPoSKeys, hex.EncodeToString(cr.DPOSPublicKey))
				}
			}, func() {
				c.ClaimedDPoSKeys = oriClaimDPoSKeys
			})
		case payload.NextCRClaimDPoSNodeVersion:
			cr = c.getNextMember(claimNodePayload.CRCouncilCommitteeDID)
			if cr == nil {
				return
			}
			oriNextClaimDPoSKeys := copyClaimedDPoSKeysMap(c.NextClaimedDPoSKeys)
			history.Append(height, func() {
				c.NextClaimedDPoSKeys[hex.EncodeToString(claimNodePayload.NodePublicKey)] = struct{}{}
				if len(cr.DPOSPublicKey) != 0 {
					delete(c.NextClaimedDPoSKeys, hex.EncodeToString(cr.DPOSPublicKey))
				}
			}, func() {
				c.NextClaimedDPoSKeys = oriNextClaimDPoSKeys
			})
		}
	} else {
		cr = c.getMember(claimNodePayload.CRCouncilCommitteeDID)
		if cr == nil {
			return
		}
	}
	oriPublicKey := cr.DPOSPublicKey
	oriMemberState := cr.MemberState
	oriInactiveCount := cr.InactiveCount
	history.Append(height, func() {
		cr.DPOSPublicKey = claimNodePayload.NodePublicKey
		if cr.MemberState == MemberInactive {
			cr.MemberState = MemberElected
			cr.InactiveCount = 0
		}
	}, func() {
		cr.DPOSPublicKey = oriPublicKey
		cr.MemberState = oriMemberState
		cr.InactiveCount = oriInactiveCount
	})
}

func (c *Committee) GetDepositAmountByPublicKey(
	publicKey string) (common.Fixed64, common.Fixed64, common.Fixed64, common.Fixed64, error) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	pubkey, err := common.HexStringToBytes(publicKey)
	if err != nil {
		return 0, 0, 0, 0, errors.New("invalid public key")
	}
	return c.state.getDepositInfoByPublicKey(pubkey)
}

func (c *Committee) GetDepositAmountByID(
	id common.Uint168) (common.Fixed64, common.Fixed64, common.Fixed64, common.Fixed64, error) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	cid, exist := c.state.getExistCIDByID(id)
	if !exist {
		return 0, 0, 0, 0, errors.New("ID does not exist")
	}
	return c.state.getDepositInfoByCID(*cid)
}

func (c *Committee) GetAvailableDepositAmount(cid common.Uint168) common.Fixed64 {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	return c.state.getAvailableDepositAmount(cid)
}

func (c *Committee) getHistoryMember(code []byte) []*CRMember {
	members := make([]*CRMember, 0)
	for _, v := range c.HistoryMembers {
		for _, m := range v {
			if bytes.Equal(m.Info.Code, code) {
				members = append(members, m)
			}
		}
	}
	return members
}

func (c *Committee) RollbackTo(height uint32) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	currentHeight := c.lastHistory.Height()
	for i := currentHeight - 1; i >= height; i-- {
		if err := c.appropriationHistory.RollbackTo(i); err != nil {
			log.Debug("committee appropriationHistory rollback err:", err)
		}
		if err := c.lastHistory.RollbackTo(i); err != nil {
			log.Debug("committee last History rollback err:", err)
		}
		if err := c.manager.history.RollbackTo(i); err != nil {
			log.Debug("manager rollback err:", err)
		}
		if err := c.state.rollbackTo(i); err != nil {
			log.Debug("State rollback err:", err)
		}
		if err := c.firstHistory.RollbackTo(i); err != nil {
			log.Debug("committee first History rollback err:", err)
		}
	}

	return nil
}

func (c *Committee) Recover(checkpoint *Checkpoint) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.state.StateKeyFrame = checkpoint.StateKeyFrame
	c.KeyFrame = checkpoint.KeyFrame

	c.manager.ProposalKeyFrame = checkpoint.ProposalKeyFrame
}

func (c *Committee) shouldEndVoting(height uint32) bool {
	return height == c.LastVotingStartHeight+c.Params.CRVotingPeriod
}

func (c *Committee) shouldChangeCommittee(height uint32) bool {
	if c.LastCommitteeHeight == 0 {
		if height < c.Params.CRCommitteeStartHeight {
			return false
		} else if height == c.Params.CRCommitteeStartHeight {
			return true
		}
	}

	if c.LastVotingStartHeight == 0 {
		return height == c.LastCommitteeHeight+c.Params.CRDutyPeriod
	}

	if height >= c.Params.DPoSV2StartHeight {
		return height == c.LastVotingStartHeight+c.Params.CRVotingPeriod+c.Params.CRClaimPeriod
	}

	return height == c.LastVotingStartHeight+c.Params.CRVotingPeriod
}

func (c *Committee) shouldCleanHistory(height uint32) bool {
	if height >= c.Params.DPoSV2StartHeight {
		return c.LastVotingStartHeight == c.LastCommitteeHeight+
			c.Params.CRDutyPeriod-c.Params.CRVotingPeriod-c.Params.CRClaimPeriod
	}

	return c.LastVotingStartHeight == c.LastCommitteeHeight+
		c.Params.CRDutyPeriod-c.Params.CRVotingPeriod
}

func (c *Committee) isInVotingPeriod(height uint32) bool {
	//todo consider emergency election later
	inVotingPeriod := func(committeeUpdateHeight uint32) bool {
		if height >= c.Params.DPoSV2StartHeight {
			return height >= committeeUpdateHeight-c.Params.CRVotingPeriod-c.Params.CRClaimPeriod &&
				height < committeeUpdateHeight-c.Params.CRClaimPeriod
		}
		return height >= committeeUpdateHeight-c.Params.CRVotingPeriod &&
			height < committeeUpdateHeight
	}
	if c.LastCommitteeHeight < c.Params.CRCommitteeStartHeight &&
		height <= c.Params.CRCommitteeStartHeight {
		return height >= c.Params.CRVotingStartHeight &&
			height < c.Params.CRCommitteeStartHeight
	} else {
		if !c.InElectionPeriod {
			if c.LastVotingStartHeight == 0 {
				return true
			}
			return height < c.LastVotingStartHeight+c.Params.CRVotingPeriod
		}
		return inVotingPeriod(c.LastCommitteeHeight + c.Params.CRDutyPeriod)
	}
}

func (c *Committee) updateNextCommitteeMembers(height uint32) error {
	candidates := c.getActiveAndExistDIDCRCandidatesDesc()
	oriInElectionPeriod := c.InElectionPeriod
	oriLastVotingStartHeight := c.LastVotingStartHeight
	if uint32(len(candidates)) < c.Params.CRMemberCount {
		c.lastHistory.Append(height, func() {
			c.InElectionPeriod = false
			c.LastVotingStartHeight = height
		}, func() {
			c.InElectionPeriod = oriInElectionPeriod
			c.LastVotingStartHeight = oriLastVotingStartHeight
		})
		return errors.New("candidates count less than required count Height" + strconv.Itoa(int(height)))
	}
	// Process current members.
	newMembers := c.processNextMembers(height, candidates)

	// Process current candidates.
	c.processCurrentCandidates(height, candidates, newMembers)

	return nil
}

func (c *Committee) changeCommitteeMembers(height uint32) error {
	if c.InElectionPeriod == true {
		c.processCurrentMembersDepositInfo(height)
	}

	if height >= c.Params.DPoSV2StartHeight {
		// Process current members.
		c.resetNextMembers(height)

		oriInElectionPeriod := c.InElectionPeriod
		oriLastCommitteeHeight := c.LastCommitteeHeight
		c.lastHistory.Append(height, func() {
			c.state.CurrentSession += 1
			c.InElectionPeriod = true
			c.LastCommitteeHeight = height
		}, func() {
			c.state.CurrentSession -= 1
			c.InElectionPeriod = oriInElectionPeriod
			c.LastCommitteeHeight = oriLastCommitteeHeight
		})

		return nil
	}

	candidates := c.getActiveAndExistDIDCRCandidatesDesc()
	oriInElectionPeriod := c.InElectionPeriod
	oriLastVotingStartHeight := c.LastVotingStartHeight
	if uint32(len(candidates)) < c.Params.CRMemberCount {
		c.lastHistory.Append(height, func() {
			c.InElectionPeriod = false
			c.LastVotingStartHeight = height
		}, func() {
			c.InElectionPeriod = oriInElectionPeriod
			c.LastVotingStartHeight = oriLastVotingStartHeight
		})
		return errors.New("candidates count less than required count Height" + strconv.Itoa(int(height)))
	}
	// Process current members.
	newMembers := c.processCurrentMembersHistory(height, candidates)

	// Process current candidates.
	c.processCurrentCandidates(height, candidates, newMembers)

	oriLastCommitteeHeight := c.LastCommitteeHeight
	c.lastHistory.Append(height, func() {
		c.state.CurrentSession += 1
		c.InElectionPeriod = true
		c.LastCommitteeHeight = height
	}, func() {
		c.state.CurrentSession -= 1
		c.InElectionPeriod = oriInElectionPeriod
		c.LastCommitteeHeight = oriLastCommitteeHeight
	})

	return nil
}

func (c *Committee) processNextMembers(height uint32,
	activeCandidates []*Candidate) map[common.Uint168]*CRMember {
	newMembers := make(map[common.Uint168]*CRMember, c.Params.CRMemberCount)
	for i := 0; i < int(c.Params.CRMemberCount); i++ {
		newMembers[activeCandidates[i].Info.DID] =
			c.generateMember(activeCandidates[i])
	}
	oriMembers := copyMembersMap(c.NextMembers)
	c.lastHistory.Append(height, func() {
		c.NextMembers = newMembers
	}, func() {
		c.NextMembers = oriMembers
	})
	return newMembers
}

func (c *Committee) resetNextMembers(height uint32) {

	oriMembers := copyMembersMap(c.Members)
	if len(c.Members) != 0 {
		if _, ok := c.HistoryMembers[c.state.CurrentSession]; !ok {
			currentSession := c.state.CurrentSession
			c.lastHistory.Append(height, func() {
				c.HistoryMembers[currentSession] =
					make(map[common.Uint168]*CRMember)
			}, func() {
				delete(c.HistoryMembers, currentSession)
			})
		}

		for _, m := range oriMembers {
			member := *m
			c.lastHistory.Append(height, func() {
				c.HistoryMembers[c.state.CurrentSession][member.Info.CID] = &member
			}, func() {
				delete(c.HistoryMembers[c.state.CurrentSession], member.Info.CID)
			})
		}
	}
	newMembers := copyMembersMap(c.NextMembers)
	oriNicknames := utils.CopyStringSet(c.state.Nicknames)
	oriVotes := utils.CopyStringSet(c.state.Votes)
	oriClaimedDPoSKyes := copyClaimedDPoSKeysMap(c.ClaimedDPoSKeys)
	oriNextClaimedDPoSKyes := copyClaimedDPoSKeysMap(c.NextClaimedDPoSKeys)
	c.lastHistory.Append(height, func() {
		c.Members = newMembers
		c.NextMembers = make(map[common.Uint168]*CRMember)
		c.state.Nicknames = map[string]struct{}{}
		c.state.Votes = map[string]struct{}{}
		c.ClaimedDPoSKeys = c.NextClaimedDPoSKeys
		c.NextClaimedDPoSKeys = make(map[string]struct{})
	}, func() {
		c.Members = oriMembers
		c.NextMembers = newMembers
		c.state.Nicknames = oriNicknames
		c.state.Votes = oriVotes
		c.ClaimedDPoSKeys = oriClaimedDPoSKyes
		c.NextClaimedDPoSKeys = oriNextClaimedDPoSKyes
	})

	return
}

func (c *Committee) processCurrentMembersHistory(height uint32,
	activeCandidates []*Candidate) map[common.Uint168]*CRMember {

	oriMembers := copyMembersMap(c.Members)
	if len(c.Members) != 0 {
		if _, ok := c.HistoryMembers[c.state.CurrentSession]; !ok {
			currentSession := c.state.CurrentSession
			c.lastHistory.Append(height, func() {
				c.HistoryMembers[currentSession] =
					make(map[common.Uint168]*CRMember)
			}, func() {
				delete(c.HistoryMembers, currentSession)
			})
		}

		for _, m := range oriMembers {
			member := *m
			c.lastHistory.Append(height, func() {
				c.HistoryMembers[c.state.CurrentSession][member.Info.CID] = &member
			}, func() {
				delete(c.HistoryMembers[c.state.CurrentSession], member.Info.CID)
			})
		}
	}

	newMembers := make(map[common.Uint168]*CRMember, c.Params.CRMemberCount)
	for i := 0; i < int(c.Params.CRMemberCount); i++ {
		newMembers[activeCandidates[i].Info.DID] =
			c.generateMember(activeCandidates[i])
	}

	oriNicknames := utils.CopyStringSet(c.state.Nicknames)
	oriVotes := utils.CopyStringSet(c.state.Votes)
	c.lastHistory.Append(height, func() {
		c.Members = newMembers
		c.state.Nicknames = map[string]struct{}{}
		c.state.Votes = map[string]struct{}{}
	}, func() {
		c.Members = oriMembers
		c.state.Nicknames = oriNicknames
		c.state.Votes = oriVotes
	})
	return newMembers
}

func (c *Committee) processCurrentMembersDepositInfo(height uint32) {
	oriMembers := copyMembersMap(c.Members)
	if len(c.Members) != 0 {
		for _, m := range oriMembers {
			member := *m
			if member.MemberState != MemberElected &&
				member.MemberState != MemberInactive &&
				member.MemberState != MemberIllegal {
				continue
			}
			oriPenalty := c.state.DepositInfo[m.Info.CID].Penalty
			oriDepositAmount := c.state.DepositInfo[m.Info.CID].DepositAmount
			penalty := c.getMemberPenalty(height, &member, false)
			c.lastHistory.Append(height, func() {
				c.state.DepositInfo[member.Info.CID].Penalty = penalty
				c.state.DepositInfo[member.Info.CID].DepositAmount -= MinDepositAmount
			}, func() {
				c.state.DepositInfo[member.Info.CID].Penalty = oriPenalty
				c.state.DepositInfo[member.Info.CID].DepositAmount = oriDepositAmount
			})
		}
	}
}

func (c *Committee) processCurrentCandidates(height uint32,
	activeCandidates []*Candidate, newMembers map[common.Uint168]*CRMember) {
	newHistoryCandidates := make(map[common.Uint168]*Candidate)
	if _, ok := c.state.HistoryCandidates[c.state.CurrentSession]; !ok {
		c.state.HistoryCandidates[c.state.CurrentSession] =
			make(map[common.Uint168]*Candidate)
	}
	membersMap := make(map[common.Uint168]struct{})
	for _, c := range activeCandidates {
		membersMap[c.Info.DID] = struct{}{}
	}
	for k, v := range c.state.Candidates {
		if _, ok := membersMap[k]; !ok {
			newHistoryCandidates[k] = v
		}
	}

	oriCandidate := copyCandidateMap(c.state.Candidates)
	currentSession := c.state.CurrentSession
	for _, candidate := range oriCandidate {
		ca := *candidate
		// if candidate changed to member, no need to update deposit coin again.
		if _, ok := newMembers[ca.Info.DID]; ok {
			continue
		}
		// if canceled enough blocks, no need to update deposit coin again.
		if ca.State == Canceled && height-ca.CancelHeight >= c.Params.CRDepositLockupBlocks {
			continue
		}
		// if CR deposit coin is returned, no need to update deposit coin again.
		if ca.State == Returned {
			continue
		}
		oriDepositAmount := c.state.DepositInfo[ca.Info.CID].DepositAmount
		c.lastHistory.Append(height, func() {
			c.state.DepositInfo[ca.Info.CID].DepositAmount -= MinDepositAmount
		}, func() {
			c.state.DepositInfo[ca.Info.CID].DepositAmount = oriDepositAmount
		})
	}
	c.lastHistory.Append(height, func() {
		c.state.Candidates = make(map[common.Uint168]*Candidate)
		c.state.HistoryCandidates[currentSession] = newHistoryCandidates
	}, func() {
		c.state.Candidates = oriCandidate
		delete(c.state.HistoryCandidates, currentSession)
	})
}

func (c *Committee) generateMember(candidate *Candidate) *CRMember {
	return &CRMember{
		Info:                  candidate.Info,
		ImpeachmentVotes:      0,
		DepositHash:           candidate.DepositHash,
		MemberState:           MemberElected,
		ActivateRequestHeight: math.MaxUint32,
	}
}

func (c *Committee) getMemberPenalty(height uint32, member *CRMember, impeached bool) common.Fixed64 {
	// Calculate penalty by election block count.
	var electionCount uint32
	if impeached {
		electionCount = height - c.LastCommitteeHeight - member.PenaltyBlockCount
	} else {
		electionCount = c.Params.CRDutyPeriod - member.PenaltyBlockCount
	}
	if member.MemberState == MemberInactive {
		electionCount -= 1
	}
	var electionRate float64
	electionRate = float64(electionCount) / float64(c.Params.CRDutyPeriod)
	// Calculate penalty by vote proposal count.
	var voteCount int
	for _, v := range c.manager.ProposalSession[c.state.CurrentSession] {
		proposal := c.manager.Proposals[v]
		for did, _ := range proposal.CRVotes {
			if member.Info.DID.IsEqual(did) {
				voteCount++
				break
			}
		}
	}
	proposalsCount := len(c.manager.ProposalSession[c.state.CurrentSession])
	voteRate := float64(1.0)
	if proposalsCount != 0 {
		voteRate = float64(voteCount) / float64(proposalsCount)
	}

	// Calculate the final penalty.
	penalty := c.state.DepositInfo[member.Info.CID].Penalty
	currentPenalty := common.Fixed64(float64(MinDepositAmount) * (1 - electionRate*voteRate))
	finalPenalty := penalty + currentPenalty

	log.Infof("Height %d member %s, not election and not vote proposal"+
		" penalty: %s, old penalty: %s, final penalty: %s",
		height, member.Info.NickName, currentPenalty, penalty, finalPenalty)
	log.Info("electionRate:", electionRate, "voteRate:", voteRate,
		"electionCount:", electionCount, "PenaltyBlockCount:", member.PenaltyBlockCount,
		"dutyPeriod:", c.Params.CRDutyPeriod, "voteCount:", voteCount,
		"proposalsCount:", proposalsCount)

	return finalPenalty
}

func (c *Committee) generateCandidate(height uint32, member *CRMember) *Candidate {
	return &Candidate{
		Info:         member.Info,
		State:        Canceled,
		CancelHeight: height,
		DepositHash:  member.DepositHash,
	}
}

func (c *Committee) getActiveAndExistDIDCRCandidatesDesc() []*Candidate {
	emptyDID := common.Uint168{}
	candidates := c.state.getCandidateFromMap(c.state.Candidates,
		func(candidate *Candidate) bool {
			if candidate.Info.DID.IsEqual(emptyDID) {
				return false
			}
			return candidate.State == Active
		})

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Votes == candidates[j].Votes {
			iCRInfo := candidates[i].Info
			jCRInfo := candidates[j].Info
			return iCRInfo.GetCodeHash().Compare(jCRInfo.GetCodeHash()) < 0
		}
		return candidates[i].Votes > candidates[j].Votes
	})
	return candidates
}

func (c *Committee) GetCandidate(cid common.Uint168) *Candidate {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.state.GetCandidate(cid)
}

func (c *Committee) GetCandidates(state CandidateState) []*Candidate {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.state.GetCandidates(state)
}

func (c *Committee) ExistCandidateByNickname(nickname string) bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.state.ExistCandidateByNickname(nickname)
}

func (c *Committee) ExistCandidateByDepositHash(hash common.Uint168) bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.state.existCandidateByDepositHash(hash)
}

func (c *Committee) GetPenalty(cid common.Uint168) common.Fixed64 {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.state.getPenalty(cid)
}

func (c *Committee) ExistProposal(hash common.Uint256) bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.manager.existProposal(hash)
}

func (c *Committee) GetProposal(hash common.Uint256) *ProposalState {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.manager.getProposal(hash)
}

func (c *Committee) AvailableWithdrawalAmount(hash common.Uint256) common.Fixed64 {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.manager.availableWithdrawalAmount(hash)
}

func (c *Committee) IsProposalFull(did common.Uint168) bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.manager.isProposalFull(did)
}

func (c *Committee) ExistDraft(hash common.Uint256) bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.manager.existDraft(hash)
}

func (c *Committee) Exist(cid common.Uint168) bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	return c.state.exist(cid)
}

func (c *Committee) GetAllCandidates() []*Candidate {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.state.GetAllCandidates()
}

func (c *Committee) GetAllProposals() ProposalsMap {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.manager.getAllProposals()
}

func (c *Committee) GetProposals(status ProposalStatus) ProposalsMap {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.manager.getProposals(status)
}

func (c *Committee) GetRegisteredSideChainByHeight(height uint32) map[common.Uint256]payload.SideChainInfo {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.manager.getRegisteredSideChainByHeight(height)
}

func (c *Committee) GetAllRegisteredSideChain() map[uint32]map[common.Uint256]payload.SideChainInfo {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.manager.getAllRegisteredSideChain()
}

func (c *Committee) GetProposalByDraftHash(draftHash common.Uint256) *ProposalState {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.manager.getProposalByDraftHash(draftHash)
}

func (c *Committee) GetRealWithdrawTransactions() map[common.Uint256]common2.OutputInfo {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	return c.manager.WithdrawableTxInfo
}

// GetCandidateByID returns candidate with specified cid or did, it will return
// nil if not found.
func (c *Committee) GetCandidateByID(id common.Uint168) *Candidate {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.state.getCandidateByID(id)
}

// GetCandidateByCID returns candidate with specified cid, it will return nil
// if not found.
func (c *Committee) GetCandidateByCID(cid common.Uint168) *Candidate {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.state.GetCandidate(cid)
}

// GetCandidateByPublicKey returns candidate with specified public key, it will
// return nil if not found.
func (c *Committee) GetCandidateByPublicKey(publicKey string) *Candidate {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	pubkey, err := common.HexStringToBytes(publicKey)
	if err != nil {
		return nil
	}
	return c.state.getCandidateByPublicKey(pubkey)
}

type CommitteeFuncsConfig struct {
	GetTxReference func(tx interfaces.Transaction) (
		map[*common2.Input]common2.Output, error)
	GetHeight                        func() uint32
	CreateCRAppropriationTransaction func() (interfaces.Transaction, common.Fixed64, error)
	CreateCRAssetsRectifyTransaction func() (interfaces.Transaction, error)
	CreateCRRealWithdrawTransaction  func(withdrawTransactionHashes []common.Uint256,
		outpus []*common2.OutputInfo) (interfaces.Transaction, error)
	IsCurrent          func() bool
	Broadcast          func(msg p2p.Message)
	AppendToTxpool     func(transaction interfaces.Transaction) elaerr.ELAError
	GetUTXO            func(programHash *common.Uint168) ([]*common2.UTXO, error)
	GetCurrentArbiters func() [][]byte
}

func (c *Committee) RegisterFuncitons(cfg *CommitteeFuncsConfig) {
	c.createCRCAppropriationTx = cfg.CreateCRAppropriationTransaction
	c.createCRAssetsRectifyTransaction = cfg.CreateCRAssetsRectifyTransaction
	c.createCRRealWithdrawTransaction = cfg.CreateCRRealWithdrawTransaction
	c.isCurrent = cfg.IsCurrent
	c.broadcast = cfg.Broadcast
	c.appendToTxpool = cfg.AppendToTxpool
	c.state.RegisterFunctions(&FunctionsConfig{
		GetHistoryMember: c.getHistoryMember,
		GetTxReference:   cfg.GetTxReference,
	})
	c.getUTXO = cfg.GetUTXO
	c.GetHeight = cfg.GetHeight
	c.getCurrentArbiters = cfg.GetCurrentArbiters
}

func (c *Committee) TryUpdateCRMemberInactivity(did common.Uint168,
	needReset bool, height uint32) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	crMember := c.getMember(did)
	if crMember == nil {
		log.Error("tryUpdateCRMemberInactivity did %+v not exist", did.String())
		return
	}

	if height < c.Params.ChangeCommitteeNewCRHeight {
		if needReset {
			crMember.InactiveCountingHeight = 0
			return
		}

		if crMember.InactiveCountingHeight == 0 {
			crMember.InactiveCountingHeight = height
		}

		if height-crMember.InactiveCountingHeight >= c.Params.MaxInactiveRounds {
			crMember.MemberState = MemberInactive
			log.Info("at Height", height, crMember.Info.NickName,
				"changed to inactive", "InactiveCountingHeight:", crMember.InactiveCountingHeight,
				"MaxInactiveRounds:", c.Params.MaxInactiveRounds)
			crMember.InactiveCountingHeight = 0
		}
	} else {
		if needReset {
			crMember.InactiveCount = 0
			return
		}

		crMember.InactiveCount++
		if crMember.InactiveCount >= c.Params.MaxInactiveRounds &&
			crMember.MemberState == MemberElected {
			log.Info("at Height", height, crMember.Info.NickName,
				"changed to inactive", "InactiveCount:", crMember.InactiveCount,
				"MaxInactiveRounds:", c.Params.MaxInactiveRounds)
			crMember.MemberState = MemberInactive
			if height >= c.Params.ChangeCommitteeNewCRHeight {
				c.state.UpdateCRInactivePenalty(crMember.Info.CID, height)
			}
			crMember.InactiveCount = 0
		}
	}
}

func (c *Committee) TryRevertCRMemberInactivity(did common.Uint168,
	oriState MemberState, oriInactiveCount uint32, height uint32) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	crMember := c.getMember(did)
	if crMember == nil {
		log.Error("tryRevertCRMemberInactivity did %+v not exist", did.String())
		return
	}

	if height < c.Params.ChangeCommitteeNewCRHeight {
		crMember.MemberState = oriState
		crMember.InactiveCountingHeight = oriInactiveCount
	} else {
		if oriInactiveCount < c.Params.MaxInactiveRounds &&
			crMember.MemberState == MemberInactive {
			c.state.RevertUpdateCRInactivePenalty(crMember.Info.CID, height)
		}

		crMember.MemberState = oriState
		crMember.InactiveCount = oriInactiveCount
	}

}

func (c *Committee) TryUpdateCRMemberIllegal(did common.Uint168, height uint32, illegalPenalty common.Fixed64) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	crMember := c.getMember(did)
	if crMember == nil {
		log.Errorf("TryUpdateCRMemberIllegal did %+v not exist", did.String())
		return
	}
	if height >= c.Params.ChangeCommitteeNewCRHeight {
		c.state.UpdateCRIllegalPenalty(crMember.Info.CID, height, illegalPenalty)
	}
	crMember.MemberState = MemberIllegal

}

func (c *Committee) TryRevertCRMemberIllegal(did common.Uint168, oriState MemberState, height uint32, illegalPenalty common.Fixed64) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	crMember := c.getMember(did)
	if crMember == nil {
		log.Errorf("TryRevertCRMemberIllegal did %+v not exist", did.String())
		return
	}
	crMember.MemberState = oriState
	if height >= c.Params.ChangeCommitteeNewCRHeight {
		c.state.RevertUpdateCRIllegalPenalty(crMember.Info.CID, height, illegalPenalty)
	}
}

func (c *Committee) Snapshot() *CommitteeKeyFrame {
	keyFrame := &CommitteeKeyFrame{
		KeyFrame:         c.KeyFrame.Snapshot(),
		StateKeyFrame:    c.state.StateKeyFrame.Snapshot(),
		ProposalKeyFrame: c.manager.ProposalKeyFrame.Snapshot(),
	}

	return keyFrame
}

func NewCommittee(params *config.Params) *Committee {
	committee := &Committee{
		state:                NewState(params),
		Params:               params,
		KeyFrame:             *NewKeyFrame(),
		manager:              NewProposalManager(params),
		firstHistory:         utils.NewHistory(maxHistoryCapacity),
		lastHistory:          utils.NewHistory(maxHistoryCapacity),
		appropriationHistory: utils.NewHistory(maxHistoryCapacity),
	}
	committee.manager.InitSecretaryGeneralPublicKey(params.SecretaryGeneral)
	committee.state.SetManager(committee.manager)
	params.CkpManager.Register(NewCheckpoint(committee))
	return committee
}
