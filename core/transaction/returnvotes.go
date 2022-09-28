// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type ReturnVotesTransaction struct {
	BaseTransaction
}

func (t *ReturnVotesTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.DPoSV2StartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before DPoSV2StartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *ReturnVotesTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.ReturnVotes:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *ReturnVotesTransaction) CheckAttributeProgram() error {
	// Check attributes
	for _, attr := range t.Attributes() {
		if !common2.IsValidAttributeType(attr.Usage) {
			return fmt.Errorf("invalid attribute usage %v", attr.Usage)
		}
	}

	// Check programs
	if len(t.Programs()) != 1 {
		return errors.New("transaction should have only one program")
	}
	if t.Programs()[0].Code == nil {
		return fmt.Errorf("invalid program code nil")
	}
	if t.Programs()[0].Parameter == nil {
		return fmt.Errorf("invalid program parameter nil")
	}

	return nil
}

func (t *ReturnVotesTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *ReturnVotesTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {

	// 1.check if unused vote rights enough
	// 2.return value if payload need to be equal to outputs
	pl, ok := t.Payload().(*payload.ReturnVotes)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	// Value must bigger than RealWithdrawSingleFee
	if pl.Value <= t.parameters.Config.RealWithdrawSingleFee {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid return votes value")), true
	}

	// check if unused vote rights enough
	var code []byte
	if t.payloadVersion == payload.ReturnVotesVersionV0 {
		code = pl.Code
	} else if t.payloadVersion == payload.ReturnVotesVersionV1 {
		code = t.Programs()[0].Code
	} else {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload version")), true
	}

	//1. get stake address
	ct, err := contract.CreateStakeContractByCode(code)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput, err), true
	}
	stakeProgramHash := ct.ToProgramHash()
	state := t.parameters.BlockChain.GetState()
	commitee := t.parameters.BlockChain.GetCRCommittee()
	voteRights := state.DposV2VoteRights[*stakeProgramHash]
	usedDposVoteRights := state.GetUsedDPoSVoteRights(stakeProgramHash)
	usedDposV2VoteRights := state.UsedDposV2Votes[*stakeProgramHash]
	cs := commitee.GetState()
	usedCRVoteRights := cs.GetUsedCRVoteRights(stakeProgramHash)
	usedCRImpeachmentVoteRights := cs.GetUsedCRImpeachmentVoteRights(stakeProgramHash)
	usedCRCProposalVoteRights := cs.GetUsedCRCProposalVoteRights(stakeProgramHash)

	if t.parameters.BlockHeight > state.DPoSV2ActiveHeight {
		if pl.Value > voteRights-usedDposV2VoteRights ||
			pl.Value > voteRights-usedCRVoteRights ||
			pl.Value > voteRights-usedCRImpeachmentVoteRights ||
			pl.Value > voteRights-usedCRCProposalVoteRights {
			return elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("vote rights not enough")), true
		}
	} else {
		if pl.Value > voteRights-usedDposVoteRights ||
			pl.Value > voteRights-usedDposV2VoteRights ||
			pl.Value > voteRights-usedCRVoteRights ||
			pl.Value > voteRights-usedCRImpeachmentVoteRights ||
			pl.Value > voteRights-usedCRCProposalVoteRights {
			return elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("vote rights not enough")), true
		}
	}

	if t.payloadVersion == payload.ReturnVotesVersionV0 {
		//check pl.Code signature
		err = t.checkReturnVotesSignature(pl)
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
	}

	return nil, false
}

// check signature
func (t *ReturnVotesTransaction) checkReturnVotesSignature(returnVotesPayload *payload.ReturnVotes) error {

	signedBuf := new(bytes.Buffer)
	err := returnVotesPayload.SerializeUnsigned(signedBuf, payload.ReturnVotesVersionV0)
	if err != nil {
		return err
	}

	return blockchain.CheckReturnVotesTransactionSignature(returnVotesPayload.Signature, returnVotesPayload.Code, signedBuf.Bytes())
}
