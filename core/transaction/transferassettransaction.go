// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"math"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

type TransferAssetTransaction struct {
	BaseTransaction
}

func (t *TransferAssetTransaction) CheckTransactionOutput() error {
	blockHeight := t.parameters.BlockHeight
	if len(t.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}

	if len(t.Outputs()) < 1 {
		return errors.New("transaction has no outputs")
	}

	// check if output address is valid
	specialOutputCount := 0
	ELAAssetID, _ := common.Uint256FromHexString(core.ELAAssetID)
	for _, output := range t.Outputs() {
		if output.AssetID != *ELAAssetID {
			return errors.New("asset ID in output is invalid")
		}

		// output value must >= 0
		if output.Value < common.Fixed64(0) {
			return errors.New("invalid transaction UTXO output")
		}

		if err := checkOutputProgramHash(blockHeight, output.ProgramHash); err != nil {
			return err
		}

		if t.Version() >= common2.TxVersion09 {
			if output.Type != common2.OTNone {
				specialOutputCount++
			}
			if err := checkTransferAssetOutputPayload(output); err != nil {
				return err
			}
		}
	}

	return nil
}

func checkTransferAssetOutputPayload(output *common2.Output) error {
	// common2.OTVote information can only be placed in TransferAsset transaction.
	switch output.Type {
	case common2.OTVote:
		if contract.GetPrefixType(output.ProgramHash) !=
			contract.PrefixStandard {
			return errors.New("output address should be standard")
		}
	case common2.OTNone:
	case common2.OTMapping:
	//case common2.OTDposV2Vote:
	//	if contract.GetPrefixType(output.ProgramHash) !=
	//		contract.PrefixDPoSV2 {
	//		return errors.New("output address should be dposV2")
	//	}
	default:
		return errors.New("transaction type dose not match the output payload type")
	}

	return output.Payload.Validate()
}

func (t *TransferAssetTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.TransferAsset:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *TransferAssetTransaction) IsAllowedInPOWConsensus() bool {
	if t.Version() >= common2.TxVersion09 {
		var containVoteOutput bool
		for _, output := range t.Outputs() {
			if output.Type == common2.OTVote {
				p := output.Payload.(*outputpayload.VoteOutput)
				for _, vote := range p.Contents {
					switch vote.VoteType {
					case outputpayload.Delegate:
					case outputpayload.CRC:
						log.Warn("not allow to vote CR in POW consensus")
						return false
					case outputpayload.CRCProposal:
						log.Warn("not allow to vote CRC proposal in POW consensus")
						return false
					case outputpayload.CRCImpeachment:
						log.Warn("not allow to vote CRImpeachment in POW consensus")
						return false
					}
				}
				containVoteOutput = true
			}
		}
		if !containVoteOutput {
			log.Warn("not allow to transfer asset in POW consensus")
			return false
		}

		inputProgramHashes := make(map[common.Uint168]struct{})
		for _, output := range t.references {
			inputProgramHashes[output.ProgramHash] = struct{}{}
		}
		outputProgramHashes := make(map[common.Uint168]struct{})
		for _, output := range t.Outputs() {
			outputProgramHashes[output.ProgramHash] = struct{}{}
		}
		for k, _ := range outputProgramHashes {
			if _, ok := inputProgramHashes[k]; !ok {
				log.Warn("output program hash is not in inputs")
				return false
			}
		}
	} else {
		log.Warn("not allow to transfer asset in POW consensus")
		return false
	}
	return true
}

func (t *TransferAssetTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight >= chainParams.CRConfiguration.CRVotingStartHeight {
		return nil
	}
	if t.Version() >= common2.TxVersion09 {
		for _, output := range t.Outputs() {
			if output.Type != common2.OTVote {
				continue
			}
			p, _ := output.Payload.(*outputpayload.VoteOutput)
			if p.Version >= outputpayload.VoteProducerAndCRVersion {
				return errors.New("not support " +
					"VoteProducerAndCRVersion before CRVotingStartHeight")
			}
		}
	}
	return nil
}
