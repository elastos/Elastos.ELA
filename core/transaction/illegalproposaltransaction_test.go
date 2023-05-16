// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//
package transaction

import (
	"bytes"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"math/rand"
)

func (s *txValidatorSpecialTxTestSuite) TestCheckDPOSIllegalProposals() {
	header := randomBlockHeader()
	buf := new(bytes.Buffer)
	header.Serialize(buf)
	evidence := &payload.ProposalEvidence{
		BlockHeight: header.Height, //different from header.Height
		BlockHeader: buf.Bytes(),
		Proposal: payload.DPOSProposal{
			Sponsor:    s.arbitrators.CurrentArbitrators[0].GetNodePublicKey(),
			BlockHash:  header.Hash(),
			ViewOffset: rand.Uint32(),
		},
	}
	evidence.Proposal.Sign, _ = crypto.Sign(s.arbitratorsPriKeys[0],
		evidence.Proposal.Data())
	s.NoError(blockchain.ValidateProposalEvidence(evidence))

	illegalProposals := &payload.DPOSIllegalProposals{
		Evidence:        *evidence,
		CompareEvidence: *evidence,
	}
	s.EqualError(blockchain.CheckDPOSIllegalProposals(illegalProposals),
		"proposals can not be same")

	header2 := randomBlockHeader()
	header2.Height = header.Height + 1 //make sure height is different
	buf = new(bytes.Buffer)
	header2.Serialize(buf)
	cmpEvidence := &payload.ProposalEvidence{
		BlockHeader: buf.Bytes(),
		BlockHeight: header2.Height,
		Proposal: payload.DPOSProposal{
			Sponsor:    s.arbitrators.CurrentArbitrators[0].GetNodePublicKey(),
			BlockHash:  header2.Hash(),
			ViewOffset: rand.Uint32(),
		},
	}
	cmpEvidence.Proposal.Sign, _ = crypto.Sign(
		s.arbitratorsPriKeys[0], cmpEvidence.Proposal.Data())
	illegalProposals.CompareEvidence = *cmpEvidence
	s.EqualError(blockchain.CheckDPOSIllegalProposals(illegalProposals),
		"should be in same height")

	header2.Height = header.Height
	buf = new(bytes.Buffer)
	header2.Serialize(buf)
	cmpEvidence.BlockHeader = buf.Bytes()
	cmpEvidence.BlockHeight = header2.Height
	cmpEvidence.Proposal.ViewOffset =
		evidence.Proposal.ViewOffset + 1 //make sure view offset not the same
	cmpEvidence.Proposal.BlockHash = header2.Hash()
	cmpEvidence.Proposal.Sign, _ = crypto.Sign(
		s.arbitratorsPriKeys[0], cmpEvidence.Proposal.Data())
	illegalProposals.CompareEvidence = *cmpEvidence

	asc := evidence.Proposal.Hash().Compare(cmpEvidence.Proposal.Hash()) < 0
	if asc {
		illegalProposals.Evidence = *cmpEvidence
		illegalProposals.CompareEvidence = *evidence
	}
	s.EqualError(blockchain.CheckDPOSIllegalProposals(illegalProposals),
		"evidence order error")

	if asc {
		illegalProposals.Evidence = *evidence
		illegalProposals.CompareEvidence = *cmpEvidence
	} else {
		illegalProposals.Evidence = *cmpEvidence
		illegalProposals.CompareEvidence = *evidence
	}
	s.EqualError(blockchain.CheckDPOSIllegalProposals(illegalProposals),
		"should in same view")

	cmpEvidence.Proposal.ViewOffset = evidence.Proposal.ViewOffset
	cmpEvidence.Proposal.Sign, _ = crypto.Sign(
		s.arbitratorsPriKeys[0], cmpEvidence.Proposal.Data())
	if evidence.Proposal.Hash().Compare(cmpEvidence.Proposal.Hash()) < 0 {
		illegalProposals.Evidence = *evidence
		illegalProposals.CompareEvidence = *cmpEvidence
	} else {
		illegalProposals.Evidence = *cmpEvidence
		illegalProposals.CompareEvidence = *evidence
	}
	s.NoError(blockchain.CheckDPOSIllegalProposals(illegalProposals))
}
