// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package crypto

import (
	"encoding/hex"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortPublicKeys(t *testing.T) {
	count := 10
	publicKeys := make([]*PublicKey, 0, count)
	dupPubKeys := make([]*PublicKey, 0, count)
	for i := 0; i < count; i++ {
		_, public, err := GenerateKeyPair()
		if err != nil {
			t.Errorf("Generate key pair failed, error %s", err.Error())
		}
		publicKeys = append(publicKeys, public)
		dupPubKeys = append(dupPubKeys, public)
	}

	SortPublicKeys(publicKeys)
	sort.Sort(pubKeySlice(dupPubKeys))

	assert.Equal(t, dupPubKeys, publicKeys)
}

type pubKeySlice []*PublicKey

func (p pubKeySlice) Len() int { return len(p) }
func (p pubKeySlice) Less(i, j int) bool {
	r := p[i].X.Cmp(p[j].X)
	if r <= 0 {
		return true
	}
	return false
}
func (p pubKeySlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func TestEncryptDecrypt(t *testing.T) {
	priKey, pubKey, _ := GenerateKeyPair()

	message := []byte("Hello World!")

	cipher, err := Encrypt(pubKey, message)
	assert.NoError(t, err)

	m, err := Decrypt(priKey, cipher)
	assert.NoError(t, err)

	assert.Equal(t, message, m)

	pub := "03813e9ca3f1de5cc28db4ef3ca6726a91a2368dff52b053206456b91c45c8f576"
	priv := "03c01fca28701cd72330a723b271195ca8bbd1b6a80dd2a2a43993413701de5e"

	privateKey, _ := hex.DecodeString(priv)
	Px, Py := Curve.ScalarBaseMult(privateKey)
	fmt.Println(Px.String(),Py.String())

	pubf , _ := hex.DecodeString(pub)
	px, py := Unmarshal(Curve, pubf)
	fmt.Println(px.String(),py.String())


}
