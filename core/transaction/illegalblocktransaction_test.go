// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//
package transaction

import (
	"bytes"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"math/rand"
	"time"
)

func (s *txValidatorTestSuite) TestIllegalBlockTransaction() {

	publicKeyStr1 := "031e12374bae471aa09ad479f66c2306f4bcc4ca5b754609a82a1839b94b4721b9"

	illegalBlock := &payload.DPOSIllegalBlocks{}

	programs := []*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr1),
		Parameter: nil,
	}}

	txn := functions.CreateTransaction(
		0,
		common2.IllegalBlockEvidence,
		payload.IllegalBlockVersion,
		illegalBlock,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		programs,
	)

	txn = CreateTransactionByType(txn, s.Chain)
	hash := illegalBlock.Hash()
	s.Chain.GetState().SpecialTxHashes[hash] = struct{}{}
	err, _ := txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: transaction already exist:tx already exists")

	s.Chain.GetState().SpecialTxHashes = make(map[common.Uint256]struct{}, 0)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:blocks can not be same")

	illegalBlock = &payload.DPOSIllegalBlocks{
		Evidence: payload.BlockEvidence{
			Header: []byte{0x02},
		},
		CompareEvidence: payload.BlockEvidence{
			Header: []byte{0x01},
		},
	}
	txn.SetPayload(illegalBlock)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:evidence order error")

	illegalBlock = &payload.DPOSIllegalBlocks{
		CoinType: 1,
		Evidence: payload.BlockEvidence{
			Header: []byte{0x01},
		},
		CompareEvidence: payload.BlockEvidence{
			Header: []byte{0x02},
		},
	}
	txn.SetPayload(illegalBlock)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:unknown coin type")

	h1 := &common2.Header{
		Version:    0,
		Previous:   *randomUint256(),
		MerkleRoot: *randomUint256(),
		Timestamp:  uint32(time.Now().Unix()),
		Bits:       10000,
		Nonce:      0,
		Height:     100,
	}

	h2 := &common2.Header{
		Version:    0,
		Previous:   *randomUint256(),
		MerkleRoot: *randomUint256(),
		Timestamp:  uint32(time.Now().Unix()),
		Bits:       10000,
		Nonce:      0,
		Height:     100,
	}
	buf := new(bytes.Buffer)
	h1.Serialize(buf)
	buf1 := new(bytes.Buffer)
	h2.Serialize(buf1)

	h1bytes := buf.Bytes()
	h2bytes := buf1.Bytes()
	if common.BytesToHexString(h1bytes) >
		common.BytesToHexString(h2bytes) {
		illegalBlock = &payload.DPOSIllegalBlocks{
			CoinType: payload.ELACoin,
			Evidence: payload.BlockEvidence{
				Header: h2bytes,
			},
			CompareEvidence: payload.BlockEvidence{
				Header: h1bytes,
			},
		}
	} else {
		illegalBlock = &payload.DPOSIllegalBlocks{
			CoinType: payload.ELACoin,
			Evidence: payload.BlockEvidence{
				Header: h1bytes,
			},
			CompareEvidence: payload.BlockEvidence{
				Header: h2bytes,
			},
		}
	}
	txn.SetPayload(illegalBlock)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:block header height should be same")
}

func (s *txValidatorSpecialTxTestSuite) TestCheckDPOSIllegalBlocks() {
	header := randomBlockHeader()
	buf := new(bytes.Buffer)
	header.Serialize(buf)
	evidence := &payload.BlockEvidence{
		Header:       buf.Bytes(),
		BlockConfirm: []byte{},
		Signers:      [][]byte{},
	}

	illegalBlocks := &payload.DPOSIllegalBlocks{
		CoinType:        payload.ELACoin,
		BlockHeight:     rand.Uint32(),
		Evidence:        *evidence,
		CompareEvidence: *evidence,
	}
	s.EqualError(blockchain.CheckDPOSIllegalBlocks(illegalBlocks),
		"blocks can not be same")

	header2 := randomBlockHeader()
	buf = new(bytes.Buffer)
	header2.Serialize(buf)
	cmpEvidence := &payload.BlockEvidence{
		Header:       buf.Bytes(),
		BlockConfirm: []byte{},
		Signers:      [][]byte{},
	}

	asc := bytes.Compare(evidence.Header, cmpEvidence.Header) < 0
	if asc {
		illegalBlocks.Evidence = *cmpEvidence
		illegalBlocks.CompareEvidence = *evidence
	} else {
		illegalBlocks.Evidence = *evidence
		illegalBlocks.CompareEvidence = *cmpEvidence
	}
	s.EqualError(blockchain.CheckDPOSIllegalBlocks(illegalBlocks),
		"evidence order error")

	illegalBlocks.CoinType = payload.CoinType(1) //
	if asc {
		illegalBlocks.Evidence = *evidence
		illegalBlocks.CompareEvidence = *cmpEvidence
	} else {
		illegalBlocks.Evidence = *cmpEvidence
		illegalBlocks.CompareEvidence = *evidence
	}
	s.EqualError(blockchain.CheckDPOSIllegalBlocks(illegalBlocks),
		"unknown coin type")

	illegalBlocks.CoinType = payload.ELACoin
	s.EqualError(blockchain.CheckDPOSIllegalBlocks(illegalBlocks),
		"block header height should be same")

	// compare evidence height is different from illegal block height
	illegalBlocks.BlockHeight = header.Height
	s.EqualError(blockchain.CheckDPOSIllegalBlocks(illegalBlocks),
		"block header height should be same")

	header2.Height = header.Height
	buf = new(bytes.Buffer)
	header2.Serialize(buf)
	cmpEvidence.Header = buf.Bytes()
	asc = common.BytesToHexString(evidence.Header) <
		common.BytesToHexString(cmpEvidence.Header)
	if asc {
		illegalBlocks.Evidence = *evidence
		illegalBlocks.CompareEvidence = *cmpEvidence
	} else {
		illegalBlocks.Evidence = *cmpEvidence
		illegalBlocks.CompareEvidence = *evidence
	}
	s.EqualError(blockchain.CheckDPOSIllegalBlocks(illegalBlocks),
		"EOF")

	// fill confirms of evidences
	confirm := &payload.Confirm{
		Proposal: payload.DPOSProposal{
			Sponsor:    s.arbitrators.CurrentArbitrators[0].GetNodePublicKey(),
			BlockHash:  header.Hash(),
			ViewOffset: rand.Uint32(),
		},
		Votes: []payload.DPOSProposalVote{},
	}
	cmpConfirm := &payload.Confirm{
		Proposal: payload.DPOSProposal{
			Sponsor:    s.arbitrators.CurrentArbitrators[0].GetNodePublicKey(),
			BlockHash:  header2.Hash(),
			ViewOffset: rand.Uint32(),
		},
		Votes: []payload.DPOSProposalVote{},
	}

	confirm.Proposal.Sign, _ = crypto.Sign(s.arbitratorsPriKeys[0],
		confirm.Proposal.Data())
	cmpConfirm.Proposal.Sign, _ = crypto.Sign(s.arbitratorsPriKeys[0],
		cmpConfirm.Proposal.Data())
	s.updateIllegaBlocks(confirm, evidence, cmpConfirm, cmpEvidence, asc,
		illegalBlocks)
	s.EqualError(blockchain.CheckDPOSIllegalBlocks(illegalBlocks),
		"[IllegalConfirmContextCheck] signers less than majority count")

	// fill votes of confirms
	for i := 0; i < 4; i++ {
		vote := payload.DPOSProposalVote{
			ProposalHash: confirm.Proposal.Hash(),
			Signer:       s.arbitrators.CurrentArbitrators[i].GetNodePublicKey(),
			Accept:       true,
		}
		vote.Sign, _ = crypto.Sign(s.arbitratorsPriKeys[i], vote.Data())
		confirm.Votes = append(confirm.Votes, vote)
	}
	for i := 1; i < 5; i++ {
		vote := payload.DPOSProposalVote{
			ProposalHash: cmpConfirm.Proposal.Hash(),
			Signer:       s.arbitrators.CurrentArbitrators[i].GetNodePublicKey(),
			Accept:       true,
		}
		vote.Sign, _ = crypto.Sign(s.arbitratorsPriKeys[i], vote.Data())
		cmpConfirm.Votes = append(cmpConfirm.Votes, vote)
	}
	s.updateIllegaBlocks(confirm, evidence, cmpConfirm, cmpEvidence, asc,
		illegalBlocks)
	s.EqualError(blockchain.CheckDPOSIllegalBlocks(illegalBlocks),
		"confirm view offset should be same")

	// correct view offset
	proposal := payload.DPOSProposal{
		Sponsor:    s.arbitrators.CurrentArbitrators[0].GetNodePublicKey(),
		BlockHash:  *randomUint256(),
		ViewOffset: confirm.Proposal.ViewOffset,
	}
	proposal.Sign, _ = crypto.Sign(s.arbitratorsPriKeys[0], proposal.Data())
	cmpConfirm.Proposal = proposal
	cmpConfirm.Votes = make([]payload.DPOSProposalVote, 0)
	for i := 1; i < 5; i++ {
		vote := payload.DPOSProposalVote{
			ProposalHash: cmpConfirm.Proposal.Hash(),
			Signer:       s.arbitrators.CurrentArbitrators[i].GetNodePublicKey(),
			Accept:       true,
		}
		vote.Sign, _ = crypto.Sign(s.arbitratorsPriKeys[i], vote.Data())
		cmpConfirm.Votes = append(cmpConfirm.Votes, vote)
	}
	s.updateIllegaBlocks(confirm, evidence, cmpConfirm, cmpEvidence, asc,
		illegalBlocks)
	s.EqualError(blockchain.CheckDPOSIllegalBlocks(illegalBlocks),
		"block and related confirm do not match")

	// correct block hash corresponding to header hash
	proposal = payload.DPOSProposal{
		Sponsor:    s.arbitrators.CurrentArbitrators[0].GetNodePublicKey(),
		BlockHash:  header2.Hash(),
		ViewOffset: confirm.Proposal.ViewOffset,
	}
	proposal.Sign, _ = crypto.Sign(s.arbitratorsPriKeys[0], proposal.Data())
	cmpConfirm.Proposal = proposal
	cmpConfirm.Votes = make([]payload.DPOSProposalVote, 0)
	for i := 1; i < 5; i++ {
		vote := payload.DPOSProposalVote{
			ProposalHash: cmpConfirm.Proposal.Hash(),
			Signer:       s.arbitrators.CurrentArbitrators[i].GetNodePublicKey(),
			Accept:       true,
		}
		vote.Sign, _ = crypto.Sign(s.arbitratorsPriKeys[i], vote.Data())
		cmpConfirm.Votes = append(cmpConfirm.Votes, vote)
	}
	s.updateIllegaBlocks(confirm, evidence, cmpConfirm, cmpEvidence, asc,
		illegalBlocks)
	s.EqualError(blockchain.CheckDPOSIllegalBlocks(illegalBlocks),
		"signers count it not match the count of confirm votes")

	// fill the same signers to evidences
	for _, v := range confirm.Votes {
		evidence.Signers = append(evidence.Signers, v.Signer)
		cmpEvidence.Signers = append(cmpEvidence.Signers, v.Signer)
	}
	s.updateIllegaBlocks(confirm, evidence, cmpConfirm, cmpEvidence, asc,
		illegalBlocks)
	s.EqualError(blockchain.CheckDPOSIllegalBlocks(illegalBlocks),
		"signers and confirm votes do not match")

	// correct signers of compare evidence
	signers := make([][]byte, 0)
	for _, v := range cmpConfirm.Votes {
		signers = append(signers, v.Signer)
	}
	cmpEvidence.Signers = signers
	s.updateIllegaBlocks(confirm, evidence, cmpConfirm, cmpEvidence, asc,
		illegalBlocks)
	s.NoError(blockchain.CheckDPOSIllegalBlocks(illegalBlocks))
}

func (s *txValidatorSpecialTxTestSuite) updateIllegaBlocks(
	confirm *payload.Confirm, evidence *payload.BlockEvidence,
	cmpConfirm *payload.Confirm, cmpEvidence *payload.BlockEvidence,
	asc bool, illegalBlocks *payload.DPOSIllegalBlocks) {
	buf := new(bytes.Buffer)
	confirm.Serialize(buf)
	evidence.BlockConfirm = buf.Bytes()
	buf = new(bytes.Buffer)
	cmpConfirm.Serialize(buf)
	cmpEvidence.BlockConfirm = buf.Bytes()
	if asc {
		illegalBlocks.Evidence = *evidence
		illegalBlocks.CompareEvidence = *cmpEvidence
	} else {
		illegalBlocks.Evidence = *cmpEvidence
		illegalBlocks.CompareEvidence = *evidence
	}
}
