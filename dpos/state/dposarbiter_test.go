// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"bytes"
	"github.com/elastos/Elastos.ELA/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDposArbiter_Deserialize(t *testing.T) {
	a, _ := NewDPoSArbiter(randomProducer())
	ar1 := a.(*dposArbiter)

	buf := new(bytes.Buffer)
	ar1.Serialize(buf)

	ar2 := &dposArbiter{}
	ar2.Deserialize(buf)

	assert.True(t, producerEqual(&ar1.producer, &ar2.producer))
	assert.True(t, ar1.ownerHash.IsEqual(ar2.ownerHash))
}

func TestDposArbiter_Clone(t *testing.T) {
	a, _ := NewDPoSArbiter(randomProducer())
	ar1 := a.(*dposArbiter)

	ar2 := ar1.Clone().(*dposArbiter)

	assert.True(t, producerEqual(&ar1.producer, &ar2.producer))
	assert.True(t, ar1.ownerHash.IsEqual(ar2.ownerHash))

	ar1.producer.info.NodePublicKey[0] = ar1.producer.info.NodePublicKey[0] + 1
	assert.False(t, producerEqual(&ar1.producer, &ar2.producer))
}

func stateKeyFrameEqual(first *StateKeyFrame, second *StateKeyFrame) bool {
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

func producerEqual(first *Producer, second *Producer) bool {
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

func rewardEqual(first *RewardData, second *RewardData) bool {
	if first.TotalVotesInRound != second.TotalVotesInRound {
		return false
	}

	return votesMapEqual(first.OwnerVotesInRound, second.OwnerVotesInRound)
}

func randomRewardData() *RewardData {
	result := NewRewardData()

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

func randomProducer() *Producer {
	p := &Producer{}
	p.SetInfo(payload.ProducerInfo{
		OwnerPublicKey: randomFakePK(),
		NodePublicKey:  randomFakePK(),
		NickName:       randomString(),
		Url:            randomString(),
		Location:       rand.Uint64(),
		NetAddress:     randomString(),
		Signature:      randomBytes(64),
	})
	p.SetState(ProducerState(rand.Uint32()))
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

func arrayEqual(first []ArbiterMember, second []ArbiterMember) bool {
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

func arbiterMemberEqual(first ArbiterMember, second ArbiterMember) bool {
	if bytes.Equal(first.GetNodePublicKey(), second.GetNodePublicKey()) &&
		bytes.Equal(first.GetOwnerPublicKey(), second.GetOwnerPublicKey()) &&
		first.GetType() == second.GetType() &&
		first.GetOwnerProgramHash().IsEqual(second.GetOwnerProgramHash()) {
		return true
	}

	return false
}

//	NextCRCArbitersMap    map[common.Uint168]ArbiterMember
func arbitersMapEqual(first map[common.Uint168]ArbiterMember,
	second map[common.Uint168]ArbiterMember) bool {
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

func randomFakePK() []byte {
	_, pub, _ := crypto.GenerateKeyPair()
	result, _ := pub.EncodePoint(true)
	return result
}
