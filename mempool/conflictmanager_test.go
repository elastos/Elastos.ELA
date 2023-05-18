// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package mempool

import (
	"crypto/rand"
	mrand "math/rand"
	"testing"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	transaction2 "github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
	"github.com/stretchr/testify/assert"
)

func init() {
	functions.GetTransactionByTxType = transaction2.GetTransaction
	functions.GetTransactionByBytes = transaction2.GetTransactionByBytes
	functions.CreateTransaction = transaction2.CreateTransaction
	functions.GetTransactionParameters = transaction2.GetTransactionparameters
}

func TestConflictManager_DPoS_OwnerPublicKey(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		pk := randomPublicKey()

		tx1 := functions.CreateTransaction(
			0,
			common2.RegisterProducer,
			0,
			&payload.ProducerInfo{
				OwnerKey:      pk,
				NodePublicKey: randomPublicKey(),
				NickName:      randomNickname(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		tx2 := functions.CreateTransaction(
			0,
			common2.UpdateProducer,
			0,
			&payload.ProducerInfo{
				OwnerKey:      pk,
				NodePublicKey: randomPublicKey(),
				NickName:      randomNickname(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		tx4 := functions.CreateTransaction(
			0,
			common2.RegisterCR,
			0,
			&payload.CRInfo{
				Code: redeemScriptFromPk(pk),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		txs := []interfaces.Transaction{tx1, tx2, tx4} //tx3,
		verifyTxListWithConflictManager(txs, db, true, t)
	})

	conflictTestProc(func(db *UtxoCacheDB) {
		currentHeight := uint32(1)
		dposState := blockchain.DefaultLedger.Blockchain.GetState()
		ownerPK1 := randomPublicKey()
		NodePublicKey1 := randomPublicKey()
		ownerPK2 := randomPublicKey()
		NodePublicKey2 := randomPublicKey()
		regProTX1 := functions.CreateTransaction(
			0,
			common2.RegisterProducer,
			0,
			&payload.ProducerInfo{
				OwnerKey:      ownerPK1,
				NodePublicKey: NodePublicKey1,
				NickName:      randomNickname(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		regProTX2 := functions.CreateTransaction(
			0,
			common2.RegisterProducer,
			0,
			&payload.ProducerInfo{
				OwnerKey:      ownerPK2,
				NodePublicKey: NodePublicKey2,
				NickName:      randomNickname(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		block := &types.Block{
			Transactions: []interfaces.Transaction{
				regProTX1,
				regProTX2,
			},
			Header: common2.Header{Height: currentHeight},
		}
		dposState.ProcessBlock(block, nil, 0)
		currentHeight++

		CancelProTX1 := functions.CreateTransaction(
			0,
			common2.CancelProducer,
			0,
			&payload.ProcessProducer{
				OwnerKey: ownerPK1,
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		CancelProTX2 := functions.CreateTransaction(
			0,
			common2.CancelProducer,
			0,
			&payload.ProcessProducer{
				OwnerKey: ownerPK2,
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		activProTX1 := functions.CreateTransaction(
			0,
			common2.ActivateProducer,
			0,
			&payload.ActivateProducer{
				NodePublicKey: NodePublicKey1,
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		activProTX2 := functions.CreateTransaction(
			0,
			common2.ActivateProducer,
			0,
			&payload.ActivateProducer{
				NodePublicKey: NodePublicKey2,
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		//register two different producer at the same time
		{
			manager := newConflictManager()
			assert.NoError(t, manager.VerifyTx(regProTX1))
			assert.NoError(t, manager.AppendTx(regProTX1))
			assert.NoError(t, manager.VerifyTx(regProTX2))
			assert.NoError(t, manager.AppendTx(regProTX2))
		}
		//activate two different producers at the same time
		{
			manager := newConflictManager()
			assert.NoError(t, manager.VerifyTx(activProTX1))
			assert.NoError(t, manager.AppendTx(activProTX1))
			assert.NoError(t, manager.VerifyTx(activProTX2))
			assert.NoError(t, manager.AppendTx(activProTX2))
		}
		//activate two different producers at the same time
		{
			manager := newConflictManager()
			assert.NoError(t, manager.VerifyTx(activProTX1))
			assert.NoError(t, manager.AppendTx(activProTX1))
			assert.NoError(t, manager.VerifyTx(activProTX2))
			assert.NoError(t, manager.AppendTx(activProTX2))
		}

		//cancel two different producers at the same time
		{
			manager := newConflictManager()
			assert.NoError(t, manager.VerifyTx(CancelProTX1))
			assert.NoError(t, manager.AppendTx(CancelProTX1))
			assert.NoError(t, manager.VerifyTx(CancelProTX2))
			assert.NoError(t, manager.AppendTx(CancelProTX2))
		}

		//active and cancel one producer at the same time  must report error
		{
			manager := newConflictManager()
			assert.NoError(t, manager.VerifyTx(CancelProTX1))
			assert.NoError(t, manager.AppendTx(CancelProTX1))
			//assert.NoError(t, manager.VerifyTx(activProTX1))
			err := manager.VerifyTx(activProTX1)
			assert.Error(t, err, "slot DPoSActivateCancel verify tx error")
			//no need append
			//assert.NoError(t, manager.AppendTx(activProTX1))
		}

		//active and cancel diffrent producers at the same time
		{
			manager := newConflictManager()
			assert.NoError(t, manager.VerifyTx(CancelProTX1))
			assert.NoError(t, manager.AppendTx(CancelProTX1))
			assert.NoError(t, manager.VerifyTx(activProTX2))
			assert.NoError(t, manager.AppendTx(activProTX2))
		}

	})
}

func TestConflictManager_DPoS_NodePublicKey(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		pk := randomPublicKey()

		tx1 := functions.CreateTransaction(
			0,
			common2.RegisterProducer,
			0,
			&payload.ProducerInfo{
				OwnerKey:      randomPublicKey(),
				NodePublicKey: pk,
				NickName:      randomNickname(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		tx2 := functions.CreateTransaction(
			0,
			common2.UpdateProducer,
			0,
			&payload.ProducerInfo{
				OwnerKey:      randomPublicKey(),
				NodePublicKey: pk,
				NickName:      randomNickname(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		tx3 := functions.CreateTransaction(
			0,
			common2.ActivateProducer,
			0,
			&payload.ActivateProducer{
				NodePublicKey: pk,
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		tx4 := functions.CreateTransaction(
			0,
			common2.RegisterCR,
			0,
			&payload.CRInfo{
				Code: redeemScriptFromPk(pk),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		txs := []interfaces.Transaction{tx1, tx2, tx3, tx4}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_DPoS_Nickname(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		name := randomNickname()
		tx1 := functions.CreateTransaction(
			0,
			common2.RegisterProducer,
			0,
			&payload.ProducerInfo{
				OwnerKey:      randomPublicKey(),
				NodePublicKey: randomPublicKey(),
				NickName:      name,
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		tx2 := functions.CreateTransaction(
			0,
			common2.UpdateProducer,
			0,
			&payload.ProducerInfo{
				OwnerKey:      randomPublicKey(),
				NodePublicKey: randomPublicKey(),
				NickName:      name,
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		txs := []interfaces.Transaction{tx1, tx2}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_CR_DID(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		cid := *randomProgramHash()

		tx1 := functions.CreateTransaction(
			0,
			common2.RegisterCR,
			0,
			&payload.CRInfo{
				CID:      cid,
				Code:     redeemScriptFromPk(randomPublicKey()),
				NickName: randomNickname(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		tx2 := functions.CreateTransaction(
			0,
			common2.UpdateCR,
			0,
			&payload.CRInfo{
				CID:      cid,
				Code:     redeemScriptFromPk(randomPublicKey()),
				NickName: randomNickname(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		tx3 := functions.CreateTransaction(
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

		txs := []interfaces.Transaction{tx1, tx2, tx3}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_CR_Nickname(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		name := randomNickname()

		tx1 := functions.CreateTransaction(
			0,
			common2.RegisterCR,
			0,
			&payload.CRInfo{
				DID:      *randomProgramHash(),
				Code:     redeemScriptFromPk(randomPublicKey()),
				NickName: name,
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		tx2 := functions.CreateTransaction(
			0,
			common2.UpdateCR,
			0,
			&payload.CRInfo{
				DID:      *randomProgramHash(),
				Code:     redeemScriptFromPk(randomPublicKey()),
				NickName: name,
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		txs := []interfaces.Transaction{tx1, tx2}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_ProgramCode(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		code := redeemScriptFromPk(randomPublicKey())

		tx1 := functions.CreateTransaction(
			0,
			common2.ReturnDepositCoin,
			0,
			&payload.ReturnDepositCoin{},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{
				{
					Code: code,
				},
			},
		)

		tx2 := functions.CreateTransaction(
			0,
			common2.ReturnCRDepositCoin,
			0,
			&payload.ReturnDepositCoin{},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{
				{
					Code: code,
				},
			},
		)

		txs := []interfaces.Transaction{tx1, tx2}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_CR_DraftHash(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		hash := *randomHash()
		tx1 := functions.CreateTransaction(
			0,
			common2.CRCProposal,
			0,
			&payload.CRCProposal{
				DraftHash:          hash,
				CRCouncilMemberDID: *randomProgramHash(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		tx2 := functions.CreateTransaction(
			0,
			common2.CRCProposal,
			0,
			&payload.CRCProposal{
				DraftHash:          hash,
				CRCouncilMemberDID: *randomProgramHash(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		txs := []interfaces.Transaction{tx1, tx2}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_CR_SponsorDID(t *testing.T) {
	did := *randomProgramHash()
	conflictTestProc(func(db *UtxoCacheDB) {
		tx1 := functions.CreateTransaction(
			0,
			common2.CRCProposal,
			0,
			&payload.CRCProposal{
				DraftHash:          *randomHash(),
				CRCouncilMemberDID: did,
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		tx2 := functions.CreateTransaction(
			0,
			common2.CRCProposal,
			0,
			&payload.CRCProposal{
				DraftHash:          *randomHash(),
				CRCouncilMemberDID: did,
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		txs := []interfaces.Transaction{tx1, tx2}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_CR_ProposalHash(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		hash := *randomHash()
		tx1 := functions.CreateTransaction(
			0,
			common2.CRCProposalWithdraw,
			0,
			&payload.CRCProposalWithdraw{
				ProposalHash: hash,
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		txs := []interfaces.Transaction{tx1}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_CR_ProposalTrackHash(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		hash := *randomHash()
		tx := functions.CreateTransaction(
			0,
			common2.CRCProposalTracking,
			0,
			&payload.CRCProposalTracking{
				ProposalHash: hash,
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		txs := []interfaces.Transaction{tx}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_CR_ProposalReviewKey(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		hash := *randomHash()
		did := *randomProgramHash()

		tx1 := functions.CreateTransaction(
			0,
			common2.CRCProposalReview,
			0,
			&payload.CRCProposalReview{
				ProposalHash: hash,
				DID:          did,
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		txs := []interfaces.Transaction{tx1}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_CR_AppropriationKey(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		tx1 := functions.CreateTransaction(
			0,
			common2.CRCAppropriation,
			0,
			&payload.CRCAppropriation{},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		txs := []interfaces.Transaction{tx1}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_SpecialTxHashes(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		tx1 := functions.CreateTransaction(
			0,
			common2.IllegalProposalEvidence,
			0,
			&payload.DPOSIllegalProposals{
				Evidence: payload.ProposalEvidence{
					BlockHeader: randomHash().Bytes(),
				},
				CompareEvidence: payload.ProposalEvidence{
					BlockHeader: randomHash().Bytes(),
				},
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		txs := []interfaces.Transaction{tx1}

		verifyTxListWithConflictManager(txs, db, true, t)
	})

	conflictTestProc(func(db *UtxoCacheDB) {

		tx1 := functions.CreateTransaction(
			0,
			common2.IllegalVoteEvidence,
			0,
			&payload.DPOSIllegalVotes{
				Evidence: payload.VoteEvidence{
					ProposalEvidence: payload.ProposalEvidence{
						BlockHeader: randomHash().Bytes(),
					},
				},
				CompareEvidence: payload.VoteEvidence{
					ProposalEvidence: payload.ProposalEvidence{
						BlockHeader: randomHash().Bytes(),
					},
				},
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		txs := []interfaces.Transaction{tx1}

		verifyTxListWithConflictManager(txs, db, true, t)
	})

	conflictTestProc(func(db *UtxoCacheDB) {
		tx := functions.CreateTransaction(
			0,
			common2.IllegalBlockEvidence,
			0,
			&payload.DPOSIllegalBlocks{
				Evidence: payload.BlockEvidence{
					Header: randomHash().Bytes(),
				},
				CompareEvidence: payload.BlockEvidence{
					Header: randomHash().Bytes(),
				},
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		txs := []interfaces.Transaction{tx}

		verifyTxListWithConflictManager(txs, db, true, t)
	})

	conflictTestProc(func(db *UtxoCacheDB) {
		tx := functions.CreateTransaction(
			0,
			common2.IllegalSidechainEvidence,
			0,
			&payload.SidechainIllegalData{
				Evidence: payload.SidechainIllegalEvidence{
					DataHash: *randomHash(),
				},
				CompareEvidence: payload.SidechainIllegalEvidence{
					DataHash: *randomHash(),
				},
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		txs := []interfaces.Transaction{tx}

		verifyTxListWithConflictManager(txs, db, true, t)
	})

	conflictTestProc(func(db *UtxoCacheDB) {
		tx := functions.CreateTransaction(
			0,
			common2.InactiveArbitrators,
			0,
			&payload.InactiveArbitrators{
				Arbitrators: [][]byte{
					randomPublicKey(),
					randomPublicKey(),
					randomPublicKey(),
				},
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		txs := []interfaces.Transaction{tx}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_Sidechain_TxHashes(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		hash := *randomHash()

		tx1 := functions.CreateTransaction(
			0,
			common2.WithdrawFromSideChain,
			0,
			&payload.WithdrawFromSideChain{
				SideChainTransactionHashes: []common.Uint256{
					hash,
					*randomHash(),
					*randomHash(),
				},
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		tx2 := functions.CreateTransaction(
			0,
			common2.WithdrawFromSideChain,
			0,
			&payload.WithdrawFromSideChain{
				SideChainTransactionHashes: []common.Uint256{
					hash,
					*randomHash(),
					*randomHash(),
				},
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		txs := []interfaces.Transaction{tx1, tx2}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_InputInferKeys(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		tx1 := functions.CreateTransaction(
			0,
			common2.RegisterProducer,
			0,
			&payload.ProducerInfo{
				OwnerKey:      randomPublicKey(),
				NodePublicKey: randomPublicKey(),
				NickName:      randomNickname(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		tx2 := functions.CreateTransaction(
			0,
			common2.UpdateProducer,
			0,
			&payload.ProducerInfo{
				OwnerKey:      randomPublicKey(),
				NodePublicKey: randomPublicKey(),
				NickName:      randomNickname(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		tx4 := functions.CreateTransaction(
			0,
			common2.RegisterCR,
			0,
			&payload.CRInfo{
				DID:      *randomProgramHash(),
				Code:     redeemScriptFromPk(randomPublicKey()),
				NickName: randomNickname(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		tx5 := functions.CreateTransaction(
			0,
			common2.UpdateCR,
			0,
			&payload.CRInfo{
				DID:      *randomProgramHash(),
				Code:     redeemScriptFromPk(randomPublicKey()),
				NickName: randomNickname(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		tx6 := functions.CreateTransaction(
			0,
			common2.UnregisterCR,
			0,
			&payload.UnregisterCR{
				CID: *randomProgramHash(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		tx7 := functions.CreateTransaction(
			0,
			common2.ReturnDepositCoin,
			0,
			&payload.ReturnDepositCoin{},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{
				{
					Code: redeemScriptFromPk(randomPublicKey()),
				},
			},
		)
		tx8 := functions.CreateTransaction(
			0,
			common2.ReturnCRDepositCoin,
			0,
			&payload.ReturnDepositCoin{},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{
				{
					Code: redeemScriptFromPk(randomPublicKey()),
				},
			},
		)

		tx9 := functions.CreateTransaction(
			0,
			common2.CRCProposal,
			0,
			&payload.CRCProposal{
				DraftHash: *randomHash(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		tx10 := functions.CreateTransaction(
			0,
			common2.CRCProposalWithdraw,
			0,
			&payload.CRCProposalWithdraw{
				ProposalHash: *randomHash(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		tx11 := functions.CreateTransaction(
			0,
			common2.CRCProposalTracking,
			0,
			&payload.CRCProposalTracking{
				ProposalHash: *randomHash(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)

		tx12 := functions.CreateTransaction(
			0,
			common2.CRCProposalReview,
			0,
			&payload.CRCProposalReview{
				ProposalHash: *randomHash(),
				DID:          *randomProgramHash(),
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		tx13 := functions.CreateTransaction(
			0,
			common2.CRCAppropriation,
			0,
			&payload.CRCAppropriation{},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		tx14 := functions.CreateTransaction(
			0,
			common2.WithdrawFromSideChain,
			0,
			&payload.WithdrawFromSideChain{
				SideChainTransactionHashes: []common.Uint256{
					*randomHash(),
					*randomHash(),
				},
			},
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			[]*program.Program{},
		)
		txs := []interfaces.Transaction{tx1, tx2, tx4, tx5, tx6, tx7, tx8, tx9, tx10, tx11, tx12, tx13, tx14}

		verifyTxListWithConflictManager(txs, db, false, t)
	})
}

func conflictTestProc(action func(*UtxoCacheDB)) {
	origin := blockchain.DefaultLedger
	utxoCacheDB := NewUtxoCacheDB()
	blockchain.DefaultLedger = &blockchain.Ledger{
		Blockchain: &blockchain.BlockChain{
			UTXOCache: blockchain.NewUTXOCache(utxoCacheDB, &config.DefaultParams),
		},
	}

	blockchain.DefaultLedger.Blockchain.SetState(state.NewState(&config.DefaultParams, nil, nil, nil,
		func() bool { return false }, func(programHash common.Uint168) (common.Fixed64,
			error) {
			amount := common.Fixed64(0)
			utxos, err := blockchain.DefaultLedger.Blockchain.GetDB().GetFFLDB().GetUTXO(&programHash)
			if err != nil {
				return amount, err
			}
			for _, utxo := range utxos {
				amount += utxo.Value
			}
			return amount, nil
		}, nil, nil, nil, nil, nil, nil))
	action(utxoCacheDB)
	blockchain.DefaultLedger = origin
}

func setPreviousTransactionIndividually(txs []interfaces.Transaction,
	utxoCacheDB *UtxoCacheDB) {
	for _, tx := range txs {
		prevTx := newPreviousTx(utxoCacheDB)
		tx.SetInputs([]*common2.Input{
			{
				Previous: common2.OutPoint{
					TxID:  prevTx.Hash(),
					Index: 0,
				},
				Sequence: 100,
			},
		})
	}
}

func setSamePreviousTransaction(txs []interfaces.Transaction,
	utxoCacheDB *UtxoCacheDB) {
	prevTx := newPreviousTx(utxoCacheDB)
	for _, tx := range txs {
		tx.SetInputs([]*common2.Input{
			{
				Previous: common2.OutPoint{
					TxID:  prevTx.Hash(),
					Index: 0,
				},
				Sequence: 100,
			},
		})
	}
}

func newPreviousTx(utxoCacheDB *UtxoCacheDB) interfaces.Transaction {
	prevTx := functions.CreateTransaction(
		0,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				Value:       common.Fixed64(mrand.Int63()),
				ProgramHash: *randomProgramHash(),
			},
		},
		0,
		[]*program.Program{},
	)
	utxoCacheDB.PutTransaction(prevTx)
	return prevTx
}

func verifyTxListWithConflictManager(txs []interfaces.Transaction,
	utxoCacheDB *UtxoCacheDB, individualPreTx bool, t *testing.T) {
	if individualPreTx {
		setPreviousTransactionIndividually(txs, utxoCacheDB)
	} else {
		setSamePreviousTransaction(txs, utxoCacheDB)
	}

	manager := newConflictManager()
	for _, addedTx := range txs {
		assert.NoError(t, manager.VerifyTx(addedTx))
		assert.NoError(t, manager.AppendTx(addedTx))
		for _, candidate := range txs {
			assert.True(t, manager.VerifyTx(candidate) != nil)
		}

		assert.NoError(t, manager.removeTx(addedTx))
		assert.True(t, manager.Empty())
		for _, candidate := range txs {
			assert.NoError(t, manager.VerifyTx(candidate))
		}
	}
}

func randomHash() *common.Uint256 {
	a := make([]byte, 32)
	rand.Read(a)
	hash, _ := common.Uint256FromBytes(a)
	return hash
}

func redeemScriptFromPk(pk []byte) []byte {
	pubKey, _ := crypto.DecodePoint(pk)
	rtn, _ := contract.CreateStandardRedeemScript(pubKey)
	return rtn
}

func randomPublicKey() []byte {
	_, pub, _ := crypto.GenerateKeyPair()
	result, _ := pub.EncodePoint(true)
	return result
}

func randomNickname() string {
	var name [20]byte
	rand.Read(name[:])
	return string(name[:])
}

func randomProgramHash() *common.Uint168 {
	a := make([]byte, 21)
	rand.Read(a)
	hash, _ := common.Uint168FromBytes(a)
	return hash
}
