// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import (
	"io"
)

const ReturnDepositCoinVersion byte = 0x00

type ReturnDepositCoin struct {
	DefaultChecker

}

func (a *ReturnDepositCoin) Data(version byte) []byte {
	return nil
}

func (a *ReturnDepositCoin) Serialize(w io.Writer, version byte) error {
	return nil
}

func (a *ReturnDepositCoin) Deserialize(r io.Reader, version byte) error {
	return nil
}
//
//// todo add description
//func (a *ReturnDepositCoin) SpecialCheck(txn *types.Transaction,
//	p *CheckParameters) (elaerr.ELAError, bool) {
//	// todo special check
//	return nil, false
//}
//
//// todo add description
//func (a *ReturnDepositCoin) SecondCheck(txn *types.Transaction,
//	p *CheckParameters) (elaerr.ELAError, bool) {
//	// todo special check
//	return nil, false
//}