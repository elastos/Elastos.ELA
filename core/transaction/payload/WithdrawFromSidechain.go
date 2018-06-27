package payload

import (
	"bytes"
	"errors"
	"io"

	"github.com/elastos/Elastos.ELA.Utility/common"
)

const WithdrawFromSideChainPayloadVersion byte = 0x00

type WithdrawFromSideChain struct {
	BlockHeight                uint32
	GenesisBlockAddress        string
	SideChainTransactionHashes []common.Uint256
}

func (t *WithdrawFromSideChain) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := t.Serialize(buf, version); err != nil {
		return []byte{0}
	}

	return buf.Bytes()
}

func (t *WithdrawFromSideChain) Serialize(w io.Writer, version byte) error {
	if err := common.WriteUint32(w, t.BlockHeight); err != nil {
		return errors.New("[WithdrawFromSideChain], BlockHeight serialize failed.")
	}
	if err := common.WriteVarString(w, t.GenesisBlockAddress); err != nil {
		return errors.New("[WithdrawFromSideChain], GenesisBlockAddress serialize failed.")
	}

	if err := common.WriteVarUint(w, uint64(len(t.SideChainTransactionHashes))); err != nil {
		return errors.New("[WithdrawFromSideChain], SideChainTransactionHashes length serialize failed")
	}

	for _, hash := range t.SideChainTransactionHashes {
		err := hash.Serialize(w)
		if err != nil {
			return errors.New("[WithdrawFromSideChain], SideChainTransactionHashes serialize failed")
		}
	}
	return nil
}

func (t *WithdrawFromSideChain) Deserialize(r io.Reader, version byte) error {
	height, err := common.ReadUint32(r)
	if err != nil {
		return errors.New("[WithdrawFromSideChain], BlockHeight deserialize failed.")
	}
	address, err := common.ReadVarString(r)
	if err != nil {
		return errors.New("[WithdrawFromSideChain], GenesisBlockAddress deserialize failed.")
	}

	length, err := common.ReadVarUint(r, 0)
	if err != nil {
		return errors.New("[WithdrawFromSideChain], SideChainTransactionHashes length deserialize failed")
	}

	t.SideChainTransactionHashes = nil
	t.SideChainTransactionHashes = make([]common.Uint256, length)
	for i := uint64(0); i < length; i++ {
		var hash common.Uint256
		err := hash.Deserialize(r)
		if err != nil {
			return errors.New("[WithdrawFromSideChain], SideChainTransactionHashes deserialize failed.")
		}
		t.SideChainTransactionHashes[i] = hash
	}

	t.BlockHeight = height
	t.GenesisBlockAddress = address

	return nil
}
