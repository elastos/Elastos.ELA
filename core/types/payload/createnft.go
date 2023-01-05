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

const CreateNFTVersion byte = 0x00

// CreateNFT defines the transaction of NFT.
type CreateNFT struct {
	// nft id, hash of detailed vote information.
	ID common.Uint256

	// format address.
	StakeAddress string

	// genesis block hash of side chain.
	GenesisBlockHash common.Uint256
}

func (a *CreateNFT) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := a.Serialize(buf, version); err != nil {
		return []byte{0}
	}
	return buf.Bytes()
}

func (a *CreateNFT) Serialize(w io.Writer, version byte) error {
	err := a.SerializeUnsigned(w, version)
	if err != nil {
		return err
	}

	return nil
}

func (a *CreateNFT) SerializeUnsigned(w io.Writer, version byte) error {
	if err := a.ID.Serialize(w); err != nil {
		return errors.New("[CreateNFT], failed to serialize ID")
	}

	if err := common.WriteVarString(w, a.StakeAddress); err != nil {
		return errors.New("[CreateNFT], failed to serialize StakeAddress")
	}

	if err := a.GenesisBlockHash.Serialize(w); err != nil {
		return errors.New("[CreateNFT], failed to serialize GenesisBlockHash")
	}

	return nil
}

func (a *CreateNFT) Deserialize(r io.Reader, version byte) error {
	err := a.DeserializeUnsigned(r, version)
	if err != nil {
		return err
	}

	return nil
}

func (a *CreateNFT) DeserializeUnsigned(r io.Reader, version byte) error {
	if err := a.ID.Deserialize(r); err != nil {
		return errors.New("[CreateNFT], failed to deserialize ID")
	}

	to, err := common.ReadVarString(r)
	if err != nil {
		return errors.New("[CreateNFT], failed to deserialize StakeAddress")
	}
	a.StakeAddress = to

	if err := a.GenesisBlockHash.Deserialize(r); err != nil {
		return errors.New("[CreateNFT], failed to deserialize GenesisBlockHash")
	}

	return nil
}
