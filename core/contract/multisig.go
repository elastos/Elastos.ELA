// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package contract

import (
	"bytes"
	"errors"
	"math/big"

	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/vm"
)

func CreateRevertToPOWRedeemScript(M int, pubkeys []*crypto.PublicKey) ([]byte, error) {
	if len(pubkeys) == 0 {
		return nil, errors.New("public keys is nil")
	}
	for _, pk := range pubkeys {
		if nil == pk {
			return nil, errors.New("public keys has nil public key")
		}
	}
	if !(M >= 1 && M <= len(pubkeys)) {
		return nil, errors.New("invalid M")
	}

	// Write M
	opCode := vm.OpCode(byte(crypto.PUSH1) + byte(M) - 1)
	buf := new(bytes.Buffer)
	buf.WriteByte(byte(opCode))

	//sort pubkey
	crypto.SortPublicKeys(pubkeys)

	for _, pubkey := range pubkeys {
		content, err := pubkey.EncodePoint(true)
		if err != nil {
			return nil, errors.New("[Contract],CreateMultiSigContract failed.")
		}
		buf.WriteByte(byte(len(content)))
		buf.Write(content)
	}

	// Write N
	N := len(pubkeys)
	opCode = vm.OpCode(byte(crypto.PUSH1) + byte(N) - 1)
	buf.WriteByte(byte(opCode))
	buf.WriteByte(vm.CHECKMULTISIG)

	return buf.Bytes(), nil
}

func CreateMultiSigRedeemScript(m int, pubkeys []*crypto.PublicKey) ([]byte, error) {
	if len(pubkeys) == 0 {
		return nil, errors.New("public keys is nil")
	}
	for _, pk := range pubkeys {
		if nil == pk {
			return nil, errors.New("public keys has nil public key")
		}
	}
	if !(m >= 1 && m <= len(pubkeys) && len(pubkeys) <= 24) {
		return nil, nil //TODO: add panic
	}

	sb := pg.NewProgramBuilder()
	sb.PushNumber(big.NewInt(int64(m)))

	//sort pubkey
	crypto.SortPublicKeys(pubkeys)

	for _, pubkey := range pubkeys {
		temp, err := pubkey.EncodePoint(true)
		if err != nil {
			return nil, errors.New("[Contract],CreateMultiSigContract failed.")
		}
		sb.PushData(temp)
	}

	sb.PushNumber(big.NewInt(int64(len(pubkeys))))
	sb.AddOp(vm.CHECKMULTISIG)

	return sb.ToArray(), nil
}

func CreateMultiSigContract(m int, pubkeys []*crypto.PublicKey) (*Contract, error) {
	redeemScript, err := CreateMultiSigRedeemScript(m, pubkeys)
	if err != nil {
		return nil, err
	}

	return &Contract{
		Code:   redeemScript,
		Prefix: PrefixMultiSig,
	}, nil
}

func CreateMultiSigContractByCode(code []byte) (*Contract, error) {
	return &Contract{
		Code:   code,
		Prefix: PrefixMultiSig,
	}, nil
}

func CreateSchnorrMultiSigRedeemScript(pubkey *crypto.PublicKey) ([]byte, error) {
	if pubkey == nil {
		return nil, errors.New("public keys is nil")
	}
	sb := pg.NewProgramBuilder()
	temp, err := pubkey.EncodePoint(true)
	if err != nil {
		return nil, errors.New("[Contract],CreateMultiSigContract failed.")
	}
	sb.PushData(temp)

	sb.AddOp(vm.SCHNORR)
	return sb.ToArray(), nil
}

func CreateSchnorrMultiSigContract(pubkeys *crypto.PublicKey) (*Contract, error) {
	redeemScript, err := CreateSchnorrMultiSigRedeemScript(pubkeys)
	if err != nil {
		return nil, err
	}

	return &Contract{
		Code:   redeemScript,
		Prefix: PrefixStandard,
	}, nil
}
