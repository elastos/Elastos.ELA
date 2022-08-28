// Copyright (c) 2017-2020 The Elastos Foundation
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

const CRInfoVersion byte = 0x00
const CRInfoDIDVersion byte = 0x01
const CRInfoSchnorrVersion byte = 0x02

// CRInfo defines the information of CR.
type CRInfo struct {
	Code      []byte
	CID       common.Uint168
	DID       common.Uint168
	NickName  string
	Url       string
	Location  uint64
	Signature []byte
}

func (a *CRInfo) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := a.Serialize(buf, version); err != nil {
		return []byte{0}
	}
	return buf.Bytes()
}

func (a *CRInfo) Serialize(w io.Writer, version byte) error {
	err := a.SerializeUnsigned(w, version)
	if err != nil {
		return err
	}

	if version != CRInfoSchnorrVersion {
		err = common.WriteVarBytes(w, a.Signature)
		if err != nil {
			return errors.New("[CRInfo], Signature serialize failed")
		}
	}

	return nil
}

func (a *CRInfo) SerializeUnsigned(w io.Writer, version byte) error {
	if version != CRInfoSchnorrVersion {
		err := common.WriteVarBytes(w, a.Code)
		if err != nil {
			return errors.New("[CRInfo], code serialize failed")
		}
	}

	if err := a.CID.Serialize(w); err != nil {
		return errors.New("[CRInfo], CID serialize failed")
	}

	if version > CRInfoVersion {
		if err := a.DID.Serialize(w); err != nil {
			return errors.New("[CRInfo], DID serialize failed")
		}
	}

	if err := common.WriteVarString(w, a.NickName); err != nil {
		return errors.New("[CRInfo], nickname serialize failed")
	}

	if err := common.WriteVarString(w, a.Url); err != nil {
		return errors.New("[CRInfo], url serialize failed")
	}

	if err := common.WriteUint64(w, a.Location); err != nil {
		return errors.New("[CRInfo], location serialize failed")
	}

	return nil
}

func (a *CRInfo) Deserialize(r io.Reader, version byte) error {
	err := a.DeserializeUnsigned(r, version)
	if err != nil {
		return err
	}

	if version != CRInfoSchnorrVersion {
		a.Signature, err = common.ReadVarBytes(r, crypto.MaxSignatureScriptLength, "signature")
		if err != nil {
			return errors.New("[CRInfo], signature deserialize failed")
		}
	}
	return nil
}

func (a *CRInfo) DeserializeUnsigned(r io.Reader, version byte) error {
	var err error
	if version != CRInfoSchnorrVersion {
		a.Code, err = common.ReadVarBytes(r, crypto.MaxMultiSignCodeLength, "code")
		if err != nil {
			return errors.New("[CRInfo], code deserialize failed")
		}
	}

	if err = a.CID.Deserialize(r); err != nil {
		return errors.New("[CRInfo], CID deserialize failed")
	}

	if version > CRInfoVersion {
		if err = a.DID.Deserialize(r); err != nil {
			return errors.New("[CRInfo], DID deserialize failed")
		}
	}

	a.NickName, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[CRInfo], nickName deserialize failed")
	}

	a.Url, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[CRInfo], url deserialize failed")
	}

	a.Location, err = common.ReadUint64(r)
	if err != nil {
		return errors.New("[CRInfo], location deserialize failed")
	}

	return nil
}
func (a *CRInfo) GetCodeHash() common.Uint160 {
	return *common.ToCodeHash(a.Code)
}

// CRMemberInfo defines the information of CR Member
type CRMemberInfo struct {
	Code        []byte
	CID         common.Uint168
	DID         common.Uint168
	DepositHash common.Uint168
	NickName    string
	Url         string
	Location    uint64
}

func (c *CRMemberInfo) Data() []byte {
	buf := new(bytes.Buffer)
	if err := c.Serialize(buf); err != nil {
		return []byte{0}
	}
	return buf.Bytes()
}

func (c *CRMemberInfo) Serialize(w io.Writer) error {
	err := common.WriteVarBytes(w, c.Code)
	if err != nil {
		return errors.New("[CRMemberInfo], Code serialize failed")
	}

	if err = c.CID.Serialize(w); err != nil {
		return errors.New("[CRMemberInfo], CID serialize failed")
	}

	if err = c.DID.Serialize(w); err != nil {
		return errors.New("[CRMemberInfo], DID serialize failed")
	}

	if err = c.DepositHash.Serialize(w); err != nil {
		return errors.New("[CRMemberInfo], DepositHash serialize failed")
	}

	err = common.WriteVarString(w, c.NickName)
	if err != nil {
		return errors.New("[CRMemberInfo], nickname serialize failed")
	}

	err = common.WriteVarString(w, c.Url)
	if err != nil {
		return errors.New("[CRMemberInfo], url serialize failed")
	}

	err = common.WriteUint64(w, c.Location)
	if err != nil {
		return errors.New("[CRMemberInfo], location serialize failed")
	}

	return nil
}

func (c *CRMemberInfo) Deserialize(r io.Reader) error {
	var err error
	c.Code, err = common.ReadVarBytes(r, crypto.MaxMultiSignCodeLength, "code")
	if err != nil {
		return errors.New("[CRMemberInfo], Code deserialize failed")
	}

	if err = c.CID.Deserialize(r); err != nil {
		return errors.New("[CRMemberInfo], CID deserialize failed")
	}

	if err = c.DID.Deserialize(r); err != nil {
		return errors.New("[CRMemberInfo], DID deserialize failed")
	}

	if err = c.DepositHash.Deserialize(r); err != nil {
		return errors.New("[CRMemberInfo], DepositHash deserialize failed")
	}

	c.NickName, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[CRMemberInfo], nickName deserialize failed")
	}

	c.Url, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[CRMemberInfo], Url deserialize failed")
	}

	c.Location, err = common.ReadUint64(r)
	if err != nil {
		return errors.New("[CRMemberInfo], location deserialize failed")
	}

	return nil
}
func (c *CRMemberInfo) GetCodeHash() common.Uint160 {
	return *common.ToCodeHash(c.Code)
}
