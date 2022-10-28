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

func (s *txValidatorSpecialTxTestSuite) TestValidateVoteEvidence() {
	header := randomBlockHeader()
	buf := new(bytes.Buffer)
	header.Serialize(buf)
	evidence := &payload.VoteEvidence{
		ProposalEvidence: payload.ProposalEvidence{
			BlockHeight: header.Height,
			BlockHeader: buf.Bytes(),
			Proposal: payload.DPOSProposal{
				BlockHash:  header.Hash(),
				Sponsor:    s.arbitrators.CurrentArbitrators[0].GetNodePublicKey(),
				ViewOffset: rand.Uint32(),
			},
		},
		Vote: payload.DPOSProposalVote{},
	}
	evidence.Proposal.Sign, _ = crypto.Sign(s.arbitratorsPriKeys[0],
		evidence.Proposal.Data())

	s.EqualError(blockchain.ValidateVoteEvidence(evidence),
		"vote and proposal should match")

	evidence.Vote.ProposalHash = evidence.Proposal.Hash()
	s.NoError(blockchain.ValidateVoteEvidence(evidence), "vote verify error")

	evidence.Vote.Signer =
		s.arbitrators.CurrentArbitrators[1].GetNodePublicKey()
	evidence.Vote.Accept = true
	evidence.Vote.Sign, _ = crypto.Sign(s.arbitratorsPriKeys[1],
		evidence.Vote.Data())
	s.NoError(blockchain.ValidateVoteEvidence(evidence))
}

func (s *txValidatorSpecialTxTestSuite) TestCheckDPOSIllegalVotes_SameProposal() {
	header := randomBlockHeader()
	buf := new(bytes.Buffer)
	header.Serialize(buf)
	evidence := &payload.VoteEvidence{
		ProposalEvidence: payload.ProposalEvidence{
			BlockHeight: header.Height,
			BlockHeader: buf.Bytes(),
			Proposal: payload.DPOSProposal{
				BlockHash:  header.Hash(),
				Sponsor:    s.arbitrators.CurrentArbitrators[0].GetNodePublicKey(),
				ViewOffset: rand.Uint32(),
			},
		},
		Vote: payload.DPOSProposalVote{
			Signer: s.arbitrators.CurrentArbitrators[1].GetNodePublicKey(),
			Accept: true,
		},
	}
	evidence.Proposal.Sign, _ = crypto.Sign(s.arbitratorsPriKeys[0],
		evidence.Proposal.Data())
	evidence.Vote.ProposalHash = evidence.Proposal.Hash()
	evidence.Vote.Sign, _ = crypto.Sign(s.arbitratorsPriKeys[1],
		evidence.Vote.Data())

	illegalVotes := &payload.DPOSIllegalVotes{
		Evidence:        *evidence,
		CompareEvidence: *evidence,
	}
	s.EqualError(blockchain.CheckDPOSIllegalVotes(illegalVotes),
		"votes can not be same")

	//create compare evidence with the same proposal
	cmpEvidence := &payload.VoteEvidence{
		ProposalEvidence: evidence.ProposalEvidence,
		Vote: payload.DPOSProposalVote{
			Signer:       s.arbitrators.CurrentArbitrators[1].GetNodePublicKey(),
			Accept:       false,
			ProposalHash: evidence.Proposal.Hash(),
		},
	}
	cmpEvidence.Vote.Sign, _ = crypto.Sign(s.arbitratorsPriKeys[1],
		cmpEvidence.Vote.Data())

	asc := evidence.Vote.Hash().Compare(cmpEvidence.Vote.Hash()) < 0
	if asc {
		illegalVotes.Evidence = *cmpEvidence
		illegalVotes.CompareEvidence = *evidence
	} else {
		illegalVotes.Evidence = *evidence
		illegalVotes.CompareEvidence = *cmpEvidence
	}
	s.EqualError(blockchain.CheckDPOSIllegalVotes(illegalVotes),
		"evidence order error")

	if asc {
		illegalVotes.Evidence = *evidence
		illegalVotes.CompareEvidence = *cmpEvidence
	} else {
		illegalVotes.Evidence = *cmpEvidence
		illegalVotes.CompareEvidence = *evidence
	}
	s.NoError(blockchain.CheckDPOSIllegalVotes(illegalVotes))
}

func (s *txValidatorSpecialTxTestSuite) TestCheckDPOSIllegalVotes_DiffProposal() {
	header := randomBlockHeader()
	buf := new(bytes.Buffer)
	header.Serialize(buf)
	evidence := &payload.VoteEvidence{
		ProposalEvidence: payload.ProposalEvidence{
			BlockHeight: header.Height,
			BlockHeader: buf.Bytes(),
			Proposal: payload.DPOSProposal{
				BlockHash:  header.Hash(),
				Sponsor:    s.arbitrators.CurrentArbitrators[0].GetNodePublicKey(),
				ViewOffset: rand.Uint32(),
			},
		},
		Vote: payload.DPOSProposalVote{
			Signer: s.arbitrators.CurrentArbitrators[1].GetNodePublicKey(),
			Accept: true,
		},
	}
	s.updateEvidenceSigns(evidence, s.arbitratorsPriKeys[0], s.arbitratorsPriKeys[1])

	//create compare evidence with the different proposal
	header2 := randomBlockHeader()
	header2.Height = header.Height + 1 //make sure height is different
	buf = new(bytes.Buffer)
	header2.Serialize(buf)
	cmpEvidence := &payload.VoteEvidence{
		ProposalEvidence: payload.ProposalEvidence{
			BlockHeight: header2.Height,
			BlockHeader: buf.Bytes(),
			Proposal: payload.DPOSProposal{
				BlockHash:  header2.Hash(),
				Sponsor:    s.arbitrators.CurrentArbitrators[0].GetNodePublicKey(),
				ViewOffset: rand.Uint32(),
			},
		},
		Vote: payload.DPOSProposalVote{
			Signer:       s.arbitrators.CurrentArbitrators[1].GetNodePublicKey(),
			Accept:       false,
			ProposalHash: evidence.Proposal.Hash(),
		},
	}
	s.updateEvidenceSigns(cmpEvidence, s.arbitratorsPriKeys[0],
		s.arbitratorsPriKeys[1])

	illegalVotes := &payload.DPOSIllegalVotes{
		Evidence:        *evidence,
		CompareEvidence: *cmpEvidence,
	}
	s.EqualError(blockchain.CheckDPOSIllegalVotes(illegalVotes),
		"should be in same height")

	header2.Height = header.Height
	buf = new(bytes.Buffer)
	header2.Serialize(buf)
	cmpEvidence.BlockHeight = header2.Height
	cmpEvidence.BlockHeader = buf.Bytes()
	cmpEvidence.Proposal.BlockHash = header2.Hash()
	cmpEvidence.Proposal.Sponsor = //set different sponsor
		s.arbitrators.CurrentArbitrators[2].GetNodePublicKey()
	s.updateEvidenceSigns(cmpEvidence, s.arbitratorsPriKeys[2],
		s.arbitratorsPriKeys[1])
	if evidence.Vote.Hash().Compare(cmpEvidence.Vote.Hash()) < 0 {
		illegalVotes.Evidence = *evidence
		illegalVotes.CompareEvidence = *cmpEvidence
	} else {
		illegalVotes.Evidence = *cmpEvidence
		illegalVotes.CompareEvidence = *evidence
	}
	s.EqualError(blockchain.CheckDPOSIllegalVotes(illegalVotes),
		"should be same sponsor")

	// set different view offset
	cmpEvidence.Proposal.Sponsor =
		s.arbitrators.CurrentArbitrators[0].GetNodePublicKey()
	cmpEvidence.Proposal.ViewOffset = evidence.Proposal.ViewOffset + 1
	s.updateEvidenceSigns(cmpEvidence, s.arbitratorsPriKeys[0],
		s.arbitratorsPriKeys[1])
	if evidence.Vote.Hash().Compare(cmpEvidence.Vote.Hash()) < 0 {
		illegalVotes.Evidence = *evidence
		illegalVotes.CompareEvidence = *cmpEvidence
	} else {
		illegalVotes.Evidence = *cmpEvidence
		illegalVotes.CompareEvidence = *evidence
	}
	s.EqualError(blockchain.CheckDPOSIllegalVotes(illegalVotes),
		"should in same view")

	// let check method pass
	cmpEvidence.Proposal.ViewOffset = evidence.Proposal.ViewOffset
	s.updateEvidenceSigns(cmpEvidence, s.arbitratorsPriKeys[0],
		s.arbitratorsPriKeys[1])
	if evidence.Vote.Hash().Compare(cmpEvidence.Vote.Hash()) < 0 {
		illegalVotes.Evidence = *evidence
		illegalVotes.CompareEvidence = *cmpEvidence
	} else {
		illegalVotes.Evidence = *cmpEvidence
		illegalVotes.CompareEvidence = *evidence
	}
	s.NoError(blockchain.CheckDPOSIllegalVotes(illegalVotes))
}

func (s *txValidatorSpecialTxTestSuite) updateEvidenceSigns(
	evidence *payload.VoteEvidence, proposalSigner, voteSigner []byte) {
	evidence.Proposal.Sign, _ = crypto.Sign(proposalSigner,
		evidence.Proposal.Data())
	evidence.Vote.ProposalHash = evidence.Proposal.Hash()
	evidence.Vote.Sign, _ = crypto.Sign(voteSigner, evidence.Vote.Data())
}
