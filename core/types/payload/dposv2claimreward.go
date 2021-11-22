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
)

const (
	DposV2ClaimRewardVersion byte = 0x00
)

type DposV2ClaimReward struct {
	Amount common.Fixed64
}

func (a *DposV2ClaimReward) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := a.Serialize(buf, version); err != nil {
		return []byte{0}
	}
	return buf.Bytes()
}

func (a *DposV2ClaimReward) Serialize(w io.Writer, version byte) error {
	err := a.SerializeUnsigned(w, version)
	if err != nil {
		return err
	}
	return nil
}

func (a *DposV2ClaimReward) SerializeUnsigned(w io.Writer, version byte) error {
	err := a.Amount.Serialize(w)
	if err != nil {
		return errors.New("[DposV2ClaimReward], write amount failed")
	}

	return nil
}

func (a *DposV2ClaimReward) Deserialize(r io.Reader, version byte) error {
	err := a.DeserializeUnsigned(r, version)
	if err != nil {
		return err
	}
	return nil
}

func (a *DposV2ClaimReward) DeserializeUnsigned(r io.Reader, version byte) error {
	err := a.Amount.Deserialize(r)
	if err != nil {
		return errors.New("[DposV2ClaimReward], read amount failed")
	}

	return err
}
