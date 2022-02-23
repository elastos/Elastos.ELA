// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	"fmt"
	"math"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/dpos/state"
	"github.com/elastos/Elastos.ELA/elanet/pact"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type DefaultChecker struct {
	// config
	parameters *TransactionParameters
	references map[*common2.Input]common2.Output
}

func (t *DefaultChecker) SanityCheck(params interfaces.Parameters) elaerr.ELAError {
	if err := t.SetParameters(params); err != nil {
		return elaerr.Simple(elaerr.ErrFail, errors.New("invalid parameters"))
	}

	if err := t.parameters.Transaction.HeightVersionCheck(); err != nil {
		log.Warn("[HeightVersionCheck],", err)
		return elaerr.Simple(elaerr.ErrTxHeightVersion, nil)
	}

	if err := t.parameters.Transaction.CheckTransactionSize(); err != nil {
		log.Warn("[CheckTransactionSize],", err)
		return elaerr.Simple(elaerr.ErrTxSize, err)
	}

	if err := t.parameters.Transaction.CheckTransactionInput(); err != nil {
		log.Warn("[CheckTransactionInput],", err)
		return elaerr.Simple(elaerr.ErrTxInvalidInput, err)
	}

	if err := t.parameters.Transaction.CheckTransactionOutput(); err != nil {
		log.Warn("[CheckTransactionOutput],", err)
		return elaerr.Simple(elaerr.ErrTxInvalidOutput, err)
	}

	if err := checkAssetPrecision(t.parameters.Transaction); err != nil {
		log.Warn("[CheckAssetPrecesion],", err)
		return elaerr.Simple(elaerr.ErrTxAssetPrecision, err)
	}

	if err := t.parameters.Transaction.CheckAttributeProgram(); err != nil {
		log.Warn("[CheckAttributeProgram],", err)
		return elaerr.Simple(elaerr.ErrTxAttributeProgram, err)
	}

	if err := t.parameters.Transaction.CheckTransactionPayload(); err != nil {
		log.Warn("[CheckTransactionPayload],", err)
		return elaerr.Simple(elaerr.ErrTxPayload, err)
	}

	if err := blockchain.CheckDuplicateSidechainTx(t.parameters.Transaction); err != nil {
		log.Warn("[CheckDuplicateSidechainTx],", err)
		return elaerr.Simple(elaerr.ErrTxSidechainDuplicate, err)
	}

	return nil
}

func (t *DefaultChecker) ContextCheck(params interfaces.Parameters) (
	map[*common2.Input]common2.Output, elaerr.ELAError) {

	if err := t.SetParameters(params); err != nil {
		log.Warn("[CheckTransactionContext] set parameters failed.")
		return nil, elaerr.Simple(elaerr.ErrTxDuplicate, errors.New("invalid parameters"))
	}

	if err := t.parameters.Transaction.HeightVersionCheck(); err != nil {
		log.Warn("[CheckTransactionContext] height version check failed.")
		return nil, elaerr.Simple(elaerr.ErrTxHeightVersion, nil)
	}

	if exist := t.IsTxHashDuplicate(t.parameters.Transaction.Hash()); exist {
		log.Warn("[CheckTransactionContext] duplicate transaction check failed.")
		return nil, elaerr.Simple(elaerr.ErrTxDuplicate, nil)
	}

	references, err := t.GetTxReference(t.parameters.Transaction)
	if err != nil {
		log.Warn("[CheckTransactionContext] get transaction reference failed")
		return nil, elaerr.Simple(elaerr.ErrTxUnknownReferredTx, nil)
	}
	t.references = references

	if t.parameters.BlockChain.GetState().GetConsensusAlgorithm() == state.POW {
		if !t.parameters.Transaction.IsAllowedInPOWConsensus() {
			log.Warnf("[CheckTransactionContext], %s transaction is not allowed in POW", t.parameters.Transaction.TxType().Name())
			return nil, elaerr.Simple(elaerr.ErrTxValidation, nil)
		}
	}

	// check double spent transaction
	if blockchain.DefaultLedger.IsDoubleSpend(t.parameters.Transaction) {
		log.Warn("[CheckTransactionContext] IsDoubleSpend check failed")
		return nil, elaerr.Simple(elaerr.ErrTxDoubleSpend, nil)
	}

	if err := t.CheckTransactionUTXOLock(t.parameters.Transaction, references); err != nil {
		log.Warn("[CheckTransactionUTXOLock],", err)
		return nil, elaerr.Simple(elaerr.ErrTxUTXOLocked, err)
	}

	cerr, end := t.parameters.Transaction.SpecialContextCheck()
	if end {
		return references, cerr
	}

	if err := t.checkTransactionFee(t.parameters.Transaction, references); err != nil {
		log.Warn("[CheckTransactionFee],", err)
		return nil, elaerr.Simple(elaerr.ErrTxBalance, err)
	}

	if err := checkDestructionAddress(references); err != nil {
		log.Warn("[CheckDestructionAddress], ", err)
		return nil, elaerr.Simple(elaerr.ErrTxInvalidInput, err)
	}

	if err := checkTransactionDepositUTXO(t.parameters.Transaction, references); err != nil {
		log.Warn("[CheckTransactionDepositUTXO],", err)
		return nil, elaerr.Simple(elaerr.ErrTxInvalidInput, err)
	}

	if err := checkTransactionDepositOutputs(t.parameters.BlockChain, t.parameters.Transaction); err != nil {
		log.Warn("[checkTransactionDepositOutputs],", err)
		return nil, elaerr.Simple(elaerr.ErrTxInvalidInput, err)
	}

	if err := checkTransactionSignature(t.parameters.Transaction, references); err != nil {
		log.Warn("[checkTransactionSignature],", err)
		return nil, elaerr.Simple(elaerr.ErrTxSignature, err)
	}

	if err := t.checkInvalidUTXO(t.parameters.Transaction); err != nil {
		log.Warn("[checkInvalidUTXO]", err)
		return nil, elaerr.Simple(elaerr.ErrBlockIneffectiveCoinbase, err)
	}

	if err := t.tryCheckVoteOutputs(); err != nil {
		log.Warn("[tryCheckVoteOutputs]", err)
		return nil, elaerr.Simple(elaerr.ErrTxInvalidOutput, err)
	}

	return references, nil
}

func (t *DefaultChecker) SetParameters(params interface{}) elaerr.ELAError {
	var ok bool
	if t.parameters, ok = params.(*TransactionParameters); !ok {
		return elaerr.Simple(elaerr.ErrTxDuplicate, errors.New("invalid parameters"))
	}

	return nil
}

func (t *DefaultChecker) CheckTransactionSize() error {
	size := t.parameters.Transaction.GetSize()
	if size <= 0 || size > int(pact.MaxBlockContextSize) {
		return fmt.Errorf("Invalid transaction size: %d bytes", size)
	}

	return nil
}

//validate the transaction of duplicate UTXO input
func (t *DefaultChecker) CheckTransactionInput() error {
	txn := t.parameters.Transaction
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

func (t *DefaultChecker) CheckTransactionOutput() error {

	txn := t.parameters.Transaction
	blockHeight := t.parameters.BlockHeight
	// check outputs count
	if len(txn.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}
	if len(txn.Outputs()) < 1 {
		return errors.New("transaction has no outputs")
	}

	// check if output address is valid
	specialOutputCount := 0
	for _, output := range txn.Outputs() {
		if output.AssetID != config.ELAAssetID {
			return errors.New("asset ID in output is invalid")
		}

		// output value must >= 0
		if output.Value < common.Fixed64(0) {
			return errors.New("invalid transaction UTXO output")
		}

		if err := checkOutputProgramHash(blockHeight, output.ProgramHash); err != nil {
			return err
		}

		if txn.Version() >= common2.TxVersion09 {
			if output.Type != common2.OTNone {
				specialOutputCount++
			}
			if err := checkOutputPayload(output); err != nil {
				return err
			}
		}
	}

	if blockHeight >= t.parameters.Config.PublicDPOSHeight && specialOutputCount > 1 {
		return errors.New("special output count should less equal than 1")
	}

	return nil
}

func (t *DefaultChecker) CheckAttributeProgram() error {
	tx := t.parameters.Transaction
	// Check attributes
	for _, attr := range tx.Attributes() {
		if !common2.IsValidAttributeType(attr.Usage) {
			return fmt.Errorf("invalid attribute usage %v", attr.Usage)
		}
	}

	// Check programs
	if len(tx.Programs()) == 0 {
		return fmt.Errorf("no programs found in transaction")
	}
	for _, program := range tx.Programs() {
		if program.Code == nil {
			return fmt.Errorf("invalid program code nil")
		}
		if program.Parameter == nil {
			return fmt.Errorf("invalid program parameter nil")
		}
	}
	return nil
}

func (t *DefaultChecker) CheckTransactionPayload() error {
	return errors.New("invalid payload type")
}

func (t *DefaultChecker) isSmallThanMinTransactionFee(fee common.Fixed64) bool {
	if fee < t.parameters.Config.MinTransactionFee {
		return true
	}
	return false
}

// validate the type of transaction is allowed or not at current height.
func (t *DefaultChecker) HeightVersionCheck() error {
	return nil
}

func (t *DefaultChecker) IsTxHashDuplicate(txHash common.Uint256) bool {
	return t.parameters.BlockChain.GetDB().IsTxHashDuplicate(txHash)
}

func (t *DefaultChecker) GetTxReference(txn interfaces.Transaction) (
	map[*common2.Input]common2.Output, error) {
	return t.parameters.BlockChain.UTXOCache.GetTxReference(txn)
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
	return nil, false
}

func (t *DefaultChecker) tryCheckVoteOutputs() error {

	txn := t.parameters.Transaction
	blockHeight := t.parameters.BlockHeight
	dposState := t.parameters.BlockChain.GetState()
	crCommittee := t.parameters.BlockChain.GetCRCommittee()

	if txn.Version() >= common2.TxVersion09 {
		producers := dposState.GetActiveProducers()
		if blockHeight < t.parameters.Config.PublicDPOSHeight {
			producers = append(producers, dposState.GetPendingCanceledProducers()...)
		}
		var candidates []*crstate.Candidate
		if crCommittee.IsInVotingPeriod(blockHeight) {
			candidates = crCommittee.GetCandidates(crstate.Active)
		} else {
			candidates = []*crstate.Candidate{}
		}
		err := t.checkVoteOutputs(blockHeight, txn.Outputs(), t.references,
			getProducerPublicKeysMap(producers),
			getDPoSV2ProducersMap(t.parameters.BlockChain.GetState().GetActivityV2Producers()),
			getCRCIDsMap(candidates))
		if err != nil {
			return err
		}
	}

	if txn.TxType() == common2.Voting {

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

func getDPoSV2ProducersMap(producers []*state.Producer) map[string]uint32 {
	pds := make(map[string]uint32)
	for _, p := range producers {
		pds[common.BytesToHexString(p.Info().OwnerPublicKey)] = p.Info().StakeUntil
	}
	return pds
}

func getCRCIDsMap(crs []*crstate.Candidate) map[common.Uint168]struct{} {
	codes := make(map[common.Uint168]struct{})
	for _, c := range crs {
		codes[c.Info.CID] = struct{}{}
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
	pds map[string]struct{}, pds2 map[string]uint32, crs map[common.Uint168]struct{}) error {
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

	crMembersMap := getCRMembersMap(t.parameters.BlockChain.GetCRCommittee().GetImpeachableMembers())
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

	if !t.parameters.BlockChain.GetCRCommittee().IsInVotingPeriod(blockHeight) {
		return errors.New("cr vote tx must during voting period")
	}

	if payloadVersion < outputpayload.VoteProducerAndCRVersion {
		return errors.New("payload VoteProducerVersion not support vote CR")
	}
	if blockHeight >= t.parameters.Config.CheckVoteCRCountHeight {
		if len(content.CandidateVotes) > outputpayload.MaxVoteProducersPerTransaction {
			return errors.New("invalid count of CR candidates ")
		}
	}
	for _, cv := range content.CandidateVotes {
		cid, err := common.Uint168FromBytes(cv.Candidate)
		if err != nil {
			return fmt.Errorf("invalid vote output payload " +
				"StakeAddress can not change to proper cid")
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
		proposal := t.parameters.BlockChain.GetCRCommittee().GetProposal(*proposalHash)
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
		referTxn, err := t.parameters.BlockChain.UTXOCache.GetTransaction(input.Previous.TxID)
		if err != nil {
			return err
		}
		if referTxn.IsCoinBaseTx() {
			if currentHeight-referTxn.LockTime() < t.parameters.Config.CoinbaseMaturity {
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
		tx.IsCRAssetsRectifyTx() || tx.IsCRCProposalRealWithdrawTx() || tx.IsNextTurnDPOSInfoTx() ||
		tx.IsDposV2ClaimRewardRealWithdraw() || tx.IsUnstakeRealWithdrawTX() {
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

func checkOutputProgramHash(height uint32, programHash common.Uint168) error {
	// main version >= 88812
	if height >= config.DefaultParams.CheckAddressHeight {
		var empty = common.Uint168{}
		if programHash.IsEqual(empty) {
			return nil
		}
		if programHash.IsEqual(config.CRAssetsAddress) {
			return nil
		}
		if programHash.IsEqual(config.CRCExpensesAddress) {
			return nil
		}

		prefix := contract.PrefixType(programHash[0])
		switch prefix {
		case contract.PrefixStandard:
		case contract.PrefixMultiSig:
		case contract.PrefixCrossChain:
		case contract.PrefixDeposit:
		default:
			return errors.New("invalid program hash prefix")
		}

		addr, err := programHash.ToAddress()
		if err != nil {
			return errors.New("invalid program hash")
		}
		_, err = common.Uint168FromAddress(addr)
		if err != nil {
			return errors.New("invalid program hash")
		}

		return nil
	}

	// old version [0, 88812)
	return nil
}

func checkOutputPayload(output *common2.Output) error {
	switch output.Type {
	case common2.OTNone:
	default:
		return errors.New("transaction type dose not match the output payload type")
	}

	return output.Payload.Validate()
}

func checkAssetPrecision(txn interfaces.Transaction) error {
	for _, output := range txn.Outputs() {
		if !checkAmountPrecise(output.Value, config.ELAPrecision) {
			return errors.New("the precision of asset is incorrect")
		}
	}
	return nil
}

func checkAmountPrecise(amount common.Fixed64, precision byte) bool {
	return amount.IntValue()%int64(math.Pow(10, float64(8-precision))) == 0
}
