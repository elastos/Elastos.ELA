package payload

import (
	"errors"
	"io"

	"github.com/elastos/Elastos.ELA.Utility/common/serialization"
)

const WithdrawAssetPayloadVersion byte = 0x00

type WithdrawAsset struct {
	BlockHeight              uint32
	GenesisBlockAddress      string
	SideChainTransactionHash string
}

func (t *WithdrawAsset) Data(version byte) []byte {
	return []byte{0}
}

func (t *WithdrawAsset) Serialize(w io.Writer, version byte) error {
	if err := serialization.WriteUint32(w, t.BlockHeight); err != nil {
		return errors.New("[WithdrawAsset], BlockHeight serialize failed.")
	}
	if err := serialization.WriteVarString(w, t.GenesisBlockAddress); err != nil {
		return errors.New("[WithdrawAsset], BlockHeight serialize failed.")
	}
	if err := serialization.WriteVarString(w, t.SideChainTransactionHash); err != nil {
		return errors.New("[WithdrawAsset], BlockHeight serialize failed.")
	}

	return nil
}

func (t *WithdrawAsset) Deserialize(r io.Reader, version byte) error {
	height, err := serialization.ReadUint32(r)
	if err != nil {
		return errors.New("[WithdrawAsset], BlockHeight deserialize failed.")
	}
	address, err := serialization.ReadVarString(r)
	if err != nil {
		return errors.New("[WithdrawAsset], BlockHeight deserialize failed.")
	}
	hash, err := serialization.ReadVarString(r)
	if err != nil {
		return errors.New("[WithdrawAsset], BlockHeight deserialize failed.")
	}

	t.BlockHeight = height
	t.GenesisBlockAddress = address
	t.SideChainTransactionHash = hash

	return nil
}
