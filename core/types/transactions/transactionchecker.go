// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transactions

import (
	"fmt"
	"github.com/elastos/Elastos.ELA/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type DefaultChecker struct {
	IsTxHashDuplicateFunction func(txhash common.Uint256) bool
	GetTxReferenceFunction    func(para *interfaces.CheckParameters) (
		map[*common2.Input]common2.Output, error)
}

func (a *DefaultChecker) ContextCheck(para *interfaces.CheckParameters) (
	map[*common2.Input]common2.Output, elaerr.ELAError) {

	if err := a.CheckTxHeightVersion(para); err != nil {
		return nil, elaerr.Simple(elaerr.ErrTxHeightVersion, nil)
	}

	if exist := a.IsTxHashDuplicate(para.Transaction.Hash()); exist {
		//log.Warn("[CheckTransactionContext] duplicate transaction check failed.")
		return nil, elaerr.Simple(elaerr.ErrTxDuplicate, nil)
	}

	references, err := a.GetTxReference(para)
	if err != nil {
		//log.Warn("[CheckTransactionContext] get transaction reference failed")
		return nil, elaerr.Simple(elaerr.ErrTxUnknownReferredTx, nil)
	}

	// todo add more common check
	// ...

	firstErr, end := a.SpecialCheck(para)
	if end {
		return nil, firstErr
	}

	return references, nil
}

func (a *DefaultChecker) CheckTxHeightVersion(para *interfaces.CheckParameters) error {
	// todo default check
	return nil
}

func (a *DefaultChecker) IsTxHashDuplicate(txHash common.Uint256) bool {
	return a.IsTxHashDuplicateFunction(txHash)
}

func (a *DefaultChecker) GetTxReference(para *interfaces.CheckParameters) (
	map[*common2.Input]common2.Output, error) {
	return a.GetTxReferenceFunction(para)
}

func (a *DefaultChecker) CheckPOWConsensusTransaction(
	para *interfaces.CheckParameters, references map[*common2.Input]common2.Output) error {
	// todo default check
	return nil
}

func (a *DefaultChecker) SpecialCheck(para *interfaces.CheckParameters) (elaerr.ELAError, bool) {
	fmt.Println("default check")
	return nil, false
}
