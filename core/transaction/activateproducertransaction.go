// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type ActivateProducerTransaction struct {
	BaseTransaction
}

func (t *ActivateProducerTransaction) CheckTransactionInput() error {
	if len(t.sanityParameters.Transaction.Inputs()) != 0 {
		return errors.New("no cost transactions must has no input")
	}
	return nil
}

func (t *ActivateProducerTransaction) IsAllowedInPOWConsensus() bool {
	return true
}

func (t *ActivateProducerTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {

	activateProducer, ok := t.Payload().(*payload.ActivateProducer)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	err := t.checkActivateProducerSignature(activateProducer)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	if t.contextParameters.BlockChain.GetCRCommittee().IsInElectionPeriod() {
		crMember := t.contextParameters.BlockChain.GetCRCommittee().GetMemberByNodePublicKey(activateProducer.NodePublicKey)
		if crMember != nil && (crMember.MemberState == crstate.MemberInactive ||
			crMember.MemberState == crstate.MemberIllegal) {
			if t.contextParameters.BlockHeight < t.contextParameters.Config.EnableActivateIllegalHeight &&
				crMember.MemberState == crstate.MemberIllegal {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New(
					"activate MemberIllegal CR is not allowed before EnableActivateIllegalHeight")), true
			}
			if t.contextParameters.BlockChain.GetCRCommittee().GetAvailableDepositAmount(crMember.Info.CID) < 0 {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("balance of CR is not enough ")), true
			}
			return nil, true
		}
	}

	producer := t.contextParameters.BlockChain.GetState().GetProducer(activateProducer.NodePublicKey)
	if producer == nil || !bytes.Equal(producer.NodePublicKey(),
		activateProducer.NodePublicKey) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("getting unknown producer")), true
	}

	if t.contextParameters.BlockHeight < t.contextParameters.Config.EnableActivateIllegalHeight {
		if producer.State() != state.Inactive {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("can not activate this producer")), true
		}
	} else {
		if t.contextParameters.BlockHeight < t.contextParameters.Config.ChangeCommitteeNewCRHeight {
			if producer.State() != state.Active &&
				producer.State() != state.Inactive &&
				producer.State() != state.Illegal {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("can not activate this producer")), true
			}
		} else {
			if producer.State() != state.Inactive &&
				producer.State() != state.Illegal {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("can not activate this producer")), true
			}
		}

	}

	if t.contextParameters.BlockHeight > producer.ActivateRequestHeight() &&
		t.contextParameters.BlockHeight-producer.ActivateRequestHeight() <= state.ActivateDuration {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("can only activate once during inactive state")), true
	}

	depositAmount := common.Fixed64(0)
	if t.contextParameters.BlockHeight < t.contextParameters.Config.CRVotingStartHeight {
		programHash, err := contract.PublicKeyToDepositProgramHash(
			producer.OwnerPublicKey())
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}

		utxos, err := t.contextParameters.BlockChain.GetDB().GetFFLDB().GetUTXO(programHash)
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}

		for _, u := range utxos {
			depositAmount += u.Value
		}
	} else {
		depositAmount = producer.TotalAmount()
	}

	if depositAmount-producer.Penalty() < crstate.MinDepositAmount {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("insufficient deposit amount")), true
	}

	return nil, true
}

func (t *ActivateProducerTransaction) checkActivateProducerSignature(
	activateProducer *payload.ActivateProducer) error {
	// check signature
	publicKey, err := crypto.DecodePoint(activateProducer.NodePublicKey)
	if err != nil {
		return errors.New("invalid public key in payload")
	}
	signedBuf := new(bytes.Buffer)
	err = activateProducer.SerializeUnsigned(signedBuf, payload.ActivateProducerVersion)
	if err != nil {
		return err
	}
	err = crypto.Verify(*publicKey, signedBuf.Bytes(), activateProducer.Signature)
	if err != nil {
		return errors.New("invalid signature in payload")
	}
	return nil
}
