// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto"
)

type originArbiter struct {
	key       []byte
	ownerHash common.Uint168
}

func (o *originArbiter) Serialize(w io.Writer) error {
	if err := o.ownerHash.Serialize(w); err != nil {
		return err
	}
	return common.WriteVarBytes(w, o.key)
}

func (o *originArbiter) Deserialize(r io.Reader) (err error) {
	if err = o.ownerHash.Deserialize(r); err != nil {
		return
	}
	o.key, err = common.ReadVarBytes(r, crypto.NegativeBigLength,
		"public key")
	return
}

func (o *originArbiter) GetType() ArbiterType {
	return Origin
}

func (o *originArbiter) GetOwnerPublicKey() []byte {
	return o.key
}

func (o *originArbiter) GetOwnerProgramHash() common.Uint168 {
	return o.ownerHash
}

func (o *originArbiter) GetNodePublicKey() []byte {
	return o.key
}

func (o *originArbiter) IsNormal() bool {
	return true
}

func (o *originArbiter) Clone() ArbiterMember {
	k := make([]byte, len(o.key))
	copy(k, o.key)
	return &originArbiter{key: k, ownerHash: o.ownerHash}
}

func NewOriginArbiter(key []byte) (ArbiterMember, error) {
	hash, err := GetOwnerKeyStandardProgramHash(key)
	if err != nil {
		return nil, err
	}
	return &originArbiter{
		key:       key,
		ownerHash: *hash,
	}, nil
}
