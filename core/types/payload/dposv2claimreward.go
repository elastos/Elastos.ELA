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

const (
	DposV2ClaimRewardVersion byte = 0x00
)

type DPoSV2ClaimReward struct {
	// target or to address
	ToAddr common.Uint168
	// code
	Code []byte
	// reward value
	Value common.Fixed64
	// signature
	Signature []byte
}

func (a *DPoSV2ClaimReward) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := a.Serialize(buf, version); err != nil {
		return []byte{0}
	}
	return buf.Bytes()
}

func (a *DPoSV2ClaimReward) Serialize(w io.Writer, version byte) error {
	err := a.SerializeUnsigned(w, version)
	if err != nil {
		return err
	}

	err = common.WriteVarBytes(w, a.Signature)
	if err != nil {
		return errors.New("[DPoSV2ClaimReward], signature serialize failed")
	}
	return nil
}

func (a *DPoSV2ClaimReward) SerializeUnsigned(w io.Writer, version byte) error {
	if err := a.ToAddr.Serialize(w); err != nil {
		return errors.New("[DPoSV2ClaimReward], ToAddr serialize failed")
	}

	err := common.WriteVarBytes(w, a.Code)
	if err != nil {
		return errors.New("[DPoSV2ClaimReward], Code serialize failed")
	}

	if err := a.Value.Serialize(w); err != nil {
		return errors.New("[DPoSV2ClaimReward], Value serialize failed")
	}
	return nil
}

func (a *DPoSV2ClaimReward) Deserialize(r io.Reader, version byte) error {
	err := a.DeserializeUnsigned(r, version)
	if err != nil {
		return err
	}

	a.Signature, err = common.ReadVarBytes(r, crypto.MaxSignatureScriptLength, "signature")
	if err != nil {
		return errors.New("[DPoSV2ClaimReward], signature deserialize failed")
	}
	return nil
}

func (a *DPoSV2ClaimReward) DeserializeUnsigned(r io.Reader, version byte) error {
	var err error
	if err := a.ToAddr.Deserialize(r); err != nil {
		return errors.New("[DPoSV2ClaimReward], ToAddr Deserialize failed")
	}

	a.Code, err = common.ReadVarBytes(r, crypto.MaxMultiSignCodeLength, "code")
	if err != nil {
		return errors.New("[DPoSV2ClaimReward], Code deserialize failed")
	}

	if err := a.Value.Deserialize(r); err != nil {
		return errors.New("[DPoSV2ClaimReward], Value Deserialize failed")
	}
	return nil
}
