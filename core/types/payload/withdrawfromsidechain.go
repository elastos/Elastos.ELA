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
)

const WithdrawFromSideChainVersion byte = 0x00
const WithdrawFromSideChainVersionV1 byte = 0x01
const WithdrawFromSideChainVersionV2 byte = 0x02 // for schnorr

type WithdrawFromSideChain struct {
	DefaultChecker

	BlockHeight                uint32
	GenesisBlockAddress        string
	SideChainTransactionHashes []common.Uint256

	// schnorr
	Signers []uint8
}

func (t *WithdrawFromSideChain) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := t.Serialize(buf, version); err != nil {
		return []byte{0}
	}

	return buf.Bytes()
}

func (t *WithdrawFromSideChain) Serialize(w io.Writer, version byte) error {
	switch version {
	case WithdrawFromSideChainVersion:
		if err := common.WriteUint32(w, t.BlockHeight); err != nil {
			return errors.New("[WithdrawFromSideChain], BlockHeight serialize failed.")
		}
		if err := common.WriteVarString(w, t.GenesisBlockAddress); err != nil {
			return errors.New("[WithdrawFromSideChain], GenesisBlockAddress serialize failed.")
		}

		if err := common.WriteVarUint(w, uint64(len(t.SideChainTransactionHashes))); err != nil {
			return errors.New("[WithdrawFromSideChain], SideChainTransactionHashes count serialize failed")
		}

		for _, hash := range t.SideChainTransactionHashes {
			err := hash.Serialize(w)
			if err != nil {
				return errors.New("[WithdrawFromSideChain], SideChainTransactionHashes serialize failed")
			}
		}
	case WithdrawFromSideChainVersionV1:
	case WithdrawFromSideChainVersionV2:
		if err := common.WriteVarUint(w, uint64(len(t.Signers))); err != nil {
			return err
		}
		for _, pk := range t.Signers {
			if err := common.WriteUint8(w, pk); err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *WithdrawFromSideChain) Deserialize(r io.Reader, version byte) error {
	switch version {
	case WithdrawFromSideChainVersion:

		height, err := common.ReadUint32(r)
		if err != nil {
			return errors.New("[WithdrawFromSideChain], BlockHeight deserialize failed.")
		}
		address, err := common.ReadVarString(r)
		if err != nil {
			return errors.New("[WithdrawFromSideChain], GenesisBlockAddress deserialize failed.")
		}

		count, err := common.ReadVarUint(r, 0)
		if err != nil {
			return errors.New("[WithdrawFromSideChain], SideChainTransactionHashes count deserialize failed")
		}

		t.SideChainTransactionHashes = make([]common.Uint256, 0)
		for i := uint64(0); i < count; i++ {
			var hash common.Uint256
			err := hash.Deserialize(r)
			if err != nil {
				return errors.New("[WithdrawFromSideChain], SideChainTransactionHash deserialize failed.")
			}
			t.SideChainTransactionHashes = append(t.SideChainTransactionHashes, hash)
		}

		t.BlockHeight = height
		t.GenesisBlockAddress = address
	case WithdrawFromSideChainVersionV1:
	case WithdrawFromSideChainVersionV2:
		count, err := common.ReadVarUint(r, 0)
		if err != nil {
			return err
		}
		t.Signers = make([]uint8, 0)
		for i := uint64(0); i < count; i++ {
			pk, err := common.ReadUint8(r)
			if err != nil {
				return err
			}
			t.Signers = append(t.Signers, pk)
		}
	}

	return nil
}
//
//// todo add description
//func (a *WithdrawFromSideChain) SpecialCheck(txn *types.Transaction,
//	p *CheckParameters) (elaerr.ELAError, bool) {
//	// todo special check
//	return nil, false
//}
//
//// todo add description
//func (a *WithdrawFromSideChain) SecondCheck(txn *types.Transaction,
//	p *CheckParameters) (elaerr.ELAError, bool) {
//	// todo special check
//	return nil, false
//}
