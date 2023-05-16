// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"math"

	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type UpdateVersionTransaction struct {
	BaseTransaction
}

func (t *UpdateVersionTransaction) CheckTransactionInput() error {
	if len(t.Inputs()) != 0 {
		return errors.New("no cost transactions must has no input")
	}
	return nil
}

func (t *UpdateVersionTransaction) CheckTransactionOutput() error {

	if len(t.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}
	if len(t.Outputs()) != 0 {
		return errors.New("no cost transactions should have no output")
	}

	return nil
}

func (t *UpdateVersionTransaction) CheckAttributeProgram() error {

	// check programs count and attributes count
	if len(t.Programs()) != 1 {
		return errors.New("inactive arbitrators transactions should have one and only one program")
	}
	if len(t.Attributes()) != 1 {
		return errors.New("inactive arbitrators transactions should have one and only one arbitrator")
	}

	// Check attributes
	for _, attr := range t.Attributes() {
		if !common2.IsValidAttributeType(attr.Usage) {
			return fmt.Errorf("invalid attribute usage %v", attr.Usage)
		}
	}

	// Check programs
	if len(t.Programs()) == 0 {
		return fmt.Errorf("no programs found in transaction")
	}
	for _, p := range t.Programs() {
		if p.Code == nil {
			return fmt.Errorf("invalid program code nil")
		}
		if len(p.Code) < program.MinProgramCodeSize {
			return fmt.Errorf("invalid program code size")
		}
		if p.Parameter == nil {
			return fmt.Errorf("invalid program parameter nil")
		}
	}

	return nil
}

func (t *UpdateVersionTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.UpdateVersion:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *UpdateVersionTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *UpdateVersionTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
	payload, ok := t.Payload().(*payload.UpdateVersion)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	if payload.EndHeight <= payload.StartHeight ||
		payload.StartHeight < t.parameters.BlockChain.GetHeight() {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid update version height")), true
	}

	if err := checkCRCArbitratorsSignatures(t.Programs()[0]); err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	return nil, true
}
