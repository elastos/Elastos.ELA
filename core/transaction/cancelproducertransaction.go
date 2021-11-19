// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"

	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type CancelProducerTransaction struct {
	BaseTransaction
}

func (t *CancelProducerTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *CancelProducerTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
	producer, err := t.checkProcessProducer(t)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	if producer.State() == state.Illegal ||
		producer.State() == state.Canceled {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("can not cancel this producer")), true
	}

	return nil, false
}

func (t *CancelProducerTransaction) checkProcessProducer(txn interfaces.Transaction) (*state.Producer, error) {
	processProducer, ok := txn.Payload().(*payload.ProcessProducer)
	if !ok {
		return nil, errors.New("invalid payload")
	}

	// check signature
	publicKey, err := crypto.DecodePoint(processProducer.OwnerPublicKey)
	if err != nil {
		return nil, errors.New("invalid public key in payload")
	}
	signedBuf := new(bytes.Buffer)
	err = processProducer.SerializeUnsigned(signedBuf, payload.ProcessProducerVersion)
	if err != nil {
		return nil, err
	}
	err = crypto.Verify(*publicKey, signedBuf.Bytes(), processProducer.Signature)
	if err != nil {
		return nil, errors.New("invalid signature in payload")
	}

	producer := t.contextParameters.BlockChain.GetState().GetProducer(processProducer.OwnerPublicKey)
	if producer == nil || !bytes.Equal(producer.OwnerPublicKey(),
		processProducer.OwnerPublicKey) {
		return nil, errors.New("getting unknown producer")
	}
	return producer, nil
}
