package msg

import (
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto"
)

const ResponseInactiveArbitersLength = 32 + 33 + 64

type ResponseInactiveArbiters struct {
	TxHash common.Uint256
	Signer []byte
	Sign   []byte
}

func (i *ResponseInactiveArbiters) CMD() string {
	return CmdResponseInactiveArbiters
}

func (i *ResponseInactiveArbiters) MaxLength() uint32 {
	return ResponseInactiveArbitersLength
}

func (i *ResponseInactiveArbiters) Serialize(w io.Writer) error {
	if err := i.SerializeUnsigned(w); err != nil {
		return err
	}

	if err := common.WriteVarBytes(w, i.Sign); err != nil {
		return err
	}
	return nil
}

func (i *ResponseInactiveArbiters) SerializeUnsigned(w io.Writer) error {
	if err := i.TxHash.Serialize(w); err != nil {
		return err
	}

	if err := common.WriteVarBytes(w, i.Signer); err != nil {
		return err
	}
	return nil
}

func (i *ResponseInactiveArbiters) Deserialize(r io.Reader) (err error) {
	if err = i.DeserializeUnsigned(r); err != nil {
		return err
	}

	if i.Sign, err = common.ReadVarBytes(r, crypto.SignatureLength, "sign data"); err != nil {
		return err
	}
	return err
}

func (i *ResponseInactiveArbiters) DeserializeUnsigned(r io.Reader) (err error) {
	if err = i.TxHash.Deserialize(r); err != nil {
		return err
	}

	if i.Signer, err = common.ReadVarBytes(r, crypto.NegativeBigLength, "public key"); err != nil {
		return err
	}
	return err
}
