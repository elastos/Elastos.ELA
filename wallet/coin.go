// Copyright (c) 2017-2022 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package wallet

import (
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"io"

	"github.com/elastos/Elastos.ELA/common"
)

type Coin struct {
	TxVersion common2.TransactionVersion
	Output    *common2.Output
	Height    uint32
}

func (coin *Coin) Serialize(w io.Writer) error {
	if err := common.WriteUint8(w, uint8(coin.TxVersion)); err != nil {
		return err
	}
	if err := coin.Output.Serialize(w, coin.TxVersion); err != nil {
		return err
	}

	return common.WriteUint32(w, coin.Height)
}

func (coin *Coin) Deserialize(r io.Reader) error {
	txVersion, err := common.ReadUint8(r)
	if err != nil {
		return err
	}
	coin.TxVersion = common2.TransactionVersion(txVersion)
	coin.Output = new(common2.Output)
	if err := coin.Output.Deserialize(r, common2.TransactionVersion(txVersion)); err != nil {
		return err
	}

	height, err := common.ReadUint32(r)
	if err != nil {
		return err
	}
	coin.Height = height

	return nil
}
