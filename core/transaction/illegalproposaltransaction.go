// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//
package transaction

import (
	"bytes"
	"errors"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type IllegalProposalTransaction struct {
	BaseTransaction
}

func (a *IllegalProposalTransaction) SpecialCheck() (result elaerr.ELAError, end bool) {
	para := a.contextParameters
	if para.BlockChain.GetState().SpecialTxExists(para.Transaction) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("tx already exists")), true
	}

	py, ok := a.payload.(*payload.DPOSIllegalProposals)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid illegal proposal payload")), true
	}

	if err := a.CheckDPOSIllegalProposals(py); err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	return nil, true
}

func validateProposalEvidence(evidence *payload.ProposalEvidence) error {

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

func (a *IllegalProposalTransaction) ProposalCheckByHeight(proposal *payload.DPOSProposal,
	height uint32) error {
	if err := a.ProposalSanityCheck(proposal); err != nil {
		return err
	}

	if err := a.ProposalContextCheckByHeight(proposal, height); err != nil {
		return err
	}

	return nil
}

func (a *IllegalProposalTransaction) ProposalContextCheckByHeight(proposal *payload.DPOSProposal,
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

func (a *IllegalProposalTransaction) ProposalSanityCheck(proposal *payload.DPOSProposal) error {
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

func (a *IllegalProposalTransaction) CheckDPOSIllegalProposals(d *payload.DPOSIllegalProposals) error {

	if err := validateProposalEvidence(&d.Evidence); err != nil {
		return err
	}

	if err := validateProposalEvidence(&d.CompareEvidence); err != nil {
		return err
	}

	if d.Evidence.BlockHeight != d.CompareEvidence.BlockHeight {
		return errors.New("should be in same height")
	}

	if d.Evidence.Proposal.Hash().IsEqual(d.CompareEvidence.Proposal.Hash()) {
		return errors.New("proposals can not be same")
	}

	if d.Evidence.Proposal.Hash().Compare(
		d.CompareEvidence.Proposal.Hash()) > 0 {
		return errors.New("evidence order error")
	}

	if !bytes.Equal(d.Evidence.Proposal.Sponsor, d.CompareEvidence.Proposal.Sponsor) {
		return errors.New("should be same sponsor")
	}

	if d.Evidence.Proposal.ViewOffset != d.CompareEvidence.Proposal.ViewOffset {
		return errors.New("should in same view")
	}

	if err := a.ProposalCheckByHeight(&d.Evidence.Proposal, d.GetBlockHeight()); err != nil {
		return err
	}

	if err := a.ProposalCheckByHeight(&d.CompareEvidence.Proposal,
		d.GetBlockHeight()); err != nil {
		return err
	}

	return nil
}
