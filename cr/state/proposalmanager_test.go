// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"math/rand"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/stretchr/testify/assert"
)

func TestProposalManager_Queries(t *testing.T) {
	manager := NewProposalManager(&config.DefaultParams)

	proposalKey := randomUint256()
	proposalState := randomProposalState()
	manager.Proposals[*proposalKey] = proposalState

	assert.True(t, manager.existProposal(*proposalKey))
	assert.False(t, manager.existProposal(*randomUint256()))

	assert.True(t, manager.existDraft(proposalState.Proposal.DraftHash))
	assert.False(t, manager.existDraft(*randomUint256()))

	assert.Equal(t, proposalState, manager.getProposal(*proposalKey))
}

func randomUint256() *common.Uint256 {
	randBytes := make([]byte, 32)
	rand.Read(randBytes)

	result, _ := common.Uint256FromBytes(randBytes)
	return result
}

func randomBytes(len int) []byte {
	a := make([]byte, len)
	rand.Read(a)
	return a
}

func randomUint168() *common.Uint168 {
	randBytes := make([]byte, 21)
	rand.Read(randBytes)
	result, _ := common.Uint168FromBytes(randBytes)

	return result
}

func randomProposalState() *ProposalState {
	pld := randomCRCProposal()
	return &ProposalState{
		Status:             ProposalStatus(rand.Int31n(7)),
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

func randomCRCProposal() *payload.CRCProposal {
	return &payload.CRCProposal{
		ProposalType:             payload.CRCProposalType(rand.Int31n(6)),
		OwnerKey:                 randomBytes(33),
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
