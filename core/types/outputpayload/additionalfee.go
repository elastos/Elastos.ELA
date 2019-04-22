package outputpayload

import (
	"errors"
	"io"

	"github.com/elastos/Elastos.ELA/common"
)

// AdditionalFee is a additional handling fee, it is required when side chain
// recharge fee is insufficient.
type AdditionalFee struct {
	// Version indicates the version of AdditionalFee payload.
	Version byte

	// TxHash is a transaction hash that requires additional fee.
	TxHash common.Uint256
}

func (c *AdditionalFee) Data() []byte {
	return nil
}

func (c *AdditionalFee) Serialize(w io.Writer) error {
	if err := common.WriteUint8(w, c.Version); err != nil {
		return err
	}

	if err := c.TxHash.Serialize(w); err != nil {
		return err
	}
	return nil
}

func (c *AdditionalFee) Deserialize(r io.Reader) error {
	var err error
	c.Version, err = common.ReadUint8(r)
	if err != nil {
		return err
	}

	err = c.TxHash.Deserialize(r)
	if err != nil {
		return err
	}
	return nil
}

func (c *AdditionalFee) GetVersion() byte {
	return c.Version
}

func (c *AdditionalFee) Validate() error {

	if c.Version != byte(0) {
		return errors.New("invalid additional fee version")
	}

	return nil
}
