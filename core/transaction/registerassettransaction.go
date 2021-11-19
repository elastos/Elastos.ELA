// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

type RegisterAssetTransaction struct {
	BaseTransaction
}

func (t *RegisterAssetTransaction) IsAllowedInPOWConsensus() bool {
	return false
}
