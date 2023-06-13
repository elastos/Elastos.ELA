// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package transaction

import (
	"bytes"
	"errors"
	"fmt"
	"math"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type IllegalVoteTransaction struct {
	BaseTransaction
}

func (t *IllegalVoteTransaction) CheckTransactionInput() error {
	if len(t.Inputs()) != 0 {
		return errors.New("no cost transactions must has no input")
	}
	return nil
}

func (t *IllegalVoteTransaction) CheckTransactionOutput() error {

	if len(t.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}
	if len(t.Outputs()) != 0 {
		return errors.New("no cost transactions should have no output")
	}

	return nil
}

func (t *IllegalVoteTransaction) CheckAttributeProgram() error {
	if len(t.Programs()) != 0 || len(t.Attributes()) != 0 {
		return errors.New("zero cost tx should have no attributes and programs")
	}
	return nil
}

func (t *IllegalVoteTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.DPOSIllegalVotes:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *IllegalVoteTransaction) IsAllowedInPOWConsensus() bool {
	return true
}

func (t *IllegalVoteTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {
	if t.parameters.BlockChain.GetState().SpecialTxExists(t) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("tx already exists")), true
	}

	if err := t.CheckDPOSIllegalVotes(t.payload.(*payload.DPOSIllegalVotes)); err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	return nil, true
}

func (t *IllegalVoteTransaction) CheckDPOSIllegalVotes(d *payload.DPOSIllegalVotes) error {

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

	if t.parameters.BlockHeight > t.parameters.Config.DPoSConfiguration.IllegalV2Height {
		if !d.Evidence.Proposal.Hash().IsEqual(d.CompareEvidence.Proposal.Hash()) {
			return errors.New("should be same proposal")
		}
	}

	if d.Evidence.Proposal.ViewOffset != d.CompareEvidence.Proposal.ViewOffset {
		return errors.New("should in same view")
	}

	if err := t.ProposalCheckByHeight(&d.Evidence.Proposal,
		d.GetBlockHeight()); err != nil {
		return err
	}

	if err := t.ProposalCheckByHeight(&d.CompareEvidence.Proposal,
		d.GetBlockHeight()); err != nil {
		return err
	}

	if err := t.VoteCheckByHeight(&d.Evidence.Vote,
		d.GetBlockHeight()); err != nil {
		return err
	}

	if err := t.VoteCheckByHeight(&d.CompareEvidence.Vote,
		d.GetBlockHeight()); err != nil {
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

func (t *IllegalVoteTransaction) VoteCheckByHeight(vote *payload.DPOSProposalVote, height uint32) error {
	if err := voteSanityCheck(vote); err != nil {
		return err
	}

	if err := t.VoteContextCheckByHeight(vote, height); err != nil {
		fmt.Println("[VoteContextCheck] error: ", err.Error())
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

func (t *IllegalVoteTransaction) VoteContextCheckByHeight(
	vote *payload.DPOSProposalVote, height uint32) error {
	var isArbiter bool
	keyFrames := blockchain.DefaultLedger.Arbitrators.GetSnapshot(height)
out:
	for _, k := range keyFrames {
		for _, a := range k.CurrentArbitrators {
			if bytes.Equal(a.GetNodePublicKey(), vote.Signer) {
				isArbiter = true
				break out
			}
		}
	}
	if !isArbiter {
		return errors.New("current arbitrators verify error")
	}

	return nil
}

func (t *IllegalVoteTransaction) validateProposalEvidence(evidence *payload.ProposalEvidence) error {

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

func (t *IllegalVoteTransaction) ProposalCheckByHeight(proposal *payload.DPOSProposal,
	height uint32) error {
	if err := t.ProposalSanityCheck(proposal); err != nil {
		return err
	}

	if err := t.ProposalContextCheckByHeight(proposal, height); err != nil {
		return err
	}

	return nil
}

func (t *IllegalVoteTransaction) ProposalContextCheckByHeight(proposal *payload.DPOSProposal,
	height uint32) error {
	var isArbiter bool
	keyFrames := blockchain.DefaultLedger.Arbitrators.GetSnapshot(height)
out:
	for _, k := range keyFrames {
		for _, a := range k.CurrentArbitrators {
			if bytes.Equal(a.GetNodePublicKey(), proposal.Sponsor) {
				isArbiter = true
				break out
			}
		}
	}
	if !isArbiter {
		return errors.New("current arbitrators verify error")
	}

	return nil
}

func (t *IllegalVoteTransaction) ProposalSanityCheck(proposal *payload.DPOSProposal) error {
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
