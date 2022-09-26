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
	ReturnVotesVersionV0 byte = 0x00
	ReturnVotesVersionV1 byte = 0x01
)

type ReturnVotes struct {
	// target or to address
	ToAddr common.Uint168
	// code
	Code []byte
	// unstake value
	Value common.Fixed64
	// signature
	Signature []byte
}

func (p *ReturnVotes) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := p.Serialize(buf, version); err != nil {
		return []byte{0}
	}

	return buf.Bytes()
}

func (p *ReturnVotes) Serialize(w io.Writer, version byte) error {
	err := p.SerializeUnsigned(w, version)
	if err != nil {
		return err
	}

	if version == ReturnVotesVersionV0 {
		err = common.WriteVarBytes(w, p.Signature)
		if err != nil {
			return errors.New("[ReturnVotes], signature serialize failed")
		}
	}

	return nil
}

func (p *ReturnVotes) SerializeUnsigned(w io.Writer, version byte) error {
	if err := p.ToAddr.Serialize(w); err != nil {
		return errors.New("[ReturnVotes], ToAddr serialize failed")
	}

	if version == ReturnVotesVersionV0 {
		err := common.WriteVarBytes(w, p.Code)
		if err != nil {
			return errors.New("[ReturnVotes], Code serialize failed")
		}
	}

	if err := p.Value.Serialize(w); err != nil {
		return errors.New("[ReturnVotes], Value serialize failed")
	}
	return nil
}

func (p *ReturnVotes) DeserializeUnsigned(r io.Reader, version byte) error {
	var err error
	if err := p.ToAddr.Deserialize(r); err != nil {
		return errors.New("[ReturnVotes], ToAddr Deserialize failed")
	}

	if version == ReturnVotesVersionV0 {
		p.Code, err = common.ReadVarBytes(r, crypto.MaxMultiSignCodeLength, "code")
		if err != nil {
			return errors.New("[ReturnVotes], Code deserialize failed")
		}
	}

	if err := p.Value.Deserialize(r); err != nil {
		return errors.New("[ReturnVotes], Value Deserialize failed")
	}
	return nil
}

func (p *ReturnVotes) Deserialize(r io.Reader, version byte) error {
	err := p.DeserializeUnsigned(r, version)
	if err != nil {
		return err
	}

	if version == ReturnVotesVersionV0 {
		p.Signature, err = common.ReadVarBytes(r, crypto.MaxSignatureScriptLength, "signature")
		if err != nil {
			return errors.New("[ReturnVotes], signature deserialize failed")
		}
	}

	return nil
}
