// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/vm"
)

type RegisterCRTransaction struct {
	BaseTransaction
}

func (t *RegisterCRTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.CRInfo:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *RegisterCRTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *RegisterCRTransaction) HeightVersionCheck() error {
	txn := t.contextParameters.Transaction
	blockHeight := t.contextParameters.BlockHeight
	chainParams := t.contextParameters.Config

	if blockHeight < chainParams.CRVotingStartHeight ||
		(blockHeight < chainParams.RegisterCRByDIDHeight &&
			txn.PayloadVersion() != payload.CRInfoVersion) {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before CRVotingStartHeight", txn.TxType().Name()))
	}
	return nil
}

func (t *RegisterCRTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
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

	if !t.contextParameters.BlockChain.GetCRCommittee().IsInVotingPeriod(t.contextParameters.BlockHeight) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("should create tx during voting period")), true
	}

	if t.contextParameters.BlockChain.GetCRCommittee().ExistCandidateByNickname(info.NickName) {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("nick name %s already inuse", info.NickName)), true
	}

	cr := t.contextParameters.BlockChain.GetCRCommittee().GetCandidate(info.CID)
	if cr != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("cid %s already exist", info.CID)), true
	}

	// get CID program hash and check length of code
	ct, err := contract.CreateCRIDContractByCode(info.Code)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}
	programHash := ct.ToProgramHash()

	// check if program code conflict with producer public keys
	if info.Code[len(info.Code)-1] == vm.CHECKSIG {
		pk := info.Code[1 : len(info.Code)-1]
		if t.contextParameters.BlockChain.GetState().ProducerExists(pk) {
			return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("public key %s already inuse in producer list",
				common.BytesToHexString(info.Code[1:len(info.Code)-1]))), true
		}
		if blockchain.DefaultLedger.Arbitrators.IsCRCArbitrator(pk) {
			return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("public key %s already inuse in CRC list",
				common.BytesToHexString(info.Code[0:len(info.Code)-1]))), true
		}
	}

	// check CID
	if !info.CID.IsEqual(*programHash) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid cid address")), true
	}

	if t.contextParameters.BlockHeight >= t.contextParameters.Config.RegisterCRByDIDHeight &&
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

	// check deposit coin
	var depositCount int
	for _, output := range t.Outputs() {
		if contract.GetPrefixType(output.ProgramHash) == contract.PrefixDeposit {
			depositCount++
			// get deposit program hash
			ct, err := contract.CreateDepositContractByCode(info.Code)
			if err != nil {
				return elaerr.Simple(elaerr.ErrTxPayload, err), true
			}
			programHash := ct.ToProgramHash()
			if !output.ProgramHash.IsEqual(*programHash) {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("deposit address does not"+
					" match the code in payload")), true
			}
			if output.Value < crstate.MinDepositAmount {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("CR deposit amount is insufficient")), true
			}
		}
	}
	if depositCount != 1 {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("there must be only one deposit address in outputs")), true
	}

	return nil, false
}

func getDIDFromCode(code []byte) (*common.Uint168, error) {
	newCode := make([]byte, len(code))
	copy(newCode, code)
	didCode := append(newCode[:len(newCode)-1], common.DID)

	if ct1, err := contract.CreateCRIDContractByCode(didCode); err != nil {
		return nil, err
	} else {
		return ct1.ToProgramHash(), nil
	}
}
