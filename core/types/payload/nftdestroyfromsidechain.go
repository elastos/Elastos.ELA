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

const NFTDestroyFromSideChainVersion byte = 0x00

type NFTDestroyFromSideChain struct {
	ID                common.Uint256 //detail votes info referkey
	OwnerStakeAddress common.Uint168 //owner OwnerStakeAddress
}

func (t *NFTDestroyFromSideChain) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := t.Serialize(buf, version); err != nil {
		return []byte{0}
	}

	return buf.Bytes()
}

func (t *NFTDestroyFromSideChain) Serialize(w io.Writer, version byte) error {

	if err := t.ID.Serialize(w); err != nil {
		return errors.New(
			"failed to serialize ID")
	}
	if err := t.OwnerStakeAddress.Serialize(w); err != nil {
		return errors.New(
			"failed to serialize OwnerStakeAddress")
	}
	return nil
}

func (t *NFTDestroyFromSideChain) Deserialize(r io.Reader, version byte) error {
	var err error

	if err = t.ID.Deserialize(r); err != nil {
		return errors.New("failed to deserialize ID")
	}
	if err = t.OwnerStakeAddress.Deserialize(r); err != nil {
		return errors.New("failed to deserialize ID")
	}
	return nil
}
