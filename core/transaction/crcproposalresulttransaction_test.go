// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

func (s *txValidatorTestSuite) TestCheckCRCProposalResultTransaction() {
	hash := *randomUint256()
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.ProposalResult,
		0,
		&payload.RecordProposalResult{
			ProposalResults: []payload.ProposalResult{
				{
					ProposalHash: hash,
					ProposalType: payload.ReserveCustomID,
					Result:       true,
				},
			},
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	txn = CreateTransactionByType(txn, s.Chain)
	err, _ := txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:should not have proposal result transaction")

	blockchain.DefaultLedger.Committee.NeedRecordProposalResult = true
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:invalid proposal results count")

	blockchain.DefaultLedger.Committee.PartProposalResults = []payload.ProposalResult{
		{
			ProposalHash: hash,
			ProposalType: payload.ReserveCustomID,
			Result:       true,
		},
	}

	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

}
