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
	//detail votes info referkey
	IDs []common.Uint256
	//owner OwnerStakeAddresses
	OwnerStakeAddresses []common.Uint168
	// genesis block hash of side chain.
	GenesisBlockHash common.Uint256
}

func (t *NFTDestroyFromSideChain) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := t.Serialize(buf, version); err != nil {
		return []byte{0}
	}

	return buf.Bytes()
}

func (t *NFTDestroyFromSideChain) Serialize(w io.Writer, version byte) error {
	if err := common.WriteVarUint(w, uint64(len(t.IDs))); err != nil {
		return errors.New("[NFTDestroyFromSideChain], ID count serialize failed")
	}

	for _, hash := range t.IDs {
		err := hash.Serialize(w)
		if err != nil {
			return errors.New("[NFTDestroyFromSideChain], ID serialize failed")
		}
	}

	if err := common.WriteVarUint(w, uint64(len(t.OwnerStakeAddresses))); err != nil {
		return errors.New("[NFTDestroyFromSideChain], OwnerStakeAddresses count serialize failed")
	}

	for _, hash := range t.OwnerStakeAddresses {
		err := hash.Serialize(w)
		if err != nil {
			return errors.New("[NFTDestroyFromSideChain], OwnerStakeAddresses serialize failed")
		}
	}

	if err := t.GenesisBlockHash.Serialize(w); err != nil {
		return errors.New("[NFTDestroyFromSideChain], failed to serialize GenesisBlockHash")
	}
	return nil
}

func (t *NFTDestroyFromSideChain) Deserialize(r io.Reader, version byte) error {
	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}
	t.IDs = make([]common.Uint256, 0)
	for i := uint64(0); i < count; i++ {
		var id common.Uint256
		err := id.Deserialize(r)
		if err != nil {
			return errors.New("[NFTDestroyFromSideChain], id deserialize failed.")
		}
		t.IDs = append(t.IDs, id)
	}

	t.OwnerStakeAddresses = make([]common.Uint168, 0)

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
		t.OwnerStakeAddresses = append(t.OwnerStakeAddresses, ownerStakeAddress)
	}

	if err := t.GenesisBlockHash.Deserialize(r); err != nil {
		return errors.New("[NFTDestroyFromSideChain], failed to deserialize GenesisBlockHash")
	}
	return nil
}
