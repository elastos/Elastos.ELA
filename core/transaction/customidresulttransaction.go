// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type ProposalResultTransaction struct {
	BaseTransaction
}

func (t *ProposalResultTransaction) IsAllowedInPOWConsensus() bool {
	return true
}

func (t *ProposalResultTransaction) SpecialCheck() (elaerr.ELAError, bool) {
	if !blockchain.DefaultLedger.Committee.IsProposalResultNeeded() {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("should not have proposal result transaction")), true
	}
	p, ok := t.Payload().(*payload.RecordProposalResult)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid proposal result payload")), true
	}
	results := blockchain.DefaultLedger.Committee.GetCustomIDResults()
	targetResults := make(map[common.Uint256]payload.ProposalResult, 0)
	for _, r := range results {
		targetResults[r.ProposalHash] = r
	}
	if len(p.ProposalResults) != len(targetResults) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid proposal results count")), true
	}
	for _, r := range p.ProposalResults {
		ret, ok := targetResults[r.ProposalHash]
		if !ok || ret.Result != r.Result || ret.ProposalType != r.ProposalType {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid proposal results")), true
		}
	}
	return nil, true
}
