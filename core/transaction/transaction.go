// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

const (
	InvalidTransactionSize = -1
)

type BaseTransaction struct {
	DefaultChecker

	version        common2.TransactionVersion // New field added in TxVersion09
	txType         common2.TxType
	payloadVersion byte
	payload        interfaces.Payload
	attributes     []*common2.Attribute
	inputs         []*common2.Input
	outputs        []*common2.Output
	lockTime       uint32
	programs       []*pg.Program
	fee            common.Fixed64
	feePerKB       common.Fixed64

	txHash *common.Uint256
}

func (tx *BaseTransaction) Version() common2.TransactionVersion {
	return tx.version
}

func (tx *BaseTransaction) TxType() common2.TxType {
	return tx.txType
}

func (tx *BaseTransaction) PayloadVersion() byte {
	return tx.payloadVersion
}

func (tx *BaseTransaction) Payload() interfaces.Payload {
	return tx.payload
}

func (tx *BaseTransaction) Attributes() []*common2.Attribute {
	return tx.attributes
}

func (tx *BaseTransaction) Inputs() []*common2.Input {
	return tx.inputs
}

func (tx *BaseTransaction) Outputs() []*common2.Output {
	return tx.outputs
}

func (tx *BaseTransaction) LockTime() uint32 {
	return tx.lockTime
}

func (tx *BaseTransaction) Programs() []*pg.Program {
	return tx.programs
}

func (tx *BaseTransaction) Fee() common.Fixed64 {
	return tx.fee
}

func (tx *BaseTransaction) FeePerKB() common.Fixed64 {
	return tx.feePerKB
}

func (tx *BaseTransaction) SetFee(fee common.Fixed64) {
	tx.fee = fee
}

func (tx *BaseTransaction) SetVersion(version common2.TransactionVersion) {
	tx.version = version
}

func (tx *BaseTransaction) SetTxType(txType common2.TxType) {
	tx.txType = txType
}

func (tx *BaseTransaction) SetFeePerKB(feePerKB common.Fixed64) {
	tx.feePerKB = feePerKB
}

func (tx *BaseTransaction) SetAttributes(attributes []*common2.Attribute) {
	tx.attributes = attributes
}

func (tx *BaseTransaction) SetPayloadVersion(payloadVersion byte) {
	tx.payloadVersion = payloadVersion
}

func (tx *BaseTransaction) SetPayload(payload interfaces.Payload) {
	tx.payload = payload
}

func (tx *BaseTransaction) SetInputs(inputs []*common2.Input) {
	tx.inputs = inputs
}

func (tx *BaseTransaction) SetOutputs(outputs []*common2.Output) {
	tx.outputs = outputs
}

func (tx *BaseTransaction) SetPrograms(programs []*pg.Program) {
	tx.programs = programs
}

func (tx *BaseTransaction) SetLockTime(lockTime uint32) {
	tx.lockTime = lockTime
}

func (tx *BaseTransaction) String() string {
	return fmt.Sprint("BaseTransaction: {\n\t",
		"Hash: ", tx.hash().String(), "\n\t",
		"Version: ", tx.version, "\n\t",
		"TxType: ", tx.txType.Name(), "\n\t",
		"PayloadVersion: ", tx.payloadVersion, "\n\t",
		"Payload: ", common.BytesToHexString(tx.payload.Data(tx.payloadVersion)), "\n\t",
		"Attributes: ", tx.attributes, "\n\t",
		"Inputs: ", tx.inputs, "\n\t",
		"Outputs: ", tx.outputs, "\n\t",
		"LockTime: ", tx.lockTime, "\n\t",
		"Programs: ", tx.programs, "\n\t",
		"}\n")
}

// Serialize the BaseTransaction
func (tx *BaseTransaction) Serialize(w io.Writer) error {
	if err := tx.SerializeUnsigned(w); err != nil {
		return errors.New("BaseTransaction txSerializeUnsigned Serialize failed, " + err.Error())
	}
	//Serialize  BaseTransaction's programs
	if err := common.WriteVarUint(w, uint64(len(tx.programs))); err != nil {
		return errors.New("BaseTransaction program count failed.")
	}
	for _, program := range tx.programs {
		if err := program.Serialize(w); err != nil {
			return errors.New("BaseTransaction Programs Serialize failed, " + err.Error())
		}
	}
	return nil
}

// Serialize the BaseTransaction data without contracts
func (tx *BaseTransaction) SerializeUnsigned(w io.Writer) error {
	// Version
	if tx.version >= common2.TxVersion09 {
		if _, err := w.Write([]byte{byte(tx.version)}); err != nil {
			return err
		}
	}
	// TxType
	if _, err := w.Write([]byte{byte(tx.txType)}); err != nil {
		return err
	}
	// PayloadVersion
	if _, err := w.Write([]byte{tx.payloadVersion}); err != nil {
		return err
	}
	// Payload
	if tx.payload == nil {
		return errors.New("BaseTransaction Payload is nil.")
	}
	if err := tx.payload.Serialize(w, tx.payloadVersion); err != nil {
		return err
	}

	//[]*txAttribute
	if err := common.WriteVarUint(w, uint64(len(tx.attributes))); err != nil {
		return errors.New("BaseTransaction item txAttribute length serialization failed.")
	}
	for _, attr := range tx.attributes {
		if err := attr.Serialize(w); err != nil {
			return err
		}
	}

	//[]*Inputs
	if err := common.WriteVarUint(w, uint64(len(tx.inputs))); err != nil {
		return errors.New("BaseTransaction item Inputs length serialization failed.")
	}
	for _, utxo := range tx.inputs {
		if err := utxo.Serialize(w); err != nil {
			return err
		}
	}

	//[]*Outputs
	if err := common.WriteVarUint(w, uint64(len(tx.outputs))); err != nil {
		return errors.New("BaseTransaction item Outputs length serialization failed.")
	}
	for _, output := range tx.outputs {
		if err := output.Serialize(w, tx.version); err != nil {
			return err
		}
	}

	return common.WriteUint32(w, tx.lockTime)
}

// Deserialize the BaseTransaction
func (tx *BaseTransaction) Deserialize(r io.Reader) error {
	// tx deserialize
	if err := tx.DeserializeUnsigned(r); err != nil {
		return errors.New("transaction Deserialize error: " + err.Error())
	}

	// tx program
	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return errors.New("transaction write program count error: " + err.Error())
	}
	for i := uint64(0); i < count; i++ {
		var program pg.Program
		if err := program.Deserialize(r); err != nil {
			return errors.New("transaction deserialize program error: " + err.Error())
		}
		tx.programs = append(tx.programs, &program)
	}
	return nil
}

func (tx *BaseTransaction) DeserializeUnsigned(r io.Reader) error {
	payloadVersion, err := common.ReadBytes(r, 1)
	if err != nil {
		return err
	}
	tx.payloadVersion = payloadVersion[0]

	tx.payload, err = interfaces.GetPayload(tx.txType, tx.payloadVersion)
	if err != nil {
		return err
	}

	err = tx.payload.Deserialize(r, tx.payloadVersion)
	if err != nil {
		return errors.New("deserialize Payload failed: " + err.Error())
	}
	// attributes
	count, err := common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}
	for i := uint64(0); i < count; i++ {
		var attr common2.Attribute
		if err := attr.Deserialize(r); err != nil {
			return err
		}
		tx.attributes = append(tx.attributes, &attr)
	}
	// inputs
	count, err = common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}
	for i := uint64(0); i < count; i++ {
		var input common2.Input
		if err := input.Deserialize(r); err != nil {
			return err
		}
		tx.inputs = append(tx.inputs, &input)
	}
	// outputs
	count, err = common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}
	for i := uint64(0); i < count; i++ {
		var output common2.Output
		if err := output.Deserialize(r, tx.version); err != nil {
			return err
		}
		tx.outputs = append(tx.outputs, &output)
	}

	tx.lockTime, err = common.ReadUint32(r)
	if err != nil {
		return err
	}

	return nil
}

func (tx *BaseTransaction) GetSize() int {
	buf := new(bytes.Buffer)
	if err := tx.Serialize(buf); err != nil {
		return InvalidTransactionSize
	}
	return buf.Len()
}

func (tx *BaseTransaction) hash() common.Uint256 {
	buf := new(bytes.Buffer)
	tx.SerializeUnsigned(buf)
	return common.Hash(buf.Bytes())
}

func (tx *BaseTransaction) Hash() common.Uint256 {
	if tx.txHash == nil {
		txHash := tx.hash()
		tx.txHash = &txHash
	}
	return *tx.txHash
}

func (tx *BaseTransaction) IsReturnSideChainDepositCoinTx() bool {
	return tx.txType == common2.ReturnSideChainDepositCoin
}

func (tx *BaseTransaction) ISCRCouncilMemberClaimNode() bool {
	return tx.txType == common2.CRCouncilMemberClaimNode
}

func (tx *BaseTransaction) IsCRAssetsRectifyTx() bool {
	return tx.txType == common2.CRAssetsRectify
}

func (tx *BaseTransaction) IsCRCAppropriationTx() bool {
	return tx.txType == common2.CRCAppropriation
}

func (tx *BaseTransaction) IsNextTurnDPOSInfoTx() bool {
	return tx.txType == common2.NextTurnDPOSInfo
}

func (tx *BaseTransaction) IsCustomIDResultTx() bool {
	return tx.txType == common2.ProposalResult
}

func (tx *BaseTransaction) IsCustomIDRelatedTx() bool {
	if tx.IsCRCProposalTx() {
		p, _ := tx.payload.(*payload.CRCProposal)
		return p.ProposalType == payload.ReserveCustomID ||
			p.ProposalType == payload.ReceiveCustomID ||
			p.ProposalType == payload.ChangeCustomIDFee
	}
	if tx.IsCustomIDResultTx() {
		return true
	}
	return false
}

func (tx *BaseTransaction) IsSideChainUpgradeTx() bool {
	if tx.IsCRCProposalTx() {
		p, _ := tx.payload.(*payload.CRCProposal)
		return p.ProposalType > payload.MinUpgradeProposalType &&
			p.ProposalType <= payload.MaxUpgradeProposalType
	}
	return false
}

func (tx *BaseTransaction) IsCRCProposalRealWithdrawTx() bool {
	return tx.txType == common2.CRCProposalRealWithdraw
}

func (tx *BaseTransaction) IsUpdateCRTx() bool {
	return tx.txType == common2.UpdateCR
}

func (tx *BaseTransaction) IsCRCProposalWithdrawTx() bool {
	return tx.txType == common2.CRCProposalWithdraw
}

func (tx *BaseTransaction) IsCRCProposalReviewTx() bool {
	return tx.txType == common2.CRCProposalReview
}

func (tx *BaseTransaction) IsCRCProposalTrackingTx() bool {
	return tx.txType == common2.CRCProposalTracking
}

func (tx *BaseTransaction) IsCRCProposalTx() bool {
	return tx.txType == common2.CRCProposal
}

func (tx *BaseTransaction) IsReturnCRDepositCoinTx() bool {
	return tx.txType == common2.ReturnCRDepositCoin
}

func (tx *BaseTransaction) IsUnregisterCRTx() bool {
	return tx.txType == common2.UnregisterCR
}

func (tx *BaseTransaction) IsRegisterCRTx() bool {
	return tx.txType == common2.RegisterCR
}

func (tx *BaseTransaction) IsIllegalTypeTx() bool {
	return tx.IsIllegalProposalTx() || tx.IsIllegalVoteTx() || tx.IsIllegalBlockTx() || tx.IsSidechainIllegalDataTx()
}

//special tx is this kind of tx who have no input and output
func (tx *BaseTransaction) IsSpecialTx() bool {
	if tx.IsIllegalTypeTx() || tx.IsInactiveArbitrators() || tx.IsNextTurnDPOSInfoTx() {
		return true
	}
	return false
}

func (tx *BaseTransaction) GetSpecialTxHash() (common.Uint256, error) {
	switch tx.txType {
	case common2.IllegalProposalEvidence, common2.IllegalVoteEvidence,
		common2.IllegalBlockEvidence, common2.IllegalSidechainEvidence, common2.InactiveArbitrators:
		illegalData, ok := tx.payload.(payload.DPOSIllegalData)
		if !ok {
			return common.Uint256{}, errors.New("special tx payload cast failed")
		}
		return illegalData.Hash(), nil
	case common2.NextTurnDPOSInfo:
		payloadData, ok := tx.payload.(*payload.NextTurnDPOSInfo)
		if !ok {
			return common.Uint256{}, errors.New("NextTurnDPOSInfo tx payload cast failed")
		}
		return payloadData.Hash(), nil
	}
	return common.Uint256{}, errors.New("wrong TxType not special tx")
}

func (tx *BaseTransaction) IsIllegalProposalTx() bool {
	return tx.txType == common2.IllegalProposalEvidence
}

func (tx *BaseTransaction) IsIllegalVoteTx() bool {
	return tx.txType == common2.IllegalVoteEvidence
}

func (tx *BaseTransaction) IsIllegalBlockTx() bool {
	return tx.txType == common2.IllegalBlockEvidence
}

func (tx *BaseTransaction) IsSidechainIllegalDataTx() bool {
	return tx.txType == common2.IllegalSidechainEvidence
}

func (tx *BaseTransaction) IsInactiveArbitrators() bool {
	return tx.txType == common2.InactiveArbitrators
}

func (tx *BaseTransaction) IsRevertToPOW() bool {
	return tx.txType == common2.RevertToPOW
}

func (tx *BaseTransaction) IsRevertToDPOS() bool {
	return tx.txType == common2.RevertToDPOS
}

func (tx *BaseTransaction) IsUpdateVersion() bool {
	return tx.txType == common2.UpdateVersion
}

func (tx *BaseTransaction) IsProducerRelatedTx() bool {
	return tx.txType == common2.RegisterProducer || tx.txType == common2.UpdateProducer ||
		tx.txType == common2.ActivateProducer || tx.txType == common2.CancelProducer
}

func (tx *BaseTransaction) IsUpdateProducerTx() bool {
	return tx.txType == common2.UpdateProducer
}

func (tx *BaseTransaction) IsReturnDepositCoin() bool {
	return tx.txType == common2.ReturnDepositCoin
}

func (tx *BaseTransaction) IsCancelProducerTx() bool {
	return tx.txType == common2.CancelProducer
}

func (tx *BaseTransaction) IsActivateProducerTx() bool {
	return tx.txType == common2.ActivateProducer
}

func (tx *BaseTransaction) IsRegisterProducerTx() bool {
	return tx.txType == common2.RegisterProducer
}

func (tx *BaseTransaction) IsSideChainPowTx() bool {
	return tx.txType == common2.SideChainPow
}

func (tx *BaseTransaction) IsNewSideChainPowTx() bool {
	if !tx.IsSideChainPowTx() || len(tx.inputs) != 0 {
		return false
	}

	return true
}

func (tx *BaseTransaction) IsTransferCrossChainAssetTx() bool {
	return tx.txType == common2.TransferCrossChainAsset
}

func (tx *BaseTransaction) IsWithdrawFromSideChainTx() bool {
	return tx.txType == common2.WithdrawFromSideChain
}

func (tx *BaseTransaction) IsRechargeToSideChainTx() bool {
	return tx.txType == common2.RechargeToSideChain
}

func (tx *BaseTransaction) IsCoinBaseTx() bool {
	return tx.txType == common2.CoinBase
}

func (tx *BaseTransaction) IsDposV2ClaimRewardTx() bool {
	return tx.txType == common2.DposV2ClaimReward
}

func (tx *BaseTransaction) IsDposV2ClaimRewardRealWithdraw() bool {
	return tx.txType == common2.DposV2ClaimRewardRealWithdraw
}

// SerializeSizeStripped returns the number of bytes it would take to serialize
// the block, excluding any witness data (if any).
func (tx *BaseTransaction) SerializeSizeStripped() int {
	// todo add cache for size according to btcd
	return tx.GetSize()
}

func (tx *BaseTransaction) IsSmallTransfer(min common.Fixed64) bool {
	var totalCrossAmt common.Fixed64
	if tx.payloadVersion == payload.TransferCrossChainVersion {
		payloadObj, ok := tx.payload.(*payload.TransferCrossChainAsset)
		if !ok {
			return false
		}
		for i := 0; i < len(payloadObj.CrossChainAddresses); i++ {
			if bytes.Compare(tx.outputs[payloadObj.OutputIndexes[i]].ProgramHash[0:1], []byte{byte(contract.PrefixCrossChain)}) == 0 {
				totalCrossAmt += tx.outputs[payloadObj.OutputIndexes[i]].Value
			}
		}
	} else {
		for _, o := range tx.outputs {
			if bytes.Compare(o.ProgramHash[0:1], []byte{byte(contract.PrefixCrossChain)}) == 0 {
				totalCrossAmt += o.Value
			}
		}
	}

	return totalCrossAmt <= min
}
