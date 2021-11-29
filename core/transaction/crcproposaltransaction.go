// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
	elaerr "github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/utils"
)

type CRCProposalTransaction struct {
	BaseTransaction
}

func (t *CRCProposalTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.CRCProposal:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *CRCProposalTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *CRCProposalTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.CRCProposalDraftDataStartHeight {
		if t.PayloadVersion() != payload.CRCProposalVersion {
			return errors.New("payload version should be CRCProposalVersion")
		}
	} else {
		if t.PayloadVersion() != payload.CRCProposalVersion01 {
			return errors.New("should have draft data")
		}
	}

	p, ok := t.Payload().(*payload.CRCProposal)
	if !ok {
		return errors.New("not support invalid CRCProposal transaction")
	}
	switch p.ProposalType {
	case payload.ChangeProposalOwner, payload.CloseProposal, payload.SecretaryGeneral:
		if blockHeight < chainParams.CRCProposalV1Height {
			return errors.New(fmt.Sprintf("not support %s CRCProposal"+
				" transactio before CRCProposalV1Height", p.ProposalType.Name()))
		}
	case payload.ReserveCustomID, payload.ReceiveCustomID, payload.ChangeCustomIDFee:
		if blockHeight < chainParams.CustomIDProposalStartHeight {
			return errors.New(fmt.Sprintf("not support %s CRCProposal"+
				" transaction before CustomIDProposalStartHeight", p.ProposalType.Name()))
		}
	case payload.RegisterSideChain:
		if blockHeight < chainParams.NewCrossChainStartHeight {
			return errors.New(fmt.Sprintf("not support %s CRCProposal"+
				" transaction before NewCrossChainStartHeight", p.ProposalType.Name()))
		}
	default:
		if blockHeight < chainParams.CRCommitteeStartHeight {
			return errors.New(fmt.Sprintf("not support %s CRCProposal"+
				" transaction before CRCommitteeStartHeight", p.ProposalType.Name()))
		}
	}

	return nil
}

func (t *CRCProposalTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {
	proposal, ok := t.Payload().(*payload.CRCProposal)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}
	// The number of the proposals of the committee can not more than 128
	if t.parameters.BlockChain.GetCRCommittee().IsProposalFull(proposal.CRCouncilMemberDID) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("proposal is full")), true
	}
	// Check draft hash of proposal.
	if t.parameters.BlockChain.GetCRCommittee().ExistDraft(proposal.DraftHash) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("duplicated draft proposal hash")), true
	}

	if !t.parameters.BlockChain.GetCRCommittee().IsProposalAllowed(t.parameters.BlockHeight - 1) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("cr proposal tx must not during voting period")), true
	}
	if len(proposal.CategoryData) > blockchain.MaxCategoryDataStringLength {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("the Proposal category data cannot be more than 4096 characters")), true
	}

	if t.PayloadVersion() >= payload.CRCProposalVersion01 {
		if len(proposal.DraftData) >= payload.MaxProposalDataSize {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("the Proposal draft data cannot be more than 1000000 byte")), true
		}
		tempDraftHash := common.Hash(proposal.DraftData)
		if !proposal.DraftHash.IsEqual(tempDraftHash) {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("the  draft data and draft hash of proposal are  inconsistent\n \n  ")), true
		}
	}

	if len(proposal.Budgets) > blockchain.MaxBudgetsCount {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("budgets exceeded the maximum limit")), true
	}
	// Check type of proposal.
	if proposal.ProposalType.Name() == "Unknown" {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("type of proposal should be known")), true
	}
	//CRCouncilMemberDID must MemberElected cr member
	// Check CR Council Member DID of proposal.
	crMember := t.parameters.BlockChain.GetCRCommittee().GetMember(proposal.CRCouncilMemberDID)
	if crMember == nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("CR Council Member should be one of the CR members")), true
	}
	if crMember.MemberState != crstate.MemberElected {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("CR Council Member should be an elected CR members")), true
	}
	switch proposal.ProposalType {
	case payload.ChangeProposalOwner:
		err := t.checkChangeProposalOwner(t.parameters, proposal, t.PayloadVersion())
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
	case payload.CloseProposal:
		err := t.checkCloseProposal(t.parameters, proposal, t.PayloadVersion())
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
	case payload.SecretaryGeneral:
		err := t.checkChangeSecretaryGeneralProposalTx(t.parameters, proposal, t.PayloadVersion())
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
	case payload.ReserveCustomID:
		err := t.checkReservedCustomID(t.parameters, proposal, t.PayloadVersion())
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
	case payload.ReceiveCustomID:
		err := t.checkReceivedCustomID(t.parameters, proposal, t.PayloadVersion())
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
	case payload.ChangeCustomIDFee:
		err := t.checkChangeCustomIDFee(t.parameters, proposal, t.PayloadVersion())
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
	case payload.RegisterSideChain:
		err := t.checkRegisterSideChainProposal(t.parameters, proposal, t.PayloadVersion())
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
	default:
		err := t.checkNormalOrELIPProposal(t.parameters, proposal, t.parameters.ProposalsUsedAmount, t.PayloadVersion())
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
	}
	return nil, false
}

func (t *CRCProposalTransaction) checkChangeProposalOwner(params *TransactionParameters, proposal *payload.CRCProposal, PayloadVersion byte) error {
	proposalState := t.parameters.BlockChain.GetCRCommittee().GetProposal(proposal.TargetProposalHash)
	if proposalState == nil {
		return errors.New("proposal doesn't exist")
	}
	if proposalState.Status != crstate.VoterAgreed {
		return errors.New("proposal status is not VoterAgreed")
	}

	if _, err := crypto.DecodePoint(proposal.OwnerPublicKey); err != nil {
		return errors.New("invalid owner public key")
	}

	if _, err := crypto.DecodePoint(proposal.NewOwnerPublicKey); err != nil {
		return errors.New("invalid new owner public key")
	}

	if bytes.Equal(proposal.NewOwnerPublicKey, proposalState.ProposalOwner) &&
		proposal.NewRecipient.IsEqual(proposalState.Recipient) {
		return errors.New("new owner or recipient must be different from the previous one")
	}

	crCouncilMember := t.parameters.BlockChain.GetCRCommittee().GetMember(proposal.CRCouncilMemberDID)
	if crCouncilMember == nil {
		return errors.New("CR Council Member should be one of the CR members")
	}
	return t.checkChangeOwnerSign(proposal, crCouncilMember.Info.Code, PayloadVersion)
}

func (t *CRCProposalTransaction) checkChangeOwnerSign(proposal *payload.CRCProposal, crMemberCode []byte,
	PayloadVersion byte) error {
	signedBuf := new(bytes.Buffer)
	err := proposal.SerializeUnsigned(signedBuf, PayloadVersion)
	if err != nil {
		return err
	}

	// Check signature of owner.
	publicKey, err := crypto.DecodePoint(proposal.OwnerPublicKey)
	if err != nil {
		return errors.New("invalid owner")
	}
	ownerContract, err := contract.CreateStandardContract(publicKey)
	if err != nil {
		return errors.New("invalid owner")
	}
	if err := blockchain.CheckCRTransactionSignature(proposal.Signature, ownerContract.Code,
		signedBuf.Bytes()); err != nil {
		return errors.New("owner signature check failed")
	}

	// Check signature of new owner.
	newOwnerPublicKey, err := crypto.DecodePoint(proposal.NewOwnerPublicKey)
	if err != nil {
		return errors.New("invalid owner")
	}
	newOwnerContract, err := contract.CreateStandardContract(newOwnerPublicKey)
	if err != nil {
		return errors.New("invalid owner")
	}

	if err := blockchain.CheckCRTransactionSignature(proposal.NewOwnerSignature, newOwnerContract.Code,
		signedBuf.Bytes()); err != nil {
		return errors.New("new owner signature check failed")
	}

	// Check signature of CR Council Member.
	if err = common.WriteVarBytes(signedBuf, proposal.Signature); err != nil {
		return errors.New("failed to write proposal owner signature")
	}
	if err = common.WriteVarBytes(signedBuf, proposal.NewOwnerSignature); err != nil {
		return errors.New("failed to write proposal new owner signature")
	}
	if err = proposal.CRCouncilMemberDID.Serialize(signedBuf); err != nil {
		return errors.New("failed to write CR Council Member's DID")
	}
	if err = blockchain.CheckCRTransactionSignature(proposal.CRCouncilMemberSignature, crMemberCode,
		signedBuf.Bytes()); err != nil {
		return errors.New("failed to check CR Council Member signature")
	}

	return nil
}

func (t *CRCProposalTransaction) checkCloseProposal(params *TransactionParameters, proposal *payload.CRCProposal, PayloadVersion byte) error {
	_, err := crypto.DecodePoint(proposal.OwnerPublicKey)
	if err != nil {
		return errors.New("DecodePoint from OwnerPublicKey error")
	}
	if ps := t.parameters.BlockChain.GetCRCommittee().GetProposal(proposal.TargetProposalHash); ps == nil {
		return errors.New("CloseProposalHash does not exist")
	} else if ps.Status != crstate.VoterAgreed {
		return errors.New("CloseProposalHash has to be voterAgreed")
	}
	if len(proposal.Budgets) > 0 {
		return errors.New("CloseProposal cannot have budget")
	}
	emptyUint168 := common.Uint168{}
	if proposal.Recipient != emptyUint168 {
		return errors.New("CloseProposal recipient must be empty")
	}
	crMember := t.parameters.BlockChain.GetCRCommittee().GetMember(proposal.CRCouncilMemberDID)
	if crMember == nil {
		return errors.New("CR Council Member should be one of the CR members")
	}
	return t.checkOwnerAndCRCouncilMemberSign(proposal, crMember.Info.Code, PayloadVersion)
}

func (t *CRCProposalTransaction) checkOwnerAndCRCouncilMemberSign(proposal *payload.CRCProposal, crMemberCode []byte,
	PayloadVersion byte) error {
	// Check signature of owner.
	publicKey, err := crypto.DecodePoint(proposal.OwnerPublicKey)
	if err != nil {
		return errors.New("invalid owner")
	}
	contract, err := contract.CreateStandardContract(publicKey)
	if err != nil {
		return errors.New("invalid owner")
	}
	signedBuf := new(bytes.Buffer)
	err = proposal.SerializeUnsigned(signedBuf, PayloadVersion)
	if err != nil {
		return err
	}
	if err := blockchain.CheckCRTransactionSignature(proposal.Signature, contract.Code,
		signedBuf.Bytes()); err != nil {
		return errors.New("owner signature check failed")
	}

	// Check signature of CR Council Member.
	if err = common.WriteVarBytes(signedBuf, proposal.Signature); err != nil {
		return errors.New("failed to write proposal owner signature")
	}
	if err = proposal.CRCouncilMemberDID.Serialize(signedBuf); err != nil {
		return errors.New("failed to write CR Council Member's DID")
	}
	if err = blockchain.CheckCRTransactionSignature(proposal.CRCouncilMemberSignature, crMemberCode,
		signedBuf.Bytes()); err != nil {
		return errors.New("failed to check CR Council Member signature")
	}
	return nil
}

func (t *CRCProposalTransaction) checkChangeSecretaryGeneralProposalTx(params *TransactionParameters, crcProposal *payload.CRCProposal, PayloadVersion byte) error {
	// The number of the proposals of the committee can not more than 128
	if !isPublicKeyDIDMatch(crcProposal.SecretaryGeneralPublicKey, &crcProposal.SecretaryGeneralDID) {
		return errors.New("SecretaryGeneral NodePublicKey and DID is not matching")
	}
	// Check owner public key
	if _, err := crypto.DecodePoint(crcProposal.OwnerPublicKey); err != nil {
		return errors.New("invalid owner public key")
	}

	//CRCouncilMemberDID must MemberElected cr member
	if !t.parameters.BlockChain.GetCRCommittee().IsElectedCRMemberByDID(crcProposal.CRCouncilMemberDID) {
		return errors.New("CR Council Member should be one elected CR members")
	}
	//verify 3 signature(owner signature , new secretary general, CRCouncilMember)

	//Check signature of owner
	signedBuf := new(bytes.Buffer)
	if err := checkProposalOwnerSign(crcProposal, signedBuf, PayloadVersion); err != nil {
		return errors.New("owner signature check failed")
	}
	// Check signature of SecretaryGeneral
	if err := checkSecretaryGeneralSign(crcProposal, signedBuf); err != nil {
		return errors.New("SecretaryGeneral signature check failed")
	}
	// Check signature of CR Council Member.
	crMember := t.parameters.BlockChain.GetCRCommittee().GetMember(crcProposal.CRCouncilMemberDID)
	if crMember == nil {
		return errors.New("CR Council Member should be one of the CR members")
	}
	if err := checkProposalCRCouncilMemberSign(crcProposal, crMember.Info.Code, signedBuf); err != nil {
		return errors.New("CR Council Member signature check failed")
	}
	return nil
}

func isPublicKeyDIDMatch(pubKey []byte, did *common.Uint168) bool {
	var code []byte
	var err error
	//get Code
	if code, err = getCode(pubKey); err != nil {
		return false
	}
	//get DID
	var didGenerated *common.Uint168
	if didGenerated, err = getDIDFromCode(code); err != nil {
		return false
	}
	if !did.IsEqual(*didGenerated) {
		return false
	}
	return true
}

func getCode(publicKey []byte) ([]byte, error) {
	if pk, err := crypto.DecodePoint(publicKey); err != nil {
		return nil, err
	} else {
		if redeemScript, err := contract.CreateStandardRedeemScript(pk); err != nil {
			return nil, err
		} else {
			return redeemScript, nil
		}
	}
}

func checkProposalOwnerSign(crcProposal *payload.CRCProposal, signedBuf *bytes.Buffer,
	PayloadVersion byte) error {
	//get ownerCode
	var code []byte
	var err error
	if code, err = getCode(crcProposal.OwnerPublicKey); err != nil {
		return err
	}
	// get verify data
	err = crcProposal.SerializeUnsigned(signedBuf, PayloadVersion)
	if err != nil {
		return err
	}
	//verify sign
	if err := blockchain.CheckCRTransactionSignature(crcProposal.Signature, code,
		signedBuf.Bytes()); err != nil {
		return errors.New("owner signature check failed")
	}
	return nil
}

func checkSecretaryGeneralSign(crcProposal *payload.CRCProposal, signedBuf *bytes.Buffer) error {
	var code []byte
	var err error
	if code, err = getCode(crcProposal.SecretaryGeneralPublicKey); err != nil {
		return err
	}
	if err = blockchain.CheckCRTransactionSignature(crcProposal.SecretaryGeneraSignature, code,
		signedBuf.Bytes()); err != nil {
		return errors.New("failed to check SecretaryGeneral signature")
	}
	return nil
}

func checkProposalCRCouncilMemberSign(crcProposal *payload.CRCProposal, code []byte,
	signedBuf *bytes.Buffer) error {

	// Check signature of CR Council Member.
	if err := common.WriteVarBytes(signedBuf, crcProposal.Signature); err != nil {
		return errors.New("failed to write proposal owner signature")
	}
	if err := common.WriteVarBytes(signedBuf, crcProposal.SecretaryGeneraSignature); err != nil {
		return errors.New("failed to write SecretaryGenera Signature")
	}
	if err := crcProposal.CRCouncilMemberDID.Serialize(signedBuf); err != nil {
		return errors.New("failed to write CR Council Member's DID")
	}
	if err := blockchain.CheckCRTransactionSignature(crcProposal.CRCouncilMemberSignature, code,
		signedBuf.Bytes()); err != nil {
		return errors.New("failed to check CR Council Member signature")
	}

	return nil
}

func (t *CRCProposalTransaction) checkReservedCustomID(params *TransactionParameters, proposal *payload.CRCProposal, PayloadVersion byte) error {

	if t.parameters.BlockChain.GetCRCommittee().GetProposalManager().ReservedCustomID {
		return errors.New("Already have one ReservedCustomID proposal")
	}
	_, err := crypto.DecodePoint(proposal.OwnerPublicKey)
	if err != nil {
		return errors.New("DecodePoint from OwnerPublicKey error")
	}

	if len(proposal.ReservedCustomIDList) == 0 {
		return errors.New("reserved custom id list is empty")
	}
	customIDMap := make(map[string]struct{})
	for _, v := range proposal.ReservedCustomIDList {
		if len(v) == 0 || len(v) > int(t.parameters.Config.MaxReservedCustomIDLength) {
			return errors.New("invalid reserved custom id length")
		}
		if _, ok := customIDMap[v]; ok {
			return errors.New("duplicated reserved custom ID")
		}
		if !common.IsLetterOrNumber(v) {
			return errors.New("invalid custom ID: only letter and number is allowed")
		}
		customIDMap[v] = struct{}{}
	}
	crMember := t.parameters.BlockChain.GetCRCommittee().GetMember(proposal.CRCouncilMemberDID)
	if crMember == nil {
		return errors.New("CR Council Member should be one of the CR members")
	}
	return t.checkOwnerAndCRCouncilMemberSign(proposal, crMember.Info.Code, PayloadVersion)
}

func (t *CRCProposalTransaction) checkReceivedCustomID(params *TransactionParameters, proposal *payload.CRCProposal, PayloadVersion byte) error {
	_, err := crypto.DecodePoint(proposal.OwnerPublicKey)
	if err != nil {
		return errors.New("DecodePoint from OwnerPublicKey error")
	}
	reservedCustomIDList := t.parameters.BlockChain.GetCRCommittee().GetReservedCustomIDLists()
	receivedCustomIDList := t.parameters.BlockChain.GetCRCommittee().GetReceivedCustomIDLists()
	pendingReceivedCustomIDMap := t.parameters.BlockChain.GetCRCommittee().GetPendingReceivedCustomIDMap()

	if len(proposal.ReceivedCustomIDList) == 0 {
		return errors.New("received custom id list is empty")
	}
	customIDMap := make(map[string]struct{})
	for _, v := range proposal.ReceivedCustomIDList {
		if len(v) == 0 || len(v) > int(t.parameters.Config.MaxReservedCustomIDLength) {
			return errors.New("invalid received custom id length")
		}
		if _, ok := customIDMap[v]; ok {
			return errors.New("duplicated received custom ID")
		}
		if _, ok := pendingReceivedCustomIDMap[v]; ok {
			return errors.New("received custom id is receiving")
		}
		if utils.StringExisted(receivedCustomIDList, v) {
			return errors.New("received custom id already received")
		}
		if !utils.StringExisted(reservedCustomIDList, v) {
			return errors.New("received custom id can not be found in reserved custom id list")
		}
		customIDMap[v] = struct{}{}
	}

	crMember := t.parameters.BlockChain.GetCRCommittee().GetMember(proposal.CRCouncilMemberDID)
	if crMember == nil {
		return errors.New("CR Council Member should be one of the CR members")
	}
	return t.checkOwnerAndCRCouncilMemberSign(proposal, crMember.Info.Code, PayloadVersion)
}

func (t *CRCProposalTransaction) checkChangeCustomIDFee(params *TransactionParameters, proposal *payload.CRCProposal, PayloadVersion byte) error {
	_, err := crypto.DecodePoint(proposal.OwnerPublicKey)
	if err != nil {
		return errors.New("DecodePoint from OwnerPublicKey error")
	}
	if proposal.RateOfCustomIDFee < 0 {
		return errors.New("invalid fee rate of custom ID")
	}
	if proposal.EIDEffectiveHeight <= 0 {
		return errors.New("invalid EID effective height")
	}
	crMember := t.parameters.BlockChain.GetCRCommittee().GetMember(proposal.CRCouncilMemberDID)
	if crMember == nil {
		return errors.New("CR Council Member should be one of the CR members")
	}
	return t.checkOwnerAndCRCouncilMemberSign(proposal, crMember.Info.Code, PayloadVersion)
}

func (t *CRCProposalTransaction) checkRegisterSideChainProposal(params *TransactionParameters, proposal *payload.CRCProposal, payloadVersion byte) error {
	_, err := crypto.DecodePoint(proposal.OwnerPublicKey)
	if err != nil {
		return errors.New("DecodePoint from OwnerPublicKey error")
	}

	if proposal.SideChainName == "" {
		return errors.New("SideChainName can not be empty")
	}

	for _, name := range t.parameters.BlockChain.GetCRCommittee().GetProposalManager().RegisteredSideChainNames {
		if name == proposal.SideChainName {
			return errors.New("SideChainName already registered")
		}
	}

	for _, mn := range t.parameters.BlockChain.GetCRCommittee().GetProposalManager().RegisteredMagicNumbers {
		if mn == proposal.MagicNumber {
			return errors.New("MagicNumber already registered")
		}
	}

	for _, gene := range t.parameters.BlockChain.GetCRCommittee().GetProposalManager().RegisteredGenesisHashes {
		if gene.IsEqual(proposal.GenesisHash) {
			return errors.New("Genesis Hash already registered")
		}
	}

	if proposal.ExchangeRate != common.Fixed64(1e8) {
		return errors.New("ExchangeRate should be 1.0")
	}

	if proposal.EffectiveHeight < t.parameters.BlockChain.GetHeight() {
		return errors.New("EffectiveHeight must be bigger than current height")
	}

	if proposal.GenesisHash == common.EmptyHash {
		return errors.New("GenesisHash can not be empty")
	}

	if len(proposal.Budgets) > 0 {
		return errors.New("RegisterSideChain cannot have budget")
	}
	emptyUint168 := common.Uint168{}
	if proposal.Recipient != emptyUint168 {
		return errors.New("RegisterSideChain recipient must be empty")
	}
	crMember := t.parameters.BlockChain.GetCRCommittee().GetMember(proposal.CRCouncilMemberDID)
	if crMember == nil {
		return errors.New("CR Council Member should be one of the CR members")
	}

	return t.checkOwnerAndCRCouncilMemberSign(proposal, crMember.Info.Code, payloadVersion)
}

func (t *CRCProposalTransaction) checkNormalOrELIPProposal(params *TransactionParameters,
	proposal *payload.CRCProposal, proposalsUsedAmount common.Fixed64, PayloadVersion byte) error {
	if proposal.ProposalType == payload.ELIP {
		if len(proposal.Budgets) != blockchain.ELIPBudgetsCount {
			return errors.New("ELIP needs to have and only have two budget")
		}
		for _, budget := range proposal.Budgets {
			if budget.Type == payload.NormalPayment {
				return errors.New("ELIP needs to have no normal payment")
			}
		}
	}
	// Check budgets of proposal
	if len(proposal.Budgets) < 1 {
		return errors.New("a proposal cannot be without a Budget")
	}
	budgets := make([]payload.Budget, len(proposal.Budgets))
	for i, budget := range proposal.Budgets {
		budgets[i] = budget
	}
	sort.Slice(budgets, func(i, j int) bool {
		return budgets[i].Stage < budgets[j].Stage
	})
	if budgets[0].Type == payload.Imprest && budgets[0].Stage != 0 {
		return errors.New("proposal imprest can only be in the first phase")
	}
	if budgets[0].Type != payload.Imprest && budgets[0].Stage != 1 {
		return errors.New("the first general type budget needs to start at the beginning")
	}
	if budgets[len(budgets)-1].Type != payload.FinalPayment {
		return errors.New("proposal final payment can only be in the last phase")
	}
	stage := budgets[0].Stage
	var amount common.Fixed64
	var imprestPaymentCount int
	var finalPaymentCount int
	for _, b := range budgets {
		switch b.Type {
		case payload.Imprest:
			imprestPaymentCount++
		case payload.NormalPayment:
		case payload.FinalPayment:
			finalPaymentCount++
		default:
			return errors.New("type of budget should be known")
		}
		if b.Stage != stage {
			return errors.New("the first phase starts incrementing")
		}
		if b.Amount < 0 {
			return errors.New("invalid amount")
		}
		stage++
		amount += b.Amount
	}
	if imprestPaymentCount > 1 {
		return errors.New("imprest payment count invalid")
	}
	if finalPaymentCount != 1 {
		return errors.New("final payment count invalid")
	}
	if amount > (t.parameters.BlockChain.GetCRCommittee().CRCCurrentStageAmount-
		t.parameters.BlockChain.GetCRCommittee().CommitteeUsedAmount)*blockchain.CRCProposalBudgetsPercentage/100 {
		return errors.New("budgets exceeds 10% of CRC committee balance")
	} else if amount > t.parameters.BlockChain.GetCRCommittee().CRCCurrentStageAmount-
		t.parameters.BlockChain.GetCRCommittee().CRCCommitteeUsedAmount-proposalsUsedAmount {
		return errors.New(fmt.Sprintf("budgets exceeds the balance of CRC"+
			" committee, proposal hash:%s, budgets:%s, need <= %s",
			common.ToReversedString(proposal.Hash(PayloadVersion)), amount, t.parameters.BlockChain.GetCRCommittee().CRCCurrentStageAmount-
				t.parameters.BlockChain.GetCRCommittee().CRCCommitteeUsedAmount-proposalsUsedAmount))
	} else if amount < 0 {
		return errors.New("budgets is invalid")
	}
	emptyUint168 := common.Uint168{}
	if proposal.Recipient == emptyUint168 {
		return errors.New("recipient is empty")
	}
	prefix := contract.GetPrefixType(proposal.Recipient)
	if prefix != contract.PrefixStandard && prefix != contract.PrefixMultiSig {
		return errors.New("invalid recipient prefix")
	}
	_, err := proposal.Recipient.ToAddress()
	if err != nil {
		return errors.New("invalid recipient")
	}
	crCouncilMember := t.parameters.BlockChain.GetCRCommittee().GetMember(proposal.CRCouncilMemberDID)
	return t.checkOwnerAndCRCouncilMemberSign(proposal, crCouncilMember.Info.Code, PayloadVersion)
}
