// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//
package transactions

import (
	"bytes"
	"errors"
	"github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/log"

	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type IllegaloteTransaction struct {
	BaseTransaction
}

func (a *IllegaloteTransaction) SpecialCheck(para *interfaces.CheckParameters) (result elaerr.ELAError, end bool) {
	if para.SpecialTxExists(para.Transaction.Hash()) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("tx already exists")), true
	}

	if err := a.CheckDPOSIllegalVotes(a.Payload.(*payload.DPOSIllegalVotes), para); err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	return nil, true
}

func (a *IllegaloteTransaction) CheckDPOSIllegalVotes(d *payload.DPOSIllegalVotes, para *interfaces.CheckParameters) error {

	if err := validateVoteEvidence(&d.Evidence); err != nil {
		return err
	}

	if err := validateVoteEvidence(&d.CompareEvidence); err != nil {
		return err
	}

	if d.Evidence.BlockHeight != d.CompareEvidence.BlockHeight {
		return errors.New("should be in same height")
	}

	if d.Evidence.Vote.Hash().IsEqual(d.CompareEvidence.Vote.Hash()) {
		return errors.New("votes can not be same")
	}

	if d.Evidence.Vote.Hash().Compare(d.CompareEvidence.Vote.Hash()) > 0 {
		return errors.New("evidence order error")
	}

	if !bytes.Equal(d.Evidence.Vote.Signer, d.CompareEvidence.Vote.Signer) {
		return errors.New("should be same signer")
	}

	if !bytes.Equal(d.Evidence.Proposal.Sponsor, d.CompareEvidence.Proposal.Sponsor) {
		return errors.New("should be same sponsor")
	}

	if d.Evidence.Proposal.ViewOffset != d.CompareEvidence.Proposal.ViewOffset {
		return errors.New("should in same view")
	}

	if err := a.ProposalCheckByHeight(&d.Evidence.Proposal,
		d.GetBlockHeight(), para); err != nil {
		return err
	}

	if err := a.ProposalCheckByHeight(&d.CompareEvidence.Proposal,
		d.GetBlockHeight(), para); err != nil {
		return err
	}

	if err := a.VoteCheckByHeight(&d.Evidence.Vote,
		d.GetBlockHeight(), para); err != nil {
		return err
	}

	if err := a.VoteCheckByHeight(&d.CompareEvidence.Vote,
		d.GetBlockHeight(), para); err != nil {
		return err
	}

	return nil
}

func validateVoteEvidence(evidence *payload.VoteEvidence) error {
	if err := validateProposalEvidence(&evidence.ProposalEvidence); err != nil {
		return err
	}

	if !evidence.Proposal.Hash().IsEqual(evidence.Vote.ProposalHash) {
		return errors.New("vote and proposal should match")
	}

	return nil
}

func (a *IllegaloteTransaction) VoteCheckByHeight(vote *payload.DPOSProposalVote, height uint32, para *interfaces.CheckParameters) error {
	if err := voteSanityCheck(vote); err != nil {
		return err
	}

	if err := a.VoteContextCheckByHeight(vote, height, para); err != nil {
		log.Warn("[VoteContextCheck] error: ", err.Error())
		return err
	}

	return nil
}

func voteSanityCheck(vote *payload.DPOSProposalVote) error {
	pubKey, err := crypto.DecodePoint(vote.Signer)
	if err != nil {
		return err
	}
	err = crypto.Verify(*pubKey, vote.Data(), vote.Sign)
	if err != nil {
		return err
	}

	return nil
}

func (a *IllegaloteTransaction) VoteContextCheckByHeight(vote *payload.DPOSProposalVote,
	height uint32, para *interfaces.CheckParameters) error {
	var isArbiter bool
	nodePublicKeys := para.GetCurrentArbitratorNodePublicKeys(height)
out:
	for _, pk := range nodePublicKeys {
		if bytes.Equal(pk, vote.Signer) {
			isArbiter = true
			break out
		}
	}
	if !isArbiter {
		return errors.New("current arbitrators verify error")
	}

	return nil
}

func (a *IllegaloteTransaction) validateProposalEvidence(evidence *payload.ProposalEvidence) error {

	header := &common.Header{}
	buf := new(bytes.Buffer)
	buf.Write(evidence.BlockHeader)

	if err := header.Deserialize(buf); err != nil {
		return err
	}

	if header.Height != evidence.BlockHeight {
		return errors.New("evidence height and block height should match")
	}

	if !header.Hash().IsEqual(evidence.Proposal.BlockHash) {
		return errors.New("proposal hash and block should match")
	}

	return nil
}

func (a *IllegaloteTransaction) ProposalCheckByHeight(proposal *payload.DPOSProposal,
	height uint32, para *interfaces.CheckParameters) error {
	if err := a.ProposalSanityCheck(proposal); err != nil {
		return err
	}

	if err := a.ProposalContextCheckByHeight(proposal, height, para); err != nil {
		return err
	}

	return nil
}

func (a *IllegaloteTransaction) ProposalContextCheckByHeight(proposal *payload.DPOSProposal,
	height uint32, para *interfaces.CheckParameters) error {
	var isArbiter bool
	//keyFrames := DefaultLedger.Arbitrators.GetSnapshot(height)
	nodePublicKeys := para.GetCurrentArbitratorNodePublicKeys(height)
out:
	for _, pk := range nodePublicKeys {
		if bytes.Equal(pk, proposal.Sponsor) {
			isArbiter = true
			break out
		}
	}
	if !isArbiter {
		return errors.New("current arbitrators verify error")
	}

	return nil
}

func (a *IllegaloteTransaction) ProposalSanityCheck(proposal *payload.DPOSProposal) error {
	pubKey, err := crypto.DecodePoint(proposal.Sponsor)
	if err != nil {
		return err
	}
	err = crypto.Verify(*pubKey, proposal.Data(), proposal.Sign)
	if err != nil {
		return err
	}

	return nil
}
