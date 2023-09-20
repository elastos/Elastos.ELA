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
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type CRCouncilMemberClaimNodeTransaction struct {
	BaseTransaction
}

func (t *CRCouncilMemberClaimNodeTransaction) IsAllowedInPOWConsensus() bool {
	return true
}

func (t *CRCouncilMemberClaimNodeTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.CRCouncilMemberClaimNode:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *CRCouncilMemberClaimNodeTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.CRConfiguration.CRClaimDPOSNodeStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before CRClaimDPOSNodeStartHeight", t.TxType().Name()))
	}
	return nil
}

func (t *CRCouncilMemberClaimNodeTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {
	manager, ok := t.Payload().(*payload.CRCouncilMemberClaimNode)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	if t.parameters.BlockHeight < t.parameters.Config.DPoSV2StartHeight &&
		!t.parameters.BlockChain.GetCRCommittee().IsInElectionPeriod() {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("CRCouncilMemberClaimNode must during election period")), true
	}
	switch t.payloadVersion {
	case payload.CurrentCRClaimDPoSNodeVersion, payload.NextCRClaimDPoSNodeVersion:
		crMember := t.parameters.BlockChain.GetCRCommittee().GetMember(manager.CRCouncilCommitteeDID)
		if crMember != nil && (!contract.IsStandard(crMember.Info.Code)) {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("CurrentCRClaimDPoSNodeVersion or NextCRClaimDPoSNodeVersion match standard code")), true
		}
	case payload.CurrentCRClaimDPoSNodeMultiSignVersion, payload.NextCRClaimDPoSNodeMultiSignVersion:
		if !contract.IsMultiSig(t.Programs()[0].Code) {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("CurrentCRClaimDPoSNodeMultiSignVersion or NextCRClaimDPoSNodeMultiSignVersion match multi code")), true
		}
		programDID, err1 := getDIDFromCode(t.Programs()[0].Code)
		if err1 != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("can not create did from program code")), true
		}
		if !programDID.IsEqual(manager.CRCouncilCommitteeDID) {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("program code not match with payload CRCouncilCommitteeDID")), true
		}
	}

	did := manager.CRCouncilCommitteeDID
	var crMember *crstate.CRMember
	comm := t.parameters.BlockChain.GetCRCommittee()
	if t.parameters.BlockHeight >= t.parameters.Config.DPoSV2StartHeight {
		switch t.payloadVersion {
		case payload.CurrentCRClaimDPoSNodeVersion, payload.CurrentCRClaimDPoSNodeMultiSignVersion:
			crMember = t.parameters.BlockChain.GetCRCommittee().GetMember(did)
			if ok := comm.PubKeyExistClaimedDPoSKeys(manager.NodePublicKey); ok {
				return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("producer already registered")), true
			}
			// check duplication of node.
			if t.parameters.BlockChain.GetState().ProducerAndCurrentCRNodePublicKeyExists(manager.NodePublicKey) {
				return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("producer already registered")), true
			}
		case payload.NextCRClaimDPoSNodeVersion, payload.NextCRClaimDPoSNodeMultiSignVersion:
			crMember = t.parameters.BlockChain.GetCRCommittee().GetNextMember(did)
			if ok := comm.PubKeyExistNextClaimedDPoSKey(manager.NodePublicKey); ok {
				return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("producer already registered")), true
			}
			// check duplication of node.
			if t.parameters.BlockChain.GetState().ProducerAndNextCRNodePublicKeyExists(manager.NodePublicKey) {
				return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("producer already registered")), true
			}
		}
	} else {
		crMember = t.parameters.BlockChain.GetCRCommittee().GetMember(did)
	}
	if crMember == nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("the originator must be members")), true
	}

	if crMember.MemberState != crstate.MemberElected && crMember.MemberState != crstate.MemberInactive {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("CR Council Member should be an elected or inactive CR members")), true
	}

	if len(crMember.DPOSPublicKey) != 0 {
		if bytes.Equal(crMember.DPOSPublicKey, manager.NodePublicKey) {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("NodePublicKey is the same as crMember.DPOSPublicKey")), true
		}
	}

	_, err := crypto.DecodePoint(manager.NodePublicKey)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid operating public key")), true
	}

	if t.payloadVersion < payload.CurrentCRClaimDPoSNodeMultiSignVersion {
		err = checkCRCouncilMemberClaimNodeSignature(manager, crMember.Info.Code)
		if err != nil {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("CR claim DPOS signature check failed")), true
		}
	}

	return nil, false
}

func checkCRCouncilMemberClaimNodeSignature(
	managementPayload *payload.CRCouncilMemberClaimNode, code []byte) error {
	signBuf := new(bytes.Buffer)
	managementPayload.SerializeUnsigned(signBuf, payload.CurrentCRClaimDPoSNodeVersion)
	if err := blockchain.CheckCRTransactionSignature(managementPayload.CRCouncilCommitteeSignature, code,
		signBuf.Bytes()); err != nil {
		return errors.New("CR signature check failed")
	}
	return nil
}
