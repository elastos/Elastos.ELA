package core

import (
	"bytes"
	"errors"
	"io"

	"github.com/elastos/Elastos.ELA.Utility/common"
)

type CrossChainAsset struct {
	CrossChainAddress string
	OutputIndex       uint64
	CrossChainAmount  common.Fixed64
}

type PayloadTransferCrossChainAsset struct {
	Assets []CrossChainAsset
}

func (a *CrossChainAsset) Serialize(w io.Writer, version byte) error {
	return common.WriteElements(w, a.CrossChainAddress, a.OutputIndex, a.CrossChainAmount)
}

func (a *CrossChainAsset) Deserialize(r io.Reader, version byte) error {
	return common.ReadElements(r, a.CrossChainAddress, a.OutputIndex, a.CrossChainAmount)
}

func (a *PayloadTransferCrossChainAsset) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := a.Serialize(buf, version); err != nil {
		return []byte{0}
	}

	return buf.Bytes()
}

func (a *PayloadTransferCrossChainAsset) Serialize(w io.Writer, version byte) error {

	if err := common.WriteVarUint(w, uint64(len(a.Assets))); err != nil {
		return errors.New("[PayloadTransferCrossChainAsset], Assets length serialize failed.")
	}

	for _, asset := range a.Assets {
		err := asset.Serialize(w, version)
		if err != nil {
			return errors.New("[PayloadTransferCrossChainAsset], Assets serialize failed.")
		}
	}

	return nil
}

func (a *PayloadTransferCrossChainAsset) Deserialize(r io.Reader, version byte) error {
	length, err := common.ReadVarUint(r, 0)
	if err != nil {
		return errors.New("[PayloadTransferCrossChainAsset], Length deserialize failed.")
	}

	a.Assets = make([]CrossChainAsset, 0)
	for i := uint64(0); i < length; i++ {
		var csAsset CrossChainAsset
		err := csAsset.Deserialize(r, version)
		if err != nil {
			return errors.New("[PayloadTransferCrossChainAsset], Assets deserialize failed.")
		}
		a.Assets = append(a.Assets, csAsset)
	}

	return nil
}
