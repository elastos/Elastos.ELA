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
	ID                []common.Uint256 //detail votes info referkey
	OwnerStakeAddress []common.Uint168 //owner OwnerStakeAddress
}

func (t *NFTDestroyFromSideChain) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := t.Serialize(buf, version); err != nil {
		return []byte{0}
	}

	return buf.Bytes()
}

func (t *NFTDestroyFromSideChain) Serialize(w io.Writer, version byte) error {
	if err := common.WriteVarUint(w, uint64(len(t.ID))); err != nil {
		return errors.New("[NFTDestroyFromSideChain], ID count serialize failed")
	}

	for _, hash := range t.ID {
		err := hash.Serialize(w)
		if err != nil {
			return errors.New("[NFTDestroyFromSideChain], ID serialize failed")
		}
	}

	if err := common.WriteVarUint(w, uint64(len(t.OwnerStakeAddress))); err != nil {
		return errors.New("[NFTDestroyFromSideChain], OwnerStakeAddress count serialize failed")
	}

	for _, hash := range t.OwnerStakeAddress {
		err := hash.Serialize(w)
		if err != nil {
			return errors.New("[NFTDestroyFromSideChain], OwnerStakeAddress serialize failed")
		}
	}

	return nil
}

func (t *NFTDestroyFromSideChain) Deserialize(r io.Reader, version byte) error {
	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}
	t.ID = make([]common.Uint256, 0)
	for i := uint64(0); i < count; i++ {
		var id common.Uint256
		err := id.Deserialize(r)
		if err != nil {
			return errors.New("[NFTDestroyFromSideChain], id deserialize failed.")
		}
		t.ID = append(t.ID, id)
	}

	t.OwnerStakeAddress = make([]common.Uint168, 0)

	count, err = common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}
	for i := uint64(0); i < count; i++ {
		var ownerStakeAddress common.Uint168
		err := ownerStakeAddress.Deserialize(r)
		if err != nil {
			return errors.New("[NFTDestroyFromSideChain], ownerStakeAddress deserialize failed.")
		}
		t.OwnerStakeAddress = append(t.OwnerStakeAddress, ownerStakeAddress)
	}

	return nil
}
