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

const (
	NoBlock RevertType = iota
	NoProducers
	NoClaimDPOSNode
)

type RevertType byte

type RevertToPOW struct {
	Type          RevertType
	WorkingHeight uint32
}

func (r RevertType) String() string {
	switch r {
	case NoBlock:
		return "NoBlock"
	case NoProducers:
		return "NoProducers"
	case NoClaimDPOSNode:
		return "NoClaimDPOSNode"
	}
	return "Unknown"
}

func (a *RevertToPOW) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := a.Serialize(buf, version); err != nil {
		return []byte{0}
	}

	return buf.Bytes()
}

func (a *RevertToPOW) Serialize(w io.Writer, version byte) error {
	if err := common.WriteUint8(w, byte(a.Type)); err != nil {
		return errors.New("[RevertToPOW], failed to serialize Type")
	}

	if err := common.WriteElement(w, a.WorkingHeight); err != nil {
		return errors.New("[RevertToPOW], failed to serialize WorkingHeight")
	}
	return nil
}

func (a *RevertToPOW) Deserialize(r io.Reader, version byte) error {
	revertType, err := common.ReadUint8(r)
	if err != nil {
		return errors.New("[RevertToPOW], failed to deserialize Type")
	}
	a.Type = RevertType(revertType)
	if err := common.ReadElement(r, &a.WorkingHeight); err != nil {
		return errors.New("[RevertToPOW], failed to deserialize WorkingHeight")
	}
	return nil
}
