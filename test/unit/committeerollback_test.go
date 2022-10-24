// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package unit

import (
	"bytes"
	"fmt"
	"github.com/elastos/Elastos.ELA/core/checkpoint"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"

	"github.com/stretchr/testify/assert"
)

func init() {
	testing.Init()

	functions.GetTransactionByTxType = transaction.GetTransaction
	functions.GetTransactionByBytes = transaction.GetTransactionByBytes
	functions.CreateTransaction = transaction.CreateTransaction
	functions.GetTransactionParameters = transaction.GetTransactionparameters
	config.DefaultParams = *config.GetDefaultParams()
}

func getCIDByPublicKeyStr(publicKey string) *common.Uint168 {
	code1 := getCodeByPubKeyStr(publicKey)
	ct1, _ := contract.CreateCRIDContractByCode(code1)
	return ct1.ToProgramHash()
}

func getDIDByPublicKey(publicKey string) *common.Uint168 {
	code1 := getCodeByPubKeyStr(publicKey)

	ct1, _ := contract.CreateCRIDContractByCode(code1)
	return ct1.ToProgramHash()
}

func getRegisterCRTx(publicKeyStr, privateKeyStr, nickName string) interfaces.Transaction {
	publicKeyStr1 := publicKeyStr
	privateKeyStr1 := privateKeyStr
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)

	code1 := getCodeByPubKeyStr(publicKeyStr1)
	cid1, _ := state.GetCIDByCode(code1)
	did1, _ := state.GetDIDByCode(code1)
	hash1, _ := contract.PublicKeyToDepositProgramHash(publicKey1)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.RegisterCR,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	crInfoPayload := &payload.CRInfo{
		Code:     code1,
		CID:      *cid1,
		DID:      *did1,
		NickName: nickName,
		Url:      "http://www.elastos_test.com",
		Location: 1,
	}
	signBuf := new(bytes.Buffer)
	crInfoPayload.SerializeUnsigned(signBuf, payload.CRInfoVersion)
	rcSig1, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crInfoPayload.Signature = rcSig1
	txn.SetPayload(crInfoPayload)

	txn.SetPrograms([]*program.Program{&program.Program{
		Code:      getCodeByPubKeyStr(publicKeyStr1),
		Parameter: nil,
	}})

	txn.SetOutputs([]*common2.Output{{
		AssetID:     common.Uint256{},
		Value:       5000 * 100000000,
		OutputLock:  0,
		ProgramHash: *hash1,
		Type:        0,
		Payload:     new(outputpayload.DefaultOutput),
	}})
	return txn
}

func getUpdateCR(publicKeyStr string, cid common.Uint168,
	nickname string) interfaces.Transaction {
	code := getCodeByPubKeyStr(publicKeyStr)
	txn := functions.CreateTransaction(
		0,
		common2.UpdateCR,
		0,
		&payload.CRInfo{
			Code:     code,
			CID:      cid,
			NickName: nickname,
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	return txn
}

func getUnregisterCR(cid common.Uint168) interfaces.Transaction {
	txn := functions.CreateTransaction(
		0,
		common2.UnregisterCR,
		0,
		&payload.UnregisterCR{
			CID: cid,
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	return txn
}

func generateReturnDeposite(publicKeyStr string) interfaces.Transaction {
	code := getCodeByPubKeyStr(publicKeyStr)
	txn := functions.CreateTransaction(
		0,
		common2.ReturnCRDepositCoin,
		0,
		&payload.ReturnDepositCoin{},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{Value: 4999 * 100000000},
		},
		0,
		[]*program.Program{
			&program.Program{
				Code: code,
			},
		},
	)
	return txn
}

func getVoteCRTx(amount common.Fixed64,
	candidateVotes []outputpayload.CandidateVotes) interfaces.Transaction {
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.TransferAsset,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				AssetID:     common.Uint256{},
				Value:       amount,
				OutputLock:  0,
				ProgramHash: common.Uint168{123},
				Type:        common2.OTVote,
				Payload: &outputpayload.VoteOutput{
					Version: outputpayload.VoteProducerAndCRVersion,
					Contents: []outputpayload.VoteContent{
						outputpayload.VoteContent{
							VoteType:       outputpayload.CRC,
							CandidateVotes: candidateVotes,
						},
					},
				},
			},
		},
		0,
		[]*program.Program{},
	)
	return txn
}

func getCRCProposalTx(elaAddress string, publicKeyStr, privateKeyStr,
	crPublicKeyStr, crPrivateKeyStr string) interfaces.Transaction {
	publicKey1, _ := common.HexStringToBytes(publicKeyStr)
	privateKey1, _ := common.HexStringToBytes(privateKeyStr)

	privateKey2, _ := common.HexStringToBytes(crPrivateKeyStr)
	code2 := getCodeByPubKeyStr(crPublicKeyStr)

	recipient, _ := common.Uint168FromAddress(elaAddress)

	draftData := randomBytes(10)

	crcProposalPayload := &payload.CRCProposal{
		ProposalType:       payload.Normal,
		OwnerPublicKey:     publicKey1,
		CRCouncilMemberDID: *getDID(code2),
		DraftHash:          common.Hash(draftData),
		Budgets:            createBudgets(3),
		Recipient:          *recipient,
	}

	signBuf := new(bytes.Buffer)
	crcProposalPayload.SerializeUnsigned(signBuf, payload.CRCProposalVersion)
	sig, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crcProposalPayload.Signature = sig

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposal,
		0,
		crcProposalPayload,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	common.WriteVarBytes(signBuf, sig)
	crcProposalPayload.CRCouncilMemberDID.Serialize(signBuf)
	crSig, _ := crypto.Sign(privateKey2, signBuf.Bytes())
	crcProposalPayload.CRCouncilMemberSignature = crSig

	txn.SetPayload(crcProposalPayload)
	txn.SetPrograms([]*program.Program{&program.Program{
		Code:      getCodeByPubKeyStr(publicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func getCRCProposalReviewTx(proposalHash common.Uint256, vote payload.VoteResult,
	crPublicKeyStr, crPrivateKeyStr string) interfaces.Transaction {

	privateKey1, _ := common.HexStringToBytes(crPrivateKeyStr)
	code := getCodeByPubKeyStr(crPublicKeyStr)

	crcProposalReviewPayload := &payload.CRCProposalReview{
		ProposalHash: proposalHash,
		VoteResult:   vote,
		DID:          *getDID(code),
	}

	signBuf := new(bytes.Buffer)
	crcProposalReviewPayload.SerializeUnsigned(signBuf, payload.CRCProposalReviewVersion)
	sig, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crcProposalReviewPayload.Signature = sig

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposalReview,
		0,
		crcProposalReviewPayload,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	txn.SetPayload(crcProposalReviewPayload)
	txn.SetPrograms([]*program.Program{&program.Program{
		Code:      getCodeByPubKeyStr(crPublicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func getCRCProposalTrackingTx(
	trackingType payload.CRCProposalTrackingType,
	proposalHash common.Uint256, stage uint8,
	ownerpublickeyStr, ownerprivatekeyStr,
	newownerpublickeyStr, newownerprivatekeyStr,
	sgPrivateKeyStr string) interfaces.Transaction {

	ownerpublickey, _ := common.HexStringToBytes(ownerpublickeyStr)
	ownerprivatekey, _ := common.HexStringToBytes(ownerprivatekeyStr)

	newownerpublickey, _ := common.HexStringToBytes(newownerpublickeyStr)
	newownerprivatekey, _ := common.HexStringToBytes(newownerprivatekeyStr)

	sgPrivateKey, _ := common.HexStringToBytes(sgPrivateKeyStr)

	documentData := randomBytes(10)
	opinionHash := randomBytes(10)

	cPayload := &payload.CRCProposalTracking{
		ProposalTrackingType:        trackingType,
		ProposalHash:                proposalHash,
		Stage:                       stage,
		MessageHash:                 common.Hash(documentData),
		OwnerPublicKey:              ownerpublickey,
		NewOwnerPublicKey:           newownerpublickey,
		SecretaryGeneralOpinionHash: common.Hash(opinionHash),
	}

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposalTracking,
		0,
		cPayload,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	signBuf := new(bytes.Buffer)
	cPayload.SerializeUnsigned(signBuf, payload.CRCProposalTrackingVersion)
	sig, _ := crypto.Sign(ownerprivatekey, signBuf.Bytes())
	cPayload.OwnerSignature = sig

	if newownerpublickeyStr != "" && newownerprivatekeyStr != "" {
		common.WriteVarBytes(signBuf, sig)
		crSig, _ := crypto.Sign(newownerprivatekey, signBuf.Bytes())
		cPayload.NewOwnerSignature = crSig
		sig = crSig
	}

	common.WriteVarBytes(signBuf, sig)
	crSig, _ := crypto.Sign(sgPrivateKey, signBuf.Bytes())
	cPayload.SecretaryGeneralSignature = crSig

	txn.SetPayload(cPayload)
	return txn
}

func getCRCProposalWithdrawTx(proposalHash common.Uint256,
	OwnerPublicKeyStr, sponsorPrivateKeyStr string, fee common.Fixed64,
	inputs []*common2.Input, outputs []*common2.Output) interfaces.Transaction {

	OwnerPublicKey, _ := common.HexStringToBytes(OwnerPublicKeyStr)
	sponsorPrivateKey, _ := common.HexStringToBytes(sponsorPrivateKeyStr)

	crcProposalWithdraw := &payload.CRCProposalWithdraw{
		ProposalHash:   proposalHash,
		OwnerPublicKey: OwnerPublicKey,
	}

	signBuf := new(bytes.Buffer)
	crcProposalWithdraw.SerializeUnsigned(signBuf, payload.CRCProposalWithdrawDefault)
	signature, _ := crypto.Sign(sponsorPrivateKey, signBuf.Bytes())
	crcProposalWithdraw.Signature = signature

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposalWithdraw,
		0,
		crcProposalWithdraw,
		[]*common2.Attribute{},
		inputs,
		outputs,
		0,
		[]*program.Program{},
	)

	return txn
}

func committeeKeyFrameEqual(first, second *state.CommitteeKeyFrame) bool {
	return keyframeEqual(first.KeyFrame, second.KeyFrame) &&
		stateKeyframeEqual(first.StateKeyFrame, second.StateKeyFrame) &&
		proposalKeyFrameEqual(first.ProposalKeyFrame, second.ProposalKeyFrame)
}

func checkResult(t *testing.T, A, B, C, D *state.CommitteeKeyFrame) {
	assert.Equal(t, true, committeeKeyFrameEqual(A, C))
	assert.Equal(t, false, committeeKeyFrameEqual(A, B))
	assert.Equal(t, true, committeeKeyFrameEqual(B, D))
	assert.Equal(t, false, committeeKeyFrameEqual(B, C))
}

func TestCommittee_RollbackRegisterAndVoteCR(t *testing.T) {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	cid1 := getCIDByPublicKeyStr(publicKeyStr1)
	nickName1 := "nickname 1"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	cid2 := getCIDByPublicKeyStr(publicKeyStr2)
	nickName2 := "nickname 2"

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"
	cid3 := getCIDByPublicKeyStr(publicKeyStr3)
	nickName3 := "nickname 3"

	registerCRTxn1 := getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	registerCRTxn2 := getRegisterCRTx(publicKeyStr2, privateKeyStr2, nickName2)
	registerCRTxn3 := getRegisterCRTx(publicKeyStr3, privateKeyStr3, nickName3)

	CkpManager := checkpoint.NewManager(config.GetDefaultParams())
	// new committee
	committee := state.NewCommittee(&config.DefaultParams,
		CkpManager)

	// avoid getting UTXOs from database
	currentHeight := config.DefaultParams.CRConfiguration.CRVotingStartHeight

	// register cr
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
	}, nil)
	assert.Equal(t, 3, len(committee.GetCandidates(state.Pending)))
	assert.Equal(t, 0, len(committee.GetCandidates(state.Active)))

	// vote cr
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
		}, nil)
	}
	keyFrameA := committee.Snapshot()

	voteCRTx := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{cid1.Bytes(), 3},
		{cid2.Bytes(), 2},
		{cid3.Bytes(), 1}})
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx,
		},
	}, nil)
	assert.Equal(t, common.Fixed64(3), committee.GetCandidate(*cid1).Votes)
	keyFrameB := committee.Snapshot()

	// rollback
	currentHeight--
	err := committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	assert.Equal(t, common.Fixed64(0), committee.GetCandidate(*cid1).Votes)
	keyFrameC := committee.Snapshot()

	// reprocess
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header:       common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{voteCRTx}}, nil)
	assert.Equal(t, common.Fixed64(3), committee.GetCandidate(*cid1).Votes)
	keyFrameD := committee.Snapshot()

	checkResult(t, keyFrameA, keyFrameB, keyFrameC, keyFrameD)
}

func TestCommittee_RollbackEndVotingPeriod(t *testing.T) {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	did1 := getCIDByPublicKeyStr(publicKeyStr1)
	nickName1 := "nickname 1"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	did2 := getCIDByPublicKeyStr(publicKeyStr2)
	nickName2 := "nickname 2"

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"
	did3 := getCIDByPublicKeyStr(publicKeyStr3)
	nickName3 := "nickname 3"

	registerCRTxn1 := getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	registerCRTxn2 := getRegisterCRTx(publicKeyStr2, privateKeyStr2, nickName2)
	registerCRTxn3 := getRegisterCRTx(publicKeyStr3, privateKeyStr3, nickName3)

	// set count of CR member
	cfg := &config.DefaultParams
	cfg.DPoSConfiguration.CRCArbiters = cfg.DPoSConfiguration.CRCArbiters[0:2]
	cfg.CRConfiguration.MemberCount = 2

	CkpManager := checkpoint.NewManager(config.GetDefaultParams())
	// new committee
	committee := state.NewCommittee(cfg, CkpManager)

	// avoid getting UTXOs from database
	currentHeight := config.DefaultParams.CRConfiguration.CRVotingStartHeight

	// register cr
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
	}, nil)
	assert.Equal(t, 3, len(committee.GetCandidates(state.Pending)))
	assert.Equal(t, 0, len(committee.GetCandidates(state.Active)))

	// vote cr
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
		}, nil)
	}

	voteCRTx := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 3},
		{did2.Bytes(), 2},
		{did3.Bytes(), 1}})

	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx,
		},
	}, nil)

	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight - 1
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 0, len(committee.GetCurrentMembers()))
	assert.Equal(t, 3, len(committee.GetAllCandidates()))

	// process
	keyFrameA := committee.Snapshot()
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 2, len(committee.GetCurrentMembers()))
	assert.Equal(t, 0, len(committee.GetAllCandidates()))
	keyFrameB := committee.Snapshot()

	// rollback
	currentHeight--
	err := committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(committee.GetCurrentMembers()))
	assert.Equal(t, 3, len(committee.GetAllCandidates()))
	keyFrameC := committee.Snapshot()

	// reprocess
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 2, len(committee.GetCurrentMembers()))
	assert.Equal(t, 0, len(committee.GetAllCandidates()))
	keyFrameD := committee.Snapshot()

	checkResult(t, keyFrameA, keyFrameB, keyFrameC, keyFrameD)
}

func TestCommittee_RollbackContinueVotingPeriod(t *testing.T) {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	did1 := getCIDByPublicKeyStr(publicKeyStr1)
	nickName1 := "nickname 1"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	did2 := getCIDByPublicKeyStr(publicKeyStr2)
	nickName2 := "nickname 2"

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"
	did3 := getCIDByPublicKeyStr(publicKeyStr3)
	nickName3 := "nickname 3"

	registerCRTxn1 := getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	registerCRTxn2 := getRegisterCRTx(publicKeyStr2, privateKeyStr2, nickName2)
	registerCRTxn3 := getRegisterCRTx(publicKeyStr3, privateKeyStr3, nickName3)

	// set count of CR member
	params := config.GetDefaultParams()
	cfg := params
	cfg.DPoSConfiguration.CRCArbiters = cfg.DPoSConfiguration.CRCArbiters[0:4]
	cfg.CRConfiguration.MemberCount = 4

	CkpManager := checkpoint.NewManager(config.GetDefaultParams())
	// new committee
	committee := state.NewCommittee(cfg, CkpManager)

	// avoid getting UTXOs from database
	currentHeight := config.DefaultParams.CRConfiguration.CRVotingStartHeight

	// register cr
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
	}, nil)
	assert.Equal(t, 3, len(committee.GetCandidates(state.Pending)))
	assert.Equal(t, 0, len(committee.GetCandidates(state.Active)))

	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
	}, nil)

	// vote cr
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
		}, nil)
	}

	voteCRTx := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 3},
		{did2.Bytes(), 2},
		{did3.Bytes(), 1}})

	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx,
		},
	}, nil)

	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight - 1
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	keyFrameA := committee.Snapshot()

	// process
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	keyFrameB := committee.Snapshot()

	// rollback
	currentHeight--
	err := committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	keyFrameC := committee.Snapshot()

	// reprocess
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	keyFrameD := committee.Snapshot()

	publicKeyStr4 := "027209c3a6bcb95e9ef766c81136bcd6f2338eee7f9caebf694825e411320bab12"
	privateKeyStr4 := "b3b1c16abd786c4994af9ee8c79d25457f66509731f74d6a9a9673ca872fa8fa"
	did4 := getCIDByPublicKeyStr(publicKeyStr4)
	nickName4 := "nickname 4"
	registerCRTxn4 := getRegisterCRTx(publicKeyStr4, privateKeyStr4, nickName4)

	// register
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn4,
		},
	}, nil)

	// vote cr
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
		}, nil)
	}
	voteCRTx2 := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did4.Bytes(), 4}})

	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx2,
		},
	}, nil)

	// set current Height to one block before ending voting period
	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight - 1 + cfg.CRConfiguration.VotingPeriod
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	keyFrameA2 := committee.Snapshot()

	// process
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 4, len(committee.GetCurrentMembers()))
	assert.Equal(t, 0, len(committee.GetAllCandidates()))
	keyFrameB2 := committee.Snapshot()

	// rollback
	currentHeight--
	err = committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	keyFrameC2 := committee.Snapshot()

	// reprocess
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 4, len(committee.GetCurrentMembers()))
	assert.Equal(t, 0, len(committee.GetAllCandidates()))
	keyFrameD2 := committee.Snapshot()

	checkResult(t, keyFrameA, keyFrameB, keyFrameC, keyFrameD)
	checkResult(t, keyFrameA2, keyFrameB2, keyFrameC2, keyFrameD2)
}

func TestCommittee_RollbackChangeCommittee(t *testing.T) {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	did1 := getCIDByPublicKeyStr(publicKeyStr1)
	nickName1 := "nickname 1"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	did2 := getCIDByPublicKeyStr(publicKeyStr2)
	nickName2 := "nickname 2"

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"
	did3 := getCIDByPublicKeyStr(publicKeyStr3)
	nickName3 := "nickname 3"

	registerCRTxn1 := getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	registerCRTxn2 := getRegisterCRTx(publicKeyStr2, privateKeyStr2, nickName2)
	registerCRTxn3 := getRegisterCRTx(publicKeyStr3, privateKeyStr3, nickName3)

	CkpManager := checkpoint.NewManager(config.GetDefaultParams())
	// new committee
	committee := state.NewCommittee(&config.DefaultParams, CkpManager)

	// set count of CR member
	cfg := &config.DefaultParams
	cfg.DPoSConfiguration.CRCArbiters = cfg.DPoSConfiguration.CRCArbiters[0:2]
	cfg.CRConfiguration.MemberCount = 2

	// avoid getting UTXOs from database
	currentHeight := cfg.CRConfiguration.CRVotingStartHeight

	// register cr   every cr DepositAmount is 5000
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
	}, nil)

	// vote cr
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
		}, nil)
	}

	voteCRTx := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 3},
		{did2.Bytes(), 2},
		{did3.Bytes(), 1}})
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx,
		},
	}, nil)
	assert.Equal(t, common.Fixed64(3), committee.GetCandidate(*did1).Votes)

	// end first voting period into election
	//did1 did2 is cr did3 is candidate
	//did1 did2 DepositAmount 5000 ela
	//did3 is candidate DepositAmount 0 ela
	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 2, len(committee.GetCurrentMembers()))

	// register cr again  did1 did2 10000  did3 5000
	currentHeight = config.DefaultParams.CRConfiguration.CRCommitteeStartHeight +
		cfg.CRConfiguration.DutyPeriod - cfg.CRConfiguration.VotingPeriod
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
	}, nil)

	// vote cr again
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
		}, nil)
	}

	voteCRTx2 := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 1},
		{did2.Bytes(), 2},
		{did3.Bytes(), 3}})
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx2,
		},
	}, nil)
	assert.Equal(t, common.Fixed64(1), committee.GetCandidate(*did1).Votes)
	keyFrameA := committee.Snapshot()

	// end second voting period
	//change commitee old cr  did1 did2 -5000  DepositAmount 5000
	//did1 is candidate -5000 DepositAmount 0
	//did3 5000
	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight + cfg.CRConfiguration.DutyPeriod
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 2, len(committee.GetCurrentMembers()))
	keyFrameB := committee.Snapshot()

	// rollback
	currentHeight--
	err := committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	assert.Equal(t, common.Fixed64(1), committee.GetCandidate(*did1).Votes)
	keyFrameC := committee.Snapshot()

	// reprocess
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 2, len(committee.GetCurrentMembers()))
	keyFrameD := committee.Snapshot()

	checkResult(t, keyFrameA, keyFrameB, keyFrameC, keyFrameD)
}

func TestCommittee_RollbackCRCProposal(t *testing.T) {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	did1 := getCIDByPublicKeyStr(publicKeyStr1)
	nickName1 := "nickname 1"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	did2 := getCIDByPublicKeyStr(publicKeyStr2)
	nickName2 := "nickname 2"

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"
	did3 := getCIDByPublicKeyStr(publicKeyStr3)
	nickName3 := "nickname 3"

	registerCRTxn1 := getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	registerCRTxn2 := getRegisterCRTx(publicKeyStr2, privateKeyStr2, nickName2)
	registerCRTxn3 := getRegisterCRTx(publicKeyStr3, privateKeyStr3, nickName3)

	CkpManager := checkpoint.NewManager(config.GetDefaultParams())
	// new committee
	committee := state.NewCommittee(&config.DefaultParams, CkpManager)

	// set count of CR member
	cfg := &config.DefaultParams
	cfg.DPoSConfiguration.CRCArbiters = cfg.DPoSConfiguration.CRCArbiters[0:2]
	cfg.CRConfiguration.MemberCount = 2
	cfg.CRConfiguration.CRAgreementCount = 2
	cfg.CRConfiguration.CRClaimDPOSNodePeriod = 1000000

	// avoid getting UTXOs from database
	currentHeight := cfg.CRConfiguration.CRVotingStartHeight

	// register cr
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
	}, nil)

	// vote cr
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
		}, nil)
	}

	voteCRTx := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 3},
		{did2.Bytes(), 2},
		{did3.Bytes(), 1}})
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx,
		},
	}, nil)
	assert.Equal(t, common.Fixed64(3), committee.GetCandidate(*did1).Votes)

	// end first voting period
	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 2, len(committee.GetCurrentMembers()))
	keyFrameA := committee.Snapshot()

	// create CRC proposal tx
	elaAddress := "EZaqDYAPFsjynGpvHwbuiiiL4dEiHtX4gD"
	proposalTx := getCRCProposalTx(elaAddress, publicKeyStr1, privateKeyStr1,
		publicKeyStr2, privateKeyStr2)
	proposalHash := proposalTx.Payload().(*payload.CRCProposal).Hash(payload.CRCProposalVersion01)
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalTx,
		}}, nil)
	assert.Equal(t, 1, len(committee.GetProposals(state.Registered)))
	//assert.Equal(t, 2, committee.GetProposal(proposalTx.Payload.(*payload.CRCProposal).Hash()))
	keyFrameB := committee.Snapshot()

	// rollback
	currentHeight--
	err := committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(committee.GetProposals(state.Registered)))
	keyFrameC := committee.Snapshot()

	// reprocess
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalTx,
		}}, nil)
	assert.Equal(t, 1, len(committee.GetProposals(state.Registered)))
	keyFrameD := committee.Snapshot()

	checkResult(t, keyFrameA, keyFrameB, keyFrameC, keyFrameD)

	// set CR agreement count
	committee.Params.CRConfiguration.CRAgreementCount = 2

	// review proposal
	proposalReviewTx1 := getCRCProposalReviewTx(proposalHash, payload.Approve,
		publicKeyStr1, privateKeyStr1)
	proposalReviewTx2 := getCRCProposalReviewTx(proposalHash, payload.Approve,
		publicKeyStr2, privateKeyStr2)
	keyFrameA2 := committee.Snapshot()

	// process
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalReviewTx1,
			proposalReviewTx2,
		}}, nil)
	keyFrameB2 := committee.Snapshot()
	assert.Equal(t, state.Registered, committee.GetProposal(proposalHash).Status)

	// rollback
	currentHeight--
	err = committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	keyFrameC2 := committee.Snapshot()

	// reprocess
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalReviewTx1,
			proposalReviewTx2,
		}}, nil)
	keyFrameD2 := committee.Snapshot()
	assert.Equal(t, state.Registered, committee.GetProposal(proposalHash).Status)

	checkResult(t, keyFrameA2, keyFrameB2, keyFrameC2, keyFrameD2)

	// change to CRAgreed
	keyFrameA3 := committee.Snapshot()
	currentHeight += cfg.CRConfiguration.ProposalCRVotingPeriod
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, state.CRAgreed, committee.GetProposal(proposalHash).Status)
	keyFrameB3 := committee.Snapshot()

	// rollback
	currentHeight--
	err = committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	assert.Equal(t, state.Registered, committee.GetProposal(proposalHash).Status)
	keyFrameC3 := committee.Snapshot()

	// reprocess
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, state.CRAgreed, committee.GetProposal(proposalHash).Status)
	keyFrameD3 := committee.Snapshot()

	checkResult(t, keyFrameA3, keyFrameB3, keyFrameC3, keyFrameD3)

	// change to VoterAgreed
	keyFrameA4 := committee.Snapshot()
	currentHeight += cfg.CRConfiguration.ProposalPublicVotingPeriod
	currentHeight += cfg.CRConfiguration.ProposalCRVotingPeriod
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, state.VoterAgreed, committee.GetProposal(proposalHash).Status)
	keyFrameB4 := committee.Snapshot()

	// rollback
	currentHeight--
	err = committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	assert.Equal(t, state.CRAgreed, committee.GetProposal(proposalHash).Status)
	keyFrameC4 := committee.Snapshot()

	// reprocess
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, state.VoterAgreed, committee.GetProposal(proposalHash).Status)
	keyFrameD4 := committee.Snapshot()

	checkResult(t, keyFrameA4, keyFrameB4, keyFrameC4, keyFrameD4)
}

func TestCommittee_RollbackCRCProposalTracking(t *testing.T) {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	did1 := getCIDByPublicKeyStr(publicKeyStr1)
	nickName1 := "nickname 1"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	did2 := getCIDByPublicKeyStr(publicKeyStr2)
	nickName2 := "nickname 2"

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"
	did3 := getCIDByPublicKeyStr(publicKeyStr3)
	nickName3 := "nickname 3"

	registerCRTxn1 := getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	registerCRTxn2 := getRegisterCRTx(publicKeyStr2, privateKeyStr2, nickName2)
	registerCRTxn3 := getRegisterCRTx(publicKeyStr3, privateKeyStr3, nickName3)

	CkpManager := checkpoint.NewManager(config.GetDefaultParams())
	// new committee
	committee := state.NewCommittee(&config.DefaultParams, CkpManager)

	// set count of CR member
	cfg := &config.DefaultParams
	cfg.DPoSConfiguration.CRCArbiters = cfg.DPoSConfiguration.CRCArbiters[0:2]
	cfg.CRConfiguration.MemberCount = 2

	// avoid getting UTXOs from database
	currentHeight := cfg.CRConfiguration.CRVotingStartHeight

	// register cr
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
	}, nil)

	// vote cr
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
		}, nil)
	}

	voteCRTx := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 3},
		{did2.Bytes(), 2},
		{did3.Bytes(), 1}})
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx,
		},
	}, nil)
	assert.Equal(t, common.Fixed64(3), committee.GetCandidate(*did1).Votes)

	// end first voting period
	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 2, len(committee.GetCurrentMembers()))

	// create CRC proposal tx
	elaAddress := "EZaqDYAPFsjynGpvHwbuiiiL4dEiHtX4gD"
	proposalTx := getCRCProposalTx(elaAddress, publicKeyStr1, privateKeyStr1,
		publicKeyStr2, privateKeyStr2)
	proposalHash := proposalTx.Payload().(*payload.CRCProposal).Hash(payload.CRCProposalVersion01)
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalTx,
		}}, nil)
	assert.Equal(t, 1, len(committee.GetProposals(state.Registered)))
	//assert.Equal(t, 2, committee.GetProposal(proposalTx.Payload.(*payload.CRCProposal).Hash()))

	// set CR agreement count
	committee.Params.CRConfiguration.CRAgreementCount = 2

	// review proposal
	proposalReviewTx1 := getCRCProposalReviewTx(proposalHash, payload.Approve,
		publicKeyStr1, privateKeyStr1)
	proposalReviewTx2 := getCRCProposalReviewTx(proposalHash, payload.Approve,
		publicKeyStr2, privateKeyStr2)

	// process
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalReviewTx1,
			proposalReviewTx2,
		}}, nil)
	assert.Equal(t, state.Registered, committee.GetProposal(proposalHash).Status)

	// change to CRAgreed
	currentHeight += cfg.CRConfiguration.ProposalCRVotingPeriod
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, state.CRAgreed, committee.GetProposal(proposalHash).Status)

	// change to VoterAgreed
	currentHeight += cfg.CRConfiguration.ProposalPublicVotingPeriod
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, state.VoterAgreed, committee.GetProposal(proposalHash).Status)

	// set secretary-general
	publicKeyStr4 := "027209c3a6bcb95e9ef766c81136bcd6f2338eee7f9caebf694825e411320bab12"
	privateKeyStr4 := "b3b1c16abd786c4994af9ee8c79d25457f66509731f74d6a9a9673ca872fa8fa"
	committee.Params.CRConfiguration.SecretaryGeneral = publicKeyStr4
	committee.GetHeight = func() uint32 {
		return currentHeight
	}

	// proposal tracking of type progress
	proposalTrackingTx := getCRCProposalTrackingTx(
		payload.Progress, proposalHash, 1, publicKeyStr1, privateKeyStr1,
		"", "", privateKeyStr4)
	keyFrameA := committee.Snapshot()

	// process
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalTrackingTx,
		}}, nil)
	assert.Equal(t, 2, len(committee.GetProposal(proposalHash).WithdrawableBudgets))
	keyFrameB := committee.Snapshot()

	// rollback
	currentHeight--
	err := committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(committee.GetProposal(proposalHash).WithdrawableBudgets))
	keyFrameC := committee.Snapshot()

	// reprocess
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalTrackingTx,
		}}, nil)
	assert.Equal(t, 2, len(committee.GetProposal(proposalHash).WithdrawableBudgets))
	keyFrameD := committee.Snapshot()

	checkResult(t, keyFrameA, keyFrameB, keyFrameC, keyFrameD)

	// proposal tracking of type finalized
	proposalTrackingTx2 := getCRCProposalTrackingTx(
		payload.Finalized, proposalHash, 0, publicKeyStr1, privateKeyStr1,
		"", "", privateKeyStr4)
	keyFrameA2 := committee.Snapshot()

	// process
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalTrackingTx2,
		}}, nil)
	assert.Equal(t, 3, len(committee.GetProposal(proposalHash).WithdrawableBudgets))
	keyFrameB2 := committee.Snapshot()

	// rollback
	currentHeight--
	err = committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(committee.GetProposal(proposalHash).WithdrawableBudgets))
	keyFrameC2 := committee.Snapshot()

	// reprocess
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalTrackingTx2,
		}}, nil)
	assert.Equal(t, 3, len(committee.GetProposal(proposalHash).WithdrawableBudgets))
	keyFrameD2 := committee.Snapshot()

	checkResult(t, keyFrameA2, keyFrameB2, keyFrameC2, keyFrameD2)
}

func TestCommittee_RollbackCRCProposalWithdraw(t *testing.T) {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	did1 := getCIDByPublicKeyStr(publicKeyStr1)
	nickName1 := "nickname 1"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	did2 := getCIDByPublicKeyStr(publicKeyStr2)
	nickName2 := "nickname 2"

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"
	did3 := getCIDByPublicKeyStr(publicKeyStr3)
	nickName3 := "nickname 3"

	registerCRTxn1 := getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	registerCRTxn2 := getRegisterCRTx(publicKeyStr2, privateKeyStr2, nickName2)
	registerCRTxn3 := getRegisterCRTx(publicKeyStr3, privateKeyStr3, nickName3)

	CkpManager := checkpoint.NewManager(config.GetDefaultParams())
	// new committee
	committee := state.NewCommittee(&config.DefaultParams, CkpManager)

	// set count of CR member
	cfg := &config.DefaultParams
	cfg.DPoSConfiguration.CRCArbiters = cfg.DPoSConfiguration.CRCArbiters[0:2]
	cfg.CRConfiguration.MemberCount = 2

	// avoid getting UTXOs from database
	currentHeight := cfg.CRConfiguration.CRVotingStartHeight

	// register cr
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
	}, nil)

	// vote cr
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
		}, nil)
	}

	voteCRTx := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 3},
		{did2.Bytes(), 2},
		{did3.Bytes(), 1}})
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx,
		},
	}, nil)
	assert.Equal(t, common.Fixed64(3), committee.GetCandidate(*did1).Votes)

	// end first voting period
	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 2, len(committee.GetCurrentMembers()))

	// create CRC proposal tx
	elaAddress := "EZaqDYAPFsjynGpvHwbuiiiL4dEiHtX4gD"
	proposalTx := getCRCProposalTx(elaAddress, publicKeyStr1, privateKeyStr1,
		publicKeyStr2, privateKeyStr2)
	proposalHash := proposalTx.Payload().(*payload.CRCProposal).Hash(payload.CRCProposalVersion01)
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalTx,
		}}, nil)
	assert.Equal(t, 1, len(committee.GetProposals(state.Registered)))

	// set CR agreement count
	committee.Params.CRConfiguration.CRAgreementCount = 2

	// review proposal
	proposalReviewTx1 := getCRCProposalReviewTx(proposalHash, payload.Approve,
		publicKeyStr1, privateKeyStr1)
	proposalReviewTx2 := getCRCProposalReviewTx(proposalHash, payload.Approve,
		publicKeyStr2, privateKeyStr2)

	// process
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalReviewTx1,
			proposalReviewTx2,
		}}, nil)
	assert.Equal(t, state.Registered, committee.GetProposal(proposalHash).Status)

	// change to CRAgreed
	currentHeight += cfg.CRConfiguration.ProposalCRVotingPeriod
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, state.CRAgreed, committee.GetProposal(proposalHash).Status)

	// change to VoterAgreed
	currentHeight += cfg.CRConfiguration.ProposalPublicVotingPeriod
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, state.VoterAgreed, committee.GetProposal(proposalHash).Status)

	// proposal withdraw
	withdrawTx := getCRCProposalWithdrawTx(proposalHash, publicKeyStr1,
		privateKeyStr1, 1, []*common2.Input{}, []*common2.Output{})
	keyFrameA := committee.Snapshot()

	// process
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			withdrawTx,
		}}, nil)
	assert.Equal(t, 1, len(committee.GetProposal(proposalHash).WithdrawnBudgets))
	keyFrameB := committee.Snapshot()

	// rollback
	currentHeight--
	err := committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(committee.GetProposal(proposalHash).WithdrawnBudgets))
	keyFrameC := committee.Snapshot()

	// reprocess
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			withdrawTx,
		}}, nil)
	assert.Equal(t, 1, len(committee.GetProposal(proposalHash).WithdrawnBudgets))
	keyFrameD := committee.Snapshot()

	checkResult(t, keyFrameA, keyFrameB, keyFrameC, keyFrameD)

	// set secretary-general
	publicKeyStr4 := "027209c3a6bcb95e9ef766c81136bcd6f2338eee7f9caebf694825e411320bab12"
	privateKeyStr4 := "b3b1c16abd786c4994af9ee8c79d25457f66509731f74d6a9a9673ca872fa8fa"
	committee.Params.CRConfiguration.SecretaryGeneral = publicKeyStr4
	committee.GetHeight = func() uint32 {
		return currentHeight
	}

	// proposal tracking of type progress
	proposalTrackingTx := getCRCProposalTrackingTx(
		payload.Progress, proposalHash, 1, publicKeyStr1, privateKeyStr1,
		"", "", privateKeyStr4)

	// process
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalTrackingTx,
		}}, nil)
	assert.Equal(t, 2, len(committee.GetProposal(proposalHash).WithdrawableBudgets))

	// proposal tracking of type finalized
	proposalTrackingTx2 := getCRCProposalTrackingTx(
		payload.Finalized, proposalHash, 0, publicKeyStr1, privateKeyStr1,
		"", "", privateKeyStr4)

	// process
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalTrackingTx2,
		}}, nil)
	assert.Equal(t, 3, len(committee.GetProposal(proposalHash).WithdrawableBudgets))

	// proposal withdraw
	withdrawTx2 := getCRCProposalWithdrawTx(proposalHash, publicKeyStr1,
		privateKeyStr1, 1, []*common2.Input{}, []*common2.Output{})
	keyFrameA2 := committee.Snapshot()

	// process
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			withdrawTx2,
		}}, nil)
	assert.Equal(t, 3, len(committee.GetProposal(proposalHash).WithdrawnBudgets))
	keyFrameB2 := committee.Snapshot()

	// rollback
	currentHeight--
	err = committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(committee.GetProposal(proposalHash).WithdrawnBudgets))
	keyFrameC2 := committee.Snapshot()

	// reprocess
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			withdrawTx2,
		}}, nil)
	assert.Equal(t, 3, len(committee.GetProposal(proposalHash).WithdrawnBudgets))
	keyFrameD2 := committee.Snapshot()

	checkResult(t, keyFrameA2, keyFrameB2, keyFrameC2, keyFrameD2)
}

func TestCommittee_RollbackTempStartVotingPeriod(t *testing.T) {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	did1 := getCIDByPublicKeyStr(publicKeyStr1)
	nickName1 := "nickname 1"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	did2 := getCIDByPublicKeyStr(publicKeyStr2)
	nickName2 := "nickname 2"

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"
	did3 := getCIDByPublicKeyStr(publicKeyStr3)
	nickName3 := "nickname 3"

	registerCRTxn1 := getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	registerCRTxn2 := getRegisterCRTx(publicKeyStr2, privateKeyStr2, nickName2)
	registerCRTxn3 := getRegisterCRTx(publicKeyStr3, privateKeyStr3, nickName3)

	CkpManager := checkpoint.NewManager(config.GetDefaultParams())
	// new committee
	committee := state.NewCommittee(&config.DefaultParams, CkpManager)

	// set count of CR member
	cfg := &config.DefaultParams
	cfg.DPoSConfiguration.CRCArbiters = cfg.DPoSConfiguration.CRCArbiters[0:2]
	cfg.CRConfiguration.MemberCount = 2
	cfg.CRConfiguration.CRAgreementCount = 2

	// avoid getting UTXOs from database
	currentHeight := cfg.CRConfiguration.CRVotingStartHeight

	// register cr
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
	}, nil)

	// vote cr
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
		}, nil)
	}

	voteCRTx := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 3},
		{did2.Bytes(), 2},
		{did3.Bytes(), 1}})
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx,
		},
	}, nil)
	assert.Equal(t, common.Fixed64(3), committee.GetCandidate(*did1).Votes)

	// end first voting period
	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 2, len(committee.GetCurrentMembers()))

	currentHeight = config.DefaultParams.CRConfiguration.CRCommitteeStartHeight +
		cfg.CRConfiguration.DutyPeriod - cfg.CRConfiguration.VotingPeriod - 1
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)

	// register cr again
	currentHeight = config.DefaultParams.CRConfiguration.CRCommitteeStartHeight +
		cfg.CRConfiguration.DutyPeriod - cfg.CRConfiguration.VotingPeriod
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
		},
	}, nil)
	assert.Equal(t, true, committee.IsInElectionPeriod())

	// vote cr again
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
		}, nil)
	}
	assert.Equal(t, true, committee.IsInElectionPeriod())

	voteCRTx2 := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 1}})
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx2,
		},
	}, nil)
	assert.Equal(t, common.Fixed64(1), committee.GetCandidate(*did1).Votes)
	keyFrameA := committee.Snapshot()

	// end second voting period
	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight + cfg.CRConfiguration.DutyPeriod
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, false, committee.IsInElectionPeriod())
	assert.Equal(t, 1, len(committee.GetCandidates(state.Active)))
	keyFrameB := committee.Snapshot()

	// rollback
	currentHeight--
	err := committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	assert.Equal(t, true, committee.IsInElectionPeriod())
	keyFrameC := committee.Snapshot()

	// reprocess
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, false, committee.IsInElectionPeriod())
	assert.Equal(t, 1, len(committee.GetCandidates(state.Active)))
	keyFrameD := committee.Snapshot()

	checkResult(t, keyFrameA, keyFrameB, keyFrameC, keyFrameD)
}

func TestCommittee_RollbackCRCAppropriationTx(t *testing.T) {

	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	did1 := getCIDByPublicKeyStr(publicKeyStr1)
	nickName1 := "nickname 1"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	did2 := getCIDByPublicKeyStr(publicKeyStr2)
	nickName2 := "nickname 2"

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"
	did3 := getCIDByPublicKeyStr(publicKeyStr3)
	nickName3 := "nickname 3"

	registerCRTxn1 := getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	registerCRTxn2 := getRegisterCRTx(publicKeyStr2, privateKeyStr2, nickName2)
	registerCRTxn3 := getRegisterCRTx(publicKeyStr3, privateKeyStr3, nickName3)

	CRCFoundationAddr := "ERyUmNH51roR9qfru37Kqkaok2NghR7L5U"
	crcFoundationUint168, _ := common.Uint168FromAddress(CRCFoundationAddr)
	CRCFoundationPbkey := "02ca89a5fe6213da1b51046733529a84f0265abac59005f6c16f62330d20f02aeb"
	txFoundation := getTransferAssetTx(CRCFoundationPbkey, 5000.0, *crcFoundationUint168)

	// set count of CR member
	cfg := &config.DefaultParams
	cfg.DPoSConfiguration.CRCArbiters = cfg.DPoSConfiguration.CRCArbiters[0:2]
	cfg.CRConfiguration.MemberCount = 2

	CkpManager := checkpoint.NewManager(config.GetDefaultParams())
	// new committee
	committee := state.NewCommittee(cfg, CkpManager)

	// avoid getting UTXOs from database
	currentHeight := config.DefaultParams.CRConfiguration.CRVotingStartHeight
	committee.RegisterFuncitons(&state.CommitteeFuncsConfig{
		GetHeight: func() uint32 {
			return currentHeight
		},
	})

	// register cr
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
			txFoundation,
		},
	}, nil)
	assert.Equal(t, 3, len(committee.GetCandidates(state.Pending)))
	assert.Equal(t, 0, len(committee.GetCandidates(state.Active)))

	// vote cr
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
		}, nil)
	}
	voteCRTx := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 3},
		{did2.Bytes(), 2},
		{did3.Bytes(), 1}})
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx,
		},
	}, nil)

	//currentHeight to before CRCommitteeStartHeight
	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight - 1
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 0, len(committee.GetCurrentMembers()))
	assert.Equal(t, 3, len(committee.GetAllCandidates()))

	// process here change committee
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
	}, nil)
	keyFrameA := committee.Snapshot()

	//process appropriation tx
	crcCommiteeAddressStr := "ESq12oQrvGqHfTkEDYJyR9MxZj1NMnonjo"
	crcCommiteeAddrHash, _ := common.Uint168FromAddress(crcCommiteeAddressStr)
	txAppropriate := getAppropriationTx(500.0, *crcCommiteeAddrHash)
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			txAppropriate,
		},
	}, nil)
	keyFrameB := committee.Snapshot()

	// rollback appropriation tx
	currentHeight--
	err := committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	keyFrameC := committee.Snapshot()

	// reprocess txAppropriate
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			txAppropriate,
		},
	}, nil)
	assert.Equal(t, 2, len(committee.GetCurrentMembers()))
	assert.Equal(t, 0, len(committee.GetAllCandidates()))
	keyFrameD := committee.Snapshot()

	assert.Equal(t, true, committeeKeyFrameEqual(keyFrameA, keyFrameC))
	assert.Equal(t, false, committeeKeyFrameEqual(keyFrameA, keyFrameB))
	assert.Equal(t, true, committeeKeyFrameEqual(keyFrameB, keyFrameD))
	assert.Equal(t, false, committeeKeyFrameEqual(keyFrameB, keyFrameC))
}

func getProgramHash(publicKeyHexStr string) (*common.Uint168, error) {
	pkBytes, _ := common.HexStringToBytes(publicKeyHexStr)
	return contract.PublicKeyToStandardProgramHash(pkBytes)
}

func getAddress(publicKeyHexStr string) (string, error) {
	address1Uint168, _ := getProgramHash(publicKeyHexStr)
	return address1Uint168.ToAddress()
}

func getTransferAssetTx(publicKeyStr string, value common.Fixed64, outPutAddr common.Uint168) interfaces.Transaction {
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	txn.SetPrograms([]*program.Program{&program.Program{
		Code:      getCodeByPubKeyStr(publicKeyStr),
		Parameter: nil,
	}})

	txn.SetOutputs([]*common2.Output{&common2.Output{
		AssetID:     common.Uint256{},
		Value:       value,
		OutputLock:  0,
		ProgramHash: outPutAddr,
		Type:        0,
		Payload:     new(outputpayload.DefaultOutput),
	}})
	return txn
}
func getAppropriationTx(value common.Fixed64, outPutAddr common.Uint168) interfaces.Transaction {
	crcAppropriationPayload := &payload.CRCAppropriation{}

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCAppropriation,
		0,
		crcAppropriationPayload,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	txn.SetOutputs([]*common2.Output{{
		AssetID:     common.Uint256{},
		Value:       value,
		OutputLock:  0,
		ProgramHash: outPutAddr,
		Type:        0,
		Payload:     new(outputpayload.DefaultOutput),
	}})
	return txn
}

func getDIDStrByPublicKey(publicKey string) (string, error) {
	code1 := getCodeByPubKeyStr(publicKey)
	ct1, _ := contract.CreateCRIDContractByCode(code1)
	return ct1.ToProgramHash().ToAddress()
}

func TestCommittee_RollbackCRCImpeachmentTx(t *testing.T) {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	did1 := getCIDByPublicKeyStr(publicKeyStr1)
	address1Uint168, _ := getProgramHash(publicKeyStr1)
	did1Str, _ := getDIDStrByPublicKey(publicKeyStr1)
	fmt.Println("did1", did1Str)

	nickName1 := "nickname 1"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	did2 := getCIDByPublicKeyStr(publicKeyStr2)
	nickName2 := "nickname 2"
	did2Str, _ := getDIDStrByPublicKey(publicKeyStr2)
	fmt.Println("did2", did2Str)

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"
	did3 := getCIDByPublicKeyStr(publicKeyStr3)
	nickName3 := "nickname 3"
	did3Str, _ := getDIDStrByPublicKey(publicKeyStr3)
	fmt.Println("did3", did3Str)

	registerCRTxn1 := getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	registerCRTxn2 := getRegisterCRTx(publicKeyStr2, privateKeyStr2, nickName2)
	registerCRTxn3 := getRegisterCRTx(publicKeyStr3, privateKeyStr3, nickName3)

	// set count of CR member
	cfg := &config.DefaultParams
	cfg.DPoSConfiguration.CRCArbiters = cfg.DPoSConfiguration.CRCArbiters[0:2]
	cfg.CRConfiguration.MemberCount = 2

	CkpManager := checkpoint.NewManager(config.GetDefaultParams())
	// new committee
	committee := state.NewCommittee(cfg, CkpManager)

	// avoid getting UTXOs from database
	currentHeight := config.DefaultParams.CRConfiguration.CRVotingStartHeight

	// register cr
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
	}, nil)
	assert.Equal(t, 3, len(committee.GetCandidates(state.Pending)))
	assert.Equal(t, 0, len(committee.GetCandidates(state.Active)))

	// vote cr
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
		}, nil)
	}
	voteCRTx := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 3},
		{did2.Bytes(), 2},
		{did3.Bytes(), 1}})
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx,
		},
	}, nil)

	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight - 1
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 0, len(committee.GetCurrentMembers()))
	assert.Equal(t, 3, len(committee.GetAllCandidates()))
	// process

	//here change committee
	committee.Params.CRConfiguration.CRAgreementCount = 2
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 2, len(committee.GetCurrentMembers()))
	assert.Equal(t, 0, len(committee.GetAllCandidates()))
	committee.GetState().DepositInfo = make(map[common.Uint168]*state.DepositInfo)
	committee.GetState().DepositInfo[*did1] = &state.DepositInfo{
		DepositAmount: 5000 * 1e8,
		TotalAmount:   5000 * 1e8,
		Penalty:       12,
	}
	keyFrameA := committee.Snapshot()

	//here process impeachment
	//generate impeachment tx
	impeachmentTx := getCRCImpeachmentTx(publicKeyStr1, did1, 3, *address1Uint168)

	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			impeachmentTx,
		},
	}, nil)
	keyFrameB := committee.Snapshot()

	// rollback
	currentHeight--
	err := committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	keyFrameC := committee.Snapshot()

	// reprocess
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			impeachmentTx,
		},
	}, nil)
	keyFrameD := committee.Snapshot()

	assert.Equal(t, true, committeeKeyFrameEqual(keyFrameA, keyFrameC))
	assert.Equal(t, false, committeeKeyFrameEqual(keyFrameA, keyFrameB))
	assert.Equal(t, true, committeeKeyFrameEqual(keyFrameB, keyFrameD))
	assert.Equal(t, false, committeeKeyFrameEqual(keyFrameB, keyFrameC))

}

func TestCommittee_RollbackCRCImpeachmentAndReelectionTx(t *testing.T) {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	did1 := getCIDByPublicKeyStr(publicKeyStr1)
	address1Uint168, _ := getProgramHash(publicKeyStr1)
	did1Str, _ := getDIDStrByPublicKey(publicKeyStr1)
	fmt.Println("did1", did1Str)

	nickName1 := "nickname 1"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	did2 := getCIDByPublicKeyStr(publicKeyStr2)
	nickName2 := "nickname 2"
	did2Str, _ := getDIDStrByPublicKey(publicKeyStr2)
	fmt.Println("did2", did2Str)

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"
	did3 := getCIDByPublicKeyStr(publicKeyStr3)
	nickName3 := "nickname 3"
	did3Str, _ := getDIDStrByPublicKey(publicKeyStr3)
	fmt.Println("did3", did3Str)

	registerCRTxn1 := getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	registerCRTxn2 := getRegisterCRTx(publicKeyStr2, privateKeyStr2, nickName2)
	registerCRTxn3 := getRegisterCRTx(publicKeyStr3, privateKeyStr3, nickName3)

	// set count of CR member
	cfg := &config.DefaultParams
	cfg.DPoSConfiguration.CRCArbiters = cfg.DPoSConfiguration.CRCArbiters[0:2]
	cfg.CRConfiguration.MemberCount = 2

	CkpManager := checkpoint.NewManager(config.GetDefaultParams())
	// new committee
	committee := state.NewCommittee(cfg, CkpManager)

	// avoid getting UTXOs from database
	currentHeight := config.DefaultParams.CRConfiguration.CRVotingStartHeight

	// register cr
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
	}, nil)
	assert.Equal(t, 3, len(committee.GetCandidates(state.Pending)))
	assert.Equal(t, 0, len(committee.GetCandidates(state.Active)))

	// vote cr
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
		}, nil)
	}
	voteCRTx := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 3},
		{did2.Bytes(), 2},
		{did3.Bytes(), 1}})
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx,
		},
	}, nil)

	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight - 1
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 0, len(committee.GetCurrentMembers()))
	assert.Equal(t, 3, len(committee.GetAllCandidates()))
	// process

	//here change committee
	committee.Params.CRConfiguration.CRAgreementCount = 2
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 2, len(committee.GetCurrentMembers()))
	assert.Equal(t, 0, len(committee.GetAllCandidates()))
	committee.GetState().DepositInfo = make(map[common.Uint168]*state.DepositInfo)
	committee.GetState().DepositInfo[*did1] = &state.DepositInfo{
		DepositAmount: 5000 * 1e8,
		TotalAmount:   5000 * 1e8,
		Penalty:       12,
	}
	committee.GetState().DepositInfo[*did2] = &state.DepositInfo{
		DepositAmount: 5000 * 1e8,
		TotalAmount:   5000 * 1e8,
		Penalty:       12,
	}
	keyFrameA := committee.Snapshot()

	//here process impeachment
	//generate impeachment tx
	impeachValue := committee.CirculationAmount*common.Fixed64(committee.
		Params.CRConfiguration.VoterRejectPercentage)/common.Fixed64(100) + 1
	impeachmentTx := getCRCImpeachmentTx(publicKeyStr1, did1, impeachValue, *address1Uint168)

	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			impeachmentTx,
		},
	}, nil)
	assert.Equal(t, false, committee.InElectionPeriod)
	keyFrameB := committee.Snapshot()

	// rollback
	currentHeight--
	err := committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	assert.Equal(t, committee.InElectionPeriod, true)

	keyFrameC := committee.Snapshot()

	// reprocess
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			impeachmentTx,
		},
	}, nil)
	assert.Equal(t, committee.InElectionPeriod, false)

	keyFrameD := committee.Snapshot()

	assert.Equal(t, true, committeeKeyFrameEqual(keyFrameA, keyFrameC))
	assert.Equal(t, false, committeeKeyFrameEqual(keyFrameA, keyFrameB))
	assert.Equal(t, true, committeeKeyFrameEqual(keyFrameB, keyFrameD))
	assert.Equal(t, false, committeeKeyFrameEqual(keyFrameB, keyFrameC))

}

func getCRCImpeachmentTx(publicKeyStr string, did *common.Uint168,
	value common.Fixed64, outPutAddr common.Uint168) interfaces.Transaction {
	outputs := []*common2.Output{}
	outputs = append(outputs, &common2.Output{
		Type:        common2.OTVote,
		ProgramHash: outPutAddr,
		Value:       common.Fixed64(10),
		Payload: &outputpayload.VoteOutput{
			Version: 1,
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.CRCImpeachment,
					CandidateVotes: []outputpayload.CandidateVotes{
						{did.Bytes(), value},
					},
				},
			},
		},
	})

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{},
		[]*common2.Input{},
		outputs,
		0,
		[]*program.Program{},
	)

	txn.SetPrograms([]*program.Program{&program.Program{
		Code:      getCodeByPubKeyStr(publicKeyStr),
		Parameter: nil,
	}})
	return txn
}

func TestCommitee_RollbackCRCBlendTx(t *testing.T) {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	did1 := getCIDByPublicKeyStr(publicKeyStr1)
	nickName1 := "nickname 1"
	address1Uint168, _ := getProgramHash(publicKeyStr1)

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	did2 := getCIDByPublicKeyStr(publicKeyStr2)
	nickName2 := "nickname 2"

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"
	did3 := getCIDByPublicKeyStr(publicKeyStr3)
	nickName3 := "nickname 3"

	privateKeyStr4 := "b3b1c16abd786c4994af9ee8c79d25457f66509731f74d6a9a9673ca872fa8fa"

	registerCRTxn1 := getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	registerCRTxn2 := getRegisterCRTx(publicKeyStr2, privateKeyStr2, nickName2)
	registerCRTxn3 := getRegisterCRTx(publicKeyStr3, privateKeyStr3, nickName3)

	CkpManager := checkpoint.NewManager(config.GetDefaultParams())
	// new committee
	committee := state.NewCommittee(&config.DefaultParams, CkpManager)
	// set count of CR member
	cfg := &config.DefaultParams
	cfg.DPoSConfiguration.CRCArbiters = cfg.DPoSConfiguration.CRCArbiters[0:2]
	cfg.CRConfiguration.MemberCount = 2
	cfg.CRConfiguration.CRAgreementCount = 2

	// avoid getting UTXOs from database
	currentHeight := cfg.CRConfiguration.CRVotingStartHeight
	// register cr
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
	}, nil)
	assert.Equal(t, 3, len(committee.GetCandidates(state.Pending)))

	// vote cr
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
		}, nil)
	}
	voteCRTx := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 3},
		{did2.Bytes(), 2},
		{did3.Bytes(), 1}})
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx,
		},
	}, nil)
	assert.Equal(t, common.Fixed64(3), committee.GetCandidate(*did1).Votes)

	elaAddress := "EZaqDYAPFsjynGpvHwbuiiiL4dEiHtX4gD"
	proposalTxA := getCRCProposalTx(elaAddress, publicKeyStr1, privateKeyStr1,
		publicKeyStr2, privateKeyStr2)
	proposalAHash := proposalTxA.Payload().(*payload.CRCProposal).Hash(payload.CRCProposalVersion01)

	proposalTxB := getCRCProposalTx(elaAddress, publicKeyStr1, privateKeyStr1,
		publicKeyStr2, privateKeyStr2)
	proposalBHash := proposalTxB.Payload().(*payload.CRCProposal).Hash(payload.CRCProposalVersion01)
	proposalTxC := getCRCProposalTx(elaAddress, publicKeyStr1, privateKeyStr1,
		publicKeyStr2, privateKeyStr2)
	proposalCHash := proposalTxC.Payload().(*payload.CRCProposal).Hash(payload.CRCProposalVersion01)
	proposalTxD := getCRCProposalTx(elaAddress, publicKeyStr1, privateKeyStr1,
		publicKeyStr2, privateKeyStr2)

	// end first voting period
	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{},
	}, nil)

	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			proposalTxA,
			proposalTxB,
			proposalTxC,
		},
	}, nil)
	assert.Equal(t, 3, len(committee.GetProposals(state.Registered)))
	assert.Equal(t, 2, len(committee.GetCurrentMembers()))

	// review proposal
	proposalReviewTxB1 := getCRCProposalReviewTx(proposalBHash, payload.Approve,
		publicKeyStr1, privateKeyStr1)
	proposalReviewTxB2 := getCRCProposalReviewTx(proposalBHash, payload.Approve,
		publicKeyStr2, privateKeyStr2)
	proposalReviewTxC1 := getCRCProposalReviewTx(proposalCHash, payload.Approve,
		publicKeyStr1, privateKeyStr1)
	proposalReviewTxC2 := getCRCProposalReviewTx(proposalCHash, payload.Approve,
		publicKeyStr2, privateKeyStr2)

	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			proposalReviewTxB1,
			proposalReviewTxB2,
			proposalReviewTxC1,
			proposalReviewTxC2,
		},
	}, nil)
	assert.Equal(t, state.Registered, committee.GetProposal(proposalBHash).Status)
	assert.Equal(t, state.Registered, committee.GetProposal(proposalCHash).Status)

	// change to CRAgreed
	currentHeight += cfg.CRConfiguration.ProposalCRVotingPeriod
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, state.CRAgreed, committee.GetProposal(proposalBHash).Status)
	assert.Equal(t, state.CRAgreed, committee.GetProposal(proposalCHash).Status)

	// register cr again
	currentHeight = config.DefaultParams.CRConfiguration.CRCommitteeStartHeight +
		cfg.CRConfiguration.DutyPeriod - cfg.CRConfiguration.VotingPeriod
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
		},
	}, nil)
	assert.Equal(t, 1, len(committee.GetCandidates(state.Pending)))
	assert.Equal(t, 0, len(committee.GetCandidates(state.Active)))
	keyFrameA := committee.Snapshot()

	voteCRTx2 := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 1}})
	// proposal tracking of type progress
	proposalTrackingBTx := getCRCProposalTrackingTx(
		payload.Progress, proposalBHash, 1, publicKeyStr1, privateKeyStr1,
		"", "", privateKeyStr4)
	// proposal withdraw
	withdrawCTx := getCRCProposalWithdrawTx(proposalCHash, publicKeyStr1,
		privateKeyStr1, 1, []*common2.Input{}, []*common2.Output{})
	impeachValue := committee.CirculationAmount*common.Fixed64(committee.
		Params.CRConfiguration.VoterRejectPercentage)/common.Fixed64(100) + 1
	//generate impeachment tx
	impeachmentTx := getCRCImpeachmentTx(publicKeyStr1, did1, impeachValue, *address1Uint168)

	proposalReviewTxA1 := getCRCProposalReviewTx(proposalAHash, payload.Approve,
		publicKeyStr1, privateKeyStr1)
	proposalReviewTxA2 := getCRCProposalReviewTx(proposalAHash, payload.Approve,
		publicKeyStr2, privateKeyStr2)

	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn2,
			registerCRTxn3,
			voteCRTx2,
			proposalTxD,
			proposalReviewTxA1,
			proposalReviewTxA2,
			proposalTrackingBTx,
			withdrawCTx,
			impeachmentTx,
		},
	}, nil)
	assert.Equal(t, 3, len(committee.GetCandidates(state.Pending)))
	assert.Equal(t, 0, len(committee.GetCandidates(state.Active)))
	assert.Equal(t, common.Fixed64(1), committee.GetCandidate(*did1).Votes)
	assert.Equal(t, 1, len(committee.GetProposals(state.Aborted)))
	assert.Equal(t, 2, len(committee.GetProposal(proposalBHash).
		WithdrawableBudgets))
	assert.Equal(t, 1, len(committee.GetProposal(proposalCHash).
		WithdrawnBudgets))
	assert.Equal(t, false, committee.InElectionPeriod)

	keyFrameB := committee.Snapshot()

	// rollback
	currentHeight--
	err := committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	//assert.Equal(t, common.Fixed64(1), committee.GetCandidate(*did1).Votes)
	keyFrameC := committee.Snapshot()

	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn2,
			registerCRTxn3,
			voteCRTx2,
			proposalTxD,
			proposalReviewTxA1,
			proposalReviewTxA2,
			proposalTrackingBTx,
			withdrawCTx,
			impeachmentTx,
		},
	}, nil)
	keyFrameD := committee.Snapshot()
	checkResult(t, keyFrameA, keyFrameB, keyFrameC, keyFrameD)
}

func TestCommitee_RollbackCRCBlendAppropriationTx(t *testing.T) {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	did1 := getCIDByPublicKeyStr(publicKeyStr1)
	nickName1 := "nickname 1"
	address1Uint168, _ := getProgramHash(publicKeyStr1)

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	did2 := getCIDByPublicKeyStr(publicKeyStr2)
	nickName2 := "nickname 2"

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"
	did3 := getCIDByPublicKeyStr(publicKeyStr3)
	nickName3 := "nickname 3"

	privateKeyStr4 := "b3b1c16abd786c4994af9ee8c79d25457f66509731f74d6a9a9673ca872fa8fa"

	registerCRTxn1 := getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	registerCRTxn2 := getRegisterCRTx(publicKeyStr2, privateKeyStr2, nickName2)
	registerCRTxn3 := getRegisterCRTx(publicKeyStr3, privateKeyStr3, nickName3)

	CkpManager := checkpoint.NewManager(config.GetDefaultParams())
	// new committee
	committee := state.NewCommittee(&config.DefaultParams, CkpManager)
	// set count of CR member
	cfg := &config.DefaultParams
	cfg.DPoSConfiguration.CRCArbiters = cfg.DPoSConfiguration.CRCArbiters[0:2]
	cfg.CRConfiguration.MemberCount = 2
	cfg.CRConfiguration.CRAgreementCount = 2

	// avoid getting UTXOs from database
	currentHeight := cfg.CRConfiguration.CRVotingStartHeight
	committee.RegisterFuncitons(&state.CommitteeFuncsConfig{
		GetHeight: func() uint32 {
			return currentHeight
		},
	})

	// register cr
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
	}, nil)
	assert.Equal(t, 3, len(committee.GetCandidates(state.Pending)))

	// vote cr
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
		}, nil)
	}
	voteCRTx := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 3},
		{did2.Bytes(), 2},
		{did3.Bytes(), 1}})
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx,
		},
	}, nil)
	assert.Equal(t, common.Fixed64(3), committee.GetCandidate(*did1).Votes)

	elaAddress := "EZaqDYAPFsjynGpvHwbuiiiL4dEiHtX4gD"
	proposalTxA := getCRCProposalTx(elaAddress, publicKeyStr1, privateKeyStr1,
		publicKeyStr2, privateKeyStr2)
	proposalAHash := proposalTxA.Payload().(*payload.CRCProposal).Hash(payload.CRCProposalVersion01)

	proposalTxB := getCRCProposalTx(elaAddress, publicKeyStr1, privateKeyStr1,
		publicKeyStr2, privateKeyStr2)
	proposalBHash := proposalTxB.Payload().(*payload.CRCProposal).Hash(payload.CRCProposalVersion01)
	proposalTxC := getCRCProposalTx(elaAddress, publicKeyStr1, privateKeyStr1,
		publicKeyStr2, privateKeyStr2)
	proposalCHash := proposalTxC.Payload().(*payload.CRCProposal).Hash(payload.CRCProposalVersion01)
	proposalTxD := getCRCProposalTx(elaAddress, publicKeyStr1, privateKeyStr1,
		publicKeyStr2, privateKeyStr2)

	// end first voting period
	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{},
	}, nil)

	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			proposalTxA,
			proposalTxB,
			proposalTxC,
		},
	}, nil)
	assert.Equal(t, 3, len(committee.GetProposals(state.Registered)))
	assert.Equal(t, 2, len(committee.GetCurrentMembers()))

	// review proposal
	proposalReviewTxB1 := getCRCProposalReviewTx(proposalBHash, payload.Approve,
		publicKeyStr1, privateKeyStr1)
	proposalReviewTxB2 := getCRCProposalReviewTx(proposalBHash, payload.Approve,
		publicKeyStr2, privateKeyStr2)
	proposalReviewTxC1 := getCRCProposalReviewTx(proposalCHash, payload.Approve,
		publicKeyStr1, privateKeyStr1)
	proposalReviewTxC2 := getCRCProposalReviewTx(proposalCHash, payload.Approve,
		publicKeyStr2, privateKeyStr2)

	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			proposalReviewTxB1,
			proposalReviewTxB2,
			proposalReviewTxC1,
			proposalReviewTxC2,
		},
	}, nil)
	assert.Equal(t, state.Registered, committee.GetProposal(proposalBHash).Status)
	assert.Equal(t, state.Registered, committee.GetProposal(proposalCHash).Status)

	// change to CRAgreed
	currentHeight += cfg.CRConfiguration.ProposalCRVotingPeriod
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, state.CRAgreed, committee.GetProposal(proposalBHash).Status)
	assert.Equal(t, state.CRAgreed, committee.GetProposal(proposalCHash).Status)

	// register cr again
	currentHeight = config.DefaultParams.CRConfiguration.CRCommitteeStartHeight +
		cfg.CRConfiguration.DutyPeriod - cfg.CRConfiguration.VotingPeriod
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
		},
	}, nil)
	assert.Equal(t, 1, len(committee.GetCandidates(state.Pending)))
	assert.Equal(t, 0, len(committee.GetCandidates(state.Active)))
	//keyFrameA := committee.Snapshot()

	voteCRTx2 := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 1}})

	// proposal tracking of type progress
	proposalTrackingBTx := getCRCProposalTrackingTx(
		payload.Progress, proposalBHash, 1, publicKeyStr1, privateKeyStr1,
		"", "", privateKeyStr4)
	// proposal withdraw
	withdrawCTx := getCRCProposalWithdrawTx(proposalCHash, publicKeyStr1,
		privateKeyStr1, 1, []*common2.Input{}, []*common2.Output{})
	impeachValue := committee.CirculationAmount*common.Fixed64(committee.
		Params.CRConfiguration.VoterRejectPercentage)/common.Fixed64(100) + 1
	//generate impeachment tx
	impeachmentTx := getCRCImpeachmentTx(publicKeyStr1, did1, impeachValue, *address1Uint168)

	proposalReviewTxA1 := getCRCProposalReviewTx(proposalAHash, payload.Approve,
		publicKeyStr1, privateKeyStr1)
	proposalReviewTxA2 := getCRCProposalReviewTx(proposalAHash, payload.Approve,
		publicKeyStr2, privateKeyStr2)

	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn2,
			registerCRTxn3,
			voteCRTx2,
			proposalTxD,
			proposalReviewTxA1,
			proposalReviewTxA2,
			proposalTrackingBTx,
			withdrawCTx,
			impeachmentTx,
		},
	}, nil)
	assert.Equal(t, 3, len(committee.GetCandidates(state.Pending)))
	assert.Equal(t, 0, len(committee.GetCandidates(state.Active)))
	assert.Equal(t, common.Fixed64(1), committee.GetCandidate(*did1).Votes)
	assert.Equal(t, 1, len(committee.GetProposals(state.Aborted)))
	assert.Equal(t, 2, len(committee.GetProposal(proposalBHash).
		WithdrawableBudgets))
	assert.Equal(t, 1, len(committee.GetProposal(proposalCHash).
		WithdrawnBudgets))
	assert.Equal(t, false, committee.InElectionPeriod)

	// rollback
	currentHeight--
	err := committee.RollbackTo(currentHeight)
	assert.NoError(t, err)

	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn2,
			registerCRTxn3,
			voteCRTx2,
		},
	}, nil)

	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight + cfg.CRConfiguration.DutyPeriod + 1
	committee.LastVotingStartHeight = currentHeight - cfg.CRConfiguration.VotingPeriod
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 2, len(committee.GetCurrentMembers()))
	keyFrameA := committee.Snapshot()

	//process appropriation tx
	crcCommiteeAddressStr := "ESq12oQrvGqHfTkEDYJyR9MxZj1NMnonjo"
	crcCommiteeAddrHash, _ := common.Uint168FromAddress(crcCommiteeAddressStr)
	txAppropriate := getAppropriationTx(500.0, *crcCommiteeAddrHash)
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			txAppropriate,
			proposalTxD,
			proposalReviewTxA1,
			proposalReviewTxA2,
			proposalTrackingBTx,
			withdrawCTx,
			impeachmentTx,
		},
	}, nil)

	keyFrameB := committee.Snapshot()

	currentHeight--
	err = committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	keyFrameC := committee.Snapshot()

	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			txAppropriate,
			proposalTxD,
			proposalReviewTxA1,
			proposalReviewTxA2,
			proposalTrackingBTx,
			withdrawCTx,
			impeachmentTx,
		},
	}, nil)
	keyFrameD := committee.Snapshot()
	checkResult(t, keyFrameA, keyFrameB, keyFrameC, keyFrameD)
}

func TestCommitee_RollbackCRCBlendTxPropoalVert(t *testing.T) {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	did1 := getCIDByPublicKeyStr(publicKeyStr1)
	nickName1 := "nickname 1"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	did2 := getCIDByPublicKeyStr(publicKeyStr2)
	nickName2 := "nickname 2"

	privateKeyStr4 := "b3b1c16abd786c4994af9ee8c79d25457f66509731f74d6a9a9673ca872fa8fa"

	registerCRTxn1 := getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	registerCRTxn2 := getRegisterCRTx(publicKeyStr2, privateKeyStr2, nickName2)

	CkpManager := checkpoint.NewManager(config.GetDefaultParams())
	// new committee
	committee := state.NewCommittee(&config.DefaultParams, CkpManager)

	// set count of CR member
	cfg := &config.DefaultParams
	cfg.DPoSConfiguration.CRCArbiters = cfg.DPoSConfiguration.CRCArbiters[0:2]
	cfg.CRConfiguration.MemberCount = 2
	cfg.CRConfiguration.CRAgreementCount = 2

	// avoid getting UTXOs from database
	currentHeight := cfg.CRConfiguration.CRVotingStartHeight

	// register cr
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
		},
	}, nil)
	keyFrameA := committee.Snapshot()
	assert.Equal(t, 2, len(committee.GetCandidates(state.Pending)))

	// vote cr
	currentHeight += 5
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
	}, nil)

	voteCRTx := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 3},
		{did2.Bytes(), 2},
	})
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx,
		},
	}, nil)
	assert.Equal(t, common.Fixed64(3), committee.GetCandidate(*did1).Votes)

	//proposal tx
	elaAddress := "EZaqDYAPFsjynGpvHwbuiiiL4dEiHtX4gD"

	proposalTxB := getCRCProposalTx(elaAddress, publicKeyStr1, privateKeyStr1,
		publicKeyStr2, privateKeyStr2)
	proposalBHash := proposalTxB.Payload().(*payload.CRCProposal).Hash(payload.CRCProposalVersion01)

	proposalTxC := getCRCProposalTx(elaAddress, publicKeyStr1, privateKeyStr1,
		publicKeyStr2, privateKeyStr2)
	proposalCHash := proposalTxC.Payload().(*payload.CRCProposal).Hash(payload.CRCProposalVersion01)

	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{},
	}, nil)

	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			proposalTxB,
			proposalTxC,
		},
	}, nil)
	assert.Equal(t, 2, len(committee.GetProposals(state.Registered)))
	assert.Equal(t, 2, len(committee.GetCurrentMembers()))

	// review proposal
	proposalReviewTxB1 := getCRCProposalReviewTx(proposalBHash, payload.Approve,
		publicKeyStr1, privateKeyStr1)
	proposalReviewTxB2 := getCRCProposalReviewTx(proposalBHash, payload.Approve,
		publicKeyStr2, privateKeyStr2)
	proposalReviewTxC1 := getCRCProposalReviewTx(proposalCHash, payload.Approve,
		publicKeyStr1, privateKeyStr1)
	proposalReviewTxC2 := getCRCProposalReviewTx(proposalCHash, payload.Approve,
		publicKeyStr2, privateKeyStr2)
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			proposalReviewTxB1,
			proposalReviewTxB2,
			proposalReviewTxC1,
			proposalReviewTxC2,
		},
	}, nil)
	assert.Equal(t, state.Registered, committee.GetProposal(proposalBHash).Status)
	assert.Equal(t, state.Registered, committee.GetProposal(proposalCHash).Status)

	// register to CRAgreed
	currentHeight += cfg.CRConfiguration.ProposalPublicVotingPeriod
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, state.CRAgreed, committee.GetProposal(proposalBHash).Status)
	assert.Equal(t, state.CRAgreed, committee.GetProposal(proposalCHash).Status)

	// change to CRAgreed
	currentHeight += cfg.CRConfiguration.ProposalCRVotingPeriod
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 1, len(committee.GetProposal(proposalBHash).
		WithdrawableBudgets))
	assert.Equal(t, 1, len(committee.GetProposal(proposalCHash).
		WithdrawableBudgets))

	proposalTrackingBTx := getCRCProposalTrackingTx(
		payload.Progress, proposalBHash, 1, publicKeyStr1, privateKeyStr1,
		"", "", privateKeyStr4)
	proposalTrackingCTx := getCRCProposalTrackingTx(
		payload.Progress, proposalCHash, 1, publicKeyStr1, privateKeyStr1,
		"", "", privateKeyStr4)

	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			proposalTrackingBTx,
			proposalTrackingCTx,
		},
	}, nil)
	assert.Equal(t, 2, len(committee.GetProposal(proposalBHash).
		WithdrawableBudgets))
	assert.Equal(t, 2, len(committee.GetProposal(proposalBHash).
		WithdrawableBudgets))

	proposalTrackingBTxFinal := getCRCProposalTrackingTx(
		payload.Finalized, proposalBHash, 1, publicKeyStr1, privateKeyStr1,
		"", "", privateKeyStr4)
	proposalTrackingCTxFinal := getCRCProposalTrackingTx(
		payload.Finalized, proposalCHash, 1, publicKeyStr1, privateKeyStr1,
		"", "", privateKeyStr4)
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			proposalTrackingBTxFinal,
			proposalTrackingCTxFinal,
		},
	}, nil)
	assert.Equal(t, 3, len(committee.GetProposal(proposalBHash).
		WithdrawableBudgets))
	assert.Equal(t, 3, len(committee.GetProposal(proposalCHash).
		WithdrawableBudgets))

	// proposal withdraw
	withdrawBTx := getCRCProposalWithdrawTx(proposalBHash, publicKeyStr1,
		privateKeyStr1, 1, []*common2.Input{}, []*common2.Output{})
	withdrawCTx := getCRCProposalWithdrawTx(proposalCHash, publicKeyStr1,
		privateKeyStr1, 1, []*common2.Input{}, []*common2.Output{})
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			withdrawBTx,
			withdrawCTx,
		},
	}, nil)
	assert.Equal(t, 3, len(committee.GetProposal(proposalBHash).
		WithdrawnBudgets))
	assert.Equal(t, 3, len(committee.GetProposal(proposalCHash).
		WithdrawnBudgets))

	// rollback
	currentHeight = cfg.CRConfiguration.CRVotingStartHeight
	err := committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	keyFrameC := committee.Snapshot()
	assert.Equal(t, true, committeeKeyFrameEqual(keyFrameA, keyFrameC))
}

func registerFuncs(s *state.State) {
	s.RegisterFunctions(&state.FunctionsConfig{
		GetHistoryMember: func(code []byte) []*state.CRMember { return nil },
		GetTxReference: func(tx interfaces.Transaction) (
			map[*common2.Input]common2.Output, error) {
			return make(map[*common2.Input]common2.Output), nil
		}})
}
func TestCommitee_RollbackCRCBlendTxCRVert(t *testing.T) {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	cid1 := getCIDByPublicKeyStr(publicKeyStr1)
	nickName1 := "nickname 1"
	newNickName1 := "newNickName1"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	cid2 := getCIDByPublicKeyStr(publicKeyStr2)
	nickName2 := "nickname 2"
	newNickName2 := "newNickName2"

	registerCRTxn1 := getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	registerCRTxn2 := getRegisterCRTx(publicKeyStr2, privateKeyStr2, nickName2)

	CkpManager := checkpoint.NewManager(config.GetDefaultParams())
	// new committee
	committee := state.NewCommittee(&config.DefaultParams, CkpManager)
	registerFuncs(committee.GetState())

	// set count of CR member
	cfg := &config.DefaultParams
	cfg.DPoSConfiguration.CRCArbiters = cfg.DPoSConfiguration.CRCArbiters[0:2]
	cfg.CRConfiguration.MemberCount = 2
	// avoid getting UTXOs from database

	currentHeight := cfg.CRConfiguration.CRVotingStartHeight
	// register cr
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
		},
	}, nil)
	keyFrameA := committee.Snapshot()
	assert.Equal(t, 2, len(committee.GetCandidates(state.Pending)))

	updateCr1 := getUpdateCR(publicKeyStr1, *cid1,
		newNickName1)
	updateCR2 := getUpdateCR(publicKeyStr2, *cid2,
		newNickName2)
	currentHeight++
	// update cr
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			updateCr1,
			updateCR2,
		},
	}, nil)
	assert.Equal(t, true, committee.ExistCandidateByNickname(newNickName1))
	assert.Equal(t, true, committee.ExistCandidateByNickname(newNickName2))

	unregister1 := getUnregisterCR(*cid1)
	unregister2 := getUnregisterCR(*cid2)
	currentHeight++
	// unregister cr
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			unregister1,
			unregister2,
		},
	}, nil)
	assert.Equal(t, state.Canceled, committee.GetCandidate(*cid1).State)
	assert.Equal(t, state.Canceled, committee.GetCandidate(*cid2).State)

	returnDepositTx1 := generateReturnDeposite(publicKeyStr1)
	returnDepositTx2 := generateReturnDeposite(publicKeyStr2)
	currentHeight += committee.Params.CRConfiguration.DepositLockupBlocks + 1
	// returnDepositTx
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			returnDepositTx1,
			returnDepositTx2,
		},
	}, nil)

	assert.Equal(t, state.Returned, committee.GetCandidate(*cid1).State)
	assert.Equal(t, state.Returned, committee.GetCandidate(*cid2).State)

	// rollback
	currentHeight = cfg.CRConfiguration.CRVotingStartHeight
	err := committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	keyFrameC := committee.Snapshot()
	assert.Equal(t, true, committeeKeyFrameEqual(keyFrameA, keyFrameC))
}

func TestCommittee_RollbackReview(t *testing.T) {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	did1 := getDIDByPublicKey(publicKeyStr1)
	nickName1 := "nickname 1"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	did2 := getDIDByPublicKey(publicKeyStr2)
	nickName2 := "nickname 2"

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"
	did3 := getDIDByPublicKey(publicKeyStr3)
	nickName3 := "nickname 3"

	registerCRTxn1 := getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	registerCRTxn2 := getRegisterCRTx(publicKeyStr2, privateKeyStr2, nickName2)
	registerCRTxn3 := getRegisterCRTx(publicKeyStr3, privateKeyStr3, nickName3)

	CkpManager := checkpoint.NewManager(config.GetDefaultParams())
	// new committee
	committee := state.NewCommittee(&config.DefaultParams, CkpManager)

	// set count of CR member
	cfg := &config.DefaultParams
	cfg.DPoSConfiguration.CRCArbiters = cfg.DPoSConfiguration.CRCArbiters[0:2]
	cfg.CRConfiguration.MemberCount = 2

	// avoid getting UTXOs from database
	currentHeight := cfg.CRConfiguration.CRVotingStartHeight
	//committee.recordBalanceHeight = currentHeight - 1

	// register cr
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
			registerCRTxn3,
		},
	}, nil)

	// vote cr
	for i := 0; i < 5; i++ {
		currentHeight++
		committee.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: currentHeight,
			},
		}, nil)
	}

	voteCRTx := getVoteCRTx(6, []outputpayload.CandidateVotes{
		{did1.Bytes(), 3},
		{did2.Bytes(), 2},
		{did3.Bytes(), 1}})
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: currentHeight,
		},
		Transactions: []interfaces.Transaction{
			voteCRTx,
		},
	}, nil)
	assert.Equal(t, common.Fixed64(3), committee.GetCandidate(*did1).Votes)

	// end first voting period
	currentHeight = cfg.CRConfiguration.CRCommitteeStartHeight
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight}}, nil)
	assert.Equal(t, 2, len(committee.GetCurrentMembers()))
	keyFrameA := committee.Snapshot()

	// create CRC proposal tx
	elaAddress := "EZaqDYAPFsjynGpvHwbuiiiL4dEiHtX4gD"
	proposalTx := getCRCProposalTx(elaAddress, publicKeyStr1, privateKeyStr1,
		publicKeyStr2, privateKeyStr2)
	proposalHash := proposalTx.Payload().(*payload.CRCProposal).Hash(payload.CRCProposalVersion01)
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalTx,
		}}, nil)
	assert.Equal(t, 1, len(committee.GetProposals(state.Registered)))
	//assert.Equal(t, 2, committee.GetProposal(proposalTx.Payload.(*payload.CRCProposal).Hash()))
	keyFrameB := committee.Snapshot()

	// rollback
	currentHeight--
	err := committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(committee.GetProposals(state.Registered)))
	keyFrameC := committee.Snapshot()

	// reprocess
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalTx,
		}}, nil)
	assert.Equal(t, 1, len(committee.GetProposals(state.Registered)))
	keyFrameD := committee.Snapshot()

	checkResult(t, keyFrameA, keyFrameB, keyFrameC, keyFrameD)

	// set CR agreement count
	committee.Params.CRConfiguration.CRAgreementCount = 2

	// review proposal Approve
	proposalReviewTx1 := getCRCProposalReviewTx(proposalHash, payload.Approve,
		publicKeyStr1, privateKeyStr1)
	proposalReviewTx2 := getCRCProposalReviewTx(proposalHash, payload.Approve,
		publicKeyStr2, privateKeyStr2)

	// process
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalReviewTx1,
			proposalReviewTx2,
		}}, nil)
	keyFrameA2 := committee.Snapshot()
	assert.Equal(t, state.Registered, committee.GetProposal(proposalHash).Status)

	// review proposal Approve
	proposalReviewTxReject1 := getCRCProposalReviewTx(proposalHash,
		payload.Reject, publicKeyStr1, privateKeyStr1)
	proposalReviewTxReject2 := getCRCProposalReviewTx(proposalHash,
		payload.Reject, publicKeyStr2, privateKeyStr2)

	// process
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalReviewTxReject1,
			proposalReviewTxReject2,
		}}, nil)
	keyFrameB2 := committee.Snapshot()

	// rollback
	currentHeight--
	err = committee.RollbackTo(currentHeight)
	assert.NoError(t, err)
	keyFrameC2 := committee.Snapshot()

	// reprocess
	currentHeight++
	committee.ProcessBlock(&types.Block{
		Header: common2.Header{Height: currentHeight},
		Transactions: []interfaces.Transaction{
			proposalReviewTxReject1,
			proposalReviewTxReject2,
		}}, nil)
	keyFrameD2 := committee.Snapshot()
	assert.Equal(t, state.Registered, committee.GetProposal(proposalHash).Status)

	checkResult(t, keyFrameA2, keyFrameB2, keyFrameC2, keyFrameD2)

}
