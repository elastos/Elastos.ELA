// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"

	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
)

type TransferAssetTransaction struct {
	BaseTransaction
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
	txn := t.contextParameters.Transaction
	blockHeight := t.contextParameters.BlockHeight
	chainParams := t.contextParameters.Config

	if blockHeight >= chainParams.CRVotingStartHeight {
		return nil
	}
	if txn.Version() >= common2.TxVersion09 {
		for _, output := range txn.Outputs() {
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
