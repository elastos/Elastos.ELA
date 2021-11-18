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
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/dpos/state"
	"math"

	"github.com/elastos/Elastos.ELA/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type DefaultChecker struct {
	sanityParameters  *TransactionParameters
	contextParameters *TransactionParameters

	references map[*common2.Input]common2.Output
}

func (a *DefaultChecker) SetParameters(params interface{}) elaerr.ELAError {
	var ok bool
	if a.contextParameters, ok = params.(*TransactionParameters); !ok {
		return elaerr.Simple(elaerr.ErrTxDuplicate, errors.New("invalid contextParameters"))
	}

	return nil

}

func (a *DefaultChecker) ContextCheck(params interfaces.Parameters) (
	map[*common2.Input]common2.Output, elaerr.ELAError) {

	if err := a.SetParameters(params); err != nil {
		return nil, elaerr.Simple(elaerr.ErrTxDuplicate, errors.New("invalid contextParameters"))
	}

	if err := a.CheckTxHeightVersion(); err != nil {
		return nil, elaerr.Simple(elaerr.ErrTxHeightVersion, nil)
	}

	if exist := a.IsTxHashDuplicate(a.contextParameters.Transaction.Hash()); exist {
		log.Warn("[CheckTransactionContext] duplicate transaction check failed.")
		return nil, elaerr.Simple(elaerr.ErrTxDuplicate, nil)
	}

	references, err := a.GetTxReference(a.contextParameters.Transaction)
	if err != nil {
		log.Warn("[CheckTransactionContext] get transaction reference failed")
		return nil, elaerr.Simple(elaerr.ErrTxUnknownReferredTx, nil)
	}
	a.references = references

	if err := a.CheckPOWConsensusTransaction(references); err != nil {
		log.Warn("[checkPOWConsensusTransaction],", err)
		return nil, elaerr.Simple(elaerr.ErrTxValidation, nil)
	}

	// check double spent transaction
	if blockchain.DefaultLedger.IsDoubleSpend(a.contextParameters.Transaction) {
		log.Warn("[CheckTransactionContext] IsDoubleSpend check failed")
		return nil, elaerr.Simple(elaerr.ErrTxDoubleSpend, nil)
	}

	if err := a.CheckTransactionUTXOLock(a.contextParameters.Transaction, references); err != nil {
		log.Warn("[CheckTransactionUTXOLock],", err)
		return nil, elaerr.Simple(elaerr.ErrTxUTXOLocked, err)
	}

	firstErr, end := a.SpecialCheck()
	if end {
		return nil, firstErr
	}

	if err := a.checkTransactionFee(a.contextParameters.Transaction, references); err != nil {
		log.Warn("[CheckTransactionFee],", err)
		return nil, elaerr.Simple(elaerr.ErrTxBalance, err)
	}

	if err := checkDestructionAddress(references); err != nil {
		log.Warn("[CheckDestructionAddress], ", err)
		return nil, elaerr.Simple(elaerr.ErrTxInvalidInput, err)
	}

	if err := checkTransactionDepositUTXO(a.contextParameters.Transaction, references); err != nil {
		log.Warn("[CheckTransactionDepositUTXO],", err)
		return nil, elaerr.Simple(elaerr.ErrTxInvalidInput, err)
	}

	if err := checkTransactionDepositOutpus(a.contextParameters.BlockChain, a.contextParameters.Transaction); err != nil {
		log.Warn("[checkTransactionDepositOutpus],", err)
		return nil, elaerr.Simple(elaerr.ErrTxInvalidInput, err)
	}

	if err := checkTransactionSignature(a.contextParameters.Transaction, references); err != nil {
		log.Warn("[CheckTransactionSignature],", err)
		return nil, elaerr.Simple(elaerr.ErrTxSignature, err)
	}

	if err := a.checkInvalidUTXO(a.contextParameters.Transaction); err != nil {
		log.Warn("[CheckTransactionCoinbaseLock]", err)
		return nil, elaerr.Simple(elaerr.ErrBlockIneffectiveCoinbase, err)
	}

	return references, nil
}

func (a *DefaultChecker) checkInvalidUTXO(txn interfaces.Transaction) error {
	currentHeight := blockchain.DefaultLedger.Blockchain.GetHeight()
	for _, input := range txn.Inputs() {
		referTxn, err := a.contextParameters.BlockChain.UTXOCache.GetTransaction(input.Previous.TxID)
		if err != nil {
			return err
		}
		if referTxn.IsCoinBaseTx() {
			if currentHeight-referTxn.LockTime() < a.contextParameters.Config.CoinbaseMaturity {
				return errors.New("the utxo of coinbase is locking")
			}
		} else if referTxn.IsNewSideChainPowTx() {
			return errors.New("cannot spend the utxo from a new sideChainPow tx")
		}
	}

	return nil
}

func checkTransactionSignature(tx interfaces.Transaction, references map[*common2.Input]common2.Output) error {
	programHashes, err := blockchain.GetTxProgramHashes(tx, references)
	if (tx.IsCRCProposalWithdrawTx() && tx.PayloadVersion() == payload.CRCProposalWithdrawDefault) ||
		tx.IsCRAssetsRectifyTx() || tx.IsCRCProposalRealWithdrawTx() || tx.IsNextTurnDPOSInfoTx() {
		return nil
	}
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	tx.SerializeUnsigned(buf)

	// sort the program hashes of owner and programs of the transaction
	common.SortProgramHashByCodeHash(programHashes)
	blockchain.SortPrograms(tx.Programs())
	return blockchain.RunPrograms(buf.Bytes(), programHashes, tx.Programs())
}

func checkTransactionDepositOutpus(bc *blockchain.BlockChain, txn interfaces.Transaction) error {
	for _, output := range txn.Outputs() {
		if contract.GetPrefixType(output.ProgramHash) == contract.PrefixDeposit {
			if txn.IsRegisterProducerTx() || txn.IsRegisterCRTx() ||
				txn.IsReturnDepositCoin() || txn.IsReturnCRDepositCoinTx() {
				continue
			}
			if bc.GetState().ExistProducerByDepositHash(output.ProgramHash) {
				continue
			}
			if bc.GetCRCommittee().ExistCandidateByDepositHash(
				output.ProgramHash) {
				continue
			}
			return errors.New("only the address that CR or Producer" +
				" registered can have the deposit UTXO")
		}
	}

	return nil
}

func checkTransactionDepositUTXO(txn interfaces.Transaction, references map[*common2.Input]common2.Output) error {
	for _, output := range references {
		if contract.GetPrefixType(output.ProgramHash) == contract.PrefixDeposit {
			if !txn.IsReturnDepositCoin() && !txn.IsReturnCRDepositCoinTx() {
				return errors.New("only the ReturnDepositCoin and " +
					"ReturnCRDepositCoin transaction can use the deposit UTXO")
			}
		} else {
			if txn.IsReturnDepositCoin() || txn.IsReturnCRDepositCoinTx() {
				return errors.New("the ReturnDepositCoin and ReturnCRDepositCoin " +
					"transaction can only use the deposit UTXO")
			}
		}
	}

	return nil
}

func checkDestructionAddress(references map[*common2.Input]common2.Output) error {
	for _, output := range references {
		if output.ProgramHash == config.DestroyELAAddress {
			return errors.New("cannot use utxo from the destruction address")
		}
	}
	return nil
}

func (a *DefaultChecker) checkTransactionFee(tx interfaces.Transaction, references map[*common2.Input]common2.Output) error {
	fee := getTransactionFee(tx, references)
	if a.isSmallThanMinTransactionFee(fee) {
		return fmt.Errorf("transaction fee not enough")
	}
	// set Fee and FeePerKB if check has passed
	tx.SetFee(fee)
	buf := new(bytes.Buffer)
	tx.Serialize(buf)
	tx.SetFeePerKB(fee * 1000 / common.Fixed64(len(buf.Bytes())))
	return nil
}

func (a *DefaultChecker) isSmallThanMinTransactionFee(fee common.Fixed64) bool {
	if fee < a.contextParameters.Config.MinTransactionFee {
		return true
	}
	return false
}

// validate the type of transaction is allowed or not at current height.
func (a *DefaultChecker) CheckTxHeightVersion() error {
	txn := a.contextParameters.Transaction
	blockHeight := a.contextParameters.BlockHeight
	chainParams := a.contextParameters.Config

	switch txn.TxType() {
	case common2.RevertToPOW, common2.RevertToDPOS:
		if blockHeight < chainParams.RevertToPOWStartHeight {
			return errors.New(fmt.Sprintf("not support %s transaction "+
				"before RevertToPOWStartHeight", txn.TxType().Name()))
		}

	case common2.RegisterCR, common2.UpdateCR:
		if blockHeight < chainParams.CRVotingStartHeight ||
			(blockHeight < chainParams.RegisterCRByDIDHeight &&
				txn.PayloadVersion() != payload.CRInfoVersion) {
			return errors.New(fmt.Sprintf("not support %s transaction "+
				"before CRVotingStartHeight", txn.TxType().Name()))
		}
	case common2.UnregisterCR, common2.ReturnCRDepositCoin:
		if blockHeight < chainParams.CRVotingStartHeight {
			return errors.New(fmt.Sprintf("not support %s transaction "+
				"before CRVotingStartHeight", txn.TxType().Name()))
		}
	case common2.CRCProposal:
		if blockHeight < chainParams.CRCProposalDraftDataStartHeight {
			if txn.PayloadVersion() != payload.CRCProposalVersion {
				return errors.New("payload version should be CRCProposalVersion")
			}
		} else {
			if txn.PayloadVersion() != payload.CRCProposalVersion01 {
				return errors.New("should have draft data")
			}
		}

		p, ok := txn.Payload().(*payload.CRCProposal)
		if !ok {
			return errors.New("not support invalid CRCProposal transaction")
		}
		switch p.ProposalType {
		case payload.ChangeProposalOwner, payload.CloseProposal, payload.SecretaryGeneral:
			if blockHeight < chainParams.CRCProposalV1Height {
				return errors.New(fmt.Sprintf("not support %s CRCProposal"+
					" transactio before CRCProposalV1Height", p.ProposalType.Name()))
			}
		case payload.ReserveCustomID, payload.ReceiveCustomID, payload.ChangeCustomIDFee:
			if blockHeight < chainParams.CustomIDProposalStartHeight {
				return errors.New(fmt.Sprintf("not support %s CRCProposal"+
					" transaction before CustomIDProposalStartHeight", p.ProposalType.Name()))
			}
		case payload.RegisterSideChain:
			if blockHeight < chainParams.NewCrossChainStartHeight {
				return errors.New(fmt.Sprintf("not support %s CRCProposal"+
					" transaction before NewCrossChainStartHeight", p.ProposalType.Name()))
			}
		default:
			if blockHeight < chainParams.CRCommitteeStartHeight {
				return errors.New(fmt.Sprintf("not support %s CRCProposal"+
					" transaction before CRCommitteeStartHeight", p.ProposalType.Name()))
			}
		}
	case common2.CRCProposalReview, common2.CRCProposalTracking:
		if blockHeight < chainParams.CRCommitteeStartHeight {
			return errors.New(fmt.Sprintf("not support %s transaction "+
				"before CRCommitteeStartHeight", txn.TxType().Name()))
		} else if blockHeight < chainParams.CRCProposalDraftDataStartHeight {
			if txn.PayloadVersion() != payload.CRCProposalVersion {
				return errors.New("payload version should be CRCProposalVersion")
			}
		} else {
			if txn.PayloadVersion() != payload.CRCProposalVersion01 {
				return errors.New("should have draft data")
			}
		}

	case common2.CRCAppropriation:
		if blockHeight < chainParams.CRCommitteeStartHeight {
			return errors.New(fmt.Sprintf("not support %s transaction "+
				"before CRCommitteeStartHeight", txn.TxType().Name()))
		}

	case common2.CRCProposalWithdraw:
		if blockHeight < chainParams.CRCommitteeStartHeight {
			return errors.New(fmt.Sprintf("not support %s transaction "+
				"before CRCommitteeStartHeight", txn.TxType().Name()))
		}
		if txn.PayloadVersion() == payload.CRCProposalWithdrawDefault &&
			blockHeight >= chainParams.CRCProposalWithdrawPayloadV1Height {
			return errors.New(fmt.Sprintf("not support %s transaction "+
				"after CRCProposalWithdrawPayloadV1Height", txn.TxType().Name()))
		}

		if txn.PayloadVersion() == payload.CRCProposalWithdrawVersion01 &&
			blockHeight < chainParams.CRCProposalWithdrawPayloadV1Height {
			return errors.New(fmt.Sprintf("not support %s transaction "+
				"before CRCProposalWithdrawPayloadV1Height", txn.TxType().Name()))
		}
	case common2.CRAssetsRectify, common2.CRCProposalRealWithdraw:
		if blockHeight < chainParams.CRAssetsRectifyTransactionHeight {
			return errors.New(fmt.Sprintf("not support %s transaction "+
				"before CRCProposalWithdrawPayloadV1Height", txn.TxType().Name()))
		}
	case common2.CRCouncilMemberClaimNode:
		if blockHeight < chainParams.CRClaimDPOSNodeStartHeight {
			return errors.New(fmt.Sprintf("not support %s transaction "+
				"before CRClaimDPOSNodeStartHeight", txn.TxType().Name()))
		}
	case common2.TransferAsset:
		if blockHeight >= chainParams.CRVotingStartHeight {
			return nil
		}
		if txn.Version() >= common2.TxVersion09 {
			for _, output := range txn.Outputs() {
				if output.Type != common2.OTVote {
					continue
				}
				p, _ := output.Payload.(*outputpayload.VoteOutput)
				if p.Version >= outputpayload.VoteProducerAndCRVersion {
					return errors.New("not support " +
						"VoteProducerAndCRVersion before CRVotingStartHeight")
				}
			}
		}
	case common2.TransferCrossChainAsset:
		if blockHeight <= chainParams.NewCrossChainStartHeight {
			if txn.PayloadVersion() != payload.TransferCrossChainVersion {
				return errors.New("not support " +
					"TransferCrossChainAsset payload version V1 before NewCrossChainStartHeight")
			}
			return nil
		} else {
			if txn.PayloadVersion() != payload.TransferCrossChainVersionV1 {
				return errors.New("not support " +
					"TransferCrossChainAsset payload version V0 after NewCrossChainStartHeight")
			}
		}
	case common2.ReturnSideChainDepositCoin:
		if blockHeight < chainParams.ReturnCrossChainCoinStartHeight {
			return errors.New(fmt.Sprintf("not support %s transaction "+
				"before ReturnCrossChainCoinStartHeight", txn.TxType().Name()))
		}
	}

	return nil
}

func (a *DefaultChecker) IsTxHashDuplicate(txHash common.Uint256) bool {
	return a.contextParameters.BlockChain.GetDB().IsTxHashDuplicate(txHash)
}

func (a *DefaultChecker) GetTxReference(txn interfaces.Transaction) (
	map[*common2.Input]common2.Output, error) {
	return a.contextParameters.BlockChain.UTXOCache.GetTxReference(txn)
}

func (a *DefaultChecker) CheckPOWConsensusTransaction(references map[*common2.Input]common2.Output) error {
	txn := a.contextParameters.Transaction
	b := a.contextParameters.BlockChain

	if b.GetState().GetConsensusAlgorithm() != state.POW {
		return nil
	}

	switch txn.TxType() {
	case common2.RegisterProducer, common2.ActivateProducer, common2.CRCouncilMemberClaimNode:
		return nil
	case common2.CRCAppropriation, common2.CRAssetsRectify, common2.CRCProposalRealWithdraw,
		common2.NextTurnDPOSInfo, common2.RevertToDPOS:
		return nil
	case common2.TransferAsset:
		if txn.Version() >= common2.TxVersion09 {
			var containVoteOutput bool
			for _, output := range txn.Outputs() {
				if output.Type == common2.OTVote {
					p := output.Payload.(*outputpayload.VoteOutput)
					for _, vote := range p.Contents {
						switch vote.VoteType {
						case outputpayload.Delegate:
						case outputpayload.CRC:
							return errors.New("not allow to vote CR in POW consensus")
						case outputpayload.CRCProposal:
							return errors.New("not allow to vote CRC proposal in POW consensus")
						case outputpayload.CRCImpeachment:
							return errors.New("not allow to vote CRImpeachment in POW consensus")
						}
					}
					containVoteOutput = true
				}
			}
			if !containVoteOutput {
				return errors.New("not allow to transfer asset in POW consensus")
			}

			inputProgramHashes := make(map[common.Uint168]struct{})
			for _, output := range references {
				inputProgramHashes[output.ProgramHash] = struct{}{}
			}
			outputProgramHashes := make(map[common.Uint168]struct{})
			for _, output := range txn.Outputs() {
				outputProgramHashes[output.ProgramHash] = struct{}{}
			}
			for k, _ := range outputProgramHashes {
				if _, ok := inputProgramHashes[k]; !ok {
					return errors.New("output program hash is not in inputs")
				}
			}
		} else {
			return errors.New("not allow to transfer asset in POW consensus")
		}
		return nil
	}

	return fmt.Errorf("not support transaction %s in POW consensus", txn.TxType().Name())
}

func (a *DefaultChecker) CheckTransactionUTXOLock(txn interfaces.Transaction, references map[*common2.Input]common2.Output) error {
	for input, output := range references {

		if output.OutputLock == 0 {
			//check next utxo
			continue
		}
		if input.Sequence != math.MaxUint32-1 {
			return errors.New("Invalid input sequence")
		}
		if txn.LockTime() < output.OutputLock {
			return errors.New("UTXO output locked")
		}
	}
	return nil
}

func (a *DefaultChecker) SpecialCheck() (elaerr.ELAError, bool) {
	fmt.Println("default check")
	return nil, false
}
