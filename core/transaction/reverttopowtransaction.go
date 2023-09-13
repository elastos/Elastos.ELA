// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type RevertToPOWTransaction struct {
	BaseTransaction
}

func (t *RevertToPOWTransaction) CheckTransactionInput() error {
	if len(t.Inputs()) != 0 {
		return errors.New("no cost transactions must has no input")
	}
	return nil
}

func (t *RevertToPOWTransaction) CheckTransactionOutput() error {

	if len(t.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}
	if len(t.Outputs()) != 0 {
		return errors.New("no cost transactions should have no output")
	}

	return nil
}

func (t *RevertToPOWTransaction) CheckAttributeProgram() error {
	if len(t.Programs()) != 0 || len(t.Attributes()) != 0 {
		return errors.New("zero cost tx should have no attributes and programs")
	}
	return nil
}

func (t *RevertToPOWTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.RevertToPOW:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *RevertToPOWTransaction) IsAllowedInPOWConsensus() bool {
	return true
}

func (t *RevertToPOWTransaction) HeightVersionCheck() error {
	if t.parameters.BlockHeight < t.parameters.Config.DPoSConfiguration.RevertToPOWStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before RevertToPOWStartHeight", t.TxType().Name()))
	}

	return nil
}

func (t *RevertToPOWTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {
	p, ok := t.Payload().(*payload.RevertToPOW)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	if p.WorkingHeight != t.parameters.BlockHeight {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid start POW block height")), true
	}

	switch p.Type {
	case payload.NoBlock:
		lastBlockTime := int64(t.parameters.BlockChain.BestChain.Timestamp)
		noBlockTime := t.parameters.Config.DPoSConfiguration.RevertToPOWNoBlockTime

		if t.parameters.TimeStamp == 0 {
			// is not in block, check by local time.
			localTime := t.MedianAdjustedTime().Unix()
			if localTime-lastBlockTime < noBlockTime {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid block time")), true
			}
		} else {
			// is in block, check by the time of existed block.
			if int64(t.parameters.TimeStamp)-lastBlockTime < noBlockTime {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid block time")), true
			}
		}
	case payload.NoProducers:
		if !t.parameters.BlockChain.GetState().GetNoProducers() {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("current producers is enough")), true
		}
	case payload.NoClaimDPOSNode:
		if !t.parameters.BlockChain.GetState().GetNoClaimDPOSNode() {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("current CR member claimed DPoS node")), true
		}
	}
	return nil, true
}

func (t *RevertToPOWTransaction) MedianAdjustedTime() time.Time {
	newTimestamp := t.parameters.BlockChain.TimeSource.AdjustedTime()
	minTimestamp := t.parameters.BlockChain.MedianTimePast.Add(time.Second)

	if newTimestamp.Before(minTimestamp) {
		newTimestamp = minTimestamp
	}

	return newTimestamp
}
