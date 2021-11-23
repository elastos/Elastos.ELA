// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"

	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type CancelProducerTransaction struct {
	BaseTransaction
}

func (t *CancelProducerTransaction) RegisterFunctions() {
	t.DefaultChecker.CheckTransactionSize = t.checkTransactionSize
	t.DefaultChecker.CheckTransactionInput = t.checkTransactionInput
	t.DefaultChecker.CheckTransactionOutput = t.checkTransactionOutput
	t.DefaultChecker.CheckTransactionPayload = t.CheckTransactionPayload
	t.DefaultChecker.HeightVersionCheck = t.heightVersionCheck
	t.DefaultChecker.IsAllowedInPOWConsensus = t.IsAllowedInPOWConsensus
	t.DefaultChecker.SpecialContextCheck = t.SpecialContextCheck
	t.DefaultChecker.CheckAttributeProgram = t.checkAttributeProgram
}

func (t *CancelProducerTransaction) CheckTransactionPayload(params *TransactionParameters) error {
	switch t.Payload().(type) {
	case *payload.ProcessProducer:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *CancelProducerTransaction) IsAllowedInPOWConsensus(params *TransactionParameters, references map[*common2.Input]common2.Output) bool {
	return false
}

func (t *CancelProducerTransaction) SpecialContextCheck(params *TransactionParameters, references map[*common2.Input]common2.Output) (elaerr.ELAError, bool) {
	producer, err := t.checkProcessProducer(params, t)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
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
	signedBuf := new(bytes.Buffer)
	err = processProducer.SerializeUnsigned(signedBuf, payload.ProcessProducerVersion)
	if err != nil {
		return nil, err
	}
	err = crypto.Verify(*publicKey, signedBuf.Bytes(), processProducer.Signature)
	if err != nil {
		return nil, errors.New("invalid signature in payload")
	}

	producer := params.BlockChain.GetState().GetProducer(processProducer.OwnerPublicKey)
	if producer == nil || !bytes.Equal(producer.OwnerPublicKey(),
		processProducer.OwnerPublicKey) {
		return nil, errors.New("getting unknown producer")
	}
	return producer, nil
}
