// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package unit

import (
	"bytes"
	"crypto/rand"
	rand2 "math/rand"
	"testing"

	"github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"

	"github.com/stretchr/testify/assert"
)

func TestCandidate_Deserialize(t *testing.T) {
	candidate1 := randomCandidate()

	buf := new(bytes.Buffer)
	candidate1.Serialize(buf)

	candidate2 := &state.Candidate{}
	candidate2.Deserialize(buf)

	assert.True(t, candidateEqual(candidate1, candidate2))
}

func candidateEqual(first *state.Candidate, second *state.Candidate) bool {
	return crInfoEqual(&first.Info, &second.Info) &&
		first.State == second.State && first.Votes == second.Votes &&
		first.RegisterHeight == second.RegisterHeight &&
		first.CancelHeight == second.CancelHeight &&
		first.DepositHash.IsEqual(second.DepositHash)
}

func depositInfoEqual(first *state.DepositInfo, second *state.DepositInfo) bool {
	return first.DepositAmount == second.DepositAmount &&
		first.Penalty == second.Penalty &&
		first.TotalAmount == second.TotalAmount
}

func crInfoEqual(first *payload.CRInfo, second *payload.CRInfo) bool {
	if !bytes.Equal(first.Code, second.Code) ||
		!first.CID.IsEqual(second.CID) ||
		first.NickName != second.NickName ||
		first.Url != second.Url ||
		first.Location != second.Location {
		return false
	}
	return true
}

func randomCRInfo() *payload.CRInfo {
	code := randomBytes(34)
	return &payload.CRInfo{
		Code:     code,
		CID:      *getCID(code),
		DID:      *getDID(code),
		NickName: randomString(),
		Url:      randomString(),
		Location: rand2.Uint64(),
	}
}

func getDID(code []byte) *common.Uint168 {
	didCode := make([]byte, len(code))
	copy(didCode, code)
	didCode = append(didCode[:len(code)-1], common.DID)
	ct1, _ := contract.CreateCRIDContractByCode(didCode)
	return ct1.ToProgramHash()
}

func randomCandidate() *state.Candidate {
	return &state.Candidate{
		Info:           *randomCRInfo(),
		State:          state.CandidateState(rand2.Uint32()),
		Votes:          common.Fixed64(rand2.Int63()),
		RegisterHeight: rand2.Uint32(),
		CancelHeight:   rand2.Uint32(),
		DepositHash:    *randomUint168(),
	}
}

func randomUint256() *common.Uint256 {
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	result, _ := common.Uint256FromBytes(randBytes)

	return result
}

func randomPublicKey() []byte {
	_, pub, _ := crypto.GenerateKeyPair()
	result, _ := pub.EncodePoint(true)
	return result
}
