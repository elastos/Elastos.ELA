package transaction

import (
	"bytes"
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

func (s *txValidatorTestSuite) TestCheckCRCProposalTrackingTransaction() {
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"

	ownerPubKey, _ := common.HexStringToBytes(publicKeyStr1)

	proposalHash := randomUint256()
	recipient := randomUint168()
	votingHeight := config.DefaultParams.CRConfiguration.CRVotingStartHeight

	// Set secretary general.
	s.Chain.GetCRCommittee().GetProposalManager().SecretaryGeneralPublicKey = publicKeyStr3
	// Check Common tracking tx.
	txn := s.getCRCProposalTrackingTx(payload.Common, *proposalHash, 0,
		publicKeyStr1, privateKeyStr1, "", "",
		publicKeyStr3, privateKeyStr3)

	pld := payload.CRCProposal{
		ProposalType:       0,
		OwnerKey:           ownerPubKey,
		CRCouncilMemberDID: *randomUint168(),
		DraftHash:          *randomUint256(),
		Budgets:            createBudgets(3),
		Recipient:          *recipient,
	}
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[*proposalHash] =
		&crstate.ProposalState{
			Proposal:      pld.ToProposalInfo(0),
			Status:        crstate.VoterAgreed,
			ProposalOwner: ownerPubKey,
		}

	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ := txn.SpecialContextCheck()
	s.NoError(err)

	txn = s.getCRCProposalTrackingTx(payload.Common, *proposalHash, 1,
		publicKeyStr1, privateKeyStr1, publicKeyStr2, privateKeyStr2,
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:stage should assignment zero value")

	txn = s.getCRCProposalTrackingTx(payload.Common, *proposalHash, 0,
		publicKeyStr1, privateKeyStr1, publicKeyStr2, privateKeyStr2,
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:the NewOwnerKey need to be empty")

	// Check Progress tracking tx.
	txn = s.getCRCProposalTrackingTx(payload.Progress, *proposalHash, 1,
		publicKeyStr1, privateKeyStr1, "", "",
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	txn = s.getCRCProposalTrackingTx(payload.Progress, *proposalHash, 1,
		publicKeyStr1, privateKeyStr1, publicKeyStr2, privateKeyStr2,
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:the NewOwnerKey need to be empty")

	// Check Terminated tracking tx.
	txn = s.getCRCProposalTrackingTx(payload.Terminated, *proposalHash, 0,
		publicKeyStr1, privateKeyStr1, "", "",
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	txn = s.getCRCProposalTrackingTx(payload.Terminated, *proposalHash, 1,
		publicKeyStr1, privateKeyStr1, publicKeyStr2, privateKeyStr2,
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:stage should assignment zero value")

	txn = s.getCRCProposalTrackingTx(payload.Terminated, *proposalHash, 0,
		publicKeyStr1, privateKeyStr1, publicKeyStr2, privateKeyStr2,
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:the NewOwnerKey need to be empty")

	// Check ChangeOwner tracking tx.
	txn = s.getCRCProposalTrackingTx(payload.ChangeOwner, *proposalHash, 0,
		publicKeyStr1, privateKeyStr1, publicKeyStr2, privateKeyStr2,
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	txn = s.getCRCProposalTrackingTx(payload.ChangeOwner, *proposalHash, 1,
		publicKeyStr1, privateKeyStr1, publicKeyStr2, privateKeyStr2,
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:stage should assignment zero value")

	txn = s.getCRCProposalTrackingTx(payload.ChangeOwner, *proposalHash, 0,
		publicKeyStr1, privateKeyStr1, "", "",
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid new proposal owner public key")

	// Check invalid proposal hash.
	txn = s.getCRCProposalTrackingTx(payload.Common, *randomUint256(), 0,
		publicKeyStr1, privateKeyStr1, "", "",
		publicKeyStr3, privateKeyStr3)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:proposal not exist")

	txn = s.getCRCProposalTrackingTx(payload.Common, *proposalHash, 0,
		publicKeyStr1, privateKeyStr1, "", "",
		publicKeyStr3, privateKeyStr3)

	// Check proposal status is not VoterAgreed.
	pld = payload.CRCProposal{
		ProposalType:       0,
		OwnerKey:           ownerPubKey,
		CRCouncilMemberDID: *randomUint168(),
		DraftHash:          *randomUint256(),
		Budgets:            createBudgets(3),
		Recipient:          *recipient,
	}
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[*proposalHash] =
		&crstate.ProposalState{
			Proposal:         pld.ToProposalInfo(0),
			TerminatedHeight: 100,
			Status:           crstate.VoterCanceled,
			ProposalOwner:    ownerPubKey,
		}
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[*proposalHash].TerminatedHeight = 100
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:proposal status is not VoterAgreed")

	// Check reach max proposal tracking count.
	pld = payload.CRCProposal{
		ProposalType:       0,
		OwnerKey:           ownerPubKey,
		CRCouncilMemberDID: *randomUint168(),
		DraftHash:          *randomUint256(),
		Budgets:            createBudgets(3),
		Recipient:          *recipient,
	}
	s.Chain.GetCRCommittee().GetProposalManager().Proposals[*proposalHash] =
		&crstate.ProposalState{
			Proposal:      pld.ToProposalInfo(0),
			TrackingCount: 128,
			Status:        crstate.VoterAgreed,
			ProposalOwner: ownerPubKey,
		}
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:reached max tracking count")

}

func (s *txValidatorTestSuite) getCRCProposalTrackingTx(
	trackingType payload.CRCProposalTrackingType,
	proposalHash common.Uint256, stage uint8,
	ownerPublicKeyStr, ownerPrivateKeyStr,
	newownerpublickeyStr, newownerprivatekeyStr,
	sgPublicKeyStr, sgPrivateKeyStr string) interfaces.Transaction {

	ownerPublicKey, _ := common.HexStringToBytes(ownerPublicKeyStr)
	ownerPrivateKey, _ := common.HexStringToBytes(ownerPrivateKeyStr)

	newownerpublickey, _ := common.HexStringToBytes(newownerpublickeyStr)
	newownerprivatekey, _ := common.HexStringToBytes(newownerprivatekeyStr)

	sgPrivateKey, _ := common.HexStringToBytes(sgPrivateKeyStr)

	documentData := randomBytes(10)
	opinionHash := randomBytes(10)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposalTracking,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	cPayload := &payload.CRCProposalTracking{
		ProposalTrackingType:        trackingType,
		ProposalHash:                proposalHash,
		Stage:                       stage,
		MessageHash:                 common.Hash(documentData),
		OwnerKey:                    ownerPublicKey,
		NewOwnerKey:                 newownerpublickey,
		SecretaryGeneralOpinionHash: common.Hash(opinionHash),
	}

	signBuf := new(bytes.Buffer)
	cPayload.SerializeUnsigned(signBuf, payload.CRCProposalTrackingVersion)
	sig, _ := crypto.Sign(ownerPrivateKey, signBuf.Bytes())
	cPayload.OwnerSignature = sig
	common.WriteVarBytes(signBuf, sig)

	if newownerpublickeyStr != "" && newownerprivatekeyStr != "" {
		crSig, _ := crypto.Sign(newownerprivatekey, signBuf.Bytes())
		cPayload.NewOwnerSignature = crSig
		sig = crSig
	} else {
		sig = []byte{}
	}

	common.WriteVarBytes(signBuf, sig)
	signBuf.Write([]byte{byte(cPayload.ProposalTrackingType)})
	cPayload.SecretaryGeneralOpinionHash.Serialize(signBuf)
	crSig, _ := crypto.Sign(sgPrivateKey, signBuf.Bytes())
	cPayload.SecretaryGeneralSignature = crSig

	txn.SetPayload(cPayload)
	return txn
}
