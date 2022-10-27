// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"

	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type CreateNFTTransaction struct {
	BaseTransaction
}

func (t *CreateNFTTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.CreateNFT:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *CreateNFTTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *CreateNFTTransaction) CheckAttributeProgram() error {

	if t.PayloadVersion() == payload.CreateNFTVersion {
		return nil
	}

	// Check attributes
	for _, attr := range t.Attributes() {
		if !common2.IsValidAttributeType(attr.Usage) {
			return fmt.Errorf("invalid attribute usage %v", attr.Usage)
		}
	}

	// Check programs
	if len(t.Programs()) != 1 {
		return fmt.Errorf("need to be only one program")
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

func (t *CreateNFTTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.DPoSConfiguration.NFTStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before NFTStartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *CreateNFTTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
	pld, ok := t.Payload().(*payload.CreateNFT)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	state := t.parameters.BlockChain.GetState()
	producers := state.GetDposV2Producers()
	for _, p := range producers {
		for stakeAddress, votesInfo := range p.GetAllDetailedDPoSV2Votes() {
			for referKey, voteInfo := range votesInfo {
				if referKey.IsEqual(pld.ID) {
					ct, _ := contract.CreateStakeContractByCode(referKey.Bytes())
					nftStakeAddress := ct.ToProgramHash()
					if stakeAddress.IsEqual(*nftStakeAddress) {
						return elaerr.Simple(elaerr.ErrTxPayload,
							errors.New("the NFT has been created yet")), true
					}
					log.Info("create NFT, vote information:", voteInfo)
					return nil, false
				}
			}
		}
	}

	// stake address need to be same from code
	ct, _ := contract.CreateStakeContractByCode(t.programs[0].Code)
	stakeAddress, err := ct.ToProgramHash().ToAddress()
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid stake address")), true
	}
	if stakeAddress != pld.StakeAddress {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("stake address not from code")), true
	}

	return elaerr.Simple(elaerr.ErrTxPayload, errors.New("the NFT ID does not exist")), true
}
