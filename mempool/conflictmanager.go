// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package mempool

import (
	"fmt"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"

	"github.com/elastos/Elastos.ELA/errors"
)

const (
	slotDPoSOwnerPublicKey                      = "DPoSOwnerPublicKey"
	slotDPoSNodePublicKey                       = "DPoSNodePublicKey"
	slotDPoSNickname                            = "DPoSNickname"
	slotCRDID                                   = "CrDID"
	slotCRNickname                              = "CrNickname"
	slotProgramCode                             = "ProgramCode"
	slotCRCProposalDraftHash                    = "CRCProposalDraftHash"
	slotCRCProposalDID                          = "CRCProposalDID"
	slotCRCProposalHash                         = "CRCProposalHash"
	slotCRCProposalTrackingHash                 = "CRCProposalTrackingHash"
	slotCRCProposalReviewKey                    = "CRCProposalReviewKey"
	slotCRCProposalCustomID                     = "CRCProposalCustomID"
	slotCRCProposalRegisterSideChainName        = "CRCProposalRegisterSideChainName"
	slotCRCProposalRegisterSideChainMagicNumber = "CRCProposalRegisterSideChainMagicNumber"
	slotCRCProposalRegisterSideChainGenesisHash = "CRCProposalRegisterSideChainGenesisHash"
	slotCRCAppropriationKey                     = "CRCAppropriationKey"
	slotCRCProposalRealWithdrawKey              = "CRCProposalRealWithdrawKey"
	slotDposV2ClaimRewardRealWithdrawKey        = "DposV2ClaimRewardRealWithdrawKey"
	slotCloseProposalTargetProposalHash         = "CloseProposalTargetProposalHash"
	slotChangeProposalOwnerTargetProposalHash   = "ChangeProposalOwnerTargetProposalHash"
	slotChangeCustomIDFee                       = "ChangeCustomIDFee"
	slotReserveCustomID                         = "ReserveCustomID"
	slotSpecialTxHash                           = "SpecialTxHash"
	slotSidechainTxHashes                       = "SidechainTxHashes"
	slotSidechainReturnDepositTxHashes          = "SidechainReturnDepositTxHashes"
	slotCustomIDProposalResult                  = "CustomIDProposalResult"
	slotTxInputsReferKeys                       = "TxInputsReferKeys"
	slotCRCouncilMemberNodePublicKey            = "CRCouncilMemberNodePublicKey"
	slotCRCouncilMemberDID                      = "CRCouncilMemberDID"
	slotCRCSecretaryGeneral                     = "CRCSecretaryGeneral"
	slotRevertToDPOSHash                        = "RevertToDPOSHash"
	slotUnstakeRealWithdraw                 = "UnstakeRealWithdraw"
)

type conflict struct {
	name string
	slot *conflictSlot
}

// conflictManager hold a set of conflict slots, and refer some query methods.
type conflictManager struct {
	conflictSlots []*conflict
}

func (m *conflictManager) VerifyTx(tx interfaces.Transaction) errors.ELAError {
	for _, v := range m.conflictSlots {
		if err := v.slot.VerifyTx(tx); err != nil {
			return errors.SimpleWithMessage(errors.ErrTxPoolFailure, err,
				fmt.Sprintf("slot %s verify tx error", v.name))
		}
	}
	return nil
}

func (m *conflictManager) AppendTx(tx interfaces.Transaction) errors.ELAError {
	for _, v := range m.conflictSlots {
		if err := v.slot.AppendTx(tx); err != nil {
			return errors.SimpleWithMessage(errors.ErrTxPoolFailure, err,
				fmt.Sprintf("slot %s append tx error", v.name))
		}
	}
	return nil
}

func (m *conflictManager) removeTx(tx interfaces.Transaction) errors.ELAError {
	for _, v := range m.conflictSlots {
		if err := v.slot.RemoveTx(tx); err != nil {
			return errors.SimpleWithMessage(errors.ErrTxPoolFailure, err,
				fmt.Sprintf("slot %s remove tx error", v.name))
		}
	}
	return nil
}

func (m *conflictManager) GetTx(key interface{},
	slotName string) interfaces.Transaction {
	for _, v := range m.conflictSlots {
		if v.name == slotName {
			return v.slot.GetTx(key)
		}
	}
	return nil
}

func (m *conflictManager) ContainsKey(key interface{}, slotName string) bool {
	for _, v := range m.conflictSlots {
		if v.name == slotName {
			return v.slot.Contains(key)
		}
	}
	return false
}

func (m *conflictManager) RemoveKey(key interface{},
	slotName string) errors.ELAError {
	for _, v := range m.conflictSlots {
		if v.name == slotName {
			return v.slot.removeKey(key)
		}
	}
	return errors.SimpleWithMessage(errors.ErrTxPoolFailure, nil,
		fmt.Sprintf("slot %s not exist", slotName))
}

func (m *conflictManager) Empty() bool {
	for _, v := range m.conflictSlots {
		if !v.slot.Empty() {
			return false
		}
	}
	return true
}

func newConflictManager() conflictManager {
	return conflictManager{
		conflictSlots: []*conflict{
			// DPoS owner public key
			{
				name: slotDPoSOwnerPublicKey,
				slot: newConflictSlot(str,
					keyTypeFuncPair{
						Type: common2.RegisterProducer,
						Func: strProducerInfoOwnerPublicKey,
					},
					keyTypeFuncPair{
						Type: common2.UpdateProducer,
						Func: strProducerInfoOwnerPublicKey,
					},
					keyTypeFuncPair{
						Type: common2.CancelProducer,
						Func: strCancelProducerOwnerPublicKey,
					},
					keyTypeFuncPair{
						Type: common2.RegisterCR,
						Func: strRegisterCRPublicKey,
					},
				),
			},
			// DPoS node public key
			{
				name: slotDPoSNodePublicKey,
				slot: newConflictSlot(str,
					keyTypeFuncPair{
						Type: common2.RegisterProducer,
						Func: strProducerInfoNodePublicKey,
					},
					keyTypeFuncPair{
						Type: common2.UpdateProducer,
						Func: strProducerInfoNodePublicKey,
					},
					keyTypeFuncPair{
						Type: common2.ActivateProducer,
						Func: strActivateProducerNodePublicKey,
					},
					keyTypeFuncPair{
						Type: common2.RegisterCR,
						Func: strRegisterCRPublicKey,
					},
					keyTypeFuncPair{
						Type: common2.CRCouncilMemberClaimNode,
						Func: strCRManagementPublicKey,
					},
				),
			},
			// CR claim DPOS node public key
			{
				name: slotCRCouncilMemberNodePublicKey,
				slot: newConflictSlot(str,
					keyTypeFuncPair{
						Type: common2.CRCouncilMemberClaimNode,
						Func: strCRManagementPublicKey,
					},
				),
			},
			// CR claim DPOS node did
			{
				name: slotCRCouncilMemberDID,
				slot: newConflictSlot(programHash,
					keyTypeFuncPair{
						Type: common2.CRCouncilMemberClaimNode,
						Func: strCRManagementDID,
					},
				),
			},
			// DPoS nickname
			{
				name: slotDPoSNickname,
				slot: newConflictSlot(str,
					keyTypeFuncPair{
						Type: common2.RegisterProducer,
						Func: strProducerInfoNickname,
					},
					keyTypeFuncPair{
						Type: common2.UpdateProducer,
						Func: strProducerInfoNickname,
					},
				),
			},
			// CR CID
			{
				name: slotCRDID,
				slot: newConflictSlot(programHash,
					keyTypeFuncPair{
						Type: common2.RegisterCR,
						Func: addrCRInfoCRCID,
					},
					keyTypeFuncPair{
						Type: common2.UpdateCR,
						Func: addrCRInfoCRCID,
					},
					keyTypeFuncPair{
						Type: common2.UnregisterCR,
						Func: addrUnregisterCRCID,
					},
				),
			},
			// CR nickname
			{
				name: slotCRNickname,
				slot: newConflictSlot(str,
					keyTypeFuncPair{
						Type: common2.RegisterCR,
						Func: strCRInfoNickname,
					},
					keyTypeFuncPair{
						Type: common2.UpdateCR,
						Func: strCRInfoNickname,
					},
				),
			},
			// CR and DPoS program code
			{
				name: slotProgramCode,
				slot: newConflictSlot(str,
					keyTypeFuncPair{
						Type: common2.ReturnDepositCoin,
						Func: strTxProgramCode,
					},
					keyTypeFuncPair{
						Type: common2.ReturnCRDepositCoin,
						Func: strTxProgramCode,
					},
				),
			},
			// Proposal change custom ID fee.
			{
				name: slotChangeCustomIDFee,
				slot: newConflictSlot(str,
					keyTypeFuncPair{
						Type: common2.CRCProposal,
						Func: strChangeCustomIDFee,
					},
				),
			},
			{
				name: slotReserveCustomID,
				slot: newConflictSlot(str,
					keyTypeFuncPair{
						Type: common2.CRCProposal,
						Func: strReserveCustomID,
					},
				),
			},
			// CRC Proposal target proposal hash
			{
				name: slotCloseProposalTargetProposalHash,
				slot: newConflictSlot(hash,
					keyTypeFuncPair{
						Type: common2.CRCProposal,
						Func: hashCloseProposalTargetProposalHash,
					},
				),
			},
			{
				name: slotChangeProposalOwnerTargetProposalHash,
				slot: newConflictSlot(hash,
					keyTypeFuncPair{
						Type: common2.CRCProposal,
						Func: hashChangeProposalOwnerTargetProposalHash,
					},
				),
			},
			// CRC proposal draft hash
			{
				name: slotCRCProposalDraftHash,
				slot: newConflictSlot(hash,
					keyTypeFuncPair{
						Type: common2.CRCProposal,
						Func: hashCRCProposalDraftHash,
					},
				),
			},
			// CRC proposal DID
			{
				name: slotCRCProposalDID,
				slot: newConflictSlot(programHash,
					keyTypeFuncPair{
						Type: common2.CRCProposal,
						Func: hashCRCProposalDID,
					},
				),
			},
			// CRC proposal CustomID
			{
				name: slotCRCProposalCustomID,
				slot: newConflictSlot(strArray,
					keyTypeFuncPair{
						Type: common2.CRCProposal,
						Func: strArrayCRCProposalCustomID,
					},
				),
			},
			// CRC Proposal register sidechain sidechain name
			{
				name: slotCRCProposalRegisterSideChainName,
				slot: newConflictSlot(str,
					keyTypeFuncPair{
						Type: common2.CRCProposal,
						Func: hashCRCProposalRegisterSideChainName,
					},
				),
			},
			// CRC Proposal register sidechain magic number
			{
				name: slotCRCProposalRegisterSideChainMagicNumber,
				slot: newConflictSlot(str,
					keyTypeFuncPair{
						Type: common2.CRCProposal,
						Func: hashCRCProposalRegisterSideChainMagicNumber,
					},
				),
			},
			// CRC Proposal register sidechain
			{
				name: slotCRCProposalRegisterSideChainGenesisHash,
				slot: newConflictSlot(hash,
					keyTypeFuncPair{
						Type: common2.CRCProposal,
						Func: hashCRCProposalRegisterSideChainGenesisHash,
					},
				),
			},
			// CRC proposal hash
			{
				name: slotCRCProposalHash,
				slot: newConflictSlot(hash,
					keyTypeFuncPair{
						Type: common2.CRCProposalWithdraw,
						Func: hashCRCProposalWithdrawProposalHash,
					},
				),
			},
			// CRC proposal tracking hash
			{
				name: slotCRCProposalTrackingHash,
				slot: newConflictSlot(hash,
					keyTypeFuncPair{
						Type: common2.CRCProposalTracking,
						Func: hashCRCProposalTrackingProposalHash,
					},
				),
			},
			// CRC proposal review key
			{
				name: slotCRCProposalReviewKey,
				slot: newConflictSlot(str,
					keyTypeFuncPair{
						Type: common2.CRCProposalReview,
						Func: strProposalReviewKey,
					},
				),
			},
			// CRC appropriation key
			{
				name: slotCRCAppropriationKey,
				slot: newConflictSlot(str,
					keyTypeFuncPair{
						Type: common2.CRCAppropriation,
						Func: strCRCAppropriation,
					},
				),
			},
			// secretary general key
			{
				name: slotCRCSecretaryGeneral,
				slot: newConflictSlot(str,
					keyTypeFuncPair{
						Type: common2.CRCProposal,
						Func: strSecretaryGeneral,
					},
				),
			},
			// CRC proposal real withdraw transaction key
			{
				name: slotCRCProposalRealWithdrawKey,
				slot: newConflictSlot(hashArray,
					keyTypeFuncPair{
						Type: common2.CRCProposalRealWithdraw,
						Func: hashArrayCRCProposalRealWithdrawTransactionHashes,
					},
				),
			},
			{
				name: slotDposV2ClaimRewardRealWithdrawKey,
				slot: newConflictSlot(hashArray,
					keyTypeFuncPair{
						Type: common2.DposV2ClaimRewardRealWithdraw,
						Func: hashArrayDposV2ClaimRewardRealWithdrawTransactionHashes,
					},
				),
			},
			// UnstakeRealWithdraw key
			{
				name: slotUnstakeRealWithdraw,
				slot: newConflictSlot(str,
					keyTypeFuncPair{
						Type: common2.UnstakeRealWithdraw,
						Func: strUnstakeRealWithdrawTX,
					},
				),
			},
			{
				name: slotRevertToDPOSHash,
				slot: newConflictSlot(str,
					keyTypeFuncPair{
						Type: common2.RevertToDPOS,
						Func: hashRevertToDPOS,
					},
				),
			},
			// special tx hash
			{
				name: slotSpecialTxHash,
				slot: newConflictSlot(hash,
					keyTypeFuncPair{
						Type: common2.IllegalProposalEvidence,
						Func: hashSpecialTxHash,
					},
					keyTypeFuncPair{
						Type: common2.IllegalVoteEvidence,
						Func: hashSpecialTxHash,
					},
					keyTypeFuncPair{
						Type: common2.IllegalBlockEvidence,
						Func: hashSpecialTxHash,
					},
					keyTypeFuncPair{
						Type: common2.IllegalSidechainEvidence,
						Func: hashSpecialTxHash,
					},
					keyTypeFuncPair{
						Type: common2.InactiveArbitrators,
						Func: hashSpecialTxHash,
					},
					keyTypeFuncPair{
						Type: common2.NextTurnDPOSInfo,
						Func: hashNextTurnDPOSInfoTxPayloadHash,
					},
				),
			},
			// custom id proposal result.
			{
				name: slotCustomIDProposalResult,
				slot: newConflictSlot(str,
					keyTypeFuncPair{
						Type: common2.ProposalResult,
						Func: hashCustomIDProposalResultTxPayloadHash,
					},
				),
			},
			// side chain transaction hashes
			{
				name: slotSidechainTxHashes,
				slot: newConflictSlot(hashArray,
					keyTypeFuncPair{
						Type: common2.WithdrawFromSideChain,
						Func: hashArraySidechainTransactionHashes,
					},
				),
			},
			// side chain transaction hashes
			{
				name: slotSidechainReturnDepositTxHashes,
				slot: newConflictSlot(hashArray,
					keyTypeFuncPair{
						Type: common2.ReturnSideChainDepositCoin,
						Func: hashArraySidechainReturnDepositTransactionHashes,
					},
				),
			},
			// tx inputs refer keys
			{
				name: slotTxInputsReferKeys,
				slot: newConflictSlot(strArray,
					keyTypeFuncPair{
						Type: allType,
						Func: strArrayTxReferences,
					},
				),
			},
		},
	}
}
