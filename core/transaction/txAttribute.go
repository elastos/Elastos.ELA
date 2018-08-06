package transaction

import (
	"Elastos.ELA/common/serialization"
	"errors"
	"io"
)

type TransactionAttributeUsage byte

const (
	Nonce          TransactionAttributeUsage = 0x00
	Script         TransactionAttributeUsage = 0x20
	Memo           TransactionAttributeUsage = 0x81
	Description    TransactionAttributeUsage = 0x90
	DescriptionUrl TransactionAttributeUsage = 0x91
	Confirmations  TransactionAttributeUsage = 0x92
)

func IsValidAttributeType(usage TransactionAttributeUsage) bool {
	return usage == Nonce || usage == Script ||
		usage == Memo || usage == Description || usage == DescriptionUrl ||
		usage == Confirmations
}

type TxAttribute struct {
	Usage TransactionAttributeUsage
	Data  []byte
	Size  uint32
}

func NewTxAttribute(u TransactionAttributeUsage, d []byte) TxAttribute {
	tx := TxAttribute{u, d, 0}
	tx.Size = tx.GetSize()
	return tx
}

func (u *TxAttribute) GetSize() uint32 {
	if u.Usage == Memo {
		return uint32(len([]byte{(byte(0xff))}) + len([]byte{(byte(0xff))}) + len(u.Data))
	}
	return 0
}

func (tx *TxAttribute) Serialize(w io.Writer) error {
	if err := serialization.WriteUint8(w, byte(tx.Usage)); err != nil {
		return errors.New("Transaction attribute Usage serialization error.")
	}
	if !IsValidAttributeType(tx.Usage) {
		return errors.New("[TxAttribute error] Unsupported attribute Description.")
	}
	if err := serialization.WriteVarBytes(w, tx.Data); err != nil {
		return errors.New("Transaction attribute Data serialization error.")
	}
	return nil
}

func (tx *TxAttribute) Deserialize(r io.Reader) error {
	val, err := serialization.ReadBytes(r, 1)
	if err != nil {
		return errors.New("Transaction attribute Usage deserialization error.")
	}
	tx.Usage = TransactionAttributeUsage(val[0])
	if !IsValidAttributeType(tx.Usage) {
		return errors.New("[TxAttribute error] Unsupported attribute Description.")
	}
	tx.Data, err = serialization.ReadVarBytes(r)
	if err != nil {
		return errors.New("Transaction attribute Data deserialization error.")
	}
	return nil
}
