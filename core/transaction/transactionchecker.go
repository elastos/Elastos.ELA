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
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/dpos/state"
	"github.com/elastos/Elastos.ELA/elanet/pact"
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

func (t *DefaultChecker) SanityCheck(params interfaces.Parameters) elaerr.ELAError {
	if err := t.SetContextParameters(params); err != nil {
		return elaerr.Simple(elaerr.ErrTxDuplicate, errors.New("invalid contextParameters"))
	}

	if err := t.HeightVersionCheck(); err != nil {
		return elaerr.Simple(elaerr.ErrTxHeightVersion, nil)
	}

	if err := t.CheckTransactionSize(); err != nil {
		log.Warn("[CheckTransactionSize],", err)
		return elaerr.Simple(elaerr.ErrTxSize, err)
	}

	if err := t.CheckTransactionInput(); err != nil {
		log.Warn("[CheckTransactionInput],", err)
		return elaerr.Simple(elaerr.ErrTxInvalidInput, err)
	}

	return nil
}

func (t *DefaultChecker) ContextCheck(params interfaces.Parameters) (
	map[*common2.Input]common2.Output, elaerr.ELAError) {

	if err := t.SetContextParameters(params); err != nil {
		return nil, elaerr.Simple(elaerr.ErrTxDuplicate, errors.New("invalid contextParameters"))
	}

	if err := t.HeightVersionCheck(); err != nil {
		return nil, elaerr.Simple(elaerr.ErrTxHeightVersion, nil)
	}

	if exist := t.IsTxHashDuplicate(t.contextParameters.Transaction.Hash()); exist {
		log.Warn("[CheckTransactionContext] duplicate transaction check failed.")
		return nil, elaerr.Simple(elaerr.ErrTxDuplicate, nil)
	}

	references, err := t.GetTxReference(t.contextParameters.Transaction)
	if err != nil {
		log.Warn("[CheckTransactionContext] get transaction reference failed")
		return nil, elaerr.Simple(elaerr.ErrTxUnknownReferredTx, nil)
	}
	t.references = references

	if t.contextParameters.BlockChain.GetState().GetConsensusAlgorithm() == state.POW {
		if !t.IsAllowedInPOWConsensus() {
			log.Warnf("[CheckTransactionContext], %s transaction is not allowed in POW", t.contextParameters.Transaction.TxType().Name())
			return nil, elaerr.Simple(elaerr.ErrTxValidation, nil)
		}
	}

	// check double spent transaction
	if blockchain.DefaultLedger.IsDoubleSpend(t.contextParameters.Transaction) {
		log.Warn("[CheckTransactionContext] IsDoubleSpend check failed")
		return nil, elaerr.Simple(elaerr.ErrTxDoubleSpend, nil)
	}

	if err := t.CheckTransactionUTXOLock(t.contextParameters.Transaction, references); err != nil {
		log.Warn("[CheckTransactionUTXOLock],", err)
		return nil, elaerr.Simple(elaerr.ErrTxUTXOLocked, err)
	}

	firstErr, end := t.SpecialContextCheck()
	if end {
		return nil, firstErr
	}

	if err := t.checkTransactionFee(t.contextParameters.Transaction, references); err != nil {
		log.Warn("[CheckTransactionFee],", err)
		return nil, elaerr.Simple(elaerr.ErrTxBalance, err)
	}

	if err := checkDestructionAddress(references); err != nil {
		log.Warn("[CheckDestructionAddress], ", err)
		return nil, elaerr.Simple(elaerr.ErrTxInvalidInput, err)
	}

	if err := checkTransactionDepositUTXO(t.contextParameters.Transaction, references); err != nil {
		log.Warn("[CheckTransactionDepositUTXO],", err)
		return nil, elaerr.Simple(elaerr.ErrTxInvalidInput, err)
	}

	if err := checkTransactionDepositOutputs(t.contextParameters.BlockChain, t.contextParameters.Transaction); err != nil {
		log.Warn("[checkTransactionDepositOutputs],", err)
		return nil, elaerr.Simple(elaerr.ErrTxInvalidInput, err)
	}

	if err := checkTransactionSignature(t.contextParameters.Transaction, references); err != nil {
		log.Warn("[checkTransactionSignature],", err)
		return nil, elaerr.Simple(elaerr.ErrTxSignature, err)
	}

	if err := t.checkInvalidUTXO(t.contextParameters.Transaction); err != nil {
		log.Warn("[checkInvalidUTXO]", err)
		return nil, elaerr.Simple(elaerr.ErrBlockIneffectiveCoinbase, err)
	}

	if err := t.tryCheckVoteOutputs(); err != nil {
		log.Warn("[tryCheckVoteOutputs]", err)
		return nil, elaerr.Simple(elaerr.ErrTxInvalidOutput, err)
	}

	return references, nil
}

func (t *DefaultChecker) SetSanityParameters(params interface{}) elaerr.ELAError {
	var ok bool
	if t.sanityParameters, ok = params.(*TransactionParameters); !ok {
		return elaerr.Simple(elaerr.ErrTxDuplicate, errors.New("invalid sanityParameters"))
	}

	return nil
}

func (t *DefaultChecker) SetContextParameters(params interface{}) elaerr.ELAError {
	var ok bool
	if t.contextParameters, ok = params.(*TransactionParameters); !ok {
		return elaerr.Simple(elaerr.ErrTxDuplicate, errors.New("invalid contextParameters"))
	}

	return nil
}

func (t *DefaultChecker) CheckTransactionSize() error {
	size := t.sanityParameters.Transaction.GetSize()
	if size <= 0 || size > int(pact.MaxBlockContextSize) {
		return fmt.Errorf("Invalid transaction size: %d bytes", size)
	}

	return nil
}

//validate the transaction of duplicate UTXO input
func (t *DefaultChecker) CheckTransactionInput() error {
	txn := t.sanityParameters.Transaction
	if len(txn.Inputs()) <= 0 {
		return errors.New("transaction has no inputs")
	}
	existingTxInputs := make(map[string]struct{})
	for _, input := range txn.Inputs() {
		if input.Previous.TxID.IsEqual(common.EmptyHash) && (input.Previous.Index == math.MaxUint16) {
			return errors.New("invalid transaction input")
		}
		if _, exists := existingTxInputs[input.ReferKey()]; exists {
			return errors.New("duplicated transaction inputs")
		} else {
			existingTxInputs[input.ReferKey()] = struct{}{}
		}
	}

	return nil
}

func (t *DefaultChecker) checkTransactionOutput() error {
	//txn := t.sanityParameters.Transaction
	//blockHeight := t.sanityParameters.BlockHeight
	//if len(txn.Outputs()) > math.MaxUint16 {
	//	return errors.New("output count should not be greater than 65535(MaxUint16)")
	//}
	//
	//if txn.IsCoinBaseTx() {
	//	if len(txn.Outputs()) < 2 {
	//		return errors.New("coinbase output is not enough, at least 2")
	//	}
	//
	//	foundationReward := txn.Outputs()[0].Value
	//	var totalReward = common.Fixed64(0)
	//	if blockHeight < b.chainParams.PublicDPOSHeight {
	//		for _, output := range txn.Outputs() {
	//			if output.AssetID != config.ELAAssetID {
	//				return errors.New("asset ID in coinbase is invalid")
	//			}
	//			totalReward += output.Value
	//		}
	//
	//		if foundationReward < common.Fixed64(float64(totalReward)*0.3) {
	//			return errors.New("reward to foundation in coinbase < 30%")
	//		}
	//	} else {
	//		// check the ratio of FoundationAddress reward with miner reward
	//		totalReward = txn.Outputs()[0].Value + txn.Outputs()[1].Value
	//		if len(txn.Outputs()) == 2 && foundationReward <
	//			common.Fixed64(float64(totalReward)*0.3/0.65) {
	//			return errors.New("reward to foundation in coinbase < 30%")
	//		}
	//	}
	//
	//	return nil
	//}
	//
	//if txn.IsIllegalTypeTx() || txn.IsInactiveArbitrators() ||
	//	txn.IsUpdateVersion() || txn.IsActivateProducerTx() ||
	//	txn.IsNextTurnDPOSInfoTx() || txn.IsRevertToPOW() ||
	//	txn.IsRevertToDPOS() || txn.IsCustomIDResultTx() {
	//	if len(txn.Outputs()) != 0 {
	//		return errors.New("no cost transactions should have no output")
	//	}
	//
	//	return nil
	//}
	//
	//if txn.IsCRCAppropriationTx() {
	//	if len(txn.Outputs()) != 2 {
	//		return errors.New("new CRCAppropriation tx must have two output")
	//	}
	//	if !txn.Outputs()[0].ProgramHash.IsEqual(b.chainParams.CRExpensesAddress) {
	//		return errors.New("new CRCAppropriation tx must have the first" +
	//			"output to CR expenses address")
	//	}
	//	if !txn.Outputs()[1].ProgramHash.IsEqual(b.chainParams.CRAssetsAddress) {
	//		return errors.New("new CRCAppropriation tx must have the second" +
	//			"output to CR assets address")
	//	}
	//}
	//
	//if txn.IsNewSideChainPowTx() {
	//	if len(txn.Outputs()) != 1 {
	//		return errors.New("new sideChainPow tx must have only one output")
	//	}
	//	if txn.Outputs()[0].Value != 0 {
	//		return errors.New("the value of new sideChainPow tx output must be 0")
	//	}
	//	if txn.Outputs()[0].Type != common2.OTNone {
	//		return errors.New("the type of new sideChainPow tx output must be OTNone")
	//	}
	//	return nil
	//}
	//
	//if len(txn.Outputs()) < 1 {
	//	return errors.New("transaction has no outputs")
	//}
	//
	//// check if output address is valid
	//specialOutputCount := 0
	//for _, output := range txn.Outputs() {
	//	if output.AssetID != config.ELAAssetID {
	//		return errors.New("asset ID in output is invalid")
	//	}
	//
	//	// output value must >= 0
	//	if output.Value < common.Fixed64(0) {
	//		return errors.New("Invalide transaction UTXO output.")
	//	}
	//
	//	if err := checkOutputProgramHash(blockHeight, output.ProgramHash); err != nil {
	//		return err
	//	}
	//
	//	if txn.Version() >= common2.TxVersion09 {
	//		if output.Type != common2.OTNone {
	//			specialOutputCount++
	//		}
	//		if err := checkOutputPayload(txn.TxType(), output); err != nil {
	//			return err
	//		}
	//	}
	//}
	//
	//if txn.IsReturnSideChainDepositCoinTx() || txn.IsWithdrawFromSideChainTx() {
	//	return nil
	//}
	//
	//if b.GetHeight() >= b.chainParams.PublicDPOSHeight && specialOutputCount > 1 {
	//	return errors.New("special output count should less equal than 1")
	//}

	return nil
}

func (t *DefaultChecker) isSmallThanMinTransactionFee(fee common.Fixed64) bool {
	if fee < t.contextParameters.Config.MinTransactionFee {
		return true
	}
	return false
}

// validate the type of transaction is allowed or not at current height.
func (t *DefaultChecker) HeightVersionCheck() error {
	return nil
}

func (t *DefaultChecker) IsTxHashDuplicate(txHash common.Uint256) bool {
	return t.contextParameters.BlockChain.GetDB().IsTxHashDuplicate(txHash)
}

func (t *DefaultChecker) GetTxReference(txn interfaces.Transaction) (
	map[*common2.Input]common2.Output, error) {
	return t.contextParameters.BlockChain.UTXOCache.GetTxReference(txn)
}

func (t *DefaultChecker) IsAllowedInPOWConsensus() bool {
	return true
}

func (t *DefaultChecker) CheckTransactionUTXOLock(txn interfaces.Transaction, references map[*common2.Input]common2.Output) error {
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

func (t *DefaultChecker) SpecialContextCheck() (elaerr.ELAError, bool) {
	fmt.Println("default check")
	return nil, false
}

func (t *DefaultChecker) tryCheckVoteOutputs() error {

	txn := t.contextParameters.Transaction
	blockHeight := t.contextParameters.BlockHeight
	state := t.contextParameters.BlockChain.GetState()
	crCommittee := t.contextParameters.BlockChain.GetCRCommittee()

	if txn.Version() >= common2.TxVersion09 {
		producers := state.GetActiveProducers()
		if blockHeight < t.contextParameters.Config.PublicDPOSHeight {
			producers = append(producers, state.GetPendingCanceledProducers()...)
		}
		var candidates []*crstate.Candidate
		if crCommittee.IsInVotingPeriod(blockHeight) {
			candidates = crCommittee.GetCandidates(crstate.Active)
		} else {
			candidates = []*crstate.Candidate{}
		}
		err := t.checkVoteOutputs(blockHeight, txn.Outputs(), t.references,
			getProducerPublicKeysMap(producers), getCRCIDsMap(candidates))
		if err != nil {
			return err
		}
	}
	return nil
}

func getProducerPublicKeysMap(producers []*state.Producer) map[string]struct{} {
	pds := make(map[string]struct{})
	for _, p := range producers {
		pds[common.BytesToHexString(p.Info().OwnerPublicKey)] = struct{}{}
	}
	return pds
}

func getCRCIDsMap(crs []*crstate.Candidate) map[common.Uint168]struct{} {
	codes := make(map[common.Uint168]struct{})
	for _, c := range crs {
		codes[c.Info().CID] = struct{}{}
	}
	return codes
}

func getCRMembersMap(members []*crstate.CRMember) map[string]struct{} {
	crMaps := make(map[string]struct{})
	for _, c := range members {
		crMaps[c.Info.CID.String()] = struct{}{}
	}
	return crMaps
}

func (t *DefaultChecker) checkVoteOutputs(
	blockHeight uint32, outputs []*common2.Output, references map[*common2.Input]common2.Output,
	pds map[string]struct{}, crs map[common.Uint168]struct{}) error {
	programHashes := make(map[common.Uint168]struct{})
	for _, output := range references {
		programHashes[output.ProgramHash] = struct{}{}
	}
	for _, o := range outputs {
		if o.Type != common2.OTVote {
			continue
		}
		if _, ok := programHashes[o.ProgramHash]; !ok {
			return errors.New("the output address of vote tx " +
				"should exist in its input")
		}
		votePayload, ok := o.Payload.(*outputpayload.VoteOutput)
		if !ok {
			return errors.New("invalid vote output payload")
		}
		for _, content := range votePayload.Contents {
			switch content.VoteType {
			case outputpayload.Delegate:
				err := t.checkVoteProducerContent(
					content, pds, votePayload.Version, o.Value)
				if err != nil {
					return err
				}
			case outputpayload.CRC:
				err := t.checkVoteCRContent(blockHeight,
					content, crs, votePayload.Version, o.Value)
				if err != nil {
					return err
				}
			case outputpayload.CRCProposal:
				err := t.checkVoteCRCProposalContent(
					content, votePayload.Version, o.Value)
				if err != nil {
					return err
				}
			case outputpayload.CRCImpeachment:
				err := t.checkCRImpeachmentContent(
					content, votePayload.Version, o.Value)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (t *DefaultChecker) checkCRImpeachmentContent(content outputpayload.VoteContent,
	payloadVersion byte, amount common.Fixed64) error {
	if payloadVersion < outputpayload.VoteProducerAndCRVersion {
		return errors.New("payload VoteProducerVersion not support vote CRCProposal")
	}

	crMembersMap := getCRMembersMap(t.contextParameters.BlockChain.GetCRCommittee().GetImpeachableMembers())
	for _, cv := range content.CandidateVotes {
		if _, ok := crMembersMap[common.BytesToHexString(cv.Candidate)]; !ok {
			return errors.New("candidate should be one of the CR members")
		}
	}

	var totalVotes common.Fixed64
	for _, cv := range content.CandidateVotes {
		totalVotes += cv.Votes
	}
	if totalVotes > amount {
		return errors.New("total votes larger than output amount")
	}
	return nil
}

func (t *DefaultChecker) checkVoteProducerContent(content outputpayload.VoteContent,
	pds map[string]struct{}, payloadVersion byte, amount common.Fixed64) error {
	for _, cv := range content.CandidateVotes {
		if _, ok := pds[common.BytesToHexString(cv.Candidate)]; !ok {
			return fmt.Errorf("invalid vote output payload "+
				"producer candidate: %s", common.BytesToHexString(cv.Candidate))
		}
	}
	if payloadVersion >= outputpayload.VoteProducerAndCRVersion {
		for _, cv := range content.CandidateVotes {
			if cv.Votes > amount {
				return errors.New("votes larger than output amount")
			}
		}
	}

	return nil
}

func (t *DefaultChecker) checkVoteCRContent(blockHeight uint32,
	content outputpayload.VoteContent, crs map[common.Uint168]struct{},
	payloadVersion byte, amount common.Fixed64) error {

	if !t.contextParameters.BlockChain.GetCRCommittee().IsInVotingPeriod(blockHeight) {
		return errors.New("cr vote tx must during voting period")
	}

	if payloadVersion < outputpayload.VoteProducerAndCRVersion {
		return errors.New("payload VoteProducerVersion not support vote CR")
	}
	if blockHeight >= t.contextParameters.Config.CheckVoteCRCountHeight {
		if len(content.CandidateVotes) > outputpayload.MaxVoteProducersPerTransaction {
			return errors.New("invalid count of CR candidates ")
		}
	}
	for _, cv := range content.CandidateVotes {
		cid, err := common.Uint168FromBytes(cv.Candidate)
		if err != nil {
			return fmt.Errorf("invalid vote output payload " +
				"Candidate can not change to proper cid")
		}
		if _, ok := crs[*cid]; !ok {
			return fmt.Errorf("invalid vote output payload "+
				"CR candidate: %s", cid.String())
		}
	}
	var totalVotes common.Fixed64
	for _, cv := range content.CandidateVotes {
		totalVotes += cv.Votes
	}
	if totalVotes > amount {
		return errors.New("total votes larger than output amount")
	}

	return nil
}

func (t *DefaultChecker) checkVoteCRCProposalContent(
	content outputpayload.VoteContent, payloadVersion byte,
	amount common.Fixed64) error {

	if payloadVersion < outputpayload.VoteProducerAndCRVersion {
		return errors.New("payload VoteProducerVersion not support vote CRCProposal")
	}

	for _, cv := range content.CandidateVotes {
		if cv.Votes > amount {
			return errors.New("votes larger than output amount")
		}
		proposalHash, err := common.Uint256FromBytes(cv.Candidate)
		if err != nil {
			return err
		}
		proposal := t.contextParameters.BlockChain.GetCRCommittee().GetProposal(*proposalHash)
		if proposal == nil || proposal.Status != crstate.CRAgreed {
			return fmt.Errorf("invalid CRCProposal: %s",
				common.ToReversedString(*proposalHash))
		}
	}

	return nil
}

func (t *DefaultChecker) checkInvalidUTXO(txn interfaces.Transaction) error {
	currentHeight := blockchain.DefaultLedger.Blockchain.GetHeight()
	for _, input := range txn.Inputs() {
		referTxn, err := t.contextParameters.BlockChain.UTXOCache.GetTransaction(input.Previous.TxID)
		if err != nil {
			return err
		}
		if referTxn.IsCoinBaseTx() {
			if currentHeight-referTxn.LockTime() < t.contextParameters.Config.CoinbaseMaturity {
				return errors.New("the utxo of coinbase is locking")
			}
		} else if referTxn.IsNewSideChainPowTx() {
			return errors.New("cannot spend the utxo from t new sideChainPow tx")
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

func checkTransactionDepositOutputs(bc *blockchain.BlockChain, txn interfaces.Transaction) error {
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

func (t *DefaultChecker) checkTransactionFee(tx interfaces.Transaction, references map[*common2.Input]common2.Output) error {
	fee := getTransactionFee(tx, references)
	if t.isSmallThanMinTransactionFee(fee) {
		return fmt.Errorf("transaction fee not enough")
	}
	// set Fee and FeePerKB if check has passed
	tx.SetFee(fee)
	buf := new(bytes.Buffer)
	tx.Serialize(buf)
	tx.SetFeePerKB(fee * 1000 / common.Fixed64(len(buf.Bytes())))
	return nil
}
