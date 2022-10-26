// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/database"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type CRCProposalTrackingTransaction struct {
	BaseTransaction
}

func (t *CRCProposalTrackingTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.CRCProposalTracking:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *CRCProposalTrackingTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *CRCProposalTrackingTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.CRConfiguration.CRCommitteeStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before CRCommitteeStartHeight", t.TxType().Name()))
	} else if blockHeight < chainParams.CRConfiguration.CRCProposalDraftDataStartHeight {
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

func (t *CRCProposalTrackingTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {
	cptPayload, ok := t.Payload().(*payload.CRCProposalTracking)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	// Check if proposal exist.
	proposalState := t.parameters.BlockChain.GetCRCommittee().GetProposal(cptPayload.ProposalHash)
	if proposalState == nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("proposal not exist")), true
	}
	if proposalState.Status != crstate.VoterAgreed {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("proposal status is not VoterAgreed")), true
	}
	// Check proposal tracking count.
	if proposalState.TrackingCount >= t.parameters.Config.CRConfiguration.MaxProposalTrackingCount {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("reached max tracking count")), true
	}

	// check message data.
	if t.PayloadVersion() >= payload.CRCProposalTrackingVersion01 {
		if len(cptPayload.MessageData) >= payload.MaxMessageDataSize {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("the message data cannot be more than 800K byte")), true
		}
		tempMessageHash := common.Hash(cptPayload.MessageData)
		if !cptPayload.MessageHash.IsEqual(tempMessageHash) {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("the message data and message hash of"+
				" proposal tracking are inconsistent")), true
		}
		if len(cptPayload.SecretaryGeneralOpinionData) >= payload.MaxSecretaryGeneralOpinionDataSize {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("the opinion data cannot be more than 200K byte")), true
		}
		tempOpinionHash := common.Hash(cptPayload.SecretaryGeneralOpinionData)
		if !cptPayload.SecretaryGeneralOpinionHash.IsEqual(tempOpinionHash) {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("the opinion data and opinion hash of"+
				" proposal tracking are inconsistent")), true
		}
	}

	switch cptPayload.ProposalTrackingType {
	case payload.Common:
		err := t.checkCRCProposalCommonTracking(
			t.parameters, cptPayload, proposalState, t.PayloadVersion())
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
	case payload.Progress:
		err := t.checkCRCProposalProgressTracking(
			t.parameters, cptPayload, proposalState, t.PayloadVersion())
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
	case payload.Rejected:
		err := t.checkCRCProposalRejectedTracking(
			t.parameters, cptPayload, proposalState, t.parameters.BlockHeight, t.PayloadVersion())
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
	case payload.Terminated:
		err := t.checkCRCProposalTerminatedTracking(
			t.parameters, cptPayload, proposalState, t.PayloadVersion())
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
	case payload.ChangeOwner:
		err := t.checkCRCProposalOwnerTracking(
			t.parameters, cptPayload, proposalState, t.PayloadVersion())
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
	case payload.Finalized:
		err := t.checkCRCProposalFinalizedTracking(
			t.parameters, cptPayload, proposalState, t.PayloadVersion())
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
	default:
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid proposal tracking type")), true
	}

	return nil, false
}

func (t *CRCProposalTrackingTransaction) checkCRCProposalCommonTracking(
	params *TransactionParameters, cptPayload *payload.CRCProposalTracking, pState *crstate.ProposalState,
	payloadVersion byte) error {
	// Check stage of proposal
	if cptPayload.Stage != 0 {
		return errors.New("stage should assignment zero value")
	}

	// Check signature.
	return t.normalCheckCRCProposalTrackingSignature(t.parameters, cptPayload, pState, payloadVersion)
}

func (t *CRCProposalTrackingTransaction) normalCheckCRCProposalTrackingSignature(
	params *TransactionParameters, cptPayload *payload.CRCProposalTracking, pState *crstate.ProposalState,
	payloadVersion byte) error {
	// Check new owner public key.
	if len(cptPayload.NewOwnerPublicKey) != 0 {
		return errors.New("the NewOwnerPublicKey need to be empty")
	}

	// Check signature of proposal owner.
	if !bytes.Equal(pState.ProposalOwner, cptPayload.OwnerPublicKey) {
		return errors.New("the OwnerPublicKey is not owner of proposal")
	}
	signedBuf := new(bytes.Buffer)
	if err := checkProposalOwnerSignature(cptPayload,
		cptPayload.OwnerPublicKey, signedBuf, payloadVersion); err != nil {
		return err
	}

	// Check new owner signature.
	if len(cptPayload.NewOwnerSignature) != 0 {
		return errors.New("the NewOwnerSignature need to be empty")
	}

	// Write new owner signature.
	err := common.WriteVarBytes(signedBuf, cptPayload.NewOwnerSignature)
	if err != nil {
		return errors.New("failed to write NewOwnerSignature")
	}

	// Check secretary general signature。
	return t.checkSecretaryGeneralSignature(t.parameters, cptPayload, pState, signedBuf, payloadVersion)
}

func checkProposalOwnerSignature(
	cptPayload *payload.CRCProposalTracking, pubKey []byte,
	signedBuf *bytes.Buffer, payloadVersion byte) error {
	publicKey, err := crypto.DecodePoint(pubKey)
	if err != nil {
		return errors.New("invalid proposal owner")
	}
	lContract, err := contract.CreateStandardContract(publicKey)
	if err != nil {
		return errors.New("invalid proposal owner publicKey")
	}
	if err = cptPayload.SerializeUnsigned(signedBuf, payloadVersion); err != nil {
		return err
	}
	if err := blockchain.CheckCRTransactionSignature(cptPayload.OwnerSignature, lContract.Code,
		signedBuf.Bytes()); err != nil {
		return errors.New("proposal owner signature check failed")
	}

	return common.WriteVarBytes(signedBuf, cptPayload.OwnerSignature)
}

func (t *CRCProposalTrackingTransaction) checkSecretaryGeneralSignature(
	params *TransactionParameters, cptPayload *payload.CRCProposalTracking, pState *crstate.ProposalState,
	signedBuf *bytes.Buffer, payloadVersion byte) error {
	var sgContract *contract.Contract
	publicKeyBytes, err := hex.DecodeString(t.parameters.BlockChain.GetCRCommittee().GetProposalManager().SecretaryGeneralPublicKey)
	if err != nil {
		return errors.New("invalid secretary general public key")
	}
	publicKey, err := crypto.DecodePoint(publicKeyBytes)
	if err != nil {
		return errors.New("invalid proposal secretary general public key")
	}
	sgContract, err = contract.CreateStandardContract(publicKey)
	if err != nil {
		return errors.New("invalid secretary general public key")
	}
	if _, err := signedBuf.Write([]byte{byte(cptPayload.ProposalTrackingType)}); err != nil {
		return errors.New("invalid ProposalTrackingType")
	}
	if err := cptPayload.SecretaryGeneralOpinionHash.Serialize(signedBuf); err != nil {
		return errors.New("invalid secretary opinion hash")
	}
	if payloadVersion >= payload.CRCProposalTrackingVersion01 {
		if err := common.WriteVarBytes(signedBuf, cptPayload.SecretaryGeneralOpinionData); err != nil {
			return errors.New("invalid secretary-general opinion data")
		}
	}
	if err = blockchain.CheckCRTransactionSignature(cptPayload.SecretaryGeneralSignature,
		sgContract.Code, signedBuf.Bytes()); err != nil {
		return errors.New("secretary general signature check failed")
	}

	return nil
}

func checkProposalNewOwnerSignature(
	cptPayload *payload.CRCProposalTracking, pubKey []byte,
	signedBuf *bytes.Buffer) error {
	publicKey, err := crypto.DecodePoint(pubKey)
	if err != nil {
		return errors.New("invalid new proposal owner public key")
	}
	lContract, err := contract.CreateStandardContract(publicKey)
	if err != nil {
		return errors.New("invalid new proposal owner publicKey")
	}
	if err := blockchain.CheckCRTransactionSignature(cptPayload.NewOwnerSignature, lContract.Code,
		signedBuf.Bytes()); err != nil {
		return errors.New("new proposal owner signature check failed")
	}

	return common.WriteVarBytes(signedBuf, cptPayload.NewOwnerSignature)
}

func (t *CRCProposalTrackingTransaction) checkCRCProposalProgressTracking(
	params *TransactionParameters, cptPayload *payload.CRCProposalTracking, pState *crstate.ProposalState,
	payloadVersion byte) error {
	// Check stage of proposal
	if int(cptPayload.Stage) >= len(pState.Proposal.Budgets) {
		return errors.New("invalid tracking Stage")
	}
	if _, ok := pState.WithdrawableBudgets[cptPayload.Stage]; ok {
		return errors.New("invalid budgets with tracking budget")
	}

	for _, budget := range pState.Proposal.Budgets {
		if cptPayload.Stage == budget.Stage {
			if budget.Type == payload.Imprest ||
				budget.Type == payload.FinalPayment {
				return errors.New("imprest and final payment not allowed to withdraw")
			}
		}
	}

	// Check signature.
	return t.normalCheckCRCProposalTrackingSignature(t.parameters, cptPayload, pState, payloadVersion)
}

func (t *CRCProposalTrackingTransaction) checkCRCProposalRejectedTracking(
	params *TransactionParameters, cptPayload *payload.CRCProposalTracking, pState *crstate.ProposalState,
	blockHeight uint32, payloadVersion byte) error {
	if blockHeight < t.parameters.Config.CRConfiguration.CRCProposalWithdrawPayloadV1Height {
		return t.checkCRCProposalProgressTracking(t.parameters, cptPayload, pState, payloadVersion)
	}
	// Check stage of proposal
	if int(cptPayload.Stage) >= len(pState.Proposal.Budgets) {
		return errors.New("invalid tracking Stage")
	}
	if _, ok := pState.WithdrawableBudgets[cptPayload.Stage]; ok {
		return errors.New("invalid budgets with tracking budget")
	}

	// Check signature.
	return t.normalCheckCRCProposalTrackingSignature(t.parameters, cptPayload, pState, payloadVersion)
}

func (t *CRCProposalTrackingTransaction) checkCRCProposalTerminatedTracking(
	params *TransactionParameters, cptPayload *payload.CRCProposalTracking, pState *crstate.ProposalState,
	payloadVersion byte) error {
	// Check stage of proposal
	if cptPayload.Stage != 0 {
		return errors.New("stage should assignment zero value")
	}

	// Check signature.
	return t.normalCheckCRCProposalTrackingSignature(t.parameters, cptPayload, pState, payloadVersion)
}

func (t *CRCProposalTrackingTransaction) checkCRCProposalOwnerTracking(
	params *TransactionParameters, cptPayload *payload.CRCProposalTracking, pState *crstate.ProposalState,
	payloadVersion byte) error {
	// Check stage of proposal
	if cptPayload.Stage != 0 {
		return errors.New("stage should assignment zero value")
	}

	// Check new owner public.
	if bytes.Equal(pState.ProposalOwner, cptPayload.NewOwnerPublicKey) {
		return errors.New("invalid new owner public key")
	}

	// Check signature.
	return t.checkCRCProposalTrackingSignature(t.parameters, cptPayload, pState, payloadVersion)
}

func (t *CRCProposalTrackingTransaction) checkCRCProposalTrackingSignature(
	params *TransactionParameters, cptPayload *payload.CRCProposalTracking, pState *crstate.ProposalState,
	payloadVersion byte) error {
	// Check signature of proposal owner.
	if !bytes.Equal(pState.ProposalOwner, cptPayload.OwnerPublicKey) {
		return errors.New("the OwnerPublicKey is not owner of proposal")
	}
	signedBuf := new(bytes.Buffer)
	if err := checkProposalOwnerSignature(cptPayload,
		cptPayload.OwnerPublicKey, signedBuf, payloadVersion); err != nil {
		return err
	}

	// Check other new owner signature.
	if err := checkProposalNewOwnerSignature(cptPayload,
		cptPayload.NewOwnerPublicKey, signedBuf); err != nil {
		return err
	}

	// Check secretary general signature。
	return t.checkSecretaryGeneralSignature(t.parameters, cptPayload, pState, signedBuf, payloadVersion)
}

func (t *CRCProposalTrackingTransaction) checkCRCProposalFinalizedTracking(
	params *TransactionParameters, cptPayload *payload.CRCProposalTracking, pState *crstate.ProposalState,
	payloadVersion byte) error {
	// Check stage of proposal
	var finalStage byte
	for _, budget := range pState.Proposal.Budgets {
		if budget.Type == payload.FinalPayment {
			finalStage = budget.Stage
		}
	}

	if cptPayload.Stage != finalStage {
		return errors.New("cptPayload.Stage is not proposal final stage")
	}

	// Check signature.
	return t.normalCheckCRCProposalTrackingSignature(t.parameters, cptPayload, pState, payloadVersion)
}

func (t *CRCProposalTrackingTransaction) GetSaveProcessor() (database.TXProcessor, elaerr.ELAError) {
	proposalTracking := t.Payload().(*payload.CRCProposalTracking)

	return func(dbTx database.Tx) error {
		err := blockchain.TryCreateBucket(dbTx, common.ProposalDraftDataBucketName)
		if err != nil {
			return err
		}

		err = blockchain.DBPutData(dbTx, common.ProposalDraftDataBucketName,
			proposalTracking.SecretaryGeneralOpinionHash[:],
			proposalTracking.SecretaryGeneralOpinionData)
		if err != nil {
			return err
		}

		return blockchain.DBPutData(dbTx, common.ProposalDraftDataBucketName,
			proposalTracking.MessageHash[:], proposalTracking.MessageData)
	}, nil
}

func (t *CRCProposalTrackingTransaction) GetRollbackProcessor() (database.TXProcessor, elaerr.ELAError) {
	proposalTracking := t.Payload().(*payload.CRCProposalTracking)
	return func(dbTx database.Tx) error {
		err := blockchain.DBRemoveData(dbTx, common.ProposalDraftDataBucketName,
			proposalTracking.SecretaryGeneralOpinionHash[:])
		if err != nil {
			return err
		}
		return blockchain.DBRemoveData(dbTx, common.ProposalDraftDataBucketName,
			proposalTracking.MessageHash[:])
	}, nil
}
