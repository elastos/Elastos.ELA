// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package interfaces

import (
	"github.com/elastos/Elastos.ELA/common"
	"io"
)

type Transaction interface {
	PayloadChecker

	String() string
	Serialize(w io.Writer) error
	SerializeUnsigned(w io.Writer) error
	SerializeSizeStripped() int
	Deserialize(r io.Reader) error
	DeserializeUnsigned(r io.Reader) error
	GetSize() int
	Hash() common.Uint256

	IsReturnSideChainDepositCoinTx() bool
	ISCRCouncilMemberClaimNode() bool
	IsCRAssetsRectifyTx() bool
	IsCRCAppropriationTx() bool
	IsNextTurnDPOSInfoTx() bool
	IsCustomIDResultTx() bool
	IsCustomIDRelatedTx() bool
	IsSideChainUpgradeTx() bool
	IsCRCProposalRealWithdrawTx() bool
	IsUpdateCRTx() bool
	IsCRCProposalWithdrawTx() bool
	IsCRCProposalReviewTx() bool
	IsCRCProposalTrackingTx() bool
	IsCRCProposalTx() bool
	IsReturnCRDepositCoinTx() bool
	IsUnregisterCRTx() bool
	IsRegisterCRTx() bool
	IsIllegalTypeTx() bool
	IsSpecialTx() bool
	GetSpecialTxHash() (common.Uint256, error)
	IsIllegalProposalTx() bool
	IsIllegalVoteTx() bool
	IsIllegalBlockTx() bool
	IsSidechainIllegalDataTx() bool
	IsInactiveArbitrators() bool
	IsRevertToPOW() bool
	IsRevertToDPOS() bool
	IsUpdateVersion() bool
	IsProducerRelatedTx() bool
	IsUpdateProducerTx() bool
	IsReturnDepositCoin() bool
	IsCancelProducerTx() bool
	IsActivateProducerTx() bool
	IsRegisterProducerTx() bool
	IsSideChainPowTx() bool
	IsNewSideChainPowTx() bool
	IsTransferCrossChainAssetTx() bool
	IsWithdrawFromSideChainTx() bool
	IsRechargeToSideChainTx() bool
	IsCoinBaseTx() bool
	IsSmallTransfer(min common.Fixed64) bool
}
