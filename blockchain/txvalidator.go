// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package blockchain

import (
	"bytes"
	"errors"
	"fmt"
	"math"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
	. "github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
	"github.com/elastos/Elastos.ELA/elanet/pact"
	elaerr "github.com/elastos/Elastos.ELA/errors"
	"github.com/elastos/Elastos.ELA/vm"
)

const (
	// MaxStringLength is the maximum length of a string field.
	MaxStringLength = 100

	// Category Data length limit not exceeding 4096 characters
	MaxCategoryDataStringLength = 4096

	// Category Data length limit not exceeding 4096 characters
	MaxDraftDataStringLength = 1000000

	// MaxBudgetsCount indicates max budgets count of one proposal.
	MaxBudgetsCount = 128

	// ELIPBudgetsCount indicates budgets count of ELIP.
	ELIPBudgetsCount = 2

	// CRCProposalBudgetsPercentage indicates the percentage of the CRC member
	// address balance available for a single proposal budget.
	CRCProposalBudgetsPercentage = 10
)

type TransactionChecker interface {
	CheckTransactionSanity(blockHeight uint32, txn interfaces.Transaction) elaerr.ELAError
	CheckTransactionContext(blockHeight uint32, txn interfaces.Transaction,
		proposalsUsedAmount common.Fixed64, timeStamp uint32) (
		map[*common2.Input]common2.Output, elaerr.ELAError)
}

type BaseChecker struct {
	BlockChain
}

// CheckTransactionSanity verifies received single transaction
func (b *BlockChain) CheckTransactionSanity(blockHeight uint32,
	txn interfaces.Transaction) elaerr.ELAError {

	para := functions.GetTransactionParameters(
		txn, blockHeight, 0, b.chainParams, b, 0)

	return txn.SanityCheck(para)
}

// CheckTransactionContext verifies a transaction with history transaction in ledger
func (b *BlockChain) CheckTransactionContext(blockHeight uint32,
	tx interfaces.Transaction, proposalsUsedAmount common.Fixed64, timeStamp uint32) (
	map[*common2.Input]common2.Output, elaerr.ELAError) {

	para := functions.GetTransactionParameters(
		tx, blockHeight, timeStamp, b.chainParams, b, proposalsUsedAmount)

	references, contextErr := tx.ContextCheck(para)
	if contextErr != nil {
		return nil, contextErr
	}

	return references, nil
}

func (b *BlockChain) CheckVoteOutputs(
	blockHeight uint32, outputs []*common2.Output, references map[*common2.Input]common2.Output,
	pds map[string]struct{}, pds2 map[string]uint32, crs map[common.Uint168]struct{}) error {
	programHashes := make(map[common.Uint168]struct{})
	for _, output := range references {
		programHashes[output.ProgramHash] = struct{}{}
	}

	var dposV2OutputCount int
	var dposV2OutputLock uint32
	var totalDPoSV2OutputVotes common.Fixed64
	for _, o := range outputs {
		if o.Type != common2.OTVote && o.Type != common2.OTDposV2Vote {
			continue
		}
		var checkProhash common.Uint168
		if o.Type == common2.OTDposV2Vote {
			checkProhash = common.Uint168FromCodeHash(byte(contract.PrefixStandard), o.ProgramHash.ToCodeHash())
		} else {
			checkProhash = o.ProgramHash
		}

		if _, ok := programHashes[checkProhash]; !ok {
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
				err := b.checkVoteProducerContent(
					content, pds, votePayload.Version, o.Value)
				if err != nil {
					return err
				}
			case outputpayload.CRC:
				err := b.checkVoteCRContent(blockHeight,
					content, crs, votePayload.Version, o.Value)
				if err != nil {
					return err
				}
			case outputpayload.CRCProposal:
				err := b.checkVoteCRCProposalContent(
					content, votePayload.Version, o.Value)
				if err != nil {
					return err
				}
			case outputpayload.CRCImpeachment:
				err := b.checkCRImpeachmentContent(
					content, votePayload.Version, o.Value)
				if err != nil {
					return err
				}
			case outputpayload.DposV2:
				dposV2OutputCount++
				dposV2OutputLock = o.OutputLock
				totalDPoSV2OutputVotes += o.Value
				err := b.checkVoteDposV2Content(o.OutputLock,
					content, pds2, votePayload.Version, o.Value)
				if err != nil {
					return err
				}
			}
		}
	}

	// If Inputs contain DPoS v2 votes, need to check:
	// 1.need to be only one DPoS V2 input
	// 2.need to be only one DPoS V2 output
	// 3.outputLock of output need to be bigger than new input
	// 4.DPoS v2 votes in outputs need to be more than Inputs
	var dposV2InputCount uint32
	var dposV2InputLock uint32
	var totalDPoSV2InputVotes common.Fixed64
	for _, o := range references {
		votePayload, ok := o.Payload.(*outputpayload.VoteOutput)
		if !ok {
			continue
		}

		var containDPoSV2Votes bool
		for _, content := range votePayload.Contents {
			if content.VoteType == outputpayload.DposV2 {
				containDPoSV2Votes = true
				break
			}
		}
		if !containDPoSV2Votes {
			continue
		}

		dposV2InputCount++
		dposV2InputLock = o.OutputLock
		totalDPoSV2InputVotes += o.Value
	}
	// need to be only one DPoS V2 input
	if dposV2InputCount > 1 {
		return errors.New("need to be only one DPoS V2 input")
	}
	// need to be only one DPoS V2 output
	if dposV2OutputCount > 1 {
		return errors.New("need to be only one DPoS V2 output")
	}
	// outputLock of output need to be bigger than new input
	if dposV2InputLock > dposV2OutputLock {
		return errors.New(fmt.Sprintf("invalid DPoS V2 output lock, "+
			"need to be bigger than input, input lockTime:%d, "+
			"output lockTime:%d", dposV2InputLock, dposV2OutputLock))
	}
	// DPoS v2 votes in outputs need to be more than Inputs
	if totalDPoSV2InputVotes > totalDPoSV2OutputVotes {
		return errors.New(fmt.Sprintf("invalid DPoS V2 output votes, "+
			"need to be bigger than input, input votes:%d, "+
			"output votes:%d", totalDPoSV2InputVotes, totalDPoSV2OutputVotes))
	}

	return nil
}

func (b *BlockChain) checkCRImpeachmentContent(content outputpayload.VoteContent,
	payloadVersion byte, amount common.Fixed64) error {
	if payloadVersion < outputpayload.VoteProducerAndCRVersion {
		return errors.New("payload VoteProducerVersion not support vote CRCProposal")
	}

	crMembersMap := getCRMembersMap(b.crCommittee.GetImpeachableMembers())
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

func (b *BlockChain) checkVoteProducerContent(content outputpayload.VoteContent,
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

func (b *BlockChain) checkVoteDposV2Content(lockTime uint32, content outputpayload.VoteContent,
	pds map[string]uint32, payloadVersion byte, amount common.Fixed64) error {
	for _, cv := range content.CandidateVotes {
		stakeUntil, ok := pds[common.BytesToHexString(cv.Candidate)]
		if !ok {
			return fmt.Errorf("invalid vote output payload "+
				"producer candidate: %s", common.BytesToHexString(cv.Candidate))
		}

		if lockTime > stakeUntil {
			return fmt.Errorf("invalid vote output lockTime "+
				"producer candidate:%s, lockTime:%d, stakeUntil:%d",
				common.BytesToHexString(cv.Candidate), lockTime, stakeUntil)
		}
	}

	if payloadVersion < outputpayload.VoteDposV2Version {
		return errors.New("payload VoteDposV2Version not support vote DposV2")
	}
	if len(content.CandidateVotes) > outputpayload.MaxDposV2ProducerPerTransaction {
		return errors.New("invalid count of DposV2 candidates ")
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

func (b *BlockChain) checkVoteCRContent(blockHeight uint32,
	content outputpayload.VoteContent, crs map[common.Uint168]struct{},
	payloadVersion byte, amount common.Fixed64) error {

	if !b.crCommittee.IsInVotingPeriod(blockHeight) {
		return errors.New("cr vote tx must during voting period")
	}

	if payloadVersion < outputpayload.VoteProducerAndCRVersion {
		return errors.New("payload VoteProducerVersion not support vote CR")
	}
	if blockHeight >= b.chainParams.CRConfiguration.CheckVoteCRCountHeight {
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

func (b *BlockChain) checkVoteCRCProposalContent(
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
		proposal := b.crCommittee.GetProposal(*proposalHash)
		if proposal == nil || proposal.Status != crstate.CRAgreed {
			return fmt.Errorf("invalid CRCProposal: %s",
				common.ToReversedString(*proposalHash))
		}
	}

	return nil
}

func getCRMembersMap(members []*crstate.CRMember) map[string]struct{} {
	crMaps := make(map[string]struct{})
	for _, c := range members {
		crMaps[c.Info.CID.String()] = struct{}{}
	}
	return crMaps
}

func CheckDestructionAddress(references map[*common2.Input]common2.Output) error {
	for _, output := range references {
		if output.ProgramHash == *config.DestroyELAProgramHash {
			return errors.New("cannot use utxo from the destruction address")
		}
	}
	return nil
}

// validate the transaction of duplicate UTXO input
func CheckTransactionInput(txn interfaces.Transaction) error {
	if txn.IsCoinBaseTx() {
		if len(txn.Inputs()) != 1 {
			return errors.New("coinbase must has only one input")
		}
		inputHash := txn.Inputs()[0].Previous.TxID
		inputIndex := txn.Inputs()[0].Previous.Index
		sequence := txn.Inputs()[0].Sequence
		if !inputHash.IsEqual(common.EmptyHash) ||
			inputIndex != math.MaxUint16 || sequence != math.MaxUint32 {
			return errors.New("invalid coinbase input")
		}

		return nil
	}

	if txn.IsIllegalTypeTx() || txn.IsInactiveArbitrators() ||
		txn.IsNewSideChainPowTx() || txn.IsUpdateVersion() ||
		txn.IsActivateProducerTx() || txn.IsNextTurnDPOSInfoTx() ||
		txn.IsRevertToPOW() || txn.IsRevertToDPOS() || txn.IsCustomIDResultTx() ||
		txn.IsDposV2ClaimRewardTx() {
		if len(txn.Inputs()) != 0 {
			return errors.New("no cost transactions must has no input")
		}
		return nil
	}

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

func (b *BlockChain) CheckTransactionOutput(txn interfaces.Transaction,
	blockHeight uint32) error {
	if len(txn.Outputs()) > math.MaxUint16 {
		return errors.New("output count should not be greater than 65535(MaxUint16)")
	}

	if txn.IsCoinBaseTx() {
		if len(txn.Outputs()) < 2 {
			return errors.New("coinbase output is not enough, at least 2")
		}

		foundationReward := txn.Outputs()[0].Value
		var totalReward = common.Fixed64(0)
		if blockHeight < b.chainParams.PublicDPOSHeight {
			for _, output := range txn.Outputs() {
				if output.AssetID != core.ELAAssetID {
					return errors.New("asset ID in coinbase is invalid")
				}
				totalReward += output.Value
			}

			if foundationReward < common.Fixed64(float64(totalReward)*0.3) {
				return errors.New("reward to foundation in coinbase < 30%")
			}
		} else {
			// check the ratio of FoundationAddress reward with miner reward
			totalReward = txn.Outputs()[0].Value + txn.Outputs()[1].Value
			if len(txn.Outputs()) == 2 && foundationReward <
				common.Fixed64(float64(totalReward)*0.3/0.65) {
				return errors.New("reward to foundation in coinbase < 30%")
			}
		}

		return nil
	}

	if txn.IsIllegalTypeTx() || txn.IsInactiveArbitrators() ||
		txn.IsUpdateVersion() || txn.IsActivateProducerTx() ||
		txn.IsNextTurnDPOSInfoTx() || txn.IsRevertToPOW() ||
		txn.IsRevertToDPOS() || txn.IsCustomIDResultTx() || txn.IsDposV2ClaimRewardTx() {
		if len(txn.Outputs()) != 0 {
			return errors.New("no cost transactions should have no output")
		}

		return nil
	}

	if txn.IsCRCAppropriationTx() {
		if len(txn.Outputs()) != 2 {
			return errors.New("new CRCAppropriation tx must have two output")
		}
		if !txn.Outputs()[0].ProgramHash.IsEqual(*b.chainParams.CRConfiguration.CRExpensesProgramHash) {
			return errors.New("new CRCAppropriation tx must have the first" +
				"output to CR expenses address")
		}
		if !txn.Outputs()[1].ProgramHash.IsEqual(*b.chainParams.CRConfiguration.CRAssetsProgramHash) {
			return errors.New("new CRCAppropriation tx must have the second" +
				"output to CR assets address")
		}
	}

	if txn.IsNewSideChainPowTx() {
		if len(txn.Outputs()) != 1 {
			return errors.New("new sideChainPow tx must have only one output")
		}
		if txn.Outputs()[0].Value != 0 {
			return errors.New("the value of new sideChainPow tx output must be 0")
		}
		if txn.Outputs()[0].Type != common2.OTNone {
			return errors.New("the type of new sideChainPow tx output must be OTNone")
		}
		return nil
	}

	if len(txn.Outputs()) < 1 {
		return errors.New("transaction has no outputs")
	}

	// check if output address is valid
	specialOutputCount := 0
	for _, output := range txn.Outputs() {
		if output.AssetID != core.ELAAssetID {
			return errors.New("asset ID in output is invalid")
		}

		// output value must >= 0
		if output.Value < common.Fixed64(0) {
			return errors.New("invalid transaction UTXO output")
		}

		if err := CheckOutputProgramHash(blockHeight, output.ProgramHash); err != nil {
			return err
		}

		if txn.Version() >= common2.TxVersion09 {
			if output.Type != common2.OTNone {
				specialOutputCount++
			}
			if err := CheckOutputPayload(txn.TxType(), output); err != nil {
				return err
			}
		}
	}

	if txn.IsReturnSideChainDepositCoinTx() || txn.IsWithdrawFromSideChainTx() {
		return nil
	}

	if b.GetHeight() >= b.chainParams.PublicDPOSHeight && specialOutputCount > 1 {
		return errors.New("special output count should less equal than 1")
	}

	return nil
}

func CheckOutputProgramHash(height uint32, programHash common.Uint168) error {
	// main version >= 88812
	if height >= config.DefaultParams.CheckAddressHeight {
		var empty = common.Uint168{}
		if programHash.IsEqual(empty) {
			return nil
		}
		if programHash.IsEqual(*config.CRAssetsProgramHash) {
			return nil
		}
		if programHash.IsEqual(*config.CRCExpensesProgramHash) {
			return nil
		}

		prefix := contract.PrefixType(programHash[0])
		switch prefix {
		case contract.PrefixStandard:
		case contract.PrefixMultiSig:
		case contract.PrefixCrossChain:
		case contract.PrefixDeposit:
		case contract.PrefixDPoSV2:
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

func CheckOutputPayload(txType common2.TxType, output *common2.Output) error {
	switch txType {
	case common2.ReturnSideChainDepositCoin:
		switch output.Type {
		case common2.OTNone:
		case common2.OTReturnSideChainDepositCoin:
		default:
			return errors.New("transaction type dose not match the output payload type")
		}
	case common2.WithdrawFromSideChain:
		switch output.Type {
		case common2.OTNone:
		case common2.OTWithdrawFromSideChain:
		default:
			return errors.New("transaction type dose not match the output payload type")
		}
	case common2.TransferCrossChainAsset:
		// common2.OTCrossChain information can only be placed in TransferCrossChainAsset transaction.
		switch output.Type {
		case common2.OTNone:
		case common2.OTCrossChain:
		default:
			return errors.New("transaction type dose not match the output payload type")
		}
	case common2.TransferAsset:
		// common2.OTVote information can only be placed in TransferAsset transaction.
		switch output.Type {
		case common2.OTVote:
			prefix := contract.GetPrefixType(output.ProgramHash)
			if prefix != contract.PrefixStandard && prefix != contract.PrefixMultiSig {
				return errors.New("output address should be standard")
			}
		case common2.OTNone:
		case common2.OTMapping:
		case common2.OTDposV2Vote:
			if contract.GetPrefixType(output.ProgramHash) !=
				contract.PrefixDPoSV2 {
				return errors.New("output address should be dposV2")
			}
		default:
			return errors.New("transaction type dose not match the output payload type")
		}
	default:
		switch output.Type {
		case common2.OTNone:
		default:
			return errors.New("transaction type dose not match the output payload type")
		}
	}

	return output.Payload.Validate()
}

func CheckTransactionDepositUTXO(txn interfaces.Transaction, references map[*common2.Input]common2.Output) error {
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

func CheckTransactionSize(txn interfaces.Transaction) error {
	size := txn.GetSize()
	if size <= 0 || size > int(pact.MaxBlockContextSize) {
		return fmt.Errorf("Invalid transaction size: %d bytes", size)
	}

	return nil
}

func CheckAssetPrecision(txn interfaces.Transaction) error {
	for _, output := range txn.Outputs() {
		if !CheckAmountPrecise(output.Value, core.ELAPrecision) {
			return errors.New("the precision of asset is incorrect")
		}
	}
	return nil
}

func (b *BlockChain) getTransactionFee(tx interfaces.Transaction,
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

func (b *BlockChain) isSmallThanMinTransactionFee(fee common.Fixed64) bool {
	if fee < b.chainParams.MinTransactionFee {
		return true
	}
	return false
}

func (b *BlockChain) CheckTransactionFee(tx interfaces.Transaction, references map[*common2.Input]common2.Output) error {
	fee := b.getTransactionFee(tx, references)
	if b.isSmallThanMinTransactionFee(fee) {
		return fmt.Errorf("transaction fee not enough")
	}
	// set Fee and FeePerKB if check has passed
	tx.SetFee(fee)
	buf := new(bytes.Buffer)
	tx.Serialize(buf)
	tx.SetFeePerKB(fee * 1000 / common.Fixed64(len(buf.Bytes())))
	return nil
}

func checkTransactionSignature(tx interfaces.Transaction, references map[*common2.Input]common2.Output) error {
	programHashes, err := GetTxProgramHashes(tx, references)
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
	SortPrograms(tx.Programs())
	return RunPrograms(buf.Bytes(), programHashes, tx.Programs())
}

func CheckAmountPrecise(amount common.Fixed64, precision byte) bool {
	return amount.IntValue()%int64(math.Pow(10, float64(8-precision))) == 0
}

// validate the transaction of duplicate sidechain transaction
func CheckDuplicateSidechainTx(txn interfaces.Transaction) error {
	if txn.IsWithdrawFromSideChainTx() {
		witPayload := txn.Payload().(*payload.WithdrawFromSideChain)
		existingHashs := make(map[common.Uint256]struct{})
		for _, hash := range witPayload.SideChainTransactionHashes {
			if _, exist := existingHashs[hash]; exist {
				return errors.New("Duplicate sidechain tx detected in a transaction")
			}
			existingHashs[hash] = struct{}{}
		}
	}
	return nil
}

func CheckSideChainPowConsensus(txn interfaces.Transaction, arbitrator []byte) error {
	payloadSideChainPow, ok := txn.Payload().(*payload.SideChainPow)
	if !ok {
		return errors.New("side mining transaction has invalid payload")
	}

	if arbitrator == nil {
		return errors.New("there is no arbiter on duty")
	}

	publicKey, err := DecodePoint(arbitrator)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	err = payloadSideChainPow.Serialize(buf, payload.SideChainPowVersion)
	if err != nil {
		return err
	}

	err = Verify(*publicKey, buf.Bytes()[0:68], payloadSideChainPow.Signature)
	if err != nil {
		return errors.New("Arbitrator is not matched. " + err.Error())
	}

	return nil
}

func GetDIDFromCode(code []byte) (*common.Uint168, error) {
	newCode := make([]byte, len(code))
	copy(newCode, code)
	didCode := append(newCode[:len(newCode)-1], common.DID)

	if ct1, err := contract.CreateCRIDContractByCode(didCode); err != nil {
		return nil, err
	} else {
		return ct1.ToProgramHash(), nil
	}
}

func getCode(publicKey []byte) ([]byte, error) {
	if pk, err := crypto.DecodePoint(publicKey); err != nil {
		return nil, err
	} else {
		if redeemScript, err := contract.CreateStandardRedeemScript(pk); err != nil {
			return nil, err
		} else {
			return redeemScript, nil
		}
	}
}

func GetDiDFromPublicKey(publicKey []byte) (*common.Uint168, error) {
	if code, err := getCode(publicKey); err != nil {
		return nil, err
	} else {
		return GetDIDFromCode(code)
	}
}

func getParameterBySignature(signature []byte) []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(byte(len(signature)))
	buf.Write(signature)
	return buf.Bytes()
}

func CheckReturnVotesTransactionSignature(signature []byte, code []byte, data []byte) error {
	signType, err := crypto.GetScriptType(code)
	if err != nil {
		return errors.New("invalid code")
	}
	if signType == vm.CHECKSIG {
		// check code and signature
		if err := CheckStandardSignature(program.Program{
			Code:      code,
			Parameter: getParameterBySignature(signature),
		}, data); err != nil {
			return err
		}
	} else if signType == vm.CHECKMULTISIG {
		// check code and signature
		if err := CheckMultiSigSignatures(program.Program{
			Code:      code,
			Parameter: signature,
		}, data); err != nil {
			return err
		}
	} else {
		return errors.New("invalid code type")
	}

	return nil
}

func CheckCRTransactionSignature(signature []byte, code []byte, data []byte) error {
	signType, err := crypto.GetScriptType(code)
	if err != nil {
		return errors.New("invalid code")
	}
	if signType == vm.CHECKSIG {
		// check code and signature
		if err := CheckStandardSignature(program.Program{
			Code:      code,
			Parameter: getParameterBySignature(signature),
		}, data); err != nil {
			return err
		}
	} else if signType == vm.CHECKMULTISIG {
		//todo  add compatible height
		if err := CheckMultiSigSignatures(program.Program{
			Code:      code,
			Parameter: signature,
		}, data); err != nil {
			return err
		}
	} else {
		return errors.New("invalid code type")
	}

	return nil
}

func CheckPayloadSignature(info *payload.CRInfo, payloadVersion byte) error {
	signedBuf := new(bytes.Buffer)
	err := info.SerializeUnsigned(signedBuf, payloadVersion)
	if err != nil {
		return err
	}
	return CheckCRTransactionSignature(info.Signature, info.Code, signedBuf.Bytes())
}

func CheckRevertToDPOSTransaction(txn interfaces.Transaction) error {
	return checkArbitratorsSignatures(txn.Programs()[0])
}

func CheckSidechainIllegalEvidence(p *payload.SidechainIllegalData) error {

	if p.IllegalType != payload.SidechainIllegalProposal &&
		p.IllegalType != payload.SidechainIllegalVote {
		return errors.New("invalid type")
	}

	_, err := crypto.DecodePoint(p.IllegalSigner)
	if err != nil {
		return err
	}

	if !DefaultLedger.Arbitrators.IsArbitrator(p.IllegalSigner) {
		return errors.New("illegal signer is not one of current arbitrators")
	}

	_, err = common.Uint168FromAddress(p.GenesisBlockAddress)
	// todo check genesis block when sidechain registered in the future
	if err != nil {
		return err
	}

	if len(p.Signs) <= int(DefaultLedger.Arbitrators.GetArbitersMajorityCount()) {
		return errors.New("insufficient signs count")
	}

	if p.Evidence.DataHash.Compare(p.CompareEvidence.DataHash) >= 0 {
		return errors.New("evidence order error")
	}

	//todo get arbitrators by payload.Height and verify each sign in signs

	return nil
}

func CheckInactiveArbitrators(txn interfaces.Transaction) error {
	p, ok := txn.Payload().(*payload.InactiveArbitrators)
	if !ok {
		return errors.New("invalid payload")
	}

	if !DefaultLedger.Arbitrators.IsCRCArbitrator(p.Sponsor) {
		return errors.New("sponsor is not belong to arbitrators")
	}

	for _, v := range p.Arbitrators {
		if !DefaultLedger.Arbitrators.IsActiveProducer(v) &&
			!DefaultLedger.Arbitrators.IsDisabledProducer(v) {
			return errors.New("inactive arbitrator is not belong to " +
				"arbitrators")
		}
		if DefaultLedger.Arbitrators.IsCRCArbitrator(v) {
			return errors.New("inactive arbiters should not include CRC")
		}
	}

	if err := checkCRCArbitratorsSignatures(txn.Programs()[0]); err != nil {
		return err
	}

	return nil
}

func checkArbitratorsSignatures(program *program.Program) error {
	code := program.Code
	// Get N parameter
	n := int(code[len(code)-2]) - crypto.PUSH1 + 1
	// Get M parameter
	m := int(code[0]) - crypto.PUSH1 + 1

	var arbitratorsCount int
	arbiters := DefaultLedger.Arbitrators.GetArbitrators()
	for _, a := range arbiters {
		if a.IsNormal {
			arbitratorsCount++
		}
	}
	minSignCount := int(float64(DefaultLedger.Arbitrators.GetArbitersCount())*
		state.MajoritySignRatioNumerator/state.MajoritySignRatioDenominator) + 1
	if m < 1 || m > n || n != arbitratorsCount || m < minSignCount {
		return errors.New("invalid multi sign script code")
	}
	publicKeys, err := crypto.ParseMultisigScript(code)
	if err != nil {
		return err
	}

	for _, pk := range publicKeys {
		if !DefaultLedger.Arbitrators.IsArbitrator(pk[1:]) {
			return errors.New("invalid multi sign public key")
		}
	}

	return nil
}

func checkCRCArbitratorsSignatures(program *program.Program) error {

	code := program.Code
	// Get N parameter
	n := int(code[len(code)-2]) - crypto.PUSH1 + 1
	// Get M parameter
	m := int(code[0]) - crypto.PUSH1 + 1

	crcArbitratorsCount := DefaultLedger.Arbitrators.GetCRCArbitersCount()
	minSignCount := int(float64(crcArbitratorsCount)*
		state.MajoritySignRatioNumerator/state.MajoritySignRatioDenominator) + 1
	if m < 1 || m > n || n != crcArbitratorsCount || m < minSignCount {
		fmt.Printf("m:%d n:%d minSignCount:%d crc:  %d", m, n, minSignCount, crcArbitratorsCount)
		return errors.New("invalid multi sign script code")
	}
	publicKeys, err := crypto.ParseMultisigScript(code)
	if err != nil {
		return err
	}

	for _, pk := range publicKeys {
		if !DefaultLedger.Arbitrators.IsCRCArbitrator(pk[1:]) {
			return errors.New("invalid multi sign public key")
		}
	}
	return nil
}

func CheckDPOSIllegalProposals(d *payload.DPOSIllegalProposals) error {

	if err := ValidateProposalEvidence(&d.Evidence); err != nil {
		return err
	}

	if err := ValidateProposalEvidence(&d.CompareEvidence); err != nil {
		return err
	}

	if d.Evidence.BlockHeight != d.CompareEvidence.BlockHeight {
		return errors.New("should be in same height")
	}

	if d.Evidence.Proposal.Hash().IsEqual(d.CompareEvidence.Proposal.Hash()) {
		return errors.New("proposals can not be same")
	}

	if d.Evidence.Proposal.Hash().Compare(
		d.CompareEvidence.Proposal.Hash()) > 0 {
		return errors.New("evidence order error")
	}

	if !bytes.Equal(d.Evidence.Proposal.Sponsor, d.CompareEvidence.Proposal.Sponsor) {
		return errors.New("should be same sponsor")
	}

	if d.Evidence.Proposal.ViewOffset != d.CompareEvidence.Proposal.ViewOffset {
		return errors.New("should in same view")
	}

	if err := ProposalCheckByHeight(&d.Evidence.Proposal, d.GetBlockHeight()); err != nil {
		return err
	}

	if err := ProposalCheckByHeight(&d.CompareEvidence.Proposal,
		d.GetBlockHeight()); err != nil {
		return err
	}

	return nil
}

func CheckDPOSIllegalVotes(d *payload.DPOSIllegalVotes) error {

	if err := ValidateVoteEvidence(&d.Evidence); err != nil {
		return err
	}

	if err := ValidateVoteEvidence(&d.CompareEvidence); err != nil {
		return err
	}

	if d.Evidence.BlockHeight != d.CompareEvidence.BlockHeight {
		return errors.New("should be in same height")
	}

	if d.Evidence.Vote.Hash().IsEqual(d.CompareEvidence.Vote.Hash()) {
		return errors.New("votes can not be same")
	}

	if d.Evidence.Vote.Hash().Compare(d.CompareEvidence.Vote.Hash()) > 0 {
		return errors.New("evidence order error")
	}

	if !bytes.Equal(d.Evidence.Vote.Signer, d.CompareEvidence.Vote.Signer) {
		return errors.New("should be same signer")
	}

	if !bytes.Equal(d.Evidence.Proposal.Sponsor, d.CompareEvidence.Proposal.Sponsor) {
		return errors.New("should be same sponsor")
	}

	if d.Evidence.Proposal.ViewOffset != d.CompareEvidence.Proposal.ViewOffset {
		return errors.New("should in same view")
	}

	if err := ProposalCheckByHeight(&d.Evidence.Proposal,
		d.GetBlockHeight()); err != nil {
		return err
	}

	if err := ProposalCheckByHeight(&d.CompareEvidence.Proposal,
		d.GetBlockHeight()); err != nil {
		return err
	}

	if err := VoteCheckByHeight(&d.Evidence.Vote,
		d.GetBlockHeight()); err != nil {
		return err
	}

	if err := VoteCheckByHeight(&d.CompareEvidence.Vote,
		d.GetBlockHeight()); err != nil {
		return err
	}

	return nil
}

func CheckDPOSIllegalBlocks(d *payload.DPOSIllegalBlocks) error {

	if d.Evidence.BlockHash().IsEqual(d.CompareEvidence.BlockHash()) {
		return errors.New("blocks can not be same")
	}

	if common.BytesToHexString(d.Evidence.Header) >
		common.BytesToHexString(d.CompareEvidence.Header) {
		return errors.New("evidence order error")
	}

	if d.CoinType == payload.ELACoin {
		var err error
		var header, compareHeader *common2.Header
		var confirm, compareConfirm *payload.Confirm

		if header, compareHeader, err = checkDPOSElaIllegalBlockHeaders(d); err != nil {
			return err
		}

		if confirm, compareConfirm, err = checkDPOSElaIllegalBlockConfirms(
			d, header, compareHeader); err != nil {
			return err
		}

		if err := checkDPOSElaIllegalBlockSigners(d, confirm, compareConfirm); err != nil {
			return err
		}
	} else {
		return errors.New("unknown coin type")
	}

	return nil
}

func checkDPOSElaIllegalBlockSigners(
	d *payload.DPOSIllegalBlocks, confirm *payload.Confirm,
	compareConfirm *payload.Confirm) error {

	signers := d.Evidence.Signers
	compareSigners := d.CompareEvidence.Signers

	if len(signers) != len(confirm.Votes) ||
		len(compareSigners) != len(compareConfirm.Votes) {
		return errors.New("signers count it not match the count of " +
			"confirm votes")
	}

	arbitratorsSet := make(map[string]interface{})
	nodePublicKeys := DefaultLedger.Arbitrators.GetAllProducersPublicKey()
	for _, pk := range nodePublicKeys {
		arbitratorsSet[pk] = nil
	}

	for _, v := range signers {
		if _, ok := arbitratorsSet[common.BytesToHexString(v)]; !ok &&
			!DefaultLedger.Arbitrators.IsDisabledProducer(v) {
			return errors.New("invalid signers within evidence")
		}
	}
	for _, v := range compareSigners {
		if _, ok := arbitratorsSet[common.BytesToHexString(v)]; !ok &&
			!DefaultLedger.Arbitrators.IsDisabledProducer(v) {
			return errors.New("invalid signers within evidence")
		}
	}

	confirmSigners := getConfirmSigners(confirm)
	for _, v := range signers {
		if _, ok := confirmSigners[common.BytesToHexString(v)]; !ok {
			return errors.New("signers and confirm votes do not match")
		}
	}

	compareConfirmSigners := getConfirmSigners(compareConfirm)
	for _, v := range compareSigners {
		if _, ok := compareConfirmSigners[common.BytesToHexString(v)]; !ok {
			return errors.New("signers and confirm votes do not match")
		}
	}

	return nil
}

func checkDPOSElaIllegalBlockConfirms(d *payload.DPOSIllegalBlocks,
	header *common2.Header, compareHeader *common2.Header) (*payload.Confirm,
	*payload.Confirm, error) {

	confirm := &payload.Confirm{}
	compareConfirm := &payload.Confirm{}

	data := new(bytes.Buffer)
	data.Write(d.Evidence.BlockConfirm)
	if err := confirm.Deserialize(data); err != nil {
		return nil, nil, err
	}

	data = new(bytes.Buffer)
	data.Write(d.CompareEvidence.BlockConfirm)
	if err := compareConfirm.Deserialize(data); err != nil {
		return nil, nil, err
	}

	if err := ConfirmSanityCheck(confirm); err != nil {
		return nil, nil, err
	}
	if err := IllegalConfirmContextCheck(confirm); err != nil {
		return nil, nil, err
	}

	if err := ConfirmSanityCheck(compareConfirm); err != nil {
		return nil, nil, err
	}
	if err := IllegalConfirmContextCheck(compareConfirm); err != nil {
		return nil, nil, err
	}

	if confirm.Proposal.ViewOffset != compareConfirm.Proposal.ViewOffset {
		return nil, nil, errors.New("confirm view offset should be same")
	}

	if !confirm.Proposal.BlockHash.IsEqual(header.Hash()) {
		return nil, nil, errors.New("block and related confirm do not match")
	}

	if !compareConfirm.Proposal.BlockHash.IsEqual(compareHeader.Hash()) {
		return nil, nil, errors.New("block and related confirm do not match")
	}

	return confirm, compareConfirm, nil
}

func checkDPOSElaIllegalBlockHeaders(d *payload.DPOSIllegalBlocks) (*common2.Header,
	*common2.Header, error) {

	header := &common2.Header{}
	compareHeader := &common2.Header{}

	data := new(bytes.Buffer)
	data.Write(d.Evidence.Header)
	if err := header.Deserialize(data); err != nil {
		return nil, nil, err
	}

	data = new(bytes.Buffer)
	data.Write(d.CompareEvidence.Header)
	if err := compareHeader.Deserialize(data); err != nil {
		return nil, nil, err
	}

	if header.Height != d.BlockHeight || compareHeader.Height != d.BlockHeight {
		return nil, nil, errors.New("block header height should be same")
	}

	//todo check header content later if needed
	// (there is no need to check headers sanity, because arbiters check these
	// headers already. On the other hand, if arbiters do evil to sign multiple
	// headers that are not valid, normal node shall not attach to the chain.
	// So there is no motivation for them to do this.)

	return header, compareHeader, nil
}

func getConfirmSigners(
	confirm *payload.Confirm) map[string]interface{} {
	result := make(map[string]interface{})
	for _, v := range confirm.Votes {
		result[common.BytesToHexString(v.Signer)] = nil
	}
	return result
}

func CheckStringField(rawStr string, field string, allowEmpty bool) error {
	if (!allowEmpty && len(rawStr) == 0) || len(rawStr) > MaxStringLength {
		return fmt.Errorf("field %s has invalid string length", field)
	}

	return nil
}

func ValidateProposalEvidence(evidence *payload.ProposalEvidence) error {

	header := &common2.Header{}
	buf := new(bytes.Buffer)
	buf.Write(evidence.BlockHeader)

	if err := header.Deserialize(buf); err != nil {
		return err
	}

	if header.Height != evidence.BlockHeight {
		return errors.New("evidence height and block height should match")
	}

	if !header.Hash().IsEqual(evidence.Proposal.BlockHash) {
		return errors.New("proposal hash and block should match")
	}

	return nil
}

func ValidateVoteEvidence(evidence *payload.VoteEvidence) error {
	if err := ValidateProposalEvidence(&evidence.ProposalEvidence); err != nil {
		return err
	}

	if !evidence.Proposal.Hash().IsEqual(evidence.Vote.ProposalHash) {
		return errors.New("vote and proposal should match")
	}

	return nil
}
