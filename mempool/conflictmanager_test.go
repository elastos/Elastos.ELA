// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package mempool

import (
	"crypto/rand"
	"github.com/elastos/Elastos.ELA/core/types/transactions"
	mrand "math/rand"
	"testing"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/stretchr/testify/assert"
)

func TestConflictManager_DPoS_OwnerPublicKey(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		pk := randomPublicKey()
		txs := []interfaces.Transaction{
			{
				TxType: common2.RegisterProducer,
				Payload: &payload.ProducerInfo{
					OwnerPublicKey: pk,
					NodePublicKey:  randomPublicKey(),
					NickName:       randomNickname(),
				},
			},
			{
				TxType: common2.UpdateProducer,
				Payload: &payload.ProducerInfo{
					OwnerPublicKey: pk,
					NodePublicKey:  randomPublicKey(),
					NickName:       randomNickname(),
				},
			},
			{
				TxType: common2.CancelProducer,
				Payload: &payload.ProcessProducer{
					OwnerPublicKey: pk,
				},
			},
			{
				TxType: common2.RegisterCR,
				Payload: &payload.CRInfo{
					Code: redeemScriptFromPk(pk),
				},
			},
		}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_DPoS_NodePublicKey(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		pk := randomPublicKey()
		txs := []interfaces.Transaction{
			{
				TxType: common2.RegisterProducer,
				Payload: &payload.ProducerInfo{
					OwnerPublicKey: randomPublicKey(),
					NodePublicKey:  pk,
					NickName:       randomNickname(),
				},
			},
			{
				TxType: common2.UpdateProducer,
				Payload: &payload.ProducerInfo{
					OwnerPublicKey: randomPublicKey(),
					NodePublicKey:  pk,
					NickName:       randomNickname(),
				},
			},
			{
				TxType: common2.ActivateProducer,
				Payload: &payload.ActivateProducer{
					NodePublicKey: pk,
				},
			},
			{
				TxType: common2.RegisterCR,
				Payload: &payload.CRInfo{
					Code: redeemScriptFromPk(pk),
				},
			},
		}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_DPoS_Nickname(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		name := randomNickname()
		txs := []interfaces.Transaction{
			{
				TxType: common2.RegisterProducer,
				Payload: &payload.ProducerInfo{
					OwnerPublicKey: randomPublicKey(),
					NodePublicKey:  randomPublicKey(),
					NickName:       name,
				},
			},
			{
				TxType: common2.UpdateProducer,
				Payload: &payload.ProducerInfo{
					OwnerPublicKey: randomPublicKey(),
					NodePublicKey:  randomPublicKey(),
					NickName:       name,
				},
			},
		}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_CR_DID(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		cid := *randomProgramHash()
		txs := []interfaces.Transaction{
			{
				TxType: common2.RegisterCR,
				Payload: &payload.CRInfo{
					CID:      cid,
					Code:     redeemScriptFromPk(randomPublicKey()),
					NickName: randomNickname(),
				},
			},
			{
				TxType: common2.UpdateCR,
				Payload: &payload.CRInfo{
					CID:      cid,
					Code:     redeemScriptFromPk(randomPublicKey()),
					NickName: randomNickname(),
				},
			},
			{
				TxType: common2.UnregisterCR,
				Payload: &payload.UnregisterCR{
					CID: cid,
				},
			},
		}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_CR_Nickname(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		name := randomNickname()
		txs := []interfaces.Transaction{
			{
				TxType: common2.RegisterCR,
				Payload: &payload.CRInfo{
					DID:      *randomProgramHash(),
					Code:     redeemScriptFromPk(randomPublicKey()),
					NickName: name,
				},
			},
			{
				TxType: common2.UpdateCR,
				Payload: &payload.CRInfo{
					DID:      *randomProgramHash(),
					Code:     redeemScriptFromPk(randomPublicKey()),
					NickName: name,
				},
			},
		}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_ProgramCode(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		code := redeemScriptFromPk(randomPublicKey())
		txs := []interfaces.Transaction{
			{
				TxType:  common2.ReturnDepositCoin,
				Payload: &payload.ReturnDepositCoin{},
				Programs: []*program.Program{
					{
						Code: code,
					},
				},
			},
			{
				TxType:  common2.ReturnCRDepositCoin,
				Payload: &payload.ReturnDepositCoin{},
				Programs: []*program.Program{
					{
						Code: code,
					},
				},
			},
		}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_CR_DraftHash(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		hash := *randomHash()
		txs := []interfaces.Transaction{
			{
				TxType: common2.CRCProposal,
				Payload: &payload.CRCProposal{
					DraftHash:          hash,
					CRCouncilMemberDID: *randomProgramHash(),
				},
			},
			{
				TxType: common2.CRCProposal,
				Payload: &payload.CRCProposal{
					DraftHash:          hash,
					CRCouncilMemberDID: *randomProgramHash(),
				},
			},
		}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_CR_SponsorDID(t *testing.T) {
	did := *randomProgramHash()
	conflictTestProc(func(db *UtxoCacheDB) {
		txs := []interfaces.Transaction{
			{
				TxType: common2.CRCProposal,
				Payload: &payload.CRCProposal{
					DraftHash:          *randomHash(),
					CRCouncilMemberDID: did,
				},
			},
			{
				TxType: common2.CRCProposal,
				Payload: &payload.CRCProposal{
					DraftHash:          *randomHash(),
					CRCouncilMemberDID: did,
				},
			},
		}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_CR_ProposalHash(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		hash := *randomHash()
		txs := []interfaces.Transaction{
			{
				TxType: common2.CRCProposalWithdraw,
				Payload: &payload.CRCProposalWithdraw{
					ProposalHash: hash,
				},
			},
		}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_CR_ProposalTrackHash(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		hash := *randomHash()
		txs := []interfaces.Transaction{
			{
				TxType: common2.CRCProposalTracking,
				Payload: &payload.CRCProposalTracking{
					ProposalHash: hash,
				},
			},
		}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_CR_ProposalReviewKey(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		hash := *randomHash()
		did := *randomProgramHash()
		txs := []interfaces.Transaction{
			{
				TxType: common2.CRCProposalReview,
				Payload: &payload.CRCProposalReview{
					ProposalHash: hash,
					DID:          did,
				},
			},
		}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_CR_AppropriationKey(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		txs := []interfaces.Transaction{
			{
				TxType:  common2.CRCAppropriation,
				Payload: &payload.CRCAppropriation{},
			},
		}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_SpecialTxHashes(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		txs := []interfaces.Transaction{
			{
				TxType: common2.IllegalProposalEvidence,
				Payload: &payload.DPOSIllegalProposals{
					Evidence: payload.ProposalEvidence{
						BlockHeader: randomHash().Bytes(),
					},
					CompareEvidence: payload.ProposalEvidence{
						BlockHeader: randomHash().Bytes(),
					},
				},
			},
		}

		verifyTxListWithConflictManager(txs, db, true, t)
	})

	conflictTestProc(func(db *UtxoCacheDB) {
		txs := []interfaces.Transaction{
			{
				TxType: common2.IllegalVoteEvidence,
				Payload: &payload.DPOSIllegalVotes{
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
			},
		}

		verifyTxListWithConflictManager(txs, db, true, t)
	})

	conflictTestProc(func(db *UtxoCacheDB) {
		txs := []interfaces.Transaction{
			{
				TxType: common2.IllegalBlockEvidence,
				Payload: &payload.DPOSIllegalBlocks{
					Evidence: payload.BlockEvidence{
						Header: randomHash().Bytes(),
					},
					CompareEvidence: payload.BlockEvidence{
						Header: randomHash().Bytes(),
					},
				},
			},
		}

		verifyTxListWithConflictManager(txs, db, true, t)
	})

	conflictTestProc(func(db *UtxoCacheDB) {
		txs := []interfaces.Transaction{
			{
				TxType: common2.IllegalSidechainEvidence,
				Payload: &payload.SidechainIllegalData{
					Evidence: payload.SidechainIllegalEvidence{
						DataHash: *randomHash(),
					},
					CompareEvidence: payload.SidechainIllegalEvidence{
						DataHash: *randomHash(),
					},
				},
			},
		}

		verifyTxListWithConflictManager(txs, db, true, t)
	})

	conflictTestProc(func(db *UtxoCacheDB) {
		txs := []interfaces.Transaction{
			{
				TxType: common2.InactiveArbitrators,
				Payload: &payload.InactiveArbitrators{
					Arbitrators: [][]byte{
						randomPublicKey(),
						randomPublicKey(),
						randomPublicKey(),
					},
				},
			},
		}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_Sidechain_TxHashes(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		hash := *randomHash()
		txs := []interfaces.Transaction{
			{
				TxType: common2.WithdrawFromSideChain,
				Payload: &payload.WithdrawFromSideChain{
					SideChainTransactionHashes: []common.Uint256{
						hash,
						*randomHash(),
						*randomHash(),
					},
				},
			},
			{
				TxType: common2.WithdrawFromSideChain,
				Payload: &payload.WithdrawFromSideChain{
					SideChainTransactionHashes: []common.Uint256{
						hash,
						*randomHash(),
						*randomHash(),
					},
				},
			},
		}

		verifyTxListWithConflictManager(txs, db, true, t)
	})
}

func TestConflictManager_InputInferKeys(t *testing.T) {
	conflictTestProc(func(db *UtxoCacheDB) {
		txs := []interfaces.Transaction{
			{
				TxType: common2.RegisterProducer,
				Payload: &payload.ProducerInfo{
					OwnerPublicKey: randomPublicKey(),
					NodePublicKey:  randomPublicKey(),
					NickName:       randomNickname(),
				},
			},
			{
				TxType: common2.UpdateProducer,
				Payload: &payload.ProducerInfo{
					OwnerPublicKey: randomPublicKey(),
					NodePublicKey:  randomPublicKey(),
					NickName:       randomNickname(),
				},
			},
			{
				TxType: common2.CancelProducer,
				Payload: &payload.ProcessProducer{
					OwnerPublicKey: randomPublicKey(),
				},
			},
			{
				TxType: common2.RegisterCR,
				Payload: &payload.CRInfo{
					DID:      *randomProgramHash(),
					Code:     redeemScriptFromPk(randomPublicKey()),
					NickName: randomNickname(),
				},
			},
			{
				TxType: common2.UpdateCR,
				Payload: &payload.CRInfo{
					DID:      *randomProgramHash(),
					Code:     redeemScriptFromPk(randomPublicKey()),
					NickName: randomNickname(),
				},
			},
			{
				TxType: common2.UnregisterCR,
				Payload: &payload.UnregisterCR{
					CID: *randomProgramHash(),
				},
			},
			{
				TxType:  common2.ReturnDepositCoin,
				Payload: &payload.ReturnDepositCoin{},
				Programs: []*program.Program{
					{
						Code: redeemScriptFromPk(randomPublicKey()),
					},
				},
			},
			{
				TxType:  common2.ReturnCRDepositCoin,
				Payload: &payload.ReturnDepositCoin{},
				Programs: []*program.Program{
					{
						Code: redeemScriptFromPk(randomPublicKey()),
					},
				},
			},
			{
				TxType: common2.CRCProposal,
				Payload: &payload.CRCProposal{
					DraftHash: *randomHash(),
				},
			},
			{
				TxType: common2.CRCProposalWithdraw,
				Payload: &payload.CRCProposalWithdraw{
					ProposalHash: *randomHash(),
				},
			},
			{
				TxType: common2.CRCProposalTracking,
				Payload: &payload.CRCProposalTracking{
					ProposalHash: *randomHash(),
				},
			},
			{
				TxType: common2.CRCProposalReview,
				Payload: &payload.CRCProposalReview{
					ProposalHash: *randomHash(),
					DID:          *randomProgramHash(),
				},
			},
			{
				TxType:  common2.CRCAppropriation,
				Payload: &payload.CRCAppropriation{},
			},
			{
				TxType: common2.WithdrawFromSideChain,
				Payload: &payload.WithdrawFromSideChain{
					SideChainTransactionHashes: []common.Uint256{
						*randomHash(),
						*randomHash(),
					},
				},
			},
		}

		verifyTxListWithConflictManager(txs, db, false, t)
	})
}

func conflictTestProc(action func(*UtxoCacheDB)) {
	origin := blockchain.DefaultLedger
	utxoCacheDB := NewUtxoCacheDB()
	blockchain.DefaultLedger = &blockchain.Ledger{
		Blockchain: &blockchain.BlockChain{
			UTXOCache: blockchain.NewUTXOCache(utxoCacheDB,
				&config.DefaultParams),
		},
	}
	action(utxoCacheDB)
	blockchain.DefaultLedger = origin
}

func setPreviousTransactionIndividually(txs []interfaces.Transaction,
	utxoCacheDB *UtxoCacheDB) {
	for _, tx := range txs {
		prevTx := newPreviousTx(utxoCacheDB)
		tx.Inputs = []*common2.Input{
			{
				Previous: common2.OutPoint{
					TxID:  prevTx.Hash(),
					Index: 0,
				},
				Sequence: 100,
			},
		}
	}
}

func setSamePreviousTransaction(txs []interfaces.Transaction,
	utxoCacheDB *UtxoCacheDB) {
	prevTx := newPreviousTx(utxoCacheDB)
	for _, tx := range txs {
		tx.Inputs = []*common2.Input{
			{
				Previous: common2.OutPoint{
					TxID:  prevTx.Hash(),
					Index: 0,
				},
				Sequence: 100,
			},
		}
	}
}

func newPreviousTx(utxoCacheDB *UtxoCacheDB) interfaces.Transaction {
	prevTx := &transactions.BaseTransaction{
		TxType:  common2.TransferAsset,
		Payload: &payload.TransferAsset{},
		Outputs: []*common2.Output{
			{
				Value:       common.Fixed64(mrand.Int63()),
				ProgramHash: *randomProgramHash(),
			},
		},
	}
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
