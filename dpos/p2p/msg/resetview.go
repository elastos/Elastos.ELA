// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package msg

import (
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto"
)

const DefaultResetViewDataSize = 100 // > 33+65

type ResetView struct {
	Sponsor []byte
	Sign    []byte
}

func (m *ResetView) CMD() string {
	return CmdResetConsensusView
}

func (m *ResetView) MaxLength() uint32 {
	return DefaultResetViewDataSize
}

func (m *ResetView) SerializeUnsigned(w io.Writer) error {
	return common.WriteVarBytes(w, m.Sponsor)
}

func (m *ResetView) DeserializeUnSigned(r io.Reader) error {
	sponsor, err := common.ReadVarBytes(r, crypto.NegativeBigLength, "public key")
	if err != nil {
		return err
	}
	m.Sponsor = sponsor
	return nil
}

func (m *ResetView) Serialize(w io.Writer) error {
	if err := m.SerializeUnsigned(w); err != nil {
		return err
	}
	return common.WriteVarBytes(w, m.Sign)
}

func (m *ResetView) Deserialize(r io.Reader) error {
	if err := m.DeserializeUnSigned(r); err != nil {
		return err
	}
	sign, err := common.ReadVarBytes(r, crypto.SignatureLength, "sign data")
	if err != nil {
		return err
	}
	m.Sign = sign
	return nil
}
