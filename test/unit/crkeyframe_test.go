// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package unit

import (
	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"testing"
	"time"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	transaction2 "github.com/elastos/Elastos.ELA/core/transaction"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/cr/state"

	"github.com/stretchr/testify/assert"
)

func init() {
	functions.GetTransactionByTxType = transaction2.GetTransaction
	functions.GetTransactionByBytes = transaction2.GetTransactionByBytes
	functions.CreateTransaction = transaction2.CreateTransaction
	functions.GetTransactionParameters = transaction2.GetTransactionparameters
	config.DefaultParams = config.GetDefaultParams()
}

func TestKeyFrame_Deserialize(t *testing.T) {
	frame := randomKeyFrame(5, rand.Uint32())

	buf := new(bytes.Buffer)
	frame.Serialize(buf)

	frame2 := &state.KeyFrame{}
	frame2.Deserialize(buf)

	assert.True(t, keyframeEqual(frame, frame2))
}

func TestKeyFrame_Snapshot(t *testing.T) {
	frame := randomKeyFrame(5, rand.Uint32())
	frame2 := frame.Snapshot()
	assert.True(t, keyframeEqual(frame, frame2))
}

func TestStateKeyFrame_Deserialize(t *testing.T) {
	frame := randomStateKeyFrame(5, true)

	buf := new(bytes.Buffer)
	frame.Serialize(buf)

	frame2 := &state.StateKeyFrame{}
	frame2.Deserialize(buf)

	assert.True(t, stateKeyframeEqual(frame, frame2))
}

func TestStateKeyFrame_Snapshot(t *testing.T) {
	frame := randomStateKeyFrame(5, true)
	frame2 := frame.Snapshot()
	assert.True(t, stateKeyframeEqual(frame, frame2))
}

func TestProposalKeyFrame_Deserialize(t *testing.T) {
	frame := randomProposalKeyframe()

	buf := new(bytes.Buffer)
	frame.Serialize(buf)

	frame2 := &state.ProposalKeyFrame{}
	frame2.Deserialize(buf)

	assert.True(t, proposalKeyFrameEqual(frame, frame2))
}

func TestProposalKeyFrame_Snapshot(t *testing.T) {
	frame := randomProposalKeyframe()
	frame2 := frame.Snapshot()
	assert.True(t, proposalKeyFrameEqual(frame, frame2))
}

func TestCheckPoint_Deserialize(t *testing.T) {
	originCheckPoint := generateCheckPoint(rand.Uint32())

	buf := new(bytes.Buffer)
	assert.NoError(t, originCheckPoint.Serialize(buf))

	cmpData := &state.Checkpoint{}
	assert.NoError(t, cmpData.Deserialize(buf))

	assert.True(t, checkPointsEqual(originCheckPoint, cmpData))
}

func generateCheckPoint(height uint32) *state.Checkpoint {
	result := &state.Checkpoint{
		KeyFrame:         *randomKeyFrame(5, rand.Uint32()),
		StateKeyFrame:    *randomStateKeyFrame(5, true),
		ProposalKeyFrame: *randomProposalKeyframe(),
		Height:           height,
	}
	return result
}

func checkPointsEqual(first *state.Checkpoint, second *state.Checkpoint) bool {
	if first.Height != second.Height {
		return false
	}
	return keyframeEqual(&first.KeyFrame, &second.KeyFrame) &&
		stateKeyframeEqual(&first.StateKeyFrame, &second.StateKeyFrame) &&
		proposalKeyFrameEqual(&first.ProposalKeyFrame, &second.ProposalKeyFrame)
}

func stateKeyframeEqual(first *state.StateKeyFrame, second *state.StateKeyFrame) bool {
	if first.CurrentSession != second.CurrentSession ||
		len(first.Candidates) != len(second.Candidates) ||
		len(first.HistoryCandidates) != len(second.HistoryCandidates) ||
		len(first.DepositHashCIDMap) != len(second.DepositHashCIDMap) ||
		len(first.CodeCIDMap) != len(second.CodeCIDMap) ||
		len(first.DepositOutputs) != len(second.DepositOutputs) ||
		len(first.CRCFoundationOutputs) != len(second.CRCFoundationOutputs) ||
		len(first.CRCCommitteeOutputs) != len(second.CRCCommitteeOutputs) ||
		len(first.Nicknames) != len(second.Nicknames) ||
		len(first.Votes) != len(second.Votes) ||
		len(first.DepositInfo) != len(second.DepositInfo) {
		return false
	}

	return candidatesMapEqual(first.Candidates, second.Candidates) &&
		candidatesHistoryMapEqual(first.HistoryCandidates, second.HistoryCandidates) &&
		depositHashCIDMapEqual(first.DepositHashCIDMap, second.DepositHashCIDMap) &&
		codeCIDMapEqual(first.CodeCIDMap, second.CodeCIDMap) &&
		amountMapEqual(first.DepositOutputs, second.DepositOutputs) &&
		amountMapEqual(first.CRCFoundationOutputs, second.CRCFoundationOutputs) &&
		amountMapEqual(first.CRCCommitteeOutputs, second.CRCCommitteeOutputs) &&
		stringMapEqual(first.Nicknames, second.Nicknames) &&
		stringMapEqual(first.Votes, second.Votes) &&
		depositInfoMapEqual(first.DepositInfo, second.DepositInfo)

}

func stringMapEqual(first map[string]struct{}, second map[string]struct{}) bool {
	for k := range first {
		if _, ok := second[k]; !ok {
			return false
		}
	}
	return true
}

func codeCIDMapEqual(first map[string]common.Uint168,
	second map[string]common.Uint168) bool {
	if len(first) != len(second) {
		return false
	}
	for k, v := range first {
		v2, ok := second[k]
		if !ok {
			return false
		}

		if !v.IsEqual(v2) {
			return false
		}
	}
	return true
}

func depositHashCIDMapEqual(first map[common.Uint168]common.Uint168,
	second map[common.Uint168]common.Uint168) bool {
	if len(first) != len(second) {
		return false
	}
	for k, v := range first {
		v2, ok := second[k]
		if !ok {
			return false
		}

		if !v.IsEqual(v2) {
			return false
		}
	}
	return true
}

func amountMapEqual(first map[string]common.Fixed64,
	second map[string]common.Fixed64) bool {
	if len(first) != len(second) {
		return false
	}
	for k, v := range first {
		v2, ok := second[k]
		if !ok {
			return false
		}

		if v != v2 {
			return false
		}
	}
	return true
}

func candidatesMapEqual(first map[common.Uint168]*state.Candidate,
	second map[common.Uint168]*state.Candidate) bool {
	if len(first) != len(second) {
		return false
	}
	for k, v := range first {
		v2, ok := second[k]
		if !ok {
			return false
		}

		if !candidateEqual(v, v2) {
			return false
		}
	}
	return true
}

func depositInfoMapEqual(first map[common.Uint168]*state.DepositInfo,
	second map[common.Uint168]*state.DepositInfo) bool {
	if len(first) != len(second) {
		return false
	}
	for k, v := range first {
		v2, ok := second[k]
		if !ok {
			return false
		}

		if !depositInfoEqual(v, v2) {
			return false
		}
	}
	return true
}

func candidatesHistoryMapEqual(first map[uint64]map[common.Uint168]*state.Candidate,
	second map[uint64]map[common.Uint168]*state.Candidate) bool {
	if len(first) != len(second) {
		return false
	}
	for k, v := range first {
		if !candidatesMapEqual(v, second[k]) {
			return false
		}
	}
	return true
}

func keyframeEqual(first *state.KeyFrame, second *state.KeyFrame) bool {
	if first.LastCommitteeHeight != second.LastCommitteeHeight ||
		first.LastVotingStartHeight != second.LastVotingStartHeight ||
		first.InElectionPeriod != second.InElectionPeriod ||
		first.NeedAppropriation != second.NeedAppropriation ||
		first.CRCFoundationBalance != second.CRCFoundationBalance ||
		first.CRCCommitteeBalance != second.CRCCommitteeBalance ||
		first.CRCCommitteeUsedAmount != second.CRCCommitteeUsedAmount ||
		first.DestroyedAmount != second.DestroyedAmount ||
		first.CirculationAmount != second.CirculationAmount ||
		len(first.Members) != len(second.Members) ||
		len(first.HistoryMembers) != len(second.HistoryMembers) ||
		first.CRCCurrentStageAmount != second.CRCCurrentStageAmount {
		return false
	}

	if !membersEuqal(first.Members, second.Members) {
		return false
	}

	for k, v := range first.HistoryMembers {
		if !membersEuqal(v, second.HistoryMembers[k]) {
			return false
		}
	}

	return true
}

func membersEuqal(first map[common.Uint168]*state.CRMember,
	second map[common.Uint168]*state.CRMember) bool {
	for k, v := range first {
		if !crMemberEqual(v, second[k]) {
			return false
		}
	}
	return true
}

func randomKeyFrame(size int, commitHeight uint32) *state.KeyFrame {
	frame := &state.KeyFrame{
		Members:                  make(map[common.Uint168]*state.CRMember, size),
		HistoryMembers:           make(map[uint64]map[common.Uint168]*state.CRMember, 0),
		PartProposalResults:      make([]payload.ProposalResult, size, size),
		LastCommitteeHeight:      commitHeight,
		LastVotingStartHeight:    rand.Uint32(),
		NeedAppropriation:        randomBool(),
		NeedRecordProposalResult: randomBool(),
		CRCFoundationBalance:     randomFix64(),
		CRCCommitteeBalance:      randomFix64(),
		CRCCommitteeUsedAmount:   randomFix64(),
		CRCCurrentStageAmount:    randomFix64(),
		DestroyedAmount:          randomFix64(),
		CirculationAmount:        randomFix64(),
		AppropriationAmount:      randomFix64(),
		CommitteeUsedAmount:      randomFix64(),
		CRAssetsAddressUTXOCount: rand.Uint32(),
	}

	for i := 0; i < size; i++ {
		m := randomCRMember()
		frame.Members[m.Info.DID] = m
	}

	for i := 0; i < size; i++ {
		session := uint64(i)
		frame.HistoryMembers[session] = make(map[common.Uint168]*state.CRMember)
		for j := 0; j <= i; j++ {
			m := randomCRMember()
			frame.HistoryMembers[session][m.Info.DID] = m
		}
	}

	for i := 0; i < size; i++ {
		frame.PartProposalResults = append(frame.PartProposalResults, randomProposalResult())
	}

	return frame
}

func crMemberEqual(first *state.CRMember, second *state.CRMember) bool {
	return crInfoEqual(&first.Info, &second.Info) &&
		first.ImpeachmentVotes == second.ImpeachmentVotes
}

func randomProposalResult() payload.ProposalResult {
	return payload.ProposalResult{
		ProposalHash: *randomUint256(),
		ProposalType: payload.SecretaryGeneral,
		Result:       randomBool(),
	}

}

func randomCRMember() *state.CRMember {
	return &state.CRMember{
		Info:             *randomCRInfo(),
		ImpeachmentVotes: common.Fixed64(rand.Uint64()),
	}
}

func randomStateKeyFrame(size int, hasPending bool) *state.StateKeyFrame {
	frame := state.NewStateKeyFrame()
	for i := 0; i < size; i++ {
		frame.DepositOutputs[randomString()] = randomFix64()
		frame.CRCFoundationOutputs[randomString()] = randomFix64()
		frame.CRCCommitteeOutputs[randomString()] = randomFix64()
		hash := randomUint168()
		frame.DepositHashCIDMap[*hash] = *hash
	}

	if hasPending {
		for i := 0; i < size; i++ {
			candidate := randomCandidate()
			candidate.State = state.Pending
			nickname := candidate.Info.NickName
			code := candidate.Info.Code
			cid := candidate.Info.CID
			frame.CodeCIDMap[common.BytesToHexString(code)] = cid
			frame.Candidates[cid] = candidate
			frame.Nicknames[nickname] = struct{}{}
			frame.DepositInfo[cid] = &state.DepositInfo{
				DepositAmount: 5000 * 1e8,
				TotalAmount:   5000 * 1e8,
			}
		}
	}
	for i := 0; i < size; i++ {
		candidate := randomCandidate()
		candidate.State = state.Active
		code := candidate.Info.Code
		cid := candidate.Info.CID
		nickname := candidate.Info.NickName
		frame.CodeCIDMap[common.BytesToHexString(code)] = cid
		frame.Candidates[cid] = candidate
		frame.Nicknames[nickname] = struct{}{}
		frame.DepositInfo[cid] = &state.DepositInfo{
			DepositAmount: 5000 * 1e8,
			TotalAmount:   5000 * 1e8,
		}
	}
	for i := 0; i < size; i++ {
		candidate := randomCandidate()
		cid := candidate.Info.CID

		nickname := candidate.Info.NickName
		code := candidate.Info.Code
		if i%2 == 0 {
			candidate.State = state.Canceled
		} else {
			candidate.State = state.Returned
		}
		frame.CodeCIDMap[common.BytesToHexString(code)] = cid
		frame.Candidates[cid] = candidate
		frame.Nicknames[nickname] = struct{}{}
		frame.DepositInfo[cid] = &state.DepositInfo{
			DepositAmount: 5000 * 1e8,
			TotalAmount:   5000 * 1e8,
		}
	}
	frame.HistoryCandidates[1] = make(map[common.Uint168]*state.Candidate)
	for i := 0; i < size; i++ {
		candidate := randomCandidate()
		frame.DepositInfo[candidate.Info.DID] = &state.DepositInfo{}
		frame.HistoryCandidates[1][candidate.Info.DID] = candidate
	}
	for i := 0; i < size; i++ {
		frame.Votes[randomString()] = struct{}{}
	}
	return frame
}

func proposalSessionEqual(first, second map[uint64][]common.Uint256) bool {
	if len(first) != len(second) {
		return false
	}
	for firstK, firstV := range first {
		secondV, ok := second[firstK]
		if !ok {
			return false
		}
		if len(firstV) != len(secondV) {
			return false
		}
		for i := range firstV {
			if firstV[i] != secondV[i] {
				return false
			}

		}
	}
	return true
}

func proposalHashEqual(first, second map[common.Uint168]state.ProposalHashSet) bool {
	if len(first) != len(second) {
		return false
	}
	for firstK, firstV := range first {
		secondV, ok := second[firstK]
		if !ok {
			return false
		}

		if !firstV.Equal(secondV) {
			return false
		}
	}
	return true
}

func proposalKeyFrameEqual(first, second *state.ProposalKeyFrame) bool {
	if len(first.Proposals) != len(second.Proposals) {
		return false
	}
	for k, v := range first.Proposals {
		proposalState, exist := second.Proposals[k]
		if !exist {
			return false
		}

		if !v.TxHash.IsEqual(proposalState.TxHash) ||
			v.Status != proposalState.Status ||
			v.VotersRejectAmount != proposalState.VotersRejectAmount ||
			v.VoteStartHeight != proposalState.VoteStartHeight ||
			v.RegisterHeight != proposalState.RegisterHeight {
			return false
		}

		for k, v := range v.CRVotes {
			vote, ok := proposalState.CRVotes[k]
			if !ok {
				return false
			}
			if vote != v {
				return false
			}
		}

		if !v.Proposal.DraftHash.IsEqual(proposalState.Proposal.DraftHash) ||
			v.Proposal.ProposalType != proposalState.Proposal.ProposalType ||
			!bytes.Equal(v.Proposal.OwnerPublicKey, proposalState.Proposal.OwnerPublicKey) ||
			!v.Proposal.CRCouncilMemberDID.IsEqual(proposalState.Proposal.CRCouncilMemberDID) {
			return false
		}

		for i := range v.Proposal.Budgets {
			if v.Proposal.Budgets[i] != proposalState.Proposal.Budgets[i] {
				return false
			}
		}
	}

	if !proposalSessionEqual(first.ProposalSession, second.ProposalSession) {
		return false
	}

	return proposalHashEqual(first.ProposalHashes, second.ProposalHashes)
}

func randomProposalKeyframe() *state.ProposalKeyFrame {
	size := uint64(5)

	proposalKeyFrame := &state.ProposalKeyFrame{Proposals: map[common.Uint256]*state.ProposalState{
		*randomUint256(): randomProposalState(),
		*randomUint256(): randomProposalState(),
		*randomUint256(): randomProposalState(),
		*randomUint256(): randomProposalState(),
		*randomUint256(): randomProposalState(),
	},
		ProposalHashes: map[common.Uint168]state.ProposalHashSet{
			*randomUint168(): randomProposalHashSet(),
			*randomUint168(): randomProposalHashSet(),
		},
		ProposalSession:           make(map[uint64][]common.Uint256, size),
		WithdrawableTxInfo:        make(map[common.Uint256]common2.OutputInfo, size),
		SecretaryGeneralPublicKey: randomString(),
	}

	for i := uint64(0); i < size; i++ {
		proposalKeyFrame.ProposalSession[i] = make([]common.Uint256, size)
		for j := 0; j <= rand.Intn(j+1); j++ {
			proposalKeyFrame.ProposalSession[i] = append(proposalKeyFrame.ProposalSession[i], *randomUint256())
		}
	}
	for i := uint64(0); i < size; i++ {
		hash := randomUint256()
		proposalKeyFrame.WithdrawableTxInfo[*hash] = *randomOutputInfo()
	}
	var reserverCustomIDs []string
	for i := uint64(0); i < size; i++ {
		reserverCustomIDs = []string{}
		for j := uint64(0); j < i; j++ {
			reserverCustomIDs = append(reserverCustomIDs, randomString())
		}
		proposalKeyFrame.ReservedCustomIDLists = append(proposalKeyFrame.ReservedCustomIDLists, reserverCustomIDs...)
	}

	var receivedCustomIDs []string
	for i := uint64(0); i < size; i++ {
		receivedCustomIDs = []string{}
		for j := uint64(0); j < i; j++ {
			receivedCustomIDs = append(receivedCustomIDs, randomString())
		}
		proposalKeyFrame.ReceivedCustomIDLists = append(proposalKeyFrame.ReceivedCustomIDLists, receivedCustomIDs...)
	}
	return proposalKeyFrame
}

func randomProposalState() *state.ProposalState {
	pld := randomCRCProposal()
	return &state.ProposalState{
		Status:             state.ProposalStatus(rand.Int31n(7)),
		Proposal:           pld.ToProposalInfo(0),
		TxHash:             *randomUint256(),
		RegisterHeight:     rand.Uint32(),
		VoteStartHeight:    rand.Uint32(),
		VotersRejectAmount: common.Fixed64(rand.Int63()),
		CRVotes: map[common.Uint168]payload.VoteResult{
			*randomUint168(): payload.VoteResult(rand.Int31n(3)),
			*randomUint168(): payload.VoteResult(rand.Int31n(3)),
			*randomUint168(): payload.VoteResult(rand.Int31n(3)),
			*randomUint168(): payload.VoteResult(rand.Int31n(3)),
			*randomUint168(): payload.VoteResult(rand.Int31n(3)),
		},
	}
}

func randomProposalHashSet() state.ProposalHashSet {
	proposalHashSet := state.NewProposalHashSet()
	count := rand.Int() % 128
	for i := 0; i < count; i++ {
		proposalHashSet.Add(*randomUint256())
	}

	return proposalHashSet
}

func randomCRCProposal() *payload.CRCProposal {
	return &payload.CRCProposal{
		ProposalType:             payload.CRCProposalType(rand.Int31n(6)),
		OwnerPublicKey:           randomBytes(33),
		CRCouncilMemberDID:       *randomUint168(),
		DraftHash:                *randomUint256(),
		Budgets:                  createBudgets(5),
		Signature:                randomBytes(64),
		CRCouncilMemberSignature: randomBytes(64),
	}
}

func createBudgets(n int) []payload.Budget {
	budgets := make([]payload.Budget, 0)
	for i := 0; i < n; i++ {
		var budgetType = payload.NormalPayment
		if i == 0 {
			budgetType = payload.Imprest
		}
		if i == n-1 {
			budgetType = payload.FinalPayment
		}
		budget := &payload.Budget{
			Stage:  byte(i),
			Type:   budgetType,
			Amount: common.Fixed64((i + 1) * 1e8),
		}
		budgets = append(budgets, *budget)
	}
	return budgets
}

func randomFix64() common.Fixed64 {
	var randNum int64
	binary.Read(crand.Reader, binary.BigEndian, &randNum)
	return common.Fixed64(randNum)
}

func randomBool() bool {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(2) == 0
}

func randomOutputInfo() *common2.OutputInfo {
	return &common2.OutputInfo{
		Recipient: *randomUint168(),
		Amount:    randomFix64(),
	}
}
