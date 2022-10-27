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

const NFTDestroyFromSideChainVersion byte = 0x00

type NFTDestroyFromSideChain struct {
	ID1       common.Uint256 //detail votes info referkey
	OwnerCode []byte         //owner Code
}

func (t *NFTDestroyFromSideChain) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := t.Serialize(buf, version); err != nil {
		return []byte{0}
	}

	return buf.Bytes()
}

func (t *NFTDestroyFromSideChain) Serialize(w io.Writer, version byte) error {
	if err := t.ID1.Serialize(w); err != nil {
		return errors.New(
			"failed to serialize ID1")
	}
	if err := common.WriteVarBytes(w, t.OwnerCode); err != nil {
		return errors.New("failed to serialize OwnerCode")
	}
	return nil
}

func (t *NFTDestroyFromSideChain) Deserialize(r io.Reader, version byte) error {
	var err error
	if err = t.ID1.Deserialize(r); err != nil {
		return errors.New("failed to deserialize ID1")
	}

	t.OwnerCode, err = common.ReadVarBytes(r, crypto.MaxMultiSignCodeLength, "OwnerCode")
	if err != nil {
		return errors.New("[NFTDestroyFromSideChain], OwnerCode deserialize failed")
	}
	return nil
}
