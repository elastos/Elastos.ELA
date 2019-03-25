package msg

import (
	"bytes"
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/p2p"
)

const (
	// maxCipherLength indicates the max length of the address cipher.
	maxCipherLength = 256
)

// Ensure Addr implement p2p.Message interface.
var _ p2p.Message = (*DAddr)(nil)

// DAddr represents a DPOS peer address.
type DAddr struct {
	// The peer ID indicates who's address it is.
	PID [33]byte

	// Which peer ID is used to encode the address cipher.
	Encode [33]byte

	// The encrypted network address using the encode peer ID.
	Cipher []byte

	// Signature of the encode peer ID and cipher to proof the sender itself.
	Signature []byte
}

func (a *DAddr) CMD() string {
	return p2p.CmdDAddr
}

func (a *DAddr) MaxLength() uint32 {
	return 387 // 33+33+256+65
}

func (a *DAddr) Serialize(w io.Writer) error {
	if _, err := w.Write(a.PID[:]); err != nil {
		return err
	}

	if _, err := w.Write(a.Encode[:]); err != nil {
		return err
	}

	if err := common.WriteVarBytes(w, a.Cipher); err != nil {
		return err
	}

	return common.WriteVarBytes(w, a.Signature)
}

func (a *DAddr) Deserialize(r io.Reader) error {
	if _, err := io.ReadFull(r, a.PID[:]); err != nil {
		return err
	}

	if _, err := io.ReadFull(r, a.Encode[:]); err != nil {
		return err
	}

	var err error
	a.Cipher, err = common.ReadVarBytes(r, maxCipherLength, "DAddr.Cipher")
	if err != nil {
		return err
	}

	a.Signature, err = common.ReadVarBytes(r, crypto.SignatureLength,
		"DAddr.Signature")
	return err
}

func (a *DAddr) Data() []byte {
	b := new(bytes.Buffer)
	b.Write(a.Encode[:])
	common.WriteVarBytes(b, a.Cipher)
	return b.Bytes()
}

func (a *DAddr) Hash() common.Uint256 {
	return common.Sha256D(a.Data())
}
