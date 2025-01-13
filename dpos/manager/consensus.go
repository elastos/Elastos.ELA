// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package manager

import (
	"bytes"
	"time"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/dpos/log"
	"github.com/elastos/Elastos.ELA/dpos/p2p/msg"
)

const (
	consensusReady = iota
	consensusRunning

	DefaultViewOffset = 0
)

type Consensus struct {
	currentHeight   uint32
	finishedHeight  uint32
	consensusStatus uint32
	viewOffset      uint32

	manager     *DPOSManager
	currentView view
}

func NewConsensus(manager *DPOSManager, tolerance time.Duration,
	viewListener ViewListener, changeViewV1Height uint32) *Consensus {
	c := &Consensus{
		consensusStatus: consensusReady,
		viewOffset:      DefaultViewOffset,
		manager:         manager,
		currentView: view{
			publicKey:          manager.publicKey,
			signTolerance:      tolerance,
			listener:           viewListener,
			arbitrators:        manager.arbitrators,
			changeViewV1Height: changeViewV1Height,
		},
	}

	return c
}

func (c *Consensus) IsOnDuty() bool {
	return c.currentView.IsOnDuty()
}

func (c *Consensus) SetOnDuty(onDuty bool) {
	c.currentView.SetOnDuty(onDuty)
}

func (c *Consensus) SetRunning() {
	c.consensusStatus = consensusRunning
	c.resetViewOffset()
}

func (c *Consensus) SetReady(height uint32) {
	c.finishedHeight = height
	c.consensusStatus = consensusReady
	c.resetViewOffset()
}

func (c *Consensus) IsRunning() bool {
	return c.consensusStatus == consensusRunning
}

func (c *Consensus) IsReady() bool {
	return c.consensusStatus == consensusReady
}

func (c *Consensus) IsArbitratorOnDuty(arbitrator []byte) bool {
	return bytes.Equal(c.GetOnDutyArbitrator(), arbitrator)
}

func (c *Consensus) GetOnDutyArbitrator() []byte {
	return c.manager.GetArbitrators().GetNextOnDutyArbitrator(c.viewOffset)
}

func (c *Consensus) StartConsensus(b *types.Block) {
	log.Info("[StartConsensus] consensus start")
	defer log.Info("[StartConsensus] consensus end")

	c.currentHeight = b.Height

	now := c.manager.timeSource.AdjustedTime()
	c.manager.GetBlockCache().Reset(b)
	c.SetRunning()

	c.manager.GetBlockCache().AddValue(b.Hash(), b)
	c.currentView.ResetView(now)

	viewEvent := log.ViewEvent{
		OnDutyArbitrator: common.BytesToHexString(c.GetOnDutyArbitrator()),
		StartTime:        now,
		Offset:           c.GetViewOffset(),
		Height:           b.Height,
	}
	c.manager.dispatcher.cfg.EventMonitor.OnViewStarted(&viewEvent)
}

func (c *Consensus) GetViewOffset() uint32 {
	return c.viewOffset
}

func (c *Consensus) ProcessBlock(b *types.Block) {
	c.manager.GetBlockCache().AddValue(b.Hash(), b)
}

func (c *Consensus) ChangeView() {
	if c.currentHeight < c.manager.chainParams.DPoSConfiguration.ChangeViewV1Height {
		c.currentView.ChangeView(&c.viewOffset, c.manager.timeSource.AdjustedTime())
	} else {
		c.currentView.ChangeViewV1(&c.viewOffset, c.manager.timeSource.AdjustedTime())
	}

}

func (c *Consensus) TryChangeView() bool {
	if c.IsRunning() {
		if c.currentHeight < c.manager.chainParams.DPoSConfiguration.ChangeViewV1Height {
			return c.currentView.TryChangeView(&c.viewOffset, c.manager.timeSource.AdjustedTime())
		} else {
			return c.currentView.TryChangeViewV1(&c.viewOffset, c.manager.timeSource.AdjustedTime())
		}

	}
	return false
}

func (c *Consensus) CollectConsensusStatus(status *msg.ConsensusStatus) error {
	status.ConsensusStatus = c.consensusStatus
	status.ViewOffset = c.viewOffset
	status.ViewStartTime = c.currentView.GetViewStartTime()
	log.Info("[CollectConsensusStatus] status.ConsensusStatus:", status.ConsensusStatus)
	return nil
}

func (c *Consensus) RecoverFromConsensusStatus(status *msg.ConsensusStatus) error {
	log.Info("[RecoverFromConsensusStatus] status.ConsensusStatus:", status.ConsensusStatus)
	c.consensusStatus = status.ConsensusStatus
	c.viewOffset = status.ViewOffset
	c.currentView.ResetView(status.ViewStartTime)
	return nil
}

func (c *Consensus) resetViewOffset() {
	c.viewOffset = DefaultViewOffset
}
