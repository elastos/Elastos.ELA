package core

import (
	"errors"
	"io"

	. "github.com/elastos/Elastos.ELA.Utility/common"
)

type AttributeUsage byte

const (
	Nonce          AttributeUsage = 0x00
	Script         AttributeUsage = 0x20
	Memo           AttributeUsage = 0x81
	Description    AttributeUsage = 0x90
	DescriptionUrl AttributeUsage = 0x91
)

func (usage AttributeUsage) Name() string {
	switch usage {
	case Nonce:
		return "Nonce"
	case Script:
		return "Script"
	case Memo:
		return "Memo"
	case Description:
		return "Description"
	case DescriptionUrl:
		return "DescriptionUrl"
	default:
		return "Unknown"
	}
}

func IsValidAttributeType(usage AttributeUsage) bool {
	return usage == Nonce || usage == Script ||
		usage == Memo || usage == Description || usage == DescriptionUrl
}

type Attribute struct {
	Usage AttributeUsage
	Data  []byte
}

func (attr Attribute) String() string {
	return "Attribute: {\n\t\t" +
		"Usage: " + attr.Usage.Name() + "\n\t\t" +
		"Data: " + BytesToHexString(attr.Data) + "\n\t\t" +
		"}"
}

func (attr *Attribute) Serialize(w io.Writer) error {
	if err := WriteUint8(w, byte(attr.Usage)); err != nil {
		return errors.New("Transaction attribute Usage serialization error.")
	}
	if !IsValidAttributeType(attr.Usage) {
		return errors.New("[Attribute error] Unsupported attribute Description.")
	}
	if err := WriteVarBytes(w, attr.Data); err != nil {
		return errors.New("Transaction attribute Data serialization error.")
	}
	return nil
}

func (attr *Attribute) Deserialize(r io.Reader) error {
	usage, err := ReadUint8(r)
	if err != nil {
		return errors.New("Transaction attribute Usage deserialization error.")
	}
	attr.Usage = AttributeUsage(usage)
	if !IsValidAttributeType(attr.Usage) {
		return errors.New("[Attribute error] Unsupported attribute Description.")
	}
	attr.Data, err = ReadVarBytes(r)
	if err != nil {
		return errors.New("Transaction attribute Data deserialization error.")
	}
	return nil
}
