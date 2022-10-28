package transaction

import (
	"bytes"
	"fmt"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
)

func (s *txValidatorTestSuite) TestCheckCRCProposalReviewTransaction() {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	tenureHeight := config.DefaultParams.CRConfiguration.CRCommitteeStartHeight
	nickName1 := "nickname 1"

	fmt.Println("getcode ", getCodeHexStr("02e23f70b9b967af35571c32b1442d787c180753bbed5cd6e7d5a5cfe75c7fc1ff"))

	member1 := s.getCRMember(publicKeyStr1, privateKeyStr1, nickName1)
	s.Chain.GetCRCommittee().Members[member1.Info.DID] = member1

	// ok
	txn := s.getCRCProposalReviewTx(publicKeyStr1, privateKeyStr1)
	crcProposalReview, _ := txn.Payload().(*payload.CRCProposalReview)
	manager := s.Chain.GetCRCommittee().GetProposalManager()
	manager.Proposals[crcProposalReview.ProposalHash] = &crstate.ProposalState{
		Status: crstate.Registered,
	}
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: tenureHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ := txn.SpecialContextCheck()
	s.NoError(err)

	// member status is not elected
	member1.MemberState = crstate.MemberImpeached
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:should be an elected CR members")

	// invalid payload
	txn.SetPayload(&payload.CRInfo{})
	member1.MemberState = crstate.MemberElected
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid payload")

	// invalid content type
	txn = s.getCRCProposalReviewTx(publicKeyStr1, privateKeyStr1)
	txn.Payload().(*payload.CRCProposalReview).VoteResult = 0x10
	crcProposalReview2, _ := txn.Payload().(*payload.CRCProposalReview)
	manager.Proposals[crcProposalReview2.ProposalHash] = &crstate.ProposalState{
		Status: crstate.Registered,
	}
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: tenureHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:VoteResult should be known")

	// proposal reviewer is not CR member
	txn = s.getCRCProposalReviewTx(publicKeyStr2, privateKeyStr2)
	crcProposalReview3, _ := txn.Payload().(*payload.CRCProposalReview)
	manager.Proposals[crcProposalReview3.ProposalHash] = &crstate.ProposalState{
		Status: crstate.Registered,
	}
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: tenureHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:did correspond crMember not exists")

	delete(manager.Proposals, crcProposalReview.ProposalHash)
	// invalid CR proposal reviewer signature
	txn = s.getCRCProposalReviewTx(publicKeyStr1, privateKeyStr1)
	txn.Payload().(*payload.CRCProposalReview).Signature = []byte{}
	crcProposalReview, _ = txn.Payload().(*payload.CRCProposalReview)
	manager.Proposals[crcProposalReview.ProposalHash] = &crstate.ProposalState{
		Status: crstate.Registered,
	}
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: tenureHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid signature length")
	delete(s.Chain.GetCRCommittee().GetProposalManager().Proposals, crcProposalReview.ProposalHash)
}

func (s *txValidatorTestSuite) getCRCProposalReviewTx(crPublicKeyStr,
	crPrivateKeyStr string) interfaces.Transaction {

	privateKey1, _ := common.HexStringToBytes(crPrivateKeyStr)
	code := getCodeByPubKeyStr(crPublicKeyStr)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposalReview,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	did, _ := blockchain.GetDIDFromCode(code)
	crcProposalReviewPayload := &payload.CRCProposalReview{
		ProposalHash: *randomUint256(),
		VoteResult:   payload.Approve,
		DID:          *did,
	}

	signBuf := new(bytes.Buffer)
	crcProposalReviewPayload.SerializeUnsigned(signBuf, payload.CRCProposalReviewVersion)
	sig, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crcProposalReviewPayload.Signature = sig

	txn.SetPayload(crcProposalReviewPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(crPublicKeyStr),
		Parameter: nil,
	}})
	return txn
}
