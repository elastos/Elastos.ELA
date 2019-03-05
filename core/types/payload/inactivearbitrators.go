package payload

import (
	"bytes"
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto"
)

const InactiveArbitersVersion byte = 0x00

type InactiveArbiters struct {
	Sponsor     []byte
	Arbiters    [][]byte
	BlockHeight uint32

	hash *common.Uint256
}

func (i *InactiveArbiters) Type() IllegalDataType {
	return InactiveArbiter
}

func (i *InactiveArbiters) GetBlockHeight() uint32 {
	return i.BlockHeight
}

func (i *InactiveArbiters) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := i.Serialize(buf, version); err != nil {
		return []byte{0}
	}
	return buf.Bytes()
}

func (i *InactiveArbiters) Serialize(w io.Writer,
	version byte) error {
	if err := common.WriteVarBytes(w, i.Sponsor); err != nil {
		return err
	}

	if err := common.WriteVarUint(w, uint64(len(i.Arbiters))); err != nil {
		return err
	}

	if err := common.WriteUint32(w, i.BlockHeight); err != nil {
		return err
	}

	for _, v := range i.Arbiters {
		if err := common.WriteVarBytes(w, v); err != nil {
			return err
		}
	}

	return nil
}

func (i *InactiveArbiters) Deserialize(r io.Reader,
	version byte) (err error) {
	if i.Sponsor, err = common.ReadVarBytes(r, crypto.NegativeBigLength,
		"public key"); err != nil {
		return err
	}

	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return err
	}

	if i.BlockHeight, err = common.ReadUint32(r); err != nil {
		return err
	}

	i.Arbiters = make([][]byte, count)
	for u := uint64(0); u < count; u++ {
		if i.Arbiters[u], err = common.ReadVarBytes(r,
			crypto.NegativeBigLength, "public key"); err != nil {
			return err
		}
	}

	return err
}

func (i *InactiveArbiters) Hash() common.Uint256 {
	if i.hash == nil {
		buf := new(bytes.Buffer)
		i.Serialize(buf, InactiveArbitersVersion)
		hash := common.Uint256(common.Sha256D(buf.Bytes()))
		i.hash = &hash
	}
	return *i.hash
}
