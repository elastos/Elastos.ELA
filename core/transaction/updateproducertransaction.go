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
	"github.com/elastos/Elastos.ELA/core/contract"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/vm"
)

type UpdateProducerTransaction struct {
	BaseTransaction
}

func (t *UpdateProducerTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.DPoSV2StartHeight {
		producerPayload, ok := t.Payload().(*payload.ProducerInfo)
		if !ok {
			return errors.New("[HeightVersionCheck] invalid UpdateProducer  payload")
		}
		if producerPayload.StakeUntil != 0 {
			return errors.New("not support update DPoS " +
				"2.0 producer transaction before RevertToPOWStartHeight")
		}
	}
	if blockHeight < chainParams.SupportMultiCodeHeight {
		if t.PayloadVersion() == payload.ProducerInfoMultiVersion {
			return errors.New(fmt.Sprintf("not support %s transaction "+
				"with payload version %d before SupportMultiCodeHeight",
				t.TxType().Name(), t.PayloadVersion()))
		}
	}
	return nil
}

func (t *UpdateProducerTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.ProducerInfo:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *UpdateProducerTransaction) IsAllowedInPOWConsensus() bool {
	return true
}

func checkChangeStakeUntil(BlockHeight uint32, newinfo *payload.ProducerInfo, producer *state.Producer) *elaerr.SimpleErr {
	switch producer.State() {
	case state.Active, state.Inactive, state.Illegal:
		if newinfo.StakeUntil < producer.Info().StakeUntil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("stake time is smaller than before"))
		} else if newinfo.StakeUntil > producer.Info().StakeUntil {
			//new StakeUntil must bigger than BlockHeight
			if BlockHeight >= newinfo.StakeUntil {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("producer StakeUntil less than BlockHeight"))
			}
		}
	default:
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("Pending Canceled or Returned producer can  not update  StakeUntil "))

	}
	return nil
}

func (t *UpdateProducerTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
	info, ok := t.Payload().(*payload.ProducerInfo)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}
	multiSignOwner := false
	if t.PayloadVersion() == payload.ProducerInfoMultiVersion {
		multiSignOwner = true
	}
	if multiSignOwner && len(info.OwnerKey) == crypto.NegativeBigLength {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("ProducerInfoMultiVersion match multi code")), true
	}
	// check nick name
	if err := checkStringField(info.NickName, "NickName", false); err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	// check url
	if err := checkStringField(info.Url, "Url", true); err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	if err := t.additionalProducerInfoCheck(info); err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	if t.PayloadVersion() < payload.ProducerInfoSchnorrVersion {
		publicKey, err := crypto.DecodePoint(info.OwnerKey)
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid owner public key in payload")), true
		}
		signedBuf := new(bytes.Buffer)
		err = info.SerializeUnsigned(signedBuf, t.payloadVersion)
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
		err = crypto.Verify(*publicKey, signedBuf.Bytes(), info.Signature)
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid signature in payload")), true
		}
	} else if t.PayloadVersion() == payload.ProducerInfoSchnorrVersion {
		if len(t.Programs()) != 1 {
			return elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("ProducerInfoSchnorrVersion can only have one program code")), true
		}
		if !contract.IsSchnorr(t.Programs()[0].Code) {
			return elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("only schnorr code can use ProducerInfoSchnorrVersion")), true
		}
		pk := t.Programs()[0].Code[2:]
		if !bytes.Equal(pk, info.OwnerKey) {
			return elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("tx program pk must equal with OwnerKey")), true
		}
	} else if t.PayloadVersion() == payload.ProducerInfoMultiVersion {
		if !contract.IsMultiSig(t.Programs()[0].Code) {
			return elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("only multi sign code can use ProducerInfoMultiVersion")), true
		}
		//t.Programs()[0].Code equal info.OwnerKey
		if !bytes.Equal(t.Programs()[0].Code, info.OwnerKey) {
			return elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("ProducerInfoMultiVersion tx program pk must equal with OwnerKey")), true
		}
	}

	producer := t.parameters.BlockChain.GetState().GetProducer(info.OwnerKey)
	if producer == nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("updating unknown producer")), true
	}
	dposState := t.parameters.BlockChain.GetState()
	//if producer is already dposv2
	switch producer.Identity() {
	case state.DPoSV1:
		//if this producer want to be dposv2
		if info.StakeUntil != 0 {
			if producer.State() != state.Active {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("only active producer can update to DPoSV1V2")), true
			}
			if t.parameters.BlockHeight+t.parameters.Config.DPoSConfiguration.DPoSV2DepositCoinMinLockTime >= info.StakeUntil {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("v2 producer StakeUntil less than DPoSV2DepositCoinMinLockTime")), true
			}
		}
	case state.DPoSV2:
		// height > stakeUntil: can't change anything anymore.
		if t.parameters.BlockHeight > producer.Info().StakeUntil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("DPoS 2.0 node has expired")), true
		}

		if info.StakeUntil < producer.Info().StakeUntil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("stake time is smaller than before")), true
		}

	case state.DPoSV1V2:
		if t.parameters.BlockHeight >= dposState.GetDPoSV2ActiveHeight() {
			if t.parameters.BlockHeight > producer.Info().StakeUntil {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("producer already expired and dposv2 already started, can not update anything ")), true
			}
			if info.StakeUntil != producer.Info().StakeUntil { //change StakeUntil
				if err := checkChangeStakeUntil(t.parameters.BlockHeight, info, producer); err != nil {
					return err, true
				}
			}
		} else { //still in dpos1.0
			if info.StakeUntil != producer.Info().StakeUntil { //change StakeUntil
				if err := checkChangeStakeUntil(t.parameters.BlockHeight, info, producer); err != nil {
					return err, true
				}
			}
		}
	}

	// check nickname usage.
	if producer.Info().NickName != info.NickName &&
		t.parameters.BlockChain.GetState().NicknameExists(info.NickName) {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("nick name %s already exist", info.NickName)), true
	}

	// check if public keys conflict with cr program code
	nodeCode := append([]byte{byte(crypto.COMPRESSEDLEN)}, info.NodePublicKey...)
	nodeCode = append(nodeCode, vm.CHECKSIG)
	if t.parameters.BlockChain.GetCRCommittee().ExistCR(nodeCode) {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("node public key %s already exist in cr list",
			common.BytesToHexString(info.NodePublicKey))), true
	}

	// It is not necessary to check if OwnerKey is others' NodePublicKey since we can not change OwnerKey

	// check node public key duplication
	if bytes.Equal(info.NodePublicKey, producer.Info().NodePublicKey) {
		return nil, false
	}

	//if update producer tx change NodePublicKey
	if t.parameters.BlockChain.GetHeight() < t.parameters.Config.PublicDPOSHeight {
		if t.parameters.BlockChain.GetState().ProducerExists(info.NodePublicKey) {
			return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("producer %s already exist",
				hex.EncodeToString(info.NodePublicKey))), true
		}
	} else {
		// here only check  if NodePublicKey is others' NodePublicKey
		if t.parameters.BlockChain.GetState().ProducerOrCRNodePublicKeyExists(info.NodePublicKey) {
			return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("producer %s already exist",
				hex.EncodeToString(info.NodePublicKey))), true
		}

		if t.parameters.BlockHeight > t.parameters.Config.DPoSV2StartHeight {
			//check if NodePublicKey is others' ownerpublickey
			//NodePublicKey can not be change into  one producer's OwnerKey only if it is the same producer.
			producer := t.parameters.BlockChain.GetState().GetProducerByOwnerPublicKey(info.NodePublicKey)
			if producer != nil && !bytes.Equal(info.OwnerKey, producer.OwnerPublicKey()) {
				return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("NodePublicKey %s can not be other producer's ownerPublicKey ",
					hex.EncodeToString(info.NodePublicKey))), true
			}
		}
	}

	return nil, false
}

func (t *UpdateProducerTransaction) additionalProducerInfoCheck(info *payload.ProducerInfo) error {
	if t.parameters.BlockChain.GetHeight() >= t.parameters.Config.PublicDPOSHeight {
		_, err := crypto.DecodePoint(info.NodePublicKey)
		if err != nil {
			return errors.New("invalid node public key in payload")
		}

		for _, m := range t.parameters.BlockChain.GetCRCommittee().GetMembers() {
			if bytes.Equal(m.DPOSPublicKey, info.NodePublicKey) {
				return errors.New("node public key can't equal with current CR Node PK")
			}
			ownerKey := common.GetOwnerKey(m.Info.Code)
			if bytes.Equal(ownerKey, info.NodePublicKey) {
				return errors.New("node public key can't equal with current CR Owner PK")
			}
		}

		for _, m := range t.parameters.BlockChain.GetCRCommittee().GetNextMembers() {
			if bytes.Equal(m.DPOSPublicKey, info.NodePublicKey) {
				return errors.New("node public key can't equal with next CR Node PK")
			}
			ownerKey := common.GetOwnerKey(m.Info.Code)
			if bytes.Equal(ownerKey, info.NodePublicKey) {
				return errors.New("node public key can't equal with current CR Owner PK")
			}
		}

		for _, p := range t.parameters.Config.DPoSConfiguration.CRCArbiters {
			if p == common.BytesToHexString(info.NodePublicKey) {
				return errors.New("node public key can't equal with CR Arbiters")
			}
		}
	}
	return nil
}
