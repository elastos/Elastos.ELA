// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"math"
	"time"

	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type RevertToPOWTransaction struct {
	BaseTransaction
}

func (t *RevertToPOWTransaction) RegisterFunctions() {
	t.DefaultChecker.CheckTransactionSize = t.checkTransactionSize
	t.DefaultChecker.CheckTransactionInput = t.CheckTransactionInput
	t.DefaultChecker.CheckTransactionOutput = t.CheckTransactionOutput
	t.DefaultChecker.CheckTransactionPayload = t.CheckTransactionPayload
	t.DefaultChecker.HeightVersionCheck = t.HeightVersionCheck
	t.DefaultChecker.IsAllowedInPOWConsensus = t.IsAllowedInPOWConsensus
	t.DefaultChecker.SpecialContextCheck = t.SpecialContextCheck
	t.DefaultChecker.CheckAttributeProgram = t.CheckAttributeProgram
}

func (t *RevertToPOWTransaction) CheckTransactionInput(params *TransactionParameters) error {
	if len(params.Transaction.Inputs()) != 0 {
		return errors.New("no cost transactions must has no input")
	}
	return nil
}

func (t *RevertToPOWTransaction)  CheckTransactionOutput(params *TransactionParameters) error {

	txn := params.Transaction
	if len(txn.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}
	if len(txn.Outputs()) != 0 {
		return errors.New("no cost transactions should have no output")
	}

	return nil
}

func (t *RevertToPOWTransaction) CheckAttributeProgram(params *TransactionParameters) error {
	if len(t.Programs()) != 0 || len(t.Attributes()) != 0 {
		return errors.New("zero cost tx should have no attributes and programs")
	}
	return nil
}

func (t *RevertToPOWTransaction) CheckTransactionPayload(params *TransactionParameters) error {
	switch t.Payload().(type) {
	case *payload.RevertToPOW:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *RevertToPOWTransaction) IsAllowedInPOWConsensus(params *TransactionParameters, references map[*common2.Input]common2.Output) bool {
	return true
}

func (t *RevertToPOWTransaction) HeightVersionCheck(params *TransactionParameters) error {
	if params.BlockHeight < params.Config.RevertToPOWStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before RevertToPOWStartHeight", t.TxType().Name()))
	}

	return nil
}

func (t *RevertToPOWTransaction) SpecialContextCheck(params *TransactionParameters, references map[*common2.Input]common2.Output) (result elaerr.ELAError, end bool) {
	p, ok := t.Payload().(*payload.RevertToPOW)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	if p.WorkingHeight != params.BlockHeight {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid start POW block height")), true
	}

	switch p.Type {
	case payload.NoBlock:
		lastBlockTime := int64(params.BlockChain.BestChain.Timestamp)
		noBlockTime := params.Config.RevertToPOWNoBlockTime

		if params.TimeStamp == 0 {
			// is not in block, check by local time.
			localTime := t.MedianAdjustedTime(params).Unix()
			if localTime-lastBlockTime < noBlockTime {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid block time")), true
			}
		} else {
			// is in block, check by the time of existed block.
			if int64(params.TimeStamp)-lastBlockTime < noBlockTime {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid block time")), true
			}
		}
	case payload.NoProducers:
		if !params.BlockChain.GetState().NoProducers {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("current producers is enough")), true
		}
	case payload.NoClaimDPOSNode:
		if !params.BlockChain.GetState().NoClaimDPOSNode {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("current CR member claimed DPoS node")), true
		}
	}
	return nil, true
}

func (t *RevertToPOWTransaction) MedianAdjustedTime(params *TransactionParameters) time.Time {
	newTimestamp := params.BlockChain.TimeSource.AdjustedTime()
	minTimestamp := params.BlockChain.MedianTimePast.Add(time.Second)

	if newTimestamp.Before(minTimestamp) {
		newTimestamp = minTimestamp
	}

	return newTimestamp
}
