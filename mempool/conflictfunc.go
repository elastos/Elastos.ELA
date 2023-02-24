// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package mempool

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/vm"
)

// hashes related functions
func hashCRCProposalDraftHash(tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"CRC proposal payload cast failed, tx:%s", tx.Hash())
	}
	return p.DraftHash, nil
}

func hashCRCProposalDID(tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"CRC proposal payload cast failed, tx:%s", tx.Hash())
	}
	return p.CRCouncilMemberDID, nil
}

func strArrayCRCProposalCustomID(tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"CRC proposal payload cast failed, tx:%s", tx.Hash())
	}
	if p.ProposalType != payload.ReceiveCustomID {
		return nil, nil
	}
	return p.ReceivedCustomIDList, nil
}

func hashChangeProposalOwnerTargetProposalHash(tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"CRC proposal payload cast failed, tx:%s", tx.Hash())
	}
	if p.ProposalType == payload.ChangeProposalOwner {
		return p.TargetProposalHash, nil
	}
	return nil, nil
}

func hashCloseProposalTargetProposalHash(tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"CRC proposal payload cast failed, tx:%s", tx.Hash())
	}
	if p.ProposalType == payload.CloseProposal {
		return p.TargetProposalHash, nil
	}
	return nil, nil
}

func hashArrayNFTDestroyFromSideChainHash(tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.NFTDestroyFromSideChain)
	if !ok {
		return nil, fmt.Errorf(
			"CRC proposal payload cast failed, tx:%s", tx.Hash())
	}

	return p.IDs, nil
}

func hashCRCProposalSecretaryGeneralDID(tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"CRC proposal payload cast failed, tx:%s", tx.Hash())
	}
	if p.ProposalType == payload.SecretaryGeneral {
		return p.SecretaryGeneralDID, nil
	}
	return nil, nil
}

func strChangeCustomIDFee(tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"CRC proposal payload cast failed, tx:%s", tx.Hash())
	}
	if p.ProposalType == payload.ChangeCustomIDFee {
		return "Change the fee of custom ID", nil
	}
	return nil, nil
}

func strReserveCustomID(tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CRCProposal)
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
	tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CRCProposal)
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
	tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CRCProposal)
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
	tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CRCProposal)
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
	tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CRCProposalWithdraw)
	if !ok {
		return nil, fmt.Errorf(
			"crcProposalWithdraw  payload cast failed, tx:%s", tx.Hash())
	}
	return p.ProposalHash, nil
}

func hashCRCProposalTrackingProposalHash(
	tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CRCProposalTracking)
	if !ok {
		return nil, fmt.Errorf(
			"crcProposalTracking  payload cast failed, tx:%s", tx.Hash())
	}
	return p.ProposalHash, nil
}

func hashSpecialTxHash(tx interfaces.Transaction) (interface{}, error) {
	illegalData, ok := tx.Payload().(payload.DPOSIllegalData)
	if !ok {
		return nil, fmt.Errorf(
			"special tx payload cast failed, tx:%s", tx.Hash())
	}
	return illegalData.Hash(), nil
}

func hashNextTurnDPOSInfoTxPayloadHash(tx interfaces.Transaction) (interface{}, error) {
	payload, ok := tx.Payload().(*payload.NextTurnDPOSInfo)
	if !ok {
		return nil, fmt.Errorf(
			"NextTurnDPOSInfo tx payload cast failed, tx:%s", tx.Hash())
	}
	return payload.Hash(), nil
}

func hashCustomIDProposalResultTxPayloadHash(tx interfaces.Transaction) (interface{}, error) {
	_, ok := tx.Payload().(*payload.RecordProposalResult)
	if !ok {
		return nil, fmt.Errorf(
			"custom ID proposal result tx payload cast failed, tx:%s", tx.Hash())
	}
	return "customIDProposalResult", nil
}

// strings related functions
func strCancelProducerOwnerPublicKey(tx interfaces.Transaction) (interface{},
	error) {
	p, ok := tx.Payload().(*payload.ProcessProducer)
	if !ok {
		err := fmt.Errorf(
			"cancel producer payload cast failed, tx:%s", tx.Hash())
		return nil, errors.Simple(errors.ErrTxPoolFailure, err)
	}
	return common.BytesToHexString(p.OwnerPublicKey), nil
}

func strActivateAndCancelKeys(tx interfaces.Transaction) (interface{},
	error) {
	if tx.TxType() != common2.CancelProducer && tx.TxType() != common2.ActivateProducer {
		err := fmt.Errorf(
			"invalid tx:%s", tx.Hash())
		return nil, errors.Simple(errors.ErrTxPoolFailure, err)
	}
	return "activatecancel", nil
}

func strProducerInfoOwnerPublicKey(tx interfaces.Transaction) (interface{}, error) {
	p, err := comGetProducerInfo(tx)
	if err != nil {
		return nil, err
	}
	return common.BytesToHexString(p.OwnerPublicKey), nil
}

func strProducerInfoNodePublicKey(tx interfaces.Transaction) (interface{}, error) {
	p, err := comGetProducerInfo(tx)
	if err != nil {
		return nil, err
	}
	return common.BytesToHexString(p.NodePublicKey), nil
}

func strCRManagementPublicKey(tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CRCouncilMemberClaimNode)
	if !ok {
		return nil, fmt.Errorf(
			"cr dpos management payload cast failed, tx:%s", tx.Hash())
	}
	return common.BytesToHexString(p.NodePublicKey), nil
}

func strCRManagementDID(tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CRCouncilMemberClaimNode)
	if !ok {
		return nil, fmt.Errorf(
			"cr dpos management payload cast failed, tx:%s", tx.Hash())
	}
	return p.CRCouncilCommitteeDID, nil
}

func strProducerInfoNickname(tx interfaces.Transaction) (interface{}, error) {
	p, err := comGetProducerInfo(tx)
	if err != nil {
		return nil, err
	}
	return p.NickName, nil
}

func strStake(tx interfaces.Transaction) (interface{}, error) {
	if len(tx.Outputs()) < 1 {
		return nil, fmt.Errorf("invlid return votes outputs count, tx:%s", tx.Hash())
	}
	p := tx.Outputs()[0].Payload
	if p == nil {
		return nil, fmt.Errorf("invlid return votes outputs payload, tx:%s", tx.Hash())
	}
	pld, ok := p.(*outputpayload.ExchangeVotesOutput)
	if !ok {
		return nil, fmt.Errorf("invlid return votes output payload, tx:%s", tx.Hash())
	}

	return pld.StakeAddress, nil
}

func strVoting(tx interfaces.Transaction) (interface{}, error) {
	_, ok := tx.Payload().(*payload.Voting)
	if !ok {
		return nil, fmt.Errorf("invlid voting payload, tx:%s", tx.Hash())
	}
	if len(tx.Programs()) < 1 {
		return nil, fmt.Errorf("invalid voting programs count, tx:%s", tx.Hash())
	}
	code := tx.Programs()[0].Code
	ct, err := contract.CreateStakeContractByCode(code)
	if err != nil {
		return nil, fmt.Errorf("invlid voint code, tx:%s", tx.Hash())
	}
	stakeProgramHash := ct.ToProgramHash()
	return *stakeProgramHash, nil
}

func strReturnVotes(tx interfaces.Transaction) (interface{}, error) {
	pld, ok := tx.Payload().(*payload.ReturnVotes)
	if !ok {
		return nil, fmt.Errorf("invlid return votes payload, tx:%s", tx.Hash())
	}

	if len(tx.Programs()) < 1 {
		return nil, fmt.Errorf("invlid return votes program, tx:%s", tx.Hash())
	}

	var code []byte
	if tx.PayloadVersion() == payload.ReturnVotesVersionV0 {
		code = pld.Code
	} else {
		code = tx.Programs()[0].Code
	}
	ct, err := contract.CreateStakeContractByCode(code)
	if err != nil {
		return nil, fmt.Errorf("invlid return votes code, tx:%s", tx.Hash())
	}
	stakeProgramHash := ct.ToProgramHash()
	return *stakeProgramHash, nil
}

func programHashDposV2ClaimReward(tx interfaces.Transaction) (interface{}, error) {
	pld, ok := tx.Payload().(*payload.DPoSV2ClaimReward)
	if !ok {
		return nil, fmt.Errorf("invlid DPoSV2ClaimReward payload, tx:%s", tx.Hash())
	}
	if len(tx.Programs()) < 1 {
		return nil, fmt.Errorf("invalid DPoSV2ClaimReward programs count, tx:%s", tx.Hash())
	}

	var code []byte
	if tx.PayloadVersion() == payload.DposV2ClaimRewardVersionV0 {
		code = pld.Code
	} else {
		code = tx.Programs()[0].Code
	}
	ct, err := contract.CreateStakeContractByCode(code)
	if err != nil {
		return nil, fmt.Errorf("invlid DPoSV2ClaimReward code, tx:%s", tx.Hash())
	}
	programHash := ct.ToProgramHash()
	return *programHash, nil
}

func strRegisterCRPublicKey(tx interfaces.Transaction) (interface{}, error) {
	p, err := comGetCRInfo(tx)
	if err != nil {
		return nil, err
	}

	var code []byte
	if tx.PayloadVersion() == payload.CRInfoSchnorrVersion {
		code = tx.Programs()[0].Code
	} else {
		code = p.Code
	}
	signType, err := crypto.GetScriptType(code)
	if err != nil {
		return nil, err
	}

	if signType == vm.CHECKSIG {
		return hex.EncodeToString(p.Code[1 : len(p.Code)-1]), nil
	} else if bytes.Equal(p.Code, []byte{}) && contract.IsSchnorr(code) {
		return hex.EncodeToString(code[2:]), nil
	} else {
		return nil, fmt.Errorf("unsupported sign script type: %d", signType)
	}
}

func strActivateProducerNodePublicKey(
	tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.ActivateProducer)
	if !ok {
		return nil, fmt.Errorf(
			"activate producer payload cast failed, tx:%s", tx.Hash())
	}
	return common.BytesToHexString(p.NodePublicKey), nil
}

func strCRInfoNickname(tx interfaces.Transaction) (interface{}, error) {
	p, err := comGetCRInfo(tx)
	if err != nil {
		return nil, err
	}
	return p.NickName, nil
}

func strTxProgramCode(tx interfaces.Transaction) (interface{}, error) {
	return common.BytesToHexString(tx.Programs()[0].Code), nil
}

func strProposalReviewKey(tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CRCProposalReview)
	if !ok {
		return nil, fmt.Errorf(
			"crcProposalReview  payload cast failed, tx:%s",
			tx.Hash())
	}
	return p.DID.String() + p.ProposalHash.String(), nil
}

func strCRCAppropriation(interfaces.Transaction) (interface{}, error) {
	// const string to ensure only one tx added to the tx pool
	return "CRC Appropriation", nil
}

func strSecretaryGeneral(tx interfaces.Transaction) (interface{}, error) {
	// const string to ensure only one tx added to the tx pool
	p, ok := tx.Payload().(*payload.CRCProposal)
	if !ok {
		return nil, fmt.Errorf(
			"CRC proposal payload cast failed, tx:%s", tx.Hash())
	}
	if p.ProposalType == payload.SecretaryGeneral {
		return "Secretary General", nil
	}
	return nil, nil
}

func strVotesRealWithdrawTX(
	tx interfaces.Transaction) (interface{}, error) {
	_, ok := tx.Payload().(*payload.VotesRealWithdrawPayload)
	if !ok {
		return nil, fmt.Errorf(
			"VotesRealWithdrawPayload cast failed, tx: %s",
			tx.Hash())
	}

	return "VotesRealWithdraw", nil
}

func hashArrayDPoSV2ClaimRewardRealWithdrawTransactionHashes(
	tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.DposV2ClaimRewardRealWithdraw)
	if !ok {
		return nil, fmt.Errorf(
			"real proposal withdraw transaction cast failed, tx: %s",
			tx.Hash())
	}

	return p.WithdrawTransactionHashes, nil
}

func hashArrayCRCProposalRealWithdrawTransactionHashes(
	tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CRCProposalRealWithdraw)
	if !ok {
		return nil, fmt.Errorf(
			"real proposal withdraw transaction cast failed, tx: %s",
			tx.Hash())
	}

	return p.WithdrawTransactionHashes, nil
}

func hashRevertToDPOS(tx interfaces.Transaction) (interface{}, error) {
	_, ok := tx.Payload().(*payload.RevertToDPOS)
	if !ok {
		return nil, fmt.Errorf(
			"RevertToDPOS transaction cast failed, tx: %s",
			tx.Hash())
	}

	return "RevertToDPOS", nil
}

// program hashes related functions
func addrCRInfoCRCID(tx interfaces.Transaction) (interface{}, error) {
	p, err := comGetCRInfo(tx)
	if err != nil {
		return nil, err
	}
	return p.CID, nil
}

func addrUnregisterCRCID(tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.UnregisterCR)
	if !ok {
		return nil, fmt.Errorf(
			"unregisterCR CR payload cast failed, tx: %s", tx.Hash())
	}
	return p.CID, nil
}

// hash array related functions
func hashArraySidechainTransactionHashes(
	tx interfaces.Transaction) (interface{}, error) {
	if tx.PayloadVersion() == payload.WithdrawFromSideChainVersion {
		p, ok := tx.Payload().(*payload.WithdrawFromSideChain)
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
	} else if tx.PayloadVersion() == payload.WithdrawFromSideChainVersionV1 {
		array := make([]common.Uint256, 0)
		for _, output := range tx.Outputs() {
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

	p, ok := tx.Payload().(*payload.WithdrawFromSideChain)
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
	tx interfaces.Transaction) (interface{}, error) {
	arrayHash := make([]common.Uint256, 0)
	for _, output := range tx.Outputs() {
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
func strDPoSOwnerNodePublicKeys(tx interfaces.Transaction) (interface{}, error) {
	p, err := comGetProducerInfo(tx)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, 2)

	ownerPubkeyStr := common.BytesToHexString(p.OwnerPublicKey)
	result = append(result, ownerPubkeyStr)

	nodePubkeyStr := common.BytesToHexString(p.NodePublicKey)
	if nodePubkeyStr != ownerPubkeyStr {
		result = append(result, nodePubkeyStr)
	}
	return result, nil
}

// str array related functions
func strArrayTxReferences(tx interfaces.Transaction) (interface{}, error) {
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

func comGetProducerInfo(tx interfaces.Transaction) (*payload.ProducerInfo, error) {
	p, ok := tx.Payload().(*payload.ProducerInfo)
	if !ok {
		return nil, fmt.Errorf(
			"register producer payload cast failed, tx:%s", tx.Hash())
	}
	return p, nil
}

func comGetCRInfo(tx interfaces.Transaction) (*payload.CRInfo, error) {
	p, ok := tx.Payload().(*payload.CRInfo)
	if !ok {
		return nil, fmt.Errorf(
			"register CR payload cast failed, tx:%s", tx.Hash())
	}
	return p, nil
}

func hashCreateNFTID(tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CreateNFT)
	if !ok {
		return nil, fmt.Errorf(
			"CreateNFT payload cast failed, tx: %s", tx.Hash())
	}
	return p.ID, nil
}

func strCreateNFTID(tx interfaces.Transaction) (interface{}, error) {
	p, ok := tx.Payload().(*payload.CreateNFT)
	if !ok {
		return nil, fmt.Errorf(
			"CreateNFT payload cast failed, tx: %s", tx.Hash())
	}
	return p.StakeAddress, nil
}
