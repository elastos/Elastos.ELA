package payload

import (
	"bytes"
	"errors"
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto"
)

const UnregisterCRVersion byte = 0x00

type UnregisterCR struct {
	Code      []byte
	Signature []byte
}

func (a *UnregisterCR) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := a.Serialize(buf, version); err != nil {
		return []byte{0}
	}
	return buf.Bytes()
}

func (a *UnregisterCR) Serialize(w io.Writer, version byte) error {
	err := a.SerializeUnsigned(w, version)
	if err != nil {
		return err
	}

	err = common.WriteVarBytes(w, a.Signature)
	if err != nil {
		return errors.New("[UnregisterCR], Signature serialize failed")
	}

	return nil
}

func (a *UnregisterCR) SerializeUnsigned(w io.Writer, version byte) error {
	err := common.WriteVarBytes(w, a.Code)
	if err != nil {
		return errors.New("[UnregisterCR], code serialize failed")
	}

	return nil
}

func (a *UnregisterCR) Deserialize(r io.Reader, version byte) error {
	err := a.DeserializeUnsigned(r, version)
	if err != nil {
		return err
	}
	a.Signature, err = common.ReadVarBytes(r, crypto.MaxSignatureScriptLength, "signature")
	if err != nil {
		return errors.New("[UnregisterCR], signature deserialize failed")
	}

	return nil
}

func (a *UnregisterCR) DeserializeUnsigned(r io.Reader, version byte) error {
	var err error
	a.Code, err = common.ReadVarBytes(r, crypto.MaxMultiSignCodeLength, "code")
	if err != nil {
		return errors.New("[UnregisterCR], code deserialize failed")
	}

	return nil
}
