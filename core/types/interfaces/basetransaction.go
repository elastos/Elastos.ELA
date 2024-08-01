// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package interfaces

import (
	"io"

	"github.com/elastos/Elastos.ELA/common"
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
)

type Transaction interface {
	TransactionChecker
	TransactionProcessor

	// get data
	Version() common2.TransactionVersion
	TxType() common2.TxType
	PayloadVersion() byte
	Payload() Payload
	Attributes() []*common2.Attribute
	Inputs() []*common2.Input
	Outputs() []*common2.Output
	LockTime() uint32
	Programs() []*pg.Program
	Fee() common.Fixed64
	FeePerKB() common.Fixed64

	// set data
	SetVersion(version common2.TransactionVersion)
	SetTxType(txType common2.TxType)
	SetFee(fee common.Fixed64)
	SetFeePerKB(feePerKB common.Fixed64)
	SetAttributes(attributes []*common2.Attribute)
	SetPayloadVersion(payloadVersion byte)
	SetPayload(payload Payload)
	SetInputs(inputs []*common2.Input)
	SetOutputs(outputs []*common2.Output)
	SetPrograms(programs []*pg.Program)
	SetLockTime(lockTime uint32)

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
	IsDposV2ClaimRewardTx() bool
	IsDposV2ClaimRewardRealWithdraw() bool
	IsVotesRealWithdrawTX() bool
	IsCreateNFTTX() bool
	IsRecordSponorTx() bool
}
