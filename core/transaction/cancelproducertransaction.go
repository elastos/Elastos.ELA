// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"

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
	publicKey, err := crypto.DecodePoint(processProducer.OwnerPublicKey)
	if err != nil {
		return nil, errors.New("invalid public key in payload")
	}

	if t.PayloadVersion() != payload.ProcessProducerSchnorrVersion {
		signedBuf := new(bytes.Buffer)
		err = processProducer.SerializeUnsigned(signedBuf, t.PayloadVersion())
		if err != nil {
			return nil, err
		}
		err = crypto.Verify(*publicKey, signedBuf.Bytes(), processProducer.Signature)
		if err != nil {
			return nil, errors.New("invalid signature in payload")
		}
	} else {
		if !contract.IsSchnorr(t.Programs()[0].Code) {
			return nil, errors.New("only schnorr code can use ProcessProducerSchnorrVersion")
		}
		pk := t.Programs()[0].Code[2:]
		if !bytes.Equal(pk, processProducer.OwnerPublicKey) {
			return nil, errors.New("tx program pk must equal with processProducer OwnerPublicKey ")
		}
	}

	producer := t.parameters.BlockChain.GetState().GetProducer(processProducer.OwnerPublicKey)
	if producer == nil || !bytes.Equal(producer.OwnerPublicKey(),
		processProducer.OwnerPublicKey) {
		return nil, errors.New("getting unknown producer")
	}
	return producer, nil
}
