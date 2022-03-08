// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import (
	"bytes"
	"errors"
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto"
)

const (
	UnstakeVersion byte = 0x00
)

type Unstake struct {
	//target or to address
	ToAddr common.Uint168
	// code
	Code []byte
	//unstake value
	Value common.Fixed64
	//signature
	Signature []byte
}

func (p *Unstake) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := p.Serialize(buf, version); err != nil {
		return []byte{0}
	}

	return buf.Bytes()
}

func (p *Unstake) Serialize(w io.Writer, version byte) error {
	err := p.SerializeUnsigned(w, version)
	if err != nil {
		return err
	}
	err = common.WriteVarBytes(w, p.Signature)
	if err != nil {
		return errors.New("[Unstake], signature serialize failed")
	}
	return nil
}

func (p *Unstake) SerializeUnsigned(w io.Writer, version byte) error {
	if err := p.ToAddr.Serialize(w); err != nil {
		return errors.New("[Unstake], ToAddr serialize failed")
	}

	err := common.WriteVarBytes(w, p.Code)
	if err != nil {
		return errors.New("[Unstake], Code serialize failed")
	}

	if err := p.Value.Serialize(w); err != nil {
		return errors.New("[Unstake], Value serialize failed")
	}
	return nil
}

func (p *Unstake) DeserializeUnsigned(r io.Reader, version byte) error {
	var err error
	if err := p.ToAddr.Deserialize(r); err != nil {
		return errors.New("[Unstake], ToAddr Deserialize failed")
	}

	p.Code, err = common.ReadVarBytes(r, crypto.MaxMultiSignCodeLength, "code")
	if err != nil {
		return errors.New("[Unstake], Code deserialize failed")
	}

	if err := p.Value.Deserialize(r); err != nil {
		return errors.New("[Unstake], Value Deserialize failed")
	}
	return nil
}

func (p *Unstake) Deserialize(r io.Reader, version byte) error {
	err := p.DeserializeUnsigned(r, version)
	if err != nil {
		return err
	}
	p.Signature, err = common.ReadVarBytes(r, crypto.SignatureLength, "signature")
	if err != nil {
		return errors.New("[Unstake], signature deserialize failed")
	}
	return nil
}
