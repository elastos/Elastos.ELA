// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import (
	"github.com/elastos/Elastos.ELA/common"
	"io"
)

const ReturnSideChainDepositCoinVersion byte = 0x00
const ReturnSideChainDepositCoinVersionV1 byte = 0x01

type ReturnSideChainDepositCoin struct {
	DefaultChecker

	// schnorr
	Signers []uint8
}

func (s *ReturnSideChainDepositCoin) Data(version byte) []byte {
	return nil
}

func (s *ReturnSideChainDepositCoin) Serialize(w io.Writer, version byte) error {
	switch version {
	case ReturnSideChainDepositCoinVersion:
	case ReturnSideChainDepositCoinVersionV1:
		if err := common.WriteVarUint(w, uint64(len(s.Signers))); err != nil {
			return err
		}
		for _, pk := range s.Signers {
			if err := common.WriteUint8(w, pk); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *ReturnSideChainDepositCoin) Deserialize(r io.Reader, version byte) error {
	switch version {
	case ReturnSideChainDepositCoinVersion:
	case ReturnSideChainDepositCoinVersionV1:
		count, err := common.ReadVarUint(r, 0)
		if err != nil {
			return err
		}
		s.Signers = make([]uint8, 0)
		for i := uint64(0); i < count; i++ {
			pk, err := common.ReadUint8(r)
			if err != nil {
				return err
			}
			s.Signers = append(s.Signers, pk)
		}
	}
	return nil
}

//
//// todo add description
//func (a *ReturnSideChainDepositCoin) SpecialCheck(txn *types.Transaction,
//	p *CheckParameters) (elaerr.ELAError, bool) {
//	// todo special check
//	return nil, false
//}
//
//// todo add description
//func (a *ReturnSideChainDepositCoin) SecondCheck(txn *types.Transaction,
//	p *CheckParameters) (elaerr.ELAError, bool) {
//	// todo special check
//	return nil, false
//}
