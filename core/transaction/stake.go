// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type StakeTransaction struct {
	BaseTransaction
}

func (t *StakeTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.DPoSV2StartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before DPoSV2StartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *StakeTransaction) CheckTransactionOutput() error {
	if len(t.Outputs()) > 2 {
		return errors.New("output count should not be greater than 2")
	}

	if len(t.Outputs()) < 1 {
		return errors.New("transaction has no outputs")
	}

	// check if output address is valid
	for _, output := range t.Outputs() {
		if output.AssetID != config.ELAAssetID {
			return errors.New("asset ID in output is invalid")
		}

		// output value must >= 0
		if output.Value < common.Fixed64(0) {
			return errors.New("invalid transaction UTXO output")
		}
	}

	// check output payload
	if t.outputs[0].Type != common2.OTStake {
		return errors.New("invalid output type")
	}
	p := t.outputs[0].Payload
	if p == nil {
		return errors.New("invalid output payload")
	}
	if _, ok := p.(*outputpayload.StakeOutput); !ok {
		return errors.New("invalid exchange vote output payload")
	}
	if err := p.Validate(); err != nil {
		return err
	}
	// check output address, need to be stake address
	addr, err := t.outputs[0].ProgramHash.ToAddress()
	if err != nil {
		return errors.New("invalid first output address")
	}
	if addr != t.parameters.Config.StakeAddress {
		return errors.New("first output address need to be stake address")
	}

	// check the second output
	if len(t.Outputs()) == 2 {
		if contract.GetPrefixType(t.Outputs()[1].ProgramHash) != contract.PrefixStandard &&
			contract.GetPrefixType(t.Outputs()[1].ProgramHash) != contract.PrefixMultiSig {
			return errors.New("second output address need to be Standard or MultiSig")
		}
	}

	return nil
}

func (t *StakeTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.Stake:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *StakeTransaction) CheckAttributeProgram() error {
	// Check attributes
	for _, attr := range t.Attributes() {
		if !common2.IsValidAttributeType(attr.Usage) {
			return fmt.Errorf("invalid attribute usage %v", attr.Usage)
		}
	}

	// Check programs
	if len(t.Programs()) != 1 {
		return errors.New("transaction should have only one program")
	}
	if t.Programs()[0].Code == nil {
		return fmt.Errorf("invalid program code nil")
	}
	if t.Programs()[0].Parameter == nil {
		return fmt.Errorf("invalid program parameter nil")
	}

	return nil
}

func (t *StakeTransaction) IsAllowedInPOWConsensus() bool {

	return true
}

func (t *StakeTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {
	return nil, false
}
