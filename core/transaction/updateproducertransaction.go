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

func (t *UpdateProducerTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.ProducerInfo:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *UpdateProducerTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *UpdateProducerTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
	info, ok := t.Payload().(*payload.ProducerInfo)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
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

	// check signature
	publicKey, err := crypto.DecodePoint(info.OwnerPublicKey)
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

	producer := t.parameters.BlockChain.GetState().GetProducer(info.OwnerPublicKey)
	if producer == nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("updating unknown producer")), true
	}
	stake := t.parameters.BlockChain.GetState()
	//if producer is already dposv2
	switch producer.Identity() {
	case state.DPoSV1:
		//if this producer want to be dposv2
		if info.StakeUntil != 0 {
			if t.parameters.BlockHeight+t.parameters.Config.DPoSV2MinVotesLockTime >= info.StakeUntil {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("v2 producer StakeUntil less than BlockHeight")), true
			}
		}
	case state.DPoSV2:
		if t.parameters.BlockHeight > producer.Info().StakeUntil && info.StakeUntil != producer.Info().StakeUntil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("producer already expired, can not update stakeuntil")), true
		}
		if info.StakeUntil < producer.Info().StakeUntil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("stake time is smaller than before")), true
		} else if info.StakeUntil > producer.Info().StakeUntil {
			//new StakeUntil must bigger than BlockHeight
			if t.parameters.BlockHeight >= info.StakeUntil {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("v2 producer StakeUntil less than BlockHeight")), true
			}
		}

		// height > stakeUntil: can't change stakeUntil anymore.
		if t.parameters.BlockHeight > producer.Info().StakeUntil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("DPoS 2.0 node has expired")), true
		}

	case state.DPoSV1V2:
		if t.parameters.BlockHeight > producer.Info().StakeUntil &&
			t.parameters.BlockHeight > stake.DPoSV2ActiveHeight &&
			info.StakeUntil != producer.Info().StakeUntil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("producer already expired and dposv2 already started, can not update stakeuntil ")), true
		}
		if info.StakeUntil < producer.Info().StakeUntil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("stake time is smaller than before")), true
		} else if info.StakeUntil > producer.Info().StakeUntil {
			//new StakeUntil must bigger than BlockHeight
			if t.parameters.BlockHeight >= info.StakeUntil {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("v2 producer StakeUntil less than BlockHeight")), true
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

	// check node public key duplication
	if bytes.Equal(info.NodePublicKey, producer.Info().NodePublicKey) {
		return nil, false
	}

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

		//check if NodePublicKey is others' ownerpublickey
		if t.parameters.BlockChain.GetState().ProducerOwnerPublicKeyExists(info.NodePublicKey) {
			return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("NodePublicKey can not be other producer's ownerPublicKey %s",
				hex.EncodeToString(info.NodePublicKey))), true
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

		for _, m := range t.parameters.BlockChain.GetCRCommittee().Members {
			if bytes.Equal(m.DPOSPublicKey, info.NodePublicKey) {
				return errors.New("node public key can't equal with current CR Node PK")
			}
			if bytes.Equal(m.Info.Code[1:len(m.Info.Code)-1], info.NodePublicKey) {
				return errors.New("node public key can't equal with current CR Owner PK")
			}
		}

		for _, m := range t.parameters.BlockChain.GetCRCommittee().NextMembers {
			if bytes.Equal(m.DPOSPublicKey, info.NodePublicKey) {
				return errors.New("node public key can't equal with next CR Node PK")
			}
			if bytes.Equal(m.Info.Code[1:len(m.Info.Code)-1], info.NodePublicKey) {
				return errors.New("node public key can't equal with current CR Owner PK")
			}
		}

		for _, p := range t.parameters.Config.CRCArbiters {
			if p == common.BytesToHexString(info.NodePublicKey) {
				return errors.New("node public key can't equal with CR Arbiters")
			}
		}
	}
	return nil
}
