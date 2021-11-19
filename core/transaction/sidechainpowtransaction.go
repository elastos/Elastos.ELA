// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	elaerr "github.com/elastos/Elastos.ELA/errors"
	"math"
)

type SideChainPOWTransaction struct {
	BaseTransaction
}

func (t *SideChainPOWTransaction) CheckTransactionInput() error {

	txn := t.sanityParameters.Transaction
	if t.IsNewSideChainPowTx() {
		if len(txn.Inputs()) != 0 {
			return errors.New("no cost transactions must has no input")
		}
	} else {
		if len(txn.Inputs()) <= 0 {
			return errors.New("transaction has no inputs")
		}
		existingTxInputs := make(map[string]struct{})
		for _, input := range txn.Inputs() {
			if input.Previous.TxID.IsEqual(common.EmptyHash) && (input.Previous.Index == math.MaxUint16) {
				return errors.New("invalid transaction input")
			}
			if _, exists := existingTxInputs[input.ReferKey()]; exists {
				return errors.New("duplicated transaction inputs")
			} else {
				existingTxInputs[input.ReferKey()] = struct{}{}
			}
		}
	}

	return nil
}
func (t *SideChainPOWTransaction) CheckTransactionOutput() error {

	txn := t.sanityParameters.Transaction
	blockHeight := t.sanityParameters.BlockHeight
	chainParams := t.sanityParameters.Config
	if len(txn.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}

	if txn.IsNewSideChainPowTx() {
		if len(txn.Outputs()) != 1 {
			return errors.New("new sideChainPow tx must have only one output")
		}
		if txn.Outputs()[0].Value != 0 {
			return errors.New("the value of new sideChainPow tx output must be 0")
		}
		if txn.Outputs()[0].Type != common2.OTNone {
			return errors.New("the type of new sideChainPow tx output must be OTNone")
		}
	} else {
		if len(txn.Outputs()) < 1 {
			return errors.New("transaction has no outputs")
		}

		// check if output address is valid
		specialOutputCount := 0
		for _, output := range txn.Outputs() {
			if output.AssetID != config.ELAAssetID {
				return errors.New("asset ID in output is invalid")
			}

			// output value must >= 0
			if output.Value < common.Fixed64(0) {
				return errors.New("invalid transaction UTXO output")
			}

			if err := checkOutputProgramHash(blockHeight, output.ProgramHash); err != nil {
				return err
			}

			if txn.Version() >= common2.TxVersion09 {
				if output.Type != common2.OTNone {
					specialOutputCount++
				}
				if err := checkOutputPayload(txn.TxType(), output); err != nil {
					return err
				}
			}
		}

		if t.sanityParameters.BlockChain.GetHeight() >= chainParams.PublicDPOSHeight && specialOutputCount > 1 {
			return errors.New("special output count should less equal than 1")
		}
	}

	return nil
}

func (t *SideChainPOWTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *SideChainPOWTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
	arbitrator := blockchain.DefaultLedger.Arbitrators.GetOnDutyCrossChainArbitrator()
	payloadSideChainPow, ok := t.Payload().(*payload.SideChainPow)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("side mining transaction has invalid payload")), true
	}

	if arbitrator == nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("there is no arbiter on duty")), true
	}

	publicKey, err := crypto.DecodePoint(arbitrator)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	buf := new(bytes.Buffer)
	err = payloadSideChainPow.Serialize(buf, payload.SideChainPowVersion)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	err = crypto.Verify(*publicKey, buf.Bytes()[0:68], payloadSideChainPow.Signature)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("Arbitrator is not matched. "+err.Error())), true
	}

	if t.IsNewSideChainPowTx() {
		return nil, true
	}

	return nil, false
}
