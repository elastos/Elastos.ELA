package payload

import (
	"bytes"
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto"
)

const NextTurnDPOSInfoVersion byte = 0x00
const NextTurnDPOSInfoVersion2 byte = 0x01

type NextTurnDPOSInfo struct {
	WorkingHeight        uint32
	CRPublicKeys         [][]byte
	DPOSPublicKeys       [][]byte
	CompleteCRPublicKeys [][]byte

	hash *common.Uint256
}

func (n *NextTurnDPOSInfo) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := n.Serialize(buf, version); err != nil {
		return []byte{0}
	}
	return buf.Bytes()
}

func (n *NextTurnDPOSInfo) Serialize(w io.Writer, version byte) error {
	err := n.SerializeUnsigned(w, version)
	if err != nil {
		return err
	}

	return nil
}

func (n *NextTurnDPOSInfo) SerializeUnsigned(w io.Writer, version byte) error {

	if err := common.WriteUint32(w, n.WorkingHeight); err != nil {
		return err
	}
	if err := common.WriteVarUint(w, uint64(len(n.CRPublicKeys))); err != nil {
		return err
	}

	for _, v := range n.CRPublicKeys {
		if err := common.WriteVarBytes(w, v); err != nil {
			return err
		}
	}

	if err := common.WriteVarUint(w, uint64(len(n.DPOSPublicKeys))); err != nil {
		return err
	}

	for _, v := range n.DPOSPublicKeys {
		if err := common.WriteVarBytes(w, v); err != nil {
			return err
		}
	}

	if version >= NextTurnDPOSInfoVersion2 {
		if err := common.WriteVarUint(w, uint64(len(n.CompleteCRPublicKeys))); err != nil {
			return err
		}

		for _, v := range n.CompleteCRPublicKeys {
			if err := common.WriteVarBytes(w, v); err != nil {
				return err
			}
		}
	}

	return nil
}

func (n *NextTurnDPOSInfo) Deserialize(r io.Reader, version byte) error {
	err := n.DeserializeUnsigned(r, version)
	if err != nil {
		return err
	}
	return nil
}

func (n *NextTurnDPOSInfo) DeserializeUnsigned(r io.Reader, version byte) error {
	var err error
	var len uint64

	var workingHeight uint32
	if workingHeight, err = common.ReadUint32(r); err != nil {
		return err
	}
	n.WorkingHeight = workingHeight

	if len, err = common.ReadVarUint(r, 0); err != nil {
		return err
	}

	n.CRPublicKeys = make([][]byte, 0, len)
	for i := uint64(0); i < len; i++ {
		var CRPublickey []byte
		if CRPublickey, err = common.ReadVarBytes(r, crypto.COMPRESSEDLEN,
			"cr public key"); err != nil {
			return err
		}
		n.CRPublicKeys = append(n.CRPublicKeys, CRPublickey)
	}

	if len, err = common.ReadVarUint(r, 0); err != nil {
		return err
	}

	n.DPOSPublicKeys = make([][]byte, 0, len)
	for i := uint64(0); i < len; i++ {
		var DPOSPublicKey []byte
		if DPOSPublicKey, err = common.ReadVarBytes(r, crypto.COMPRESSEDLEN,
			"dpos public key"); err != nil {
			return err
		}
		n.DPOSPublicKeys = append(n.DPOSPublicKeys, DPOSPublicKey)
	}

	if version >= NextTurnDPOSInfoVersion2 {
		if len, err = common.ReadVarUint(r, 0); err != nil {
			return err
		}
		n.CompleteCRPublicKeys = make([][]byte, 0, len)
		for i := uint64(0); i < len; i++ {
			var publicKey []byte
			if publicKey, err = common.ReadVarBytes(r, crypto.COMPRESSEDLEN,
				"complete crcs"); err != nil {
				return err
			}
			n.CompleteCRPublicKeys = append(n.CompleteCRPublicKeys, publicKey)
		}
	}

	return nil
}

func (n *NextTurnDPOSInfo) Hash() common.Uint256 {
	if n.hash == nil {
		buf := new(bytes.Buffer)
		n.SerializeUnsigned(buf, NextTurnDPOSInfoVersion)
		hash := common.Hash(buf.Bytes())
		n.hash = &hash
	}
	return *n.hash
}
