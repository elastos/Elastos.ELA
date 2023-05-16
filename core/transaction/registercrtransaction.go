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
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	switch t.payloadVersion {
	case payload.CRInfoVersion:
		if blockHeight < chainParams.CRConfiguration.CRVotingStartHeight {
			return errors.New(fmt.Sprintf("not support %s transaction "+
				"before CRVotingStartHeight", t.TxType().Name()))
		}
	case payload.CRInfoDIDVersion:
		if blockHeight < chainParams.CRConfiguration.RegisterCRByDIDHeight {
			return errors.New(fmt.Sprintf("not support %s transaction "+
				"before RegisterCRByDIDHeight", t.TxType().Name()))
		}
	case payload.CRInfoSchnorrVersion:
		if blockHeight < chainParams.CRSchnorrStartHeight {
			return errors.New(fmt.Sprintf("not support %s transaction "+
				"before CRSchnorrStartHeight", t.TxType().Name()))
		}
	default:
		return errors.New(fmt.Sprintf("invalid payload version, "+
			"%s transaction", t.TxType().Name()))
	}

	if blockHeight < chainParams.DPoSConfiguration.NFTStartHeight {
		if t.PayloadVersion() == payload.CRInfoSchnorrVersion ||
			t.PayloadVersion() == payload.CRInfoMultiSignVersion {
			return errors.New(fmt.Sprintf("not support %s transaction "+
				"with payload version %d before NFTStartHeight",
				t.TxType().Name(), t.PayloadVersion()))
		}
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

	if !t.parameters.BlockChain.GetCRCommittee().IsInVotingPeriod(t.parameters.BlockHeight) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("should create tx during voting period")), true
	}

	if t.parameters.BlockChain.GetCRCommittee().ExistCandidateByNickname(info.NickName) {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("nick name %s already inuse", info.NickName)), true
	}

	cr := t.parameters.BlockChain.GetCRCommittee().GetCandidate(info.CID)
	if cr != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("cid %s already exist", info.CID)), true
	}

	// get CID program hash and check length of code
	var code []byte
	if t.payloadVersion == payload.CRInfoSchnorrVersion ||
		t.payloadVersion == payload.CRInfoMultiSignVersion {
		code = t.Programs()[0].Code
	} else {
		code = info.Code
	}
	ct, err := contract.CreateCRIDContractByCode(code)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}
	programHash := ct.ToProgramHash()

	// check CID
	if !info.CID.IsEqual(*programHash) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid cid address")), true
	}

	// check if program code conflict with producer public keys
	var pk []byte
	if contract.IsSchnorr(code) {
		pk = code[2:]
		// todo check
	} else if code[len(code)-1] == vm.CHECKSIG {
		pk = code[1 : len(code)-1]
	} else if code[len(code)-1] == vm.CHECKMULTISIG {
		pk = code
	} else {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("invalid code %s",
			common.BytesToHexString(code))), true
	}
	if t.parameters.BlockChain.GetState().ProducerExists(pk) {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("public key %s already inuse in producer list",
			common.BytesToHexString(pk))), true
	}
	if blockchain.DefaultLedger.Arbitrators.IsCRCArbitrator(pk) {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("public key %s already inuse in CRC list",
			common.BytesToHexString(pk))), true
	}

	if t.parameters.BlockHeight >= t.parameters.Config.CRConfiguration.RegisterCRByDIDHeight &&
		t.PayloadVersion() == payload.CRInfoDIDVersion {
		// get DID program hash

		programHash, err = getDIDFromCode(code)
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid info.Code")), true
		}
		// check DID
		if !info.DID.IsEqual(*programHash) {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid did address")), true
		}
	}

	// check code and signature
	if t.payloadVersion != payload.CRInfoSchnorrVersion &&
		t.payloadVersion != payload.CRInfoMultiSignVersion {
		if err := blockchain.CheckPayloadSignature(info, t.PayloadVersion()); err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
	}

	// check deposit coin
	var depositCount int
	for _, output := range t.Outputs() {
		if contract.GetPrefixType(output.ProgramHash) == contract.PrefixDeposit {
			depositCount++
			// get deposit program hash
			ct, err := contract.CreateDepositContractByCode(code)
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
	var didCode []byte
	if contract.IsSchnorr(code) {
		didCode = append(newCode[1:], common.DID)
	} else {
		didCode = append(newCode[:len(newCode)-1], common.DID)
	}

	if ct1, err := contract.CreateCRIDContractByCode(didCode); err != nil {
		return nil, err
	} else {
		return ct1.ToProgramHash(), nil
	}
}
