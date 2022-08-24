// Copyright (c) 2017-2022 The Elastos Foundation
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
	Amount    common.Fixed64
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
	err := a.Amount.Serialize(w)
	if err != nil {
		return errors.New("[DPoSV2ClaimReward], write amount failed")
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
	err := a.Amount.Deserialize(r)
	if err != nil {
		return errors.New("[DPoSV2ClaimReward], read amount failed")
	}

	return err
}
