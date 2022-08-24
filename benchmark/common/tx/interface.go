// Copyright (c) 2017-2022 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package tx

import (
	"github.com/elastos/Elastos.ELA/account"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
)

type AssignerType byte

const (
	NoChanges AssignerType = 0x00
	FixAmount AssignerType = 0x01
)

type Generator interface {
	Generate() interfaces.Transaction
}

type Assigner interface {
	SignAndChange(tx interfaces.Transaction) error
}

func NewGenerator(txType common2.TxType, ac ...*account.Account) Generator {
	switch txType {
	case common2.TransferAsset:
		return &transferAssetGenerator{account: ac}
	default:
		return nil
	}
}

func NewAssigner(assignerType AssignerType, ac *account.Account,
	utxo *common2.UTXO) Assigner {
	switch assignerType {
	case NoChanges:
		return &noChangesEvenAssigner{account: ac, utxo: utxo}
	case FixAmount:
		return &fixAmountAssigner{account: ac, utxo: utxo}
	default:
		return nil
	}
}
