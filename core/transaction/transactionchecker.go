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

func (a *DefaultChecker) SanityCheck(p interfaces.Parameters) elaerr.ELAError {
	return nil
}

func (a *DefaultChecker) ContextCheck(params interfaces.Parameters) (
	map[*common2.Input]common2.Output, elaerr.ELAError) {

	if err := a.SetParameters(params); err != nil {
		return nil, elaerr.Simple(elaerr.ErrTxDuplicate, errors.New("invalid contextParameters"))
	}

	if err := a.HeightVersionCheck(); err != nil {
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

	if a.contextParameters.BlockChain.GetState().GetConsensusAlgorithm() == state.POW {
		if !a.IsAllowedInPOWConsensus() {
			log.Warnf("[CheckTransactionContext], %s transaction is not allowed in POW", a.contextParameters.Transaction.TxType().Name())
			return nil, elaerr.Simple(elaerr.ErrTxValidation, nil)
		}
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

	firstErr, end := a.SpecialContextCheck()
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
func (a *DefaultChecker) HeightVersionCheck() error {
	return nil
}

func (a *DefaultChecker) IsTxHashDuplicate(txHash common.Uint256) bool {
	return a.contextParameters.BlockChain.GetDB().IsTxHashDuplicate(txHash)
}

func (a *DefaultChecker) GetTxReference(txn interfaces.Transaction) (
	map[*common2.Input]common2.Output, error) {
	return a.contextParameters.BlockChain.UTXOCache.GetTxReference(txn)
}

func (a *DefaultChecker) IsAllowedInPOWConsensus() bool {
	return true
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

func (a *DefaultChecker) SpecialContextCheck() (elaerr.ELAError, bool) {
	fmt.Println("default check")
	return nil, false
}
