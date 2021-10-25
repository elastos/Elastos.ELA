// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import (
	"fmt"
	"github.com/elastos/Elastos.ELA/common"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type CheckParameters struct {
	// others
	BlockHeight            uint32
	CRCommitteeStartHeight uint32
	ConsensusAlgorithm     byte
	DestroyELAAddress      common.Uint168
	CRAssetsAddress        common.Uint168
	FoundationAddress      common.Uint168

	TxHash common.Uint256
}

type DefaultChecker struct {
	IsTxHashDuplicateFunction func(txhash common.Uint256) bool
	GetTxReference func(para *CheckParameters) error
}

func (a *DefaultChecker) SpecialCheck(para *CheckParameters) (elaerr.ELAError, bool) {
	fmt.Println("default check")
	return nil, false
}

func (a *DefaultChecker) CheckTxHeightVersion(para *CheckParameters) error {
	// todo default check
	return nil
}

func (a *DefaultChecker) IsTxHashDuplicate(txHash common.Uint256) bool {
	return a.IsTxHashDuplicateFunction(txHash)
}

func (a *DefaultChecker) ContextCheck(para *CheckParameters) elaerr.ELAError {

	if err := a.CheckTxHeightVersion(para); err != nil {
		return elaerr.Simple(elaerr.ErrTxHeightVersion, nil)
	}

	//// check if duplicated with transaction in ledger
	//if exist := b.db.IsTxHashDuplicate(txn.Hash()); exist {
	//	log.Warn("[CheckTransactionContext] duplicate transaction check failed.")
	//	return nil, elaerr.Simple(elaerr.ErrTxDuplicate, nil)
	//}
	if exist := a.IsTxHashDuplicate(para.TxHash); exist {
		//log.Warn("[CheckTransactionContext] duplicate transaction check failed.")
		return elaerr.Simple(elaerr.ErrTxDuplicate, nil)
	}

	// todo add more common check
	// ...

	firstErr, end := a.SpecialCheck(para)
	if end {
		return firstErr
	}

	return nil
}
