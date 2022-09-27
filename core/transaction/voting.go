// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type VotingTransaction struct {
	BaseTransaction
}

func (t *VotingTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.DPoSV2StartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before DPoSV2StartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *VotingTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.Voting:
		return t.Payload().(*payload.Voting).Validate()

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

	for _, vote := range pld.Contents {
		switch vote.VoteType {
		case outputpayload.Delegate, outputpayload.DposV2:
		case outputpayload.CRC:
			log.Warn("not allow to vote CR in POW consensus")
			return false
		case outputpayload.CRCProposal:
			log.Warn("not allow to vote CRC proposal in POW consensus")
			return false
		case outputpayload.CRCImpeachment:
			log.Warn("not allow to vote CRImpeachment in POW consensus")
			return false
		default:
			log.Warn("unknown vote type")
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
	blockHeight := t.parameters.BlockHeight
	crCommittee := t.parameters.BlockChain.GetCRCommittee()
	producers := t.parameters.BlockChain.GetState().GetActiveV1Producers()
	pds := getProducerPublicKeysMap(producers)
	pds2 := getDPoSV2ProducersMap(t.parameters.BlockChain.GetState().GetActivityV2Producers())

	// vote rights should be more than vote rights used in payload
	code := t.Programs()[0].Code
	ct, err := contract.CreateStakeContractByCode(code)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput, err), true
	}
	stakeProgramHash := ct.ToProgramHash()
	state := t.parameters.BlockChain.GetState()
	//commitee := t.parameters.BlockChain.GetCRCommittee()
	voteRights := state.DposV2VoteRights
	totalVotes, exist := voteRights[*stakeProgramHash]
	if !exist {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput, errors.New("has no vote rights")), true
	}
	usedDPoSV2VoteRights, _ := state.UsedDposV2Votes[*stakeProgramHash]

	var candidates []*crstate.Candidate
	if crCommittee.IsInVotingPeriod(blockHeight) {
		candidates = crCommittee.GetCandidates(crstate.Active)
	} else {
		candidates = []*crstate.Candidate{}
	}
	crs := getCRCIDsMap(candidates)

	pld := t.Payload().(*payload.Voting)
	switch t.PayloadVersion() {
	case payload.VoteVersion:
		if len(pld.Contents) == 0 {
			return elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("contents is nil")), true
		}

		for _, content := range pld.Contents {
			for _, vi := range content.VotesInfo {
				if vi.Votes <= 0 {
					return elaerr.Simple(elaerr.ErrTxPayload,
						errors.New("invalid votes, need to be bigger than zero")), true
				}
			}

			switch content.VoteType {
			case outputpayload.Delegate:
				if blockHeight > state.DPoSV2ActiveHeight {
					return elaerr.Simple(elaerr.ErrTxPayload,
						errors.New("delegate votes is not allowed in DPoS V2")), true
				}

				err := t.checkVoteProducerContent(
					content, pds, totalVotes)
				if err != nil {
					return elaerr.Simple(elaerr.ErrTxPayload, err), true
				}
			case outputpayload.CRC:
				if !t.parameters.BlockChain.GetCRCommittee().IsInVotingPeriod(t.parameters.BlockHeight) {
					return elaerr.Simple(elaerr.ErrTxPayload, errors.New("should vote CR during voting period")), true
				}
				err := t.checkVoteCRContent(blockHeight,
					content, crs, totalVotes)
				if err != nil {
					return elaerr.Simple(elaerr.ErrTxPayload, err), true
				}
			case outputpayload.CRCProposal:
				err := t.checkVoteCRCProposalContent(
					content, totalVotes)
				if err != nil {
					return elaerr.Simple(elaerr.ErrTxPayload, err), true
				}
			case outputpayload.CRCImpeachment:
				err := t.checkCRImpeachmentContent(
					content, totalVotes)
				if err != nil {
					return elaerr.Simple(elaerr.ErrTxPayload, err), true
				}
			case outputpayload.DposV2:
				err := t.checkDPoSV2Content(content, pds2, totalVotes-usedDPoSV2VoteRights)
				if err != nil {
					return elaerr.Simple(elaerr.ErrTxPayload, err), true
				}
			}
		}
	case payload.RenewalVoteVersion:
		if len(pld.RenewalContents) == 0 {
			return elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("renewal contents is nil")), true
		}
		for _, content := range pld.RenewalContents {
			producer := state.GetProducer(content.VotesInfo.Candidate)
			if producer == nil {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("producer can not found")), true
			}
			vote, err := producer.GetDetailedDPoSV2Votes(*stakeProgramHash, content.ReferKey)
			if err != nil {
				return elaerr.Simple(elaerr.ErrTxPayload, err), true
			}
			if vote.VoteType != outputpayload.DposV2 {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid vote type")), true
			}
			if len(vote.Info) != 1 || vote.Info[0].Votes != content.VotesInfo.Votes {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("votes not equal")), true
			}
			if content.VotesInfo.LockTime <= vote.Info[0].LockTime {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("new lock time <= old lock time")), true
			}
			if content.VotesInfo.LockTime > producer.Info().StakeUntil {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("new lock time > producer StakeUntil")), true
			}
			if content.VotesInfo.LockTime-vote.BlockHeight > t.parameters.Config.DPoSV2MaxVotesLockTime {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid lock time > DPoSV2MaxVotesLockTime")), true
			}
			if !bytes.Equal(vote.Info[0].Candidate, content.VotesInfo.Candidate) {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("candidate should be the same one")), true
			}
		}
	default:
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload version")), true
	}

	return nil, false
}

func (t *VotingTransaction) checkVoteProducerContent(content payload.VotesContent,
	pds map[string]struct{}, voteRights common.Fixed64) error {

	if len(content.VotesInfo) > outputpayload.MaxVoteProducersPerTransaction {
		return errors.New("votes count bigger than MaxVoteProducersPerTransaction")
	}

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
		if maxVotes < cv.Votes {
			maxVotes = cv.Votes
		}
	}
	if maxVotes > voteRights {
		return errors.New("DPoS vote rights not enough")
	}

	return nil
}

func (t *VotingTransaction) checkVoteCRContent(blockHeight uint32,
	content payload.VotesContent, crs map[common.Uint168]struct{},
	voteRights common.Fixed64) error {

	if !t.parameters.BlockChain.GetCRCommittee().IsInVotingPeriod(blockHeight) {
		return errors.New("cr vote tx must during voting period")
	}

	var totalVotes common.Fixed64
	for _, cv := range content.VotesInfo {
		if cv.LockTime != 0 {
			return errors.New("votes lock time need to be zero")
		}
		cid, err := common.Uint168FromBytes(cv.Candidate)
		if err != nil {
			return fmt.Errorf("invalid vote output payload " +
				"Candidate can not change to proper cid")
		}
		if _, ok := crs[*cid]; !ok {
			cidAddress, _ := cid.ToAddress()
			return fmt.Errorf("invalid vote output payload "+
				"CR candidate: %s", cidAddress)
		}
		totalVotes += cv.Votes
	}
	if totalVotes > voteRights {
		return errors.New("CR vote rights not enough")
	}

	return nil
}

func (t *VotingTransaction) checkVoteCRCProposalContent(
	content payload.VotesContent, voteRights common.Fixed64) error {
	var maxVotes common.Fixed64
	for _, cv := range content.VotesInfo {
		if cv.LockTime != 0 {
			return errors.New("votes lock time need to be zero")
		}
		if maxVotes < cv.Votes {
			maxVotes = cv.Votes
		}
		proposalHash, err := common.Uint256FromBytes(cv.Candidate)
		if err != nil {
			return err
		}
		proposal := t.parameters.BlockChain.GetCRCommittee().GetProposal(*proposalHash)
		if proposal == nil || proposal.Status != crstate.CRAgreed {
			return fmt.Errorf("invalid CRCProposal: %s",
				common.ToReversedString(*proposalHash))
		}
	}

	if maxVotes > voteRights {
		return errors.New("CRCProposal vote rights not enough")
	}

	return nil
}

func (t *VotingTransaction) checkCRImpeachmentContent(
	content payload.VotesContent, voteRights common.Fixed64) error {
	crMembersMap := getCRMembersMap(t.parameters.BlockChain.GetCRCommittee().GetImpeachableMembers())
	var totalVotes common.Fixed64
	for _, cv := range content.VotesInfo {
		if cv.LockTime != 0 {
			return errors.New("votes lock time need to be zero")
		}
		crState, ok := crMembersMap[common.BytesToHexString(cv.Candidate)]
		if !ok {
			return errors.New("candidate should be one of the CR members")
		}
		if crState == crstate.MemberImpeached ||
			crState == crstate.MemberTerminated ||
			crState == crstate.MemberReturned {
			return errors.New("CR member state is wrong")
		}
		totalVotes += cv.Votes
	}

	if totalVotes > voteRights {
		return errors.New("CRImpeachment vote rights not enough")
	}

	return nil
}

func (t *VotingTransaction) checkDPoSV2Content(content payload.VotesContent,
	pds map[string]uint32, voteRights common.Fixed64) error {
	// totalVotes should be more than output value
	var totalVotes common.Fixed64
	for _, cv := range content.VotesInfo {
		lockUntil, ok := pds[common.BytesToHexString(cv.Candidate)]
		if !ok {
			return fmt.Errorf("invalid vote output payload "+
				"producer candidate: %s", common.BytesToHexString(cv.Candidate))
		}
		lockTime := cv.LockTime - t.parameters.BlockHeight
		if cv.LockTime <= t.parameters.BlockHeight || cv.LockTime > lockUntil ||
			lockTime < t.parameters.Config.DPoSV2MinVotesLockTime ||
			lockTime > t.parameters.Config.DPoSV2MaxVotesLockTime {

			return errors.New("invalid DPoS 2.0 votes lock time")
		}
		totalVotes += cv.Votes
	}
	if totalVotes > voteRights {
		return errors.New("DPoSV2 vote rights not enough")
	}

	return nil
}
