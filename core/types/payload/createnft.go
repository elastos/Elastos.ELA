// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import (
	"bytes"
	"errors"
	"github.com/elastos/Elastos.ELA/crypto"
	"io"

	"github.com/elastos/Elastos.ELA/common"
)

const CreateNFTVersion byte = 0x00
const CreateNFTVersion2 byte = 0x01

// CreateNFT defines the transaction of NFT.
type CreateNFT struct {
	// Referkey is the hash of detailed vote information
	// NFT ID: hash of (detailed vote information + createNFT tx hash).
	ReferKey common.Uint256

	// stake address of detailed vote.
	StakeAddress string

	// genesis block hash of side chain.
	GenesisBlockHash common.Uint256

	// the start height of votes
	StartHeight uint32

	// the end height of votes: start height + lock time.
	EndHeight uint32

	// the DPoS 2.0 votes.
	Votes common.Fixed64

	// the DPoS 2.0 vote rights.
	VoteRights common.Fixed64

	// the votes to the producer, and TargetOwnerPublicKey is the producer's
	// owner key.
	TargetOwnerKey []byte
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
	if err := a.ReferKey.Serialize(w); err != nil {
		return errors.New("[CreateNFT], failed to serialize ID")
	}

	if err := common.WriteVarString(w, a.StakeAddress); err != nil {
		return errors.New("[CreateNFT], failed to serialize StakeAddress")
	}

	if err := a.GenesisBlockHash.Serialize(w); err != nil {
		return errors.New("[CreateNFT], failed to serialize GenesisBlockHash")
	}

	if version >= CreateNFTVersion2 {
		if err := common.WriteUint32(w, a.StartHeight); err != nil {
			return errors.New("[CreateNFT], failed to serialize StartHeight")
		}
		if err := common.WriteUint32(w, a.EndHeight); err != nil {
			return errors.New("[CreateNFT], failed to serialize EndHeight")
		}
		if err := a.Votes.Serialize(w); err != nil {
			return errors.New("[CreateNFT], failed to serialize Votes")
		}
		if err := a.VoteRights.Serialize(w); err != nil {
			return errors.New("[CreateNFT], failed to serialize VoteRights")
		}
		if err := common.WriteVarBytes(w, a.TargetOwnerKey); err != nil {
			return errors.New("[CreateNFT], failed to serialize TargetOwnerPublicKey")
		}
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
	if err := a.ReferKey.Deserialize(r); err != nil {
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

	if version >= CreateNFTVersion2 {

		if a.StartHeight, err = common.ReadUint32(r); err != nil {
			return errors.New("[CreateNFT], failed to deserialize StartHeight")
		}
		if a.EndHeight, err = common.ReadUint32(r); err != nil {
			return errors.New("[CreateNFT], failed to deserialize EndHeight")
		}
		if err := a.Votes.Deserialize(r); err != nil {
			return errors.New("[CreateNFT], failed to deserialize Votes")
		}
		if err := a.VoteRights.Deserialize(r); err != nil {
			return errors.New("[CreateNFT], failed to deserialize VoteRights")
		}
		if a.TargetOwnerKey, err = common.ReadVarBytes(r, crypto.MaxMultiSignCodeLength, "TargetOwnerKey"); err != nil {
			return errors.New("[CreateNFT], failed to deserialize TargetOwnerKey")
		}
	}

	return nil
}

type NFTInfo struct {
	ReferKey         common.Uint256
	GenesisBlockHash common.Uint256
	CreateNFTTxHash  common.Uint256
}

func (n *NFTInfo) Serialize(w io.Writer) error {
	if err := n.ReferKey.Serialize(w); err != nil {
		return errors.New("[NFTInfo], failed to serialize ReferKey")
	}
	if err := n.GenesisBlockHash.Serialize(w); err != nil {
		return errors.New("[NFTInfo], failed to serialize GenesisBlockHash")
	}
	if err := n.CreateNFTTxHash.Serialize(w); err != nil {
		return errors.New("[NFTInfo], failed to serialize CreateNFTTxHash")
	}

	return nil
}

func (n *NFTInfo) Deserialize(r io.Reader) error {
	if err := n.ReferKey.Deserialize(r); err != nil {
		return errors.New("[NFTInfo], failed to deserialize ReferKey")
	}
	if err := n.GenesisBlockHash.Deserialize(r); err != nil {
		return errors.New("[NFTInfo], failed to deserialize GenesisBlockHash")
	}
	if err := n.CreateNFTTxHash.Deserialize(r); err != nil {
		return errors.New("[NFTInfo], failed to deserialize CreateNFTTxHash")
	}
	return nil
}
