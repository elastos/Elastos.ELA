// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package contract

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/utils/test"

	"github.com/stretchr/testify/assert"
)

func TestCalculateDepositAddr(t *testing.T) {
	test.SkipShort(t)
	publicKeyStrs := make([]string, 0)
	publicKeyStrs = append(publicKeyStrs, "0261056c3bb7fd2399a1e8e8ca00c9f213d9a9d8b5e986b75ee627ecdedbdadda1")
	publicKeyStrs = append(publicKeyStrs, "039a6c4f6b0c679bb8023ccae91340b6489c79d482d07d42aba1d52a9e85bc29af")
	publicKeyStrs = append(publicKeyStrs, "039fa77a2b64c3065023d6e0ef279dc87ffef7f634f28d5a58b707f2f6054385eb")
	publicKeyStrs = append(publicKeyStrs, "02f7042c66d5da58a41677f52eaeba5dd776454c3df3cfb8e84484e249e0c689bc")

	var publicKeys []*crypto.PublicKey
	for _, publicKeyStr := range publicKeyStrs {
		publicKeyBytes, _ := hex.DecodeString(publicKeyStr)
		publicKey, _ := crypto.DecodePoint(publicKeyBytes)
		publicKeys = append(publicKeys, publicKey)
	}

	multiCode, _ := CreateMultiSigRedeemScript(3, publicKeys)

	ct, err := CreateDepositContractByCode(multiCode)
	if err != nil {
		fmt.Println("error:", err)
	}
	addr, err := ct.ToProgramHash().ToAddress()
	if err != nil {
		fmt.Println("error 2:", err)
	}
	fmt.Println("addr:", addr)

}

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

func TestBasicAlgorithm(t *testing.T) {
	standardCodeStr := "2102cd62afdc81cde4b0a671991556e5b352e07d1c6ed0c95298618798f707f47b15ac"
	standardCodeByte, _ := hex.DecodeString(standardCodeStr)
	standardPubKey := common.GetPublicKeyFromCode(standardCodeByte)
	standardCodeHash, _ := PublicKeyToStandardCodeHash(standardPubKey)
	fmt.Println("standardCodeHash", standardCodeHash)

	multiCodeStr := "522103424727948233d29f3186222a8cad449f34cb0de3f2122196344064a1dc44c4db2102a8097e33e19987d53df6e52c7a" +
		"34516693c3179199b1889926be3c34029c98d92102cd62afdc81cde4b0a671991556e5b352e07d1c6ed0c95298618798f707f47b1553ae"
	multiCodeByte, _ := hex.DecodeString(multiCodeStr)
	ct, _ := CreateMultiSigContractByCode(multiCodeByte)
	fmt.Println("multiCode hash", ct.ToCodeHash())

	fmt.Println("multiCode ProgramHash", ct.ToProgramHash())
	fmt.Println("multiCode ProgramHash2", common.ToProgramHash(byte(PrefixStandard), multiCodeByte))

}
