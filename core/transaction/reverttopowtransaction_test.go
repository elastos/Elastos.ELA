// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"time"
)

func (s *txValidatorTestSuite) TestRevertToPowTransaction() {

	{
		revertToPow := &payload.RevertToPOW{
			Type:          payload.NoBlock,
			WorkingHeight: 100,
		}

		txn := functions.CreateTransaction(
			0,
			common2.RevertToPOW,
			0,
			revertToPow,
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			nil,
		)

		txn = CreateTransactionByType(txn, s.Chain)
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:invalid start POW block height")

		s.Chain.BestChain.Height = 100
		txn = CreateTransactionByType(txn, s.Chain)
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:invalid block time")

		s.Chain.BestChain.Timestamp = uint32(time.Now().Unix())
		txn = CreateTransactionByType(txn, s.Chain)
		err, _ = txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:invalid block time")

		s.Chain.BestChain.Timestamp = uint32(time.Now().Unix())
		txn.SetParameters(&TransactionParameters{
			Transaction: txn,
			BlockHeight: s.Chain.BestChain.Height,
			TimeStamp:   s.Chain.BestChain.Timestamp + (uint32(s.Chain.GetParams().DPoSConfiguration.RevertToPOWNoBlockTime) + 1000),
			Config:      s.Chain.GetParams(),
			BlockChain:  s.Chain,
		})
		err, _ = txn.SpecialContextCheck()
		s.NoError(err)
	}

	{
		revertToPow := &payload.RevertToPOW{
			Type:          payload.NoProducers,
			WorkingHeight: 100,
		}

		txn := functions.CreateTransaction(
			0,
			common2.RevertToPOW,
			0,
			revertToPow,
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			nil,
		)

		txn = CreateTransactionByType(txn, s.Chain)
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:current producers is enough")

		s.Chain.GetState().NoProducers = true
		err, _ = txn.SpecialContextCheck()
		s.NoError(err)

	}

	{
		revertToPow := &payload.RevertToPOW{
			Type:          payload.NoClaimDPOSNode,
			WorkingHeight: 100,
		}

		txn := functions.CreateTransaction(
			0,
			common2.RevertToPOW,
			0,
			revertToPow,
			[]*common2.Attribute{},
			[]*common2.Input{},
			[]*common2.Output{},
			0,
			nil,
		)

		txn = CreateTransactionByType(txn, s.Chain)
		err, _ := txn.SpecialContextCheck()
		s.EqualError(err,
			"transaction validate error: payload content invalid:current CR member claimed DPoS node")

		s.Chain.GetState().NoClaimDPOSNode = true
		err, _ = txn.SpecialContextCheck()
		s.NoError(err)

	}
}
