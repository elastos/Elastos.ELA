// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import "io"

type TransferAsset struct{
	DefaultChecker
}

func (a *TransferAsset) Data(version byte) []byte {
	//TODO: implement TransferAsset.Data()
	return []byte{0}
}

func (a *TransferAsset) Serialize(w io.Writer, version byte) error {
	return nil
}

func (a *TransferAsset) Deserialize(r io.Reader, version byte) error {
	return nil
}

//
//// todo add description
//func (a *TransferAsset) SpecialCheck(txn *types.Transaction,
//	p *CheckParameters) (elaerr.ELAError, bool) {
//	// todo special check
//	return nil, false
//}
//
//// todo add description
//func (a *TransferAsset) SecondCheck(txn *types.Transaction,
//	p *CheckParameters) (elaerr.ELAError, bool) {
//	// todo special check
//	return nil, false
//}
