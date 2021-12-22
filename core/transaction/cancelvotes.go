// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type CancelVotesTransaction struct {
	BaseTransaction
}

func (t *CancelVotesTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.DposV2StartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before DposV2StartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *CancelVotesTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.CancelVote:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *CancelVotesTransaction) CheckAttributeProgram() error {
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

func (t *CancelVotesTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *CancelVotesTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {

	state := t.parameters.BlockChain.GetState()
	committee := t.parameters.BlockChain.GetCRCommittee()
	pld := t.Payload().(*payload.CancelVote)
	var vote payload.DetailedVoteInfo
	for _, referKey := range pld.ReferKeys {
		var err error
		// 1.check if refer key is CRC votes
		vote, err = committee.GetDetailedCRVotes(referKey)
		if err == nil {
			continue
		}
		// 2.check if refer key is CRC Impeachment votes
		vote, err = committee.GetDetailedCRImpeachmentVotes(referKey)
		if err == nil {
			continue
		}
		// 3.check if refer key is CRC Proposal votes
		vote, err = committee.GetDetailedCRCProposalVotes(referKey)
		if err == nil {
			continue
		}
		// 4.check if refer key is DPoS V1 votes
		vote, err = state.GetDetailedDPoSV1Votes(referKey)
		if err == nil {
			continue
		}
		// 5.should not be DPoS V2 votes, because DPoS V2 will be canceled when
		// block height reached the stakeUntil height
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("vote type "+
			"need to be CR CRImpeachment CRCProposal or DPoSV1")), true
	}

	// check if the signature if from the owner
	ct, err := contract.CreateStakeContractByCode(t.Programs()[0].Code)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}
	stakeProgramHash := ct.ToProgramHash()

	if !vote.StakeProgramHash.IsEqual(*stakeProgramHash) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid signer")), true
	}

	return nil, false
}

func (t *CancelVotesTransaction) checkVoteProducerContent(content payload.VotesContent,
	pds map[string]struct{}, amount common.Fixed64, voteRights common.Fixed64) error {
	for _, cv := range content.VotesInfo {
		if _, ok := pds[common.BytesToHexString(cv.Candidate)]; !ok {
			return fmt.Errorf("invalid vote output payload "+
				"producer candidate: %s", common.BytesToHexString(cv.Candidate))
		}
	}
	var maxVotes common.Fixed64
	for _, cv := range content.VotesInfo {
		if cv.LockTime != 0 {
			return errors.New("votes lock time need to be zero")
		}
		if cv.Votes > amount {
			return errors.New("votes larger than output amount")
		}
		if maxVotes < cv.Votes {
			maxVotes = cv.Votes
		}
	}
	if maxVotes > voteRights {
		return errors.New("DPoS vote rights not enough")
	}

	return nil
}
