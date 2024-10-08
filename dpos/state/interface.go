// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
)

type Arbitrators interface {
	Start()
	CheckDPOSIllegalTx(block *types.Block) error
	ProcessSpecialTxPayload(p interfaces.Payload, height uint32) error
	CheckCRCAppropriationTx(block *types.Block) error
	CheckNextTurnDPOSInfoTx(block *types.Block) error
	CheckCustomIDResultsTx(block *types.Block) error
	CheckRevertToDPOSTX(block *types.Block) error

	IsArbitrator(pk []byte) bool
	IsNextCRCArbitrator(pk []byte) bool
	IsMemberElectedNextCRCArbitrator(pk []byte) bool
	GetCurrentAndLastArbitrators() ([]*ArbiterInfo, []*ArbiterInfo)
	GetArbitrators() []*ArbiterInfo
	GetCandidates() [][]byte
	GetAllProducersPublicKey() []string
	GetNextArbitrators() []*ArbiterInfo
	GetNextCandidates() [][]byte
	GetCurrentNeedConnectArbiters() []peer.PID
	GetNextNeedConnectArbiters() []peer.PID
	GetNeedConnectCRArbiters() []peer.PID
	GetNeedConnectArbiters() []peer.PID
	GetDutyIndexByHeight(height uint32) int
	GetDutyIndex() int

	GetCurrentRewardData() RewardData
	GetNextRewardData() RewardData
	GetArbitersRoundReward() map[common.Uint168]common.Fixed64
	GetFinalRoundChange() common.Fixed64
	SetNeedRevertToDPOSTX(need bool)
	SetNeedNextTurnDPOSInfo(need bool)
	IsInPOWMode() bool
	GetRevertToPOWBlockHeight() uint32
	GetLastBlockTimestamp() uint32
	IsInactiveMode() bool
	IsUnderstaffedMode() bool

	GetConnectedProducer(publicKey []byte) ArbiterMember
	GetCRCArbiters() []*ArbiterInfo
	GetNextCRCArbiters() [][]byte
	GetAllNextCRCArbiters() [][]byte
	CRCProducerCount() int
	IsCRCArbitrator(pk []byte) bool
	IsActiveProducer(pk []byte) bool
	IsDisabledProducer(pk []byte) bool
	IsNeedNextTurnDPOSInfo() bool
	IsDPoSV2Run(blockHeight uint32) bool

	GetOnDutyArbitrator() []byte
	GetNextOnDutyArbitrator(offset uint32) []byte

	GetOnDutyCrossChainArbitrator() []byte
	GetCrossChainArbiters() []*ArbiterInfo
	GetCrossChainArbitersCount() int
	GetCrossChainArbitersMajorityCount() int

	GetArbitersCount() int
	GetCRCArbitersCount() int
	GetArbitersMajorityCount() int
	HasArbitersMajorityCount(num int) bool
	HasArbitersMinorityCount(num int) bool
	GetDPoSV2ActiveHeight() uint32
	HasArbitersHalfMinorityCount(num int) bool

	GetSnapshot(height uint32) []*CheckPoint
	DumpInfo(height uint32)
}

type IArbitratorsRecord interface {
	GetHeightsDesc() ([]uint32, error)
	GetCheckPoint(height uint32) (*CheckPoint, error)
	SaveArbitersState(point *CheckPoint) error
}
