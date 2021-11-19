// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type SideChainPOWTransaction struct {
	BaseTransaction
}

func (t *SideChainPOWTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *SideChainPOWTransaction) SpecialCheck() (elaerr.ELAError, bool) {
	arbitrator := blockchain.DefaultLedger.Arbitrators.GetOnDutyCrossChainArbitrator()
	payloadSideChainPow, ok := t.Payload().(*payload.SideChainPow)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("side mining transaction has invalid payload")), true
	}

	if arbitrator == nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("there is no arbiter on duty")), true
	}

	publicKey, err := crypto.DecodePoint(arbitrator)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	buf := new(bytes.Buffer)
	err = payloadSideChainPow.Serialize(buf, payload.SideChainPowVersion)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	err = crypto.Verify(*publicKey, buf.Bytes()[0:68], payloadSideChainPow.Signature)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("Arbitrator is not matched. " + err.Error())), true
	}

	if t.IsNewSideChainPowTx() {
		return nil, true
	}

	return nil, false
}
