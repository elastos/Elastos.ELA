// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
	elaerr "github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/vm"
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

	// check duplication of node.
	if t.payloadVersion != payload.ProducerInfoMultiVersion {
		if t.parameters.BlockChain.GetState().ProducerOrCRNodePublicKeyExists(info.NodePublicKey) {
			return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("Same NodePublicKey producer/cr already registered")), true
		}
		// check duplication of owner.
		if t.parameters.BlockChain.GetState().ProducerOwnerPublicKeyExists(info.OwnerPublicKey) {
			return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("producer owner already registered")), true
		}
	} else {
		//TODO multicode
	}

	if t.parameters.BlockHeight >= t.parameters.Config.DPoSV2StartHeight {
		if t.payloadVersion != payload.ProducerInfoMultiVersion {
			// OwnerPublicKey is  already other's NodePublicKey
			if t.parameters.BlockChain.GetState().ProducerOrCRNodePublicKeyExists(info.OwnerPublicKey) {
				return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("OwnerPublicKey is  already other's NodePublicKey")), true
			}

			// NodePublicKey is  already other's OwnerPublicKey
			if t.parameters.BlockChain.GetState().ProducerOwnerPublicKeyExists(info.NodePublicKey) {
				return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("NodePublicKey is  already other's OwnerPublicKey")), true
			}
		} else {
			//TODO multicode
		}
	}

	// check duplication of nickname.
	if t.parameters.BlockChain.GetState().NicknameExists(info.NickName) {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("nick name %s already inuse", info.NickName)), true
	}

	if t.payloadVersion != payload.ProducerInfoMultiVersion {
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
	} else {
		//TODO multicode
	}

	if err := t.additionalProducerInfoCheck(info); err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	// check signature
	publicKey, err := crypto.DecodePoint(info.OwnerPublicKey)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid owner public key in payload")), true
	}

	if t.PayloadVersion() != payload.ProducerInfoSchnorrVersion {
		signedBuf := new(bytes.Buffer)
		err = info.SerializeUnsigned(signedBuf, t.payloadVersion)
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, err), true
		}
		if t.PayloadVersion() != payload.ProducerInfoMultiVersion {
			err = crypto.Verify(*publicKey, signedBuf.Bytes(), info.Signature)
			if err != nil {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid signature in payload")), true
			}
		} else {
			//TODO multicode
		}
	} else {
		if len(t.Programs()) != 1 {
			return elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("ProducerInfoSchnorrVersion can only have one program code")), true
		}
		if !contract.IsSchnorr(t.Programs()[0].Code) {
			return elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("only schnorr code can use ProducerInfoSchnorrVersion")), true
		}
		pk := t.Programs()[0].Code[2:]
		if !bytes.Equal(pk, info.OwnerPublicKey) {
			return elaerr.Simple(elaerr.ErrTxPayload,
				errors.New("tx program pk must equal with OwnerPublicKey")), true
		}
	}

	height := t.parameters.BlockChain.GetHeight()
	state := t.parameters.BlockChain.GetState()
	if height < t.parameters.Config.DPoSV2StartHeight && t.payloadVersion == payload.ProducerInfoDposV2Version {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("can not register dposv2 before dposv2 start height")), true
	} else if height > state.DPoSV2ActiveHeight && t.payloadVersion == payload.ProducerInfoVersion {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("can not register dposv1 after dposv2 active height")), true
	} else if height < t.parameters.Config.SupportMultiCodeHeight && t.payloadVersion == payload.ProducerInfoMultiVersion {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("not support ProducerInfoMultiVersion when height is not reach  SupportMultiCodeHeight")), true
	}
	var hash *common.Uint168

	if t.PayloadVersion() != payload.ProducerInfoMultiVersion {
		hash, err = contract.PublicKeyToDepositProgramHash(info.OwnerPublicKey)
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid public key")), true
		}
	} else {
		//TODO multicode
	}

	if t.PayloadVersion() == payload.ProducerInfoVersion || t.PayloadVersion() == payload.ProducerInfoMultiVersion {
		// check deposit coin
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

	} else if t.PayloadVersion() == payload.ProducerInfoDposV2Version || t.PayloadVersion() == payload.ProducerInfoSchnorrVersion {
		if t.parameters.BlockHeight+t.parameters.Config.DPoSConfiguration.DPoSV2DepositCoinMinLockTime >= info.StakeUntil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("v2 producer StakeUntil less than DPoSV2DepositCoinMinLockTime")), true
		}
		//if info.StakeUntil > t.parameters.BlockHeight+t.parameters.Config.DPoSV2MaxVotesLockTime {
		//	return elaerr.Simple(elaerr.ErrTxPayload, errors.New("v2 producer StakeUntil bigger than DPoSV2MaxVotesLockTime")), true
		//}
		var depositCount int
		for _, output := range t.Outputs() {
			if contract.GetPrefixType(output.ProgramHash) == contract.PrefixDeposit {
				depositCount++
				if !output.ProgramHash.IsEqual(*hash) {
					return elaerr.Simple(elaerr.ErrTxPayload, errors.New("deposit address does not match the public key in payload")), true
				}
				if output.Value < crstate.MinDPoSV2DepositAmount {
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
		if t.PayloadVersion() != payload.ProducerInfoMultiVersion {
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
		} else {
			//TODO multicode
		}
	}
	return nil
}
