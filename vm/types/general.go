// Copyright (c) 2017-2022 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package types

import (
	"github.com/elastos/Elastos.ELA/vm/interfaces"
	"math/big"
)

type GeneralInterface struct {
	object interfaces.IGeneralInterface
}

func NewGeneralInterface(value interfaces.IGeneralInterface) *GeneralInterface {
	var ii GeneralInterface
	ii.object = value
	return &ii
}

func (ii *GeneralInterface) Equals() bool {
	return false
}

func (ii *GeneralInterface) GetBigInteger() big.Int {
	return big.Int{}
}

func (ii *GeneralInterface) GetBoolean() bool {
	if ii.object == nil {
		return false
	}
	return true
}

func (ii *GeneralInterface) GetByteArray() []byte {
	return ii.object.Bytes()
}

func (ii *GeneralInterface) GetInterface() {
}

func (ii *GeneralInterface) GetArray() []StackItem {
	return []StackItem{}
}
