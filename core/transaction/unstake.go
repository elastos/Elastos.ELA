// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type UnstakeTransaction struct {
	BaseTransaction
}

func (t *UnstakeTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.DposV2StartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before DposV2StartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *UnstakeTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.Unstake:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *UnstakeTransaction) CheckAttributeProgram() error {
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

func (t *UnstakeTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *UnstakeTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {

	// 1.check if unused vote rights enough
	// 2.return value if payload need to be equal to outputs
	pl, ok := t.Payload().(*payload.Unstake)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}
	// check if unused vote rights enough
	code := pl.Code
	//1. get stake address
	ct, err := contract.CreateStakeContractByCode(code)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput, err), true
	}
	stakeProgramHash := ct.ToProgramHash()
	state := t.parameters.BlockChain.GetState()
	voteRights := state.DposV2VoteRights[*stakeProgramHash]
	usedDposVoteRights := state.DposVotes[*stakeProgramHash]
	usedDposV2VoteRights := state.DposV2Votes[*stakeProgramHash]
	usedCRVoteRights := state.CRVotes[*stakeProgramHash]
	usedCRImpeachmentVoteRights := state.CRImpeachmentVotes[*stakeProgramHash]
	usedCRCProposalVoteRights := state.CRCProposalVotes[*stakeProgramHash]

	if pl.Value > voteRights-usedDposVoteRights ||
		pl.Value > voteRights-usedDposV2VoteRights ||
		pl.Value > voteRights-usedCRVoteRights ||
		pl.Value > voteRights-usedCRImpeachmentVoteRights ||
		pl.Value > voteRights-usedCRCProposalVoteRights {
		return elaerr.Simple(elaerr.ErrTxPayload,
			errors.New("vote rights not enough")), true
	}
	//check pl.Code signature
	err = t.checkUnstakeSignature(pl)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}
	return nil, false
}

// check signature
func (t *UnstakeTransaction) checkUnstakeSignature(unstakePayload *payload.Unstake) error {

	pub := unstakePayload.Code[1 : len(unstakePayload.Code)-1]
	publicKey, err := crypto.DecodePoint(pub)
	if err != nil {
		return errors.New("invalid public key in payload")
	}
	signedBuf := new(bytes.Buffer)
	err = unstakePayload.SerializeUnsigned(signedBuf, payload.UnstakeVersion)
	if err != nil {
		return err
	}
	err = crypto.Verify(*publicKey, signedBuf.Bytes(), unstakePayload.Signature)
	if err != nil {
		return errors.New("invalid signature in unstakePayload")
	}
	return nil
}
