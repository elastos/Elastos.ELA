// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"math"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type CoinBaseTransaction struct {
	BaseTransaction
}

func (t *CoinBaseTransaction) CheckTransactionInput() error {
	if len(t.Inputs()) != 1 {
		return errors.New("coinbase must has only one input")
	}
	inputHash := t.Inputs()[0].Previous.TxID
	inputIndex := t.Inputs()[0].Previous.Index
	sequence := t.Inputs()[0].Sequence
	if !inputHash.IsEqual(common.EmptyHash) ||
		inputIndex != math.MaxUint16 || sequence != math.MaxUint32 {
		return errors.New("invalid coinbase input")
	}

	return nil
}

func (t *CoinBaseTransaction) CheckTransactionOutput() error {

	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if len(t.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}
	if len(t.Outputs()) < 2 {
		return errors.New("coinbase output is not enough, at least 2")
	}

	foundationReward := t.Outputs()[0].Value
	var totalReward = common.Fixed64(0)
	if blockHeight < chainParams.PublicDPOSHeight {
		for _, output := range t.Outputs() {
			if output.AssetID != config.ELAAssetID {
				return errors.New("asset ID in coinbase is invalid")
			}
			totalReward += output.Value
		}

		if foundationReward < common.Fixed64(float64(totalReward)*0.3) {
			return errors.New("reward to foundation in coinbase < 30%")
		}
	} else {
		// check the ratio of FoundationAddress reward with miner reward
		totalReward = t.Outputs()[0].Value + t.Outputs()[1].Value
		if len(t.Outputs()) == 2 && foundationReward <
			common.Fixed64(float64(totalReward)*0.3/0.65) {
			return errors.New("reward to foundation in coinbase < 30%")
		}
	}

	return nil
}

func (t *CoinBaseTransaction) CheckAttributeProgram() error {
	// no need to check attribute and program
	if len(t.Programs()) != 0 {
		return errors.New("transaction should have no programs")
	}
	return nil
}

func (t *CoinBaseTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.CoinBase:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *CoinBaseTransaction) IsAllowedInPOWConsensus() bool {

	return true
}

func (a *CoinBaseTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {

	para := a.parameters
	if para.BlockHeight >= para.Config.CRCommitteeStartHeight {
		if para.BlockChain.GetState().GetConsensusAlgorithm() == 0x01 {
			if !a.outputs[0].ProgramHash.IsEqual(para.Config.DestroyELAAddress) {
				return elaerr.Simple(elaerr.ErrTxInvalidOutput,
					errors.New("first output address should be "+
						"DestroyAddress in POW consensus algorithm")), true
			}
		} else {
			if !a.outputs[0].ProgramHash.IsEqual(para.Config.CRAssetsAddress) {
				return elaerr.Simple(elaerr.ErrTxInvalidOutput,
					errors.New("first output address should be CR assets address")), true
			}
		}
	} else if !a.outputs[0].ProgramHash.IsEqual(para.Config.Foundation) {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput,
			errors.New("first output address should be foundation address")), true
	}

	return nil, true
}

func (a *CoinBaseTransaction) ContextCheck(paras interfaces.Parameters) (map[*common2.Input]common2.Output, elaerr.ELAError) {

	if err := a.SetParameters(paras); err != nil {
		log.Warn("[CheckTransactionContext] set parameters failed.")
		return nil, elaerr.Simple(elaerr.ErrTxDuplicate, errors.New("invalid parameters"))
	}

	if err := a.HeightVersionCheck(); err != nil {
		log.Warn("[CheckTransactionContext] height version check failed.")
		return nil, elaerr.Simple(elaerr.ErrTxHeightVersion, nil)
	}

	// check if duplicated with transaction in ledger
	if exist := a.IsTxHashDuplicate(*a.txHash); exist {
		log.Warn("[CheckTransactionContext] duplicate transaction check failed.")
		return nil, elaerr.Simple(elaerr.ErrTxDuplicate, nil)
	}

	err, end := a.SpecialContextCheck()
	if end {
		log.Warn("[CheckTransactionContext] SpecialContextCheck failed:", err)
		return nil, err
	}

	return nil, nil
}
