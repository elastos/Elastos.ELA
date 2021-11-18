// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
)

type TransactionParameters struct {
	Transaction interfaces.Transaction

	BlockHeight         uint32
	TimeStamp           uint32
	Config              *config.Params
	BlockChain          *blockchain.BlockChain
	ProposalsUsedAmount common.Fixed64
}
