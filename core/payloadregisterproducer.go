package core

import (
	"bytes"
	"errors"
	"io"

	. "github.com/elastos/Elastos.ELA.Utility/common"
)

const PayloadRegisterProducerVersion byte = 0x00

type PayloadRegisterProducer struct {
	PublicKey string
	NickName  string
	Url       string
	Location  uint64
}

func (a *PayloadRegisterProducer) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := a.Serialize(buf, version); err != nil {
		return []byte{0}
	}
	return buf.Bytes()
}

func (a *PayloadRegisterProducer) Serialize(w io.Writer, version byte) error {
	err := WriteElements(w, a.PublicKey, a.NickName, a.Url, a.Location)
	if err != nil {
		return errors.New("[PayloadRegisterProducer], serialize failed.")
	}
	return nil
}

func (a *PayloadRegisterProducer) Deserialize(r io.Reader, version byte) error {
	err := ReadElements(r, a.PublicKey, a.NickName, a.Url, a.Location)
	if err != nil {
		return errors.New("[PayloadRegisterProducer], Deserialize failed.")
	}
	return nil
}
