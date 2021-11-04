// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transactions

import (
	"errors"

	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type CoinBaseTransaction struct {
	BaseTransaction
}

// todo add description
func (a *CoinBaseTransaction) SpecialCheck(para *interfaces.CheckParameters) (result elaerr.ELAError, end bool) {
	// todo special check, all check witch used isCoinbase function, need to move here.

	if para.BlockHeight >= para.CRCommitteeStartHeight {
		if para.ConsensusAlgorithm == 0x01 {
			if !a.outputs[0].ProgramHash.IsEqual(para.DestroyELAAddress) {
				return elaerr.Simple(elaerr.ErrTxInvalidOutput,
					errors.New("first output address should be "+
						"DestroyAddress in POW consensus algorithm")), true
			}
		} else {
			if !a.outputs[0].ProgramHash.IsEqual(para.CRAssetsAddress) {
				return elaerr.Simple(elaerr.ErrTxInvalidOutput,
					errors.New("first output address should be CR assets address")), true
			}
		}
	} else if !a.outputs[0].ProgramHash.IsEqual(para.FoundationAddress) {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput,
			errors.New("first output address should be foundation address")), true
	}

	return nil, true
}

func (a *CoinBaseTransaction) ContextCheck(para *interfaces.CheckParameters) (map[*common2.Input]common2.Output, elaerr.ELAError) {

	if err := a.CheckTxHeightVersion(para); err != nil {
		return nil, elaerr.Simple(elaerr.ErrTxHeightVersion, nil)
	}

	//// check if duplicated with transaction in ledger
	//if exist := b.db.IsTxHashDuplicate(txn.Hash()); exist {
	//	log.Warn("[CheckTransactionContext] duplicate transaction check failed.")
	//	return nil, elaerr.Simple(elaerr.ErrTxDuplicate, nil)
	//}
	if exist := a.IsTxHashDuplicate(*a.txHash); exist {
		//log.Warn("[CheckTransactionContext] duplicate transaction check failed.")
		return nil, elaerr.Simple(elaerr.ErrTxDuplicate, nil)
	}

	firstErr, end := a.SpecialCheck(para)
	if end {
		return nil, firstErr
	}

	return nil, nil
}
