// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transactions

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

	Version        common2.TransactionVersion // New field added in TxVersion09
	TxType         common2.TxType
	PayloadVersion byte
	Payload        interfaces.Payload
	Attributes     []*common2.Attribute
	Inputs         []*common2.Input
	Outputs        []*common2.Output
	LockTime       uint32
	Programs       []*pg.Program
	Fee            common.Fixed64
	FeePerKB       common.Fixed64

	txHash *common.Uint256
}

func (tx *BaseTransaction) String() string {
	return fmt.Sprint("BaseTransaction: {\n\t",
		"Hash: ", tx.hash().String(), "\n\t",
		"Version: ", tx.Version, "\n\t",
		"TxType: ", tx.TxType.Name(), "\n\t",
		"PayloadVersion: ", tx.PayloadVersion, "\n\t",
		"Payload: ", common.BytesToHexString(tx.Payload.Data(tx.PayloadVersion)), "\n\t",
		"Attributes: ", tx.Attributes, "\n\t",
		"Inputs: ", tx.Inputs, "\n\t",
		"Outputs: ", tx.Outputs, "\n\t",
		"LockTime: ", tx.LockTime, "\n\t",
		"Programs: ", tx.Programs, "\n\t",
		"}\n")
}

// Serialize the BaseTransaction
func (tx *BaseTransaction) Serialize(w io.Writer) error {
	if err := tx.SerializeUnsigned(w); err != nil {
		return errors.New("BaseTransaction txSerializeUnsigned Serialize failed, " + err.Error())
	}
	//Serialize  BaseTransaction's programs
	if err := common.WriteVarUint(w, uint64(len(tx.Programs))); err != nil {
		return errors.New("BaseTransaction program count failed.")
	}
	for _, program := range tx.Programs {
		if err := program.Serialize(w); err != nil {
			return errors.New("BaseTransaction Programs Serialize failed, " + err.Error())
		}
	}
	return nil
}

// Serialize the BaseTransaction data without contracts
func (tx *BaseTransaction) SerializeUnsigned(w io.Writer) error {
	// Version
	if tx.Version >= common2.TxVersion09 {
		if _, err := w.Write([]byte{byte(tx.Version)}); err != nil {
			return err
		}
	}
	// TxType
	if _, err := w.Write([]byte{byte(tx.TxType)}); err != nil {
		return err
	}
	// PayloadVersion
	if _, err := w.Write([]byte{tx.PayloadVersion}); err != nil {
		return err
	}
	// Payload
	if tx.Payload == nil {
		return errors.New("BaseTransaction Payload is nil.")
	}
	if err := tx.Payload.Serialize(w, tx.PayloadVersion); err != nil {
		return err
	}

	//[]*txAttribute
	if err := common.WriteVarUint(w, uint64(len(tx.Attributes))); err != nil {
		return errors.New("BaseTransaction item txAttribute length serialization failed.")
	}
	for _, attr := range tx.Attributes {
		if err := attr.Serialize(w); err != nil {
			return err
		}
	}

	//[]*Inputs
	if err := common.WriteVarUint(w, uint64(len(tx.Inputs))); err != nil {
		return errors.New("BaseTransaction item Inputs length serialization failed.")
	}
	for _, utxo := range tx.Inputs {
		if err := utxo.Serialize(w); err != nil {
			return err
		}
	}

	//[]*Outputs
	if err := common.WriteVarUint(w, uint64(len(tx.Outputs))); err != nil {
		return errors.New("BaseTransaction item Outputs length serialization failed.")
	}
	for _, output := range tx.Outputs {
		if err := output.Serialize(w, tx.Version); err != nil {
			return err
		}
	}

	return common.WriteUint32(w, tx.LockTime)
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
		tx.Programs = append(tx.Programs, &program)
	}
	return nil
}

func (tx *BaseTransaction) DeserializeUnsigned(r io.Reader) error {
	flagByte, err := common.ReadBytes(r, 1)
	if err != nil {
		return err
	}

	if common2.TransactionVersion(flagByte[0]) >= common2.TxVersion09 {
		tx.Version = common2.TransactionVersion(flagByte[0])
		txType, err := common.ReadBytes(r, 1)
		if err != nil {
			return err
		}
		tx.TxType = common2.TxType(txType[0])
	} else {
		tx.Version = common2.TxVersionDefault
		tx.TxType = common2.TxType(flagByte[0])
	}

	payloadVersion, err := common.ReadBytes(r, 1)
	if err != nil {
		return err
	}
	tx.PayloadVersion = payloadVersion[0]

	tx.Payload, err = GetPayload(tx.TxType, tx.PayloadVersion)
	if err != nil {
		return err
	}

	err = tx.Payload.Deserialize(r, tx.PayloadVersion)
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
		tx.Attributes = append(tx.Attributes, &attr)
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
		tx.Inputs = append(tx.Inputs, &input)
	}
	// outputs
	count, err = common.ReadVarUint(r, 0)
	if err != nil {
		return err
	}
	for i := uint64(0); i < count; i++ {
		var output common2.Output
		if err := output.Deserialize(r, tx.Version); err != nil {
			return err
		}
		tx.Outputs = append(tx.Outputs, &output)
	}

	tx.LockTime, err = common.ReadUint32(r)
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
	return tx.TxType == common2.ReturnSideChainDepositCoin
}

func (tx *BaseTransaction) ISCRCouncilMemberClaimNode() bool {
	return tx.TxType == common2.CRCouncilMemberClaimNode
}

func (tx *BaseTransaction) IsCRAssetsRectifyTx() bool {
	return tx.TxType == common2.CRAssetsRectify
}

func (tx *BaseTransaction) IsCRCAppropriationTx() bool {
	return tx.TxType == common2.CRCAppropriation
}

func (tx *BaseTransaction) IsNextTurnDPOSInfoTx() bool {
	return tx.TxType == common2.NextTurnDPOSInfo
}

func (tx *BaseTransaction) IsCustomIDResultTx() bool {
	return tx.TxType == common2.ProposalResult
}

func (tx *BaseTransaction) IsCustomIDRelatedTx() bool {
	if tx.IsCRCProposalTx() {
		p, _ := tx.Payload.(*payload.CRCProposal)
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
		p, _ := tx.Payload.(*payload.CRCProposal)
		return p.ProposalType > payload.MinUpgradeProposalType &&
			p.ProposalType <= payload.MaxUpgradeProposalType
	}
	return false
}

func (tx *BaseTransaction) IsCRCProposalRealWithdrawTx() bool {
	return tx.TxType == common2.CRCProposalRealWithdraw
}

func (tx *BaseTransaction) IsUpdateCRTx() bool {
	return tx.TxType == common2.UpdateCR
}

func (tx *BaseTransaction) IsCRCProposalWithdrawTx() bool {
	return tx.TxType == common2.CRCProposalWithdraw
}

func (tx *BaseTransaction) IsCRCProposalReviewTx() bool {
	return tx.TxType == common2.CRCProposalReview
}

func (tx *BaseTransaction) IsCRCProposalTrackingTx() bool {
	return tx.TxType == common2.CRCProposalTracking
}

func (tx *BaseTransaction) IsCRCProposalTx() bool {
	return tx.TxType == common2.CRCProposal
}

func (tx *BaseTransaction) IsReturnCRDepositCoinTx() bool {
	return tx.TxType == common2.ReturnCRDepositCoin
}

func (tx *BaseTransaction) IsUnregisterCRTx() bool {
	return tx.TxType == common2.UnregisterCR
}

func (tx *BaseTransaction) IsRegisterCRTx() bool {
	return tx.TxType == common2.RegisterCR
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
	switch tx.TxType {
	case common2.IllegalProposalEvidence, common2.IllegalVoteEvidence,
		common2.IllegalBlockEvidence, common2.IllegalSidechainEvidence, common2.InactiveArbitrators:
		illegalData, ok := tx.Payload.(payload.DPOSIllegalData)
		if !ok {
			return common.Uint256{}, errors.New("special tx payload cast failed")
		}
		return illegalData.Hash(), nil
	case common2.NextTurnDPOSInfo:
		payloadData, ok := tx.Payload.(*payload.NextTurnDPOSInfo)
		if !ok {
			return common.Uint256{}, errors.New("NextTurnDPOSInfo tx payload cast failed")
		}
		return payloadData.Hash(), nil
	}
	return common.Uint256{}, errors.New("wrong TxType not special tx")
}

func (tx *BaseTransaction) IsIllegalProposalTx() bool {
	return tx.TxType == common2.IllegalProposalEvidence
}

func (tx *BaseTransaction) IsIllegalVoteTx() bool {
	return tx.TxType == common2.IllegalVoteEvidence
}

func (tx *BaseTransaction) IsIllegalBlockTx() bool {
	return tx.TxType == common2.IllegalBlockEvidence
}

func (tx *BaseTransaction) IsSidechainIllegalDataTx() bool {
	return tx.TxType == common2.IllegalSidechainEvidence
}

func (tx *BaseTransaction) IsInactiveArbitrators() bool {
	return tx.TxType == common2.InactiveArbitrators
}

func (tx *BaseTransaction) IsRevertToPOW() bool {
	return tx.TxType == common2.RevertToPOW
}

func (tx *BaseTransaction) IsRevertToDPOS() bool {
	return tx.TxType == common2.RevertToDPOS
}

func (tx *BaseTransaction) IsUpdateVersion() bool {
	return tx.TxType == common2.UpdateVersion
}

func (tx *BaseTransaction) IsProducerRelatedTx() bool {
	return tx.TxType == common2.RegisterProducer || tx.TxType == common2.UpdateProducer ||
		tx.TxType == common2.ActivateProducer || tx.TxType == common2.CancelProducer
}

func (tx *BaseTransaction) IsUpdateProducerTx() bool {
	return tx.TxType == common2.UpdateProducer
}

func (tx *BaseTransaction) IsReturnDepositCoin() bool {
	return tx.TxType == common2.ReturnDepositCoin
}

func (tx *BaseTransaction) IsCancelProducerTx() bool {
	return tx.TxType == common2.CancelProducer
}

func (tx *BaseTransaction) IsActivateProducerTx() bool {
	return tx.TxType == common2.ActivateProducer
}

func (tx *BaseTransaction) IsRegisterProducerTx() bool {
	return tx.TxType == common2.RegisterProducer
}

func (tx *BaseTransaction) IsSideChainPowTx() bool {
	return tx.TxType == common2.SideChainPow
}

func (tx *BaseTransaction) IsNewSideChainPowTx() bool {
	if !tx.IsSideChainPowTx() || len(tx.Inputs) != 0 {
		return false
	}

	return true
}

func (tx *BaseTransaction) IsTransferCrossChainAssetTx() bool {
	return tx.TxType == common2.TransferCrossChainAsset
}

func (tx *BaseTransaction) IsWithdrawFromSideChainTx() bool {
	return tx.TxType == common2.WithdrawFromSideChain
}

func (tx *BaseTransaction) IsRechargeToSideChainTx() bool {
	return tx.TxType == common2.RechargeToSideChain
}

func (tx *BaseTransaction) IsCoinBaseTx() bool {
	return tx.TxType == common2.CoinBase
}

// SerializeSizeStripped returns the number of bytes it would take to serialize
// the block, excluding any witness data (if any).
func (tx *BaseTransaction) SerializeSizeStripped() int {
	// todo add cache for size according to btcd
	return tx.GetSize()
}

func GetPayload(txType common2.TxType, payloadVersion byte) (interfaces.Payload, error) {
	// todo use payloadVersion

	var p interfaces.Payload
	switch txType {
	case common2.CoinBase:
		p = new(payload.CoinBase)
	case common2.RegisterAsset:
		p = new(payload.RegisterAsset)
	case common2.TransferAsset:
		p = new(payload.TransferAsset)
	case common2.Record:
		p = new(payload.Record)
	case common2.SideChainPow:
		p = new(payload.SideChainPow)
	case common2.WithdrawFromSideChain:
		p = new(payload.WithdrawFromSideChain)
	case common2.TransferCrossChainAsset:
		p = new(payload.TransferCrossChainAsset)
	case common2.RegisterProducer:
		p = new(payload.ProducerInfo)
	case common2.CancelProducer:
		p = new(payload.ProcessProducer)
	case common2.UpdateProducer:
		p = new(payload.ProducerInfo)
	case common2.ReturnDepositCoin:
		p = new(payload.ReturnDepositCoin)
	case common2.ActivateProducer:
		p = new(payload.ActivateProducer)
	case common2.IllegalProposalEvidence:
		p = new(payload.DPOSIllegalProposals)
	case common2.IllegalVoteEvidence:
		p = new(payload.DPOSIllegalVotes)
	case common2.IllegalBlockEvidence:
		p = new(payload.DPOSIllegalBlocks)
	case common2.IllegalSidechainEvidence:
		p = new(payload.SidechainIllegalData)
	case common2.InactiveArbitrators:
		p = new(payload.InactiveArbitrators)
	case common2.RevertToDPOS:
		p = new(payload.RevertToDPOS)
	case common2.UpdateVersion:
		p = new(payload.UpdateVersion)
	case common2.RegisterCR:
		p = new(payload.CRInfo)
	case common2.UpdateCR:
		p = new(payload.CRInfo)
	case common2.UnregisterCR:
		p = new(payload.UnregisterCR)
	case common2.ReturnCRDepositCoin:
		p = new(payload.ReturnDepositCoin)
	case common2.CRCProposal:
		p = new(payload.CRCProposal)
	case common2.CRCProposalReview:
		p = new(payload.CRCProposalReview)
	case common2.CRCProposalWithdraw:
		p = new(payload.CRCProposalWithdraw)
	case common2.CRCProposalTracking:
		p = new(payload.CRCProposalTracking)
	case common2.CRCAppropriation:
		p = new(payload.CRCAppropriation)
	case common2.CRAssetsRectify:
		p = new(payload.CRAssetsRectify)
	case common2.CRCProposalRealWithdraw:
		p = new(payload.CRCProposalRealWithdraw)
	case common2.CRCouncilMemberClaimNode:
		p = new(payload.CRCouncilMemberClaimNode)
	case common2.NextTurnDPOSInfo:
		p = new(payload.NextTurnDPOSInfo)
	case common2.RevertToPOW:
		p = new(payload.RevertToPOW)
	case common2.ProposalResult:
		p = new(payload.RecordProposalResult)
	case common2.ReturnSideChainDepositCoin:
		p = new(payload.ReturnSideChainDepositCoin)
	default:
		return nil, errors.New("[BaseTransaction], invalid transaction type.")
	}
	return p, nil
}

func (tx *BaseTransaction) IsSmallTransfer(min common.Fixed64) bool {
	var totalCrossAmt common.Fixed64
	if tx.PayloadVersion == payload.TransferCrossChainVersion {
		payloadObj, ok := tx.Payload.(*payload.TransferCrossChainAsset)
		if !ok {
			return false
		}
		for i := 0; i < len(payloadObj.CrossChainAddresses); i++ {
			if bytes.Compare(tx.Outputs[payloadObj.OutputIndexes[i]].ProgramHash[0:1], []byte{byte(contract.PrefixCrossChain)}) == 0 {
				totalCrossAmt += tx.Outputs[payloadObj.OutputIndexes[i]].Value
			}
		}
	} else {
		for _, o := range tx.Outputs {
			if bytes.Compare(o.ProgramHash[0:1], []byte{byte(contract.PrefixCrossChain)}) == 0 {
				totalCrossAmt += o.Value
			}
		}
	}

	return totalCrossAmt <= min
}
