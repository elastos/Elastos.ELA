// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package functions

import (
	"io"

	"github.com/elastos/Elastos.ELA/common"
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
)

var GetTransactionByTxType func(txType common2.TxType) (interfaces.Transaction, error)

var GetTransactionByBytes func(r io.Reader) (interfaces.Transaction, error)

var CreateTransaction func(
	version common2.TransactionVersion,
	txType common2.TxType,
	payloadVersion byte,
	payload interfaces.Payload,
	attributes []*common2.Attribute,
	inputs []*common2.Input,
	outputs []*common2.Output,
	lockTime uint32,
	programs []*pg.Program,
) interfaces.Transaction

var GetTransactionParameters func(
	transaction interfaces.Transaction,
	blockHeight uint32,
	timeStamp uint32,
	config interface{},
	blockChain interface{},
	proposalsUsedAmount common.Fixed64) interfaces.Parameters
