// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type UnstakeTransaction struct {
	BaseTransaction
}

func (t *UnstakeTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.DposV2StartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before DposV2StartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *UnstakeTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.Unstake:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *UnstakeTransaction) CheckTransactionOutput() error {
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

	if contract.GetPrefixType(t.Outputs()[0].ProgramHash) != contract.PrefixStandard &&
		contract.GetPrefixType(t.Outputs()[0].ProgramHash) != contract.PrefixMultiSig {
		return errors.New("first output address need to be Standard or MultiSig address")
	}

	if len(t.Outputs()) == 2 {
		// check output address, need to be stake address
		addr, err := t.outputs[1].ProgramHash.ToAddress()
		if err != nil {
			return errors.New("invalid  output address")
		}
		if addr != t.parameters.Config.StakeAddress {
			return errors.New("second output address need to be stake address")
		}
	}

	return nil
}

func (t *UnstakeTransaction) CheckAttributeProgram() error {
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

func (t *UnstakeTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *UnstakeTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {

	// 1.check if unused vote rights enough
	// 2.return value if payload need to be equal to outputs

	// check if unused vote rights enough
	code := t.Programs()[0].Code
	ct, err := contract.CreateStakeContractByCode(code)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput, err), true
	}
	stakeProgramHash := ct.ToProgramHash()
	state := t.parameters.BlockChain.GetState()
	voteRights := state.DposV2VoteRights[*stakeProgramHash]
	usedDposVoteRights := state.DposVotes[*stakeProgramHash]
	usedDposV2VoteRights := state.DposV2Votes[*stakeProgramHash]
	usedCRVoteRights := state.CRVotes[*stakeProgramHash]
	usedCRImpeachmentVoteRights := state.CRImpeachmentVotes[*stakeProgramHash]
	usedCRCProposalVoteRights := state.CRCProposalVotes[*stakeProgramHash]

	pl := t.Payload().(*payload.Unstake)
	if pl.Value > voteRights-usedDposVoteRights ||
		pl.Value > voteRights-usedDposV2VoteRights ||
		pl.Value > voteRights-usedCRVoteRights ||
		pl.Value > voteRights-usedCRImpeachmentVoteRights ||
		pl.Value > voteRights-usedCRCProposalVoteRights {
		return elaerr.Simple(elaerr.ErrTxPayload,
			errors.New("vote rights not enough")), true
	}

	// return value if payload need to be equal to outputs
	inputsStakeAddr := make(map[common.Uint168]struct{})
	inputsStakeAmount := common.Fixed64(0)
	for _, o := range t.references {
		addr, err := o.ProgramHash.ToAddress()
		if err != nil {
			continue
		}
		if addr != t.parameters.Config.StakeAddress {
			continue
		}
		if contract.GetPrefixType(o.ProgramHash) != contract.PrefixDposV2 {
			continue
		}
		inputsStakeAddr[o.ProgramHash] = struct{}{}
		inputsStakeAmount += o.Value
	}
	if len(inputsStakeAddr) != 1 {
		return elaerr.Simple(elaerr.ErrTxInvalidInput,
			errors.New("has different input address")), true
	}

	var stakeAmount common.Fixed64
	if len(t.Outputs()) == 1 {
		// if have no change, need to use inputs amount
		stakeAmount = inputsStakeAmount
	} else {
		// if have change, need to use inputs amount - change amount
		stakeAmount = inputsStakeAmount - t.outputs[1].Value
	}
	if stakeAmount != pl.Value {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput,
			errors.New("payload value is not equal to output value")), true
	}

	return nil, false
}
