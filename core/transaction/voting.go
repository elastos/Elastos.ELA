// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"

	"github.com/elastos/Elastos.ELA/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type VotingTransaction struct {
	BaseTransaction
}

func (t *VotingTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.DposV2StartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before DposV2StartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *VotingTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.Voting:
		return t.Payload().(*payload.Voting).Vote.Validate()

	}

	return errors.New("invalid payload type")
}

func (t *VotingTransaction) CheckAttributeProgram() error {
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

func (t *VotingTransaction) IsAllowedInPOWConsensus() bool {
	pld := t.Payload().(*payload.Voting)

	for _, vote := range pld.Vote.Contents {
		switch vote.VoteType {
		case outputpayload.Delegate:
		case outputpayload.CRC:
			log.Warn("not allow to vote CR in POW consensus")
			return false
		case outputpayload.CRCProposal:
			log.Warn("not allow to vote CRC proposal in POW consensus")
			return false
		case outputpayload.CRCImpeachment:
			log.Warn("not allow to vote CRImpeachment in POW consensus")
			return false
		}
	}

	inputProgramHashes := make(map[common.Uint168]struct{})
	for _, output := range t.references {
		inputProgramHashes[output.ProgramHash] = struct{}{}
	}
	outputProgramHashes := make(map[common.Uint168]struct{})
	for _, output := range t.Outputs() {
		outputProgramHashes[output.ProgramHash] = struct{}{}
	}
	for k, _ := range outputProgramHashes {
		if _, ok := inputProgramHashes[k]; !ok {
			log.Warn("output program hash is not in inputs")
			return false
		}
	}

	return true
}

func (t *VotingTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {

	// 1.check if the signer has vote rights and check if votes enough
	// 2.check different type of votes, enough? candidate exist?
	code := t.Programs()[0].Code
	pld := t.Payload().(*payload.Voting)
	ct, err := contract.CreateStakeContractByCode(code)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput, err), true
	}
	stakeProgramHash := ct.ToProgramHash()
	voteRights := t.parameters.BlockChain.GetState().DposV2VoteRights
	votes, exist := voteRights[*stakeProgramHash]
	if !exist {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput, errors.New("has no vote rights")), true
	}

	// todo check if not used vote rights enough?
	//for _, vote := range pld.Vote.Contents {
	//	switch vote.VoteType {
	//	case outputpayload.Delegate:
	//		for _,
	//
	//		if votes < vote.CandidateVotes {
	//			return elaerr.Simple(elaerr.ErrTxInvalidOutput, errors.New("code not match output program hash")), true
	//		}
	//
	//	case outputpayload.CRC:
	//		log.Warn("not allow to vote CR in POW consensus")
	//		return false
	//	case outputpayload.CRCProposal:
	//		log.Warn("not allow to vote CRC proposal in POW consensus")
	//		return false
	//	case outputpayload.CRCImpeachment:
	//		log.Warn("not allow to vote CRImpeachment in POW consensus")
	//		return false
	//	}
	//}
	//
	//if votes < pld {
	//	return elaerr.Simple(elaerr.ErrTxInvalidOutput, errors.New("code not match output program hash")), true
	//}

	return nil, false
}
