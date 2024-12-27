// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"bytes"
	"errors"
	"io"
	"math"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/checkpoint"
	"github.com/elastos/Elastos.ELA/core/types"
)

const (
	// CheckpointKey defines key of DPoS checkpoint.
	CheckpointKey = "cp_dpos"

	// checkpointExtension defines checkpoint file extension of DPoS checkpoint.
	checkpointExtension = ".dcp"

	// CheckPointInterval defines interval height between two neighbor check
	// points.
	CheckPointInterval = uint32(720)

	// checkpointEffectiveHeight defines the minimal height Arbiters obj
	// should scan to recover effective state.
	checkpointEffectiveHeight = uint32(7)
)

// CheckPoint defines all variables need record in database
type CheckPoint struct {
	StateKeyFrame
	Height                      uint32
	DutyIndex                   int
	LastArbitrators             []ArbiterMember
	CurrentArbitrators          []ArbiterMember
	NextArbitrators             []ArbiterMember
	NextCandidates              []ArbiterMember
	CurrentCandidates           []ArbiterMember
	CurrentReward               RewardData
	NextReward                  RewardData
	LastDPoSRewards             map[string]map[string]common.Fixed64
	CurrentCRCArbitersMap       map[common.Uint168]ArbiterMember
	CurrentOnDutyCRCArbitersMap map[common.Uint168]ArbiterMember
	NextCRCArbitersMap          map[common.Uint168]ArbiterMember
	NextCRCArbiters             []ArbiterMember

	CRCChangedHeight           uint32
	AccumulativeReward         common.Fixed64
	FinalRoundChange           common.Fixed64
	ClearingHeight             uint32
	ArbitersRoundReward        map[common.Uint168]common.Fixed64
	IllegalBlocksPayloadHashes map[common.Uint256]interface{}

	ForceChanged bool
	arbitrators  *Arbiters
}

func (c *CheckPoint) StartHeight() uint32 {
	return uint32(math.Min(float64(c.arbitrators.ChainParams.VoteStartHeight),
		float64(c.arbitrators.ChainParams.CRCOnlyDPOSHeight-
			c.arbitrators.ChainParams.DPoSConfiguration.PreConnectOffset)))
}

func (c *CheckPoint) OnBlockSaved(block *types.DposBlock) {
	if block.Height <= c.GetHeight() {
		return
	}
	c.arbitrators.ProcessBlock(block.Block, block.Confirm)
}

func (c *CheckPoint) OnRollbackSeekTo(height uint32) {
	c.arbitrators.RollbackSeekTo(height)
}

func (c *CheckPoint) OnReset() error {
	log.Info("dpos state OnReset")
	ar := &Arbiters{}
	ar.State = &State{
		StateKeyFrame: NewStateKeyFrame(),
	}
	if err := ar.initArbitrators(c.arbitrators.ChainParams); err != nil {
		return err
	}
	c.initFromArbitrators(ar)
	c.arbitrators.RecoverFromCheckPoints(c)
	return nil
}

func (c *CheckPoint) OnRollbackTo(height uint32) error {
	if height < c.StartHeight() {
		ar := &Arbiters{}
		if err := ar.initArbitrators(c.arbitrators.ChainParams); err != nil {
			return err
		}
		c.initFromArbitrators(ar)
		c.arbitrators.RecoverFromCheckPoints(c)
		return nil
	}
	return c.arbitrators.RollbackTo(height)
}

func (c *CheckPoint) Key() string {
	return CheckpointKey
}

func (c *CheckPoint) OnInit() {
	c.arbitrators.RecoverFromCheckPoints(c)
}

func (c *CheckPoint) Snapshot() checkpoint.ICheckPoint {
	// init check point
	c.initFromArbitrators(c.arbitrators)

	buf := new(bytes.Buffer)
	if err := c.Serialize(buf); err != nil {
		c.LogError(err)
		return nil
	}
	result := &CheckPoint{}
	if err := result.Deserialize(buf); err != nil {
		c.LogError(err)
		return nil
	}
	return result
}

func (c *CheckPoint) GetHeight() uint32 {
	return c.Height
}

func (c *CheckPoint) SetHeight(height uint32) {
	c.Height = height
}

func (c *CheckPoint) SavePeriod() uint32 {
	return CheckPointInterval
}

func (c *CheckPoint) EffectivePeriod() uint32 {
	return checkpointEffectiveHeight
}

func (c *CheckPoint) DataExtension() string {
	return checkpointExtension
}

func (c *CheckPoint) Priority() checkpoint.Priority {
	return checkpoint.Medium
}

func (c *CheckPoint) Generator() func(buf []byte) checkpoint.ICheckPoint {
	return func(buf []byte) checkpoint.ICheckPoint {
		stream := new(bytes.Buffer)
		stream.Write(buf)

		result := &CheckPoint{}
		if err := result.Deserialize(stream); err != nil {
			c.LogError(err)
			return nil
		}
		return result
	}
}

func (c *CheckPoint) LogError(err error) {
	log.Warn("[CheckPoint] error: ", err.Error())
}

// Serialize write data to writer
func (c *CheckPoint) Serialize(w io.Writer) (err error) {
	if err = common.WriteUint32(w, c.Height); err != nil {
		return
	}

	if err = common.WriteUint32(w, uint32(c.DutyIndex)); err != nil {
		return
	}

	if err = c.writeArbiters(w, c.LastArbitrators); err != nil {
		return
	}

	if err = c.writeArbiters(w, c.CurrentArbitrators); err != nil {
		return
	}

	if err = c.writeArbiters(w, c.CurrentCandidates); err != nil {
		return
	}

	if err = c.writeArbiters(w, c.NextArbitrators); err != nil {
		return
	}

	if err = c.writeArbiters(w, c.NextCandidates); err != nil {
		return
	}

	if err = c.CurrentReward.Serialize(w); err != nil {
		return
	}

	if err = c.NextReward.Serialize(w); err != nil {
		return
	}

	if err = c.serializeDPoSRewardsMap(w, c.LastDPoSRewards); err != nil {
		return
	}

	if err = c.serializeCRCArbitersMap(w, c.CurrentCRCArbitersMap); err != nil {
		return
	}
	if err = c.serializeCRCArbitersMap(w, c.CurrentOnDutyCRCArbitersMap); err != nil {
		return
	}

	if err = c.serializeCRCArbitersMap(w, c.NextCRCArbitersMap); err != nil {
		return
	}
	if err = c.writeArbiters(w, c.NextCRCArbiters); err != nil {
		return
	}

	if err = common.WriteUint32(w, c.CRCChangedHeight); err != nil {
		return
	}

	if err = c.AccumulativeReward.Serialize(w); err != nil {
		return
	}

	if err = c.FinalRoundChange.Serialize(w); err != nil {
		return
	}

	if err = common.WriteUint32(w, c.ClearingHeight); err != nil {
		return
	}

	if err = c.serializeRoundRewardMap(w, c.ArbitersRoundReward); err != nil {
		return
	}

	if err = c.serializeIllegalPayloadHashesMap(w, c.IllegalBlocksPayloadHashes); err != nil {
		return
	}

	if err = common.WriteElements(w, c.ForceChanged); err != nil {
		return
	}
	return c.StateKeyFrame.Serialize(w)
}

func (c *CheckPoint) serializeCRCArbitersMap(w io.Writer,
	rmap map[common.Uint168]ArbiterMember) (err error) {
	if err = common.WriteVarUint(w, uint64(len(rmap))); err != nil {
		return
	}
	for k, v := range rmap {
		if err = k.Serialize(w); err != nil {
			return
		}

		if err = common.WriteUint8(w, uint8(v.GetType())); err != nil {
			return
		}

		if err = v.Serialize(w); err != nil {
			return
		}
	}
	return
}

func (c *CheckPoint) serializeDPoSRewardsMap(w io.Writer,
	rmap map[string]map[string]common.Fixed64) (err error) {
	if err = common.WriteVarUint(w, uint64(len(rmap))); err != nil {
		return
	}
	for k, v := range rmap {
		if err = common.WriteVarString(w, k); err != nil {
			return
		}
		if err = common.WriteVarUint(w, uint64(len(v))); err != nil {
			return
		}
		for k2, v2 := range v {
			if err = common.WriteVarString(w, k2); err != nil {
				return
			}
			if err = v2.Serialize(w); err != nil {
				return
			}
		}
	}
	return
}

func (c *CheckPoint) serializeRoundRewardMap(w io.Writer,
	rmap map[common.Uint168]common.Fixed64) (err error) {
	if err = common.WriteVarUint(w, uint64(len(rmap))); err != nil {
		return
	}
	for k, v := range rmap {
		if err = k.Serialize(w); err != nil {
			return
		}

		if err = v.Serialize(w); err != nil {
			return
		}
	}
	return
}

func (c *CheckPoint) serializeIllegalPayloadHashesMap(w io.Writer,
	rmap map[common.Uint256]interface{}) (err error) {
	if err = common.WriteVarUint(w, uint64(len(rmap))); err != nil {
		return
	}
	for k, _ := range rmap {
		if err = k.Serialize(w); err != nil {
			return
		}
	}
	return
}

// Deserialize read data to reader
func (c *CheckPoint) Deserialize(r io.Reader) (err error) {
	if c.Height, err = common.ReadUint32(r); err != nil {
		return
	}

	var dutyIndex uint32
	if dutyIndex, err = common.ReadUint32(r); err != nil {
		return
	}
	c.DutyIndex = int(dutyIndex)

	if c.LastArbitrators, err = c.readArbiters(r); err != nil {
		return
	}

	if c.CurrentArbitrators, err = c.readArbiters(r); err != nil {
		return
	}

	if c.CurrentCandidates, err = c.readArbiters(r); err != nil {
		return
	}

	if c.NextArbitrators, err = c.readArbiters(r); err != nil {
		return
	}

	if c.NextCandidates, err = c.readArbiters(r); err != nil {
		return
	}

	if err = c.CurrentReward.Deserialize(r); err != nil {
		return
	}

	if err = c.NextReward.Deserialize(r); err != nil {
		return
	}

	if c.LastDPoSRewards, err = c.deserializeDPoSRewardsMap(r); err != nil {
		return
	}

	if c.CurrentCRCArbitersMap, err = c.deserializeCRCArbitersMap(r); err != nil {
		return
	}
	if c.CurrentOnDutyCRCArbitersMap, err = c.deserializeCRCArbitersMap(r); err != nil {
		return
	}
	if c.NextCRCArbitersMap, err = c.deserializeCRCArbitersMap(r); err != nil {
		return
	}
	if c.NextCRCArbiters, err = c.readArbiters(r); err != nil {
		return
	}

	if c.CRCChangedHeight, err = common.ReadUint32(r); err != nil {
		return
	}

	if err = c.AccumulativeReward.Deserialize(r); err != nil {
		return
	}

	if err = c.FinalRoundChange.Deserialize(r); err != nil {
		return
	}

	if c.ClearingHeight, err = common.ReadUint32(r); err != nil {
		return
	}

	if c.ArbitersRoundReward, err = c.deserializeRoundRewardMap(r); err != nil {
		return
	}

	c.IllegalBlocksPayloadHashes, err = c.deserializeIllegalPayloadHashes(r)
	if err != nil {
		return
	}
	if err = common.ReadElement(r, &c.ForceChanged); err != nil {
		return
	}
	return c.StateKeyFrame.Deserialize(r)
}

func (c *CheckPoint) deserializeCRCArbitersMap(r io.Reader) (
	rmap map[common.Uint168]ArbiterMember, err error) {

	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	rmap = make(map[common.Uint168]ArbiterMember)
	for i := uint64(0); i < count; i++ {
		var k common.Uint168
		if err = k.Deserialize(r); err != nil {
			return
		}

		var arbiterType uint8
		if arbiterType, err = common.ReadUint8(r); err != nil {
			return
		}
		var am ArbiterMember
		switch ArbiterType(arbiterType) {
		case Origin:
			am = &originArbiter{}
		case DPoS:
			am = &dposArbiter{}
		case CROrigin:
			am = &dposArbiter{}
		case CRC:
			am = &crcArbiter{}
		default:
			err = errors.New("invalid arbiter type")
			return
		}
		if err = am.Deserialize(r); err != nil {
			return
		}

		rmap[k] = am
	}
	return
}

func (c *CheckPoint) deserializeIllegalPayloadHashes(
	r io.Reader) (hmap map[common.Uint256]interface{}, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	hmap = make(map[common.Uint256]interface{})
	for i := uint64(0); i < count; i++ {
		var k common.Uint256
		if err = k.Deserialize(r); err != nil {
			return
		}
		hmap[k] = nil
	}
	return
}

func (c *CheckPoint) deserializeDPoSRewardsMap(
	r io.Reader) (rmap map[string]map[string]common.Fixed64, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	rmap = make(map[string]map[string]common.Fixed64)
	for i := uint64(0); i < count; i++ {
		var k string
		if k, err = common.ReadVarString(r); err != nil {
			return
		}

		var count2 uint64
		if count2, err = common.ReadVarUint(r, 0); err != nil {
			return
		}
		rmap2 := make(map[string]common.Fixed64)
		for i := uint64(0); i < count2; i++ {
			var k2 string
			if k2, err = common.ReadVarString(r); err != nil {
				return
			}
			reward := common.Fixed64(0)
			if err = reward.Deserialize(r); err != nil {
				return
			}
			rmap2[k2] = reward
		}

		rmap[k] = rmap2
	}
	return
}

func (c *CheckPoint) deserializeRoundRewardMap(
	r io.Reader) (rmap map[common.Uint168]common.Fixed64, err error) {
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return
	}
	rmap = make(map[common.Uint168]common.Fixed64)
	for i := uint64(0); i < count; i++ {
		var k common.Uint168
		if err = k.Deserialize(r); err != nil {
			return
		}
		reward := common.Fixed64(0)
		if err = reward.Deserialize(r); err != nil {
			return
		}
		rmap[k] = reward
	}
	return
}

func (c *CheckPoint) writeArbiters(w io.Writer,
	arbiters []ArbiterMember) error {
	if err := common.WriteVarUint(w, uint64(len(arbiters))); err != nil {
		return err
	}

	for _, ar := range arbiters {
		if err := SerializeArbiterMember(ar, w); err != nil {
			return err
		}
	}
	return nil
}

func (c *CheckPoint) readArbiters(r io.Reader) ([]ArbiterMember, error) {
	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return nil, err
	}

	arbiters := make([]ArbiterMember, 0, count)
	for i := uint64(0); i < count; i++ {
		arbiter, err := ArbiterMemberFromReader(r)
		if err != nil {
			return nil, err
		}
		arbiters = append(arbiters, arbiter)
	}
	return arbiters, nil
}

func (c *CheckPoint) initFromArbitrators(ar *Arbiters) {
	c.CurrentCandidates = ar.CurrentCandidates
	c.NextArbitrators = ar.nextArbitrators
	c.NextCandidates = ar.nextCandidates
	c.CurrentReward = ar.CurrentReward
	c.NextReward = ar.NextReward
	c.LastDPoSRewards = ar.LastDPoSRewards
	c.LastArbitrators = ar.LastArbitrators
	c.CurrentArbitrators = ar.CurrentArbitrators
	c.StateKeyFrame = *ar.State.StateKeyFrame
	c.DutyIndex = ar.DutyIndex
	c.AccumulativeReward = ar.accumulativeReward
	c.FinalRoundChange = ar.finalRoundChange
	c.ClearingHeight = ar.clearingHeight
	c.ArbitersRoundReward = ar.arbitersRoundReward
	c.IllegalBlocksPayloadHashes = ar.illegalBlocksPayloadHashes
	c.CRCChangedHeight = ar.crcChangedHeight
	c.CurrentCRCArbitersMap = ar.CurrentCRCArbitersMap
	c.NextCRCArbitersMap = ar.nextCRCArbitersMap
	c.NextCRCArbiters = ar.nextCRCArbiters
	c.ForceChanged = ar.forceChanged

}

func NewCheckpoint(ar *Arbiters) *CheckPoint {
	cp := &CheckPoint{
		arbitrators: ar,
	}
	cp.initFromArbitrators(ar)
	return cp
}
