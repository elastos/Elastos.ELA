// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package blockchain

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/elastos/Elastos.ELA/test/unit"
	"math"
	"testing"
	"time"

	"github.com/elastos/Elastos.ELA/common"

	"github.com/stretchr/testify/assert"
)

const TestTimes int = 100000

func TestCheckTimeOfReword(t *testing.T) {
	testData := prepareData()
	start := time.Now()
	reward := calculateReward(testData)
	spend := time.Now().Sub(start)
	fmt.Printf("calculate %d rewards, time spend %s", len(reward), spend)

	assert.True(t, spend < time.Second)
}

func prepareData() []voterInfo {
	data := make([]voterInfo, TestTimes)
	for i := 0; i < TestTimes; i++ {
		data[i] = randomVoterInfo()
	}
	return data
}

func randomVoterInfo() voterInfo {
	return voterInfo{
		Address:  *unit.randomUint168(),
		Votes:    randomFix64(),
		LockTime: randomUint32(),
	}
}

func randomUint32() uint32 {
	var randNum uint32
	binary.Read(rand.Reader, binary.BigEndian, &randNum)
	return randNum
}

type voterInfo struct {
	Address  common.Uint168
	Votes    common.Fixed64
	LockTime uint32
}

func calculateReward(voters []voterInfo) map[common.Uint168]common.Fixed64 {
	// calculate total reward
	var totalVotes common.Fixed64
	for _, v := range voters {
		totalVotes += v.Votes * common.Fixed64(math.Log10(float64(v.LockTime)))
	}

	// calculate all rewards of each voter
	rewards := make(map[common.Uint168]common.Fixed64)
	for _, v := range voters {
		reward := v.Votes * common.Fixed64(math.Log10(float64(v.LockTime))) / totalVotes
		rewards[v.Address] = reward
	}

	return rewards
}

func randomFix64() common.Fixed64 {
	var randNum int64
	binary.Read(rand.Reader, binary.BigEndian, &randNum)
	return common.Fixed64(randNum)
}