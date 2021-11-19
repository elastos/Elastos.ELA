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
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	elaerr "github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/vm"
)

type UpdateProducerTransaction struct {
	BaseTransaction
}

func (t *UpdateProducerTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *UpdateProducerTransaction) SpecialCheck() (elaerr.ELAError, bool) {
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
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid owner public key in payload")),true
	}
	signedBuf := new(bytes.Buffer)
	err = info.SerializeUnsigned(signedBuf, payload.ProducerInfoVersion)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err),true
	}
	err = crypto.Verify(*publicKey, signedBuf.Bytes(), info.Signature)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid signature in payload")),true
	}

	producer := t.contextParameters.BlockChain.GetState().GetProducer(info.OwnerPublicKey)
	if producer == nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("updating unknown producer")),true
	}

	// check nickname usage.
	if producer.Info().NickName != info.NickName &&
		t.contextParameters.BlockChain.GetState().NicknameExists(info.NickName) {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("nick name %s already exist", info.NickName)),true
	}

	// check if public keys conflict with cr program code
	nodeCode := append([]byte{byte(crypto.COMPRESSEDLEN)}, info.NodePublicKey...)
	nodeCode = append(nodeCode, vm.CHECKSIG)
	if t.contextParameters.BlockChain.GetCRCommittee().ExistCR(nodeCode) {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("node public key %s already exist in cr list",
			common.BytesToHexString(info.NodePublicKey))),true
	}

	// check node public key duplication
	if bytes.Equal(info.NodePublicKey, producer.Info().NodePublicKey) {
		return nil, true
	}

	if t.contextParameters.BlockChain.GetHeight() < t.contextParameters.Config.PublicDPOSHeight {
		if t.contextParameters.BlockChain.GetState().ProducerExists(info.NodePublicKey) {
			return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("producer %s already exist",
				hex.EncodeToString(info.NodePublicKey))),true
		}
	} else {
		if t.contextParameters.BlockChain.GetState().ProducerNodePublicKeyExists(info.NodePublicKey) {
			return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("producer %s already exist",
				hex.EncodeToString(info.NodePublicKey))),true
		}
	}

	return nil, false
}

func (t *UpdateProducerTransaction) additionalProducerInfoCheck(info *payload.ProducerInfo) error {
	if t.contextParameters.BlockChain.GetHeight() >= t.contextParameters.Config.PublicDPOSHeight {
		_, err := crypto.DecodePoint(info.NodePublicKey)
		if err != nil {
			return errors.New("invalid node public key in payload")
		}

		if blockchain.DefaultLedger.Arbitrators.IsCRCArbitrator(info.NodePublicKey) {
			return errors.New("node public key can't equal with CRC")
		}

		if blockchain.DefaultLedger.Arbitrators.IsCRCArbitrator(info.OwnerPublicKey) {
			return errors.New("owner public key can't equal with CRC")
		}
	}
	return nil
}