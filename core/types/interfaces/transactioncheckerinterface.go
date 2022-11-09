// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package interfaces

import (
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type TransactionChecker interface {
	BaseTransactionChecker

	SanityCheck(p Parameters) elaerr.ELAError

	ContextCheck(p Parameters) (map[*common2.Input]common2.Output, elaerr.ELAError)

	SetParameters(p interface{}) elaerr.ELAError

	SetReferences(ref map[*common2.Input]common2.Output)
}

type BaseTransactionChecker interface {

	// check height version
	HeightVersionCheck() error

	/// SANITY CHECK
	// rewrite this function to check the transaction size, otherwise the
	// transaction size if compare with default value: MaxBlockContextSize
	CheckTransactionSize() error
	// check transaction inputs
	CheckTransactionInput() error
	// check transaction outputs
	CheckTransactionOutput() error
	// check transaction attributes and programs
	CheckAttributeProgram() error
	// check transaction payload type
	CheckTransactionPayload() error

	/// CONTEXT CHECK
	// if the transaction should create in POW need to return true
	IsAllowedInPOWConsensus() bool
	// the special context check of transaction, such as check the transaction payload
	SpecialContextCheck() (error elaerr.ELAError, end bool)
	//Check Transaction Fee
	CheckTransactionFee(references map[*common2.Input]common2.Output) error
}
