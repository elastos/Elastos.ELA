// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package mempool

import (
	"encoding/hex"
	"fmt"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/transactions"
	"strconv"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/vm"
)

// hashes related functions
func hashCRCProposalDraftHash(tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"CRC proposal payload cast failed, tx:%s", tx.Hash())
	}
	return p.DraftHash, nil
}

func hashCRCProposalDID(tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"CRC proposal payload cast failed, tx:%s", tx.Hash())
	}
	return p.CRCouncilMemberDID, nil
}

func strArrayCRCProposalCustomID(tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"CRC proposal payload cast failed, tx:%s", tx.Hash())
	}
	if p.ProposalType != payload.ReceiveCustomID {
		return nil, nil
	}
	return p.ReceivedCustomIDList, nil
}

func hashChangeProposalOwnerTargetProposalHash(tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"CRC proposal payload cast failed, tx:%s", tx.Hash())
	}
	if p.ProposalType == payload.ChangeProposalOwner {
		return p.TargetProposalHash, nil
	}
	return nil, nil
}

func hashCloseProposalTargetProposalHash(tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"CRC proposal payload cast failed, tx:%s", tx.Hash())
	}
	if p.ProposalType == payload.CloseProposal {
		return p.TargetProposalHash, nil
	}
	return nil, nil
}

func hashCRCProposalSecretaryGeneralDID(tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"CRC proposal payload cast failed, tx:%s", tx.Hash())
	}
	if p.ProposalType == payload.SecretaryGeneral {
		return p.SecretaryGeneralDID, nil
	}
	return nil, nil
}

func strChangeCustomIDFee(tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"CRC proposal payload cast failed, tx:%s", tx.Hash())
	}
	if p.ProposalType == payload.ChangeCustomIDFee {
		return "Change the fee of custom ID", nil
	}
	return nil, nil
}

func strReserveCustomID(tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"CRC proposal payload cast failed, tx:%s", tx.Hash())
	}
	if p.ProposalType == payload.ReserveCustomID {
		return "Reserve custom ID", nil
	}
	return nil, nil
}

func hashCRCProposalRegisterSideChainName(
	tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"crcProposal payload cast failed, tx:%s", tx.Hash())
	}
	if p.ProposalType == payload.RegisterSideChain {
		return p.SideChainName, nil
	}

	return nil, nil
}

func hashCRCProposalRegisterSideChainMagicNumber(
	tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"crcProposal payload cast failed, tx:%s", tx.Hash())
	}
	if p.ProposalType == payload.RegisterSideChain {
		return strconv.Itoa(int(p.MagicNumber)), nil
	}
	return nil, nil
}

func hashCRCProposalRegisterSideChainGenesisHash(
	tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"crcProposal payload cast failed, tx:%s", tx.Hash())
	}
	if p.ProposalType == payload.RegisterSideChain {
		return p.GenesisHash, nil
	}
	return nil, nil
}

func hashCRCProposalWithdrawProposalHash(
	tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.CRCProposalWithdraw)
	if !ok {
		return nil, fmt.Errorf(
			"crcProposalWithdraw  payload cast failed, tx:%s", tx.Hash())
	}
	return p.ProposalHash, nil
}

func hashCRCProposalTrackingProposalHash(
	tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.CRCProposalTracking)
	if !ok {
		return nil, fmt.Errorf(
			"crcProposalTracking  payload cast failed, tx:%s", tx.Hash())
	}
	return p.ProposalHash, nil
}

func hashSpecialTxHash(tx *transactions.BaseTransaction) (interface{}, error) {
	illegalData, ok := tx.Payload.(payload.DPOSIllegalData)
	if !ok {
		return nil, fmt.Errorf(
			"special tx payload cast failed, tx:%s", tx.Hash())
	}
	return illegalData.Hash(), nil
}

func hashNextTurnDPOSInfoTxPayloadHash(tx *transactions.BaseTransaction) (interface{}, error) {
	payload, ok := tx.Payload.(*payload.NextTurnDPOSInfo)
	if !ok {
		return nil, fmt.Errorf(
			"NextTurnDPOSInfo tx payload cast failed, tx:%s", tx.Hash())
	}
	return payload.Hash(), nil
}

func hashCustomIDProposalResultTxPayloadHash(tx *transactions.BaseTransaction) (interface{}, error) {
	_, ok := tx.Payload.(*payload.RecordProposalResult)
	if !ok {
		return nil, fmt.Errorf(
			"custom ID proposal result tx payload cast failed, tx:%s", tx.Hash())
	}
	return "customIDProposalResult", nil
}

// strings related functions
func strCancelProducerOwnerPublicKey(tx *transactions.BaseTransaction) (interface{},
	error) {
	p, ok := tx.Payload.(*payload.ProcessProducer)
	if !ok {
		err := fmt.Errorf(
			"cancel producer payload cast failed, tx:%s", tx.Hash())
		return nil, errors.Simple(errors.ErrTxPoolFailure, err)
	}
	return common.BytesToHexString(p.OwnerPublicKey), nil
}

func strProducerInfoOwnerPublicKey(tx *transactions.BaseTransaction) (interface{}, error) {
	p, err := comGetProducerInfo(tx)
	if err != nil {
		return nil, err
	}
	return common.BytesToHexString(p.OwnerPublicKey), nil
}

func strProducerInfoNodePublicKey(tx *transactions.BaseTransaction) (interface{}, error) {
	p, err := comGetProducerInfo(tx)
	if err != nil {
		return nil, err
	}
	return common.BytesToHexString(p.NodePublicKey), nil
}

func strCRManagementPublicKey(tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.CRCouncilMemberClaimNode)
	if !ok {
		return nil, fmt.Errorf(
			"cr dpos management payload cast failed, tx:%s", tx.Hash())
	}
	return common.BytesToHexString(p.NodePublicKey), nil
}

func strCRManagementDID(tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.CRCouncilMemberClaimNode)
	if !ok {
		return nil, fmt.Errorf(
			"cr dpos management payload cast failed, tx:%s", tx.Hash())
	}
	return p.CRCouncilCommitteeDID, nil
}

func strProducerInfoNickname(tx *transactions.BaseTransaction) (interface{}, error) {
	p, err := comGetProducerInfo(tx)
	if err != nil {
		return nil, err
	}
	return p.NickName, nil
}

func strRegisterCRPublicKey(tx *transactions.BaseTransaction) (interface{}, error) {
	p, err := comGetCRInfo(tx)
	if err != nil {
		return nil, err
	}

	signType, err := crypto.GetScriptType(p.Code)
	if err != nil {
		return nil, err
	}

	if signType == vm.CHECKSIG {
		return hex.EncodeToString(p.Code[1 : len(p.Code)-1]), nil
	} else {
		return nil, fmt.Errorf("unsupported sign script type: %d", signType)
	}
}

func strActivateProducerNodePublicKey(
	tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.ActivateProducer)
	if !ok {
		return nil, fmt.Errorf(
			"activate producer payload cast failed, tx:%s", tx.Hash())
	}
	return common.BytesToHexString(p.NodePublicKey), nil
}

func strCRInfoNickname(tx *transactions.BaseTransaction) (interface{}, error) {
	p, err := comGetCRInfo(tx)
	if err != nil {
		return nil, err
	}
	return p.NickName, nil
}

func strTxProgramCode(tx *transactions.BaseTransaction) (interface{}, error) {
	return common.BytesToHexString(tx.Programs[0].Code), nil
}

func strProposalReviewKey(tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.CRCProposalReview)
	if !ok {
		return nil, fmt.Errorf(
			"crcProposalReview  payload cast failed, tx:%s",
			tx.Hash())
	}
	return p.DID.String() + p.ProposalHash.String(), nil
}

func strCRCAppropriation(*transactions.BaseTransaction) (interface{}, error) {
	// const string to ensure only one tx added to the tx pool
	return "CRC Appropriation", nil
}

func strSecretaryGeneral(tx *transactions.BaseTransaction) (interface{}, error) {
	// const string to ensure only one tx added to the tx pool
	p, ok := tx.Payload.(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"CRC proposal payload cast failed, tx:%s", tx.Hash())
	}
	if p.ProposalType == payload.SecretaryGeneral {
		return "Secretary General", nil
	}
	return nil, nil
}

func hashArrayCRCProposalRealWithdrawTransactionHashes(
	tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.CRCProposalRealWithdraw)
	if !ok {
		return nil, fmt.Errorf(
			"real proposal withdraw transaction cast failed, tx: %s",
			tx.Hash())
	}

	return p.WithdrawTransactionHashes, nil
}

func hashRevertToDPOS(tx *transactions.BaseTransaction) (interface{}, error) {
	_, ok := tx.Payload.(*payload.RevertToDPOS)
	if !ok {
		return nil, fmt.Errorf(
			"RevertToDPOS transaction cast failed, tx: %s",
			tx.Hash())
	}

	return "RevertToDPOS", nil
}

// program hashes related functions
func addrCRInfoCRCID(tx *transactions.BaseTransaction) (interface{}, error) {
	p, err := comGetCRInfo(tx)
	if err != nil {
		return nil, err
	}
	return p.CID, nil
}

func addrUnregisterCRCID(tx *transactions.BaseTransaction) (interface{}, error) {
	p, ok := tx.Payload.(*payload.UnregisterCR)
	if !ok {
		return nil, fmt.Errorf(
			"unregisterCR CR payload cast failed, tx: %s", tx.Hash())
	}
	return p.CID, nil
}

// hash array related functions
func hashArraySidechainTransactionHashes(
	tx *transactions.BaseTransaction) (interface{}, error) {
	if tx.PayloadVersion == payload.WithdrawFromSideChainVersion {
		p, ok := tx.Payload.(*payload.WithdrawFromSideChain)
		if !ok {
			return nil, fmt.Errorf(
				"withdraw from sidechain payload cast failed, tx: %s",
				tx.Hash())
		}

		array := make([]common.Uint256, 0, len(p.SideChainTransactionHashes))
		for _, v := range p.SideChainTransactionHashes {
			array = append(array, v)
		}
		return array, nil
	} else if tx.PayloadVersion == payload.WithdrawFromSideChainVersionV1 {
		array := make([]common.Uint256, 0)
		for _, output := range tx.Outputs {
			if output.Type != common2.OTWithdrawFromSideChain {
				continue
			}
			witPayload, ok := output.Payload.(*outputpayload.Withdraw)
			if !ok {
				continue
			}
			array = append(array, witPayload.SideChainTransactionHash)
		}
		return array, nil
	}

	p, ok := tx.Payload.(*payload.WithdrawFromSideChain)
	if !ok {
		return nil, fmt.Errorf(
			"withdraw from sidechain payload cast failed, tx: %s",
			tx.Hash())
	}

	array := make([]common.Uint256, 0, len(p.SideChainTransactionHashes))
	for _, v := range p.SideChainTransactionHashes {
		array = append(array, v)
	}
	return array, nil
}

// hash array related functions
func hashArraySidechainReturnDepositTransactionHashes(
	tx *transactions.BaseTransaction) (interface{}, error) {
	arrayHash := make([]common.Uint256, 0)
	for _, output := range tx.Outputs {
		if output.Type == common2.OTReturnSideChainDepositCoin {
			payload, ok := output.Payload.(*outputpayload.ReturnSideChainDeposit)
			if ok {
				arrayHash = append(arrayHash, payload.DepositTransactionHash)
			} else {
				return nil, fmt.Errorf(
					"sidechain return deposit tx from sidechain output payload cast failed, tx: %s",
					tx.Hash())
			}
		}
	}
	return arrayHash, nil
}

// str array related functions
func strArrayTxReferences(tx *transactions.BaseTransaction) (interface{}, error) {
	reference, err :=
		blockchain.DefaultLedger.Blockchain.UTXOCache.GetTxReference(tx)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(reference))
	for k := range reference {
		result = append(result, k.ReferKey())
	}
	return result, nil
}

// common functions

func comGetProducerInfo(tx *transactions.BaseTransaction) (*payload.ProducerInfo, error) {
	p, ok := tx.Payload.(*payload.ProducerInfo)
	if !ok {
		return nil, fmt.Errorf(
			"register producer payload cast failed, tx:%s", tx.Hash())
	}
	return p, nil
}

func comGetCRInfo(tx *transactions.BaseTransaction) (*payload.CRInfo, error) {
	p, ok := tx.Payload.(*payload.CRInfo)
	if !ok {
		return nil, fmt.Errorf(
			"register CR payload cast failed, tx:%s", tx.Hash())
	}
	return p, nil
}
