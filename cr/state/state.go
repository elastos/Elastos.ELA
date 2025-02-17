// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"encoding/hex"
	"errors"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/utils"
)

const (
	// MinDepositAmount is the minimum deposit as a producer.
	MinDepositAmount = 5000 * 100000000

	// MinDPoSV2DepositAmount is the minimum deposit as a DPoS 2.0 producer.
	MinDPoSV2DepositAmount = 2000 * 100000000

	// maxHistoryCapacity indicates the maximum capacity of change History.
	maxHistoryCapacity = 720

	// ActivateDuration is about how long we should activate from pending or
	// inactive State.
	ActivateDuration = 6
)

// State hold all CR candidates related information, and process block by block
// to update Votes and any other changes about candidates.
type State struct {
	StateKeyFrame
	manager *ProposalManager

	getHistoryMember func(code []byte) []*CRMember
	GetTxReference   func(tx interfaces.Transaction) (
		map[*common2.Input]common2.Output, error)

	params  *config.Configuration
	History *utils.History
}

// SetManager set current proposal manager that holds State of proposals.
func (s *State) SetManager(manager *ProposalManager) {
	s.manager = manager
}

type FunctionsConfig struct {
	GetHistoryMember func(code []byte) []*CRMember
	GetTxReference   func(tx interfaces.Transaction) (
		map[*common2.Input]common2.Output, error)
}

func (s *State) UpdateCRInactivePenalty(cid common.Uint168, height uint32) {
	depositInfo, ok := s.DepositInfo[cid]
	if !ok {
		return
	}
	depositInfo.Penalty += s.params.DPoSConfiguration.InactivePenalty
}

func (s *State) RevertUpdateCRInactivePenalty(cid common.Uint168, height uint32) {
	depositInfo, ok := s.DepositInfo[cid]
	if !ok {
		return
	}
	var penalty = s.params.DPoSConfiguration.InactivePenalty
	if depositInfo.Penalty < penalty {
		depositInfo.Penalty = common.Fixed64(0)
	} else {
		depositInfo.Penalty -= penalty
	}
}

func (s *State) UpdateCRIllegalPenalty(cid common.Uint168, illegalPenalty common.Fixed64) {
	depositInfo, ok := s.DepositInfo[cid]
	if !ok {
		return
	}

	depositInfo.Penalty += illegalPenalty
}

func (s *State) RevertUpdateCRIllegalPenalty(cid common.Uint168, illegalPenalty common.Fixed64) {
	depositInfo, ok := s.DepositInfo[cid]
	if !ok {
		return
	}

	depositInfo.Penalty -= illegalPenalty
}

// RegisterFunctions set the tryStartVotingPeriod and processImpeachment function
// to change member State.
func (s *State) RegisterFunctions(cfg *FunctionsConfig) {
	s.getHistoryMember = cfg.GetHistoryMember
	s.GetTxReference = cfg.GetTxReference
}

// GetAllCandidates returns all candidates holding within State.
func (s *State) GetAllCandidates() []*Candidate {
	return s.getCandidateFromMap(s.Candidates, nil)
}

func (s *State) exist(cid common.Uint168) bool {
	_, ok := s.DepositInfo[cid]
	return ok
}

// GetTotalAmount returns total amount with specified candidate or member cid.
func (s *State) GetTotalAmount(cid common.Uint168) common.Fixed64 {
	return s.DepositInfo[cid].TotalAmount
}

// GetDepositAmount returns deposit amount with specified candidate or member cid.
func (s *State) GetDepositAmount(cid common.Uint168) common.Fixed64 {
	return s.DepositInfo[cid].DepositAmount
}

// getPenalty returns penalty with specified candidate or member cid.
func (s *State) getPenalty(cid common.Uint168) common.Fixed64 {
	return s.DepositInfo[cid].Penalty
}

// getAvailableDepositAmount returns available deposit amount with specified
// candidate or member cid.
func (s *State) getAvailableDepositAmount(cid common.Uint168) common.Fixed64 {
	depositInfo, ok := s.DepositInfo[cid]
	if !ok {
		return 0
	}
	return depositInfo.TotalAmount - depositInfo.DepositAmount -
		depositInfo.Penalty
}

// getDepositInfoByCID returns available Penalty DepositAmount and TotalAmount with
// specified cid.
func (s *State) getDepositInfoByCID(
	cid common.Uint168) (common.Fixed64, common.Fixed64, common.Fixed64, common.Fixed64, error) {
	depositInfo, ok := s.DepositInfo[cid]
	if !ok {
		return 0, 0, 0, 0, errors.New("deposit information does not exist")
	}
	return depositInfo.TotalAmount - depositInfo.DepositAmount - depositInfo.Penalty,
		depositInfo.Penalty, depositInfo.DepositAmount, depositInfo.TotalAmount, nil
}

// getDepositInfoByPublicKey return available Penalty DepositAmount and TotalAmount
// by the given public key.
func (s *State) getDepositInfoByPublicKey(
	publicKey []byte) (common.Fixed64, common.Fixed64, common.Fixed64, common.Fixed64, error) {
	cid, err := getCIDByPublicKey(publicKey)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	depositInfo, ok := s.DepositInfo[*cid]
	if !ok {
		return 0, 0, 0, 0, errors.New("CID does not exist")
	}
	return depositInfo.TotalAmount - depositInfo.DepositAmount -
		depositInfo.Penalty, depositInfo.Penalty, depositInfo.DepositAmount, depositInfo.TotalAmount, nil
}

// ExistCandidate judges if there is a candidate with specified program code.
func (s *State) ExistCandidate(programCode []byte) bool {
	_, ok := s.CodeCIDMap[common.BytesToHexString(programCode)]
	return ok
}

// ExistCandidate judges if there is a candidate with specified program code.
func (s *State) existCandidate(programCode []byte) bool {
	_, ok := s.CodeCIDMap[common.BytesToHexString(programCode)]
	return ok
}

// ExistCandidateByCID judges if there is a candidate with specified cid.
func (s *State) ExistCandidateByCID(cid common.Uint168) (ok bool) {
	if _, ok = s.Candidates[cid]; ok {
		return
	}
	return
}

// existCandidateByDepositHash judges if there is a candidate with deposit hash.
func (s *State) existCandidateByDepositHash(hash common.Uint168) bool {
	_, ok := s.DepositHashCIDMap[hash]
	return ok
}

// ExistCandidateByNickname judges if there is a candidate with specified
// nickname.
func (s *State) ExistCandidateByNickname(nickname string) bool {
	_, ok := s.Nicknames[nickname]
	return ok
}

// IsCRTransaction returns if a transaction will change the CR and Votes State.
func (s *State) IsCRTransaction(tx interfaces.Transaction) bool {
	switch tx.TxType() {
	// Transactions will changes the producers State.
	case common2.RegisterCR, common2.UpdateCR,
		common2.UnregisterCR, common2.ReturnCRDepositCoin:
		return true

	// Transactions will change the producer Votes State.
	case common2.TransferAsset:
		if tx.Version() >= common2.TxVersion09 {
			for _, output := range tx.Outputs() {
				if output.Type != common2.OTVote {
					continue
				}
				p, _ := output.Payload.(*outputpayload.VoteOutput)
				if p.Version < outputpayload.VoteProducerAndCRVersion {
					continue
				}
				for _, content := range p.Contents {
					if content.VoteType == outputpayload.CRC {
						return true
					}
				}
			}
		}
	}

	// Cancel Votes.
	for _, input := range tx.Inputs() {
		_, ok := s.Votes[input.ReferKey()]
		if ok {
			return true
		}
	}

	return false
}

// rollbackTo restores the database State to the given Height, if no enough
// History to rollback to return error.
func (s *State) rollbackTo(height uint32) error {
	return s.History.RollbackTo(height)
}

// registerCR handles the register CR transaction.
func (s *State) registerCR(tx interfaces.Transaction, height uint32) {
	info := tx.Payload().(*payload.CRInfo)
	nickname := info.NickName
	var code string
	if tx.PayloadVersion() == payload.CRInfoSchnorrVersion ||
		tx.PayloadVersion() == payload.CRInfoMultiSignVersion {
		code = common.BytesToHexString(tx.Programs()[0].Code)
		info.Code = tx.Programs()[0].Code
	} else {
		code = common.BytesToHexString(info.Code)
	}
	codeBytes, _ := hex.DecodeString(code)
	depositContract, _ := contract.CreateDepositContractByCode(codeBytes)
	candidate := Candidate{
		Info:           *info,
		RegisterHeight: height,
		Votes:          0,
		State:          Pending,
		DepositHash:    *depositContract.ToProgramHash(),
	}

	amount := common.Fixed64(0)
	for i, output := range tx.Outputs() {
		if output.ProgramHash.IsEqual(candidate.DepositHash) {
			amount += output.Value
			op := common2.NewOutPoint(tx.Hash(), uint16(i))
			s.DepositOutputs[op.ReferKey()] = output.Value
		}
	}

	var firstTimeRegister bool
	if _, ok := s.DepositInfo[info.CID]; !ok {
		firstTimeRegister = true
	}
	s.History.Append(height, func() {
		if firstTimeRegister {
			s.DepositInfo[info.CID] = &DepositInfo{}
			s.CodeCIDMap[code] = info.CID
			s.DepositHashCIDMap[candidate.DepositHash] = info.CID
		}
		s.Nicknames[nickname] = struct{}{}
		s.Candidates[info.CID] = &candidate
		s.DepositInfo[info.CID].DepositAmount += MinDepositAmount
		s.DepositInfo[info.CID].TotalAmount += amount
	}, func() {
		delete(s.Candidates, info.CID)
		delete(s.Nicknames, nickname)
		s.DepositInfo[info.CID].DepositAmount -= MinDepositAmount
		s.DepositInfo[info.CID].TotalAmount -= amount
		if firstTimeRegister {
			delete(s.DepositInfo, info.CID)
			delete(s.CodeCIDMap, code)
			delete(s.DepositHashCIDMap, candidate.DepositHash)
		}
	})
}

// updateCR handles the update CR transaction.
func (s *State) updateCR(info *payload.CRInfo, height uint32) {
	candidate := s.GetCandidate(info.CID)
	crInfo := candidate.Info
	s.History.Append(height, func() {
		s.updateCandidateInfo(&crInfo, info)
	}, func() {
		s.updateCandidateInfo(info, &crInfo)
	})
}

// unregisterCR handles the cancel producer transaction.
func (s *State) unregisterCR(info *payload.UnregisterCR, height uint32) {
	candidate := s.GetCandidate(info.CID)
	if candidate == nil {
		return
	}
	oriState := candidate.State
	s.History.Append(height, func() {
		candidate.CancelHeight = height
		candidate.State = Canceled
		delete(s.Nicknames, candidate.Info.NickName)
	}, func() {
		candidate.CancelHeight = 0
		candidate.State = oriState
		s.Nicknames[candidate.Info.NickName] = struct{}{}
	})
}

// updateCandidateInfo updates the candidate's Info with value compare,
// any change will be updated.
func (s *State) updateCandidateInfo(origin *payload.CRInfo, update *payload.CRInfo) {
	candidate := s.GetCandidate(origin.CID)

	// compare and update node nickname.
	if origin.NickName != update.NickName {
		delete(s.Nicknames, origin.NickName)
		s.Nicknames[update.NickName] = struct{}{}
	}

	candidate.Info = *update
}

// processDeposit takes a transaction output with deposit program hash.
func (s *State) processDeposit(tx interfaces.Transaction, height uint32) {
	for i, output := range tx.Outputs() {
		if contract.GetPrefixType(output.ProgramHash) == contract.PrefixDeposit {
			if s.addCRCRelatedAssert(output, height) {
				op := common2.NewOutPoint(tx.Hash(), uint16(i))
				s.DepositOutputs[op.ReferKey()] = output.Value
			}
		}
	}
}

// returnDeposit change producer State to ReturnedDeposit
func (s *State) returnDeposit(tx interfaces.Transaction, height uint32) {
	var inputValue common.Fixed64
	for _, input := range tx.Inputs() {
		inputValue += s.DepositOutputs[input.ReferKey()]
	}

	returnCandidateAction := func(candidate *Candidate, originState CandidateState) {
		s.History.Append(height, func() {
			candidate.State = Returned
		}, func() {
			candidate.State = originState
		})
	}

	returnMemberAction := func(member *CRMember, originState MemberState) {
		s.History.Append(height, func() {
			member.MemberState = MemberReturned
		}, func() {
			member.MemberState = originState
		})
	}

	updateAmountAction := func(cid common.Uint168) {
		s.History.Append(height, func() {
			s.DepositInfo[cid].TotalAmount -= inputValue
		}, func() {
			s.DepositInfo[cid].TotalAmount += inputValue
		})
	}

	for _, program := range tx.Programs() {
		cid, _ := GetCIDByCode(program.Code)

		if candidate := s.GetCandidate(*cid); candidate != nil {
			var changeValue common.Fixed64
			for _, o := range tx.Outputs() {
				if candidate.DepositHash.IsEqual(o.ProgramHash) {
					changeValue += o.Value
				}
			}
			balance := s.DepositInfo[*cid].TotalAmount - inputValue + changeValue -
				s.DepositInfo[*cid].Penalty -
				s.DepositInfo[*cid].DepositAmount

			if candidate.State == Canceled &&
				height-candidate.CancelHeight > s.params.CRConfiguration.DepositLockupBlocks &&
				balance <= s.params.MinTransactionFee {
				returnCandidateAction(candidate, candidate.State)
			}
		}
		if candidates := s.getHistoryCandidate(*cid); len(candidates) != 0 {
			for _, c := range candidates {
				if c.State != Returned {
					returnCandidateAction(c, c.State)
				}
			}
		}
		if members := s.getHistoryMember(program.Code); len(members) != 0 {
			for _, m := range members {
				returnMemberAction(m, m.MemberState)
			}
		}

		updateAmountAction(*cid)
	}
}

// addCRCRelatedAssert will plus deposit amount for CRC referenced in
// program hash of transaction output.
func (s *State) addCRCRelatedAssert(output *common2.Output, height uint32) bool {
	if cid, ok := s.getCIDByDepositHash(output.ProgramHash); ok {
		s.History.Append(height, func() {
			s.DepositInfo[cid].TotalAmount += output.Value
		}, func() {
			s.DepositInfo[cid].TotalAmount -= output.Value
		})
		return true
	}
	return false
}

// getCIDByDepositHash will try to get cid of candidate or member with specified
// program hash.
func (s *State) getCIDByDepositHash(hash common.Uint168) (common.Uint168, bool) {
	cid, ok := s.DepositHashCIDMap[hash]
	return cid, ok
}

// processVoteCRC record candidate Votes.
func (s *State) processVoteCRC(height uint32, candidate []byte, votes common.Fixed64) {
	cid, err := common.Uint168FromBytes(candidate)
	if err != nil {
		return
	}
	c := s.GetCandidate(*cid)
	if candidate == nil {
		return
	}
	s.History.Append(height, func() {
		c.Votes += votes
	}, func() {
		c.Votes -= votes
	})
}

// processVoteCRC record candidate Votes.
func (s *State) processCancelVoteCRC(height uint32, candidate []byte, votes common.Fixed64) {
	cid, err := common.Uint168FromBytes(candidate)
	if err != nil {
		return
	}
	c := s.GetCandidate(*cid)
	if candidate == nil {
		return
	}
	s.History.Append(height, func() {
		c.Votes -= votes
	}, func() {
		c.Votes += votes
	})
}

// processVoteCRCProposal record proposal reject Votes.
func (s *State) processCancelVoteCRCProposal(height uint32,
	candidate []byte, votes common.Fixed64) {
	proposalHash, err := common.Uint256FromBytes(candidate)
	if err != nil {
		return
	}
	proposalState := s.manager.getProposal(*proposalHash)
	if proposalState == nil || proposalState.Status != CRAgreed {
		return
	}
	s.History.Append(height, func() {
		proposalState.VotersRejectAmount -= votes
	}, func() {
		proposalState.VotersRejectAmount += votes
	})
}

// processVoteCRCProposal record proposal reject Votes.
func (s *State) processVoteCRCProposal(height uint32,
	candidate []byte, votes common.Fixed64) {
	proposalHash, err := common.Uint256FromBytes(candidate)
	if err != nil {
		return
	}
	proposalState := s.manager.getProposal(*proposalHash)
	if proposalState == nil || proposalState.Status != CRAgreed {
		return
	}
	s.History.Append(height, func() {
		proposalState.VotersRejectAmount += votes
	}, func() {
		proposalState.VotersRejectAmount -= votes
	})
}

// GetCandidate returns candidate with specified cid, it will return nil
// nil if not found.
func (s *State) GetCandidate(cid common.Uint168) *Candidate {
	if c, ok := s.Candidates[cid]; ok {
		return c
	}
	return nil
}

// getExistCIDByPublicKey return existing candidate by the given CID or DID.
func (s *State) getCandidateByID(id common.Uint168) *Candidate {
	for k, v := range s.CodeCIDMap {
		if v.IsEqual(id) {
			return s.GetCandidate(v)
		}
		code, err := common.HexStringToBytes(k)
		if err != nil {
			return nil
		}
		did, err := GetDIDByCode(code)
		if err != nil {
			return nil
		}
		if did.IsEqual(id) {
			return s.GetCandidate(v)
		}
	}
	return nil
}

// getExistCIDByID return existing CID by the given CID or DID.
func (s *State) getExistCIDByID(id common.Uint168) (*common.Uint168, bool) {
	for k, v := range s.CodeCIDMap {
		cid := v
		if cid.IsEqual(id) {
			return &cid, true
		}
		code, err := common.HexStringToBytes(k)
		if err != nil {
			return nil, false
		}
		did, err := GetDIDByCode(code)
		if err != nil {
			return nil, false
		}
		if did.IsEqual(id) {
			return &cid, true
		}
	}
	return nil, false
}

// getExistDIDByID return existing DID by the given CID or DID.
func (s *State) getExistDIDByID(id common.Uint168) (*common.Uint168, bool) {
	for k, v := range s.CodeCIDMap {
		code, err := common.HexStringToBytes(k)
		if err != nil {
			return nil, false
		}
		did, err := GetDIDByCode(code)
		if err != nil {
			return nil, false
		}
		if did.IsEqual(id) || v.IsEqual(id) {
			return did, true
		}
	}
	return nil, false
}

// getCandidateByPublicKey return existing candidate by the given public key.
func (s *State) getCandidateByPublicKey(publicKey []byte) *Candidate {
	cid, err := getCIDByPublicKey(publicKey)
	if err != nil {
		return nil
	}
	return s.GetCandidate(*cid)
}

func (s *State) getHistoryCandidate(cid common.Uint168) []*Candidate {
	candidates := make([]*Candidate, 0)
	for _, v := range s.HistoryCandidates {
		if c, ok := v[cid]; ok {
			candidates = append(candidates, c)
		}
	}
	return candidates
}

func (s *State) getCandidateByCode(programCode []byte) *Candidate {
	cid, ok := s.getCIDByCode(programCode)
	if !ok {
		return nil
	}
	return s.GetCandidate(cid)
}

func (s *State) getCIDByCode(programCode []byte) (cid common.Uint168,
	exist bool) {
	codeStr := common.BytesToHexString(programCode)
	cid, exist = s.CodeCIDMap[codeStr]
	return
}

func (s *State) GetCodeByCid(cid common.Uint168) (code string,
	exist bool) {
	for k, v := range s.CodeCIDMap {
		if v == cid {
			return k, true
		}
	}
	return "", false
}

// GetCandidates returns candidates with specified candidate State.
func (s *State) GetCandidates(state CandidateState) []*Candidate {
	switch state {
	case Pending, Active, Canceled, Returned:
		return s.getCandidateFromMap(s.Candidates,
			func(candidate *Candidate) bool {
				return candidate.State == state
			})
	default:
		return []*Candidate{}
	}
}

func (s *State) getCandidateFromMap(cmap map[common.Uint168]*Candidate,
	filter func(*Candidate) bool) []*Candidate {
	result := make([]*Candidate, 0, len(cmap))
	for _, v := range cmap {
		if filter != nil && !filter(v) {
			continue
		}
		result = append(result, v)
	}
	return result
}

func NewState(chainParams *config.Configuration) *State {
	return &State{
		StateKeyFrame: *NewStateKeyFrame(),
		params:        chainParams,
		History:       utils.NewHistory(maxHistoryCapacity),
	}
}
