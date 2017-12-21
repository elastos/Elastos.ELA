package signature

import (
	"Elastos.ELA/common"
	"Elastos.ELA/common/log"
	"Elastos.ELA/core/contract/program"
	"Elastos.ELA/crypto"
	. "Elastos.ELA/errors"
	"Elastos.ELA/vm/interfaces"
	"bytes"
	"io"
)

//SignableData describe the data need be signed.
type SignableData interface {
	interfaces.ISignableObject

	//Get the the SignableData's program hashes
	GetProgramHashes() ([]common.Uint160, error)

	SetPrograms([]*program.Program)

	GetPrograms() []*program.Program

	//TODO: add SerializeUnsigned
	SerializeUnsigned(io.Writer) error
}

func SignBySigner(data SignableData, signer Signer) ([]byte, error) {
	log.Debug()
	//fmt.Println("data",data)
	rtx, err := crypto.Sign(signer.PrivKey(), GetHashData(data))

	if err != nil {
		return nil, NewDetailErr(err, ErrNoCode, "[Signature],SignBySigner failed.")
	}
	return rtx, nil
}

func GetHashData(data SignableData) []byte {
	b_buf := new(bytes.Buffer)
	data.SerializeUnsigned(b_buf)
	return b_buf.Bytes()
}
