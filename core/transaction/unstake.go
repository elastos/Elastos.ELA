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
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type UnstakeTransaction struct {
	BaseTransaction
}

func (t *UnstakeTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.DPoSV2StartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before DPoSV2StartHeight", t.TxType().Name()))
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
	return nil, true
	// todo complete me
	//
	//// 1.check if unused vote rights enough
	//// 2.return value if payload need to be equal to outputs
	//pl, ok := t.Payload().(*payload.Unstake)
	//if !ok {
	//	return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	//}
	//// check if unused vote rights enough
	//code := pl.Code
	////1. get stake address
	//ct, err := contract.CreateStakeContractByCode(code)
	//if err != nil {
	//	return elaerr.Simple(elaerr.ErrTxInvalidOutput, err), true
	//}
	//stakeProgramHash := ct.ToProgramHash()
	//state := t.parameters.BlockChain.GetState()
	//commitee := t.parameters.BlockChain.GetCRCommittee()
	//voteRights := state.DposV2VoteRights[*stakeProgramHash]
	//usedDposVoteRights := state.UsedDPoSVotes[*stakeProgramHash]
	//usedDposV2VoteRights := state.UsedDposV2Votes[*stakeProgramHash]
	//cs := commitee.GetState()
	//usedCRVoteRights := cs.UsedCRVotes[*stakeProgramHash]
	//usedCRImpeachmentVoteRights := cs.UsdedCRImpeachmentVotes[*stakeProgramHash]
	//usedCRCProposalVoteRights := cs.UsedCRCProposalVotes[*stakeProgramHash]
	//
	//if t.parameters.BlockHeight > state.DPoSV2ActiveHeight {
	//	if pl.Value > voteRights-usedDposV2VoteRights ||
	//		pl.Value > voteRights-usedCRVoteRights ||
	//		pl.Value > voteRights-usedCRImpeachmentVoteRights ||
	//		pl.Value > voteRights-usedCRCProposalVoteRights {
	//		return elaerr.Simple(elaerr.ErrTxPayload,
	//			errors.New("vote rights not enough")), true
	//	}
	//} else {
	//	if pl.Value > voteRights-usedDposVoteRights ||
	//		pl.Value > voteRights-usedDposV2VoteRights ||
	//		pl.Value > voteRights-usedCRVoteRights ||
	//		pl.Value > voteRights-usedCRImpeachmentVoteRights ||
	//		pl.Value > voteRights-usedCRCProposalVoteRights {
	//		return elaerr.Simple(elaerr.ErrTxPayload,
	//			errors.New("vote rights not enough")), true
	//	}
	//}
	//
	////check pl.Code signature
	//err = t.checkUnstakeSignature(pl)
	//if err != nil {
	//	return elaerr.Simple(elaerr.ErrTxPayload, err), true
	//}
	//return nil, false
}

// check signature
func (t *UnstakeTransaction) checkUnstakeSignature(unstakePayload *payload.Unstake) error {

	signedBuf := new(bytes.Buffer)
	err := unstakePayload.SerializeUnsigned(signedBuf, payload.UnstakeVersion)
	if err != nil {
		return err
	}

	return blockchain.CheckCRTransactionSignature(unstakePayload.Signature, unstakePayload.Code, signedBuf.Bytes())
}
