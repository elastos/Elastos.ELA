package wallet

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"

	"github.com/urfave/cli"
)

func outputPayloadAndDigest(data []byte) error {
	d := sha256.Sum256(data)

	digest, err := common.Uint256FromBytes(d[:])
	if err != nil {
		return errors.New("convert digest from bytes err: " + err.Error())
	}

	fmt.Println("payload: ", common.BytesToHexString(data))
	fmt.Println("digest: ", digest.ReversedString())
	return nil
}

func payloadProposalCRCouncilMemberUnsigned(c *cli.Context) error {
	// deserialize
	sPayload := c.String(cmdcom.TransactionPayloadFlag.Name)
	pBuf, err := common.HexStringToBytes(sPayload)
	if err != nil {
		return errors.New("payload hexstring to bytes error: " + err.Error())
	}

	p := &payload.CRCProposal{}
	r := bytes.NewBuffer(pBuf)
	err = common.ReadElement(r, &p.ProposalType)
	if err != nil {
		return errors.New("[CRCProposal], ProposalType deserialize failed")
	}

	err = p.DeserializeUnSigned(bytes.NewBuffer(pBuf), payload.CRCProposalVersion01)
	if err != nil {
		return errors.New("payload deserialize error: " + err.Error())
	}

	// process args
	sOwnerSign := c.String(cmdcom.TransactionOwnerSignatureFlag.Name)
	ownerSign, err := common.HexStringToBytes(sOwnerSign)
	if err != nil {
		return errors.New("invalid owner signature: " + err.Error())
	}
	p.Signature = ownerSign

	sCRCouncilMemberDID := c.String(cmdcom.TransactionCRCouncilMemberDIDFlag.Name)
	crcouncilMemberDID, err := common.Uint168FromAddress(sCRCouncilMemberDID)
	if err != nil {
		return errors.New("invalid crcouncil member did: " + err.Error())
	}
	p.CRCouncilMemberDID = *crcouncilMemberDID

	// serialize
	w := new(bytes.Buffer)
	if err := p.SerializeUnsigned(w, payload.CRCProposalVersion01); err != nil {
		return errors.New("serialize payload err: " + err.Error())
	}

	if err := common.WriteVarBytes(w, p.Signature); err != nil {
		return errors.New("serialize payload err: " + err.Error())
	}

	if err := p.CRCouncilMemberDID.Serialize(w); err != nil {
		return errors.New("serialize payload err: " + err.Error())
	}

	return outputPayloadAndDigest(w.Bytes())
}

func createProposalTransactionCommon(c *cli.Context) error {
	var name string

	name = cmdcom.TransactionFeeFlag.Name
	feeStr := c.String(name)
	if feeStr == "" {
		return errors.New(fmt.Sprintf("use --%s to specify transfer fee", name))
	}
	fee, err := common.StringToFixed64(feeStr)
	if err != nil {
		return errors.New("invalid transaction fee")
	}

	name = strings.Split(cmdcom.AccountWalletFlag.Name, ",")[0]
	walletPath := c.String(name)
	if walletPath == "" {
		return errors.New(fmt.Sprintf("use --%s to specify wallet path", name))
	}

	name = cmdcom.TransactionPayloadFlag.Name
	sPayload := c.String(name)
	pBuf, err := common.HexStringToBytes(sPayload)
	if err != nil {
		return errors.New("payload hexstring to bytes error: " + err.Error())
	}

	p := &payload.CRCProposal{}
	r := bytes.NewBuffer(pBuf)
	err = common.ReadElement(r, &p.ProposalType)
	if err != nil {
		return errors.New("[CRCProposal], ProposalType deserialize failed")
	}

	if err := p.DeserializeUnSigned(r, payload.CRCProposalVersion01); err != nil {
		return err
	}

	sign, err := common.ReadVarBytes(r, crypto.SignatureLength, "sign data")
	if err != nil {
		return err
	}
	p.Signature = sign

	if err := p.CRCouncilMemberDID.Deserialize(r); err != nil {
		return errors.New("failed to deserialize CRCouncilMemberDID")
	}

	sCRCouncilMemberSignature := c.String(cmdcom.TransactionCRCouncilMemberSignatureFlag.Name)
	crcouncilMemberSignature, err := common.HexStringToBytes(sCRCouncilMemberSignature)
	if err != nil {
		return errors.New("crc member sign hex string to bytes error :" + err.Error())
	}

	p.CRCouncilMemberSignature = crcouncilMemberSignature

	var txn interfaces.Transaction
	outputs := make([]*OutputInfo, 0)
	txn, err = createTransaction(walletPath, "", *fee, 0, 0, common2.CRCProposal,
		payload.CRCProposalVersion01, p, outputs...)
	if err != nil {
		return errors.New("create transaction failed: " + err.Error())
	}

	OutputTx(0, 1, txn)

	return nil
}

func payloadProposalNormalOwnerUnsigned(c *cli.Context) error {
	categoryData := c.String(cmdcom.TransactionCategoryDataFlag.Name)

	sOwnerPubKey := c.String(cmdcom.TransactionOwnerPublicKeyFlag.Name)
	ownerPublicKey, err := common.HexStringToBytes(sOwnerPubKey)
	if err != nil {
		return errors.New("invalid owner pub key: " + err.Error())
	}

	sDraftHash := c.String(cmdcom.TransactionDraftHashFlag.Name)
	draftHash, err := common.Uint256FromReversedHexString(sDraftHash)
	if err != nil {
		return errors.New("draft hash from hex string err: " + err.Error())
	}

	sDraftData := c.String(cmdcom.TransactionDraftDataFlag.Name)
	draftData, err := common.HexStringToBytes(sDraftData)
	if err != nil {
		return errors.New("draft data from hex string err: " + err.Error())
	}

	// should be like this: type1,stage1,amount1|type2,stage2,amount2
	sBudgets := c.String(cmdcom.TransactionBudgetsFlag.Name)

	var budgets []payload.Budget
	for _, s := range strings.Split(sBudgets, "|") {
		sBudget := strings.Split(s, ",")
		if len(sBudget) != 3 {
			return errors.New("invalid budget format")
		}

		budgetType, err := strconv.ParseUint(sBudget[0], 0, 0)
		if err != nil {
			return errors.New("parse budget type error: " + err.Error())
		}
		budgetStage, err := strconv.ParseUint(sBudget[1], 0, 0)
		if err != nil {
			return errors.New("parse budget stage error: " + err.Error())
		}
		budgetAmount, err := common.StringToFixed64(sBudget[2])
		if err != nil {
			return errors.New("parse budget amount error: " + err.Error())
		}

		budget := payload.Budget{
			Type:   payload.InstallmentType(budgetType),
			Stage:  byte(budgetStage),
			Amount: *budgetAmount,
		}

		budgets = append(budgets, budget)
	}

	sRecipient := c.String(cmdcom.TransactionRecipientFlag.Name)
	recipient, err := common.Uint168FromAddress(sRecipient)
	if err != nil {
		return errors.New("invalid recipient address: " + err.Error())
	}

	p := &payload.CRCProposal{
		ProposalType:   payload.Normal,
		CategoryData:   categoryData,
		OwnerPublicKey: ownerPublicKey,
		DraftHash:      *draftHash,
		DraftData:      draftData,
		Budgets:        budgets,
		Recipient:      *recipient,
	}

	w := new(bytes.Buffer)
	if err := p.SerializeUnsigned(w, payload.CRCProposalVersion01); err != nil {
		return errors.New("serialize payload err: " + err.Error())
	}

	return outputPayloadAndDigest(w.Bytes())
}

func payloadProposalNormalCRCouncilMemberUnsigned(c *cli.Context) error {
	return payloadProposalCRCouncilMemberUnsigned(c)
}

func createNormalProposalTransaction(c *cli.Context) error {
	return createProposalTransactionCommon(c)
}

func payloadProposalReviewOwnerUnsigned(c *cli.Context) error {
	sProposalHash := c.String(cmdcom.TransactionProposalHashFlag.Name)
	proposalHash, err := common.Uint256FromReversedHexString(sProposalHash)
	if err != nil {
		return errors.New("invalid proposal hash")
	}

	voteResult := c.Uint(cmdcom.TransactionVoteResultFlag.Name)

	sOpinionHash := c.String(cmdcom.TransactionOpinionHashFlag.Name)
	opinionHash, err := common.Uint256FromReversedHexString(sOpinionHash)
	if err != nil {
		return errors.New("invalid opinion hash")
	}

	sOpinionData := c.String(cmdcom.TransactionOpinionDataFlag.Name)
	opinionData, err := common.HexStringToBytes(sOpinionData)
	if err != nil {
		return errors.New("invalid opinion data")
	}

	sDID := c.String(cmdcom.TransactionDIDFlag.Name)
	did, err := common.Uint168FromAddress(sDID)
	if err != nil {
		return errors.New("did from address err: " + err.Error())
	}

	p := &payload.CRCProposalReview{
		ProposalHash: *proposalHash,
		VoteResult:   payload.VoteResult(voteResult),
		OpinionHash:  *opinionHash,
		OpinionData:  opinionData,
		DID:          *did,
		Signature:    nil,
	}

	w := new(bytes.Buffer)
	if err := p.SerializeUnsigned(w, payload.CRCProposalReviewVersion01); err != nil {
		return errors.New("serialize payload err: " + err.Error())
	}

	return outputPayloadAndDigest(w.Bytes())
}

func createProposalReviewTransaction(c *cli.Context) error {
	var name string

	name = cmdcom.TransactionFeeFlag.Name
	feeStr := c.String(name)
	if feeStr == "" {
		return errors.New(fmt.Sprintf("use --%s to specify transfer fee", name))
	}
	fee, err := common.StringToFixed64(feeStr)
	if err != nil {
		return errors.New("invalid transaction fee")
	}

	name = strings.Split(cmdcom.AccountWalletFlag.Name, ",")[0]
	walletPath := c.String(name)
	if walletPath == "" {
		return errors.New(fmt.Sprintf("use --%s to specify wallet path", name))
	}

	p := &payload.CRCProposalReview{}
	name = cmdcom.TransactionPayloadFlag.Name
	sPayload := c.String(name)
	pBuf, err := common.HexStringToBytes(sPayload)
	if err != nil {
		return errors.New("payload hexstring to bytes error: " + err.Error())
	}

	r := bytes.NewBuffer(pBuf)
	err = p.DeserializeUnsigned(r, payload.CRCProposalReviewVersion01)
	if err != nil {
		return errors.New("deserialize payload: " + err.Error())
	}

	sSign := c.String(cmdcom.TransactionOwnerSignatureFlag.Name)
	sign, err := common.HexStringToBytes(sSign)
	if err != nil {
		return errors.New("invalid owner signature: " + err.Error())
	}
	p.Signature = sign

	var txn interfaces.Transaction
	outputs := make([]*OutputInfo, 0)
	txn, err = createTransaction(walletPath, "", *fee, 0, 0, common2.CRCProposalReview,
		payload.CRCProposalReviewVersion01, p, outputs...)
	if err != nil {
		return errors.New("create transaction failed: " + err.Error())
	}

	OutputTx(0, 1, txn)

	return nil
}

func payloadProposalTrackingOwnerUnsigned(c *cli.Context) error {
	sProposalHash := c.String(cmdcom.TransactionProposalHashFlag.Name)
	proposalHash, err := common.Uint256FromReversedHexString(sProposalHash)
	if err != nil {
		return errors.New("invalid proposal hash: " + err.Error())
	}

	sMessageHash := c.String(cmdcom.TransactionMessageHashFlag.Name)
	messageHash, err := common.Uint256FromReversedHexString(sMessageHash)
	if err != nil {
		return errors.New("invalid message hash: " + err.Error())
	}

	sMessageData := c.String(cmdcom.TransactionMessageDataFlag.Name)
	messageData, err := common.HexStringToBytes(sMessageData)
	if err != nil {
		return errors.New("invalid message data: " + err.Error())
	}

	stage := c.Uint(cmdcom.CRCProposalStageFlag.Name)

	sOwnerPublicKey := c.String(cmdcom.TransactionOwnerPublicKeyFlag.Name)
	ownerPubKey, err := common.HexStringToBytes(sOwnerPublicKey)
	if err != nil {
		return errors.New("invalid owner pubkey: " + err.Error())
	}

	sNewOwnerPublicKey := c.String(cmdcom.TransactionNewOwnerPublicKeyFlag.Name)
	newOwnerPublicKey, err := common.HexStringToBytes(sNewOwnerPublicKey)
	if err != nil {
		return errors.New("invalid new owner pubkey: " + err.Error())
	}

	p := &payload.CRCProposalTracking{
		ProposalHash:      *proposalHash,
		MessageHash:       *messageHash,
		MessageData:       messageData,
		Stage:             uint8(stage),
		OwnerPublicKey:    ownerPubKey,
		NewOwnerPublicKey: newOwnerPublicKey,
	}

	pSignBuf := new(bytes.Buffer)
	if err := p.SerializeUnsigned(pSignBuf, payload.CRCProposalTrackingVersion01); err != nil {
		return errors.New("serialize payload err: " + err.Error())
	}

	d := sha256.Sum256(pSignBuf.Bytes())

	digest, err := common.Uint256FromBytes(d[:])
	if err != nil {
		return errors.New("convert digest from bytes err: " + err.Error())
	}

	fmt.Println("payload: ", common.BytesToHexString(pSignBuf.Bytes()))
	fmt.Println("digest: ", digest.ReversedString())
	return nil
}

func payloadProposalTrackingNewOwnerUnsigned(c *cli.Context) error {
	sPayload := c.String(cmdcom.TransactionPayloadFlag.Name)
	pBuf, err := common.HexStringToBytes(sPayload)
	if err != nil {
		return errors.New("payload hexstring to bytes error: " + err.Error())
	}

	p := &payload.CRCProposalTracking{}
	r := bytes.NewBuffer(pBuf)
	err = p.DeserializeUnSigned(r, payload.CRCProposalTrackingVersion01)
	if err != nil {
		return errors.New("payload deserialize error: " + err.Error())
	}

	sOwnerSign := c.String(cmdcom.TransactionOwnerSignatureFlag.Name)
	ownerSign, err := common.HexStringToBytes(sOwnerSign)
	if err != nil {
		return errors.New("invalid owner signature: " + err.Error())
	}
	p.OwnerSignature = ownerSign

	w := new(bytes.Buffer)
	if err := p.SerializeUnsigned(w, payload.CRCProposalTrackingVersion01); err != nil {
		return errors.New("serialize payload err: " + err.Error())
	}
	if err := common.WriteVarBytes(w, p.OwnerSignature); err != nil {
		return errors.New("failed to serialize OwnerSignature")
	}

	d := sha256.Sum256(w.Bytes())

	digest, err := common.Uint256FromBytes(d[:])
	if err != nil {
		return errors.New("convert digest from bytes err: " + err.Error())
	}

	fmt.Println("payload: ", common.BytesToHexString(w.Bytes()))
	fmt.Println("digest: ", digest.ReversedString())
	return nil
}

func payloadProposalTrackingSecretaryGeneralUnsigned(c *cli.Context) error {
	sPayload := c.String(cmdcom.TransactionPayloadFlag.Name)
	pBuf, err := common.HexStringToBytes(sPayload)
	if err != nil {
		return errors.New("payload hexstring to bytes error: " + err.Error())
	}

	// deserialize from payload
	p := &payload.CRCProposalTracking{}
	r := bytes.NewBuffer(pBuf)
	err = p.DeserializeUnSigned(r, payload.CRCProposalTrackingVersion01)
	if err != nil {
		return errors.New("payload deserialize error: " + err.Error())
	}

	ownerSign, err := common.ReadVarBytes(r, crypto.SignatureLength,
		"owner signature")
	if err != nil {
		return errors.New("failed to deserialize ownerSign")
	}
	p.OwnerSignature = ownerSign

	// input from command line
	sNewOwnerSign := c.String(cmdcom.TransactionNewOwnerSignatureFlag.Name)
	newOwnerSign, err := common.HexStringToBytes(sNewOwnerSign)
	if err != nil {
		return errors.New("invalid new owner sign: " + err.Error())
	}
	p.NewOwnerSignature = newOwnerSign

	t := c.Uint(cmdcom.TransactionProposalTrackingTypeFlag.Name)
	p.ProposalTrackingType = payload.CRCProposalTrackingType(t)

	sSecretaryGeneralOpinionHash := c.String(cmdcom.TransactionSecretaryGeneralOpinionHashFlag.Name)
	secretaryGeneralOpinionHash, err := common.Uint256FromReversedHexString(sSecretaryGeneralOpinionHash)
	if err != nil {
		return errors.New("invalid secretary general opinion hash: " + err.Error())
	}
	p.SecretaryGeneralOpinionHash = *secretaryGeneralOpinionHash

	sSecretaryGeneralOpinionData := c.String(cmdcom.TransactionSecretaryGeneralOpinionDataFlag.Name)
	secretaryGeneralOpinionData, err := common.HexStringToBytes(sSecretaryGeneralOpinionData)
	if err != nil {
		return errors.New("invalid secretary general opinion data: " + err.Error())
	}
	p.SecretaryGeneralOpinionData = secretaryGeneralOpinionData

	// serialize payload
	w := new(bytes.Buffer)
	if err := p.SerializeUnsigned(w, payload.CRCProposalTrackingVersion01); err != nil {
		return errors.New("serialize payload err: " + err.Error())
	}
	if err := common.WriteVarBytes(w, p.OwnerSignature); err != nil {
		return errors.New("failed to serialize OwnerSignature")
	}
	if err := common.WriteVarBytes(w, p.NewOwnerSignature); err != nil {
		return errors.New("failed to serialize NewOwnerSignature")
	}

	if _, err := w.Write([]byte{byte(p.ProposalTrackingType)}); err != nil {
		return errors.New("failed to serialize ProposalTrackingType")
	}

	if err := p.SecretaryGeneralOpinionHash.Serialize(w); err != nil {
		return errors.New("failed to serialize SecretaryGeneralOpinionHash")
	}

	if err := common.WriteVarBytes(w, p.SecretaryGeneralOpinionData); err != nil {
		return errors.New("failed to serialize SecretaryGeneralOpinionData")
	}

	d := sha256.Sum256(w.Bytes())

	digest, err := common.Uint256FromBytes(d[:])
	if err != nil {
		return errors.New("convert digest from bytes err: " + err.Error())
	}

	fmt.Println("payload: ", common.BytesToHexString(w.Bytes()))
	fmt.Println("digest: ", digest.ReversedString())
	return nil
}

func createProposalTrackingTransaction(c *cli.Context) error {
	var name string

	sPayload := c.String(cmdcom.TransactionPayloadFlag.Name)
	pBuf, err := common.HexStringToBytes(sPayload)
	if err != nil {
		return errors.New("payload hexstring to bytes error: " + err.Error())
	}

	name = cmdcom.TransactionFeeFlag.Name
	feeStr := c.String(name)
	if feeStr == "" {
		return errors.New(fmt.Sprintf("use --%s to specify transfer fee", name))
	}
	fee, err := common.StringToFixed64(feeStr)
	if err != nil {
		return errors.New("invalid transaction fee")
	}

	name = strings.Split(cmdcom.AccountWalletFlag.Name, ",")[0]
	walletPath := c.String(name)
	if walletPath == "" {
		return errors.New(fmt.Sprintf("use --%s to specify wallet path", name))
	}

	p := &payload.CRCProposalTracking{}
	r := bytes.NewBuffer(pBuf)
	err = p.DeserializeUnSigned(r, payload.CRCProposalTrackingVersion01)
	if err != nil {
		return errors.New("payload deserialize error: " + err.Error())
	}

	ownerSign, err := common.ReadVarBytes(r, crypto.SignatureLength,
		"owner signature")
	if err != nil {
		return errors.New("failed to deserialize ownerSign")
	}
	p.OwnerSignature = ownerSign

	NewOwnerSignature, err := common.ReadVarBytes(r, crypto.SignatureLength,
		"new owner signature")
	if err != nil {
		return errors.New("failed to deserialize NewOwnerSignature")
	}
	p.NewOwnerSignature = NewOwnerSignature

	pType, err := common.ReadBytes(r, 1)
	if err != nil {
		return errors.New("failed to deserialize ProposalTrackingType")
	}
	p.ProposalTrackingType = payload.CRCProposalTrackingType(pType[0])

	if err = p.SecretaryGeneralOpinionHash.Deserialize(r); err != nil {
		return errors.New("failed to deserialize SecretaryGeneralOpinionHash")
	}

	if p.SecretaryGeneralOpinionData, err = common.ReadVarBytes(r,
		payload.MaxSecretaryGeneralOpinionDataSize, "opinion data"); err != nil {
		return errors.New("failed to deserialize SecretaryGeneralOpinionData")
	}

	sSecretaryGeneralSign := c.String(cmdcom.TransactionSecretaryGeneralSignatureFlag.Name)
	sgSign, err := common.HexStringToBytes(sSecretaryGeneralSign)
	if err != nil {
		return errors.New("invalid secretary general sign err: " + err.Error())
	}

	p.SecretaryGeneralSignature = sgSign

	var txn interfaces.Transaction
	outputs := make([]*OutputInfo, 0)
	txn, err = createTransaction(walletPath, "", *fee, 0, 0, common2.CRCProposalTracking,
		payload.CRCProposalTrackingVersion01, p, outputs...)
	if err != nil {
		return errors.New("create transaction failed: " + err.Error())
	}

	OutputTx(0, 1, txn)

	return nil
}

func payloadProposalSecretaryGeneralElectionUnsigned(c *cli.Context) error {
	categoryData := c.String(cmdcom.TransactionCategoryDataFlag.Name)

	sPubKey := c.String(cmdcom.TransactionOwnerPublicKeyFlag.Name)
	ownerPublicKey, err := common.HexStringToBytes(sPubKey)
	if err != nil {
		return errors.New("invalid owner pub key: " + err.Error())
	}

	sDraftHash := c.String(cmdcom.TransactionDraftHashFlag.Name)
	draftHash, err := common.Uint256FromReversedHexString(sDraftHash)
	if err != nil {
		return errors.New("draft hash from hex string err: " + err.Error())
	}

	sDraftData := c.String(cmdcom.TransactionDraftDataFlag.Name)
	draftData, err := common.HexStringToBytes(sDraftData)
	if err != nil {
		return errors.New("draft data from hex string err: " + err.Error())
	}

	sPubKey = c.String(cmdcom.TransactionSecretaryPublicKeyFlag.Name)
	secretaryPubKey, err := common.HexStringToBytes(sPubKey)
	if err != nil {
		return errors.New("invalid secretary general pubkey: " + err.Error())
	}

	sDID := c.String(cmdcom.TransactionSecretaryDIDFlag.Name)
	did, err := common.Uint168FromAddress(sDID)
	if err != nil {
		return errors.New("invalid secretary did: " + err.Error())
	}

	p := &payload.CRCProposal{
		ProposalType:              payload.SecretaryGeneral,
		CategoryData:              categoryData,
		OwnerPublicKey:            ownerPublicKey,
		DraftHash:                 *draftHash,
		DraftData:                 draftData,
		SecretaryGeneralPublicKey: secretaryPubKey,
		SecretaryGeneralDID:       *did,
	}

	w := new(bytes.Buffer)
	if err := p.SerializeUnsigned(w, payload.CRCProposalVersion01); err != nil {
		return errors.New("serialize payload err: " + err.Error())
	}

	return outputPayloadAndDigest(w.Bytes())
}

func payloadProposalSecretaryGeneralElectionCRCouncilMemberUnsigned(c *cli.Context) error {
	sPayload := c.String(cmdcom.TransactionPayloadFlag.Name)
	pBuf, err := common.HexStringToBytes(sPayload)
	if err != nil {
		return errors.New("payload hexstring to bytes error: " + err.Error())
	}
	p := &payload.CRCProposal{}
	r := bytes.NewBuffer(pBuf)
	err = common.ReadElement(r, &p.ProposalType)
	if err != nil {
		return errors.New("[CRCProposal], ProposalType deserialize failed")
	}
	err = p.DeserializeUnSigned(r, payload.CRCProposalVersion01)
	if err != nil {
		return errors.New("payload deserialize error: " + err.Error())
	}

	// process args
	sSign := c.String(cmdcom.TransactionOwnerSignatureFlag.Name)
	ownerSign, err := common.HexStringToBytes(sSign)
	if err != nil {
		return errors.New("invalid crcouncil member signature: " + err.Error())
	}
	p.Signature = ownerSign

	sSign = c.String(cmdcom.TransactionSecretarySignatureFlag.Name)
	secretarySign, err := common.HexStringToBytes(sSign)
	if err != nil {
		return errors.New("invalid crcouncil member signature: " + err.Error())
	}
	p.SecretaryGeneraSignature = secretarySign

	sDID := c.String(cmdcom.TransactionCRCouncilMemberDIDFlag.Name)
	did, err := common.Uint168FromAddress(sDID)
	if err != nil {
		return errors.New("invalid did: " + err.Error())
	}
	p.CRCouncilMemberDID = *did

	// serialize payload
	w := new(bytes.Buffer)
	if err := p.SerializeUnsigned(w, payload.CRCProposalVersion01); err != nil {
		return errors.New("serialize payload err: " + err.Error())
	}
	if err := common.WriteVarBytes(w, p.Signature); err != nil {
		return err
	}
	if err := common.WriteVarBytes(w, p.SecretaryGeneraSignature); err != nil {
		return err
	}

	if err := p.CRCouncilMemberDID.Serialize(w); err != nil {
		return errors.New("failed to serialize CRCouncilMemberDID")
	}

	return outputPayloadAndDigest(w.Bytes())
}

func createProposalSecretaryGeneralElectionTransaction(c *cli.Context) error {
	var name string

	name = cmdcom.TransactionFeeFlag.Name
	feeStr := c.String(name)
	if feeStr == "" {
		return errors.New(fmt.Sprintf("use --%s to specify transfer fee", name))
	}
	fee, err := common.StringToFixed64(feeStr)
	if err != nil {
		return errors.New("invalid transaction fee")
	}

	name = strings.Split(cmdcom.AccountWalletFlag.Name, ",")[0]
	walletPath := c.String(name)
	if walletPath == "" {
		return errors.New(fmt.Sprintf("use --%s to specify wallet path", name))
	}

	sPayload := c.String(cmdcom.TransactionPayloadFlag.Name)
	pBuf, err := common.HexStringToBytes(sPayload)
	if err != nil {
		return errors.New("payload hexstring to bytes error: " + err.Error())
	}

	// deserialize payload
	p := &payload.CRCProposal{}
	r := bytes.NewBuffer(pBuf)
	err = common.ReadElement(r, &p.ProposalType)
	if err != nil {
		return errors.New("[CRCProposal], ProposalType deserialize failed")
	}
	err = p.DeserializeUnSigned(r, payload.CRCProposalVersion01)
	if err != nil {
		return errors.New("payload deserialize error: " + err.Error())
	}

	sign, err := common.ReadVarBytes(r, crypto.SignatureLength, "sign data")
	if err != nil {
		return err
	}
	p.Signature = sign

	SecretaryGeneraSignature, err := common.ReadVarBytes(r, crypto.SignatureLength, "secretary general sign data")
	if err != nil {
		return err
	}
	p.SecretaryGeneraSignature = SecretaryGeneraSignature

	if err := p.CRCouncilMemberDID.Deserialize(r); err != nil {
		return errors.New("failed to deserialize CRCouncilMemberDID")
	}

	// process args
	sCRCouncilMemberSign := c.String(cmdcom.TransactionCRCouncilMemberSignatureFlag.Name)
	crcouncilMemberSign, err := common.HexStringToBytes(sCRCouncilMemberSign)
	if err != nil {
		return errors.New("invalid crcouncil member signature: " + err.Error())
	}
	p.CRCouncilMemberSignature = crcouncilMemberSign

	var txn interfaces.Transaction
	outputs := make([]*OutputInfo, 0)
	txn, err = createTransaction(walletPath, "", *fee, 0, 0, common2.CRCProposal,
		payload.CRCProposalVersion01, p, outputs...)
	if err != nil {
		return errors.New("create transaction failed: " + err.Error())
	}

	OutputTx(0, 1, txn)

	return nil
}

func payloadProposalChangeOwnerUnsigned(c *cli.Context) error {
	categoryData := c.String(cmdcom.TransactionCategoryDataFlag.Name)

	sPubKey := c.String(cmdcom.TransactionOwnerPublicKeyFlag.Name)
	ownerPublicKey, err := common.HexStringToBytes(sPubKey)
	if err != nil {
		return errors.New("invalid owner pub key: " + err.Error())
	}

	sDraftHash := c.String(cmdcom.TransactionDraftHashFlag.Name)
	draftHash, err := common.Uint256FromReversedHexString(sDraftHash)
	if err != nil {
		return errors.New("draft hash from hex string err: " + err.Error())
	}

	sDraftData := c.String(cmdcom.TransactionDraftDataFlag.Name)
	draftData, err := common.HexStringToBytes(sDraftData)
	if err != nil {
		return errors.New("draft data from hex string err: " + err.Error())
	}

	sTargetProposalHash := c.String(cmdcom.TransactionTargetProposalHashFlag.Name)
	targetProposalHash, err := common.Uint256FromReversedHexString(sTargetProposalHash)
	if err != nil {
		return errors.New("invalid target proposal hash: " + err.Error())
	}

	sRecipient := c.String(cmdcom.TransactionNewRecipientFlag.Name)
	recipient, err := common.Uint168FromAddress(sRecipient)
	if err != nil {
		return errors.New("invalid secretary did: " + err.Error())
	}

	sPubKey = c.String(cmdcom.TransactionNewOwnerPublicKeyFlag.Name)
	newOwnerPublicKey, err := common.HexStringToBytes(sPubKey)
	if err != nil {
		return errors.New("invalid owner pub key: " + err.Error())
	}

	p := &payload.CRCProposal{
		ProposalType:       payload.ChangeProposalOwner,
		CategoryData:       categoryData,
		OwnerPublicKey:     ownerPublicKey,
		DraftHash:          *draftHash,
		DraftData:          draftData,
		TargetProposalHash: *targetProposalHash,
		NewRecipient:       *recipient,
		NewOwnerPublicKey:  newOwnerPublicKey,
	}

	w := new(bytes.Buffer)
	if err := p.SerializeUnsigned(w, payload.CRCProposalVersion01); err != nil {
		return errors.New("serialize payload err: " + err.Error())
	}

	return outputPayloadAndDigest(w.Bytes())
}

func payloadProposalChangeOwnerCRCouncilMemberUnsigned(c *cli.Context) error {
	// deserialize
	sPayload := c.String(cmdcom.TransactionPayloadFlag.Name)
	pBuf, err := common.HexStringToBytes(sPayload)
	if err != nil {
		return errors.New("payload hexstring to bytes error: " + err.Error())
	}
	p := &payload.CRCProposal{}
	r := bytes.NewBuffer(pBuf)
	err = common.ReadElement(r, &p.ProposalType)
	if err != nil {
		return errors.New("[CRCProposal], ProposalType deserialize failed")
	}
	err = p.DeserializeUnSigned(r, payload.CRCProposalVersion01)
	if err != nil {
		return errors.New("payload deserialize error: " + err.Error())
	}

	// process args
	sOwnerSign := c.String(cmdcom.TransactionOwnerSignatureFlag.Name)
	ownerSign, err := common.HexStringToBytes(sOwnerSign)
	if err != nil {
		return err
	}
	p.Signature = ownerSign

	sNewOwnerSign := c.String(cmdcom.TransactionNewOwnerSignatureFlag.Name)
	newOwnerSign, err := common.HexStringToBytes(sNewOwnerSign)
	if err != nil {
		return err
	}
	p.NewOwnerSignature = newOwnerSign

	sDID := c.String(cmdcom.TransactionCRCouncilMemberDIDFlag.Name)
	did, err := common.Uint168FromAddress(sDID)
	if err != nil {
		return errors.New("failed to get CRCouncilMemberDID")
	}
	p.CRCouncilMemberDID = *did

	w := new(bytes.Buffer)
	if err := p.SerializeUnsigned(w, payload.CRCProposalVersion01); err != nil {
		return errors.New("serialize payload err: " + err.Error())
	}

	if err := common.WriteVarBytes(w, p.Signature); err != nil {
		return err
	}

	if err := common.WriteVarBytes(w, p.NewOwnerSignature); err != nil {
		return err
	}

	if err := p.CRCouncilMemberDID.Serialize(w); err != nil {
		return errors.New("failed to serialize CRCouncilMemberDID")
	}

	return outputPayloadAndDigest(w.Bytes())
}

func createProposalChangeOwnerTransaction(c *cli.Context) error {
	var name string

	name = cmdcom.TransactionFeeFlag.Name
	feeStr := c.String(name)
	if feeStr == "" {
		return errors.New(fmt.Sprintf("use --%s to specify transfer fee", name))
	}
	fee, err := common.StringToFixed64(feeStr)
	if err != nil {
		return errors.New("invalid transaction fee")
	}

	name = strings.Split(cmdcom.AccountWalletFlag.Name, ",")[0]
	walletPath := c.String(name)
	if walletPath == "" {
		return errors.New(fmt.Sprintf("use --%s to specify wallet path", name))
	}

	sPayload := c.String(cmdcom.TransactionPayloadFlag.Name)
	pBuf, err := common.HexStringToBytes(sPayload)
	if err != nil {
		return errors.New("payload hexstring to bytes error: " + err.Error())
	}

	// deserialize payload
	p := &payload.CRCProposal{}
	r := bytes.NewBuffer(pBuf)
	err = common.ReadElement(r, &p.ProposalType)
	if err != nil {
		return errors.New("[CRCProposal], ProposalType deserialize failed")
	}
	err = p.DeserializeUnSigned(r, payload.CRCProposalVersion01)
	if err != nil {
		return errors.New("payload deserialize error: " + err.Error())
	}
	sign, err := common.ReadVarBytes(r, crypto.SignatureLength, "sign data")
	if err != nil {
		return err
	}
	p.Signature = sign

	newOwnerSign, err := common.ReadVarBytes(r, crypto.SignatureLength, "sign data")
	if err != nil {
		return err
	}
	p.NewOwnerSignature = newOwnerSign

	if err := p.CRCouncilMemberDID.Deserialize(r); err != nil {
		return errors.New("failed to deserialize CRCouncilMemberDID")
	}

	// process args
	sCRCouncilMemberSign := c.String(cmdcom.TransactionCRCouncilMemberSignatureFlag.Name)
	crcouncilMemberSign, err := common.HexStringToBytes(sCRCouncilMemberSign)
	if err != nil {
		return errors.New("invalid crcouncil member signature: " + err.Error())
	}
	p.CRCouncilMemberSignature = crcouncilMemberSign

	var txn interfaces.Transaction
	outputs := make([]*OutputInfo, 0)
	txn, err = createTransaction(walletPath, "", *fee, 0, 0, common2.CRCProposal,
		payload.CRCProposalVersion01, p, outputs...)
	if err != nil {
		return errors.New("create transaction failed: " + err.Error())
	}

	OutputTx(0, 1, txn)
	return nil
}

func payloadProposalTerminateOwnerUnsigned(c *cli.Context) error {
	categoryData := c.String(cmdcom.TransactionCategoryDataFlag.Name)

	sPubKey := c.String(cmdcom.TransactionOwnerPublicKeyFlag.Name)
	ownerPublicKey, err := common.HexStringToBytes(sPubKey)
	if err != nil {
		return errors.New("invalid owner pub key: " + err.Error())
	}

	sDraftHash := c.String(cmdcom.TransactionDraftHashFlag.Name)
	draftHash, err := common.Uint256FromReversedHexString(sDraftHash)
	if err != nil {
		return errors.New("draft hash from hex string err: " + err.Error())
	}

	sDraftData := c.String(cmdcom.TransactionDraftDataFlag.Name)
	draftData, err := common.HexStringToBytes(sDraftData)
	if err != nil {
		return errors.New("draft data from hex string err: " + err.Error())
	}

	sTargetProposalHash := c.String(cmdcom.TransactionTargetProposalHashFlag.Name)
	targetProposalHash, err := common.Uint256FromReversedHexString(sTargetProposalHash)
	if err != nil {
		return errors.New("invalid target proposal hash: " + err.Error())
	}

	p := &payload.CRCProposal{
		ProposalType:       payload.CloseProposal,
		CategoryData:       categoryData,
		OwnerPublicKey:     ownerPublicKey,
		DraftHash:          *draftHash,
		DraftData:          draftData,
		TargetProposalHash: *targetProposalHash,
	}

	w := new(bytes.Buffer)
	if err := p.SerializeUnsigned(w, payload.CRCProposalVersion01); err != nil {
		return errors.New("serialize payload err: " + err.Error())
	}

	return outputPayloadAndDigest(w.Bytes())
}

func payloadProposalTerminateCRCouncilMemberUnsigned(c *cli.Context) error {
	return payloadProposalCRCouncilMemberUnsigned(c)
}

func createProposalTerminateTransaction(c *cli.Context) error {
	return createProposalTransactionCommon(c)
}

func payloadProposalReserverCustomIDOwnerUnsigned(c *cli.Context) error {
	categoryData := c.String(cmdcom.TransactionCategoryDataFlag.Name)

	sPubKey := c.String(cmdcom.TransactionOwnerPublicKeyFlag.Name)
	ownerPublicKey, err := common.HexStringToBytes(sPubKey)
	if err != nil {
		return errors.New("invalid owner pub key: " + err.Error())
	}

	sDraftHash := c.String(cmdcom.TransactionDraftHashFlag.Name)
	draftHash, err := common.Uint256FromReversedHexString(sDraftHash)
	if err != nil {
		return errors.New("draft hash from hex string err: " + err.Error())
	}

	sDraftData := c.String(cmdcom.TransactionDraftDataFlag.Name)
	draftData, err := common.HexStringToBytes(sDraftData)
	if err != nil {
		return errors.New("draft data from hex string err: " + err.Error())
	}

	sReservedCustomIDList := c.String(cmdcom.TransactionReservedCustomIDListFlag.Name)
	var reservedCustomIDList []string
	for _, s := range strings.Split(sReservedCustomIDList, "|") {
		reservedCustomIDList = append(reservedCustomIDList, s)
	}

	p := &payload.CRCProposal{
		ProposalType:         payload.ReserveCustomID,
		CategoryData:         categoryData,
		OwnerPublicKey:       ownerPublicKey,
		DraftHash:            *draftHash,
		DraftData:            draftData,
		ReservedCustomIDList: reservedCustomIDList,
	}

	w := new(bytes.Buffer)
	if err := p.SerializeUnsigned(w, payload.CRCProposalVersion01); err != nil {
		return errors.New("serialize payload err: " + err.Error())
	}

	return outputPayloadAndDigest(w.Bytes())
}

func payloadProposalReserverCustomIDCRCouncilMemberUnsigned(c *cli.Context) error {
	return payloadProposalCRCouncilMemberUnsigned(c)
}

func createProposalReserveCustomIDTransaction(c *cli.Context) error {
	return createProposalTransactionCommon(c)
}

func payloadProposalReceiveCustomIDOwnerUnsigned(c *cli.Context) error {
	categoryData := c.String(cmdcom.TransactionCategoryDataFlag.Name)

	sPubKey := c.String(cmdcom.TransactionOwnerPublicKeyFlag.Name)
	ownerPublicKey, err := common.HexStringToBytes(sPubKey)
	if err != nil {
		return errors.New("invalid owner pub key: " + err.Error())
	}

	sDraftHash := c.String(cmdcom.TransactionDraftHashFlag.Name)
	draftHash, err := common.Uint256FromReversedHexString(sDraftHash)
	if err != nil {
		return errors.New("draft hash from hex string err: " + err.Error())
	}

	sDraftData := c.String(cmdcom.TransactionDraftDataFlag.Name)
	draftData, err := common.HexStringToBytes(sDraftData)
	if err != nil {
		return errors.New("draft data from hex string err: " + err.Error())
	}

	sReceivedCustomIDList := c.String(cmdcom.TransactionReceivedCustomIDListFlag.Name)
	var receivedCustomIDList []string
	for _, s := range strings.Split(sReceivedCustomIDList, "|") {
		receivedCustomIDList = append(receivedCustomIDList, s)
	}

	sReceiverDID := c.String(cmdcom.TransactionReceiverDIDFlag.Name)
	receiverDID, err := common.Uint168FromAddress(sReceiverDID)
	if err != nil {
		return errors.New("invalid receiver did")
	}

	p := &payload.CRCProposal{
		ProposalType:         payload.ReceiveCustomID,
		CategoryData:         categoryData,
		OwnerPublicKey:       ownerPublicKey,
		DraftHash:            *draftHash,
		DraftData:            draftData,
		ReceivedCustomIDList: receivedCustomIDList,
		ReceiverDID:          *receiverDID,
	}

	w := new(bytes.Buffer)
	if err := p.SerializeUnsigned(w, payload.CRCProposalVersion01); err != nil {
		return errors.New("serialize payload err: " + err.Error())
	}

	return outputPayloadAndDigest(w.Bytes())
}

func payloadProposalReceiveCustomIDCRCouncilMemberUnsigned(c *cli.Context) error {
	return payloadProposalCRCouncilMemberUnsigned(c)
}

func createProposalReceiveCustomIDTransaction(c *cli.Context) error {
	return createProposalTransactionCommon(c)
}

func payloadProposalChangeCustomIDFeeOwnerUnsigned(c *cli.Context) error {
	categoryData := c.String(cmdcom.TransactionCategoryDataFlag.Name)

	sPubKey := c.String(cmdcom.TransactionOwnerPublicKeyFlag.Name)
	ownerPublicKey, err := common.HexStringToBytes(sPubKey)
	if err != nil {
		return errors.New("invalid owner pub key: " + err.Error())
	}

	sDraftHash := c.String(cmdcom.TransactionDraftHashFlag.Name)
	draftHash, err := common.Uint256FromReversedHexString(sDraftHash)
	if err != nil {
		return errors.New("draft hash from hex string err: " + err.Error())
	}

	sDraftData := c.String(cmdcom.TransactionDraftDataFlag.Name)
	draftData, err := common.HexStringToBytes(sDraftData)
	if err != nil {
		return errors.New("draft data from hex string err: " + err.Error())
	}

	sCustomIDFeeRate := c.String(cmdcom.TransactionCustomIDFeeRateInfoFlag.Name)
	splits := strings.Split(sCustomIDFeeRate, "|")
	if len(splits) != 2 {
		return errors.New("invalid custom id fee rate")
	}

	rate, err := strconv.ParseUint(splits[0], 10, 64)
	if err != nil {
		return errors.New("invalid custom id fee rate: " + err.Error())
	}
	height, err := strconv.ParseUint(splits[1], 10, 32)
	if err != nil {
		return errors.New("invalid custom id fee rate: " + err.Error())
	}

	customIDFeeRate := payload.CustomIDFeeRateInfo{
		RateOfCustomIDFee:  common.Fixed64(rate),
		EIDEffectiveHeight: uint32(height),
	}

	p := &payload.CRCProposal{
		ProposalType:        payload.ChangeCustomIDFee,
		CategoryData:        categoryData,
		OwnerPublicKey:      ownerPublicKey,
		DraftHash:           *draftHash,
		DraftData:           draftData,
		CustomIDFeeRateInfo: customIDFeeRate,
	}

	w := new(bytes.Buffer)
	if err := p.SerializeUnsigned(w, payload.CRCProposalVersion01); err != nil {
		return errors.New("serialize payload err: " + err.Error())
	}

	return outputPayloadAndDigest(w.Bytes())
}

func payloadProposalChangeCustomIDFeeCRCouncilMemberUnsigned(c *cli.Context) error {
	return payloadProposalCRCouncilMemberUnsigned(c)
}

func createProposalChangeCustomIDFeeTransaction(c *cli.Context) error {
	return createProposalTransactionCommon(c)
}

func payloadProposalRegisterSidechainOwnerUnsigned(c *cli.Context) error {
	categoryData := c.String(cmdcom.TransactionCategoryDataFlag.Name)

	sPubKey := c.String(cmdcom.TransactionOwnerPublicKeyFlag.Name)
	ownerPublicKey, err := common.HexStringToBytes(sPubKey)
	if err != nil {
		return errors.New("invalid owner pub key: " + err.Error())
	}

	sDraftHash := c.String(cmdcom.TransactionDraftHashFlag.Name)
	draftHash, err := common.Uint256FromReversedHexString(sDraftHash)
	if err != nil {
		return errors.New("draft hash from hex string err: " + err.Error())
	}

	sDraftData := c.String(cmdcom.TransactionDraftDataFlag.Name)
	draftData, err := common.HexStringToBytes(sDraftData)
	if err != nil {
		return errors.New("draft data from hex string err: " + err.Error())
	}

	sSidechainInfo := c.String(cmdcom.TransactionRegisterSideChainFlag.Name)
	splits := strings.Split(sSidechainInfo, "|")
	if len(splits) != 6 {
		return errors.New("invalid side chain info")
	}

	sidechainName := splits[0]
	magicNumber, err := strconv.ParseUint(splits[1], 0, 32)
	if err != nil {
		return errors.New("invalid magic number of side chain info: " + err.Error())
	}
	genesisHash, err := common.Uint256FromReversedHexString(splits[2])
	if err != nil {
		return errors.New("invalid genesis hash of side chain info: " + err.Error())
	}
	exchangeRate, err := strconv.ParseUint(splits[3], 0, 64)
	if err != nil {
		return errors.New("invalid exchange rate of side chain info: " + err.Error())
	}
	effectiveHeight, err := strconv.ParseUint(splits[4], 0, 32)
	if err != nil {
		return errors.New("invalid effective height of side chain info: " + err.Error())
	}
	resourcePath := splits[5]
	sidechainInfo := payload.SideChainInfo{
		SideChainName:   sidechainName,
		MagicNumber:     uint32(magicNumber),
		GenesisHash:     *genesisHash,
		ExchangeRate:    common.Fixed64(exchangeRate),
		EffectiveHeight: uint32(effectiveHeight),
		ResourcePath:    resourcePath,
	}

	p := &payload.CRCProposal{
		ProposalType:   payload.RegisterSideChain,
		CategoryData:   categoryData,
		OwnerPublicKey: ownerPublicKey,
		DraftHash:      *draftHash,
		DraftData:      draftData,
		SideChainInfo:  sidechainInfo,
	}

	w := new(bytes.Buffer)
	if err := p.SerializeUnsigned(w, payload.CRCProposalVersion01); err != nil {
		return errors.New("serialize payload err: " + err.Error())
	}

	return outputPayloadAndDigest(w.Bytes())
}

func payloadProposalRegisterSidechainCRCouncilMemberUnsigned(c *cli.Context) error {
	return payloadProposalCRCouncilMemberUnsigned(c)
}

func createProposalRegisterSidechainTransaction(c *cli.Context) error {
	return createProposalTransactionCommon(c)
}

var proposalNormalOwnerPayload = cli.Command{
	Name:  "ownerpayload",
	Usage: "Generate owner unsigned payload",
	Flags: []cli.Flag{
		cmdcom.TransactionCategoryDataFlag,
		cmdcom.TransactionOwnerPublicKeyFlag,
		cmdcom.TransactionDraftHashFlag,
		cmdcom.TransactionDraftDataFlag,
		cmdcom.TransactionBudgetsFlag,
		cmdcom.TransactionRecipientFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalNormalOwnerUnsigned(c); err != nil {
			fmt.Println("error: ", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalNormalCRCouncilMemberPayload = cli.Command{
	Name:  "crcouncilmemberpayload",
	Usage: "Generate CR council member unsigned payload",
	Flags: []cli.Flag{
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionOwnerSignatureFlag,
		cmdcom.TransactionCRCouncilMemberDIDFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalNormalCRCouncilMemberUnsigned(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalNormal = cli.Command{
	Name:  "normal",
	Usage: "Build a normal proposal tx",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionCRCouncilMemberSignatureFlag,
	},
	Subcommands: []cli.Command{
		proposalNormalOwnerPayload,
		proposalNormalCRCouncilMemberPayload,
	},
	Action: func(c *cli.Context) error {
		if err := createNormalProposalTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalReviewOwnerPayload = cli.Command{
	Name:  "ownerpayload",
	Usage: "owner unsigned payload",
	Flags: []cli.Flag{
		cmdcom.TransactionProposalHashFlag,
		cmdcom.TransactionVoteResultFlag,
		cmdcom.TransactionOpinionHashFlag,
		cmdcom.TransactionOpinionDataFlag,
		cmdcom.TransactionDIDFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalReviewOwnerUnsigned(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalReview = cli.Command{
	Name:  "review",
	Usage: "Build a proposal review tx",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionOwnerSignatureFlag,
	},
	Subcommands: []cli.Command{
		proposalReviewOwnerPayload,
	},
	Action: func(c *cli.Context) error {
		if err := createProposalReviewTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalTrackingOwnerPayload = cli.Command{
	Name:  "ownerpayload",
	Usage: "Generate owner unsigned payload",
	Flags: []cli.Flag{
		cmdcom.TransactionProposalHashFlag,
		cmdcom.TransactionMessageHashFlag,
		cmdcom.TransactionMessageDataFlag,
		cmdcom.CRCProposalStageFlag,
		cmdcom.TransactionOwnerPublicKeyFlag,
		cmdcom.TransactionNewOwnerPublicKeyFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalTrackingOwnerUnsigned(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalTrackingNewOwnerPayload = cli.Command{
	Name:  "newownerpayload",
	Usage: "Generate new owner unsigned payload",
	Flags: []cli.Flag{
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionOwnerSignatureFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalTrackingNewOwnerUnsigned(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalTrackingSecretaryGeneralPayload = cli.Command{
	Name:  "secretarygeneralpayload",
	Usage: "Generate secretary general unsigned payload",
	Flags: []cli.Flag{
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionNewOwnerSignatureFlag,
		cmdcom.TransactionProposalTrackingTypeFlag,
		cmdcom.TransactionSecretaryGeneralOpinionHashFlag,
		cmdcom.TransactionSecretaryGeneralOpinionDataFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalTrackingSecretaryGeneralUnsigned(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalTracking = cli.Command{
	Name:  "tracking",
	Usage: "Build a proposal tracking tx",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionCRCouncilMemberSignatureFlag,
	},
	Subcommands: []cli.Command{
		proposalTrackingOwnerPayload,
		proposalTrackingNewOwnerPayload,
		proposalTrackingSecretaryGeneralPayload,
	},
	Action: func(c *cli.Context) error {
		if err := createProposalTrackingTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalSecretaryGeneralElectionUnsignedPayload = cli.Command{
	Name:  "unsignedpayload",
	Usage: "unsigned payload for owner and secretary general",
	Flags: []cli.Flag{
		cmdcom.TransactionCategoryDataFlag,
		cmdcom.TransactionOwnerPublicKeyFlag,
		cmdcom.TransactionDraftHashFlag,
		cmdcom.TransactionDraftDataFlag,
		cmdcom.TransactionSecretaryPublicKeyFlag,
		cmdcom.TransactionSecretaryDIDFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalSecretaryGeneralElectionUnsigned(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalSecretaryGeneralElectionCRCouncilMemberPayload = cli.Command{
	Name:  "crcouncilmemberpayload",
	Usage: "crcouncil member unsigned payload",
	Flags: []cli.Flag{
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionOwnerSignatureFlag,
		cmdcom.TransactionSecretarySignatureFlag,
		cmdcom.TransactionCRCouncilMemberDIDFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalSecretaryGeneralElectionCRCouncilMemberUnsigned(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalSecretaryGeneralElection = cli.Command{
	Name:  "election",
	Usage: "Build a proposal secretary general election tx",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionCRCouncilMemberSignatureFlag,
	},
	Subcommands: []cli.Command{
		proposalSecretaryGeneralElectionUnsignedPayload,
		proposalSecretaryGeneralElectionCRCouncilMemberPayload,
	},
	Action: func(c *cli.Context) error {
		if err := createProposalSecretaryGeneralElectionTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalChangeOwnerUnsignedPayload = cli.Command{
	Name:  "unsignedpayload",
	Usage: "unsigned payload for owner and new owner",
	Flags: []cli.Flag{
		cmdcom.TransactionCategoryDataFlag,
		cmdcom.TransactionOwnerPublicKeyFlag,
		cmdcom.TransactionDraftHashFlag,
		cmdcom.TransactionDraftDataFlag,
		cmdcom.TransactionTargetProposalHashFlag,
		cmdcom.TransactionNewRecipientFlag,
		cmdcom.TransactionNewOwnerPublicKeyFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalChangeOwnerUnsigned(c); err != nil {
			fmt.Println("error: ", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalChangeOwnerCRCouncilMemberUnsignedPayload = cli.Command{
	Name:  "crcouncilmemberpayload",
	Usage: "crcouncil member payload",
	Flags: []cli.Flag{
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionOwnerSignatureFlag,
		cmdcom.TransactionNewOwnerSignatureFlag,
		cmdcom.TransactionCRCouncilMemberDIDFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalChangeOwnerCRCouncilMemberUnsigned(c); err != nil {
			fmt.Println("error: ", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalChangeOwner = cli.Command{
	Name:  "changeowner",
	Usage: "Build a proposal change owner tx",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionCRCouncilMemberSignatureFlag,
	},
	Subcommands: []cli.Command{
		proposalChangeOwnerUnsignedPayload,
		proposalChangeOwnerCRCouncilMemberUnsignedPayload,
	},
	Action: func(c *cli.Context) error {
		if err := createProposalChangeOwnerTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalTerminateOwnerUnsignedPayload = cli.Command{
	Name:  "ownerpayload",
	Usage: "owner unsigned payload",
	Flags: []cli.Flag{
		cmdcom.TransactionCategoryDataFlag,
		cmdcom.TransactionOwnerPublicKeyFlag,
		cmdcom.TransactionDraftHashFlag,
		cmdcom.TransactionDraftDataFlag,
		cmdcom.TransactionTargetProposalHashFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalTerminateOwnerUnsigned(c); err != nil {
			fmt.Println("error: ", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalTerminateCRCouncilMemberUnsignedPayload = cli.Command{
	Name:  "crcouncilmemberpayload",
	Usage: "crcouncil member unsigned payload",
	Flags: []cli.Flag{
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionOwnerSignatureFlag,
		cmdcom.TransactionCRCouncilMemberDIDFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalTerminateCRCouncilMemberUnsigned(c); err != nil {
			fmt.Println("error: ", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalTerminate = cli.Command{
	Name:  "terminate",
	Usage: "Build a proposal terminate tx",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionCRCouncilMemberSignatureFlag,
	},
	Subcommands: []cli.Command{
		proposalTerminateOwnerUnsignedPayload,
		proposalTerminateCRCouncilMemberUnsignedPayload,
	},
	Action: func(c *cli.Context) error {
		if err := createProposalTerminateTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalReserverCustomIDOwnerUnsignedPayload = cli.Command{
	Name:  "ownerpayload",
	Usage: "owner unsigned payload",
	Flags: []cli.Flag{
		cmdcom.TransactionCategoryDataFlag,
		cmdcom.TransactionOwnerPublicKeyFlag,
		cmdcom.TransactionDraftHashFlag,
		cmdcom.TransactionDraftDataFlag,
		cmdcom.TransactionReservedCustomIDListFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalReserverCustomIDOwnerUnsigned(c); err != nil {
			fmt.Println("error: ", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalReserverCustomIDCRCouncilMemberUnsignedPayload = cli.Command{
	Name:  "crcouncilmemberpayload",
	Usage: "crcouncil member unsigned payload",
	Flags: []cli.Flag{
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionOwnerSignatureFlag,
		cmdcom.TransactionCRCouncilMemberDIDFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalReserverCustomIDCRCouncilMemberUnsigned(c); err != nil {
			fmt.Println("error: ", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalReserveCustomID = cli.Command{
	Name:  "reservecustomid",
	Usage: "Build a proposal reserve custom id tx",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionCRCouncilMemberSignatureFlag,
	},
	Subcommands: []cli.Command{
		proposalReserverCustomIDOwnerUnsignedPayload,
		proposalReserverCustomIDCRCouncilMemberUnsignedPayload,
	},
	Action: func(c *cli.Context) error {
		if err := createProposalReserveCustomIDTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalReceiveCustomIDOwnerUnsignedPayload = cli.Command{
	Name:  "ownerpayload",
	Usage: "owner unsigned payload",
	Flags: []cli.Flag{
		cmdcom.TransactionCategoryDataFlag,
		cmdcom.TransactionOwnerPublicKeyFlag,
		cmdcom.TransactionDraftHashFlag,
		cmdcom.TransactionDraftDataFlag,
		cmdcom.TransactionReceivedCustomIDListFlag,
		cmdcom.TransactionReceiverDIDFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalReceiveCustomIDOwnerUnsigned(c); err != nil {
			fmt.Println("error: ", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalReceiveCustomIDCRCouncilMemberUnsignedPayload = cli.Command{
	Name:  "crcouncilmemberpayload",
	Usage: "crcouncil member unsigned payload",
	Flags: []cli.Flag{
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionOwnerSignatureFlag,
		cmdcom.TransactionCRCouncilMemberDIDFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalReceiveCustomIDCRCouncilMemberUnsigned(c); err != nil {
			fmt.Println("error: ", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalReceiveCustomID = cli.Command{
	Name:  "receivecustomid",
	Usage: "Build a proposal receive custom id tx",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionCRCouncilMemberSignatureFlag,
	},
	Subcommands: []cli.Command{
		proposalReceiveCustomIDOwnerUnsignedPayload,
		proposalReceiveCustomIDCRCouncilMemberUnsignedPayload,
	},
	Action: func(c *cli.Context) error {
		if err := createProposalReceiveCustomIDTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalChangeCustomIDFeeOwnerUnsignedPayload = cli.Command{
	Name:  "ownerpayload",
	Usage: "owner unsigned payload",
	Flags: []cli.Flag{
		cmdcom.TransactionCategoryDataFlag,
		cmdcom.TransactionOwnerPublicKeyFlag,
		cmdcom.TransactionDraftHashFlag,
		cmdcom.TransactionDraftDataFlag,
		cmdcom.TransactionCustomIDFeeRateInfoFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalChangeCustomIDFeeOwnerUnsigned(c); err != nil {
			fmt.Println("error: ", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalChangeCustomIDFeeCRCouncilMemberUnsignedPayload = cli.Command{
	Name:  "crcouncilmemberpayload",
	Usage: "crcouncil member unsigned payload",
	Flags: []cli.Flag{
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionOwnerSignatureFlag,
		cmdcom.TransactionCRCouncilMemberDIDFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalChangeCustomIDFeeCRCouncilMemberUnsigned(c); err != nil {
			fmt.Println("error: ", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalChangeCustomIDFee = cli.Command{
	Name:  "changecustomidfee",
	Usage: "Build a proposal change custom id fee tx",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionCRCouncilMemberSignatureFlag,
	},
	Subcommands: []cli.Command{
		proposalChangeCustomIDFeeOwnerUnsignedPayload,
		proposalChangeCustomIDFeeCRCouncilMemberUnsignedPayload,
	},
	Action: func(c *cli.Context) error {
		if err := createProposalChangeCustomIDFeeTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalRegisterSidechainOwnerUnsignedPayload = cli.Command{
	Name:  "ownerpayload",
	Usage: "owner unsigned payload",
	Flags: []cli.Flag{
		cmdcom.TransactionCategoryDataFlag,
		cmdcom.TransactionOwnerPublicKeyFlag,
		cmdcom.TransactionDraftHashFlag,
		cmdcom.TransactionDraftDataFlag,
		cmdcom.TransactionRegisterSideChainFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalRegisterSidechainOwnerUnsigned(c); err != nil {
			fmt.Println("error: ", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalRegisterSidechainCRCouncilMemberUnsignedPayload = cli.Command{
	Name:  "crcouncilmemberpayload",
	Usage: "crcouncil member unsigned payload",
	Flags: []cli.Flag{
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionOwnerSignatureFlag,
		cmdcom.TransactionCRCouncilMemberDIDFlag,
	},
	Action: func(c *cli.Context) error {
		if err := payloadProposalRegisterSidechainCRCouncilMemberUnsigned(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposalRegisterSidechain = cli.Command{
	Name:  "registersidechain",
	Usage: "Build a proposal register sidechain tx",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.TransactionPayloadFlag,
		cmdcom.TransactionCRCouncilMemberSignatureFlag,
	},
	Subcommands: []cli.Command{
		proposalRegisterSidechainOwnerUnsignedPayload,
		proposalRegisterSidechainCRCouncilMemberUnsignedPayload,
	},
	Action: func(c *cli.Context) error {
		if err := createProposalRegisterSidechainTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var proposal = cli.Command{
	Name:  "proposal",
	Usage: "Build a proposal tx",
	Flags: []cli.Flag{},
	Subcommands: []cli.Command{
		proposalNormal,
		proposalReview,
		proposalTracking,
		proposalSecretaryGeneralElection,
		proposalChangeOwner,
		proposalTerminate,
		proposalReserveCustomID,
		proposalReceiveCustomID,
		proposalChangeCustomIDFee,
		proposalRegisterSidechain,
	},
	Action: func(c *cli.Context) error {
		cli.ShowSubcommandHelp(c)
		return nil
	},
}
