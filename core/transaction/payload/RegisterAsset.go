package payload

import (
	"DNA_POW/common"
	"DNA_POW/core/asset"
	. "DNA_POW/errors"
	"io"
)

const RegisterPayloadVersion byte = 0x00

type RegisterAsset struct {
	Asset  *asset.Asset
	Amount common.Fixed64
	Controller common.Uint160
}

func (a *RegisterAsset) Data(version byte) []byte {
	//TODO: implement RegisterAsset.Data()
	return []byte{0}

}

func (a *RegisterAsset) Serialize(w io.Writer, version byte) error {
	a.Asset.Serialize(w)
	a.Amount.Serialize(w)
	a.Controller.Serialize(w)
	return nil
}

func (a *RegisterAsset) Deserialize(r io.Reader, version byte) error {
	//asset
	a.Asset = new(asset.Asset)
	err := a.Asset.Deserialize(r)
	if err != nil {
		return NewDetailErr(err, ErrNoCode, "[RegisterAsset], Asset Deserialize failed.")
	}

	//Amount
	a.Amount = *new(common.Fixed64)
	err = a.Amount.Deserialize(r)
	if err != nil {
		return NewDetailErr(err, ErrNoCode, "[RegisterAsset], Ammount Deserialize failed.")
	}

	//Controller *common.Uint160
	a.Controller = *new(common.Uint160)
	err = a.Controller.Deserialize(r)
	if err != nil {
		return NewDetailErr(err, ErrNoCode, "[RegisterAsset], Ammount Deserialize failed.")
	}
	return nil
}
