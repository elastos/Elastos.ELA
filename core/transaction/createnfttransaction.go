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
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

const CanNotCreateNFTHeight = 6

type CreateNFTTransaction struct {
	BaseTransaction
}

func (t *CreateNFTTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.CreateNFT:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *CreateNFTTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *CreateNFTTransaction) CheckAttributeProgram() error {
	// Check attributes
	for _, attr := range t.Attributes() {
		if !common2.IsValidAttributeType(attr.Usage) {
			return fmt.Errorf("invalid attribute usage %v", attr.Usage)
		}
	}

	// Check programs
	if len(t.Programs()) != 1 {
		return fmt.Errorf("need to be only one program")
	}
	for _, p := range t.Programs() {
		if p.Code == nil {
			return fmt.Errorf("invalid program code nil")
		}
		if len(p.Code) < program.MinProgramCodeSize {
			return fmt.Errorf("invalid program code size")
		}
		if p.Parameter == nil {
			return fmt.Errorf("invalid program parameter nil")
		}
	}

	return nil
}

func (t *CreateNFTTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.DPoSConfiguration.NFTStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before NFTStartHeight", t.TxType().Name()))
	}
	if blockHeight < chainParams.DPoSConfiguration.NFTV2StartHeight &&
		t.payloadVersion >= payload.CreateNFTVersion2 {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before NFTV2StartHeight", t.TxType().Name()))
	}
	if blockHeight >= chainParams.DPoSConfiguration.NFTV2StartHeight &&
		t.payloadVersion != payload.CreateNFTVersion2 {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"with payload version 0 after NFTV2StartHeight", t.TxType().Name()))
	}

	return nil
}

func (t *CreateNFTTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
	pld, ok := t.Payload().(*payload.CreateNFT)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	state := t.parameters.BlockChain.GetState()
	crCommittee := t.parameters.BlockChain.GetCRCommittee()

	producers := state.GetDposV2Producers()
	nftID := common.GetNFTID(pld.ReferKey, t.hash())
	var existVote bool
	var nftAmount common.Fixed64
	var votesStakeAddress common.Uint168
	var detailedVotes payload.DetailedVoteInfo
	for _, p := range producers {
		for stakeAddress, votesInfo := range p.GetAllDetailedDPoSV2Votes() {
			for referKey, voteInfo := range votesInfo {
				if referKey.IsEqual(pld.ReferKey) {
					ct, _ := contract.CreateStakeContractByCode(nftID.Bytes())
					nftStakeAddress := ct.ToProgramHash()
					if stakeAddress.IsEqual(*nftStakeAddress) {
						return elaerr.Simple(elaerr.ErrTxPayload,
							errors.New("the NFT has been created yet")), true
					}
					if t.parameters.BlockHeight >= voteInfo.Info[0].LockTime-CanNotCreateNFTHeight {
						return elaerr.Simple(elaerr.ErrTxPayload,
							errors.New("vote is almost expired")), true
					}
					log.Info("create NFT, vote information:", voteInfo)
					existVote = true
					nftAmount = voteInfo.Info[0].Votes
					votesStakeAddress = stakeAddress
					detailedVotes = voteInfo
				}
			}
		}
	}

	if !existVote {
		return elaerr.Simple(elaerr.ErrTxPayload,
			errors.New("has no DPoS 2.0 votes equal to the NFT ID")), true
	}

	// stake address need to be same from code
	ct, _ := contract.CreateStakeContractByCode(t.programs[0].Code)
	stakeProgramHash := ct.ToProgramHash()
	stakeAddress, err := stakeProgramHash.ToAddress()
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid stake address")), true
	}
	if stakeAddress != pld.StakeAddress {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("stake address not from code")), true
	}
	if !votesStakeAddress.IsEqual(*stakeProgramHash) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid stake address from NFT ID")), true
	}

	// nft has not been created before
	if nftInfo := state.GetNFTInfo(nftID); nftInfo != nil {
		log.Warnf("NFT has been create before, side chain genesis block "+
			"hash: %s", nftInfo.GenesisBlockHash.String())
		return elaerr.Simple(elaerr.ErrTxPayload,
			errors.New("NFT has been created before")), true
	}

	// check the vote rights is enough or not
	totalVoteRights := state.GetDposV2VoteRights(*stakeProgramHash)
	var usedCRVotes common.Fixed64
	if ucv := crCommittee.GetUsedCRVotes(*stakeProgramHash); ucv != nil {
		for _, v := range ucv {
			usedCRVotes += v.Votes
		}
	}
	var usedCRImpeachmentVotes common.Fixed64
	if ucv := crCommittee.GetUsedCRImpeachmentVotes(*stakeProgramHash); ucv != nil {
		for _, v := range ucv {
			usedCRImpeachmentVotes += v.Votes
		}
	}
	var usedCRProposalVotes common.Fixed64
	if ucv := crCommittee.GetUsedCRCProposalVotes(*stakeProgramHash); ucv != nil {
		for _, v := range ucv {
			if usedCRProposalVotes < v.Votes {
				usedCRProposalVotes = v.Votes
			}
		}
	}
	var usedDPoSVotes common.Fixed64
	if udv, ok := state.GetUsedDposVotes(*stakeProgramHash); ok {
		for _, v := range udv {
			if usedDPoSVotes < v.Votes {
				usedDPoSVotes = v.Votes
			}
		}
	}

	blockHeight := t.parameters.BlockHeight
	if blockHeight < state.GetDPoSV2ActiveHeight() {
		if nftAmount > totalVoteRights-usedDPoSVotes {
			log.Errorf("vote rights is not enough, nft amount:%s, "+
				"total vote rights:%s, used DPoS 1.0 votes:%s",
				nftAmount, totalVoteRights, usedDPoSVotes)
			return elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("vote rights is not enough")), true
		}
	}

	if nftAmount > totalVoteRights-usedCRVotes ||
		nftAmount > totalVoteRights-usedCRImpeachmentVotes ||
		nftAmount > totalVoteRights-usedCRProposalVotes {
		log.Errorf("vote rights is not enough, nft amount:%s, "+
			"total vote rights:%s, used CR votes:%s, "+
			"used CR impeachment votes:%s, used CR proposal votes:%s",
			nftAmount, totalVoteRights, usedCRVotes,
			usedCRImpeachmentVotes, usedCRProposalVotes)
		return elaerr.Simple(elaerr.ErrTxPayload,
			errors.New("vote rights is not enough")), true
	}

	if t.payloadVersion == payload.CreateNFTVersion2 {
		if detailedVotes.BlockHeight != pld.StartHeight {
			return elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("invalid StartHeight")), true
		}
		if detailedVotes.Info[0].LockTime != pld.EndHeight {
			return elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("invalid EndHeight")), true
		}
		if nftAmount != pld.Votes {
			return elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("invalid Votes")), true
		}
		if detailedVotes.VoteRights() != pld.VoteRights {
			return elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("invalid VoteRights")), true
		}
		if !bytes.Equal(detailedVotes.Info[0].Candidate, pld.TargetOwnerKey) {
			return elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("invalid TargetOwnerKey")), true
		}
	} else if t.payloadVersion > payload.CreateNFTVersion2 {
		return elaerr.Simple(elaerr.ErrTxPayload,
			errors.New("invalid payload version")), true
	}
	return nil, false
}
