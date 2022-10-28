package transaction

import (
	"bytes"
	"encoding/hex"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
)

func (s *txValidatorTestSuite) TestCheckSecretaryGeneralProposalTransaction() {

	ownerPublicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	ownerPrivateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"

	crPublicKeyStr := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	crPrivateKeyStr := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"

	secretaryPublicKeyStr := "031e12374bae471aa09ad479f66c2306f4bcc4ca5b754609a82a1839b94b4721b9"
	secretaryPrivateKeyStr := "94396a69462208b8fd96d83842855b867d3b0e663203cb31d0dfaec0362ec034"

	tenureHeight := config.DefaultParams.CRConfiguration.CRCommitteeStartHeight + 1
	ownerNickName := "nickname owner"
	crNickName := "nickname cr"

	memberOwner := s.getCRMember(ownerPublicKeyStr1, ownerPrivateKeyStr1, ownerNickName)
	memberCr := s.getCRMember(crPublicKeyStr, crPrivateKeyStr, crNickName)

	memebers := make(map[common.Uint168]*crstate.CRMember)

	s.Chain.GetCRCommittee().Members = memebers
	s.Chain.GetCRCommittee().CRCCommitteeBalance = common.Fixed64(100 * 1e8)
	s.Chain.GetCRCommittee().CRCCurrentStageAmount = common.Fixed64(100 * 1e8)
	s.Chain.GetCRCommittee().InElectionPeriod = true
	s.Chain.GetCRCommittee().NeedAppropriation = false

	//owner not elected cr
	txn := s.getSecretaryGeneralCRCProposalTx(ownerPublicKeyStr1, ownerPrivateKeyStr1, crPublicKeyStr, crPrivateKeyStr,
		secretaryPublicKeyStr, secretaryPrivateKeyStr)

	//CRCouncilMember not elected cr
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ := txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CR Council Member should be one of the CR members")
	memebers[memberCr.Info.DID] = memberCr
	memebers[memberOwner.Info.DID] = memberOwner

	//owner signature check failed
	rightSign := txn.Payload().(*payload.CRCProposal).Signature
	txn.Payload().(*payload.CRCProposal).Signature = []byte{}
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:owner signature check failed")
	txn.Payload().(*payload.CRCProposal).Signature = rightSign

	//SecretaryGeneral signature check failed
	secretaryGeneralSign := txn.Payload().(*payload.CRCProposal).SecretaryGeneraSignature
	txn.Payload().(*payload.CRCProposal).SecretaryGeneraSignature = []byte{}
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:SecretaryGeneral signature check failed")
	txn.Payload().(*payload.CRCProposal).SecretaryGeneraSignature = secretaryGeneralSign

	//CRCouncilMemberSignature signature check failed
	crcouncilMemberSignature := txn.Payload().(*payload.CRCProposal).CRCouncilMemberSignature
	txn.Payload().(*payload.CRCProposal).CRCouncilMemberSignature = []byte{}
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CR Council Member signature check failed")
	txn.Payload().(*payload.CRCProposal).CRCouncilMemberSignature = crcouncilMemberSignature

	//SecretaryGeneralPublicKey and SecretaryGeneralDID not match
	secretaryGeneralPublicKey := txn.Payload().(*payload.CRCProposal).SecretaryGeneralPublicKey
	txn.Payload().(*payload.CRCProposal).SecretaryGeneralPublicKey, _ = common.HexStringToBytes(ownerPublicKeyStr1)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:SecretaryGeneral NodePublicKey and DID is not matching")
	txn.Payload().(*payload.CRCProposal).SecretaryGeneralPublicKey = secretaryGeneralPublicKey

	// ok
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	//ChangeSecretaryGeneralProposal tx must InElectionPeriod and not during voting period
	config.DefaultParams.DPoSV2StartHeight = 2000000
	s.Chain.GetCRCommittee().LastCommitteeHeight = config.DefaultParams.CRConfiguration.CRCommitteeStartHeight
	tenureHeight = config.DefaultParams.CRConfiguration.CRCommitteeStartHeight + config.DefaultParams.CRConfiguration.DutyPeriod -
		config.DefaultParams.CRConfiguration.VotingPeriod + 1
	txn.SetParameters(&TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              &config.DefaultParams,
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:cr proposal tx must not during voting period")
}

func (s *txValidatorTestSuite) getSecretaryGeneralCRCProposalTx(ownerPublicKeyStr, ownerPrivateKeyStr,
	crPublicKeyStr, crPrivateKeyStr, secretaryPublicKeyStr, secretaryPrivateKeyStr string) interfaces.Transaction {

	ownerPublicKey, _ := common.HexStringToBytes(ownerPublicKeyStr)
	ownerPrivateKey, _ := common.HexStringToBytes(ownerPrivateKeyStr)

	secretaryPublicKey, _ := common.HexStringToBytes(secretaryPublicKeyStr)
	secretaryGeneralDID, _ := blockchain.GetDiDFromPublicKey(secretaryPublicKey)
	secretaryGeneralPrivateKey, _ := common.HexStringToBytes(secretaryPrivateKeyStr)

	crPrivateKey, _ := common.HexStringToBytes(crPrivateKeyStr)
	crCode := getCodeByPubKeyStr(crPublicKeyStr)

	draftData := randomBytes(10)
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposal,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	recipient := *randomUint168()
	recipient[0] = uint8(contract.PrefixStandard)
	crDID, _ := blockchain.GetDIDFromCode(crCode)
	crcProposalPayload := &payload.CRCProposal{
		ProposalType:              payload.SecretaryGeneral,
		CategoryData:              "111",
		OwnerPublicKey:            ownerPublicKey,
		DraftHash:                 common.Hash(draftData),
		SecretaryGeneralPublicKey: secretaryPublicKey,
		SecretaryGeneralDID:       *secretaryGeneralDID,
		CRCouncilMemberDID:        *crDID,
	}

	signBuf := new(bytes.Buffer)
	crcProposalPayload.SerializeUnsigned(signBuf, payload.CRCProposalVersion)
	sig, _ := crypto.Sign(ownerPrivateKey, signBuf.Bytes())
	crcProposalPayload.Signature = sig

	secretaryGeneralSig, _ := crypto.Sign(secretaryGeneralPrivateKey, signBuf.Bytes())
	crcProposalPayload.SecretaryGeneraSignature = secretaryGeneralSig

	common.WriteVarBytes(signBuf, sig)
	common.WriteVarBytes(signBuf, secretaryGeneralSig)
	crcProposalPayload.CRCouncilMemberDID.Serialize(signBuf)
	crSig, _ := crypto.Sign(crPrivateKey, signBuf.Bytes())
	crcProposalPayload.CRCouncilMemberSignature = crSig

	txn.SetPayload(crcProposalPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(ownerPublicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func (s *txValidatorTestSuite) TestCheckCRCProposalRegisterSideChainTransaction() {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"

	tenureHeight := config.DefaultParams.CRConfiguration.CRCommitteeStartHeight + 1
	nickName1 := "nickname 1"

	member1 := s.getCRMember(publicKeyStr1, privateKeyStr1, nickName1)
	memebers := make(map[common.Uint168]*crstate.CRMember)
	memebers[member1.Info.DID] = member1
	s.Chain.GetCRCommittee().Members = memebers
	s.Chain.GetCRCommittee().CRCCommitteeBalance = common.Fixed64(100 * 1e8)
	s.Chain.GetCRCommittee().CRCCurrentStageAmount = common.Fixed64(100 * 1e8)
	s.Chain.GetCRCommittee().InElectionPeriod = true
	s.Chain.GetCRCommittee().NeedAppropriation = false

	{
		// no error
		txn := s.getCRCRegisterSideChainProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
		txn = CreateTransactionByType(txn, s.Chain)
		txn.SetParameters(&TransactionParameters{
			Transaction:         txn,
			BlockHeight:         tenureHeight,
			TimeStamp:           s.Chain.BestChain.Timestamp,
			Config:              s.Chain.GetParams(),
			BlockChain:          s.Chain,
			ProposalsUsedAmount: 0,
		})
		err, _ := txn.SpecialContextCheck()
		s.NoError(err)

		// genesis hash can not be blank
		payload, _ := txn.Payload().(*payload.CRCProposal)
		payload.GenesisHash = common.Uint256{}
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err, "transaction validate error: payload content invalid:GenesisHash can not be empty")
	}

	{
		txn := s.getCRCRegisterSideChainProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
		payload, _ := txn.Payload().(*payload.CRCProposal)
		payload.SideChainName = ""
		txn = CreateTransactionByType(txn, s.Chain)
		txn.SetParameters(&TransactionParameters{
			Transaction:         txn,
			BlockHeight:         tenureHeight,
			TimeStamp:           s.Chain.BestChain.Timestamp,
			Config:              s.Chain.GetParams(),
			BlockChain:          s.Chain,
			ProposalsUsedAmount: 0,
		})
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err, "transaction validate error: payload content invalid:SideChainName can not be empty")
	}

	{
		s.Chain.GetCRCommittee().GetProposalManager().RegisteredSideChainNames = []string{"NEO"}
		txn := s.getCRCRegisterSideChainProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
		txn = CreateTransactionByType(txn, s.Chain)
		txn.SetParameters(&TransactionParameters{
			Transaction:         txn,
			BlockHeight:         tenureHeight,
			TimeStamp:           s.Chain.BestChain.Timestamp,
			Config:              s.Chain.GetParams(),
			BlockChain:          s.Chain,
			ProposalsUsedAmount: 0,
		})
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err, "transaction validate error: payload content invalid:SideChainName already registered")
	}

}

func (s *txValidatorTestSuite) getCRCRegisterSideChainProposalTx(publicKeyStr, privateKeyStr,
	crPublicKeyStr, crPrivateKeyStr string) interfaces.Transaction {

	normalPrivateKey, _ := common.HexStringToBytes(privateKeyStr)
	normalPublicKey, _ := common.HexStringToBytes(publicKeyStr)
	crPrivateKey, _ := common.HexStringToBytes(crPrivateKeyStr)

	draftData := randomBytes(10)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposal,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	CRCouncilMemberDID, _ := blockchain.GetDIDFromCode(getCodeByPubKeyStr(crPublicKeyStr))
	crcProposalPayload := &payload.CRCProposal{
		ProposalType:       payload.RegisterSideChain,
		OwnerPublicKey:     normalPublicKey,
		CRCouncilMemberDID: *CRCouncilMemberDID,
		DraftHash:          common.Hash(draftData),
		SideChainInfo: payload.SideChainInfo{
			SideChainName:   "NEO",
			MagicNumber:     100,
			GenesisHash:     *randomUint256(),
			ExchangeRate:    100000000,
			EffectiveHeight: 100000,
		},
	}

	signBuf := new(bytes.Buffer)
	crcProposalPayload.SerializeUnsigned(signBuf, payload.CRCProposalVersion)

	sig, _ := crypto.Sign(normalPrivateKey, signBuf.Bytes())
	crcProposalPayload.Signature = sig

	common.WriteVarBytes(signBuf, sig)
	crcProposalPayload.CRCouncilMemberDID.Serialize(signBuf)
	crSig, _ := crypto.Sign(crPrivateKey, signBuf.Bytes())
	crcProposalPayload.CRCouncilMemberSignature = crSig

	txn.SetPayload(crcProposalPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func (s *txValidatorTestSuite) TestCheckCRCProposalTransaction() {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"

	tenureHeight := config.DefaultParams.CRConfiguration.CRCommitteeStartHeight + 1
	nickName1 := "nickname 1"
	nickName2 := "nickname 2"

	member1 := s.getCRMember(publicKeyStr1, privateKeyStr1, nickName1)
	memebers := make(map[common.Uint168]*crstate.CRMember)
	memebers[member1.Info.DID] = member1
	s.Chain.GetCRCommittee().Members = memebers
	s.Chain.GetCRCommittee().CRCCommitteeBalance = common.Fixed64(100 * 1e8)
	s.Chain.GetCRCommittee().CRCCurrentStageAmount = common.Fixed64(100 * 1e8)
	s.Chain.GetCRCommittee().InElectionPeriod = true
	s.Chain.GetCRCommittee().NeedAppropriation = false

	// ok
	txn := s.getCRCProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)

	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ := txn.SpecialContextCheck()
	s.NoError(err)

	// member status is not elected
	member1.MemberState = crstate.MemberImpeached
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CR Council Member should be an elected CR members")

	// register cr proposal in voting period
	member1.MemberState = crstate.MemberElected
	tenureHeight = config.DefaultParams.CRConfiguration.CRCommitteeStartHeight +
		config.DefaultParams.CRConfiguration.DutyPeriod - config.DefaultParams.CRConfiguration.VotingPeriod
	s.Chain.GetCRCommittee().InElectionPeriod = false
	txn.SetParameters(&TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:cr proposal tx must not during voting period")

	// recipient is empty
	s.Chain.GetCRCommittee().InElectionPeriod = true
	tenureHeight = config.DefaultParams.CRConfiguration.CRCommitteeStartHeight + 1
	txn.Payload().(*payload.CRCProposal).Recipient = common.Uint168{}
	txn.SetParameters(&TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:recipient is empty")

	// invalid payload
	txn.SetPayload(&payload.CRInfo{})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid payload")

	// invalid proposal type
	txn = s.getCRCProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
	txn.Payload().(*payload.CRCProposal).ProposalType = 0x1000
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:type of proposal should be known")

	// invalid outputs of ELIP.
	txn.Payload().(*payload.CRCProposal).ProposalType = 0x0100
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:ELIP needs to have and only have two budget")

	// invalid budgets.
	txn.Payload().(*payload.CRCProposal).ProposalType = 0x0000
	s.Chain.GetCRCommittee().CRCCommitteeBalance = common.Fixed64(10 * 1e8)
	s.Chain.GetCRCommittee().CRCCurrentStageAmount = common.Fixed64(10 * 1e8)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:budgets exceeds 10% of CRC committee balance")

	s.Chain.GetCRCommittee().CRCCommitteeBalance = common.Fixed64(100 * 1e8)
	s.Chain.GetCRCommittee().CRCCurrentStageAmount = common.Fixed64(100 * 1e8)
	s.Chain.GetCRCommittee().CRCCommitteeUsedAmount = common.Fixed64(99 * 1e8)
	err, _ = txn.SpecialContextCheck()
	s.Error(err, "transaction validate error: payload content invalid:budgets exceeds the balance of CRC committee")

	s.Chain.GetCRCommittee().CRCCommitteeUsedAmount = common.Fixed64(0)

	// CRCouncilMemberSignature is not signed by CR member
	txn = s.getCRCProposalTx(publicKeyStr1, privateKeyStr1, publicKeyStr2, privateKeyStr2)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CR Council Member should be one of the CR members")

	// invalid owner
	txn = s.getCRCProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
	txn.Payload().(*payload.CRCProposal).OwnerPublicKey = []byte{}
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid owner")

	// invalid owner signature
	txn = s.getCRCProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	txn.Payload().(*payload.CRCProposal).OwnerPublicKey = publicKey1
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:owner signature check failed")

	// invalid CR owner signature
	txn = s.getCRCProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
	txn.Payload().(*payload.CRCProposal).CRCouncilMemberSignature = []byte{}
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:failed to check CR Council Member signature")

	// proposal status is not VoterAgreed
	newOwnerPublicKeyStr := publicKeyStr2
	publicKey2, _ := hex.DecodeString(publicKeyStr2)
	proposalState, proposal := s.createSpecificStatusProposal(publicKey1, publicKey2, tenureHeight,
		crstate.Registered, payload.Normal)

	s.Chain.GetCRCommittee().GetProposalManager().Proposals[proposal.Hash(payload.CRCProposalVersion01)] = proposalState

	txn = s.getCRChangeProposalOwnerProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1,
		newOwnerPublicKeyStr, proposal.Hash(payload.CRCProposalVersion01))
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:proposal status is not VoterAgreed")

	//proposal sponsors must be members
	targetHash := proposal.Hash(payload.CRCProposalVersion01)
	newOwnerPublicKey, _ := hex.DecodeString(newOwnerPublicKeyStr)
	proposalState2, proposal2 := s.createSpecificStatusProposal(publicKey1, publicKey2, tenureHeight+1,
		crstate.VoterAgreed, payload.ChangeProposalOwner)
	proposal2.TargetProposalHash = targetHash
	proposal2.OwnerPublicKey = newOwnerPublicKey
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[targetHash] = proposalState2
	txn = s.getCRChangeProposalOwnerProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1,
		newOwnerPublicKeyStr, targetHash)

	s.Chain.GetCRCommittee().InElectionPeriod = false
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:cr proposal tx must not during voting period")

	// invalid proposal owner
	s.Chain.GetCRCommittee().InElectionPeriod = true
	proposalState3, proposal3 := s.createSpecificStatusProposal(publicKey1, publicKey2, tenureHeight,
		crstate.Registered, payload.Normal)
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[proposal3.Hash(payload.CRCProposalVersion01)] = proposalState3

	txn = s.getCRCCloseProposalTxWithHash(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1,
		proposal.Hash(payload.CRCProposalVersion01))

	// invalid closeProposalHash
	txn = s.getCRCCloseProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CloseProposalHash does not exist")

	// invalid proposal status
	hash := proposal.Hash(payload.CRCProposalVersion01)
	member2 := s.getCRMember(publicKeyStr2, privateKeyStr2, nickName2)
	memebers[member2.Info.DID] = member2
	txn = s.getCRCCloseProposalTxWithHash(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1,
		proposal.Hash(payload.CRCProposalVersion01))

	proposalState.Status = crstate.Registered
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[hash] = proposalState
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CloseProposalHash has to be voterAgreed")

	// invalid receipt
	proposalState, proposal = s.createSpecificStatusProposal(publicKey1, publicKey2, tenureHeight,
		crstate.VoterAgreed, payload.Normal)
	hash = proposal.Hash(payload.CRCProposalVersion01)
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[hash] = proposalState
	txn = s.getCRCCloseProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
	txn.Payload().(*payload.CRCProposal).TargetProposalHash = hash
	txn.Payload().(*payload.CRCProposal).Recipient = *randomUint168()
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CloseProposal recipient must be empty")

	// invalid budget
	txn = s.getCRCCloseProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
	txn.Payload().(*payload.CRCProposal).TargetProposalHash = hash
	txn.Payload().(*payload.CRCProposal).Budgets = []payload.Budget{{
		payload.Imprest,
		0x01,
		common.Fixed64(10000000000),
	}}
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CloseProposal cannot have budget")

	// proposals can not more than MaxCommitteeProposalCount
	txn = s.getCRCProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
	crcProposal, _ := txn.Payload().(*payload.CRCProposal)
	proposalHashSet := crstate.NewProposalHashSet()
	for i := 0; i < int(s.Chain.GetParams().CRConfiguration.MaxCommitteeProposalCount); i++ {
		proposalHashSet.Add(*randomUint256())
	}
	s.Chain.GetCRCommittee().GetProposalManager().ProposalHashes[crcProposal.
		CRCouncilMemberDID] = proposalHashSet
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:proposal is full")

	s.Chain.GetParams().CRConfiguration.MaxCommitteeProposalCount = s.Chain.GetParams().CRConfiguration.MaxCommitteeProposalCount + 100
	// invalid reserved custom id
	txn = s.getCRCReservedCustomIDProposalTx(publicKeyStr2, privateKeyStr2, publicKeyStr1, privateKeyStr1)
	proposal, _ = txn.Payload().(*payload.CRCProposal)
	proposal.ReservedCustomIDList = append(proposal.ReservedCustomIDList, randomName(260))
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction:         txn,
		BlockHeight:         tenureHeight,
		TimeStamp:           s.Chain.BestChain.Timestamp,
		Config:              s.Chain.GetParams(),
		BlockChain:          s.Chain,
		ProposalsUsedAmount: 0,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid reserved custom id length")
}

func (s *txValidatorTestSuite) getCRCProposalTx(publicKeyStr, privateKeyStr,
	crPublicKeyStr, crPrivateKeyStr string) interfaces.Transaction {

	publicKey1, _ := common.HexStringToBytes(publicKeyStr)
	privateKey1, _ := common.HexStringToBytes(privateKeyStr)

	privateKey2, _ := common.HexStringToBytes(crPrivateKeyStr)
	code2 := getCodeByPubKeyStr(crPublicKeyStr)

	draftData := randomBytes(10)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposal,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	recipient := *randomUint168()
	recipient[0] = uint8(contract.PrefixStandard)
	did2, _ := blockchain.GetDIDFromCode(code2)
	crcProposalPayload := &payload.CRCProposal{
		ProposalType:       payload.Normal,
		OwnerPublicKey:     publicKey1,
		CRCouncilMemberDID: *did2,
		DraftHash:          common.Hash(draftData),
		Budgets:            createBudgets(3),
		Recipient:          recipient,
	}

	signBuf := new(bytes.Buffer)
	crcProposalPayload.SerializeUnsigned(signBuf, payload.CRCProposalVersion)
	sig, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crcProposalPayload.Signature = sig

	common.WriteVarBytes(signBuf, sig)
	crcProposalPayload.CRCouncilMemberDID.Serialize(signBuf)
	crSig, _ := crypto.Sign(privateKey2, signBuf.Bytes())
	crcProposalPayload.CRCouncilMemberSignature = crSig

	txn.SetPayload(crcProposalPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func (s *txValidatorTestSuite) getCRChangeProposalOwnerProposalTx(publicKeyStr, privateKeyStr,
	crPublicKeyStr, crPrivateKeyStr, newOwnerPublicKeyStr string, targetHash common.Uint256) interfaces.Transaction {

	privateKey, _ := common.HexStringToBytes(privateKeyStr)
	crPrivateKey, _ := common.HexStringToBytes(crPrivateKeyStr)
	crPublicKey, _ := common.HexStringToBytes(crPublicKeyStr)
	crDid, _ := blockchain.GetDIDFromCode(getCodeByPubKeyStr(crPublicKeyStr))
	newOwnerPublicKey, _ := common.HexStringToBytes(newOwnerPublicKeyStr)
	draftData := randomBytes(10)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposal,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	crcProposalPayload := &payload.CRCProposal{
		ProposalType:       payload.ChangeProposalOwner,
		OwnerPublicKey:     crPublicKey,
		NewOwnerPublicKey:  newOwnerPublicKey,
		TargetProposalHash: targetHash,
		DraftHash:          common.Hash(draftData),
		CRCouncilMemberDID: *crDid,
	}

	signBuf := new(bytes.Buffer)
	crcProposalPayload.SerializeUnsigned(signBuf, payload.CRCProposalVersion)
	sig, _ := crypto.Sign(privateKey, signBuf.Bytes())
	crcProposalPayload.Signature = sig

	common.WriteVarBytes(signBuf, sig)
	crcProposalPayload.CRCouncilMemberDID.Serialize(signBuf)
	crSig, _ := crypto.Sign(crPrivateKey, signBuf.Bytes())
	crcProposalPayload.CRCouncilMemberSignature = crSig

	txn.SetPayload(crcProposalPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func (s *txValidatorTestSuite) createSpecificStatusProposal(publicKey1, publicKey2 []byte, height uint32,
	status crstate.ProposalStatus, proposalType payload.CRCProposalType) (*crstate.ProposalState, *payload.CRCProposal) {
	draftData := randomBytes(10)
	recipient := *randomUint168()
	recipient[0] = uint8(contract.PrefixStandard)
	code2 := getCodeByPubKeyStr(hex.EncodeToString(publicKey2))
	CRCouncilMemberDID, _ := blockchain.GetDIDFromCode(code2)
	proposal := &payload.CRCProposal{
		ProposalType:       proposalType,
		OwnerPublicKey:     publicKey1,
		CRCouncilMemberDID: *CRCouncilMemberDID,
		DraftHash:          common.Hash(draftData),
		Budgets:            createBudgets(3),
		Recipient:          recipient,
	}
	budgetsStatus := make(map[uint8]crstate.BudgetStatus)
	for _, budget := range proposal.Budgets {
		if budget.Type == payload.Imprest {
			budgetsStatus[budget.Stage] = crstate.Withdrawable
			continue
		}
		budgetsStatus[budget.Stage] = crstate.Unfinished
	}
	proposalState := &crstate.ProposalState{
		Status:              status,
		Proposal:            proposal.ToProposalInfo(0),
		TxHash:              common.Hash(randomBytes(10)),
		CRVotes:             map[common.Uint168]payload.VoteResult{},
		VotersRejectAmount:  common.Fixed64(0),
		RegisterHeight:      height,
		VoteStartHeight:     0,
		WithdrawnBudgets:    make(map[uint8]common.Fixed64),
		WithdrawableBudgets: make(map[uint8]common.Fixed64),
		BudgetsStatus:       budgetsStatus,
		FinalPaymentStatus:  false,
		TrackingCount:       0,
		TerminatedHeight:    0,
		ProposalOwner:       proposal.OwnerPublicKey,
	}
	return proposalState, proposal
}

func (s *txValidatorTestSuite) getCRCCloseProposalTxWithHash(publicKeyStr, privateKeyStr,
	crPublicKeyStr, crPrivateKeyStr string, closeProposalHash common.Uint256) interfaces.Transaction {
	draftData := randomBytes(10)

	privateKey1, _ := common.HexStringToBytes(privateKeyStr)
	publicKey1, _ := common.HexStringToBytes(publicKeyStr)

	privateKey2, _ := common.HexStringToBytes(crPrivateKeyStr)
	//publicKey2, _ := common.HexStringToBytes(crPublicKeyStr)
	code2 := getCodeByPubKeyStr(crPublicKeyStr)
	//did2, _ := getDIDFromCode(code2)

	//draftData := randomBytes(10)
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposal,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	CRCouncilMemberDID, _ := blockchain.GetDIDFromCode(code2)
	crcProposalPayload := &payload.CRCProposal{
		ProposalType:       payload.CloseProposal,
		OwnerPublicKey:     publicKey1,
		CRCouncilMemberDID: *CRCouncilMemberDID,
		DraftHash:          common.Hash(draftData),
		TargetProposalHash: closeProposalHash,
	}

	signBuf := new(bytes.Buffer)
	crcProposalPayload.SerializeUnsigned(signBuf, payload.CRCProposalVersion)
	sig, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crcProposalPayload.Signature = sig

	common.WriteVarBytes(signBuf, sig)
	crcProposalPayload.CRCouncilMemberDID.Serialize(signBuf)
	crSig, _ := crypto.Sign(privateKey2, signBuf.Bytes())
	crcProposalPayload.CRCouncilMemberSignature = crSig

	txn.SetPayload(crcProposalPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func (s *txValidatorTestSuite) getCRCCloseProposalTx(publicKeyStr, privateKeyStr,
	crPublicKeyStr, crPrivateKeyStr string) interfaces.Transaction {

	privateKey1, _ := common.HexStringToBytes(privateKeyStr)

	privateKey2, _ := common.HexStringToBytes(crPrivateKeyStr)
	publicKey2, _ := common.HexStringToBytes(crPublicKeyStr)
	code2 := getCodeByPubKeyStr(crPublicKeyStr)
	//did2, _ := getDIDFromCode(code2)

	draftData := randomBytes(10)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposal,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	CRCouncilMemberDID, _ := blockchain.GetDIDFromCode(code2)
	crcProposalPayload := &payload.CRCProposal{
		ProposalType:       payload.CloseProposal,
		OwnerPublicKey:     publicKey2,
		CRCouncilMemberDID: *CRCouncilMemberDID,
		DraftHash:          common.Hash(draftData),
		TargetProposalHash: common.Hash(randomBytes(10)),
	}

	signBuf := new(bytes.Buffer)
	crcProposalPayload.SerializeUnsigned(signBuf, payload.CRCProposalVersion)
	sig, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crcProposalPayload.Signature = sig

	common.WriteVarBytes(signBuf, sig)
	crcProposalPayload.CRCouncilMemberDID.Serialize(signBuf)
	crSig, _ := crypto.Sign(privateKey2, signBuf.Bytes())
	crcProposalPayload.CRCouncilMemberSignature = crSig

	txn.SetPayload(crcProposalPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func (s *txValidatorTestSuite) getCRCReservedCustomIDProposalTx(publicKeyStr, privateKeyStr,
	crPublicKeyStr, crPrivateKeyStr string) interfaces.Transaction {

	privateKey1, _ := common.HexStringToBytes(privateKeyStr)

	privateKey2, _ := common.HexStringToBytes(crPrivateKeyStr)
	publicKey2, _ := common.HexStringToBytes(crPublicKeyStr)
	code2 := getCodeByPubKeyStr(crPublicKeyStr)
	//did2, _ := getDIDFromCode(code2)

	draftData := randomBytes(10)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposal,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	CRCouncilMemberDID, _ := blockchain.GetDIDFromCode(code2)
	crcProposalPayload := &payload.CRCProposal{
		ProposalType:         payload.ReserveCustomID,
		OwnerPublicKey:       publicKey2,
		CRCouncilMemberDID:   *CRCouncilMemberDID,
		DraftHash:            common.Hash(draftData),
		ReservedCustomIDList: []string{randomName(3), randomName(3), randomName(3)},
	}

	signBuf := new(bytes.Buffer)
	crcProposalPayload.SerializeUnsigned(signBuf, payload.CRCProposalVersion)
	sig, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crcProposalPayload.Signature = sig

	common.WriteVarBytes(signBuf, sig)
	crcProposalPayload.CRCouncilMemberDID.Serialize(signBuf)
	crSig, _ := crypto.Sign(privateKey2, signBuf.Bytes())
	crcProposalPayload.CRCouncilMemberSignature = crSig

	txn.SetPayload(crcProposalPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr),
		Parameter: nil,
	}})
	return txn
}
