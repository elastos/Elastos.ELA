// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type UpdateCRTransaction struct {
	BaseTransaction
}

func (t *UpdateCRTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.CRInfo:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *UpdateCRTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *UpdateCRTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
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

	var code []byte
	if t.payloadVersion == payload.CRInfoSchnorrVersion ||
		t.payloadVersion == payload.CRInfoMultiSignVersion {
		code = t.Programs()[0].Code
	} else {
		code = info.Code
	}

	// get CID program hash and check length of code
	ct, err := contract.CreateCRIDContractByCode(code)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}
	programHash := ct.ToProgramHash()

	// check CID
	if !info.CID.IsEqual(*programHash) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid cid address")), true
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

	if !t.parameters.BlockChain.GetCRCommittee().IsInVotingPeriod(t.parameters.BlockHeight) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("should create tx during voting period")), true
	}

	cr := t.parameters.BlockChain.GetCRCommittee().GetCandidate(info.CID)
	if cr == nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("updating unknown CR")), true
	}
	if cr.State != crstate.Pending && cr.State != crstate.Active {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("updating canceled or returned CR")), true
	}

	// check nickname usage.
	if cr.Info.NickName != info.NickName &&
		t.parameters.BlockChain.GetCRCommittee().ExistCandidateByNickname(info.NickName) {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("nick name %s already exist", info.NickName)), true
	}

	// check code and signature
	if t.payloadVersion != payload.CRInfoSchnorrVersion &&
		t.payloadVersion != payload.CRInfoMultiSignVersion {
		if err := blockchain.CheckPayloadSignature(info, t.PayloadVersion()); err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
	} else {
		c, exist := t.parameters.BlockChain.GetCRCommittee().GetState().GetCodeByCid(info.CID)
		if !exist {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("can not find code from cid")), true
		}
		cf, _ := hex.DecodeString(c)
		if !bytes.Equal(cf, t.Programs()[0].Code) {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid transaction code")), true
		}
	}

	return nil, false
}
