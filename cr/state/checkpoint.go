// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"bytes"
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/checkpoint"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/utils"
)

const (
	// checkpointKey defines key of DPoS checkpoint.
	checkpointKey = "cp_cr"

	// checkpointExtension defines checkpoint file extension of DPoS checkpoint.
	checkpointExtension = ".ccp"

	// checkpointHeight defines interval Height between two neighbor check
	// points.
	checkpointHeight = uint32(720)

	// EffectiveHeight defines interval to change src file to default file.
	EffectiveHeight = uint32(7)
)

// Checkpoint hold all CR related states to recover from scratch.
type Checkpoint struct {
	KeyFrame
	StateKeyFrame
	ProposalKeyFrame
	Height    uint32
	committee *Committee
}

func (c *Checkpoint) OnBlockSaved(block *types.DposBlock) {
	if block.Height <= c.GetHeight() {
		return
	}
	c.committee.ProcessBlock(block.Block, block.Confirm)
}

func (c *Checkpoint) OnRollbackTo(height uint32) error {
	keyFrame := NewKeyFrame()
	if height < c.StartHeight() {
		committee := &Committee{
			state:                NewState(c.committee.params),
			params:               c.committee.params,
			KeyFrame:             *keyFrame,
			firstHistory:         utils.NewHistory(maxHistoryCapacity),
			lastHistory:          utils.NewHistory(maxHistoryCapacity),
			appropriationHistory: utils.NewHistory(maxHistoryCapacity),
		}
		c.initFromCommittee(committee)
		c.committee.Recover(c)
		c.committee.state.registerFunctions(&FunctionsConfig{
			GetHistoryMember: committee.getHistoryMember,
		})
		return nil
	}
	return c.committee.RollbackTo(height)
}

func (c *Checkpoint) OnRollbackSeekTo(height uint32) {
	c.committee.firstHistory.RollbackSeekTo(height)
	c.committee.lastHistory.RollbackSeekTo(height)
	c.committee.appropriationHistory.RollbackSeekTo(height)
	c.committee.manager.history.RollbackSeekTo(height)
	c.committee.state.history.RollbackSeekTo(height)
	c.committee.state.manager.history.RollbackSeekTo(height)
}

func (c *Checkpoint) Key() string {
	return checkpointKey
}

func (c *Checkpoint) Snapshot() checkpoint.ICheckPoint {
	// init check point
	c.initFromCommittee(c.committee)

	buf := new(bytes.Buffer)
	if err := c.Serialize(buf); err != nil {
		c.LogError(err)
		return nil
	}
	result := &Checkpoint{}
	if err := result.Deserialize(buf); err != nil {
		c.LogError(err)
		return nil
	}
	return result
}

func (c *Checkpoint) GetHeight() uint32 {
	return c.Height
}

func (c *Checkpoint) SetHeight(height uint32) {
	c.Height = height
}

func (c *Checkpoint) SavePeriod() uint32 {
	return checkpointHeight
}

func (c *Checkpoint) EffectivePeriod() uint32 {
	return EffectiveHeight
}

func (c *Checkpoint) DataExtension() string {
	return checkpointExtension
}

func (c *Checkpoint) Generator() func(buf []byte) checkpoint.ICheckPoint {
	return func(buf []byte) checkpoint.ICheckPoint {
		stream := new(bytes.Buffer)
		stream.Write(buf)

		result := &Checkpoint{}
		if err := result.Deserialize(stream); err != nil {
			c.LogError(err)
			return nil
		}
		return result
	}
}

func (c *Checkpoint) LogError(err error) {
	log.Warn(err)
}

func (c *Checkpoint) Priority() checkpoint.Priority {
	return checkpoint.MediumHigh
}

func (c *Checkpoint) OnInit() {
	c.committee.Recover(c)
}

func (c *Checkpoint) StartHeight() uint32 {
	return c.committee.params.CRVotingStartHeight
}

func (c *Checkpoint) Serialize(w io.Writer) (err error) {
	if err = common.WriteUint32(w, c.Height); err != nil {
		return
	}

	if err = c.KeyFrame.Serialize(w); err != nil {
		return
	}
	if err = c.StateKeyFrame.Serialize(w); err != nil {
		return
	}
	return c.ProposalKeyFrame.Serialize(w)
}

func (c *Checkpoint) Deserialize(r io.Reader) (err error) {
	if c.Height, err = common.ReadUint32(r); err != nil {
		return
	}

	if err = c.KeyFrame.Deserialize(r); err != nil {
		return
	}
	if err = c.StateKeyFrame.Deserialize(r); err != nil {
		return
	}
	return c.ProposalKeyFrame.Deserialize(r)
}

func (c *Checkpoint) initFromCommittee(committee *Committee) {
	c.StateKeyFrame = committee.state.StateKeyFrame
	c.KeyFrame = committee.KeyFrame
	if committee.manager != nil {
		c.ProposalKeyFrame = committee.manager.ProposalKeyFrame
	}
}

func NewCheckpoint(committee *Committee) *Checkpoint {
	cp := &Checkpoint{
		Height:    uint32(0),
		committee: committee,
	}
	cp.initFromCommittee(committee)
	committee.getCheckpoint = func(height uint32) *Checkpoint {
		if height > cp.GetHeight() {
			return cp
		}
		return nil
	}

	return cp
}
