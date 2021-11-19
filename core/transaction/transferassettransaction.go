// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"

	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
)

type TransferAssetTransaction struct {
	BaseTransaction
}

func (t *TransferAssetTransaction) CheckTxHeightVersion() error {
	txn := t.contextParameters.Transaction
	blockHeight := t.contextParameters.BlockHeight
	chainParams := t.contextParameters.Config

	if blockHeight >= chainParams.CRVotingStartHeight {
		return nil
	}
	if txn.Version() >= common2.TxVersion09 {
		for _, output := range txn.Outputs() {
			if output.Type != common2.OTVote {
				continue
			}
			p, _ := output.Payload.(*outputpayload.VoteOutput)
			if p.Version >= outputpayload.VoteProducerAndCRVersion {
				return errors.New("not support " +
					"VoteProducerAndCRVersion before CRVotingStartHeight")
			}
		}
	}
	return nil
}
