// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/database"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type CRCProposalReviewTransaction struct {
	BaseTransaction
}

func (t *CRCProposalReviewTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.CRCProposalReview:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *CRCProposalReviewTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *CRCProposalReviewTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.CRCommitteeStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before CRCommitteeStartHeight", t.TxType().Name()))
	} else if blockHeight < chainParams.CRCProposalDraftDataStartHeight {
		if t.PayloadVersion() != payload.CRCProposalVersion {
			return errors.New("payload version should be CRCProposalVersion")
		}
	} else {
		if t.PayloadVersion() != payload.CRCProposalVersion01 {
			return errors.New("should have draft data")
		}
	}
	return nil
}

func (t *CRCProposalReviewTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {
	crcProposalReview, ok := t.Payload().(*payload.CRCProposalReview)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}
	// Check if the proposal exist.
	proposalState := t.parameters.BlockChain.GetCRCommittee().GetProposal(crcProposalReview.ProposalHash)
	if proposalState == nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("proposal not exist")), true
	}
	if proposalState.Status != crstate.Registered {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("proposal status is not Registered")), true
	}

	if crcProposalReview.VoteResult < payload.Approve ||
		(crcProposalReview.VoteResult > payload.Abstain) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("VoteResult should be known")), true
	}
	crMember := t.parameters.BlockChain.GetCRCommittee().GetMember(crcProposalReview.DID)
	if crMember == nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("did correspond crMember not exists")), true
	}
	if crMember.MemberState != crstate.MemberElected {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("should be an elected CR members")), true
	}
	exist := t.parameters.BlockChain.GetCRCommittee().ExistProposal(crcProposalReview.
		ProposalHash)
	if !exist {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("ProposalHash must exist")), true
	}

	// check opinion data.
	if t.PayloadVersion() >= payload.CRCProposalReviewVersion01 {
		if len(crcProposalReview.OpinionData) >= payload.MaxOpinionDataSize {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("the opinion data cannot be more than 1M byte")), true
		}
		tempDraftHash := common.Hash(crcProposalReview.OpinionData)
		if !crcProposalReview.OpinionHash.IsEqual(tempDraftHash) {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("the opinion data and opinion hash of"+
				" proposal review are inconsistent")), true
		}
	}

	// check signature.
	signedBuf := new(bytes.Buffer)
	err := crcProposalReview.SerializeUnsigned(signedBuf, t.PayloadVersion())
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}
	err = blockchain.CheckCRTransactionSignature(crcProposalReview.Signature, crMember.Info.Code,
		signedBuf.Bytes())
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}
	return nil, false
}

func (t *CRCProposalReviewTransaction) GetSaveProcessor() (database.TXProcessor, elaerr.ELAError) {
	proposalReview := t.Payload().(*payload.CRCProposalReview)
	return func(dbTx database.Tx) error {
		err := blockchain.TryCreateBucket(dbTx, common.ProposalDraftDataBucketName)
		if err != nil {
			return err
		}

		return blockchain.DBPutData(dbTx, common.ProposalDraftDataBucketName,
			proposalReview.OpinionHash[:], proposalReview.OpinionData)
	}, nil
}

func (t *CRCProposalReviewTransaction) GetRollbackProcessor() (database.TXProcessor, elaerr.ELAError) {
	proposalReview := t.Payload().(*payload.CRCProposalReview)
	return func(dbTx database.Tx) error {
		return blockchain.DBRemoveData(dbTx, common.ProposalDraftDataBucketName, proposalReview.OpinionHash[:])
	}, nil
}
