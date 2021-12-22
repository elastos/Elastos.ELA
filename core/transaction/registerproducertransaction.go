// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/vm"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type RegisterProducerTransaction struct {
	BaseTransaction
}

func (t *RegisterProducerTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.ProducerInfo:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *RegisterProducerTransaction) IsAllowedInPOWConsensus() bool {
	return true
}

func (t *RegisterProducerTransaction) SpecialContextCheck() (elaerr.ELAError, bool) {
	info, ok := t.Payload().(*payload.ProducerInfo)
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

	if t.parameters.BlockChain.GetHeight() < t.parameters.Config.PublicDPOSHeight {
		// check duplication of node.
		if t.parameters.BlockChain.GetState().ProducerExists(info.NodePublicKey) {
			return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("producer already registered")), true
		}

		// check duplication of owner.
		if t.parameters.BlockChain.GetState().ProducerExists(info.OwnerPublicKey) {
			return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("producer owner already registered")), true
		}
	} else if t.parameters.BlockChain.GetHeight() < t.parameters.Config.DposV2StartHeight &&
		t.PayloadVersion() == payload.ProducerInfoVersion {

		// check duplication of node.
		if t.parameters.BlockChain.GetState().ProducerNodePublicKeyExists(info.NodePublicKey) {
			return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("producer already registered")), true
		}

		// check duplication of owner.
		if t.parameters.BlockChain.GetState().ProducerOwnerPublicKeyExists(info.OwnerPublicKey) {
			return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("producer owner already registered")), true
		}
	}

	if t.PayloadVersion() == payload.ProducerInfoVersion {
		// check duplication of nickname.
		if t.parameters.BlockChain.GetState().NicknameExists(info.NickName) {
			return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("nick name %s already inuse", info.NickName)), true
		}

		// check if public keys conflict with cr program code
		ownerCode := append([]byte{byte(crypto.COMPRESSEDLEN)}, info.OwnerPublicKey...)
		ownerCode = append(ownerCode, vm.CHECKSIG)
		if t.parameters.BlockChain.GetCRCommittee().ExistCR(ownerCode) {
			return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("owner public key %s already exist in cr list",
				common.BytesToHexString(info.OwnerPublicKey))), true
		}
		nodeCode := append([]byte{byte(crypto.COMPRESSEDLEN)}, info.NodePublicKey...)
		nodeCode = append(nodeCode, vm.CHECKSIG)
		if t.parameters.BlockChain.GetCRCommittee().ExistCR(nodeCode) {
			return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("node public key %s already exist in cr list",
				common.BytesToHexString(info.NodePublicKey))), true
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

		// check deposit coin
		hash, err := contract.PublicKeyToDepositProgramHash(info.OwnerPublicKey)
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid public key")), true
		}
		var depositCount int
		for _, output := range t.Outputs() {
			if contract.GetPrefixType(output.ProgramHash) == contract.PrefixDeposit {
				depositCount++
				if !output.ProgramHash.IsEqual(*hash) {
					return elaerr.Simple(elaerr.ErrTxPayload, errors.New("deposit"+
						" address does not match the public key in payload")), true
				}
				if output.Value < crstate.MinDepositAmount {
					return elaerr.Simple(elaerr.ErrTxPayload, errors.New("producer deposit amount is insufficient")), true
				}
			}
		}
		if depositCount != 1 {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("there must be only one deposit address in outputs")), true
		}
	} else if t.PayloadVersion() == payload.ProducerInfoDposV2Version {
		if info.StakeUntil < t.parameters.Config.DposV2StartHeight {
			return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("stakeuntil must bigger than DposV2StartHeight")), true
		}

		// check duplication of node.
		nodeKeyExist := t.parameters.BlockChain.GetState().ProducerNodePublicKeyExists(info.NodePublicKey)

		// check duplication of owner.
		ownerKeyExist := t.parameters.BlockChain.GetState().ProducerOwnerPublicKeyExists(info.OwnerPublicKey)

		// check duplication of nickname.
		nickNameExist := t.parameters.BlockChain.GetState().NicknameExists(info.NickName)

		if nodeKeyExist != ownerKeyExist || ownerKeyExist != nickNameExist {
			return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("NodePublicKey %v OwnerPublicKey %v NickName %v", nodeKeyExist, ownerKeyExist, nickNameExist)), true
		}

		if !nodeKeyExist {
			// check if public keys conflict with cr program code
			ownerCode := append([]byte{byte(crypto.COMPRESSEDLEN)}, info.OwnerPublicKey...)
			ownerCode = append(ownerCode, vm.CHECKSIG)
			if t.parameters.BlockChain.GetCRCommittee().ExistCR(ownerCode) {
				return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("owner public key %s already exist in cr list",
					common.BytesToHexString(info.OwnerPublicKey))), true
			}
			nodeCode := append([]byte{byte(crypto.COMPRESSEDLEN)}, info.NodePublicKey...)
			nodeCode = append(nodeCode, vm.CHECKSIG)
			if t.parameters.BlockChain.GetCRCommittee().ExistCR(nodeCode) {
				return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("node public key %s already exist in cr list",
					common.BytesToHexString(info.NodePublicKey))), true
			}

			if err := t.additionalProducerInfoCheck(info); err != nil {
				return elaerr.Simple(elaerr.ErrTxPayload, err), true
			}
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

		// check deposit coin
		hash, err := contract.PublicKeyToDepositProgramHash(info.OwnerPublicKey)
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid public key")), true
		}
		var depositCount int
		for _, output := range t.Outputs() {
			if contract.GetPrefixType(output.ProgramHash) == contract.PrefixDeposit {
				depositCount++
				if !output.ProgramHash.IsEqual(*hash) {
					return elaerr.Simple(elaerr.ErrTxPayload, errors.New("deposit address does not match the public key in payload")), true
				}
				if output.Value < crstate.MinDepositAmount {
					return elaerr.Simple(elaerr.ErrTxPayload, errors.New("producer deposit amount is insufficient")), true
				}
			}
		}
		if depositCount != 1 {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("there must be only one deposit address in outputs")), true
		}
	}

	return nil, false
}

func checkStringField(rawStr string, field string, allowEmpty bool) error {
	if (!allowEmpty && len(rawStr) == 0) || len(rawStr) > blockchain.MaxStringLength {
		return fmt.Errorf("field %s has invalid string length", field)
	}

	return nil
}

func (t *RegisterProducerTransaction) additionalProducerInfoCheck(info *payload.ProducerInfo) error {
	if t.parameters.BlockChain.GetHeight() >= t.parameters.Config.PublicDPOSHeight {
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
