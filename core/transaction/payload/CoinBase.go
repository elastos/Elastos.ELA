package payload

import (
	. "DNA_POW/common/serialization"
	"io"
)

const CoinBasePayloadVersion byte = 0x04

type CoinBase struct {
	data []byte
}

func (a *CoinBase) Data(version byte) []byte {
	return a.data
}

func (a *CoinBase) Serialize(w io.Writer, version byte) error {
	return WriteVarBytes(w, a.data)
}

func (a *CoinBase) Deserialize(r io.Reader, version byte) error {
	temp, err := ReadVarBytes(r)
	a.data = temp
	return err
}
