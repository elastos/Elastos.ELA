// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package common

import (
	"crypto/rand"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortProgramHashes(t *testing.T) {
	count := 10
	hashes := make([]Uint168, 0, count)
	dupHashes := make([]Uint168, 0, count)
	for i := 0; i < count; i++ {
		var hash Uint168
		rand.Read(hash[:])
		hashes = append(hashes, hash)
		dupHashes = append(dupHashes, hash)
	}

	SortProgramHashByCodeHash(hashes)
	sort.Sort(byProgramHashes(dupHashes))

	for i, hash := range hashes {
		if !hash.IsEqual(dupHashes[i]) {
			t.Errorf("Sorted program hashes not the same")
		}
	}

	fmt.Println(hashes)
	fmt.Println(dupHashes)

}

type byProgramHashes []Uint168

func (a byProgramHashes) Len() int      { return len(a) }
func (a byProgramHashes) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byProgramHashes) Less(i, j int) bool {
	if a[i].Compare(a[j]) > 0 {
		return false
	} else {
		return true
	}
}

func TestIsLetterOrNumber(t *testing.T) {
	assert.True(t, IsLetterOrNumber("alice"))
	assert.True(t, IsLetterOrNumber("alice1"))
	assert.True(t, IsLetterOrNumber("Alice2"))
	assert.True(t, IsLetterOrNumber("123"))
	assert.False(t, IsLetterOrNumber("艾丽斯"))
	assert.False(t, IsLetterOrNumber(":alice"))
	assert.False(t, IsLetterOrNumber("alice bob"))
	assert.False(t, IsLetterOrNumber("(alice)"))
	assert.False(t, IsLetterOrNumber("("))
	assert.False(t, IsLetterOrNumber(")"))
	assert.False(t, IsLetterOrNumber("["))
	assert.False(t, IsLetterOrNumber("]"))
	assert.False(t, IsLetterOrNumber("{"))
	assert.False(t, IsLetterOrNumber("}"))
	assert.False(t, IsLetterOrNumber("<"))
	assert.False(t, IsLetterOrNumber(">"))
	assert.False(t, IsLetterOrNumber("+"))
	assert.False(t, IsLetterOrNumber("-"))
	assert.False(t, IsLetterOrNumber("*"))
	assert.False(t, IsLetterOrNumber("~"))
	assert.False(t, IsLetterOrNumber("!"))
	assert.False(t, IsLetterOrNumber("@"))
	assert.False(t, IsLetterOrNumber("#"))
	assert.False(t, IsLetterOrNumber("$"))
	assert.False(t, IsLetterOrNumber("%"))
	assert.False(t, IsLetterOrNumber("^"))
	assert.False(t, IsLetterOrNumber("&"))
	assert.False(t, IsLetterOrNumber(":"))
	assert.False(t, IsLetterOrNumber(";"))
	assert.False(t, IsLetterOrNumber("'"))
	assert.False(t, IsLetterOrNumber(""))
	assert.False(t, IsLetterOrNumber(","))
	assert.False(t, IsLetterOrNumber("."))
	assert.False(t, IsLetterOrNumber("?"))
	assert.False(t, IsLetterOrNumber("/"))
	assert.False(t, IsLetterOrNumber(" "))
}
