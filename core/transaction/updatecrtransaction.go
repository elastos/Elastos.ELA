// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type UpdateCRTransaction struct {
	BaseTransaction
}

func (t *UpdateCRTransaction) RegisterFunctions() {
	t.DefaultChecker.CheckTransactionSize = t.checkTransactionSize
	t.DefaultChecker.CheckTransactionInput = t.checkTransactionInput
	t.DefaultChecker.CheckTransactionOutput = t.checkTransactionOutput
	t.DefaultChecker.CheckTransactionPayload = t.CheckTransactionPayload
	t.DefaultChecker.HeightVersionCheck = t.heightVersionCheck
	t.DefaultChecker.IsAllowedInPOWConsensus = t.IsAllowedInPOWConsensus
	t.DefaultChecker.SpecialContextCheck = t.SpecialContextCheck
	t.DefaultChecker.CheckAttributeProgram = t.checkAttributeProgram
}

func (t *UpdateCRTransaction) CheckTransactionPayload(params *TransactionParameters) error {
	switch t.Payload().(type) {
	case *payload.CRInfo:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *UpdateCRTransaction) IsAllowedInPOWConsensus(params *TransactionParameters, references map[*common2.Input]common2.Output) bool {
	return false
}

func (t *UpdateCRTransaction) SpecialContextCheck(params *TransactionParameters, references map[*common2.Input]common2.Output) (elaerr.ELAError, bool) {
	info, ok := t.Payload().(*payload.CRInfo)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	if err := checkStringField(info.NickName, "NickName", false); err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	// check url
	if err := checkStringField(info.Url, "Url", true); err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	// get CID program hash and check length of code
	ct, err := contract.CreateCRIDContractByCode(info.Code)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}
	programHash := ct.ToProgramHash()
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	// check CID
	if !info.CID.IsEqual(*programHash) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid cid address")), true
	}

	if params.BlockHeight >=params.Config.RegisterCRByDIDHeight &&
		t.PayloadVersion() == payload.CRInfoDIDVersion {
		// get DID program hash

		programHash, err = getDIDFromCode(info.Code)
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid info.Code")), true
		}
		// check DID
		if !info.DID.IsEqual(*programHash) {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid did address")), true
		}
	}

	// check code and signature
	if err := blockchain.CrInfoSanityCheck(info, t.PayloadVersion()); err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}
	if !params.BlockChain.GetCRCommittee().IsInVotingPeriod(params.BlockHeight) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("should create tx during voting period")), true
	}

	cr := params.BlockChain.GetCRCommittee().GetCandidate(info.CID)
	if cr == nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("updating unknown CR")), true
	}
	if cr.State() != crstate.Pending && cr.State() != crstate.Active {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("updating canceled or returned CR")), true
	}

	// check nickname usage.
	if cr.Info().NickName != info.NickName &&
		params.BlockChain.GetCRCommittee().ExistCandidateByNickname(info.NickName) {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("nick name %s already exist", info.NickName)), true
	}

	return nil, false
}
