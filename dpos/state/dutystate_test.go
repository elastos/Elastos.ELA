package state

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDutyState_SortArbiters(t *testing.T) {
	arbiters := make([][]byte, 10)
	for i := range arbiters {
		pubKey := make([]byte, 33)
		rand.Read(pubKey)
		arbiters[i] = pubKey
	}

	copy := make([][]byte, 10)
	for i := range arbiters {
		copy[i] = arbiters[i]
	}

	sort.Slice(arbiters, func(i, j int) bool {
		stri := hex.EncodeToString(arbiters[i])
		strj := hex.EncodeToString(arbiters[j])
		return strings.Compare(stri, strj) < 0
	})

	sort.Slice(copy, func(i, j int) bool {
		return bytes.Compare(copy[i], copy[j]) < 0
	})

	assert.Equal(t, arbiters, copy)
}
