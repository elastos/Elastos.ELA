// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package contract

import (
	"errors"
	"math/big"

	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/crypto"
)

// from btc, witness program script public: opcode_version + size + [pk]
const TapRootVersion int64 = 1

func CreateSchnorrRedeemScript(pubkey *crypto.PublicKey) ([]byte, error) {
	if nil == pubkey {
		return nil, errors.New("public Key is nil")
	}
	temp, err := pubkey.EncodePoint(true)
	if err != nil {
		return nil, errors.New("create schnorr redeem script, encode public key failed")
	}
	sb := program.NewProgramBuilder()
	sb.PushNumber(big.NewInt(TapRootVersion))
	sb.PushData(temp)

	return sb.ToArray(), nil
}

func CreateSchnorrContract(pubkeys *crypto.PublicKey) (*Contract, error) {
	redeemScript, err := CreateSchnorrRedeemScript(pubkeys)
	if err != nil {
		return nil, err
	}

	return &Contract{
		Code:   redeemScript,
		Prefix: PrefixStandard,
	}, nil
}
