// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"bytes"
	"encoding/hex"
	"math/rand"
	"strconv"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"

	"github.com/stretchr/testify/assert"
)

func Test_RandomIndex(t *testing.T) {
	var x = make([]byte, 8)
	blockHashStr := "303fdce09b22cdb99bf29cec7358bcc518c059d189a729103a7900ebfe356746"
	blockHash, _ := common.Uint256FromHexString(blockHashStr)
	copy(x, blockHash[24:])
	seed, _, _ := readi64(x)
	oriSeed := seed

	var first []int
	for i := 0; i < 100; i++ {
		seed++
		rand.Seed(seed)
		first = append(first, rand.Intn(100))
	}

	var second []int
	seed = oriSeed
	for i := 0; i < 100; i++ {
		seed++
		rand.Seed(seed)
		second = append(second, rand.Intn(100))
	}

	var third []int
	rand.Seed(oriSeed)
	for i := 0; i < 100; i++ {
		third = append(third, rand.Intn(100))
	}

	var fourth []int
	rand.Seed(oriSeed)
	for i := 0; i < 100; i++ {
		fourth = append(fourth, rand.Intn(100))
	}

	assert.Equal(t, first, second, "invalid random: seed++")
	assert.Equal(t, third, fourth, "invalid random: same seed")
}

func TestArbitrators_GetSortedProducers(t *testing.T) {
	producers := []int{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
		21, 22, 23, 24, 25, 26, 27, 28, 29, 30,
		31, 32, 33, 34, 35, 36, 37, 38, 39, 40,
		41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
		51, 52, 53, 54, 55, 56, 57, 58, 59, 60,
	}
	targetProducers := []int{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
		21, 22, 23, 30, 24, 25, 26, 27, 28, 29,
		31, 32, 33, 34, 35, 36, 37, 38, 39, 40,
		41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
		51, 52, 53, 54, 55, 56, 57, 58, 59, 60,
	}
	normalCount := 23
	candidateProducer := 30
	selectedCandidateIndex := 29
	candidateCounts := 24
	arbitratorsCount := 24
	newProducers := make([]int, 0, candidateCounts+arbitratorsCount)
	newProducers = append(newProducers, producers[:normalCount]...)
	newProducers = append(newProducers, candidateProducer)
	newProducers = append(newProducers, producers[normalCount:selectedCandidateIndex]...)
	newProducers = append(newProducers, producers[selectedCandidateIndex+1:]...)

	for i, p := range newProducers {
		assert.Equal(t, p, targetProducers[i])
	}
}

func TestArbitrators_GetSnapshot(t *testing.T) {
	var bestHeight uint32

	arbitrators, _ := NewArbitrators(&config.DefaultParams,
		nil, nil, nil,
		nil, nil, nil)
	arbitrators.RegisterFunction(func() uint32 { return bestHeight },
		func() *common.Uint256 { return &common.Uint256{} },
		nil, nil)

	// define three height versions:
	// firstSnapshotHeight < secondSnapshotHeight < bestHeight
	bestHeight = 30
	firstSnapshotHeight := uint32(10)
	firstSnapshotPk := randomFakePK()
	secondSnapshotHeight := uint32(20)
	secondSnapshotPk := randomFakePK()
	ar, _ := NewOriginArbiter(firstSnapshotPk)
	arbitrators.currentArbitrators = []ArbiterMember{ar}

	// take the first snapshot
	arbitrators.snapshot(firstSnapshotHeight)
	ar, _ = NewOriginArbiter(secondSnapshotPk)
	arbitrators.currentArbitrators = []ArbiterMember{ar}

	// height1
	frames := arbitrators.GetSnapshot(firstSnapshotHeight)
	assert.Equal(t, 1, len(frames))
	assert.True(t, bytes.Equal(firstSnapshotPk,
		frames[0].CurrentArbitrators[0].GetNodePublicKey()))

	// < height1
	frames = arbitrators.GetSnapshot(firstSnapshotHeight - 1)
	assert.Equal(t, []*CheckPoint{}, frames)

	// > height1
	frames = arbitrators.GetSnapshot(firstSnapshotHeight + 1)
	assert.Equal(t, 1, len(frames))
	assert.True(t, bytes.Equal(firstSnapshotPk,
		frames[0].CurrentArbitrators[0].GetNodePublicKey()))

	// height2
	frames = arbitrators.GetSnapshot(secondSnapshotHeight)
	assert.Equal(t, 1, len(frames))
	assert.True(t, bytes.Equal(firstSnapshotPk,
		frames[0].CurrentArbitrators[0].GetNodePublicKey()))

	// bestHeight
	frames = arbitrators.GetSnapshot(bestHeight)
	assert.Equal(t, 1, len(frames))
	assert.True(t, bytes.Equal(firstSnapshotPk,
		frames[0].CurrentArbitrators[0].GetNodePublicKey()))

	// bestHeight+1
	frames = arbitrators.GetSnapshot(bestHeight + 1)
	assert.Equal(t, 1, len(frames))
	assert.True(t, bytes.Equal(secondSnapshotPk,
		frames[0].CurrentArbitrators[0].GetNodePublicKey()))

	// > bestHeight
	frames = arbitrators.GetSnapshot(bestHeight + 1)
	assert.Equal(t, 1, len(frames))
	assert.True(t, bytes.Equal(secondSnapshotPk,
		frames[0].CurrentArbitrators[0].GetNodePublicKey()))

	// take the second snapshot
	arbitrators.snapshot(secondSnapshotHeight)
	ar, _ = NewOriginArbiter(randomFakePK())
	arbitrators.currentArbitrators = []ArbiterMember{ar}

	// height1
	frames = arbitrators.GetSnapshot(firstSnapshotHeight)
	assert.Equal(t, 1, len(frames))
	assert.True(t, bytes.Equal(firstSnapshotPk,
		frames[0].CurrentArbitrators[0].GetNodePublicKey()))

	// < height1
	frames = arbitrators.GetSnapshot(firstSnapshotHeight - 1)
	assert.Equal(t, []*CheckPoint{}, frames)

	// > height1
	frames = arbitrators.GetSnapshot(firstSnapshotHeight + 1)
	assert.Equal(t, 1, len(frames))
	assert.True(t, bytes.Equal(firstSnapshotPk,
		frames[0].CurrentArbitrators[0].GetNodePublicKey()))

	// height2
	frames = arbitrators.GetSnapshot(secondSnapshotHeight)
	assert.Equal(t, 1, len(frames))
	assert.True(t, bytes.Equal(secondSnapshotPk,
		frames[0].CurrentArbitrators[0].GetNodePublicKey()))

	// > height2
	frames = arbitrators.GetSnapshot(secondSnapshotHeight + 1)
	assert.Equal(t, 1, len(frames))
	assert.True(t, bytes.Equal(secondSnapshotPk,
		frames[0].CurrentArbitrators[0].GetNodePublicKey()))

	// bestHeight
	frames = arbitrators.GetSnapshot(bestHeight)
	assert.Equal(t, 1, len(frames))
	assert.True(t, bytes.Equal(secondSnapshotPk,
		frames[0].CurrentArbitrators[0].GetNodePublicKey()))

	// > bestHeight
	frames = arbitrators.GetSnapshot(bestHeight + 1)
	assert.Equal(t, 1, len(frames))
	assert.True(t, bytes.Equal(arbitrators.currentArbitrators[0].
		GetNodePublicKey(), frames[0].CurrentArbitrators[0].GetNodePublicKey()))

	// take snapshot more than MaxSnapshotLength
	loopSnapshotHeight := bestHeight
	bestHeight += 50
	for i := loopSnapshotHeight; i < loopSnapshotHeight+MaxSnapshotLength; i++ {
		arbitrators.snapshot(i)
	}
	assert.Equal(t, MaxSnapshotLength, len(arbitrators.snapshots))
	assert.Equal(t, MaxSnapshotLength, len(arbitrators.snapshotKeysDesc))
	_, exist := arbitrators.snapshots[firstSnapshotHeight]
	assert.False(t, exist)
	_, exist = arbitrators.snapshots[secondSnapshotHeight]
	assert.False(t, exist)
}

func randomFakePK() []byte {
	_, pub, _ := crypto.GenerateKeyPair()
	result, _ := pub.EncodePoint(true)
	return result
}

//func TestArbitrators_UsingProducerAsArbiter(t *testing.T) {
//	var bestHeight uint32 = 0
//	var param = config.DefaultParams
//	a, _ := NewArbitrators(&param,
//		nil, nil, nil, nil,
//		nil, nil)
//	a.State = NewState(&param, nil, nil, nil,
//		nil, nil, nil,
//		nil, nil)
//	a.crCommittee = state.NewCommittee(&param)
//	a.crCommittee.InElectionPeriod = true
//	a.RegisterFunction(func() uint32 { return bestHeight },
//		func() *common.Uint256 { return &common.Uint256{} },
//		nil, nil)
//	fakeActiveProducer(a)
//	claimedCR, mem3 := fakeCRMembers(a)
//	a.chainParams.PublicDPOSHeight = 3
//	a.chainParams.CRClaimDPOSNodeStartHeight = 5
//	a.chainParams.ChangeCommitteeNewCRHeight = 7
//	a.updateNextArbitrators(8, 8)
//	a.history.Commit(8)
//	a.changeCurrentArbitrators(9)
//	a.history.Commit(9)
//	assert.Equal(t, 36, len(a.currentArbitrators), "current arbiter num should be 36")
//	assert.True(t, existInCurrentArbiters(claimedCR, a.currentArbitrators))
//	assert.Equal(t, 50-10-24, len(a.currentCandidates), "candidate num should be 16")
//
//	// set member to inactive to check if crc arbiter is used as dpos public key
//	mem3.MemberState = state.MemberInactive
//	a.updateNextArbitrators(10, 10)
//	a.history.Commit(10)
//	a.changeCurrentArbitrators(11)
//	a.history.Commit(11)
//	hash, _ := contract.PublicKeyToStandardProgramHash(mem3.Info.Code[1 : len(mem3.Info.Code)-1])
//	mem3DposPubKey := a.currentCRCArbitersMap[*hash].GetNodePublicKey()
//	assert.Equal(t, 36, len(a.currentArbitrators), "current arbiter num should be 36")
//	assert.True(t, existInOriginalArbiters([][]byte{mem3DposPubKey}, a.chainParams.CRCArbiters))
//	assert.Equal(t, 50-9-24, len(a.currentCandidates), "candidate num should be 17")
//}

func existInOriginalArbiters(keys [][]byte, crcArbiters []string) bool {
	for _, k := range keys {
		for _, c := range crcArbiters {
			if c == hex.EncodeToString(k) {
				return true
			}
		}
	}
	return false
}

func existInCurrentArbiters(keys [][]byte, src []ArbiterMember) bool {
	for _, k := range keys {
		for _, a := range src {
			if bytes.Equal(a.GetNodePublicKey(), k) {
				return true
			}
		}
	}
	return false
}

func fakeActiveProducer(a *arbitrators) {
	// 50 producer
	for i := 0; i < 50; i++ {
		a.State.ActivityProducers[randomString()] = randomProducer()
	}
}

func fakeCRMembers(a *arbitrators) (claimedCR [][]byte, toBeUsedMember *state.CRMember) {
	claimedCR1 := randomPublicKey()
	did1 := *randomUint168()
	a.crCommittee.Members[did1] = &state.CRMember{
		Info: payload.CRInfo{
			Code:     getCode(randomPublicKey()),
			DID:      did1,
			NickName: "CR1",
		},
		DPOSPublicKey: claimedCR1,
	}

	claimedCR2 := randomPublicKey()
	did2 := *randomUint168()
	a.crCommittee.Members[did2] = &state.CRMember{
		Info: payload.CRInfo{
			Code:     getCode(randomPublicKey()),
			DID:      did2,
			NickName: "CR2",
		},
		DPOSPublicKey: claimedCR2,
	}

	did3 := *randomUint168()
	mem3 := &state.CRMember{
		Info: payload.CRInfo{
			Code:     getCode(randomPublicKey()),
			DID:      did3,
			NickName: "CR3",
		},
	}
	a.crCommittee.Members[did3] = mem3

	for i := 4; i <= 12; i++ {
		did := *randomUint168()
		a.crCommittee.Members[did] = &state.CRMember{
			Info: payload.CRInfo{
				Code:     getCode(randomPublicKey()),
				DID:      did,
				NickName: "CR" + strconv.Itoa(i),
			},
		}
	}
	return [][]byte{claimedCR1, claimedCR2}, mem3
}

func getCode(publicKey []byte) []byte {
	pk, _ := crypto.DecodePoint(publicKey)
	redeemScript, _ := contract.CreateStandardRedeemScript(pk)
	return redeemScript
}
