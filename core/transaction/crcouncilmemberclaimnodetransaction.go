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
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type CRCouncilMemberClaimNodeTransaction struct {
	BaseTransaction
}

func (t *CRCouncilMemberClaimNodeTransaction) RegisterFunctions() {
	t.DefaultChecker.CheckTransactionSize = t.checkTransactionSize
	t.DefaultChecker.CheckTransactionInput = t.checkTransactionInput
	t.DefaultChecker.CheckTransactionOutput = t.checkTransactionOutput
	t.DefaultChecker.CheckTransactionPayload = t.CheckTransactionPayload
	t.DefaultChecker.HeightVersionCheck = t.heightVersionCheck
	t.DefaultChecker.IsAllowedInPOWConsensus = t.IsAllowedInPOWConsensus
	t.DefaultChecker.SpecialContextCheck = t.SpecialContextCheck
	t.DefaultChecker.CheckAttributeProgram = t.checkAttributeProgram
}

func (t *CRCouncilMemberClaimNodeTransaction) IsAllowedInPOWConsensus(params *TransactionParameters, references map[*common2.Input]common2.Output) bool {
	return true
}

func (t *CRCouncilMemberClaimNodeTransaction) CheckTransactionPayload(params *TransactionParameters) error {
	switch t.Payload().(type) {
	case *payload.CRCouncilMemberClaimNode:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *CRCouncilMemberClaimNodeTransaction) HeightVersionCheck(params *TransactionParameters) error {
	txn := params.Transaction
	blockHeight := params.BlockHeight
	chainParams := params.Config

	if blockHeight < chainParams.CRClaimDPOSNodeStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before CRClaimDPOSNodeStartHeight", txn.TxType().Name()))
	}
	return nil
}

func (t *CRCouncilMemberClaimNodeTransaction) SpecialContextCheck(params *TransactionParameters, references map[*common2.Input]common2.Output) (result elaerr.ELAError, end bool) {
	manager, ok := t.Payload().(*payload.CRCouncilMemberClaimNode)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}

	if !params.BlockChain.GetCRCommittee().IsInElectionPeriod() {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("CRCouncilMemberClaimNode must during election period")), true

	}
	did := manager.CRCouncilCommitteeDID
	crMember := params.BlockChain.GetCRCommittee().GetMember(did)
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

	// check duplication of node.
	if params.BlockChain.GetState().ProducerNodePublicKeyExists(manager.NodePublicKey) {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("producer already registered")), true
	}

	err = checkCRCouncilMemberClaimNodeSignature(manager, crMember.Info.Code)
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("CR claim DPOS signature check failed")), true
	}

	return nil, false
}

func checkCRCouncilMemberClaimNodeSignature(
	managementPayload *payload.CRCouncilMemberClaimNode, code []byte) error {
	signBuf := new(bytes.Buffer)
	managementPayload.SerializeUnsigned(signBuf, payload.CRManagementVersion)
	if err := blockchain.CheckCRTransactionSignature(managementPayload.CRCouncilCommitteeSignature, code,
		signBuf.Bytes()); err != nil {
		return errors.New("CR signature check failed")
	}
	return nil
}