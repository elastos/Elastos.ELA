// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
	"github.com/elastos/Elastos.ELA/events"
	"github.com/elastos/Elastos.ELA/utils"
)

type ChangeType byte

const (
	// MajoritySignRatioNumerator defines the ratio numerator to achieve
	// majority signatures.
	MajoritySignRatioNumerator = float64(2)

	// MajoritySignRatioDenominator defines the ratio denominator to achieve
	// majority signatures.
	MajoritySignRatioDenominator = float64(3)

	// MaxNormalInactiveChangesCount defines the max count Arbiters can
	// change when more than 1/3 arbiters don't sign cause to confirm fail
	MaxNormalInactiveChangesCount = 3

	// MaxSnapshotLength defines the max length the SnapshotByHeight map should take
	MaxSnapshotLength = 20

	none         = ChangeType(0x00)
	updateNext   = ChangeType(0x01)
	normalChange = ChangeType(0x02)
)

var (
	ErrInsufficientProducer = errors.New("producers count less than min Arbiters count")
)

type ArbiterInfo struct {
	NodePublicKey   []byte
	IsNormal        bool
	IsCRMember      bool
	ClaimedDPOSNode bool
}

type Arbiters struct {
	*State
	*degradation
	ChainParams      *config.Params
	CRCommittee      *state.Committee
	bestHeight       func() uint32
	bestBlockHash    func() *common.Uint256
	getBlockByHeight func(uint32) (*types.Block, error)

	mtx       sync.Mutex
	started   bool
	DutyIndex int

	CurrentReward RewardData
	NextReward    RewardData

	CurrentArbitrators []ArbiterMember
	CurrentCandidates  []ArbiterMember
	nextArbitrators    []ArbiterMember
	nextCandidates     []ArbiterMember

	// current cr arbiters map
	CurrentCRCArbitersMap map[common.Uint168]ArbiterMember
	// next cr arbiters map
	nextCRCArbitersMap map[common.Uint168]ArbiterMember
	// next cr arbiters
	nextCRCArbiters []ArbiterMember

	crcChangedHeight           uint32
	accumulativeReward         common.Fixed64
	finalRoundChange           common.Fixed64
	clearingHeight             uint32
	arbitersRoundReward        map[common.Uint168]common.Fixed64
	illegalBlocksPayloadHashes map[common.Uint256]interface{}

	Snapshots        map[uint32][]*CheckPoint
	SnapshotKeysDesc []uint32

	forceChanged bool

	History *utils.History
}

func (a *Arbiters) Start() {
	a.mtx.Lock()
	a.started = true
	a.mtx.Unlock()
}

func (a *Arbiters) SetNeedRevertToDPOSTX(need bool) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	a.NeedRevertToDPOSTX = need
}

func (a *Arbiters) IsInPOWMode() bool {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	return a.isInPOWMode()
}

func (a *Arbiters) isInPOWMode() bool {
	return a.ConsensusAlgorithm == POW
}

func (a *Arbiters) GetRevertToPOWBlockHeight() uint32 {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	return a.RevertToPOWBlockHeight
}

func (a *Arbiters) RegisterFunction(bestHeight func() uint32,
	bestBlockHash func() *common.Uint256,
	getBlockByHeight func(uint32) (*types.Block, error),
	getTxReference func(tx interfaces.Transaction) (
		map[*common2.Input]common2.Output, error)) {
	a.bestHeight = bestHeight
	a.bestBlockHash = bestBlockHash
	a.getBlockByHeight = getBlockByHeight
	a.GetTxReference = getTxReference
}

func (a *Arbiters) IsNeedNextTurnDPOSInfo() bool {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	return a.NeedNextTurnDPOSInfo
}

func (a *Arbiters) RecoverFromCheckPoints(point *CheckPoint) {
	a.mtx.Lock()
	a.recoverFromCheckPoints(point)
	a.mtx.Unlock()
}

func (a *Arbiters) recoverFromCheckPoints(point *CheckPoint) {
	a.DutyIndex = point.DutyIndex
	a.CurrentArbitrators = point.CurrentArbitrators
	a.CurrentCandidates = point.CurrentCandidates
	a.nextArbitrators = point.NextArbitrators
	a.nextCandidates = point.NextCandidates
	a.CurrentReward = point.CurrentReward
	a.NextReward = point.NextReward
	a.StateKeyFrame = &point.StateKeyFrame
	a.accumulativeReward = point.AccumulativeReward
	a.finalRoundChange = point.FinalRoundChange
	a.clearingHeight = point.ClearingHeight
	a.arbitersRoundReward = point.ArbitersRoundReward
	a.illegalBlocksPayloadHashes = point.IllegalBlocksPayloadHashes

	a.crcChangedHeight = point.CRCChangedHeight
	a.CurrentCRCArbitersMap = point.CurrentCRCArbitersMap
	a.nextCRCArbitersMap = point.NextCRCArbitersMap
	a.nextCRCArbiters = point.NextCRCArbiters
	a.forceChanged = point.ForceChanged
}

func (a *Arbiters) ProcessBlock(block *types.Block, confirm *payload.Confirm) {
	a.State.ProcessBlock(block, confirm, a.IsDPoSV2Run(block.Height), a.DutyIndex)
	a.IncreaseChainHeight(block, confirm)
}

func (a *Arbiters) CheckDPOSIllegalTx(block *types.Block) error {

	a.mtx.Lock()
	hashes := a.illegalBlocksPayloadHashes
	a.mtx.Unlock()

	if hashes == nil || len(hashes) == 0 {
		return nil
	}

	foundMap := make(map[common.Uint256]bool)
	for k := range hashes {
		foundMap[k] = false
	}

	for _, tx := range block.Transactions {
		if tx.IsIllegalBlockTx() {
			foundMap[tx.Payload().(*payload.DPOSIllegalBlocks).Hash()] = true
		}
	}

	for _, found := range foundMap {
		if !found {
			return errors.New("expect an illegal blocks transaction in this block")
		}
	}
	return nil
}

func (a *Arbiters) CheckRevertToDPOSTX(block *types.Block) error {
	a.mtx.Lock()
	needRevertToDPOSTX := a.NeedRevertToDPOSTX
	a.mtx.Unlock()

	var revertToDPOSTxCount uint32
	for _, tx := range block.Transactions {
		if tx.IsRevertToDPOS() {
			revertToDPOSTxCount++
		}
	}

	var needRevertToDPOSTXCount uint32
	if needRevertToDPOSTX {
		needRevertToDPOSTXCount = 1
	}

	if revertToDPOSTxCount != needRevertToDPOSTXCount {
		return fmt.Errorf("current block height %d, RevertToDPOSTX "+
			"transaction count should be %d, current block contains %d",
			block.Height, needRevertToDPOSTXCount, revertToDPOSTxCount)
	}

	return nil
}

func (a *Arbiters) CheckNextTurnDPOSInfoTx(block *types.Block) error {
	a.mtx.Lock()
	needNextTurnDposInfo := a.NeedNextTurnDPOSInfo
	a.mtx.Unlock()

	var nextTurnDPOSInfoTxCount uint32
	for _, tx := range block.Transactions {
		if tx.IsNextTurnDPOSInfoTx() {
			nextTurnDPOSInfoTxCount++
		}
	}

	var needNextTurnDPOSInfoCount uint32
	if needNextTurnDposInfo {
		needNextTurnDPOSInfoCount = 1
	}

	if nextTurnDPOSInfoTxCount != needNextTurnDPOSInfoCount {
		return fmt.Errorf("current block height %d, NextTurnDPOSInfo "+
			"transaction count should be %d, current block contains %d",
			block.Height, needNextTurnDPOSInfoCount, nextTurnDPOSInfoTxCount)
	}

	return nil
}

func (a *Arbiters) CheckCRCAppropriationTx(block *types.Block) error {
	a.mtx.Lock()
	needAppropriation := a.CRCommittee.NeedAppropriation
	a.mtx.Unlock()

	var appropriationCount uint32
	for _, tx := range block.Transactions {
		if tx.IsCRCAppropriationTx() {
			appropriationCount++
		}
	}

	var needAppropriationCount uint32
	if needAppropriation {
		needAppropriationCount = 1
	}

	if appropriationCount != needAppropriationCount {
		return fmt.Errorf("current block height %d, appropriation "+
			"transaction count should be %d, current block contains %d",
			block.Height, needAppropriationCount, appropriationCount)
	}

	return nil
}

func (a *Arbiters) CheckCustomIDResultsTx(block *types.Block) error {
	a.mtx.Lock()
	needCustomProposalResult := a.CRCommittee.NeedRecordProposalResult
	a.mtx.Unlock()

	var cidProposalResultCount uint32
	for _, tx := range block.Transactions {
		if tx.IsCustomIDResultTx() {
			cidProposalResultCount++
		}
	}

	var needCIDProposalResultCount uint32
	if needCustomProposalResult {
		needCIDProposalResultCount = 1
	}

	if cidProposalResultCount != needCIDProposalResultCount {
		return fmt.Errorf("current block height %d, custom ID result "+
			"transaction count should be %d, current block contains %d",
			block.Height, needCIDProposalResultCount, cidProposalResultCount)
	}

	return nil
}

func (a *Arbiters) ProcessSpecialTxPayload(p interfaces.Payload,
	height uint32) error {
	switch obj := p.(type) {
	case *payload.DPOSIllegalBlocks:
		a.mtx.Lock()
		a.illegalBlocksPayloadHashes[obj.Hash()] = nil
		a.mtx.Unlock()
	case *payload.InactiveArbitrators:
		if !a.AddInactivePayload(obj) {
			log.Debug("[ProcessSpecialTxPayload] duplicated payload")
			return nil
		}
	default:
		return errors.New("[ProcessSpecialTxPayload] invalid payload type")
	}

	a.State.ProcessSpecialTxPayload(p, height)
	return a.ForceChange(height)
}

func (a *Arbiters) RollbackSeekTo(height uint32) {
	a.mtx.Lock()
	a.History.RollbackSeekTo(height)
	a.State.RollbackSeekTo(height)
	a.mtx.Unlock()
}

func (a *Arbiters) RollbackTo(height uint32) error {
	a.mtx.Lock()
	a.History.RollbackTo(height)
	a.degradation.RollbackTo(height)
	err := a.State.RollbackTo(height)
	a.mtx.Unlock()

	return err
}

func (a *Arbiters) GetDutyIndexByHeight(height uint32) (index int) {
	a.mtx.Lock()
	if height >= a.ChainParams.DPOSNodeCrossChainHeight {
		if len(a.CurrentArbitrators) == 0 {
			index = 0
		} else {
			index = a.DutyIndex % len(a.CurrentArbitrators)
		}
	} else if height >= a.ChainParams.CRClaimDPOSNodeStartHeight {
		if len(a.CurrentCRCArbitersMap) == 0 {
			index = 0
		} else {
			index = a.DutyIndex % len(a.CurrentCRCArbitersMap)
		}
	} else if height >= a.ChainParams.CRCOnlyDPOSHeight-1 {
		if len(a.CurrentCRCArbitersMap) == 0 {
			index = 0
		} else {
			index = int(height-a.ChainParams.CRCOnlyDPOSHeight+1) % len(a.CurrentCRCArbitersMap)
		}
	} else {
		if len(a.CurrentArbitrators) == 0 {
			index = 0
		} else {
			index = int(height) % len(a.CurrentArbitrators)
		}
	}
	a.mtx.Unlock()
	return index
}

func (a *Arbiters) GetDutyIndex() int {
	a.mtx.Lock()
	index := a.DutyIndex
	a.mtx.Unlock()

	return index
}

func (a *Arbiters) GetArbitersRoundReward() map[common.Uint168]common.Fixed64 {
	a.mtx.Lock()
	result := a.arbitersRoundReward
	a.mtx.Unlock()

	return result
}

func (a *Arbiters) GetFinalRoundChange() common.Fixed64 {
	a.mtx.Lock()
	result := a.finalRoundChange
	a.mtx.Unlock()

	return result
}

func (a *Arbiters) GetLastBlockTimestamp() uint32 {
	a.mtx.Lock()
	result := a.LastBlockTimestamp
	a.mtx.Unlock()

	return result
}

func (a *Arbiters) ForceChange(height uint32) error {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	return a.forceChange(height)
}

func (a *Arbiters) forceChange(height uint32) error {
	block, err := a.getBlockByHeight(height)
	if err != nil {
		block, err = a.getBlockByHeight(a.bestHeight())
		if err != nil {
			return err
		}
	}
	a.SnapshotByHeight(height)

	if !a.isDPoSV2Run(block.Height) {
		if err := a.clearingDPOSReward(block, block.Height, false); err != nil {
			panic(fmt.Sprintf("normal change fail when clear DPOS reward: "+
				" transaction, height: %d, error: %s", block.Height, err))
		}
	}

	if err := a.UpdateNextArbitrators(height+1, height); err != nil {
		log.Info("force change failed at height:", height)
		return err
	}

	if err := a.ChangeCurrentArbitrators(height); err != nil {
		return err
	}

	if a.started {
		go events.Notify(events.ETDirectPeersChanged,
			a.getNeedConnectArbiters())

		currentArbiters := a.getCurrentNeedConnectArbiters()
		nextArbiters := a.getNextNeedConnectArbiters()

		go events.Notify(events.ETDirectPeersChangedV2,
			&peer.PeersInfo{CurrentPeers: currentArbiters, NextPeers: nextArbiters})
	}
	oriForceChanged := a.forceChanged
	a.History.Append(height, func() {
		a.forceChanged = true
	}, func() {
		a.forceChanged = oriForceChanged
	})
	a.History.Commit(height)

	a.dumpInfo(height)
	return nil
}

func (a *Arbiters) tryHandleError(height uint32, err error) error {
	if err == ErrInsufficientProducer {
		log.Warn("found error: ", err, ", degrade to CRC only state")
		a.TrySetUnderstaffed(height)
		return nil
	} else {
		return err
	}
}

func (a *Arbiters) normalChange(height uint32) error {
	if err := a.ChangeCurrentArbitrators(height); err != nil {
		log.Warn("[NormalChange] change current arbiters error: ", err)
		return err
	}
	if err := a.UpdateNextArbitrators(height+1, height); err != nil {
		log.Warn("[NormalChange] update next arbiters error: ", err)
		return err
	}
	return nil
}

func (a *Arbiters) notifyNextTurnDPOSInfoTx(blockHeight, versionHeight uint32, forceChange bool) {
	if blockHeight+uint32(a.ChainParams.GeneralArbiters+len(a.ChainParams.CRCArbiters)) >= a.DPoSV2ActiveHeight {
		nextTurnDPOSInfoTx := a.createNextTurnDPOSInfoTransactionV1(blockHeight, forceChange)
		go events.Notify(events.ETAppendTxToTxPool, nextTurnDPOSInfoTx)

		return
	}

	nextTurnDPOSInfoTx := a.createNextTurnDPOSInfoTransactionV0(blockHeight, forceChange)
	go events.Notify(events.ETAppendTxToTxPool, nextTurnDPOSInfoTx)
	return
}

func (a *Arbiters) IncreaseChainHeight(block *types.Block, confirm *payload.Confirm) {
	var notify = true
	var snapshotVotes = true
	a.mtx.Lock()

	var containsIllegalBlockEvidence bool
	for _, tx := range block.Transactions {
		if tx.IsIllegalBlockTx() {
			containsIllegalBlockEvidence = true
			break
		}
	}
	var forceChanged bool
	if containsIllegalBlockEvidence {
		if err := a.forceChange(block.Height); err != nil {
			log.Errorf("Found illegal blocks, ForceChange failed:%s", err)
			a.cleanArbitrators(block.Height)
			a.revertToPOWAtNextTurn(block.Height)
			log.Warn(fmt.Sprintf("force change fail at height: %d, error: %s",
				block.Height, err))
		}
		forceChanged = true
	} else {
		changeType, versionHeight := a.getChangeType(block.Height + 1)
		switch changeType {
		case updateNext:
			if err := a.UpdateNextArbitrators(versionHeight, block.Height); err != nil {
				a.revertToPOWAtNextTurn(block.Height)
				log.Warn(fmt.Sprintf("update next arbiters at height: %d, "+
					"error: %s, revert to POW mode", block.Height, err))
			}
		case normalChange:
			if a.isDPoSV2Run(block.Height) {
				if block.Height == a.DPoSV2ActiveHeight {
					if err := a.clearingDPOSReward(block, block.Height, true); err != nil {
						panic(fmt.Sprintf("normal change fail when clear DPOS reward: "+
							" transaction, height: %d, error: %s", block.Height, err))
					}
				} else {
					a.accumulateReward(block, confirm)
				}
			} else {
				if err := a.clearingDPOSReward(block, block.Height, true); err != nil {
					panic(fmt.Sprintf("normal change fail when clear DPOS reward: "+
						" transaction, height: %d, error: %s", block.Height, err))
				}
			}
			if err := a.normalChange(block.Height); err != nil {
				a.revertToPOWAtNextTurn(block.Height)
				log.Warn(fmt.Sprintf("normal change fail at height: %d, "+
					"error: %s， revert to POW mode", block.Height, err))
			}
		case none:
			a.accumulateReward(block, confirm)
			notify = false
			snapshotVotes = false
		}
	}

	oriIllegalBlocks := a.illegalBlocksPayloadHashes
	a.History.Append(block.Height, func() {
		a.illegalBlocksPayloadHashes = make(map[common.Uint256]interface{})
	}, func() {
		a.illegalBlocksPayloadHashes = oriIllegalBlocks
	})
	a.History.Commit(block.Height)
	bestHeight := a.bestHeight()
	if a.ConsensusAlgorithm != POW && block.Height >= bestHeight {
		if len(a.CurrentArbitrators) == 0 && (a.NoClaimDPOSNode || a.NoProducers) {
			a.createRevertToPOWTransaction(block.Height)
		}
	}
	if snapshotVotes {
		if err := a.snapshotVotesStates(block.Height); err != nil {
			panic(fmt.Sprintf("snap shot votes states error:%s", err))
		}
		a.History.Commit(block.Height)
	}
	if block.Height > bestHeight-MaxSnapshotLength {
		a.SnapshotByHeight(block.Height)
	}
	if block.Height >= bestHeight && (a.NeedNextTurnDPOSInfo || forceChanged) {
		a.notifyNextTurnDPOSInfoTx(block.Height, block.Height+1, forceChanged)
	}
	a.mtx.Unlock()
	if a.started && notify {
		go events.Notify(events.ETDirectPeersChanged, a.GetNeedConnectArbiters())

		currentArbiters := a.GetCurrentNeedConnectArbiters()
		nextArbiters := a.GetNextNeedConnectArbiters()

		go events.Notify(events.ETDirectPeersChangedV2,
			&peer.PeersInfo{CurrentPeers: currentArbiters, NextPeers: nextArbiters})
	}
}

func (a *Arbiters) createRevertToPOWTransaction(blockHeight uint32) {

	var revertType payload.RevertType
	if a.NoClaimDPOSNode {
		revertType = payload.NoClaimDPOSNode
	} else {
		revertType = payload.NoProducers
	}
	revertToPOWPayload := payload.RevertToPOW{
		Type:          revertType,
		WorkingHeight: blockHeight + 1,
	}
	tx := functions.CreateTransaction(
		common2.TxVersion09,
		common2.RevertToPOW,
		payload.RevertToPOWVersion,
		&revertToPOWPayload,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	go events.Notify(events.ETAppendTxToTxPoolWithoutRelay, tx)
}

func (a *Arbiters) revertToPOWAtNextTurn(height uint32) {
	oriNextArbitrators := a.nextArbitrators
	oriNextCandidates := a.nextCandidates
	oriNextCRCArbitersMap := a.nextCRCArbitersMap
	oriNextCRCArbiters := a.nextCRCArbiters
	oriNoProducers := a.NoProducers

	a.History.Append(height, func() {
		a.nextArbitrators = make([]ArbiterMember, 0)
		a.nextCandidates = make([]ArbiterMember, 0)
		a.nextCRCArbitersMap = make(map[common.Uint168]ArbiterMember)
		a.nextCRCArbiters = make([]ArbiterMember, 0)
		if a.ConsensusAlgorithm == DPOS {
			a.NoProducers = true
		}
	}, func() {
		a.nextArbitrators = oriNextArbitrators
		a.nextCandidates = oriNextCandidates
		a.nextCRCArbitersMap = oriNextCRCArbitersMap
		a.nextCRCArbiters = oriNextCRCArbiters
		a.NoProducers = oriNoProducers
	})
}
func (a *Arbiters) AccumulateReward(block *types.Block, confirm *payload.Confirm) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	a.accumulateReward(block, confirm)
}

// is already DPoS V2. when we are here we need new reward.
func (a *Arbiters) IsDPoSV2Run(blockHeight uint32) bool {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	return a.isDPoSV2Run(blockHeight)
}

func (a *Arbiters) GetDPoSV2ActiveHeight() uint32 {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	return a.DPoSV2ActiveHeight
}

// is already DPoS V2. when we are here we need new reward.
func (a *Arbiters) isDPoSV2Run(blockHeight uint32) bool {
	return blockHeight >= a.DPoSV2ActiveHeight
}

func (a *Arbiters) getDPoSV2Rewards(dposReward common.Fixed64, sponsor []byte) (rewards map[string]common.Fixed64) {
	log.Debugf("accumulateReward dposReward %v", dposReward)
	ownerPubKeyStr := a.getProducerKey(sponsor)
	ownerPubKeyBytes, _ := hex.DecodeString(ownerPubKeyStr)
	ownerProgramHash, _ := contract.PublicKeyToStandardProgramHash(ownerPubKeyBytes)
	ownerAddr, _ := ownerProgramHash.ToAddress()

	rewards = make(map[string]common.Fixed64)
	if _, ok := a.CurrentCRCArbitersMap[*ownerProgramHash]; ok { // crc
		// all reward to DPoS node owner
		rewards[ownerAddr] += dposReward
	} else {

		// DPoS votes reward is: reward * 3 /4
		votesReward := dposReward * 3 / 4

		producer := a.getProducer(sponsor)
		if producer == nil {
			log.Error("accumulateReward Sponsor not exist ", hex.EncodeToString(sponsor))
			return
		}
		producersN := make(map[common.Uint168]float64)
		var totalNI float64
		for sVoteAddr, sVoteDetail := range producer.detailedDPoSV2Votes {
			var totalN float64
			for _, votes := range sVoteDetail {
				weightF := math.Log10(float64(votes.Info[0].LockTime-votes.BlockHeight) / 7200 * 10)
				N := common.Fixed64(float64(votes.Info[0].Votes) * weightF)
				totalN += float64(N)
			}

			producersN[sVoteAddr] = totalN
			totalNI += totalN
		}

		for sVoteAddr, N := range producersN {
			b := sVoteAddr.Bytes()
			b[0] = byte(contract.PrefixStandard)
			standardUint168, _ := common.Uint168FromBytes(b)
			addr, _ := standardUint168.ToAddress()
			p := N / totalNI * float64(votesReward)
			rewards[addr] += common.Fixed64(p)
			log.Debugf("getDPoSV2Rewards addr %s  add p %v %f rewards[addr]%v \n", addr, common.Fixed64(p), rewards[addr])
		}

		var totalUsedVotesReward common.Fixed64
		for _, v := range rewards {
			totalUsedVotesReward += v
		}

		// DPoS node reward is: reward - totalUsedVotesReward
		dposNodeReward := dposReward - totalUsedVotesReward
		rewards[ownerAddr] += dposNodeReward
		log.Debugf("getDPoSV2Rewards totalUsedVotesReward %v dposNodeReward %v,  \n", totalUsedVotesReward, dposNodeReward)

	}

	return rewards
}

func (a *Arbiters) accumulateReward(block *types.Block, confirm *payload.Confirm) {
	if block.Height < a.ChainParams.PublicDPOSHeight {
		oriDutyIndex := a.DutyIndex
		a.History.Append(block.Height, func() {
			a.DutyIndex = oriDutyIndex + 1
		}, func() {
			a.DutyIndex = oriDutyIndex
		})
		return
	}

	var accumulative common.Fixed64
	accumulative = a.accumulativeReward
	var dposReward common.Fixed64
	if block.Height < a.ChainParams.CRVotingStartHeight || !a.forceChanged {
		dposReward = a.getBlockDPOSReward(block)
		accumulative += dposReward
	}

	if a.isDPoSV2Run(block.Height) {
		log.Debugf("accumulateReward dposReward %v", dposReward)
		oriDutyIndex := a.DutyIndex
		oriForceChanged := a.forceChanged
		oriDposV2RewardInfo := a.DposV2RewardInfo
		rewards := a.getDPoSV2Rewards(dposReward, confirm.Proposal.Sponsor)

		a.History.Append(block.Height, func() {
			for k, v := range rewards {
				a.DposV2RewardInfo[k] += v
			}
			a.forceChanged = false
			a.DutyIndex = oriDutyIndex + 1
		}, func() {
			a.DposV2RewardInfo = oriDposV2RewardInfo
			a.forceChanged = oriForceChanged
			a.DutyIndex = oriDutyIndex
		})

	} else {
		oriAccumulativeReward := a.accumulativeReward
		oriArbitersRoundReward := a.arbitersRoundReward
		oriFinalRoundChange := a.finalRoundChange
		oriForceChanged := a.forceChanged
		oriDutyIndex := a.DutyIndex
		a.History.Append(block.Height, func() {
			a.accumulativeReward = accumulative
			a.arbitersRoundReward = nil
			a.finalRoundChange = 0
			a.forceChanged = false
			a.DutyIndex = oriDutyIndex + 1
		}, func() {
			a.accumulativeReward = oriAccumulativeReward
			a.arbitersRoundReward = oriArbitersRoundReward
			a.finalRoundChange = oriFinalRoundChange
			a.forceChanged = oriForceChanged
			a.DutyIndex = oriDutyIndex
		})
	}
}

func (a *Arbiters) clearingDPOSReward(block *types.Block, historyHeight uint32,
	smoothClearing bool) (err error) {
	if block.Height < a.ChainParams.PublicDPOSHeight ||
		block.Height == a.clearingHeight {
		return nil
	}

	dposReward := a.getBlockDPOSReward(block)
	accumulativeReward := a.accumulativeReward
	if smoothClearing {
		accumulativeReward += dposReward
		dposReward = 0
	}

	var change common.Fixed64
	var roundReward map[common.Uint168]common.Fixed64
	if roundReward, change, err = a.distributeDPOSReward(block.Height,
		accumulativeReward); err != nil {
		return
	}

	oriRoundReward := a.arbitersRoundReward
	oriAccumulativeReward := a.accumulativeReward
	oriClearingHeight := a.clearingHeight
	oriChange := a.finalRoundChange
	a.History.Append(historyHeight, func() {
		a.arbitersRoundReward = roundReward
		a.accumulativeReward = dposReward
		a.clearingHeight = block.Height
		a.finalRoundChange = change
	}, func() {
		a.arbitersRoundReward = oriRoundReward
		a.accumulativeReward = oriAccumulativeReward
		a.clearingHeight = oriClearingHeight
		a.finalRoundChange = oriChange
	})

	return nil
}

func (a *Arbiters) distributeDPOSReward(height uint32,
	reward common.Fixed64) (roundReward map[common.Uint168]common.Fixed64,
	change common.Fixed64, err error) {
	var realDPOSReward common.Fixed64
	if height >= a.ChainParams.ChangeCommitteeNewCRHeight+2*uint32(len(a.CurrentArbitrators)) {
		roundReward, realDPOSReward, err = a.distributeWithNormalArbitratorsV3(height, reward)
	} else if height >= a.ChainParams.CRClaimDPOSNodeStartHeight+2*uint32(len(a.CurrentArbitrators)) {
		roundReward, realDPOSReward, err = a.distributeWithNormalArbitratorsV2(height, reward)
	} else if height >= a.ChainParams.CRCommitteeStartHeight+2*uint32(len(a.CurrentArbitrators)) {
		roundReward, realDPOSReward, err = a.distributeWithNormalArbitratorsV1(height, reward)
	} else {
		roundReward, realDPOSReward, err = a.distributeWithNormalArbitratorsV0(height, reward)
	}

	if err != nil {
		return nil, 0, err
	}

	change = reward - realDPOSReward
	if change < 0 {
		log.Error("reward:", reward, "realDPOSReward:", realDPOSReward, "height:", height,
			"b", a.ChainParams.CRClaimDPOSNodeStartHeight+2*uint32(len(a.CurrentArbitrators)),
			"c", a.ChainParams.CRCommitteeStartHeight+2*uint32(len(a.CurrentArbitrators)))
		return nil, 0, errors.New("real dpos reward more than reward limit")
	}

	return
}

func (a *Arbiters) distributeWithNormalArbitratorsV3(height uint32, reward common.Fixed64) (
	map[common.Uint168]common.Fixed64, common.Fixed64, error) {
	//if len(a.CurrentArbitrators) == 0 {
	//	return nil, 0, errors.New("not found arbiters when " +
	//		"distributeWithNormalArbitratorsV3")
	//}

	roundReward := map[common.Uint168]common.Fixed64{}
	totalBlockConfirmReward := float64(reward) * 0.25
	totalTopProducersReward := float64(reward) - totalBlockConfirmReward
	// Consider that there is no only CR consensus.
	arbitersCount := len(a.ChainParams.CRCArbiters) + a.ChainParams.GeneralArbiters
	individualBlockConfirmReward := common.Fixed64(
		math.Floor(totalBlockConfirmReward / float64(arbitersCount)))
	totalVotesInRound := a.CurrentReward.TotalVotesInRound
	log.Debugf("distributeWithNormalArbitratorsV3 TotalVotesInRound %f", a.CurrentReward.TotalVotesInRound)

	if a.ConsensusAlgorithm == POW || len(a.CurrentArbitrators) == 0 ||
		len(a.ChainParams.CRCArbiters) == len(a.CurrentArbitrators) {
		// if no normal DPOS node, need to destroy reward.
		roundReward[a.ChainParams.DestroyELAAddress] = reward
		return roundReward, reward, nil
	}
	log.Debugf("totalTopProducersReward totalTopProducersReward %f", totalTopProducersReward)

	rewardPerVote := totalTopProducersReward / float64(totalVotesInRound)

	realDPOSReward := common.Fixed64(0)
	for _, arbiter := range a.CurrentArbitrators {
		ownerHash := arbiter.GetOwnerProgramHash()
		rewardHash := ownerHash
		var r common.Fixed64
		if arbiter.GetType() == CRC {
			r = individualBlockConfirmReward
			log.Debugf("1233 r =individualBlockConfirmReward %s", individualBlockConfirmReward.String())
			m, ok := arbiter.(*crcArbiter)
			if !ok || m.crMember.MemberState != state.MemberElected {
				rewardHash = a.ChainParams.DestroyELAAddress
			} else if len(m.crMember.DPOSPublicKey) == 0 {
				nodePK := arbiter.GetNodePublicKey()
				ownerPK := a.getProducerKey(nodePK)
				opk, err := common.HexStringToBytes(ownerPK)
				if err != nil {
					panic("get owner public key err:" + err.Error())
				}
				programHash, err := contract.PublicKeyToStandardProgramHash(opk)
				if err != nil {
					panic("public key to standard program hash err:" + err.Error())
				}
				votes := a.CurrentReward.OwnerVotesInRound[*programHash]
				individualCRCProducerReward := common.Fixed64(math.Floor(float64(
					votes) * rewardPerVote))
				r = individualBlockConfirmReward + individualCRCProducerReward
				rewardHash = *programHash
				log.Debugf("000 rewardHash%s  individualCRCProducerReward %s individualBlockConfirmReward %s votes %s", rewardHash.String(),
					individualCRCProducerReward.String(), individualBlockConfirmReward.String(), votes.String())
			} else {
				pk := arbiter.GetOwnerPublicKey()
				programHash, err := contract.PublicKeyToStandardProgramHash(pk)
				if err != nil {
					rewardHash = a.ChainParams.DestroyELAAddress
				} else {
					rewardHash = *programHash
				}
			}
		} else {
			votes := a.CurrentReward.OwnerVotesInRound[ownerHash]
			individualProducerReward := common.Fixed64(math.Floor(float64(
				votes) * rewardPerVote))
			r = individualBlockConfirmReward + individualProducerReward
			log.Debugf("111 ownerHash%s  individualProducerReward %s individualBlockConfirmReward %s rewardPerVote %f", ownerHash.String(),
				individualProducerReward.String(), individualBlockConfirmReward.String(), rewardPerVote)
		}
		roundReward[rewardHash] += r
		realDPOSReward += r
		log.Debugf("distributeWithNormalArbitratorsV3 rewardHash%s  r %s realDPOSReward %s", rewardHash.String(),
			r.String(), realDPOSReward.String())
	}

	for _, candidate := range a.CurrentCandidates {
		ownerHash := candidate.GetOwnerProgramHash()
		votes := a.CurrentReward.OwnerVotesInRound[ownerHash]
		individualProducerReward := common.Fixed64(math.Floor(float64(
			votes) * rewardPerVote))
		roundReward[ownerHash] = individualProducerReward
		log.Debugf("distributeWithNormalArbitratorsV3 ownerHash%s  individualProducerReward %s realDPOSReward %s",
			ownerHash.String(), individualProducerReward.String(), realDPOSReward.String())
		realDPOSReward += individualProducerReward
	}
	// Abnormal CR`s reward need to be destroyed.
	for i := len(a.CurrentArbitrators); i < arbitersCount; i++ {
		roundReward[a.ChainParams.DestroyELAAddress] += individualBlockConfirmReward
	}
	return roundReward, realDPOSReward, nil
}

func (a *Arbiters) distributeWithNormalArbitratorsV2(height uint32, reward common.Fixed64) (
	map[common.Uint168]common.Fixed64, common.Fixed64, error) {
	//if len(a.CurrentArbitrators) == 0 {
	//	return nil, 0, errors.New("not found arbiters when " +
	//		"distributeWithNormalArbitratorsV2")
	//}

	roundReward := map[common.Uint168]common.Fixed64{}
	totalBlockConfirmReward := float64(reward) * 0.25
	totalTopProducersReward := float64(reward) - totalBlockConfirmReward
	// Consider that there is no only CR consensus.
	arbitersCount := len(a.ChainParams.CRCArbiters) + a.ChainParams.GeneralArbiters
	individualBlockConfirmReward := common.Fixed64(
		math.Floor(totalBlockConfirmReward / float64(arbitersCount)))
	totalVotesInRound := a.CurrentReward.TotalVotesInRound
	if len(a.CurrentArbitrators) == 0 ||
		len(a.ChainParams.CRCArbiters) == len(a.CurrentArbitrators) {
		//if len(a.ChainParams.CRCArbiters) == len(a.CurrentArbitrators) {
		// if no normal DPOS node, need to destroy reward.
		roundReward[a.ChainParams.DestroyELAAddress] = reward
		return roundReward, reward, nil
	}
	rewardPerVote := totalTopProducersReward / float64(totalVotesInRound)

	realDPOSReward := common.Fixed64(0)
	for _, arbiter := range a.CurrentArbitrators {
		ownerHash := arbiter.GetOwnerProgramHash()
		rewardHash := ownerHash
		var r common.Fixed64
		if _, ok := a.CurrentCRCArbitersMap[ownerHash]; ok {
			r = individualBlockConfirmReward
			m, ok := arbiter.(*crcArbiter)
			if !ok || m.crMember.MemberState != state.MemberElected || len(m.crMember.DPOSPublicKey) == 0 {
				rewardHash = a.ChainParams.DestroyELAAddress
			} else {
				pk := arbiter.GetOwnerPublicKey()
				programHash, err := contract.PublicKeyToStandardProgramHash(pk)
				if err != nil {
					rewardHash = a.ChainParams.DestroyELAAddress
				} else {
					rewardHash = *programHash
				}
			}
		} else {
			votes := a.CurrentReward.OwnerVotesInRound[ownerHash]
			individualProducerReward := common.Fixed64(math.Floor(float64(
				votes) * rewardPerVote))
			r = individualBlockConfirmReward + individualProducerReward
		}
		roundReward[rewardHash] += r
		realDPOSReward += r
	}
	for _, candidate := range a.CurrentCandidates {
		ownerHash := candidate.GetOwnerProgramHash()
		votes := a.CurrentReward.OwnerVotesInRound[ownerHash]
		individualProducerReward := common.Fixed64(math.Floor(float64(
			votes) * rewardPerVote))
		roundReward[ownerHash] = individualProducerReward

		realDPOSReward += individualProducerReward
	}
	// Abnormal CR`s reward need to be destroyed.
	for i := len(a.CurrentArbitrators); i < arbitersCount; i++ {
		roundReward[a.ChainParams.DestroyELAAddress] += individualBlockConfirmReward
	}
	return roundReward, realDPOSReward, nil
}

func (a *Arbiters) distributeWithNormalArbitratorsV1(height uint32, reward common.Fixed64) (
	map[common.Uint168]common.Fixed64, common.Fixed64, error) {
	if len(a.CurrentArbitrators) == 0 {
		return nil, 0, errors.New("not found arbiters when " +
			"distributeWithNormalArbitratorsV1")
	}

	roundReward := map[common.Uint168]common.Fixed64{}
	totalBlockConfirmReward := float64(reward) * 0.25
	totalTopProducersReward := float64(reward) - totalBlockConfirmReward
	individualBlockConfirmReward := common.Fixed64(
		math.Floor(totalBlockConfirmReward / float64(len(a.CurrentArbitrators))))
	totalVotesInRound := a.CurrentReward.TotalVotesInRound
	if len(a.ChainParams.CRCArbiters) == len(a.CurrentArbitrators) {
		roundReward[a.ChainParams.CRCAddress] = reward
		return roundReward, reward, nil
	}
	rewardPerVote := totalTopProducersReward / float64(totalVotesInRound)
	realDPOSReward := common.Fixed64(0)
	for _, arbiter := range a.CurrentArbitrators {
		ownerHash := arbiter.GetOwnerProgramHash()
		rewardHash := ownerHash
		var r common.Fixed64
		if _, ok := a.CurrentCRCArbitersMap[ownerHash]; ok {
			r = individualBlockConfirmReward
			m, ok := arbiter.(*crcArbiter)
			if !ok || m.crMember.MemberState != state.MemberElected {
				rewardHash = a.ChainParams.DestroyELAAddress
			} else {
				pk := arbiter.GetOwnerPublicKey()
				programHash, err := contract.PublicKeyToStandardProgramHash(pk)
				if err != nil {
					rewardHash = a.ChainParams.DestroyELAAddress
				} else {
					rewardHash = *programHash
				}
			}
		} else {
			votes := a.CurrentReward.OwnerVotesInRound[ownerHash]
			individualProducerReward := common.Fixed64(math.Floor(float64(
				votes) * rewardPerVote))
			r = individualBlockConfirmReward + individualProducerReward
		}
		roundReward[rewardHash] += r
		realDPOSReward += r
	}
	for _, candidate := range a.CurrentCandidates {
		ownerHash := candidate.GetOwnerProgramHash()
		votes := a.CurrentReward.OwnerVotesInRound[ownerHash]
		individualProducerReward := common.Fixed64(math.Floor(float64(
			votes) * rewardPerVote))
		roundReward[ownerHash] = individualProducerReward

		realDPOSReward += individualProducerReward
	}
	return roundReward, realDPOSReward, nil
}

func (a *Arbiters) GetCurrentNeedConnectArbiters() []peer.PID {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	return a.getNeedConnectArbiters()
}

func (a *Arbiters) getCurrentNeedConnectArbiters() []peer.PID {
	height := a.History.Height() + 1
	if height < a.ChainParams.CRCOnlyDPOSHeight-a.ChainParams.PreConnectOffset {
		return nil
	}

	pids := make(map[string]peer.PID)
	for _, p := range a.CurrentCRCArbitersMap {
		abt, ok := p.(*crcArbiter)
		if !ok || abt.crMember.MemberState != state.MemberElected {
			continue
		}
		var pid peer.PID
		copy(pid[:], p.GetNodePublicKey())
		pids[common.BytesToHexString(p.GetNodePublicKey())] = pid
	}

	for _, v := range a.CurrentArbitrators {
		key := common.BytesToHexString(v.GetNodePublicKey())
		var pid peer.PID
		copy(pid[:], v.GetNodePublicKey())
		pids[key] = pid
	}

	peers := make([]peer.PID, 0, len(pids))
	for _, pid := range pids {
		peers = append(peers, pid)
	}

	return peers
}
func (a *Arbiters) GetNextNeedConnectArbiters() []peer.PID {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	return a.getNeedConnectArbiters()
}

func (a *Arbiters) getNextNeedConnectArbiters() []peer.PID {
	height := a.History.Height() + 1
	if height < a.ChainParams.CRCOnlyDPOSHeight-a.ChainParams.PreConnectOffset {
		return nil
	}

	pids := make(map[string]peer.PID)
	for _, p := range a.nextCRCArbitersMap {
		abt, ok := p.(*crcArbiter)
		if !ok || abt.crMember.MemberState != state.MemberElected {
			continue
		}
		var pid peer.PID
		copy(pid[:], p.GetNodePublicKey())
		pids[common.BytesToHexString(p.GetNodePublicKey())] = pid
	}

	for _, v := range a.nextArbitrators {
		key := common.BytesToHexString(v.GetNodePublicKey())
		var pid peer.PID
		copy(pid[:], v.GetNodePublicKey())
		pids[key] = pid
	}

	peers := make([]peer.PID, 0, len(pids))
	for _, pid := range pids {
		peers = append(peers, pid)
	}

	return peers
}

func (a *Arbiters) GetNeedConnectArbiters() []peer.PID {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	return a.getNeedConnectArbiters()
}

func (a *Arbiters) getNeedConnectArbiters() []peer.PID {
	height := a.History.Height() + 1
	if height < a.ChainParams.CRCOnlyDPOSHeight-a.ChainParams.PreConnectOffset {
		return nil
	}

	pids := make(map[string]peer.PID)
	for _, p := range a.CurrentCRCArbitersMap {
		abt, ok := p.(*crcArbiter)
		if !ok || abt.crMember.MemberState != state.MemberElected {
			continue
		}
		var pid peer.PID
		copy(pid[:], p.GetNodePublicKey())
		pids[common.BytesToHexString(p.GetNodePublicKey())] = pid
	}

	for _, p := range a.nextCRCArbitersMap {
		abt, ok := p.(*crcArbiter)
		if !ok || abt.crMember.MemberState != state.MemberElected {
			continue
		}
		var pid peer.PID
		copy(pid[:], p.GetNodePublicKey())
		pids[common.BytesToHexString(p.GetNodePublicKey())] = pid
	}

	if height != a.ChainParams.CRCOnlyDPOSHeight-
		a.ChainParams.PreConnectOffset {
		for _, v := range a.CurrentArbitrators {
			key := common.BytesToHexString(v.GetNodePublicKey())
			var pid peer.PID
			copy(pid[:], v.GetNodePublicKey())
			pids[key] = pid
		}
	}

	for _, v := range a.nextArbitrators {
		key := common.BytesToHexString(v.GetNodePublicKey())
		var pid peer.PID
		copy(pid[:], v.GetNodePublicKey())
		pids[key] = pid
	}

	peers := make([]peer.PID, 0, len(pids))
	for _, pid := range pids {
		peers = append(peers, pid)
	}

	return peers
}

func (a *Arbiters) IsArbitrator(pk []byte) bool {
	arbitrators := a.GetArbitrators()

	for _, v := range arbitrators {
		if !v.IsNormal {
			continue
		}
		if bytes.Equal(pk, v.NodePublicKey) {
			return true
		}
	}
	return false
}

func (a *Arbiters) GetArbitrators() []*ArbiterInfo {
	a.mtx.Lock()
	result := a.getArbitrators()
	a.mtx.Unlock()

	return result
}

func (a *Arbiters) GetCurrentArbitratorKeys() [][]byte {
	var ret [][]byte
	for _, info := range a.getArbitrators() {
		ret = append(ret, info.NodePublicKey)
	}
	return ret
}

func (a *Arbiters) getArbitrators() []*ArbiterInfo {
	result := make([]*ArbiterInfo, 0, len(a.CurrentArbitrators))
	for _, v := range a.CurrentArbitrators {
		isNormal := true
		isCRMember := false
		claimedDPOSNode := false
		abt, ok := v.(*crcArbiter)
		if ok {
			isCRMember = true
			if !abt.isNormal {
				isNormal = false
			}
			if len(abt.crMember.DPOSPublicKey) != 0 {
				claimedDPOSNode = true
			}
		}
		result = append(result, &ArbiterInfo{
			NodePublicKey:   v.GetNodePublicKey(),
			IsNormal:        isNormal,
			IsCRMember:      isCRMember,
			ClaimedDPOSNode: claimedDPOSNode,
		})
	}
	return result
}

func (a *Arbiters) GetCandidates() [][]byte {
	a.mtx.Lock()
	result := make([][]byte, 0, len(a.CurrentCandidates))
	for _, v := range a.CurrentCandidates {
		result = append(result, v.GetNodePublicKey())
	}
	a.mtx.Unlock()

	return result
}

func (a *Arbiters) GetNextArbitrators() []*ArbiterInfo {
	a.mtx.Lock()
	result := make([]*ArbiterInfo, 0, len(a.nextArbitrators))
	for _, v := range a.nextArbitrators {
		isNormal := true
		isCRMember := false
		claimedDPOSNode := false
		abt, ok := v.(*crcArbiter)
		if ok {
			isCRMember = true
			if !abt.isNormal {
				isNormal = false
			}
			if len(abt.crMember.DPOSPublicKey) != 0 {
				claimedDPOSNode = true
			}
		}
		result = append(result, &ArbiterInfo{
			NodePublicKey:   v.GetNodePublicKey(),
			IsNormal:        isNormal,
			IsCRMember:      isCRMember,
			ClaimedDPOSNode: claimedDPOSNode,
		})
	}
	a.mtx.Unlock()

	return result
}

func (a *Arbiters) GetNextCandidates() [][]byte {
	a.mtx.Lock()
	result := make([][]byte, 0, len(a.nextCandidates))
	for _, v := range a.nextCandidates {
		result = append(result, v.GetNodePublicKey())
	}
	a.mtx.Unlock()

	return result
}

func (a *Arbiters) GetCRCArbiters() []*ArbiterInfo {
	a.mtx.Lock()
	result := a.getCRCArbiters()
	a.mtx.Unlock()

	return result
}

func (a *Arbiters) getCRCArbiters() []*ArbiterInfo {
	result := make([]*ArbiterInfo, 0, len(a.CurrentCRCArbitersMap))
	for _, v := range a.CurrentCRCArbitersMap {
		isNormal := true
		isCRMember := false
		claimedDPOSNode := false
		abt, ok := v.(*crcArbiter)
		if ok {
			isCRMember = true
			if !abt.isNormal {
				isNormal = false
			}
			if len(abt.crMember.DPOSPublicKey) != 0 {
				claimedDPOSNode = true
			}
		}
		result = append(result, &ArbiterInfo{
			NodePublicKey:   v.GetNodePublicKey(),
			IsNormal:        isNormal,
			IsCRMember:      isCRMember,
			ClaimedDPOSNode: claimedDPOSNode,
		})
	}

	return result
}

func (a *Arbiters) GetAllNextCRCArbiters() [][]byte {
	a.mtx.Lock()
	result := make([][]byte, 0, len(a.nextCRCArbiters))
	for _, v := range a.nextCRCArbiters {
		result = append(result, v.GetNodePublicKey())
	}
	a.mtx.Unlock()

	return result
}

func (a *Arbiters) GetNextCRCArbiters() [][]byte {
	a.mtx.Lock()
	result := make([][]byte, 0, len(a.nextCRCArbiters))
	for _, v := range a.nextCRCArbiters {
		if !v.IsNormal() {
			continue
		}
		result = append(result, v.GetNodePublicKey())
	}
	a.mtx.Unlock()

	return result
}

func (a *Arbiters) GetCurrentRewardData() RewardData {
	a.mtx.Lock()
	result := a.CurrentReward
	a.mtx.Unlock()

	return result
}

func (a *Arbiters) GetNextRewardData() RewardData {
	a.mtx.Lock()
	result := a.NextReward
	a.mtx.Unlock()

	return result
}

func (a *Arbiters) IsCRCArbitrator(pk []byte) bool {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	for _, v := range a.CurrentCRCArbitersMap {
		if bytes.Equal(v.GetNodePublicKey(), pk) {
			return true
		}
	}
	return false
}

func (a *Arbiters) isNextCRCArbitrator(pk []byte) bool {
	for _, v := range a.nextCRCArbitersMap {
		if bytes.Equal(v.GetNodePublicKey(), pk) {
			return true
		}
	}
	return false
}

func (a *Arbiters) IsNextCRCArbitrator(pk []byte) bool {
	for _, v := range a.nextCRCArbiters {
		if bytes.Equal(v.GetNodePublicKey(), pk) {
			return true
		}
	}
	return false
}

func (a *Arbiters) IsMemberElectedNextCRCArbitrator(pk []byte) bool {
	for _, v := range a.nextCRCArbiters {
		if bytes.Equal(v.GetNodePublicKey(), pk) && v.(*crcArbiter).crMember.MemberState == state.MemberElected {
			return true
		}
	}
	return false
}

func (a *Arbiters) IsActiveProducer(pk []byte) bool {
	return a.State.IsActiveProducer(pk)
}

func (a *Arbiters) IsDisabledProducer(pk []byte) bool {
	return a.State.IsInactiveProducer(pk) || a.State.IsIllegalProducer(pk) || a.State.IsCanceledProducer(pk)
}

func (a *Arbiters) GetConnectedProducer(publicKey []byte) ArbiterMember {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	for _, v := range a.CurrentCRCArbitersMap {
		if bytes.Equal(v.GetNodePublicKey(), publicKey) {
			return v
		}
	}

	for _, v := range a.nextCRCArbitersMap {
		if bytes.Equal(v.GetNodePublicKey(), publicKey) {
			return v
		}
	}

	findByPk := func(arbiters []ArbiterMember) ArbiterMember {
		for _, v := range arbiters {
			if bytes.Equal(v.GetNodePublicKey(), publicKey) {
				return v
			}
		}
		return nil
	}
	if ar := findByPk(a.CurrentArbitrators); ar != nil {
		return ar
	}
	if ar := findByPk(a.CurrentCandidates); ar != nil {
		return ar
	}
	if ar := findByPk(a.nextArbitrators); ar != nil {
		return ar
	}
	if ar := findByPk(a.nextCandidates); ar != nil {
		return ar
	}

	return nil
}

func (a *Arbiters) CRCProducerCount() int {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	return len(a.CurrentCRCArbitersMap)
}

func (a *Arbiters) getOnDutyArbitrator() []byte {
	return a.getNextOnDutyArbitratorV(a.bestHeight()+1, 0).GetNodePublicKey()
}

func (a *Arbiters) GetOnDutyArbitrator() []byte {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	arbiter := a.getNextOnDutyArbitratorV(a.bestHeight()+1, 0)
	if arbiter != nil && arbiter.IsNormal() {
		return arbiter.GetNodePublicKey()
	}
	return []byte{}
}

func (a *Arbiters) GetNextOnDutyArbitrator(offset uint32) []byte {
	arbiter := a.getNextOnDutyArbitratorV(a.bestHeight()+1, offset)
	if arbiter == nil {
		return []byte{}
	}
	return arbiter.GetNodePublicKey()
}

func (a *Arbiters) GetOnDutyCrossChainArbitrator() []byte {
	var arbiter []byte
	height := a.bestHeight()
	if height < a.ChainParams.CRCOnlyDPOSHeight-1 {
		arbiter = a.GetOnDutyArbitrator()
	} else if height < a.ChainParams.CRClaimDPOSNodeStartHeight {
		a.mtx.Lock()
		crcArbiters := a.getCRCArbiters()
		sort.Slice(crcArbiters, func(i, j int) bool {
			return bytes.Compare(crcArbiters[i].NodePublicKey, crcArbiters[j].NodePublicKey) < 0
		})
		ondutyIndex := int(height-a.ChainParams.CRCOnlyDPOSHeight+1) % len(crcArbiters)
		arbiter = crcArbiters[ondutyIndex].NodePublicKey
		a.mtx.Unlock()
	} else if height < a.ChainParams.DPOSNodeCrossChainHeight {
		a.mtx.Lock()
		crcArbiters := a.getCRCArbiters()
		sort.Slice(crcArbiters, func(i, j int) bool {
			return bytes.Compare(crcArbiters[i].NodePublicKey,
				crcArbiters[j].NodePublicKey) < 0
		})
		index := a.DutyIndex % len(a.CurrentCRCArbitersMap)
		if crcArbiters[index].IsNormal {
			arbiter = crcArbiters[index].NodePublicKey
		} else {
			arbiter = nil
		}
		a.mtx.Unlock()
	} else {
		a.mtx.Lock()
		if len(a.CurrentArbitrators) != 0 && a.CurrentArbitrators[a.DutyIndex].IsNormal() {
			arbiter = a.CurrentArbitrators[a.DutyIndex].GetNodePublicKey()
		} else {
			arbiter = nil
		}
		a.mtx.Unlock()
	}

	return arbiter
}

func (a *Arbiters) GetCrossChainArbiters() []*ArbiterInfo {
	bestHeight := a.bestHeight()
	if bestHeight < a.ChainParams.CRCOnlyDPOSHeight-1 {
		return a.GetArbitrators()
	}
	if bestHeight < a.ChainParams.DPOSNodeCrossChainHeight {
		crcArbiters := a.GetCRCArbiters()
		sort.Slice(crcArbiters, func(i, j int) bool {
			return bytes.Compare(crcArbiters[i].NodePublicKey, crcArbiters[j].NodePublicKey) < 0
		})
		return crcArbiters
	}

	return a.GetArbitrators()
}

func (a *Arbiters) GetCrossChainArbitersCount() int {
	if a.bestHeight() < a.ChainParams.CRCOnlyDPOSHeight-1 {
		return len(a.ChainParams.OriginArbiters)
	}

	return len(a.ChainParams.CRCArbiters)
}

func (a *Arbiters) GetCrossChainArbitersMajorityCount() int {
	minSignCount := int(float64(a.GetCrossChainArbitersCount()) *
		MajoritySignRatioNumerator / MajoritySignRatioDenominator)
	return minSignCount
}

func (a *Arbiters) getNextOnDutyArbitratorV(height, offset uint32) ArbiterMember {
	// main version is >= H1
	if height >= a.ChainParams.CRCOnlyDPOSHeight {
		arbitrators := a.CurrentArbitrators
		if len(arbitrators) == 0 {
			return nil
		}
		index := (a.DutyIndex + int(offset)) % len(arbitrators)
		arbiter := arbitrators[index]

		return arbiter
	}

	// old version
	return a.getNextOnDutyArbitratorV0(height, offset)
}

func (a *Arbiters) GetArbitersCount() int {
	a.mtx.Lock()
	result := len(a.CurrentArbitrators)
	if result == 0 {
		result = a.ChainParams.GeneralArbiters + len(a.ChainParams.CRCArbiters)
	}
	a.mtx.Unlock()
	return result
}

func (a *Arbiters) GetCRCArbitersCount() int {
	a.mtx.Lock()
	result := len(a.CurrentCRCArbitersMap)
	a.mtx.Unlock()
	return result
}

func (a *Arbiters) GetArbitersMajorityCount() int {
	a.mtx.Lock()
	var currentArbitratorsCount int
	if len(a.CurrentArbitrators) != 0 {
		currentArbitratorsCount = len(a.CurrentArbitrators)
	} else {
		currentArbitratorsCount = len(a.ChainParams.CRCArbiters) + a.ChainParams.GeneralArbiters
	}
	minSignCount := int(float64(currentArbitratorsCount) *
		MajoritySignRatioNumerator / MajoritySignRatioDenominator)
	a.mtx.Unlock()
	return minSignCount
}

func (a *Arbiters) HasArbitersMajorityCount(num int) bool {
	return num > a.GetArbitersMajorityCount()
}

func (a *Arbiters) HasArbitersMinorityCount(num int) bool {
	a.mtx.Lock()
	count := len(a.CurrentArbitrators)
	a.mtx.Unlock()
	return num >= count-a.GetArbitersMajorityCount()
}

func (a *Arbiters) HasArbitersHalfMinorityCount(num int) bool {
	a.mtx.Lock()
	count := len(a.CurrentArbitrators)
	a.mtx.Unlock()
	return num >= (count-a.GetArbitersMajorityCount())/2
}

func (a *Arbiters) getChangeType(height uint32) (ChangeType, uint32) {

	// special change points:
	//		H1 - PreConnectOffset -> 	[updateNext, H1]: update next arbiters and let CRC arbiters prepare to connect
	//		H1 -> 						[normalChange, H1]: should change to new election (that only have CRC arbiters)
	//		H2 - PreConnectOffset -> 	[updateNext, H2]: update next arbiters and let normal arbiters prepare to connect
	//		H2 -> 						[normalChange, H2]: should change to new election (arbiters will have both CRC and normal arbiters)
	if height == a.ChainParams.CRCOnlyDPOSHeight-
		a.ChainParams.PreConnectOffset {
		return updateNext, a.ChainParams.CRCOnlyDPOSHeight
	} else if height == a.ChainParams.CRCOnlyDPOSHeight {
		return normalChange, a.ChainParams.CRCOnlyDPOSHeight
	} else if height == a.ChainParams.PublicDPOSHeight-
		a.ChainParams.PreConnectOffset {
		return updateNext, a.ChainParams.PublicDPOSHeight
	} else if height == a.ChainParams.PublicDPOSHeight {
		return normalChange, a.ChainParams.PublicDPOSHeight
	}

	// main version >= H2
	if height > a.ChainParams.PublicDPOSHeight &&
		a.DutyIndex == len(a.CurrentArbitrators)-1 {
		return normalChange, height
	}

	if height > a.ChainParams.RevertToPOWStartHeight &&
		a.DutyIndex == len(a.ChainParams.CRCArbiters)+a.ChainParams.GeneralArbiters-1 {
		return normalChange, height
	}

	return none, height
}

func (a *Arbiters) cleanArbitrators(height uint32) {
	oriCurrentCRCArbitersMap := copyCRCArbitersMap(a.CurrentCRCArbitersMap)
	oriCurrentArbitrators := a.CurrentArbitrators
	oriCurrentCandidates := a.CurrentCandidates
	oriNextCRCArbitersMap := copyCRCArbitersMap(a.nextCRCArbitersMap)
	oriNextArbitrators := a.nextArbitrators
	oriNextCandidates := a.nextCandidates
	oriDutyIndex := a.DutyIndex
	a.History.Append(height, func() {
		a.CurrentCRCArbitersMap = make(map[common.Uint168]ArbiterMember)
		a.CurrentArbitrators = make([]ArbiterMember, 0)
		a.CurrentCandidates = make([]ArbiterMember, 0)
		a.nextCRCArbitersMap = make(map[common.Uint168]ArbiterMember)
		a.nextArbitrators = make([]ArbiterMember, 0)
		a.nextCandidates = make([]ArbiterMember, 0)
		a.DutyIndex = 0
	}, func() {
		a.CurrentCRCArbitersMap = oriCurrentCRCArbitersMap
		a.CurrentArbitrators = oriCurrentArbitrators
		a.CurrentCandidates = oriCurrentCandidates
		a.nextCRCArbitersMap = oriNextCRCArbitersMap
		a.nextArbitrators = oriNextArbitrators
		a.nextCandidates = oriNextCandidates
		a.DutyIndex = oriDutyIndex
	})
}

func (a *Arbiters) ChangeCurrentArbitrators(height uint32) error {
	oriCurrentCRCArbitersMap := copyCRCArbitersMap(a.CurrentCRCArbitersMap)
	oriCurrentArbitrators := a.CurrentArbitrators
	oriCurrentCandidates := a.CurrentCandidates
	oriCurrentReward := a.CurrentReward
	oriDutyIndex := a.DutyIndex
	a.History.Append(height, func() {
		sort.Slice(a.nextArbitrators, func(i, j int) bool {
			return bytes.Compare(a.nextArbitrators[i].GetNodePublicKey(),
				a.nextArbitrators[j].GetNodePublicKey()) < 0
		})
		a.CurrentCRCArbitersMap = copyCRCArbitersMap(a.nextCRCArbitersMap)
		a.CurrentArbitrators = a.nextArbitrators
		a.CurrentCandidates = a.nextCandidates
		a.CurrentReward = a.NextReward
		a.DutyIndex = 0
	}, func() {
		a.CurrentCRCArbitersMap = oriCurrentCRCArbitersMap
		a.CurrentArbitrators = oriCurrentArbitrators
		a.CurrentCandidates = oriCurrentCandidates
		a.CurrentReward = oriCurrentReward
		a.DutyIndex = oriDutyIndex
	})
	return nil
}

func (a *Arbiters) IsSameWithNextArbitrators() bool {

	if len(a.nextArbitrators) != len(a.CurrentArbitrators) {
		return false
	}
	for index, v := range a.CurrentArbitrators {
		if bytes.Equal(v.GetNodePublicKey(), a.nextArbitrators[index].GetNodePublicKey()) {
			return false
		}
	}
	return true
}

func (a *Arbiters) ConvertToArbitersStr(arbiters [][]byte) []string {
	var arbitersStr []string
	for _, v := range arbiters {
		arbitersStr = append(arbitersStr, common.BytesToHexString(v))
	}
	return arbitersStr
}

func (a *Arbiters) createNextTurnDPOSInfoTransactionV0(blockHeight uint32, forceChange bool) interfaces.Transaction {

	var nextTurnDPOSInfo payload.NextTurnDPOSInfo
	nextTurnDPOSInfo.CRPublicKeys = make([][]byte, 0)
	nextTurnDPOSInfo.DPOSPublicKeys = make([][]byte, 0)
	var workingHeight uint32
	if forceChange {
		workingHeight = blockHeight
	} else {
		workingHeight = blockHeight + uint32(a.ChainParams.GeneralArbiters+len(a.ChainParams.CRCArbiters))
	}
	nextTurnDPOSInfo.WorkingHeight = workingHeight
	for _, v := range a.nextArbitrators {
		if a.isNextCRCArbitrator(v.GetNodePublicKey()) {
			if abt, ok := v.(*crcArbiter); ok && abt.crMember.MemberState != state.MemberElected {
				nextTurnDPOSInfo.CRPublicKeys = append(nextTurnDPOSInfo.CRPublicKeys, []byte{})
			} else {
				nextTurnDPOSInfo.CRPublicKeys = append(nextTurnDPOSInfo.CRPublicKeys, v.GetNodePublicKey())
			}
		} else {
			nextTurnDPOSInfo.DPOSPublicKeys = append(nextTurnDPOSInfo.DPOSPublicKeys, v.GetNodePublicKey())
		}
	}

	log.Debugf("[createNextTurnDPOSInfoTransaction] CRPublicKeys %v, DPOSPublicKeys%v\n",
		a.ConvertToArbitersStr(nextTurnDPOSInfo.CRPublicKeys), a.ConvertToArbitersStr(nextTurnDPOSInfo.DPOSPublicKeys))

	return functions.CreateTransaction(
		common2.TxVersion09,
		common2.NextTurnDPOSInfo,
		0,
		&nextTurnDPOSInfo,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
}

func (a *Arbiters) createNextTurnDPOSInfoTransactionV1(blockHeight uint32, forceChange bool) interfaces.Transaction {

	var nextTurnDPOSInfo payload.NextTurnDPOSInfo
	nextTurnDPOSInfo.CRPublicKeys = make([][]byte, 0)
	nextTurnDPOSInfo.DPOSPublicKeys = make([][]byte, 0)
	var workingHeight uint32
	if forceChange {
		workingHeight = blockHeight
	} else {
		workingHeight = blockHeight + uint32(a.ChainParams.GeneralArbiters+len(a.ChainParams.CRCArbiters))
	}
	nextTurnDPOSInfo.WorkingHeight = workingHeight
	for _, v := range a.nextCRCArbiters {
		nodePK := v.GetNodePublicKey()
		if v.IsNormal() {
			nextTurnDPOSInfo.CRPublicKeys = append(nextTurnDPOSInfo.CRPublicKeys, nodePK)
		} else {
			nextTurnDPOSInfo.CRPublicKeys = append(nextTurnDPOSInfo.CRPublicKeys, []byte{})
		}
	}
	for _, v := range a.nextArbitrators {
		if a.isNextCRCArbitrator(v.GetNodePublicKey()) {
			if abt, ok := v.(*crcArbiter); ok && abt.crMember.MemberState != state.MemberElected {
				nextTurnDPOSInfo.DPOSPublicKeys = append(nextTurnDPOSInfo.DPOSPublicKeys, []byte{})
			} else {
				nextTurnDPOSInfo.DPOSPublicKeys = append(nextTurnDPOSInfo.DPOSPublicKeys, v.GetNodePublicKey())
			}
		} else {
			nextTurnDPOSInfo.DPOSPublicKeys = append(nextTurnDPOSInfo.DPOSPublicKeys, v.GetNodePublicKey())
		}
	}

	log.Debugf("[createNextTurnDPOSInfoTransaction] CRPublicKeys %v, DPOSPublicKeys%v\n",
		a.ConvertToArbitersStr(nextTurnDPOSInfo.CRPublicKeys), a.ConvertToArbitersStr(nextTurnDPOSInfo.DPOSPublicKeys))

	return functions.CreateTransaction(
		common2.TxVersion09,
		common2.NextTurnDPOSInfo,
		0,
		&nextTurnDPOSInfo,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
}

func (a *Arbiters) updateNextTurnInfo(height uint32, producers []ArbiterMember, unclaimed int) {
	nextCRCArbiters := a.nextArbitrators
	if !a.isDposV2Active() {
		a.nextArbitrators = append(a.nextArbitrators, producers...)
	} else {
		a.nextArbitrators = producers
	}
	sort.Slice(a.nextArbitrators, func(i, j int) bool {
		return bytes.Compare(a.nextArbitrators[i].GetNodePublicKey(), a.nextArbitrators[j].GetNodePublicKey()) < 0
	})
	if height >= a.ChainParams.CRClaimDPOSNodeStartHeight {
		//need sent a NextTurnDPOSInfo tx into mempool
		sort.Slice(nextCRCArbiters, func(i, j int) bool {
			return bytes.Compare(nextCRCArbiters[i].GetNodePublicKey(), nextCRCArbiters[j].GetNodePublicKey()) < 0
		})
		a.nextCRCArbiters = copyByteList(nextCRCArbiters)
	}
}

func (a *Arbiters) getProducers(count int, height uint32) ([]ArbiterMember, error) {
	if !a.isDPoSV2Run(height) {
		return a.GetNormalArbitratorsDesc(height, count,
			a.getSortedProducers(), 0)
	} else {
		return a.GetNormalArbitratorsDesc(height, count,
			a.getSortedProducersDposV2(), 0)
	}
}

func (a *Arbiters) getSortedProducers() []*Producer {
	votedProducers := a.State.GetVotedProducers()
	sort.Slice(votedProducers, func(i, j int) bool {
		if votedProducers[i].votes == votedProducers[j].votes {
			return bytes.Compare(votedProducers[i].info.NodePublicKey,
				votedProducers[j].NodePublicKey()) < 0
		}
		return votedProducers[i].Votes() > votedProducers[j].Votes()
	})

	return votedProducers
}

func (a *Arbiters) getSortedProducersDposV2() []*Producer {
	votedProducers := a.State.GetDposV2ActiveProducers()
	sort.Slice(votedProducers, func(i, j int) bool {
		if votedProducers[i].GetTotalDPoSV2VoteRights() == votedProducers[j].GetTotalDPoSV2VoteRights() {
			return bytes.Compare(votedProducers[i].info.NodePublicKey,
				votedProducers[j].NodePublicKey()) < 0
		}
		return votedProducers[i].GetTotalDPoSV2VoteRights() > votedProducers[j].GetTotalDPoSV2VoteRights()
	})

	return votedProducers
}

func (a *Arbiters) getSortedProducersWithRandom(height uint32, unclaimedCount int) ([]*Producer, error) {

	votedProducers := a.getSortedProducers()
	if height < a.ChainParams.NoCRCDPOSNodeHeight {
		return votedProducers, nil
	}

	// if the last random producer is not found or the poll ranked in the top
	// 23(may be 35) or the state is not active, need to get a candidate as
	// DPOS node at random.
	if a.LastRandomCandidateHeight != 0 &&
		height-a.LastRandomCandidateHeight < a.ChainParams.RandomCandidatePeriod {
		for i, p := range votedProducers {
			if common.BytesToHexString(p.info.OwnerPublicKey) == a.LastRandomCandidateOwner {
				if i < unclaimedCount+a.ChainParams.GeneralArbiters-1 || p.state != Active {
					// need get again at random.
					break
				}
				normalCount := a.ChainParams.GeneralArbiters - 1
				selectedCandidateIndex := i

				newProducers := make([]*Producer, 0, len(votedProducers))
				newProducers = append(newProducers, votedProducers[:unclaimedCount+normalCount]...)
				newProducers = append(newProducers, p)
				newProducers = append(newProducers, votedProducers[unclaimedCount+normalCount:selectedCandidateIndex]...)
				newProducers = append(newProducers, votedProducers[selectedCandidateIndex+1:]...)
				log.Info("no need random producer, current random producer:",
					p.info.NickName, "current height:", height)
				return newProducers, nil
			}
		}
	}

	candidateIndex, err := a.getCandidateIndexAtRandom(height, unclaimedCount, len(votedProducers))
	if err != nil {
		return nil, err
	}

	normalCount := a.ChainParams.GeneralArbiters - 1
	selectedCandidateIndex := unclaimedCount + normalCount + candidateIndex
	candidateProducer := votedProducers[selectedCandidateIndex]

	// todo need to use History?
	a.LastRandomCandidateHeight = height
	a.LastRandomCandidateOwner = common.BytesToHexString(candidateProducer.info.OwnerPublicKey)

	newProducers := make([]*Producer, 0, len(votedProducers))
	newProducers = append(newProducers, votedProducers[:unclaimedCount+normalCount]...)
	newProducers = append(newProducers, candidateProducer)
	newProducers = append(newProducers, votedProducers[unclaimedCount+normalCount:selectedCandidateIndex]...)
	newProducers = append(newProducers, votedProducers[selectedCandidateIndex+1:]...)

	log.Info("random producer, current random producer:",
		candidateProducer.info.NickName, "current height:", height)
	return newProducers, nil
}

func (a *Arbiters) getRandomDposV2Producers(height uint32, unclaimedCount int, choosingArbiters map[common.Uint168]ArbiterMember) ([]string, error) {
	block, _ := a.getBlockByHeight(height - 1)
	if block == nil {
		return nil, errors.New("block is not found")
	}
	var x = make([]byte, 8)
	blockHash := block.HashWithAux()
	copy(x, blockHash[24:])
	seed, _, ok := Readi64(x)
	if !ok {
		return nil, errors.New("invalid block hash")
	}
	r := rand.New(rand.NewSource(seed))

	votedProducers := a.getSortedProducersDposV2()
	// crc also need to be random selected
	var producerKeys []string
	for _, crc := range choosingArbiters {
		producerKeys = append(producerKeys, hex.EncodeToString(crc.GetOwnerPublicKey()))
	}
	sort.Slice(producerKeys, func(i, j int) bool {
		return strings.Compare(producerKeys[i], producerKeys[j]) < 0
	})
	for _, vp := range votedProducers[unclaimedCount:] {
		producerKeys = append(producerKeys, hex.EncodeToString(vp.info.OwnerPublicKey))
	}
	newProducers := make([]string, 0, len(producerKeys))
	normalCount := a.ChainParams.GeneralArbiters + len(a.ChainParams.CRCArbiters)

	for i := 0; i < normalCount; i++ {
		if len(producerKeys) == 0 {
			log.Warn("left producerKeys is 0")
			break
		}
		s := r.Intn(len(producerKeys))
		if len(producerKeys) == 1 && s == 0 {
			producerKeys = producerKeys[0:0]
		} else {
			newProducers = append(newProducers, producerKeys[s])
			tmpProducers := producerKeys[s+1:]
			producerKeys = producerKeys[0:s]
			producerKeys = append(producerKeys, tmpProducers...)
		}
	}

	for i := 0; i < len(producerKeys); i++ {
		newProducers = append(newProducers, producerKeys[i])
	}

	return newProducers, nil
}

func (a *Arbiters) getCandidateIndexAtRandom(height uint32, unclaimedCount, votedProducersCount int) (int, error) {
	block, _ := a.getBlockByHeight(height - 1)
	if block == nil {
		return 0, errors.New("block is not found")
	}
	var x = make([]byte, 8)
	blockHash := block.Hash()
	copy(x, blockHash[24:])
	seed, _, ok := Readi64(x)
	if !ok {
		return 0, errors.New("invalid block hash")
	}
	rand.Seed(seed)
	normalCount := a.ChainParams.GeneralArbiters - 1
	count := votedProducersCount - unclaimedCount - normalCount
	if count < 1 {
		return 0, errors.New("producers is not enough")
	}
	candidatesCount := minInt(count, a.ChainParams.CandidateArbiters+1)
	return rand.Intn(candidatesCount), nil
}

func (a *Arbiters) isDposV2Active() bool {
	return len(a.DposV2EffectedProducers) >= a.ChainParams.GeneralArbiters*3/2
}

func (a *Arbiters) UpdateNextArbitrators(versionHeight, height uint32) error {

	if height >= a.ChainParams.CRClaimDPOSNodeStartHeight {
		oriNeedNextTurnDPOSInfo := a.NeedNextTurnDPOSInfo
		a.History.Append(height, func() {
			a.NeedNextTurnDPOSInfo = true
		}, func() {
			a.NeedNextTurnDPOSInfo = oriNeedNextTurnDPOSInfo
		})
	}

	_, recover := a.InactiveModeSwitch(versionHeight, a.IsAbleToRecoverFromInactiveMode)
	if recover {
		a.LeaveEmergency(a.History, height)
	} else {
		a.TryLeaveUnderStaffed(a.IsAbleToRecoverFromUnderstaffedState)
	}

	if a.DPoSV2ActiveHeight == math.MaxUint32 && a.isDposV2Active() {
		oriHeight := height
		a.History.Append(height, func() {
			a.DPoSV2ActiveHeight = height + a.ChainParams.CRMemberCount + uint32(a.ChainParams.GeneralArbiters)
		}, func() {
			a.DPoSV2ActiveHeight = oriHeight
		})
	}

	unclaimed, choosingArbiters, err := a.resetNextArbiterByCRC(versionHeight, height)
	if err != nil {
		return err
	}

	if !a.IsInactiveMode() && !a.IsUnderstaffedMode() {

		count := a.ChainParams.GeneralArbiters
		var votedProducers []*Producer
		var votedProducersStr []string
		if a.isDposV2Active() {
			votedProducersStr, err = a.getRandomDposV2Producers(height, unclaimed, choosingArbiters)
			if err != nil {
				return err
			}
		} else {
			votedProducers, err = a.getSortedProducersWithRandom(height, unclaimed)
			if err != nil {
				return err
			}
		}
		var producers []ArbiterMember
		var err error
		if a.isDposV2Active() {
			producers, err = a.GetDposV2NormalArbitratorsDesc(count+int(a.ChainParams.CRMemberCount), votedProducersStr, choosingArbiters)
		} else {
			producers, err = a.GetNormalArbitratorsDesc(versionHeight, count,
				votedProducers, unclaimed)
		}
		if err != nil {
			if height > a.ChainParams.ChangeCommitteeNewCRHeight {
				return err
			}
			if err := a.tryHandleError(versionHeight, err); err != nil {
				return err
			}
			oriNextCandidates := a.nextCandidates
			oriNextArbitrators := a.nextArbitrators
			oriNextCRCArbiters := a.nextCRCArbiters
			a.History.Append(height, func() {
				a.nextCandidates = make([]ArbiterMember, 0)
				a.updateNextTurnInfo(height, producers, unclaimed)
			}, func() {
				a.nextCandidates = oriNextCandidates
				a.nextArbitrators = oriNextArbitrators
				a.nextCRCArbiters = oriNextCRCArbiters
			})
		} else {
			if !a.isDposV2Active() {
				if height >= a.ChainParams.NoCRCDPOSNodeHeight {
					count := len(a.ChainParams.CRCArbiters) + a.ChainParams.GeneralArbiters
					var newSelected bool
					for _, p := range votedProducers {
						producer := p
						ownerPK := common.BytesToHexString(producer.info.OwnerPublicKey)
						if ownerPK == a.LastRandomCandidateOwner &&
							height-a.LastRandomCandidateHeight == uint32(count) {
							newSelected = true
						}
					}
					if newSelected {
						for _, p := range votedProducers {
							producer := p
							if producer.selected {
								a.History.Append(height, func() {
									producer.selected = false
								}, func() {
									producer.selected = true
								})
							}
							ownerPK := common.BytesToHexString(producer.info.OwnerPublicKey)
							oriRandomInactiveCount := producer.randomCandidateInactiveCount
							if ownerPK == a.LastRandomCandidateOwner {
								a.History.Append(height, func() {
									producer.selected = true
									producer.randomCandidateInactiveCount = 0
								}, func() {
									producer.selected = false
									producer.randomCandidateInactiveCount = oriRandomInactiveCount
								})
							}
						}
					}
				}
			}

			oriNextArbitrators := a.nextArbitrators
			oriNextCRCArbiters := a.nextCRCArbiters
			a.History.Append(height, func() {
				a.updateNextTurnInfo(height, producers, unclaimed)
			}, func() {
				// next Arbiters will rollback in resetNextArbiterByCRC
				a.nextArbitrators = oriNextArbitrators
				a.nextCRCArbiters = oriNextCRCArbiters
			})

			var candidates []ArbiterMember
			if !a.isDposV2Active() {
				candidates, err = a.GetCandidatesDesc(versionHeight, count+unclaimed,
					votedProducers)
			} else {
				candidates, err = a.GetDposV2CandidatesDesc(count+int(a.ChainParams.CRMemberCount),
					votedProducersStr, choosingArbiters)
			}
			if err != nil {
				return err
			}
			oriNextCandidates := a.nextCandidates
			a.History.Append(height, func() {
				a.nextCandidates = candidates
			}, func() {
				a.nextCandidates = oriNextCandidates
			})
		}
	} else {
		oriNextCandidates := a.nextCandidates
		oriNextArbitrators := a.nextArbitrators
		oriNextCRCArbiters := a.nextCRCArbiters
		a.History.Append(height, func() {
			a.nextCandidates = make([]ArbiterMember, 0)
			a.updateNextTurnInfo(height, nil, unclaimed)
		}, func() {
			a.nextCandidates = oriNextCandidates
			a.nextArbitrators = oriNextArbitrators
			a.nextCRCArbiters = oriNextCRCArbiters
		})
	}
	return nil
}

func (a *Arbiters) resetNextArbiterByCRC(versionHeight uint32, height uint32) (int, map[common.Uint168]ArbiterMember, error) {
	var unclaimed int
	var needReset bool
	crcArbiters := map[common.Uint168]ArbiterMember{}
	if a.CRCommittee != nil && a.CRCommittee.IsInElectionPeriod() {
		if versionHeight >= a.ChainParams.CRClaimDPOSNodeStartHeight {
			var err error
			if versionHeight < a.ChainParams.ChangeCommitteeNewCRHeight {
				if crcArbiters, err = a.getCRCArbitersV1(height); err != nil {
					return unclaimed, nil, err
				}
			} else {
				if crcArbiters, unclaimed, err = a.getCRCArbitersV2(height); err != nil {
					return unclaimed, nil, err
				}
			}
		} else {
			var err error
			if crcArbiters, err = a.getCRCArbitersV0(); err != nil {
				return unclaimed, nil, err
			}
		}
		needReset = true
	} else if versionHeight >= a.ChainParams.ChangeCommitteeNewCRHeight {
		var votedProducers []*Producer
		if a.isDposV2Active() {
			log.Info("change to DPoS 2.0 at height:", height)
			votedProducers = a.State.GetDposV2ActiveProducers()
		} else {
			votedProducers = a.State.GetVotedProducers()
		}

		if len(votedProducers) < len(a.ChainParams.CRCArbiters) {
			return unclaimed, nil, errors.New("votedProducers less than CRCArbiters")
		}

		if a.isDposV2Active() {
			sort.Slice(votedProducers, func(i, j int) bool {
				if votedProducers[i].GetTotalDPoSV2VoteRights() == votedProducers[j].GetTotalDPoSV2VoteRights() {
					return bytes.Compare(votedProducers[i].info.NodePublicKey,
						votedProducers[j].NodePublicKey()) < 0
				}
				return votedProducers[i].GetTotalDPoSV2VoteRights() > votedProducers[j].GetTotalDPoSV2VoteRights()
			})
		} else {
			sort.Slice(votedProducers, func(i, j int) bool {
				if votedProducers[i].votes == votedProducers[j].votes {
					return bytes.Compare(votedProducers[i].info.NodePublicKey,
						votedProducers[j].NodePublicKey()) < 0
				}
				return votedProducers[i].Votes() > votedProducers[j].Votes()
			})
		}

		for i := 0; i < len(a.ChainParams.CRCArbiters); i++ {
			producer := votedProducers[i]
			ar, err := NewDPoSArbiter(producer)
			if err != nil {
				return unclaimed, nil, err
			}
			crcArbiters[ar.GetOwnerProgramHash()] = ar
		}
		unclaimed = len(a.ChainParams.CRCArbiters)
		needReset = true

	} else if versionHeight >= a.ChainParams.CRCommitteeStartHeight {
		for _, pk := range a.ChainParams.CRCArbiters {
			pubKey, err := hex.DecodeString(pk)
			if err != nil {
				return unclaimed, nil, err
			}
			producer := &Producer{ // here need crc NODE public key
				info: payload.ProducerInfo{
					OwnerPublicKey: pubKey,
					NodePublicKey:  pubKey,
				},
				activateRequestHeight: math.MaxUint32,
			}
			ar, err := NewDPoSArbiter(producer)
			if err != nil {
				return unclaimed, nil, err
			}
			crcArbiters[ar.GetOwnerProgramHash()] = ar
		}
		needReset = true
	}

	if needReset {
		oriNextArbitersMap := a.nextCRCArbitersMap
		oriCRCChangedHeight := a.crcChangedHeight
		a.History.Append(height, func() {
			a.nextCRCArbitersMap = crcArbiters
			a.crcChangedHeight = a.CRCommittee.LastCommitteeHeight
		}, func() {
			a.nextCRCArbitersMap = oriNextArbitersMap
			a.crcChangedHeight = oriCRCChangedHeight
		})
	}

	oriNextArbiters := a.nextArbitrators
	a.History.Append(height, func() {
		a.nextArbitrators = make([]ArbiterMember, 0)
		for _, v := range a.nextCRCArbitersMap {
			a.nextArbitrators = append(a.nextArbitrators, v)
		}
	}, func() {
		a.nextArbitrators = oriNextArbiters
	})

	if len(crcArbiters) == 0 {
		for _, v := range a.nextCRCArbiters {
			crcArbiters[v.GetOwnerProgramHash()] = v
		}
	}

	return unclaimed, crcArbiters, nil
}

func (a *Arbiters) getCRCArbitersV2(height uint32) (map[common.Uint168]ArbiterMember, int, error) {
	crMembers := a.CRCommittee.GetAllMembersCopy()
	if len(crMembers) != len(a.ChainParams.CRCArbiters) {
		return nil, 0, errors.New("CRC members count mismatch with CRC arbiters")
	}

	// get public key map
	crPublicKeysMap := make(map[string]struct{})
	for _, cr := range crMembers {
		if len(cr.DPOSPublicKey) != 0 {
			crPublicKeysMap[common.BytesToHexString(cr.DPOSPublicKey)] = struct{}{}
		}
	}
	arbitersPublicKeysMap := make(map[string]struct{})
	for _, ar := range a.ChainParams.CRCArbiters {
		arbitersPublicKeysMap[ar] = struct{}{}
	}

	// get unclaimed arbiter keys list
	unclaimedArbiterKeys := make([]string, 0)
	for k, _ := range arbitersPublicKeysMap {
		if _, ok := crPublicKeysMap[k]; !ok {
			unclaimedArbiterKeys = append(unclaimedArbiterKeys, k)
		}
	}
	sort.Slice(unclaimedArbiterKeys, func(i, j int) bool {
		return strings.Compare(unclaimedArbiterKeys[i], unclaimedArbiterKeys[j]) < 0
	})
	producers, err := a.getProducers(int(a.ChainParams.CRMemberCount), height)
	if err != nil {
		return nil, 0, err
	}
	var unclaimedCount int
	crcArbiters := map[common.Uint168]ArbiterMember{}
	claimHeight := a.ChainParams.CRClaimDPOSNodeStartHeight
	for _, cr := range crMembers {
		var pk []byte
		if len(cr.DPOSPublicKey) == 0 {
			if height >= a.ChainParams.CRDPoSNodeHotFixHeight {
				//if cr.MemberState != state.MemberElected {
				var err error
				pk, err = common.HexStringToBytes(unclaimedArbiterKeys[0])
				if err != nil {
					return nil, 0, err
				}
				unclaimedArbiterKeys = unclaimedArbiterKeys[1:]
				//} else {
				//	pk = producers[unclaimedCount].GetNodePublicKey()
				//	unclaimedCount++
				//}
			} else {
				if cr.MemberState != state.MemberElected {
					var err error
					pk, err = common.HexStringToBytes(unclaimedArbiterKeys[0])
					if err != nil {
						return nil, 0, err
					}
					unclaimedArbiterKeys = unclaimedArbiterKeys[1:]
				} else {
					pk = producers[unclaimedCount].GetNodePublicKey()
					unclaimedCount++
				}
			}
		} else {
			pk = cr.DPOSPublicKey
		}
		crPublicKey := cr.Info.Code[1 : len(cr.Info.Code)-1]
		isNormal := true
		if height >= claimHeight && cr.MemberState != state.MemberElected {
			isNormal = false
		}
		ar, err := NewCRCArbiter(pk, crPublicKey, cr, isNormal)
		if err != nil {
			return nil, 0, err
		}
		crcArbiters[ar.GetOwnerProgramHash()] = ar
	}

	return crcArbiters, unclaimedCount, nil
}

func (a *Arbiters) getCRCArbitersV1(height uint32) (map[common.Uint168]ArbiterMember, error) {
	crMembers := a.CRCommittee.GetAllMembersCopy()
	if len(crMembers) != len(a.ChainParams.CRCArbiters) {
		return nil, errors.New("CRC members count mismatch with CRC arbiters")
	}

	// get public key map
	crPublicKeysMap := make(map[string]struct{})
	for _, cr := range crMembers {
		if len(cr.DPOSPublicKey) != 0 {
			crPublicKeysMap[common.BytesToHexString(cr.DPOSPublicKey)] = struct{}{}
		}
	}
	arbitersPublicKeysMap := make(map[string]struct{})
	for _, ar := range a.ChainParams.CRCArbiters {
		arbitersPublicKeysMap[ar] = struct{}{}
	}

	// get unclaimed arbiter keys list
	unclaimedArbiterKeys := make([]string, 0)
	for k, _ := range arbitersPublicKeysMap {
		if _, ok := crPublicKeysMap[k]; !ok {
			unclaimedArbiterKeys = append(unclaimedArbiterKeys, k)
		}
	}
	sort.Slice(unclaimedArbiterKeys, func(i, j int) bool {
		return strings.Compare(unclaimedArbiterKeys[i], unclaimedArbiterKeys[j]) < 0
	})
	crcArbiters := map[common.Uint168]ArbiterMember{}
	claimHeight := a.ChainParams.CRClaimDPOSNodeStartHeight
	for _, cr := range crMembers {
		var pk []byte
		if len(cr.DPOSPublicKey) == 0 {
			var err error
			pk, err = common.HexStringToBytes(unclaimedArbiterKeys[0])
			if err != nil {
				return nil, err
			}
			unclaimedArbiterKeys = unclaimedArbiterKeys[1:]
		} else {
			pk = cr.DPOSPublicKey
		}
		crPublicKey := cr.Info.Code[1 : len(cr.Info.Code)-1]
		isNormal := true
		if height >= claimHeight && cr.MemberState != state.MemberElected {
			isNormal = false
		}
		ar, err := NewCRCArbiter(pk, crPublicKey, cr, isNormal)
		if err != nil {
			return nil, err
		}
		crcArbiters[ar.GetOwnerProgramHash()] = ar
	}

	return crcArbiters, nil
}

func (a *Arbiters) getCRCArbitersV0() (map[common.Uint168]ArbiterMember, error) {
	crMembers := a.CRCommittee.GetAllMembersCopy()
	if len(crMembers) != len(a.ChainParams.CRCArbiters) {
		return nil, errors.New("CRC members count mismatch with CRC arbiters")
	}

	crcArbiters := map[common.Uint168]ArbiterMember{}
	for i, v := range a.ChainParams.CRCArbiters {
		pk, err := common.HexStringToBytes(v)
		if err != nil {
			return nil, err
		}
		ar, err := NewCRCArbiter(pk, pk, crMembers[i], true)
		if err != nil {
			return nil, err
		}
		crcArbiters[ar.GetOwnerProgramHash()] = ar
	}

	return crcArbiters, nil
}

func (a *Arbiters) GetCandidatesDesc(height uint32, startIndex int,
	producers []*Producer) ([]ArbiterMember, error) {
	// main version >= H2
	if height >= a.ChainParams.PublicDPOSHeight {
		if len(producers) < startIndex {
			return make([]ArbiterMember, 0), nil
		}

		result := make([]ArbiterMember, 0)
		for i := startIndex; i < len(producers) && i < startIndex+a.
			ChainParams.CandidateArbiters; i++ {
			ar, err := NewDPoSArbiter(producers[i])
			if err != nil {
				return nil, err
			}
			result = append(result, ar)
		}
		return result, nil
	}

	// old version [0, H2)
	return nil, nil
}

func (a *Arbiters) GetDposV2CandidatesDesc(startIndex int,
	producers []string, choosingArbiters map[common.Uint168]ArbiterMember) ([]ArbiterMember, error) {
	if len(producers) < startIndex {
		return make([]ArbiterMember, 0), nil
	}

	result := make([]ArbiterMember, 0)
	for i := startIndex; i < len(producers) && i < startIndex+a.
		ChainParams.CandidateArbiters; i++ {
		ownkey, _ := hex.DecodeString(producers[i])
		hash, _ := contract.PublicKeyToStandardProgramHash(ownkey)
		crc, exist := choosingArbiters[*hash]
		if exist {
			result = append(result, crc)
		} else {
			ar, err := NewDPoSArbiter(a.getProducer(ownkey))
			if err != nil {
				return nil, err
			}
			result = append(result, ar)
		}
	}
	return result, nil
}

func (a *Arbiters) GetDposV2NormalArbitratorsDesc(
	arbitratorsCount int, producers []string, choosingArbiters map[common.Uint168]ArbiterMember) ([]ArbiterMember, error) {

	return a.getDposV2NormalArbitratorsDescV2(arbitratorsCount, producers, choosingArbiters)
}

func (a *Arbiters) GetNormalArbitratorsDesc(height uint32,
	arbitratorsCount int, producers []*Producer, start int) ([]ArbiterMember, error) {

	// main version >= H2
	if height >= a.ChainParams.PublicDPOSHeight {
		return a.getNormalArbitratorsDescV2(arbitratorsCount, producers, start)
	}

	// version [H1, H2)
	if height >= a.ChainParams.CRCOnlyDPOSHeight {
		return a.getNormalArbitratorsDescV1()
	}

	// version [0, H1)
	return a.getNormalArbitratorsDescV0()
}

func (a *Arbiters) snapshotVotesStates(height uint32) error {
	log.Debugf("snapshotVotesStates height %d begin", height)
	var nextReward RewardData
	recordVotes := func(nodePublicKey []byte) error {
		producer := a.GetProducer(nodePublicKey)
		if producer == nil {
			return errors.New("get producer by node public key failed")
		}
		programHash, err := contract.PublicKeyToStandardProgramHash(
			producer.OwnerPublicKey())
		if err != nil {
			return err
		}
		nextReward.OwnerVotesInRound[*programHash] = producer.Votes()
		nextReward.TotalVotesInRound += producer.Votes()
		return nil
	}

	nextReward.OwnerVotesInRound = make(map[common.Uint168]common.Fixed64, 0)
	nextReward.TotalVotesInRound = 0
	for _, ar := range a.nextArbitrators {
		if height > a.ChainParams.ChangeCommitteeNewCRHeight {
			if ar.GetType() == CRC && (!ar.IsNormal() ||
				(len(ar.(*crcArbiter).crMember.DPOSPublicKey) != 0 && ar.IsNormal())) {
				continue
			}
			if err := recordVotes(ar.GetNodePublicKey()); err != nil {
				continue
			}
		} else {
			if !a.isNextCRCArbitrator(ar.GetNodePublicKey()) {
				if err := recordVotes(ar.GetNodePublicKey()); err != nil {
					continue
				}
			}
		}
	}
	log.Debugf("snapshotVotesStates len(a.nextCandidates) %d", len(a.nextCandidates))
	log.Debugf("snapshotVotesStates a.nextCandidates %v", a.nextCandidates)

	for _, ar := range a.nextCandidates {
		if a.isNextCRCArbitrator(ar.GetNodePublicKey()) {
			continue
		}
		producer := a.GetProducer(ar.GetNodePublicKey())
		if producer == nil {
			return errors.New("get producer by node public key failed")
		}
		programHash, err := contract.PublicKeyToStandardProgramHash(producer.OwnerPublicKey())
		if err != nil {
			return err
		}
		nextReward.OwnerVotesInRound[*programHash] = producer.Votes()
		nextReward.TotalVotesInRound += producer.Votes()
	}
	log.Debugf("snapshotVotesStates a.NextReward %v", a.NextReward)
	log.Debugf("snapshotVotesStates a.TotalVotesInRound %f", a.NextReward.TotalVotesInRound)

	oriNextReward := a.NextReward
	a.History.Append(height, func() {
		a.NextReward = nextReward
	}, func() {
		a.NextReward = oriNextReward
	})
	return nil
}

func (a *Arbiters) DumpInfo(height uint32) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	a.dumpInfo(height)
}

func (a *Arbiters) dumpInfo(height uint32) {
	var printer func(string, ...interface{})
	changeType, _ := a.getChangeType(height + 1)
	switch changeType {
	case updateNext:
		fallthrough
	case normalChange:
		printer = log.Infof
	case none:
		printer = log.Debugf
	}

	var crInfo string
	crParams := make([]interface{}, 0)
	if len(a.CurrentArbitrators) != 0 {
		crInfo, crParams = getArbitersInfoWithOnduty("CURRENT ARBITERS",
			a.CurrentArbitrators, a.DutyIndex, a.getOnDutyArbitrator())
	} else {
		crInfo, crParams = getArbitersInfoWithoutOnduty("CURRENT ARBITERS", a.CurrentArbitrators)
	}
	nrInfo, nrParams := getArbitersInfoWithoutOnduty("NEXT ARBITERS", a.nextArbitrators)
	ccInfo, ccParams := getArbitersInfoWithoutOnduty("CURRENT CANDIDATES", a.CurrentCandidates)
	ncInfo, ncParams := getArbitersInfoWithoutOnduty("NEXT CANDIDATES", a.nextCandidates)
	printer(crInfo+nrInfo+ccInfo+ncInfo, append(append(append(crParams, nrParams...), ccParams...), ncParams...)...)
}

func (a *Arbiters) getBlockDPOSReward(block *types.Block) common.Fixed64 {
	totalTxFx := common.Fixed64(0)
	for _, tx := range block.Transactions {
		totalTxFx += tx.Fee()
	}

	return common.Fixed64(math.Ceil(float64(totalTxFx+
		a.ChainParams.GetBlockReward(block.Height)) * 0.35))
}

func (a *Arbiters) newCheckPoint(height uint32) *CheckPoint {
	point := &CheckPoint{
		Height:                     height,
		DutyIndex:                  a.DutyIndex,
		CurrentCandidates:          make([]ArbiterMember, 0),
		NextArbitrators:            make([]ArbiterMember, 0),
		NextCandidates:             make([]ArbiterMember, 0),
		CurrentReward:              *NewRewardData(),
		NextReward:                 *NewRewardData(),
		CurrentCRCArbitersMap:      make(map[common.Uint168]ArbiterMember),
		NextCRCArbitersMap:         make(map[common.Uint168]ArbiterMember),
		NextCRCArbiters:            make([]ArbiterMember, 0),
		CRCChangedHeight:           a.crcChangedHeight,
		AccumulativeReward:         a.accumulativeReward,
		FinalRoundChange:           a.finalRoundChange,
		ClearingHeight:             a.clearingHeight,
		ForceChanged:               a.forceChanged,
		ArbitersRoundReward:        make(map[common.Uint168]common.Fixed64),
		IllegalBlocksPayloadHashes: make(map[common.Uint256]interface{}),
		CurrentArbitrators:         a.CurrentArbitrators,
		StateKeyFrame:              *a.State.snapshot(),
	}
	point.CurrentArbitrators = copyByteList(a.CurrentArbitrators)
	point.CurrentCandidates = copyByteList(a.CurrentCandidates)
	point.NextArbitrators = copyByteList(a.nextArbitrators)
	point.NextCandidates = copyByteList(a.nextCandidates)
	point.CurrentReward = *copyReward(&a.CurrentReward)
	point.NextReward = *copyReward(&a.NextReward)
	point.NextCRCArbitersMap = copyCRCArbitersMap(a.nextCRCArbitersMap)
	point.CurrentCRCArbitersMap = copyCRCArbitersMap(a.CurrentCRCArbitersMap)
	point.NextCRCArbiters = copyByteList(a.nextCRCArbiters)

	for k, v := range a.arbitersRoundReward {
		point.ArbitersRoundReward[k] = v
	}
	for k := range a.illegalBlocksPayloadHashes {
		point.IllegalBlocksPayloadHashes[k] = nil
	}

	return point
}
func (a *Arbiters) Snapshot() *CheckPoint {
	return a.newCheckPoint(0)
}

func (a *Arbiters) SnapshotByHeight(height uint32) {
	var frames []*CheckPoint
	if v, ok := a.Snapshots[height]; ok {
		frames = v
	} else {
		// remove the oldest keys if SnapshotByHeight capacity is over
		if len(a.SnapshotKeysDesc) >= MaxSnapshotLength {
			for i := MaxSnapshotLength - 1; i < len(a.SnapshotKeysDesc); i++ {
				delete(a.Snapshots, a.SnapshotKeysDesc[i])
			}
			a.SnapshotKeysDesc = a.SnapshotKeysDesc[0 : MaxSnapshotLength-1]
		}

		a.SnapshotKeysDesc = append(a.SnapshotKeysDesc, height)
		sort.Slice(a.SnapshotKeysDesc, func(i, j int) bool {
			return a.SnapshotKeysDesc[i] > a.SnapshotKeysDesc[j]
		})
	}
	checkpoint := a.newCheckPoint(height)
	frames = append(frames, checkpoint)
	a.Snapshots[height] = frames
}

func (a *Arbiters) GetSnapshot(height uint32) []*CheckPoint {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	if height > a.bestHeight() {
		return []*CheckPoint{
			{
				CurrentArbitrators: a.CurrentArbitrators,
			},
		}
	} else {
		return a.getSnapshot(height)
	}
}

func (a *Arbiters) getSnapshot(height uint32) []*CheckPoint {
	result := make([]*CheckPoint, 0)
	if height >= a.SnapshotKeysDesc[len(a.SnapshotKeysDesc)-1] {
		// if height is in range of SnapshotKeysDesc, get the key with the same
		// election as height
		key := a.SnapshotKeysDesc[0]
		for i := 1; i < len(a.SnapshotKeysDesc); i++ {
			if height >= a.SnapshotKeysDesc[i] &&
				height < a.SnapshotKeysDesc[i-1] {
				key = a.SnapshotKeysDesc[i]
			}
		}

		return a.Snapshots[key]
	}
	return result
}

func getArbitersInfoWithOnduty(title string, arbiters []ArbiterMember,
	dutyIndex int, ondutyArbiter []byte) (string, []interface{}) {
	info := "\n" + title + "\nDUTYINDEX: %d\n%5s %66s %6s \n----- " +
		strings.Repeat("-", 66) + " ------\n"
	params := make([]interface{}, 0)
	params = append(params, (dutyIndex+1)%len(arbiters))
	params = append(params, "INDEX", "PUBLICKEY", "ONDUTY")
	for i, arbiter := range arbiters {
		info += "%-5d %-66s %6t\n"
		var publicKey string
		if arbiter.IsNormal() {
			publicKey = common.BytesToHexString(arbiter.GetNodePublicKey())
		}
		params = append(params, i+1, publicKey, bytes.Equal(
			arbiter.GetNodePublicKey(), ondutyArbiter))
	}
	info += "----- " + strings.Repeat("-", 66) + " ------"
	return info, params
}

func getArbitersInfoWithoutOnduty(title string,
	arbiters []ArbiterMember) (string, []interface{}) {

	info := "\n" + title + "\n%5s %66s\n----- " + strings.Repeat("-", 66) + "\n"
	params := make([]interface{}, 0)
	params = append(params, "INDEX", "PUBLICKEY")
	for i, arbiter := range arbiters {
		info += "%-5d %-66s\n"
		var publicKey string
		if arbiter.IsNormal() {
			publicKey = common.BytesToHexString(arbiter.GetNodePublicKey())
		}
		params = append(params, i+1, publicKey)
	}
	info += "----- " + strings.Repeat("-", 66)
	return info, params
}

func (a *Arbiters) initArbitrators(chainParams *config.Params) error {
	originArbiters := make([]ArbiterMember, len(chainParams.OriginArbiters))
	for i, arbiter := range chainParams.OriginArbiters {
		b, err := common.HexStringToBytes(arbiter)
		if err != nil {
			return err
		}
		ar, err := NewOriginArbiter(b)
		if err != nil {
			return err
		}

		originArbiters[i] = ar
	}

	crcArbiters := make(map[common.Uint168]ArbiterMember)
	for _, pk := range chainParams.CRCArbiters {
		pubKey, err := hex.DecodeString(pk)
		if err != nil {
			return err
		}
		producer := &Producer{ // here need crc NODE public key
			info: payload.ProducerInfo{
				OwnerPublicKey: pubKey,
				NodePublicKey:  pubKey,
			},
			activateRequestHeight: math.MaxUint32,
		}
		ar, err := NewDPoSArbiter(producer)
		if err != nil {
			return err
		}
		crcArbiters[ar.GetOwnerProgramHash()] = ar
	}

	a.CurrentArbitrators = originArbiters
	a.nextArbitrators = originArbiters
	a.nextCRCArbitersMap = crcArbiters
	a.CurrentCRCArbitersMap = crcArbiters
	a.CurrentReward = RewardData{
		OwnerVotesInRound: make(map[common.Uint168]common.Fixed64),
		TotalVotesInRound: 0,
	}
	a.NextReward = RewardData{
		OwnerVotesInRound: make(map[common.Uint168]common.Fixed64),
		TotalVotesInRound: 0,
	}
	return nil
}

func NewArbitrators(chainParams *config.Params, committee *state.Committee,
	getProducerDepositAmount func(common.Uint168) (common.Fixed64, error),
	tryUpdateCRMemberInactivity func(did common.Uint168, needReset bool, height uint32),
	tryRevertCRMemberInactivityfunc func(did common.Uint168, oriState state.MemberState, oriInactiveCount uint32, height uint32),
	tryUpdateCRMemberIllegal func(did common.Uint168, height uint32, illegalPenalty common.Fixed64),
	tryRevertCRMemberIllegal func(did common.Uint168, oriState state.MemberState, height uint32, illegalPenalty common.Fixed64),
	updateCRInactivePenalty func(cid common.Uint168, height uint32),
	revertUpdateCRInactivePenalty func(cid common.Uint168, height uint32)) (
	*Arbiters, error) {
	a := &Arbiters{
		ChainParams:                chainParams,
		CRCommittee:                committee,
		nextCandidates:             make([]ArbiterMember, 0),
		accumulativeReward:         common.Fixed64(0),
		finalRoundChange:           common.Fixed64(0),
		arbitersRoundReward:        nil,
		illegalBlocksPayloadHashes: make(map[common.Uint256]interface{}),
		Snapshots:                  make(map[uint32][]*CheckPoint),
		SnapshotKeysDesc:           make([]uint32, 0),
		crcChangedHeight:           0,
		degradation: &degradation{
			inactiveTxs:       make(map[common.Uint256]interface{}),
			inactivateHeight:  0,
			understaffedSince: 0,
			state:             DSNormal,
		},
		History: utils.NewHistory(maxHistoryCapacity),
	}
	if err := a.initArbitrators(chainParams); err != nil {
		return nil, err
	}
	a.State = NewState(chainParams, a.GetArbitrators, a.CRCommittee.GetAllMembers,
		a.CRCommittee.IsInElectionPeriod,
		getProducerDepositAmount, tryUpdateCRMemberInactivity, tryRevertCRMemberInactivityfunc,
		tryUpdateCRMemberIllegal, tryRevertCRMemberIllegal,
		updateCRInactivePenalty,
		revertUpdateCRInactivePenalty)

	chainParams.CkpManager.Register(NewCheckpoint(a))
	return a, nil
}
