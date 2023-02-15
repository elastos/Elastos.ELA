// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type CancelProducerTransaction struct {
	BaseTransaction
}

func (t *CancelProducerTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.ProcessProducer:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *CancelProducerTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *CancelProducerTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.SupportMultiCodeHeight {
		if t.PayloadVersion() == payload.ProcessMultiCodeVersion {
			return errors.New(fmt.Sprintf("not support %s transaction "+
				"with payload version %d before SupportMultiCodeHeight",
				t.TxType().Name(), t.PayloadVersion()))
		}
	}

	return nil
}

func (t *CancelProducerTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
	producer, err := t.checkProcessProducer(t.parameters, t)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	height := t.parameters.BlockHeight
	switch producer.Identity() {
	case state.DPoSV1:
	case state.DPoSV2:
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("can not cancel DPoS V2 producer")), true
	case state.DPoSV1V2:
		if height <= producer.Info().StakeUntil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("can not cancel DPoS V1&V2 producer")), true
		}
	}

	if producer.State() == state.Illegal ||
		producer.State() == state.Canceled {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("can not cancel this producer")), true
	}

	return nil, false
}

func (t *CancelProducerTransaction) checkProcessProducer(params *TransactionParameters, txn interfaces.Transaction) (*state.Producer, error) {
	processProducer, ok := txn.Payload().(*payload.ProcessProducer)
	if !ok {
		return nil, errors.New("invalid payload")
	}

	// check signature
	if t.PayloadVersion() == payload.ProcessProducerVersion {
		publicKey, err := crypto.DecodePoint(processProducer.OwnerKey)
		if err != nil {
			return nil, errors.New("invalid public key in payload")
		}
		signedBuf := new(bytes.Buffer)
		err = processProducer.SerializeUnsigned(signedBuf, t.PayloadVersion())
		if err != nil {
			return nil, err
		}
		err = crypto.Verify(*publicKey, signedBuf.Bytes(), processProducer.Signature)
		if err != nil {
			return nil, errors.New("invalid signature in payload")
		}
	} else if t.PayloadVersion() == payload.ProcessProducerSchnorrVersion {
		if !contract.IsSchnorr(t.Programs()[0].Code) {
			return nil, errors.New("only schnorr code can use ProcessProducerSchnorrVersion")
		}
		pk := t.Programs()[0].Code[2:]
		//todo OwnerKey should be public key
		if !bytes.Equal(pk, processProducer.OwnerKey) {
			return nil, errors.New("tx program pk must equal with processProducer OwnerKey ")
		}
	} else if t.PayloadVersion() == payload.ProcessMultiCodeVersion {
		if !contract.IsMultiSig(t.Programs()[0].Code) {
			return nil, elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("only multi sign code can use ProcessMultiCodeVersion"))
		}
	}

	producer := t.parameters.BlockChain.GetState().GetProducer(processProducer.OwnerKey)
	if producer == nil || !bytes.Equal(producer.OwnerPublicKey(),
		processProducer.OwnerKey) {
		return nil, errors.New("getting unknown producer")
	}
	return producer, nil
}
