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
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type CRCProposalWithdrawTransaction struct {
	BaseTransaction
}

func (t *CRCProposalWithdrawTransaction) CheckAttributeProgram() error {

	if len(t.Programs()) != 0 && t.parameters.BlockHeight <
		t.parameters.Config.CRConfiguration.CRCProposalWithdrawPayloadV1Height {
		return errors.New("crcproposalwithdraw tx should have no programs")
	}
	if t.PayloadVersion() == payload.CRCProposalWithdrawDefault {
		return nil
	}

	// Check attributes
	for _, attr := range t.Attributes() {
		if !common2.IsValidAttributeType(attr.Usage) {
			return fmt.Errorf("invalid attribute usage %v", attr.Usage)
		}
	}

	// Check programs
	if len(t.Programs()) == 0 {
		return fmt.Errorf("no programs found in transaction")
	}
	for _, program := range t.Programs() {
		if program.Code == nil {
			return fmt.Errorf("invalid program code nil")
		}
		if program.Parameter == nil {
			return fmt.Errorf("invalid program parameter nil")
		}
	}

	return nil
}

func (t *CRCProposalWithdrawTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.CRCProposalWithdraw:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *CRCProposalWithdrawTransaction) IsAllowedInPOWConsensus() bool {
	return false
}

func (t *CRCProposalWithdrawTransaction) HeightVersionCheck() error {
	blockHeight := t.parameters.BlockHeight
	chainParams := t.parameters.Config

	if blockHeight < chainParams.CRConfiguration.CRCommitteeStartHeight {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before CRCommitteeStartHeight", t.TxType().Name()))
	}
	if t.PayloadVersion() == payload.CRCProposalWithdrawDefault &&
		blockHeight >= chainParams.CRConfiguration.CRCProposalWithdrawPayloadV1Height {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"after CRCProposalWithdrawPayloadV1Height", t.TxType().Name()))
	}

	if t.PayloadVersion() == payload.CRCProposalWithdrawVersion01 &&
		blockHeight < chainParams.CRConfiguration.CRCProposalWithdrawPayloadV1Height {
		return errors.New(fmt.Sprintf("not support %s transaction "+
			"before CRCProposalWithdrawPayloadV1Height", t.TxType().Name()))
	}
	return nil
}

func (t *CRCProposalWithdrawTransaction) SpecialContextCheck() (result elaerr.ELAError, end bool) {
	CRExpensesAddress, _ := common.Uint168FromAddress(t.parameters.Config.CRConfiguration.CRExpensesAddress)
	if t.PayloadVersion() == payload.CRCProposalWithdrawDefault {
		for _, output := range t.references {
			if output.ProgramHash != *CRExpensesAddress {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("proposal withdrawal transaction for non-crc committee address")), true
			}
		}
	}

	withdrawPayload, ok := t.Payload().(*payload.CRCProposalWithdraw)
	if !ok {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("invalid payload")), true
	}
	// Check if the proposal exist.
	proposalState := t.parameters.BlockChain.GetCRCommittee().GetProposal(withdrawPayload.ProposalHash)
	if proposalState == nil {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("proposal not exist")), true
	}
	if proposalState.Status != crstate.VoterAgreed &&
		proposalState.Status != crstate.Finished &&
		proposalState.Status != crstate.Aborted &&
		proposalState.Status != crstate.Terminated {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("proposal status is not VoterAgreed , "+
			"Finished, Aborted or Terminated")), true
	}

	if !bytes.Equal(proposalState.ProposalOwner, withdrawPayload.OwnerPublicKey) {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("the OwnerPublicKey is not owner of proposal")), true
	}
	fee := getTransactionFee(t, t.references)
	if t.isSmallThanMinTransactionFee(fee) {
		return elaerr.Simple(elaerr.ErrTxPayload, fmt.Errorf("transaction fee not enough")), true
	}
	withdrawAmount := t.parameters.BlockChain.GetCRCommittee().AvailableWithdrawalAmount(withdrawPayload.ProposalHash)
	if withdrawAmount == 0 {
		return elaerr.Simple(elaerr.ErrTxPayload, errors.New("no need to withdraw")), true
	}
	if t.PayloadVersion() == payload.CRCProposalWithdrawDefault {
		// Check output[0] must equal with Recipient
		if t.Outputs()[0].ProgramHash != proposalState.Recipient {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("txn.Outputs()[0].ProgramHash != Recipient")), true
		}

		// Check output[1] if exist must equal with CRCComitteeAddresss
		if len(t.Outputs()) > 1 {
			if t.Outputs()[1].ProgramHash != *CRExpensesAddress {
				return elaerr.Simple(elaerr.ErrTxPayload, errors.New("txn.Outputs()[1].ProgramHash !=CRCComitteeAddresss")), true
			}
		}

		if len(t.Outputs()) > 2 {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("CRCProposalWithdraw tx should not have over two output")), true
		}

		//Recipient count + fee must equal to availableWithdrawalAmount
		if t.Outputs()[0].Value+fee != withdrawAmount {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("txn.Outputs()[0].Value + fee != withdrawAmout ")), true
		}
	} else if t.PayloadVersion() == payload.CRCProposalWithdrawVersion01 {
		// Recipient address must be the current recipient address
		if withdrawPayload.Recipient != proposalState.Recipient {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("withdrawPayload.Recipient != Recipient")), true
		}
		// Recipient Amount + fee must equal to availableWithdrawalAmount
		if withdrawPayload.Amount != withdrawAmount {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("withdrawPayload.Amount != withdrawAmount ")), true
		}
		if withdrawPayload.Amount <= t.parameters.Config.CRConfiguration.RealWithdrawSingleFee {
			return elaerr.Simple(elaerr.ErrTxPayload, errors.New("withdraw amount should be bigger than RealWithdrawSingleFee")), true
		}
	}

	signedBuf := new(bytes.Buffer)
	err := withdrawPayload.SerializeUnsigned(signedBuf, t.PayloadVersion())
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}
	var code []byte
	if code, err = getCode(withdrawPayload.OwnerPublicKey); err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}
	err = blockchain.CheckCRTransactionSignature(withdrawPayload.Signature, code, signedBuf.Bytes())
	if err != nil {
		return elaerr.Simple(elaerr.ErrTxPayload, err), true
	}

	return nil, false
}

func (t *CRCProposalWithdrawTransaction) isSmallThanMinTransactionFee(fee common.Fixed64) bool {
	if fee < t.parameters.Config.MinTransactionFee {
		return true
	}
	return false
}

func getTransactionFee(tx interfaces.Transaction,
	references map[*common2.Input]common2.Output) common.Fixed64 {
	var outputValue common.Fixed64
	var inputValue common.Fixed64
	for _, output := range tx.Outputs() {
		outputValue += output.Value
	}
	for _, output := range references {
		inputValue += output.Value
	}
	return inputValue - outputValue
}
