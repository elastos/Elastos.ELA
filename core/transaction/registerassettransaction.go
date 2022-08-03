// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

type RegisterAssetTransaction struct {
	BaseTransaction
}

func (t *RegisterAssetTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *RegisterAssetTransaction) CheckTransactionPayload() error {
	switch pld := t.Payload().(type) {
	case *payload.RegisterAsset:
		if pld.Asset.Precision < payload.MinPrecision || pld.Asset.Precision > payload.MaxPrecision {
			return errors.New("invalid asset precision")
		}
		if !checkAmountPrecise(pld.Amount, pld.Asset.Precision) {
			return errors.New("invalid asset value, out of precise")
		}
		return nil
	}

	return errors.New("invalid payload type")
}
