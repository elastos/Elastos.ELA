// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"github.com/elastos/Elastos.ELA/test/unit"
	"testing"

	"github.com/elastos/Elastos.ELA/common/config"

	"github.com/stretchr/testify/assert"
)

func TestProposalManager_Queries(t *testing.T) {
	manager := NewProposalManager(&config.DefaultParams)

	proposalKey := unit.randomUint256()
	proposalState := unit.randomProposalState()
	manager.Proposals[*proposalKey] = proposalState

	assert.True(t, manager.existProposal(*proposalKey))
	assert.False(t, manager.existProposal(*unit.randomUint256()))

	assert.True(t, manager.existDraft(proposalState.Proposal.DraftHash))
	assert.False(t, manager.existDraft(*unit.randomUint256()))

	assert.Equal(t, proposalState, manager.getProposal(*proposalKey))
}
