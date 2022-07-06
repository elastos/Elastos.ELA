// Copyright (c) 2017-2022 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"bytes"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/elastos/Elastos.ELA/crypto"

	"github.com/stretchr/testify/assert"
)

func TestOriginArbiter_Deserialize(t *testing.T) {
	a1, err := NewOriginArbiter(randomPublicKey())
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	a1.Serialize(buf)

	a2 := &originArbiter{}
	a2.Deserialize(buf)

	assert.Equal(t, a1.GetType(), a2.GetType())
	assert.True(t, bytes.Equal(a1.GetNodePublicKey(), a2.GetNodePublicKey()))
	assert.True(t, bytes.Equal(a1.GetOwnerPublicKey(), a2.GetOwnerPublicKey()))
	assert.True(t, a1.GetOwnerProgramHash().IsEqual(a2.GetOwnerProgramHash()))
}

func TestOriginArbiter_Clone(t *testing.T) {
	a1 := &originArbiter{key: make([]byte, crypto.NegativeBigLength)}
	for i := 0; i < crypto.NegativeBigLength; i++ {
		a1.key[i] = byte(i)
	}

	a2 := a1.Clone().(*originArbiter)
	assert.True(t, bytes.Equal(a1.key, a2.key))

	a2.key[0] = 10 // should only change data of a2
	assert.False(t, bytes.Equal(a1.key, a2.key))

	factor := strconv.FormatFloat(math.Log10(float64(72000)/7200*10), 'f', 2, 64)
	assert.True(t, strings.EqualFold(factor, "2.00"))
}

func randomPublicKey() []byte {
	_, pub, _ := crypto.GenerateKeyPair()
	result, _ := pub.EncodePoint(true)
	return result
}

