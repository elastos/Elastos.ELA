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

const RevertToPOWVersion byte = 0x00

type RevertToPOW struct {
	StartPOWBlockHeight uint32
}

func (a *RevertToPOW) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := a.Serialize(buf, version); err != nil {
		return []byte{0}
	}

	return buf.Bytes()
}

func (a *RevertToPOW) Serialize(w io.Writer, version byte) error {
	err := common.WriteUint32(w, a.StartPOWBlockHeight)
	if err != nil {
		return errors.New("[RevertToPOW], failed to serialize StartPOWBlockHeight")
	}
	return nil
}

func (a *RevertToPOW) Deserialize(r io.Reader, version byte) error {
	var err error
	a.StartPOWBlockHeight, err = common.ReadUint32(r)
	if err != nil {
		return errors.New("[RevertToPOW], failed to deserialize StartPOWBlockHeight")
	}
	return nil
}
