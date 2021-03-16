// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
// 

package payload

import (
	"bytes"
	"errors"
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto"
)

const IllegalDepositTxsVersion byte = 0x00

type IllegalDepositTxs struct {
	Height              uint32
	GenesisBlockAddress string
	DepositTxs          []common.Uint256
	Signs               [][]byte

	hash *common.Uint256
}

func (s *IllegalDepositTxs) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := s.Serialize(buf, version); err != nil {
		return []byte{0}
	}
	return buf.Bytes()
}

func (s *IllegalDepositTxs) GetBlockHeight() uint32 {
	return s.Height
}

func (s *IllegalDepositTxs) SerializeUnsigned(w io.Writer, version byte) error {
	if err := common.WriteUint32(w, s.Height); err != nil {
		return err
	}

	if err := common.WriteVarString(w, s.GenesisBlockAddress); err != nil {
		return err
	}

	err := common.WriteVarUint(w, uint64(len(s.DepositTxs)))
	if err != nil {
		return errors.New(
			"failed to serialize length of DepositTxs")
	}

	for _, hash := range s.DepositTxs {
		if err := hash.Serialize(w); err != nil {
			return errors.New(
				"failed to serialize DepositTxs")
		}
	}

	return nil
}

func (s *IllegalDepositTxs) Serialize(w io.Writer, version byte) error {
	if err := s.SerializeUnsigned(w, version); err != nil {
		return err
	}

	if err := common.WriteVarUint(w, uint64(len(s.Signs))); err != nil {
		return err
	}
	for _, v := range s.Signs {
		if err := common.WriteVarBytes(w, v); err != nil {
			return err
		}
	}

	return nil
}

func (s *IllegalDepositTxs) DeserializeUnsigned(r io.Reader,
	version byte) error {
	var err error

	if s.Height, err = common.ReadUint32(r); err != nil {
		return err
	}

	if s.GenesisBlockAddress, err = common.ReadVarString(r); err != nil {
		return err
	}

	var txLen uint64
	if txLen, err = common.ReadVarUint(r, 0); err != nil {
		return err
	}
	s.DepositTxs = make([]common.Uint256, txLen)
	for i := 0; i < int(txLen); i++ {
		err := s.DepositTxs[i].Deserialize(r)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *IllegalDepositTxs) Deserialize(r io.Reader, version byte) error {
	var err error
	if err = s.DeserializeUnsigned(r, version); err != nil {
		return err
	}

	var signLen uint64
	if signLen, err = common.ReadVarUint(r, 0); err != nil {
		return err
	}
	s.Signs = make([][]byte, signLen)
	for i := 0; i < int(signLen); i++ {
		s.Signs[i], err = common.ReadVarBytes(r, crypto.SignatureLength,
			"Signature")
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *IllegalDepositTxs) Hash() common.Uint256 {
	if s.hash == nil {
		buf := new(bytes.Buffer)
		s.SerializeUnsigned(buf, IllegalDepositTxsVersion)
		hash := common.Hash(buf.Bytes())
		s.hash = &hash
	}
	return *s.hash
}
