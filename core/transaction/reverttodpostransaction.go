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

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type RevertToDPOSTransaction struct {
	BaseTransaction
}

func (t *RevertToDPOSTransaction) RegisterFunctions() {
	t.DefaultChecker.CheckTransactionSize = t.checkTransactionSize
	t.DefaultChecker.CheckTransactionInput = t.CheckTransactionInput
	t.DefaultChecker.CheckTransactionOutput = t.CheckTransactionOutput
	t.DefaultChecker.CheckTransactionPayload = t.CheckTransactionPayload
	t.DefaultChecker.HeightVersionCheck = t.HeightVersionCheck
	t.DefaultChecker.IsAllowedInPOWConsensus = t.IsAllowedInPOWConsensus
	t.DefaultChecker.SpecialContextCheck = t.SpecialContextCheck
	t.DefaultChecker.CheckAttributeProgram = t.CheckAttributeProgram
}

func (t *RevertToDPOSTransaction) CheckTransactionInput(params *TransactionParameters) error {
	if len(params.Transaction.Inputs()) != 0 {
		return errors.New("no cost transactions must has no input")
	}
	return nil
}

func (t *RevertToDPOSTransaction) CheckTransactionOutput(params *TransactionParameters) error {

	txn := params.Transaction
	if len(txn.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}
	if len(txn.Outputs()) != 0 {
		return errors.New("no cost transactions should have no output")
	}

	return nil
}
func (t *RevertToDPOSTransaction) CheckAttributeProgram(params *TransactionParameters) error {

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
	for _, program := range t.Programs() {
		if program.Code == nil {
			return fmt.Errorf("invalid program code nil")
		}
		if program.Parameter == nil {
			return fmt.Errorf("invalid program parameter nil")
		}
	}

	return nil
}

func (t *RevertToDPOSTransaction) CheckTransactionPayload(params *TransactionParameters) error {
	switch t.Payload().(type) {
	case *payload.RevertToDPOS:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *RevertToDPOSTransaction) IsAllowedInPOWConsensus(params *TransactionParameters, references map[*common2.Input]common2.Output) bool {
	return true
}

func (t *RevertToDPOSTransaction) HeightVersionCheck(params *TransactionParameters) error {
	if params.BlockHeight < params.Config.RevertToPOWStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before RevertToPOWStartHeight", t.TxType().Name()))
	}

	return nil
}

func (t *RevertToDPOSTransaction) SpecialContextCheck(params *TransactionParameters, references map[*common2.Input]common2.Output) (elaerr.ELAError, bool) {
	p, ok := t.Payload().(*payload.RevertToDPOS)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload.RevertToDPOS")), true
	}
	if p.WorkHeightInterval != payload.WorkHeightInterval {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid WorkHeightInterval")), true

	}

	// check dpos state
	if params.BlockChain.GetState().GetConsensusAlgorithm() != state.POW {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid GetConsensusAlgorithm() != state.POW")), true
	}

	// to avoid init DPOSWorkHeight repeatedly
	if params.BlockChain.GetState().DPOSWorkHeight > params.BlockHeight {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("already receieved  revertodpos")), true
	}

	if err := checkArbitratorsSignatures(t.Programs()[0]); err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	return nil, true
}

func checkArbitratorsSignatures(program *program.Program) error {
	code := program.Code
	// Get N parameter
	n := int(code[len(code)-2]) - crypto.PUSH1 + 1
	// Get M parameter
	m := int(code[0]) - crypto.PUSH1 + 1

	var arbitratorsCount int
	arbiters := blockchain.DefaultLedger.Arbitrators.GetArbitrators()
	for _, a := range arbiters {
		if a.IsNormal {
			arbitratorsCount++
		}
	}
	minSignCount := int(float64(blockchain.DefaultLedger.Arbitrators.GetArbitersCount())*
		state.MajoritySignRatioNumerator/state.MajoritySignRatioDenominator) + 1
	if m < 1 || m > n || n != arbitratorsCount || m < minSignCount {
		return errors.New("invalid multi sign script code")
	}
	publicKeys, err := crypto.ParseMultisigScript(code)
	if err != nil {
		return err
	}

	for _, pk := range publicKeys {
		if !blockchain.DefaultLedger.Arbitrators.IsArbitrator(pk[1:]) {
			return errors.New("invalid multi sign public key")
		}
	}

	return nil
}
