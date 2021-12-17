// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package contract

import (
	"encoding/hex"
	"github.com/elastos/Elastos.ELA/common"
	"testing"

	"github.com/elastos/Elastos.ELA/crypto"

	"github.com/stretchr/testify/assert"
)

func TestToProgramHash(t *testing.T) {
	// Exit address
	publicKeyHex := "022c9652d3ad5cc065aa9147dc2ad022f80001e8ed233de20f352950d351d472b7"
	publicKey, err := hex.DecodeString(publicKeyHex)
	pub, _ := crypto.DecodePoint(publicKey)
	ct, err := CreateStandardContract(pub)
	if err != nil {
		t.Errorf("[PublicKeyToStandardProgramHash] failed")
	}
	programHash := ct.ToProgramHash()
	addr, _ := programHash.ToAddress()
	if !assert.Equal(t, "ENTogr92671PKrMmtWo3RLiYXfBTXUe13Z", addr) {
		t.FailNow()
	}

	pub1, _ := hex.DecodeString("03943E045CBCC5845C6FBE2E151998640DB770C68CC65E7CA9BC842DC59ECA3358")
	var code [33]byte
	code[0] = 0xAC
	copy(code[1:], pub1)
	con := Contract{
		Code:   code[:],
		Prefix: PrefixStandard,
	}
	uint168First := con.ToProgramHash()
	ct1, _ := CreateStakeContractByCode(code[:])
	stakeAddress := ct1.ToProgramHash()
	b := stakeAddress.Bytes()
	b[0] = byte(PrefixStandard)
	uint168Second, _ := common.Uint168FromBytes(b)
	if !assert.Equal(t, uint168First, uint168Second) {
		t.FailNow()
	}
}
