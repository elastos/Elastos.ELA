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

const ResponseRevertToDPOSLength = 512

type ResponseRevertToDPOS struct {
	TxHash common.Uint256
	Signer []byte
	Sign   []byte
}

func (i *ResponseRevertToDPOS) CMD() string {
	return CmdResponseRevertToDPOS
}

func (i *ResponseRevertToDPOS) MaxLength() uint32 {
	return ResponseRevertToDPOSLength
}

func (i *ResponseRevertToDPOS) Serialize(w io.Writer) error {
	if err := i.SerializeUnsigned(w); err != nil {
		return err
	}

	if err := common.WriteVarBytes(w, i.Sign); err != nil {
		return err
	}
	return nil
}

func (i *ResponseRevertToDPOS) SerializeUnsigned(w io.Writer) error {
	if err := i.TxHash.Serialize(w); err != nil {
		return err
	}

	if err := common.WriteVarBytes(w, i.Signer); err != nil {
		return err
	}
	return nil
}

func (i *ResponseRevertToDPOS) Deserialize(r io.Reader) (err error) {
	if err = i.DeserializeUnsigned(r); err != nil {
		return err
	}

	if i.Sign, err = common.ReadVarBytes(r, crypto.SignatureLength, "sign data"); err != nil {
		return err
	}
	return err
}

func (i *ResponseRevertToDPOS) DeserializeUnsigned(r io.Reader) (err error) {
	if err = i.TxHash.Deserialize(r); err != nil {
		return err
	}

	if i.Signer, err = common.ReadVarBytes(r, crypto.NegativeBigLength, "public key"); err != nil {
		return err
	}
	return err
}
