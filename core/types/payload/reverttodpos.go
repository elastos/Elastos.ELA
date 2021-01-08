// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import (
	"bytes"
	"io"

	"github.com/elastos/Elastos.ELA/common"
)

const WorkHeightInterval = 10
const RevertToDPOSVersion byte = 0x00

type RevertToDPOS struct {
	WorkHeightInterval     uint32
	RevertToPOWBlockHeight uint32
}

func (i *RevertToDPOS) GetBlockHeight() uint32 {
	return i.WorkHeightInterval
}

func (i *RevertToDPOS) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := i.Serialize(buf, version); err != nil {
		return []byte{0}
	}
	return buf.Bytes()
}

func (i *RevertToDPOS) SerializeUnsigned(w io.Writer, version byte) error {
	if err := common.WriteUint32(w, i.WorkHeightInterval); err != nil {
		return err
	}
	if err := common.WriteUint32(w, i.RevertToPOWBlockHeight); err != nil {
		return err
	}
	return nil
}

func (i *RevertToDPOS) Serialize(w io.Writer, version byte) error {
	if err := i.SerializeUnsigned(w, version); err != nil {
		return err
	}
	return nil
}

func (i *RevertToDPOS) DeserializeUnsigned(r io.Reader,
	version byte) (err error) {
	if i.WorkHeightInterval, err = common.ReadUint32(r); err != nil {
		return err
	}
	if i.RevertToPOWBlockHeight, err = common.ReadUint32(r); err != nil {
		return err
	}
	return err
}

func (i *RevertToDPOS) Deserialize(r io.Reader,
	version byte) (err error) {
	if err = i.DeserializeUnsigned(r, version); err != nil {
		return err
	}
	return nil
}
