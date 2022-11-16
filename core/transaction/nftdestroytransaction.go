// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"

	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type NFTDestroyTransactionFromSideChain struct {
	BaseTransaction
}

func (t *NFTDestroyTransactionFromSideChain) CheckTransactionPayload() error {
	_, ok := t.Payload().(*payload.NFTDestroyFromSideChain)
	if !ok {
		return errors.New("Invalid NFTDestroyFromSideChain payload type")
	}

	return nil
}

func (t *NFTDestroyTransactionFromSideChain) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config
	if blockHeight < chainParams.DPoSConfiguration.NFTStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before NFTStartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *NFTDestroyTransactionFromSideChain) IsAllowedInPOWConsensus() bool {
	return false
}

//todo need rewrite CheckTransactionFee

func (t *NFTDestroyTransactionFromSideChain) SpecialContextCheck() (elaerr.ELAError, bool) {
	nftDestroyPayload, ok := t.Payload().(*payload.NFTDestroyFromSideChain)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}
	state := t.parameters.BlockChain.GetState()
	//producers := state.GetDposV2Producers()

	canDestroyIDs := state.CanNFTDestroy(nftDestroyPayload.ID)
	if len(canDestroyIDs) != len(nftDestroyPayload.ID) {
		return elaerr.Simple(elaerr.ErrTxPayload,
			errors.New(" NFT can not destroy")), true
	}

	var err error
	err = t.checkNFTDestroyTransactionFromSideChain()
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	return nil, false
}

func (t *NFTDestroyTransactionFromSideChain) checkNFTDestroyTransactionFromSideChain() error {

	height := t.parameters.BlockHeight
	for _, p := range t.Programs() {
		publicKeys, m, n, err := crypto.ParseCrossChainScriptV1(p.Code)
		if err != nil {
			return err
		}
		var arbiters []*state.ArbiterInfo
		var minCount uint32
		if height >= t.parameters.Config.DPoSConfiguration.DPOSNodeCrossChainHeight {
			arbiters = blockchain.DefaultLedger.Arbitrators.GetArbitrators()
			minCount = uint32(t.parameters.Config.DPoSConfiguration.NormalArbitratorsCount) + 1
		} else {
			arbiters = blockchain.DefaultLedger.Arbitrators.GetCRCArbiters()
			minCount = t.parameters.Config.CRConfiguration.CRAgreementCount
		}
		var arbitersCount int
		for _, c := range arbiters {
			if !c.IsNormal {
				continue
			}
			arbitersCount++
		}
		if n != arbitersCount {
			return errors.New("invalid arbiters total count in code")
		}
		if m < int(minCount) {
			return errors.New("invalid arbiters sign count in code")
		}
		if err := checkCrossChainArbitrators(publicKeys); err != nil {
			return err
		}
	}

	return nil
}
