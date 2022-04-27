// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package unit

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/transaction"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/dpos/state"

	"github.com/stretchr/testify/assert"
)

func init() {
	testing.Init()

	functions.GetTransactionByTxType = transaction.GetTransaction
	functions.GetTransactionByBytes = transaction.GetTransactionByBytes
	functions.CreateTransaction = transaction.CreateTransaction
	functions.GetTransactionParameters = transaction.GetTransactionparameters
	config.DefaultParams = config.GetDefaultParams()
}

func TestRewardData_Deserialize(t *testing.T) {
	originData := randomRewardData()

	buf := new(bytes.Buffer)
	assert.NoError(t, originData.Serialize(buf))

	cmpData := state.NewRewardData()
	assert.NoError(t, cmpData.Deserialize(buf))

	assert.True(t, rewardEqual(originData, cmpData))
}

func TestDPOSStateKeyFrame_Deserialize(t *testing.T) {
	originFrame := randomDPOSStateKeyFrame()

	buf := new(bytes.Buffer)
	assert.NoError(t, originFrame.Serialize(buf))

	cmpData := &state.StateKeyFrame{}
	assert.NoError(t, cmpData.Deserialize(buf))

	assert.True(t, stateKeyFrameEqual(originFrame, cmpData))
}

func TestDPOSCheckPoint_Deserialize(t *testing.T) {
	originCheckPoint := generateDPOSCheckPoint(rand.Uint32())

	buf := new(bytes.Buffer)
	assert.NoError(t, originCheckPoint.Serialize(buf))

	cmpData := &state.CheckPoint{}
	assert.NoError(t, cmpData.Deserialize(buf))

	assert.True(t, dposCheckPointsEqual(originCheckPoint, cmpData))
}

func dposCheckPointsEqual(first *state.CheckPoint, second *state.CheckPoint) bool {
	if first.Height != second.Height || first.DutyIndex != second.DutyIndex ||
		first.CurrentReward.TotalVotesInRound !=
			second.CurrentReward.TotalVotesInRound ||
		second.NextReward.TotalVotesInRound !=
			second.NextReward.TotalVotesInRound ||
		first.ForceChanged != second.ForceChanged {
		return false
	}

	if !arrayEqual(first.CurrentArbitrators, second.CurrentArbitrators) ||
		!arrayEqual(first.CurrentCandidates, second.CurrentCandidates) ||
		!arrayEqual(first.NextArbitrators, second.NextArbitrators) ||
		!arrayEqual(first.NextCandidates, second.NextCandidates) ||
		!arrayEqual(first.NextCRCArbiters, second.NextCRCArbiters) {
		return false
	}

	if !stateKeyFrameEqual(&first.StateKeyFrame, &second.StateKeyFrame) {
		return false
	}

	if !arbitersMapEqual(first.NextCRCArbitersMap, second.NextCRCArbitersMap) ||
		!arbitersMapEqual(first.CurrentCRCArbitersMap, second.CurrentCRCArbitersMap) {
		return false
	}
	return votesMapEqual(first.CurrentReward.OwnerVotesInRound,
		second.CurrentReward.OwnerVotesInRound) &&
		votesMapEqual(first.NextReward.OwnerVotesInRound,
			second.NextReward.OwnerVotesInRound)
}

func generateDPOSCheckPoint(height uint32) *state.CheckPoint {
	result := &state.CheckPoint{
		Height:                height,
		DutyIndex:             int(rand.Uint32()),
		NextArbitrators:       []state.ArbiterMember{},
		NextCandidates:        []state.ArbiterMember{},
		CurrentCandidates:     []state.ArbiterMember{},
		CurrentArbitrators:    []state.ArbiterMember{},
		CurrentReward:         *state.NewRewardData(),
		NextReward:            *state.NewRewardData(),
		CurrentCRCArbitersMap: make(map[common.Uint168]state.ArbiterMember),
		NextCRCArbitersMap:    make(map[common.Uint168]state.ArbiterMember),
		NextCRCArbiters:       make([]state.ArbiterMember, 0),
		CRCChangedHeight:      123,
		ForceChanged:          true,
		StateKeyFrame:         *randomDPOSStateKeyFrame(),
	}
	result.CurrentReward.TotalVotesInRound = common.Fixed64(rand.Uint64())
	result.NextReward.TotalVotesInRound = common.Fixed64(rand.Uint64())

	for i := 0; i < 5; i++ {
		ar, _ := state.NewOriginArbiter(randomFakePK())
		result.CurrentArbitrators = append(result.CurrentArbitrators, ar)
		ar, _ = state.NewOriginArbiter(randomFakePK())
		result.CurrentCandidates = append(result.CurrentCandidates, ar)
		ar, _ = state.NewOriginArbiter(randomFakePK())
		result.NextArbitrators = append(result.NextArbitrators, ar)
		ar, _ = state.NewOriginArbiter(randomFakePK())
		result.NextCandidates = append(result.NextCandidates, ar)

		ar, _ = state.NewOriginArbiter(randomFakePK())
		result.NextCRCArbiters = append(result.NextCRCArbiters, ar)
		ar, _ = state.NewOriginArbiter(randomFakePK())
		result.CurrentCRCArbitersMap[ar.GetOwnerProgramHash()] = ar
		ar, _ = state.NewOriginArbiter(randomFakePK())
		result.NextCRCArbitersMap[ar.GetOwnerProgramHash()] = ar

		result.CurrentReward.OwnerVotesInRound[*randomProgramHash()] =
			common.Fixed64(rand.Uint64())

		result.NextReward.OwnerVotesInRound[*randomProgramHash()] =
			common.Fixed64(rand.Uint64())
	}

	return result
}

func stateKeyFrameEqual(first *state.StateKeyFrame, second *state.StateKeyFrame) bool {
	if len(first.NodeOwnerKeys) != len(second.NodeOwnerKeys) ||
		len(first.PendingProducers) != len(second.PendingProducers) ||
		len(first.ActivityProducers) != len(second.ActivityProducers) ||
		len(first.InactiveProducers) != len(second.InactiveProducers) ||
		len(first.CanceledProducers) != len(second.CanceledProducers) ||
		len(first.IllegalProducers) != len(second.IllegalProducers) ||
		len(first.PendingCanceledProducers) != len(second.PendingCanceledProducers) ||
		len(first.Votes) != len(second.Votes) ||
		len(first.DposV2Votes) != len(second.DposV2Votes) ||
		len(first.DepositOutputs) != len(second.DepositOutputs) ||
		len(first.DposV2RewardInfo) != len(second.DposV2RewardInfo) ||
		len(first.DposV2RewardClaimingInfo) != len(second.DposV2RewardClaimingInfo) ||
		len(first.DposV2RewardClaimedInfo) != len(second.DposV2RewardClaimedInfo) ||
		len(first.WithdrawableTxInfo) != len(second.WithdrawableTxInfo) ||
		len(first.Nicknames) != len(second.Nicknames) ||
		len(first.SpecialTxHashes) != len(second.SpecialTxHashes) ||
		len(first.PreBlockArbiters) != len(second.PreBlockArbiters) ||
		len(first.ProducerDepositMap) != len(second.ProducerDepositMap) ||
		len(first.EmergencyInactiveArbiters) != len(second.EmergencyInactiveArbiters) {
		return false
	}

	for k, vf := range first.NodeOwnerKeys {
		vs, ok := second.NodeOwnerKeys[k]
		if !ok {
			return false
		}
		if vf != vs {
			return false
		}
	}

	for k, vf := range first.PendingProducers {
		vs, ok := second.PendingProducers[k]
		if !ok {
			return false
		}
		if !producerEqual(vf, vs) {
			return false
		}
	}

	for k, vf := range first.ActivityProducers {
		vs, ok := second.ActivityProducers[k]
		if !ok {
			return false
		}
		if !producerEqual(vf, vs) {
			return false
		}
	}

	for k, vf := range first.InactiveProducers {
		vs, ok := second.InactiveProducers[k]
		if !ok {
			return false
		}
		if !producerEqual(vf, vs) {
			return false
		}
	}

	for k, vf := range first.CanceledProducers {
		vs, ok := second.CanceledProducers[k]
		if !ok {
			return false
		}
		if !producerEqual(vf, vs) {
			return false
		}
	}

	for k, vf := range first.IllegalProducers {
		vs, ok := second.IllegalProducers[k]
		if !ok {
			return false
		}
		if !producerEqual(vf, vs) {
			return false
		}
	}

	for k, vf := range first.PendingCanceledProducers {
		vs, ok := second.PendingCanceledProducers[k]
		if !ok {
			return false
		}
		if !producerEqual(vf, vs) {
			return false
		}
	}

	for k := range first.Votes {
		_, ok := second.Votes[k]
		if !ok {
			return false
		}
	}

	for k, v1 := range first.DposV2Votes {
		v2, ok := second.DposV2Votes[k]
		if !ok {
			return false
		}
		if v1 != v2 {
			return false
		}
	}

	for k := range first.DepositOutputs {
		_, ok := second.DepositOutputs[k]
		if !ok {
			return false
		}
	}

	for k := range first.DposV2RewardInfo {
		_, ok := second.DposV2RewardInfo[k]
		if !ok {
			return false
		}
	}

	for k := range first.DposV2RewardClaimingInfo {
		_, ok := second.DposV2RewardClaimingInfo[k]
		if !ok {
			return false
		}
	}

	for k := range first.DposV2RewardClaimedInfo {
		_, ok := second.DposV2RewardClaimedInfo[k]
		if !ok {
			return false
		}
	}

	for k, vf := range first.WithdrawableTxInfo {
		vs, ok := second.WithdrawableTxInfo[k]
		if !ok {
			return false
		}
		if vf.Amount != vs.Amount {
			return false
		}
		if vf.Recipient != vs.Recipient {
			return false
		}
	}

	for k := range first.Nicknames {
		_, ok := second.Nicknames[k]
		if !ok {
			return false
		}
	}

	for k := range first.SpecialTxHashes {
		_, ok := second.SpecialTxHashes[k]
		if !ok {
			return false
		}
	}

	for k := range first.PreBlockArbiters {
		_, ok := second.PreBlockArbiters[k]
		if !ok {
			return false
		}
	}

	for k := range first.ProducerDepositMap {
		_, ok := second.ProducerDepositMap[k]
		if !ok {
			return false
		}
	}

	for k := range first.EmergencyInactiveArbiters {
		_, ok := second.EmergencyInactiveArbiters[k]
		if !ok {
			return false
		}
	}

	return first.VersionStartHeight == second.VersionStartHeight &&
		first.VersionEndHeight == second.VersionEndHeight && first.DPoSV2ActiveHeight == second.DPoSV2ActiveHeight
}

func randomDPOSStateKeyFrame() *state.StateKeyFrame {
	result := &state.StateKeyFrame{
		NodeOwnerKeys:             make(map[string]string),
		PendingProducers:          make(map[string]*state.Producer),
		ActivityProducers:         make(map[string]*state.Producer),
		InactiveProducers:         make(map[string]*state.Producer),
		CanceledProducers:         make(map[string]*state.Producer),
		IllegalProducers:          make(map[string]*state.Producer),
		PendingCanceledProducers:  make(map[string]*state.Producer),
		Votes:                     make(map[string]struct{}),
		DposV2VoteRights:          make(map[common.Uint168]common.Fixed64),
		DposVotes:                 make(map[common.Uint168]common.Fixed64),
		DposV2Votes:               make(map[common.Uint168]common.Fixed64),
		DepositOutputs:            make(map[string]common.Fixed64),
		DposV2RewardInfo:          make(map[string]common.Fixed64),
		DposV2RewardClaimingInfo:  make(map[string]common.Fixed64),
		DposV2RewardClaimedInfo:   make(map[string]common.Fixed64),
		Nicknames:                 make(map[string]struct{}),
		SpecialTxHashes:           make(map[common.Uint256]struct{}),
		PreBlockArbiters:          make(map[string]struct{}),
		ProducerDepositMap:        make(map[common.Uint168]struct{}),
		EmergencyInactiveArbiters: make(map[string]struct{}),
		VersionStartHeight:        rand.Uint32(),
		VersionEndHeight:          rand.Uint32(),
		DPoSV2ActiveHeight:        rand.Uint32(),
	}

	for i := 0; i < 5; i++ {
		result.NodeOwnerKeys[randomString()] = randomString()
		result.PendingProducers[randomString()] = randomProducer()
		result.ActivityProducers[randomString()] = randomProducer()
		result.InactiveProducers[randomString()] = randomProducer()
		result.CanceledProducers[randomString()] = randomProducer()
		result.IllegalProducers[randomString()] = randomProducer()
		result.PendingCanceledProducers[randomString()] = randomProducer()
		result.Votes[randomString()] = struct{}{}
		result.DposV2VoteRights[*randomUint168()] = randomFix64()
		result.DposVotes[*randomUint168()] = randomFix64()
		result.DposV2Votes[*randomUint168()] = randomFix64()
		result.DepositOutputs[randomString()] = common.Fixed64(rand.Uint64())
		result.DposV2RewardInfo[randomString()] = common.Fixed64(rand.Uint64())
		result.DposV2RewardClaimingInfo[randomString()] = common.Fixed64(rand.Uint64())
		result.DposV2RewardClaimedInfo[randomString()] = common.Fixed64(rand.Uint64())
		result.Nicknames[randomString()] = struct{}{}
		result.SpecialTxHashes[*randomHash()] = struct{}{}
		result.PreBlockArbiters[randomString()] = struct{}{}
		result.ProducerDepositMap[*randomProgramHash()] = struct{}{}
		result.EmergencyInactiveArbiters[randomString()] = struct{}{}
	}
	return result
}

func producerEqual(first *state.Producer, second *state.Producer) bool {
	if first.State() != second.State() ||
		first.RegisterHeight() != second.RegisterHeight() ||
		first.CancelHeight() != second.CancelHeight() ||
		first.InactiveSince() != second.InactiveSince() ||
		first.ActivateRequestHeight() != second.ActivateRequestHeight() ||
		first.IllegalHeight() != second.IllegalHeight() ||
		first.Penalty() != second.Penalty() ||
		first.Votes() != second.Votes() ||
		first.DposV2Votes() != second.DposV2Votes() {
		return false
	}

	info1 := first.Info()
	info2 := second.Info()
	return producerInfoEqual(&info1, &info2)
}

func producerInfoEqual(first *payload.ProducerInfo,
	second *payload.ProducerInfo) bool {
	if first.NickName != second.NickName ||
		first.Url != second.Url ||
		first.Location != second.Location ||
		first.NetAddress != second.NetAddress {
		return false
	}

	return bytes.Equal(first.OwnerPublicKey, second.OwnerPublicKey) &&
		bytes.Equal(first.NodePublicKey, second.NodePublicKey) &&
		bytes.Equal(first.Signature, second.Signature)
}

func rewardEqual(first *state.RewardData, second *state.RewardData) bool {
	if first.TotalVotesInRound != second.TotalVotesInRound {
		return false
	}

	return votesMapEqual(first.OwnerVotesInRound, second.OwnerVotesInRound)
}

func randomRewardData() *state.RewardData {
	result := state.NewRewardData()

	for i := 0; i < 5; i++ {
		result.OwnerVotesInRound[*randomProgramHash()] =
			common.Fixed64(rand.Uint64())
	}

	return result
}

func randomVotes() *common2.Output {
	return &common2.Output{
		AssetID:     *randomHash(),
		Value:       common.Fixed64(rand.Uint64()),
		OutputLock:  rand.Uint32(),
		ProgramHash: *randomProgramHash(),
		Type:        common2.OTVote,
		Payload: &outputpayload.VoteOutput{
			Version: byte(rand.Uint32()),
			Contents: []outputpayload.VoteContent{
				{
					VoteType: outputpayload.Delegate,
					CandidateVotes: []outputpayload.CandidateVotes{
						{randomFakePK(), 0},
					},
				},
			},
		},
	}
}

func randomHash() *common.Uint256 {
	a := make([]byte, 32)
	rand.Read(a)
	hash, _ := common.Uint256FromBytes(a)
	return hash
}

func randomProgramHash() *common.Uint168 {
	a := make([]byte, 21)
	rand.Read(a)
	hash, _ := common.Uint168FromBytes(a)
	return hash
}

func randomProducer() *state.Producer {
	p := &state.Producer{}
	p.SetInfo(payload.ProducerInfo{
		OwnerPublicKey: randomFakePK(),
		NodePublicKey:  randomFakePK(),
		NickName:       randomString(),
		Url:            randomString(),
		Location:       rand.Uint64(),
		NetAddress:     randomString(),
		Signature:      randomBytes(64),
	})
	p.SetState(state.ProducerState(rand.Uint32()))
	p.SetRegisterHeight(rand.Uint32())
	p.SetCancelHeight(rand.Uint32())
	p.SetInactiveSince(rand.Uint32())
	p.SetActivateRequestHeight(rand.Uint32())
	p.SetIllegalHeight(rand.Uint32())
	p.SetPenalty(common.Fixed64(rand.Uint64()))
	p.SetVotes(common.Fixed64(rand.Intn(10000) + 1))
	p.SetDposV2Votes(common.Fixed64(rand.Intn(10000) + 1))

	return p
}

func hashesEqual(first []*common.Uint168, second []*common.Uint168) bool {
	if len(first) != len(second) {
		return false
	}

	for _, vf := range first {
		found := false
		for _, vs := range second {
			if vs.IsEqual(*vf) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func votesMapEqual(first map[common.Uint168]common.Fixed64,
	second map[common.Uint168]common.Fixed64) bool {
	if len(first) != len(second) {
		return false
	}

	for k, vf := range first {
		if vs, ok := second[k]; !ok || vs != vf {
			return false
		}
	}
	return true
}

func arrayEqual(first []state.ArbiterMember, second []state.ArbiterMember) bool {
	if len(first) != len(second) {
		return false
	}

	for _, vf := range first {
		found := false
		for _, vs := range second {
			if arbiterMemberEqual(vf, vs) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func arbiterMemberEqual(first state.ArbiterMember, second state.ArbiterMember) bool {
	if bytes.Equal(first.GetNodePublicKey(), second.GetNodePublicKey()) &&
		bytes.Equal(first.GetOwnerPublicKey(), second.GetOwnerPublicKey()) &&
		first.GetType() == second.GetType() &&
		first.GetOwnerProgramHash().IsEqual(second.GetOwnerProgramHash()) {
		return true
	}

	return false
}

//	NextCRCArbitersMap    map[common.Uint168]ArbiterMember
func arbitersMapEqual(first map[common.Uint168]state.ArbiterMember,
	second map[common.Uint168]state.ArbiterMember) bool {
	if len(first) != len(second) {
		return false
	}

	for k, vf := range first {
		if vs, ok := second[k]; !ok || !arbiterMemberEqual(vs, vf) {
			return false
		}
	}
	return true
}
